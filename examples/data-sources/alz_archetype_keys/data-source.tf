data "alz_archetype_keys" "example" {
  base_archetype            = "root"
  policy_definitions_to_add = ["MyPolicyDefinition"]
  policy_assignments_to_add = ["MyPolicyAssignment"]
}
