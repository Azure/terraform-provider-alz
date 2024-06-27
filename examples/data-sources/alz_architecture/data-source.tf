data "azurerm_client_config" "example" {}

data "alz_architecture" "example" {
  name                     = "alz"
  root_management_group_id = azurerm_client_config.example.tenant_id
  location                 = "westus2"
}
