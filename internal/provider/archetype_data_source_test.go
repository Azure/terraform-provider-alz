// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package provider

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/Azure/alzlib"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armpolicy"
	"github.com/Azure/terraform-provider-alz/internal/alztypes"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/stretchr/testify/assert"
)

// TestAccAlzArchetypeDataSource tests the data source for alz_archetype.
// It checks that the policy parameter substitution & location defaults are applied.
func TestAccAlzArchetypeDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactoriesUnique(),
		ExternalProviders: map[string]resource.ExternalProvider{
			"random": {
				Source: "hashicorp/random",
			},
		},
		Steps: []resource.TestStep{
			{
				Config: testAccExampleDataSourceConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.alz_archetype.test", "id", "example"),
					resource.TestCheckOutput("test_location_replacement", "westeurope"),
					resource.TestCheckOutput("test_parameter_replacement", "test"),
				),
			},
		},
	})
}

// TestAccFullAlz is a full in-memory creation of the ALZ reference architecture.
func TestAccAlzArchetypeDataSourceFullAlz(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactoriesUnique(),
		Steps: []resource.TestStep{
			{
				Config: testAccFullAlzConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.alz_archetype.root", "id", "root"),
					resource.TestCheckResourceAttr("data.alz_archetype.root", "alz_policy_assignments.%", "13"),
					resource.TestCheckResourceAttr("data.alz_archetype.root", "alz_policy_definitions.%", "132"),
					resource.TestCheckResourceAttr("data.alz_archetype.root", "alz_policy_set_definitions.%", "15"),
					resource.TestCheckResourceAttr("data.alz_archetype.root", "alz_role_definitions.%", "5"),
				),
			},
		},
	})
}

// testAccExampleDataSourceConfig returns a test configuration for TestAccAlzArchetypeDataSource.
func testAccExampleDataSourceConfig() string {
	cwd, _ := os.Getwd()
	libPath := filepath.Join(cwd, "testdata/testacc_lib")

	return fmt.Sprintf(`
provider "alz" {
  use_alz_lib = false
  lib_urls = [
    "%s",
  ]
}

resource "random_pet" "test" {}

data "alz_archetype" "test" {
  id             = "example"
  parent_id      = "test"
  base_archetype = "test"
  defaults = {
    location = "westeurope"
    log_analytics_workspace_id = "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg/providers/Microsoft.OperationalInsights/workspaces/la"
  }

  policy_assignments_to_modify = {
    BlobServicesDiagnosticsLogsToWorkspace = {
      parameters = jsonencode({
        logAnalytics = "test"
      })
    }
  }
}


# Test that the data source is returning the correct value for the policy location
output "test_location_replacement" {
  value = jsondecode(data.alz_archetype.test.alz_policy_assignments["BlobServicesDiagnosticsLogsToWorkspace"]).location
}
output "test_parameter_replacement" {
  value = jsondecode(data.alz_archetype.test.alz_policy_assignments["BlobServicesDiagnosticsLogsToWorkspace"]).properties.parameters.logAnalytics.value
}
`, libPath)
}

// testAccExampleDataSourceConfig returns a test configuration for TestAccAlzArchetypeDataSource.
func testAccFullAlzConfig() string {
	return `provider "alz" {}

data "alz_archetype" "root" {
  id             = "root"
  parent_id      = "00000000-0000-0000-0000-000000000000"
  base_archetype = "root"
  defaults = {
    location = "westeurope"
    log_analytics_workspace_id = "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg/providers/Microsoft.OperationalInsights/workspaces/la"
  }
}

data "alz_archetype" "landing_zones" {
  id             = "landing_zones"
  parent_id      = data.alz_archetype.root.id
  base_archetype = "landing_zones"
  defaults = {
    location = "westeurope"
    log_analytics_workspace_id = "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg/providers/Microsoft.OperationalInsights/workspaces/la"
  }
}

data "alz_archetype" "platform" {
  id             = "platform"
  parent_id      = data.alz_archetype.root.id
  base_archetype = "platform"
  defaults = {
    location = "westeurope"
    log_analytics_workspace_id = "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg/providers/Microsoft.OperationalInsights/workspaces/la"
  }
}

data "alz_archetype" "sandboxes" {
  id             = "sandboxes"
  parent_id      = data.alz_archetype.root.id
  base_archetype = "sandboxes"
  defaults = {
    location = "westeurope"
    log_analytics_workspace_id = "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg/providers/Microsoft.OperationalInsights/workspaces/la"
  }
}

data "alz_archetype" "connectivity" {
  id             = "connectivity"
  parent_id      = data.alz_archetype.platform.id
  base_archetype = "connectivity"
  defaults = {
    location = "westeurope"
    log_analytics_workspace_id = "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg/providers/Microsoft.OperationalInsights/workspaces/la"
  }
}

data "alz_archetype" "identity" {
  id             = "identity"
  parent_id      = data.alz_archetype.platform.id
  base_archetype = "identity"
  defaults = {
    location = "westeurope"
    log_analytics_workspace_id = "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg/providers/Microsoft.OperationalInsights/workspaces/la"
  }
}

data "alz_archetype" "management" {
  id             = "management"
  parent_id      = data.alz_archetype.platform.id
  base_archetype = "management"
  defaults = {
    location = "westeurope"
    log_analytics_workspace_id = "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg/providers/Microsoft.OperationalInsights/workspaces/la"
  }
}

data "alz_archetype" "corp" {
  id             = "corp"
  parent_id      = data.alz_archetype.landing_zones.id
  base_archetype = "corp"
  defaults = {
    location = "westeurope"
    log_analytics_workspace_id = "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg/providers/Microsoft.OperationalInsights/workspaces/la"
  }
}

data "alz_archetype" "online" {
  id             = "online"
  parent_id      = data.alz_archetype.landing_zones.id
  base_archetype = "online"
  defaults = {
    location = "westeurope"
    log_analytics_workspace_id = "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg/providers/Microsoft.OperationalInsights/workspaces/la"
  }
}

# Test that the data source is returning the correct value for the policy location
output "test" {
  value = {
    root          = data.alz_archetype.root
    landing_zones = data.alz_archetype.landing_zones
    platform      = data.alz_archetype.platform
    sandboxes     = data.alz_archetype.sandboxes
    connectivity  = data.alz_archetype.connectivity
    identity      = data.alz_archetype.identity
    management    = data.alz_archetype.management
    corp          = data.alz_archetype.corp
    online        = data.alz_archetype.online
  }
}
`
}

func TestConvertPolicyAssignmentParametersToSdkType(t *testing.T) {
	// Test with nil input
	res := convertPolicyAssignmentParametersToSdkType(nil)
	assert.Nil(t, res)

	// Test with empty input
	res = convertPolicyAssignmentParametersToSdkType(make(map[string]interface{}))
	assert.NotNil(t, res)
	assert.Empty(t, res)

	// Test with non-empty input
	src := map[string]interface{}{
		"param1": "value1",
		"param2": 123,
		"param3": true,
	}
	res = convertPolicyAssignmentParametersToSdkType(src)
	assert.NotNil(t, res)
	assert.Len(t, res, len(src))
	for k, v := range src {
		assert.Contains(t, res, k)
		assert.Equal(t, v, res[k].Value)
	}
}

func TestConvertAlzPolicyRoleAssignments(t *testing.T) {
	// Test with nil input
	res := convertAlzPolicyRoleAssignments(nil)
	assert.Nil(t, res)
	assert.Empty(t, res)

	// Test with empty input
	res = convertAlzPolicyRoleAssignments(make([]alzlib.PolicyRoleAssignment, 0))
	assert.Nil(t, res)
	assert.Empty(t, res)

	// Test with non-empty input
	src := []alzlib.PolicyRoleAssignment{
		{
			RoleDefinitionId: "test1",
			Scope:            "test1",
			AssignmentName:   "test1",
		},
	}
	res = convertAlzPolicyRoleAssignments(src)
	assert.NotNil(t, res)
	assert.Len(t, res, len(src))
	for _, v := range src {
		key := genPolicyRoleAssignmentId(v)
		assert.Equal(t, v.RoleDefinitionId, res[key].RoleDefinitionId.ValueString())
		assert.Equal(t, v.Scope, res[key].Scope.ValueString())
		assert.Equal(t, v.AssignmentName, res[key].AssignmentName.ValueString())
	}
}

// TestPolicyAssignmentType2ArmPolicyValues tests the policyAssignmentType2ArmPolicyValues function.
func TestPolicyAssignmentType2ArmPolicyValues(t *testing.T) {
	paramsIn, _ := alztypes.PolicyParameterType{}.ValueFromString(context.Background(), types.StringValue(`{
		"param1": "value1",
		"param2": 123,
		"param3": true
	}`))
	pa := PolicyAssignmentType{
		EnforcementMode: types.StringValue("DoNotEnforce"),
		NonComplianceMessage: []PolicyAssignmentNonComplianceMessage{
			{
				Message:                     types.StringValue("Non-compliance message 1"),
				PolicyDefinitionReferenceId: types.StringValue("PolicyDefinition1"),
			},
			{
				Message:                     types.StringValue("Non-compliance message 2"),
				PolicyDefinitionReferenceId: types.StringValue("PolicyDefinition2"),
			},
		},
		Parameters: paramsIn.(alztypes.PolicyParameterValue),
	}

	enforcementMode, identity, nonComplianceMessages, parameters, err := policyAssignmentType2ArmPolicyValues(pa)

	assert.NoError(t, err)
	assert.Equal(t, armpolicy.EnforcementModeDoNotEnforce, *enforcementMode)
	assert.Nil(t, identity)
	assert.Len(t, nonComplianceMessages, 2)
	assert.Equal(t, "Non-compliance message 1", *nonComplianceMessages[0].Message)
	assert.Equal(t, "PolicyDefinition1", *nonComplianceMessages[0].PolicyDefinitionReferenceID)
	assert.Equal(t, "Non-compliance message 2", *nonComplianceMessages[1].Message)
	assert.Equal(t, "PolicyDefinition2", *nonComplianceMessages[1].PolicyDefinitionReferenceID)
	assert.Len(t, parameters, 3)
	assert.Equal(t, "value1", parameters["param1"].Value)
	assert.Equal(t, float64(123), parameters["param2"].Value)
	assert.Equal(t, true, parameters["param3"].Value)
}
