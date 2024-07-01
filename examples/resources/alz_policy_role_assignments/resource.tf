data "azurerm_client_config" "example" {}

data "alz_architecture" "example" {
  name                     = "alz"
  root_management_group_id = azurerm_client_config.example.tenant_id
  location                 = "westus2"
}

resource "azurerm_management_group_policy_assignment" "this" {
  for_each = local.alz_policy_assignments_decoded
  # Insert required configuration here
}


locals {
  # Create new map from the data source but use known (at plan time) map keys from `alz_archetype_keys`
  alz_policy_assignments_decoded = { for k in data.alz_archetype_keys.example.alz_policy_assignment_keys : k => jsondecode(data.alz_archetype.this.alz_policy_assignments[k]) }

  # Create a map of role assignment for the scope of the management group
  policy_role_assignments = data.alz_archetype.this.alz_policy_role_assignments != null ? {
    for pra_key, pra_val in data.alz_archetype.this.alz_policy_role_assignments : pra_key => {
      scope              = pra_val.scope
      role_definition_id = pra_val.role_definition_id
      principal_id       = one(azurerm_management_group_policy_assignment.example[pra_val.assignment_name].identity).principal_id
    }
  } : {}
}

resource "alz_policy_role_assignments" "example" {
  id          = "alz-root"
  assignments = local.policy_role_assignments
}
