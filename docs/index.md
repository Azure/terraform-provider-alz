---
page_title: "Provider: ALZ"
subcategory: ""
description: |-
  Use the Azure Landing Zones (ALZ) provider to generate data to allow you to easily provision your ALZ configuration.
  You must configure the provider with the proper credentials before you can use it.

  This provider has been designed to work with the [ALZ Terraform module](https://registry.terraform.io/modules/Azure/avm-ptn-alz/azurerm/latest).
  We suggest that you consume this provider from within the module, rather than directly in your Terraform configuration.
---

# ALZ

Use the Azure Landing Zones (ALZ) provider to generate data to allow you to simplify provisioning of your ALZ configuration.
Its principal use is to generate data to deploy resources with the [AzureRM](https://github.com/Azure/terraform-provider-azurerm) provider.
However, the provider does deploy some resources directly, in order to work around limitations in Terraform.

Use the navigation to the left to read about the available resources.

~> **Important** This provider has been designed to work with the [ALZ Terraform module](https://registry.terraform.io/modules/Azure/avm-ptn-alz/azurerm/latest). We suggest that you consume this provider from within the module, rather than directly in your Terraform configuration.

~> **Warning** This provider is still in development but is ready for initial testing and feedback via [GitHub Issues](https://github.com/Azure/terraform-provider-alz/issues).

## Example Usage

```terraform
provider "alz" {
  alz_lib_ref = "platform/alz@v2024.03.00" # using a specific release from the ALZ platform library
  lib_urls = [
    "${path.root}/lib",                                     # local library
    "github.com/MyOrg/MyRepo//some/dir?ref=v1.1.0&depth=1", # checking out a specific version
  ]
}
```

## Authentication and Configuration

Configuration for the ALZ provider can be derived from several sources, which are applied in the following order:

1. Parameters in the provider configuration
1. Environment variables

## Versions

For production use, you should constrain the acceptable provider versions via
configuration, to ensure that new versions with breaking changes will not be
automatically installed by `terraform init` in the future:

```terraform
terraform {
  required_providers {
    alz = {
      source  = "azure/alz"
      version = "<version>" # change this to your desired version, https://www.terraform.io/language/expressions/version-constraints
    }
  }
}
```

As this provider is still at version zero, you should constrain the acceptable
provider versions on the minor version.

## Azure Landing Zones Library

The provider will download the Azure Landing Zones Library from the [Azure Landing Zones Library GitHub repository](https://github.com/Azure/Azure-Landing-Zones-Library).
The asserts are in the `platform/alz` directory and are version tagged in order to provide a consistent experience.
Within the library are the following types of asserts:

- **policy definitions** - These are the policy definitions that are used to enforce the policies in the Azure Policy service.
- **policy assignments** - These are the policy assignments that are used to assign the policy definitions to the appropriate scope.
- **policy set definitions** - These are the policy set definitions that are used to group policy definitions together.
- **role definitions** - These are the role definitions that are used to define the roles in the Azure Role-Based Access Control (RBAC) service.
- **archetype definitions** - These group together the policy definitions, policy assignments, policy set definitions, and role definitions that and can be assigned to a management group.
- **archetype overrides** - These create new archetypes based off an existing archetype.

~> **Important** If the provider does not have access to download the library, please download and use the `lib_urls` to specify the local directory.

For more information please visit the [GitHub repository](https://github.com/Azure/Azure-Landing-Zones-Library).

<!-- schema generated by tfplugindocs -->
## Schema

### Optional

- `alz_lib_ref` (String) The reference (tag) in the ALZ library to use. Default is `platform/alz/2024.03.00`.
- `auxiliary_tenant_ids` (List of String) A list of auxiliary tenant ids which should be used. If not specified, value will be attempted to be read from the `ARM_AUXILIARY_TENANT_IDS` environment variable. When configuring from the environment, use a semicolon as a delimiter.
- `client_certificate_password` (String, Sensitive) The password associated with the client certificate. For use when authenticating as a service principal using a client certificate. If not specified, value will be attempted to be read from the `ARM_CLIENT_CERTIFICATE_PASSWORD` environment variable.
- `client_certificate_path` (String) The path to the client certificate associated with the service principal for use when authenticating as a service principal using a client certificate. If not specified, value will be attempted to be read from the `ARM_CLIENT_CERTIFICATE_PATH` environment variable.
- `client_id` (String) The client id which should be used. For use when authenticating as a service principal. If not specified, value will be attempted to be read from the `ARM_CLIENT_ID` environment variable.
- `client_secret` (String, Sensitive) The client secret which should be used. For use when authenticating as a service principal using a client secret. If not specified, value will be attempted to be read from the `ARM_CLIENT_SECRET` environment variable.
- `environment` (String) The cloud environment which should be used. Possible values are `public`, `usgovernment` and `china`. Defaults to `public`. If not specified, value will be attempted to be read from the `ARM_ENVIRONMENT` environment variable.
- `lib_overwrite_enabled` (Boolean) Whether to allow overwriting of the library by other lib directories. Default is `false`.
- `lib_urls` (List of String) A list of directories or URLs to use for ALZ libraries. The URLs will be processed in order. See <https://pkg.go.dev/github.com/hashicorp/go-getter#readme-url-format> for URL syntax. Note that if use_alz_lib is set to true then it will always be the first library used.
- `oidc_request_token` (String, Sensitive) The bearer token for the request to the OIDC provider. For use when authenticating using OpenID Connect. If not specified, value will be attempted to be read from the first non-empty value of the `ARM_OIDC_REQUEST_TOKEN` and `ACTIONS_ID_TOKEN_REQUEST_TOKEN` environment variables.
- `oidc_request_url` (String) The URL for the OIDC provider from which to request an id token. For use when authenticating as a service principal using OpenID Connect. If not specified, value will be attempted to be read from the first non-empty value of the `ARM_OIDC_REQUEST_URL` and `ACTIONS_ID_TOKEN_REQUEST_URL` environment variables.
- `oidc_token` (String, Sensitive) The OIDC id token for use when authenticating as a service principal using OpenID Connect. If not specified, value will be attempted to be read from the `ARM_OIDC_TOKEN` environment variable.
- `oidc_token_file_path` (String) The path to a file containing an OIDC id token for use when authenticating using OpenID Connect. If not specified, value will be attempted to be read from the `ARM_OIDC_TOKEN_FILE_PATH` environment variable.
- `skip_provider_registration` (Boolean) Should the provider skip registering all of the resource providers that it supports, if they're not already registered? Default is `false`. If not specified, value will be attempted to be read from the `ARM_SKIP_PROVIDER_REGISTRATION` environment variable.
- `tenant_id` (String) The Tenant ID which should be used. If not specified, value will be attempted to be read from the `ARM_TENANT_ID` environment variable.
- `use_alz_lib` (Boolean) Use the default ALZ library to resolve archetypes. Default is `true`. The ALZ library is always used first, and then the directories or URLs specified in `lib_urls` are used in order.
- `use_cli` (Boolean) Allow Azure CLI to be used for authentication. Default is `true`. If not specified, value will be attempted to be read from the `ARM_USE_CLI` environment variable.
- `use_msi` (Boolean) Allow managed service identity to be used for authentication. Default is `false`. If not specified, value will be attempted to be read from the `ARM_USE_MSI` environment variable.
- `use_oidc` (Boolean) Allow OpenID Connect to be used for authentication. Default is `false`. If not specified, value will be attempted to be read from the `ARM_USE_OIDC` environment variable.
