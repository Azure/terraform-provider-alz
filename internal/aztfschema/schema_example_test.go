package aztfschema_test

import (
	"fmt"
	"sort"

	"github.com/Azure/terraform-provider-alz/internal/aztfschema"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
)

// ExampleGenerator_WithAuthAttrs is an example of using the WithAuthAttrs method.
// It shows how you can generate a provider schema with authentication attributes,
// merging in your own custom attributes.
func ExampleGenerator_WithAuthAttrs() {
	// Create a new provider schema and merge in the authentication attributes.
	mySchema := schema.Schema{
		MarkdownDescription: "Example schema with authentication attributes",
		Attributes: aztfschema.NewGenerator().WithAuthAttrs().Merge(map[string]schema.Attribute{
			"example_attribute": schema.StringAttribute{
				Optional: true,
			},
		}),
	}

	keys := make([]string, 0, len(mySchema.Attributes))
	for k := range mySchema.Attributes {
		keys = append(keys, k)
	}

	sort.Strings(keys)

	for _, k := range keys {
		fmt.Println(k)
	}

	// Output:
	// auxiliary_tenant_ids
	// client_certificate
	// client_certificate_password
	// client_certificate_path
	// client_id
	// client_id_file_path
	// client_secret
	// client_secret_file_path
	// environment
	// example_attribute
	// oidc_azure_service_connection_id
	// oidc_request_token
	// oidc_request_url
	// oidc_token
	// oidc_token_file_path
	// skip_provider_registration
	// tenant_id
	// use_aks_workload_identity
	// use_cli
	// use_msi
	// use_oidc
}
