data "azurerm_client_config" "current" {}

data "alz_archetype_keys" "example" {
  base_archetype = "root"
}

data "alz_archetype" "example" {
  defaults = {
    location = "westeurope"
  }
  id             = "alz-root"
  base_archetype = "root"
  display_name   = "alz-root"
  parent_id      = data.azurerm_client_config.current.tenant_id
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
