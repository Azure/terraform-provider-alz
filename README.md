# Azure Landing Zones (ALZ) Terraform Provider

> ⚠️ ***Warning*** ⚠️ This provider is currently in development and is not yet ready for use. It should be considered experimental and is subject to change.

The ALZ Terraform Provider is a data source provider for Azure Landing Zones.
It is used to generate data for the [Azure Landing Zones Terraform Module](https://github.com/Azure/terraform-azurerm-alz).

It simplifies the task of creating Azure Management Group hierarchies, together with Azure Policy and authorization.

*This provider is built on the [Terraform Plugin Framework](https://github.com/hashicorp/terraform-plugin-framework).*

Please see the [GitHub template repository documentation](https://help.github.com/en/github/creating-cloning-and-archiving-repositories/creating-a-repository-from-a-template) for how to create a new repository from this template on GitHub.

## Using the provider

***This example will be extended when the Terraform module is created.***

Here is how to use the provider to generate the data required to create the ALZ organizational root management group.

```hcl
provider "alz" {
  defaults = {
    location = "westeurope"
  }

  # See documentation for the customization options.
}

data "alz_management_group" "root" {
  name      = "root"
  archetype = "root"

  # See documentation for the customization options.
}

output "root_mg" {
  value = data.alz_management_group.root
}
```

## Developing the Provider

The [DEVELOPER.md](https://github.com/Azure/terraform-provider-alz/blob/main/DEVELOPER.md) file is a basic outline on how to build and develop the provider while more detailed guides geared towards contributors can be found in the /contributing directory of this repository.
