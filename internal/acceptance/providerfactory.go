package acceptance

import (
	"testing"

	"github.com/Azure/terraform-provider-alz/internal/provider"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

// AccTestProtoV6ProviderFactoriesUnique is used to ensure that the provider instance used for
// each acceptance test is unique.
// This is necessary because this provider make use of state stored in the provider instance.
// See type AlzProvider.
func AccTestProtoV6ProviderFactoriesUnique() map[string]func() (tfprotov6.ProviderServer, error) {
	return map[string]func() (tfprotov6.ProviderServer, error){
		"alz": providerserver.NewProtocol6WithError(provider.New("test")()),
	}
}

// AccTestPreCheck ensures that the environment is properly configured for acceptance testing.
func AccTestPreCheck(t *testing.T) {
	// You can add code here to run prior to any test case execution, for example assertions
	// about the appropriate environment variables being set are common to see in a pre-check
	// function.
}
