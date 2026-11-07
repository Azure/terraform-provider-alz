xa-trust/reconcile-api22.py


#!/usr/bin/env python3
"""XA-TRUST BCVL - Reconciliador API-22 Merkle Proof
Audit: XA-TRUST-BCVL-EXEC-001
Regra: HTTP_200 ↛ PROOF_VALID
"""
import json
import hashlib
import sys
import os

EVIDENCE_PATH = "xa-trust/evidence/test-execution.json"

def sha256_hex(data: bytes) -> str:
    return hashlib.sha256(data).hexdigest()

def leaf_hash(data: str) -> str:
    # Folha Merkle = sha256("leaf:" + dado)
    return sha256_hex(f"leaf:{data}".encode("utf-8"))

def parent_hash(left: str, right: str) -> str:
    # Nó pai = sha256(left + right) - ordem importa
    return sha256_hex(f"{left}{right}".encode("utf-8"))

def verify_merkle_proof(leaf_data: str, proof: list, expected_root: str) -> dict:
    current = leaf_hash(leaf_data)
    steps = []
    for i, p in enumerate(proof):
        sibling = p.get("hash", "")
        position = p.get("position", "right")
        if position == "left":
            nxt = parent_hash(sibling, current)
            steps.append(f"step {i}: parent(left={sibling[:8]}.., current={current[:8]}..) = {nxt[:8]}..")
        else:
            nxt = parent_hash(current, sibling)
            steps.append(f"step {i}: parent(current={current[:8]}.., right={sibling[:8]}..) = {nxt[:8]}..")
        current = nxt
    return {
        "computed_root": current,
        "expected_root": expected_root,
        "match": current == expected_root,
        "steps": steps
    }

def main():
    if not os.path.exists(EVIDENCE_PATH):
        print(f"[FAIL] Arquivo não encontrado: {EVIDENCE_PATH}")
        print("Rode primeiro: python3 xa-trust/producer.py")
        sys.exit(1)

    with open(EVIDENCE_PATH, "r", encoding="utf-8") as f:
        data = json.load(f)

    # 1. Encontra API-22 em qualquer formato
    api22 = None
    # Formato 1: lista de results
    results = data.get("results", []) or data.get("apis", []) or []
    for r in results:
        rid = str(r.get("api_id", "") or r.get("id", "") or r.get("api", ""))
        if "22" in rid or "merkle/proof" in str(r).lower():
            api22 = r
            break
    # Formato 2: dict por API
    if api22 is None and isinstance(data, dict):
        api22 = data.get("API-22") or data.get("api-22")

    if api22 is None:
        print("[FAIL] API-22 não encontrada no test-execution.json")
        print(f"Chaves disponíveis: {list(data.keys())[:20]}")
        sys.exit(1)

    print("=== API-22 DECLARADO ===")
    print(json.dumps(api22, indent=2))

    http_status = api22.get("http_status", api22.get("http", api22.get("status", 0)))
    declared_result = api22.get("result", api22.get("declared", api22.get("outcome", "")))
    print(f"\nHTTP declarado: {http_status}")
    print(f"Resultado declarado: {declared_result}")

    # 2. Pega os campos Merkle - tenta 3 formatos comuns
    merkle = api22.get("merkle", api22.get("proof_data", api22))
    leaf = merkle.get("leaf", merkle.get("leaf_data", merkle.get("data", "")))
    proof = merkle.get("proof", merkle.get("path", merkle.get("siblings", [])))
    expected_root = merkle.get("root", merkle.get("expected_root", merkle.get("merkle_root", "")))

    if not leaf or not proof or not expected_root:
        print("\n[CONFLICT] Campos Merkle incompletos no test-execution.json")
        print(f"leaf presente: {bool(leaf)} | proof presente: {bool(proof)} | root presente: {bool(expected_root)}")
        print("Não dá para recalcular. E7 continua NOT_PROMOTED.")
        result = {
            "audit_id": "XA-TRUST-BCVL-EXEC-001",
            "api": "API-22",
            "http_status": http_status,
            "declared": declared_result,
            "recalculated": "NOT_POSSIBLE_MISSING_FIELDS",
            "verdict": "CONFLICT",
            "E7": "NOT_PROMOTED",
            "E8": "NOT_ESTABLISHED"
        }
        with open("xa-trust/evidence/reconciliation-api22.json", "w") as f:
            json.dump(result, f, indent=2)
        sys.exit(1)

    # 3. Recalcula
    recalc = verify_merkle_proof(leaf, proof, expected_root)

    print("\n=== RECALCULO INDEPENDENTE ===")
    for s in recalc["steps"]:
        print(s)
    print(f"Computed root: {recalc['computed_root']}")
    print(f"Expected root: {recalc['expected_root']}")
    print(f"Match: {recalc['match']}")

    # 4. Veredicto
    http_ok = str(http_status) in ["200", "201", "204"]
    declared_valid = str(declared_result).upper() in ["PROOF_VALID", "VALID", "PASS", "VERIFIED"]
    recalc_valid = recalc["match"]

    print("\n=== VEREDICTO ===")
    print(f"HTTP 2xx: {http_ok}")
    print(f"Declarado PROOF_VALID: {declared_valid}")
    print(f"Recalculado PROOF_VALID: {recalc_valid}")

    if http_ok and declared_valid and recalc_valid:
        verdict = "RECONCILED"
        print("\n[OK] RECONCILED - HTTP, declaração e recálculo concordam. Pode promover E7 para API-22.")
    elif http_ok and declared_valid and not recalc_valid:
        verdict = "CONFLICT"
        print("\n[FAIL] CONFLICT - HTTP 200 + declarado PROOF_VALID, mas recálculo FALHOU.")
        print("Isso é erro de dado OU erro de declaração. NÃO promove E7.")
    elif http_ok and not declared_valid:
        verdict = "CONFLICT"
        print("\n[FAIL] CONFLICT - HTTP 200 mas declaração não é PROOF_VALID.")
    else:
        verdict = "CONFLICT"
        print("\n[FAIL] CONFLICT - HTTP / declaração / recálculo divergem.")

    result = {
        "audit_id": "XA-TRUST-BCVL-EXEC-001",
        "api": "API-22",
        "endpoint": "POST /ledger/merkle/proof-verify",
        "http_status": http_status,
        "declared": declared_result,
        "recalculated_match": recalc_valid,
        "computed_root": recalc["computed_root"],
        "expected_root": recalc["expected_root"],
        "verdict": verdict,
        "E7": "VERIFIED" if verdict == "RECONCILED" else "NOT_PROMOTED",
        "E8": "NOT_ESTABLISHED"
    }
    os.makedirs("xa-trust/evidence", exist_ok=True)
    with open("xa-trust/evidence/reconciliation-api22.json", "w") as f:
        json.dump(result, f, indent=2)

    print(f"\nSalvo em: xa-trust/evidence/reconciliation-api22.json")
    sys.exit(0 if verdict == "RECONCILED" else 1)

if __name__ == "__main__":
    main()



python3 -m py_compile xa-trust/reconcile-api22.py && echo "compila OK"
python3 xa-trust/reconcile-api22.py
cat xa-trust/evidence/reconciliation-api22.json




[OK] RECONCILED - HTTP, declaração e recálculo concordam.
"verdict": "RECONCILED",
"E7": "VERIFIED"


[FAIL] CONFLICT - HTTP 200 + declarado PROOF_VALID, mas recálculo FALHOU.
Computed root: a1b2c3...
Expected root: 9f8e7d...
Match: False


cat xa-trust/evidence/reconciliation-api22.json



[![OpenSSF Scorecard](https://api.scorecard.dev/projects/github.com/Azure/terraform-provider-alz/badge)](https://scorecard.dev/viewer/?uri=github.com/Azure/terraform-provider-alz)

# Azure Landing Zones (ALZ) Terraform Provider

> ❗ ***Important*** ❗ This provider has been designed to work with the ALZ Terraform module. We suggest that you consume this provider from within the module, rather than directly in your Terraform configuration.

The ALZ Terraform Provider is primarily a data source provider for Azure Landing Zones.
It is used to generate data for the [Azure Landing Zones Terraform Module](https://github.com/Azure/terraform-azurerm-alz).

It simplifies the task of creating Azure Management Group hierarchies, together with Azure Policy and authorization.

*This provider is built on the [Terraform Plugin Framework](https://github.com/hashicorp/terraform-plugin-framework).*

## Using the provider

See the associated [module documentation](https://github.com/Azure/terraform-azurerm-alz) and examples for how to use the provider.

## Developing the Provider

The [DEVELOPER.md](https://github.com/Azure/terraform-provider-alz/blob/main/DEVELOPER.md) file is a basic outline on how to build and develop the provider while more detailed guides geared towards contributors can be found in the /contributing directory of this repository.
