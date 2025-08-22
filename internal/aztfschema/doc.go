/*
Package aztfschema provides helpers for building Terraform Plugin Framework
schemas for Azure authentication along with reusable, strongly typed models.

The package focuses on two areas:

  - Generator: a fluent helper that merges a standard set of Azure
    authentication attributes (for example, client_id, tenant_id,
    subscription_id, OIDC- and MSI-related flags) into provider or resource
    schemas, including appropriate validators.

  - Models: AuthModel and AuthModelWithSubscriptionID types built on
    terraform-plugin-framework types that can:

  - populate opinionated defaults from struct tags via SetOpinionatedDefaults

  - read values from environment variables via ConfigureFromEnv

  - produce an aztfauth.Option via the AuthOption method, which links this
    package to the aztfauth package for creating Azure credentials

Use these utilities to ensure consistent, well-documented authentication
options across providers and resources that target the HashiCorp Terraform
Plugin Framework. The AuthOption method is intended to be passed to
aztfauth.NewCredential to obtain an azcore.TokenCredential chain configured
from the model.
*/
package aztfschema
