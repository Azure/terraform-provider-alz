data "azurerm_client_config" "example" {}

data "alz_architecture" "example" {
  name                     = "alz"
  root_management_group_id = azurerm_client_config.example.tenant_id
  location                 = "northeurope"
}

output "managment_groups" {
  description = "The management groups of the architecture as list of objects."
  value       = data.alz_architecture.example.managment_groups
}
