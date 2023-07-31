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
	mapset "github.com/deckarep/golang-set/v2"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/matt-FFFFFF/terraform-provider-alz/internal/alztypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAccAlzArchetypeDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
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
					resource.TestCheckResourceAttr("data.alz_archetype.test", "alz_policy_assignments.%", "1"),
					resource.TestCheckOutput("test", "westeurope"),
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
		lib_dirs = [
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

		// policy_assignments_to_add = {
		// 	myassign = {
		// 		display_name 		       = random_pet.test.id
		// 		policy_definition_id = "/subscriptions/00000000-0000-0000-0000-000000000000/providers/Microsoft.Authorization/policyDefinitions/1234"
		// 		non_compliance_message = [
		// 			{
		// 				message = random_pet.test.id
		// 			},
		// 			{
		// 				message                        = "test2"
		// 				policy_definition_reference_id = "1234"
		// 			}
		// 		]
		// 		parameters = jsonencode({
		// 			myparam  = "test"
		// 			myparam2 = 2
		// 		})
		// 	}
		// }
	}

	# Test that the data source is returning the correct value for the policy location
	output "test" {
		value = jsondecode(data.alz_archetype.test.alz_policy_assignments["BlobServicesDiagnosticsLogsToWorkspace"]).location
	}
	`, libPath)
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
	res, diags := convertAlzPolicyRoleAssignments(context.Background(), nil)
	assert.NotNil(t, res)
	assert.Empty(t, res)
	assert.Empty(t, diags)

	// Test with empty input
	res, diags = convertAlzPolicyRoleAssignments(context.Background(), make(map[string]alzlib.PolicyAssignmentAdditionalRoleAssignments))
	assert.NotNil(t, res)
	assert.Empty(t, res)
	assert.Empty(t, diags)

	// Test with non-empty input
	src := map[string]alzlib.PolicyAssignmentAdditionalRoleAssignments{
		"assignment1": {
			RoleDefinitionIds: []string{"role1", "role2"},
			AdditionalScopes:  []string{"scope1", "scope2"},
		},
	}
	res, diags = convertAlzPolicyRoleAssignments(context.Background(), src)
	assert.NotNil(t, res)
	assert.Len(t, res, len(src))
	assert.Empty(t, diags)
	for k, v := range src {
		assert.Contains(t, res, k)
		assert.Len(t, res[k].RoleDefinitionIds.Elements(), len(v.RoleDefinitionIds))
		assert.Len(t, res[k].AdditionalScopes.Elements(), len(v.AdditionalScopes))
		for i, rd := range v.RoleDefinitionIds {
			assert.Contains(t, res[k].RoleDefinitionIds.Elements(), types.StringValue(rd))
			assert.Equal(t, rd, res[k].RoleDefinitionIds.Elements()[i].(basetypes.StringValue).ValueString())
		}
		for i, as := range v.AdditionalScopes {
			assert.Contains(t, res[k].AdditionalScopes.Elements(), types.StringValue(as))
			assert.Equal(t, as, res[k].AdditionalScopes.Elements()[i].(basetypes.StringValue).ValueString())
		}
	}
}

func TestPolicyAssignmentType2ArmPolicyAssignment(t *testing.T) {
	t.Parallel()

	az := alzlib.NewAlzLib()

	az.Init(context.Background(), os.DirFS("testdata/testacc_lib"))

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
						DisplayName:           to.Ptr(""),
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
						DisplayName:           to.Ptr(""),
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
						DisplayName:           to.Ptr(""),
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
						DisplayName:        to.Ptr(""),
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
						DisplayName:           to.Ptr(""),
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
