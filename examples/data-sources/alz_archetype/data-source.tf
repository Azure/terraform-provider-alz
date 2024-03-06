data "azurerm_client_config" "current" {}

data "alz_archetype" "example" {
  defaults = {
    location = "westeurope"
  }
  id             = "alz-root"
  base_archetype = "root"
  display_name   = "alz-root"
  parent_id      = data.azurerm_client_config.current.tenant_id
}
