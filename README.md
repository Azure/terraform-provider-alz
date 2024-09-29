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
