package aztfschema_test

import (
	"fmt"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/terraform-provider-alz/internal/aztfschema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// ProviderModel is the struct for the provider schema,
// it embeds the AuthModelWithSubscriptionID from aztfschema.
type ProviderModel struct {
	aztfschema.AuthModelWithSubscriptionID
	MyCustomAttr types.String `tfsdk:"my_custom_attr"`
}

// Example showing how to embed AuthModelWithSubscriptionID into your own model.
// Use the options at the end in the aztfauth package to get a token.
func ExampleAuthModelWithSubscriptionID_AuthOption() {

	model := ProviderModel{
		AuthModelWithSubscriptionID: aztfschema.AuthModelWithSubscriptionID{
			SubscriptionID: types.StringValue("00000000-0000-0000-0000-000000000000"),
			AuthModel: aztfschema.AuthModel{
				ClientID:           types.StringValue("00000000-0000-0000-0000-000000000000"),
				TenantID:           types.StringValue("00000000-0000-0000-0000-000000000000"),
				UseOIDC:            types.BoolValue(true),
				AuxiliaryTenantIDs: types.ListNull(types.StringType),
			},
		},
		MyCustomAttr: types.StringValue("custom-value"),
	}

	// This will configure the model from environment variables,
	// if the values are not already set.
	model.ConfigureFromEnv()

	// This will set any default values for the model if they are not already set.
	// Enables CLI, disables OIDC & MSI
	model.SetOpinionatedDefaults()

	opts := model.AuthOption(azcore.ClientOptions{})

	fmt.Println("MSI auth:", opts.UseMSI)
	fmt.Println("Client ID:", opts.ClientId)
	fmt.Println("Tenant ID:", opts.TenantId)
	fmt.Println("Use OIDC:", opts.UseOIDCToken)
	fmt.Println("Auxiliary Tenant IDs:", opts.AdditionallyAllowedTenants)

	// Output:
	// MSI auth: false
	// Client ID: 00000000-0000-0000-0000-000000000000
	// Tenant ID: 00000000-0000-0000-0000-000000000000
	// Use OIDC: true
	// Auxiliary Tenant IDs: []
}
