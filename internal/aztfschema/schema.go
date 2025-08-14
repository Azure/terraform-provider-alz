package aztfschema

import (
	"maps"

	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Generator helps to generate a Terraform provider schema that includes the standard authentication attributes.
// Do not create instances of this type directly - use the NewGenerator function instead.
// The methods follow a fluent interface pattern, therefore can be used directly in the provider schema definition.
type Generator struct {
	attrs map[string]schema.Attribute
}

// NewGenerator creates a new Generator instance.
func NewGenerator() *Generator {
	return &Generator{
		attrs: make(map[string]schema.Attribute),
	}
}

// WithAuthAttrs adds the authentication attributes to the provided schema attribute map.
func (g *Generator) WithAuthAttrs() *Generator {
	maps.Insert(g.attrs, maps.All(authAttrs))
	return g
}

// WithSubscriptionID adds the subscription ID attribute to the provided schema attribute map.
func (g *Generator) WithSubscriptionID() *Generator {
	maps.Insert(g.attrs, maps.All(subIDAttr))
	return g
}

// Merge adds the provided attributes to the existing schema attribute map.
// This allows for the non-authentication attributes to be included as well.
func (g *Generator) Merge(in map[string]schema.Attribute) map[string]schema.Attribute {
	maps.Insert(g.attrs, maps.All(in))
	return g.attrs
}

// subIDAttr is the subscription ID attribute.
var subIDAttr map[string]schema.Attribute = map[string]schema.Attribute{
	"subscription_id": schema.StringAttribute{
		Optional:            true,
		MarkdownDescription: "The Subscription ID which should be used. This can also be sourced from the `ARM_SUBSCRIPTION_ID` Environment Variable.",
	},
}

// authAttrs is the authentication attributes map.
var authAttrs map[string]schema.Attribute = map[string]schema.Attribute{
	"client_id": schema.StringAttribute{
		Optional:            true,
		MarkdownDescription: "The Client ID which should be used. This can also be sourced from the `ARM_CLIENT_ID`, `AZURE_CLIENT_ID` Environment Variable.",
	},

	"client_id_file_path": schema.StringAttribute{
		Optional:            true,
		MarkdownDescription: "The path to a file containing the Client ID which should be used. This can also be sourced from the `ARM_CLIENT_ID_FILE_PATH` Environment Variable.",
	},

	"tenant_id": schema.StringAttribute{
		Optional:            true,
		MarkdownDescription: "The Tenant ID should be used. This can also be sourced from the `ARM_TENANT_ID` Environment Variable.",
	},

	"auxiliary_tenant_ids": schema.ListAttribute{
		ElementType:         types.StringType,
		Optional:            true,
		Validators:          []validator.List{listvalidator.SizeAtMost(3)},
		MarkdownDescription: "List of auxiliary Tenant IDs required for multi-tenancy and cross-tenant scenarios. This can also be sourced from the `ARM_AUXILIARY_TENANT_IDS` Environment Variable.",
	},

	"environment": schema.StringAttribute{
		Optional: true,
		Validators: []validator.String{
			stringvalidator.OneOfCaseInsensitive("public", "usgovernment", "china"),
		},
		MarkdownDescription: "The Cloud Environment which should be used. Possible values are `public`, `usgovernment` and `china`. Defaults to `public`. This can also be sourced from the `ARM_ENVIRONMENT` or `AZURE_ENVIRONMENT` Environment Variables.",
	},

	// TODO@mgd: the metadata_host is used to retrieve metadata from Azure to identify current environment, this is used to eliminate Azure Stack usage, in which case the provider doesn't support.
	// "metadata_host": {
	// 	Type:        schema.TypeString,
	// 	Required:    true,
	// 	DefaultFunc: schema.EnvDefaultFunc("ARM_METADATA_HOSTNAME", ""),
	// 	Description: "The Hostname which should be used for the Azure Metadata Service.",
	// },

	// Client Certificate specific fields
	"client_certificate_path": schema.StringAttribute{
		Optional:            true,
		MarkdownDescription: "The path to the Client Certificate associated with the Service Principal which should be used. This can also be sourced from the `ARM_CLIENT_CERTIFICATE_PATH` Environment Variable.",
	},

	"client_certificate": schema.StringAttribute{
		Optional:            true,
		MarkdownDescription: "A base64-encoded PKCS#12 bundle to be used as the client certificate for authentication. This can also be sourced from the `ARM_CLIENT_CERTIFICATE` environment variable.",
	},

	"client_certificate_password": schema.StringAttribute{
		Optional:            true,
		MarkdownDescription: "The password associated with the Client Certificate. This can also be sourced from the `ARM_CLIENT_CERTIFICATE_PASSWORD` Environment Variable.",
	},

	// Client Secret specific fields
	"client_secret": schema.StringAttribute{
		Optional:            true,
		MarkdownDescription: "The Client Secret which should be used. This can also be sourced from the `ARM_CLIENT_SECRET` Environment Variable.",
	},

	"client_secret_file_path": schema.StringAttribute{
		Optional:            true,
		MarkdownDescription: "The path to a file containing the Client Secret which should be used. For use When authenticating as a Service Principal using a Client Secret. This can also be sourced from the `ARM_CLIENT_SECRET_FILE_PATH` Environment Variable.",
	},

	"skip_provider_registration": schema.BoolAttribute{
		Optional:            true,
		MarkdownDescription: "Should the Provider skip registering the Resource Providers it supports? This can also be sourced from the `ARM_SKIP_PROVIDER_REGISTRATION` Environment Variable. Defaults to `false`.",
	},

	// OIDC specific fields
	"oidc_request_token": schema.StringAttribute{
		Optional:            true,
		MarkdownDescription: "The bearer token for the request to the OIDC provider. This can also be sourced from the `ARM_OIDC_REQUEST_TOKEN`, `ACTIONS_ID_TOKEN_REQUEST_TOKEN`, or `SYSTEM_ACCESSTOKEN` Environment Variables.",
	},

	"oidc_request_url": schema.StringAttribute{
		Optional:            true,
		MarkdownDescription: "The URL for the OIDC provider from which to request an ID token. This can also be sourced from the `ARM_OIDC_REQUEST_URL`, `ACTIONS_ID_TOKEN_REQUEST_URL`, or `SYSTEM_OIDCREQUESTURI` Environment Variables.",
	},

	"oidc_token": schema.StringAttribute{
		Optional:            true,
		MarkdownDescription: "The ID token when authenticating using OpenID Connect (OIDC). This can also be sourced from the `ARM_OIDC_TOKEN` environment Variable.",
	},

	"oidc_token_file_path": schema.StringAttribute{
		Optional:            true,
		MarkdownDescription: "The path to a file containing an ID token when authenticating using OpenID Connect (OIDC). This can also be sourced from the `ARM_OIDC_TOKEN_FILE_PATH`, `AZURE_FEDERATED_TOKEN_FILE` environment Variable.",
	},

	"oidc_azure_service_connection_id": schema.StringAttribute{
		Optional:            true,
		MarkdownDescription: "The Azure Pipelines Service Connection ID to use for authentication. This can also be sourced from the `ARM_ADO_PIPELINE_SERVICE_CONNECTION_ID`, `ARM_OIDC_AZURE_SERVICE_CONNECTION_ID`, or `AZURESUBSCRIPTION_SERVICE_CONNECTION_ID` Environment Variables.",
	},

	"use_oidc": schema.BoolAttribute{
		Optional:            true,
		MarkdownDescription: "Should OIDC be used for Authentication? This can also be sourced from the `ARM_USE_OIDC` Environment Variable. Defaults to `false`.",
	},

	// Azure CLI specific fields
	"use_cli": schema.BoolAttribute{
		Optional:            true,
		MarkdownDescription: "Should Azure CLI be used for authentication? This can also be sourced from the `ARM_USE_CLI` environment variable. Defaults to `true`.",
	},

	// Managed Service Identity specific fields
	"use_msi": schema.BoolAttribute{
		Optional:            true,
		MarkdownDescription: "Should Managed Identity be used for Authentication? This can also be sourced from the `ARM_USE_MSI` Environment Variable. Defaults to `false`.",
	},

	"use_aks_workload_identity": schema.BoolAttribute{
		Optional:            true,
		MarkdownDescription: "Should AKS Workload Identity be used for Authentication? This can also be sourced from the `ARM_USE_AKS_WORKLOAD_IDENTITY` Environment Variable. Defaults to `false`. When set, `client_id`, `tenant_id` and `oidc_token_file_path` will be detected from the environment and do not need to be specified.",
	},
	// TODO@mgd: azidentity doesn't support msi_endpoint
	// "msi_endpoint": {
	// 	Type:        schema.TypeString,
	// 	Optional:    true,
	// 	DefaultFunc: schema.EnvDefaultFunc("ARM_MSI_ENDPOINT", ""),
	// 	Description: "The path to a custom endpoint for Managed Service Identity - in most circumstances this should be detected automatically. ",
	// },
}
