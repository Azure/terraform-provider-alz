data "azapi_client_config" "example" {}

data "alz_architecture" "example" {
  name                     = "alz"
  root_management_group_id = data.azapi_client_config.example.tenant_id
  location                 = "swedencentral"
}

output "management_groups" {
  description = "The management groups of the architecture as list of objects."
  value       = data.alz_architecture.example.management_groups
}
