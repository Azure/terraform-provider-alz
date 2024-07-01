package provider

import (
	"testing"

	"github.com/Azure/terraform-provider-alz/internal/provider/gen"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/stretchr/testify/assert"
)

// TestAccAlzArchetypeDataSource tests the data source for alz_archetype.
// It checks that the policy parameter substitution & location defaults are applied.
func TestAccAlzPolicyRoleAssignmentsResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactoriesUnique(),
		ExternalProviders: map[string]resource.ExternalProvider{
			"azurerm": {
				Source:            "hashicorp/azurerm",
				VersionConstraint: "~> 3.107",
			},
		},
		Steps: []resource.TestStep{
			{
				Config: testAccAlzPolicyRoleAssignmentsResourceConfigOne(),
				Check:  resource.ComposeAggregateTestCheckFunc(),
			},
			{
				Config: testAccAlzPolicyRoleAssignmentsResourceConfigTwo(),
				Check:  resource.ComposeAggregateTestCheckFunc(),
			},
			{
				Config: testAccAlzPolicyRoleAssignmentsResourceConfigOne(),
				Check:  resource.ComposeAggregateTestCheckFunc(),
			},
		},
	})
}

// testAccArchitectureDataSourceConfig returns a test configuration for TestAccAlzArchetypeDataSource.
func testAccAlzPolicyRoleAssignmentsResourceConfigOne() string {
	return `
provider "alz" {}

provider "azurerm" {
  features {}
}

data "azurerm_client_config" "current" {}

resource "alz_policy_role_assignments" "test" {
	assignments = [
		{
			principal_id       = data.azurerm_client_config.current.object_id
			role_definition_id = "/providers/Microsoft.Authorization/roleDefinitions/acdd72a7-3385-48ef-bd42-f606fba81ae7" # reader
			scope              = "/subscriptions/${data.azurerm_client_config.current.subscription_id}"
		}
	]
}
`
}

// testAccArchitectureDataSourceConfig returns a test configuration for TestAccAlzArchetypeDataSource.
func testAccAlzPolicyRoleAssignmentsResourceConfigTwo() string {
	return `
provider "alz" {}

provider "azurerm" {
  features {}
}

data "azurerm_client_config" "current" {}

resource "alz_policy_role_assignments" "test" {
	assignments = [
		{
			principal_id       = data.azurerm_client_config.current.object_id
			role_definition_id = "/providers/Microsoft.Authorization/roleDefinitions/acdd72a7-3385-48ef-bd42-f606fba81ae7" # reader
			scope              = "/subscriptions/${data.azurerm_client_config.current.subscription_id}"
		},
		{
			principal_id       = data.azurerm_client_config.current.object_id
			role_definition_id = "/providers/Microsoft.Authorization/roleDefinitions/b24988ac-6180-42a0-ab88-20f7382dd24c" # contributer
			scope              = "/subscriptions/${data.azurerm_client_config.current.subscription_id}"
		}
	]
}
`
}

func TestStandardizeRoleAssignmentRoleDefinititionId(t *testing.T) {
	// Test a valid input.
	input := "/subscriptions/00000000-0000-0000-0000-000000000000/providers/Microsoft.Authorization/roleDefinitions/92aaf0da-9dab-42b6-94a3-d43ce8d16293"
	expectedOutput := "/providers/Microsoft.Authorization/roleDefinitions/92aaf0da-9dab-42b6-94a3-d43ce8d16293"
	output := standardizeRoleAssignmentRoleDefinititionId(input)
	assert.Equal(t, expectedOutput, output)

	// Test an invalid input.
	input = "/subscriptions/00000000-0000-0000-0000-000000000000/providers/Microsoft.Authorization/roleDefinitions"
	expectedOutput = "/subscriptions/00000000-0000-0000-0000-000000000000/providers/Microsoft.Authorization/roleDefinitions"
	output = standardizeRoleAssignmentRoleDefinititionId(input)
	assert.Equal(t, expectedOutput, output)
}

func TestPolicyRoleAssignmentFromSlice(t *testing.T) {
	slice := []gen.AssignmentsValue{
		{
			PrincipalId:      types.StringValue("principal1"),
			RoleDefinitionId: types.StringValue("role1"),
			Scope:            types.StringValue("scope1"),
		},
		{
			PrincipalId:      types.StringValue("principal2"),
			RoleDefinitionId: types.StringValue("role2"),
			Scope:            types.StringValue("scope2"),
		},
		{
			PrincipalId:      types.StringValue("principal3"),
			RoleDefinitionId: types.StringValue("role3"),
			Scope:            types.StringValue("scope3"),
		},
	}

	want := &slice[1]
	got := policyRoleAssignmentFromSlice(slice, *want)

	assert.Equal(t, got, want)

	// Test not present.
	want = &gen.AssignmentsValue{}
	got = policyRoleAssignmentFromSlice(slice, *want)
	assert.Nil(t, got)
}

func TestGenPolicyRoleAssignmentId(t *testing.T) {
	pra := gen.AssignmentsValue{
		PrincipalId:      types.StringValue("principal1"),
		RoleDefinitionId: types.StringValue("role1"),
		Scope:            types.StringValue("scope1"),
	}
	expectedOutput := "3882958e-d42e-55eb-aed9-4c9827d1cf2d"
	output := genPolicyRoleAssignmentId(pra)
	assert.Equal(t, expectedOutput, output)
}
