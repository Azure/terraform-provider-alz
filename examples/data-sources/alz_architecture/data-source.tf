data "azapi_client_config" "example" {}

data "alz_architecture" "example" {
  name                     = "alz"
  root_management_group_id = azapi_client_config.example.tenant_id
  location                 = "swedencentral"
}

output "managment_groups" {
  description = "The management groups of the architecture as list of objects."
  value       = data.alz_architecture.example.managment_groups
}
