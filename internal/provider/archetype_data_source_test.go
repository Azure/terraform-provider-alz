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
	"github.com/Azure/alzlib/to"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armpolicy"
	"github.com/Azure/terraform-provider-alz/internal/alztypes"
	mapset "github.com/deckarep/golang-set/v2"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
					resource.TestCheckResourceAttr("data.alz_archetype.root", "alz_policy_definitions.%", "126"),
					resource.TestCheckResourceAttr("data.alz_archetype.root", "alz_policy_set_definitions.%", "13"),
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

  policy_assignments_to_add = {
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

// TestAddAttrStringElementsToSet tests that addAttrStringElementsToSet adds a value to a set.
func TestAddAttrStringElementsToSet(t *testing.T) {
	arch := &alzlib.Archetype{
		PolicyDefinitions: mapset.NewThreadUnsafeSet[string]("a", "b", "c"),
	}
	vals := []attr.Value{
		basetypes.NewStringValue("d"),
	}
	assert.NoError(t, addAttrStringElementsToSet(arch.PolicyDefinitions, vals))
	assert.True(t, arch.PolicyDefinitions.Contains("d"))
}

// TestDeleteAttrStringElementsToSet tests that deleteAttrStringElementsToSet removes a value from a set.
func TestDeleteAttrStringElementsToSet(t *testing.T) {
	arch := &alzlib.Archetype{
		PolicyDefinitions: mapset.NewThreadUnsafeSet[string]("a", "b", "c"),
	}
	vals := []attr.Value{
		basetypes.NewStringValue("c"),
	}
	assert.NoError(t, deleteAttrStringElementsFromSet(arch.PolicyDefinitions, vals))
	assert.True(t, !arch.PolicyDefinitions.Contains("c"))
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

func TestPolicyAssignmentType2ArmPolicyAssignment(t *testing.T) {
	t.Parallel()

	az := alzlib.NewAlzLib()

	require.NoError(t, az.Init(context.Background(), os.DirFS("testdata/testacc_lib")))

	testCases := []struct {
		name     string
		input    map[string]PolicyAssignmentType
		expected map[string]*armpolicy.Assignment
		err      error
	}{
		{
			name:     "empty input",
			input:    map[string]PolicyAssignmentType{},
			expected: map[string]*armpolicy.Assignment{},
			err:      nil,
		},
		{
			name: "policy definition id and display name",
			input: map[string]PolicyAssignmentType{
				"test1": {
					PolicyDefinitionName: types.StringValue("BlobServicesDiagnosticsLogsToWorkspace"),
					DisplayName:          types.StringValue("BlobServicesDiagnosticsLogsToWorkspace"),
				},
			},
			expected: map[string]*armpolicy.Assignment{
				"test1": {
					ID:   to.Ptr("/providers/Microsoft.Management/managementGroups/placeholder/providers/Microsoft.Authorization/policyAssignments/test1"),
					Name: to.Ptr("test1"),
					Type: to.Ptr("Microsoft.Authorization/policyAssignments"),
					Properties: &armpolicy.AssignmentProperties{
						DisplayName:           to.Ptr("BlobServicesDiagnosticsLogsToWorkspace"),
						PolicyDefinitionID:    to.Ptr("/providers/Microsoft.Management/managementGroups/placeholder/providers/Microsoft.Authorization/policyDefinitions/BlobServicesDiagnosticsLogsToWorkspace"),
						EnforcementMode:       nil,
						NonComplianceMessages: []*armpolicy.NonComplianceMessage(nil),
						Parameters:            map[string]*armpolicy.ParameterValuesValue(nil),
					},
				},
			},
			err: nil,
		},
		{
			name: "policy definition id",
			input: map[string]PolicyAssignmentType{
				"test1": {
					PolicyDefinitionName: types.StringValue("BlobServicesDiagnosticsLogsToWorkspace"),
				},
			},
			expected: map[string]*armpolicy.Assignment{
				"test1": {
					ID:   to.Ptr("/providers/Microsoft.Management/managementGroups/placeholder/providers/Microsoft.Authorization/policyAssignments/test1"),
					Name: to.Ptr("test1"),
					Type: to.Ptr("Microsoft.Authorization/policyAssignments"),
					Properties: &armpolicy.AssignmentProperties{
						DisplayName:           nil,
						PolicyDefinitionID:    to.Ptr("/providers/Microsoft.Management/managementGroups/placeholder/providers/Microsoft.Authorization/policyDefinitions/BlobServicesDiagnosticsLogsToWorkspace"),
						EnforcementMode:       nil,
						NonComplianceMessages: []*armpolicy.NonComplianceMessage(nil),
						Parameters:            map[string]*armpolicy.ParameterValuesValue(nil),
					},
				},
			},
			err: nil,
		},
		{
			name: "policy set definition id",
			input: map[string]PolicyAssignmentType{
				"test2": {
					PolicySetDefinitionName: types.StringValue("test"),
				},
			},
			expected: map[string]*armpolicy.Assignment{
				"test2": {
					ID:   to.Ptr("/providers/Microsoft.Management/managementGroups/placeholder/providers/Microsoft.Authorization/policyAssignments/test2"),
					Name: to.Ptr("test2"),
					Type: to.Ptr("Microsoft.Authorization/policyAssignments"),
					Properties: &armpolicy.AssignmentProperties{
						DisplayName:           nil,
						PolicyDefinitionID:    to.Ptr("/providers/Microsoft.Management/managementGroups/placeholder/providers/Microsoft.Authorization/policySetDefinitions/test"),
						EnforcementMode:       nil,
						NonComplianceMessages: []*armpolicy.NonComplianceMessage(nil),
						Parameters:            map[string]*armpolicy.ParameterValuesValue(nil),
					},
				},
			},
			err: nil,
		},
		{
			name: "policy definition id and enforcement mode",
			input: map[string]PolicyAssignmentType{
				"test3": {
					PolicyDefinitionName: types.StringValue("BlobServicesDiagnosticsLogsToWorkspace"),
					EnforcementMode:      types.StringValue("DoNotEnforce"),
				},
			},
			expected: map[string]*armpolicy.Assignment{
				"test3": {
					ID:   to.Ptr("/providers/Microsoft.Management/managementGroups/placeholder/providers/Microsoft.Authorization/policyAssignments/test3"),
					Name: to.Ptr("test3"),
					Type: to.Ptr("Microsoft.Authorization/policyAssignments"),
					Properties: &armpolicy.AssignmentProperties{
						DisplayName:           nil,
						PolicyDefinitionID:    to.Ptr("/providers/Microsoft.Management/managementGroups/placeholder/providers/Microsoft.Authorization/policyDefinitions/BlobServicesDiagnosticsLogsToWorkspace"),
						EnforcementMode:       to.Ptr(armpolicy.EnforcementModeDoNotEnforce),
						NonComplianceMessages: []*armpolicy.NonComplianceMessage(nil),
						Parameters:            map[string]*armpolicy.ParameterValuesValue(nil),
					},
				},
			},
			err: nil,
		},
		{
			name: "policy definition id and non-compliance message",
			input: map[string]PolicyAssignmentType{
				"test4": {
					PolicyDefinitionName: types.StringValue("BlobServicesDiagnosticsLogsToWorkspace"),
					NonComplianceMessage: []PolicyAssignmentNonComplianceMessage{
						{
							Message: types.StringValue("test message"),
						},
					},
				},
			},
			expected: map[string]*armpolicy.Assignment{
				"test4": {
					ID:   to.Ptr("/providers/Microsoft.Management/managementGroups/placeholder/providers/Microsoft.Authorization/policyAssignments/test4"),
					Name: to.Ptr("test4"),
					Type: to.Ptr("Microsoft.Authorization/policyAssignments"),
					Properties: &armpolicy.AssignmentProperties{
						DisplayName:        nil,
						PolicyDefinitionID: to.Ptr("/providers/Microsoft.Management/managementGroups/placeholder/providers/Microsoft.Authorization/policyDefinitions/BlobServicesDiagnosticsLogsToWorkspace"),
						EnforcementMode:    nil,
						NonComplianceMessages: []*armpolicy.NonComplianceMessage{
							{
								Message:                     to.Ptr("test message"),
								PolicyDefinitionReferenceID: nil,
							},
						},
						Parameters: map[string]*armpolicy.ParameterValuesValue(nil),
					},
				},
			},
			err: nil,
		},
		{
			name: "policy definition id and parameters",
			input: map[string]PolicyAssignmentType{
				"test5": {
					PolicyDefinitionName: types.StringValue("BlobServicesDiagnosticsLogsToWorkspace"),
					Parameters: alztypes.PolicyParameterValue{
						StringValue: types.StringValue(`{"param1": "value1", "param2": 2}`),
					},
				},
			},
			expected: map[string]*armpolicy.Assignment{
				"test5": {
					ID:   to.Ptr("/providers/Microsoft.Management/managementGroups/placeholder/providers/Microsoft.Authorization/policyAssignments/test5"),
					Name: to.Ptr("test5"),
					Type: to.Ptr("Microsoft.Authorization/policyAssignments"),
					Properties: &armpolicy.AssignmentProperties{
						DisplayName:           nil,
						PolicyDefinitionID:    to.Ptr("/providers/Microsoft.Management/managementGroups/placeholder/providers/Microsoft.Authorization/policyDefinitions/BlobServicesDiagnosticsLogsToWorkspace"),
						EnforcementMode:       nil,
						NonComplianceMessages: []*armpolicy.NonComplianceMessage(nil),
						Parameters: map[string]*armpolicy.ParameterValuesValue{
							"param1": {Value: "value1"},
							"param2": {Value: float64(2)},
						},
					},
				},
			},
			err: nil,
		},
		{
			name: "unknown policy definition name",
			input: map[string]PolicyAssignmentType{
				"test6": {
					PolicyDefinitionName: types.StringValue("unknown"),
				},
			},
			expected: nil,
			err:      fmt.Errorf("policy definition unknown not found in AlzLib"),
		},
		{
			name: "unknown policy set definition name",
			input: map[string]PolicyAssignmentType{
				"test7": {
					PolicySetDefinitionName: types.StringValue("unknown"),
				},
			},
			expected: nil,
			err:      fmt.Errorf("policy set definition unknown not found in AlzLib"),
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			actual, err := policyAssignmentType2ArmPolicyAssignment(tc.input, az)
			require.Equal(t, tc.err, err)
			if tc.err != nil {
				return
			}
			for ek, ev := range tc.expected {
				assert.Equal(t, *ev.ID, *actual[ek].ID)
				assert.Equal(t, *ev.Name, *actual[ek].Name)
				assert.Equal(t, *ev.Type, *actual[ek].Type)
				assert.Equal(t, ev.Properties.DisplayName, actual[ek].Properties.DisplayName)
				assert.Equal(t, *ev.Properties.PolicyDefinitionID, *actual[ek].Properties.PolicyDefinitionID)
				assert.Equal(t, ev.Properties.EnforcementMode, actual[ek].Properties.EnforcementMode)
				assert.Equal(t, ev.Properties.NonComplianceMessages, actual[ek].Properties.NonComplianceMessages)
				assert.Equal(t, ev.Properties.Parameters, actual[ek].Properties.Parameters)
			}
			//assert.Equal(t, tc.expected, actual)
		})
	}
}
