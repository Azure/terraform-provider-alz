package provider

import (
	"context"
	"testing"

	"github.com/Azure/alzlib/deployment"
	"github.com/Azure/alzlib/to"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armpolicy"
	"github.com/Azure/terraform-provider-alz/internal/provider/gen"
	mapset "github.com/deckarep/golang-set/v2"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/stretchr/testify/assert"
)

// TestAccAlzArchetypeDataSource tests the data source for alz_archetype.
// It checks that the policy parameter substitution & location defaults are applied.
func TestAccAlzArchitectureDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactoriesUnique(),
		ExternalProviders: map[string]resource.ExternalProvider{
			"azapi": {
				Source:            "azure/azapi",
				VersionConstraint: "~> 1.14",
			},
		},
		Steps: []resource.TestStep{
			{
				Config: testAccArchitectureDataSourceConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.alz_architecture.test", "id", "alz"),
					// resource.TestCheckOutput("test_location_replacement", "westeurope"),
					// resource.TestCheckOutput("test_parameter_replacement", "test"),
				),
			},
		},
	})
}

// testAccArchitectureDataSourceConfig returns a test configuration for TestAccAlzArchetypeDataSource.
func testAccArchitectureDataSourceConfig() string {
	// cwd, _ := os.Getwd()
	// libPath := filepath.Join(cwd, "testdata/testacc_lib")

	return `
provider "alz" {
  library_references = [
  {
	  path = "platform/alz"
		ref  = "2024.07.02"
	}
	]
}

data "azapi_client_config" "current" {}

data "alz_architecture" "test" {
  name                     = "alz"
	root_management_group_id = data.azapi_client_config.current.tenant_id
	location                 = "northeurope"

	timeouts {
		read = "5m"
	}
}
`
}

// TestConvertPolicyAssignmentResourceSelectorsToSdkType tests the conversion of policy assignment resource selectors from framework to Azure Go SDK types.
func TestConvertPolicyAssignmentResourceSelectorsToSdkType(t *testing.T) {
	ctx := context.Background()

	rs1s1in, _ := basetypes.NewSetValueFrom(ctx, types.StringType, []string{"in1", "in2"})
	rs1s1notIn, _ := basetypes.NewSetValueFrom(ctx, types.StringType, []string{"notin1", "notin2"})
	rs1s2in, _ := basetypes.NewSetValueFrom(ctx, types.StringType, []string{"in3", "in4"})
	rs1s2notIn, _ := basetypes.NewSetValueFrom(ctx, types.StringType, []string{"notin3", "notin4"})
	rs2s1in, _ := basetypes.NewSetValueFrom(ctx, types.StringType, []string{"in5", "in6"})
	rs2s1notIn, _ := basetypes.NewSetValueFrom(ctx, types.StringType, []string{"notin5", "notin6"})

	notSetStringType, _ := basetypes.NewSetValueFrom(ctx, types.BoolType, []bool{true})
	t.Run("EmptyInput", func(t *testing.T) {
		src := []gen.ResourceSelectorsValue{}
		res, diags := convertPolicyAssignmentResourceSelectorsToSdkType(ctx, src)
		assert.False(t, diags.HasError())
		assert.Nil(t, res)
	})

	t.Run("NonEmptyInput", func(t *testing.T) {
		src := []gen.ResourceSelectorsValue{
			{
				Name: types.StringValue("selector1"),
				ResourceSelectorSelectors: types.ListValueMust(gen.NewResourceSelectorSelectorsValueNull().Type(ctx), []attr.Value{
					gen.ResourceSelectorSelectorsValue{
						Kind:  types.StringValue("kind1"),
						In:    rs1s1in,
						NotIn: rs1s1notIn,
					},
					gen.ResourceSelectorSelectorsValue{
						Kind:  types.StringValue("kind2"),
						In:    rs1s2in,
						NotIn: rs1s2notIn,
					},
				}),
			},
			{
				Name: types.StringValue("selector2"),
				ResourceSelectorSelectors: types.ListValueMust(gen.NewResourceSelectorSelectorsValueNull().Type(ctx), []attr.Value{
					gen.ResourceSelectorSelectorsValue{
						Kind:  types.StringValue("kind3"),
						In:    rs2s1in,
						NotIn: rs2s1notIn,
					},
				}),
			},
		}

		expected := []*armpolicy.ResourceSelector{
			{
				Name: to.Ptr("selector1"),
				Selectors: []*armpolicy.Selector{
					{
						Kind:  to.Ptr(armpolicy.SelectorKind("kind1")),
						In:    to.SliceOfPtrs("in1", "in2"),
						NotIn: to.SliceOfPtrs("notin1", "notin2"),
					},
					{
						Kind:  to.Ptr(armpolicy.SelectorKind("kind2")),
						In:    to.SliceOfPtrs("in3", "in4"),
						NotIn: to.SliceOfPtrs("notin3", "notin4"),
					},
				},
			},
			{
				Name: to.Ptr("selector2"),
				Selectors: []*armpolicy.Selector{
					{
						Kind:  to.Ptr(armpolicy.SelectorKind("kind3")),
						In:    to.SliceOfPtrs("in5", "in6"),
						NotIn: to.SliceOfPtrs("notin5", "notin6"),
					},
				},
			},
		}

		res, diags := convertPolicyAssignmentResourceSelectorsToSdkType(ctx, src)
		assert.False(t, diags.HasError())
		assert.Equal(t, expected, res)
	})

	t.Run("ConversionError", func(t *testing.T) {
		src := []gen.ResourceSelectorsValue{
			{
				Name: types.StringValue("selector1"),
				ResourceSelectorSelectors: types.ListValueMust(gen.NewResourceSelectorSelectorsValueNull().Type(ctx), []attr.Value{
					gen.ResourceSelectorSelectorsValue{
						Kind: types.StringValue("kind1"),
						In:   notSetStringType,
					},
				}),
			},
		}

		// Simulate an error during conversion
		res, diags := convertPolicyAssignmentResourceSelectorsToSdkType(ctx, src)
		assert.True(t, diags.HasError())
		assert.Nil(t, res)
	})
}

// TestConvertPolicyAssignmentIdentityToSdkType tests the conversion of policy assignment identity from framework to Azure Go SDK types.
func TestConvertPolicyAssignmentIdentityToSdkType(t *testing.T) {
	// Test with unknown identity type
	typ := types.StringValue("UnknownType")
	ids := basetypes.NewSetUnknown(types.StringType)
	identity, diags := convertPolicyAssignmentIdentityToSdkType(typ, ids)
	assert.Nil(t, identity)
	assert.True(t, diags.HasError())

	// Test with SystemAssigned identity type
	typ = types.StringValue("SystemAssigned")
	ids = basetypes.NewSetNull(types.StringType)
	identity, diags = convertPolicyAssignmentIdentityToSdkType(typ, ids)
	assert.NotNil(t, identity)
	assert.False(t, diags.HasError())
	assert.Equal(t, armpolicy.ResourceIdentityTypeSystemAssigned, *identity.Type)

	// Test with UserAssigned identity type and empty ids
	typ = types.StringValue("UserAssigned")
	ids = basetypes.NewSetNull(types.StringType)
	identity, diags = convertPolicyAssignmentIdentityToSdkType(typ, ids)
	assert.Nil(t, identity)
	assert.True(t, diags.HasError())

	// Test with UserAssigned identity type and multiple ids
	typ = types.StringValue("UserAssigned")
	ids, _ = types.SetValueFrom(context.Background(), types.StringType, []string{"id1", "id2"})
	identity, diags = convertPolicyAssignmentIdentityToSdkType(typ, ids)
	assert.Nil(t, identity)
	assert.True(t, diags.HasError())

	// Test with UserAssigned identity type and valid id
	typ = types.StringValue("UserAssigned")
	ids, _ = types.SetValueFrom(context.Background(), types.StringType, []string{"id1"})
	identity, diags = convertPolicyAssignmentIdentityToSdkType(typ, ids)
	assert.NotNil(t, identity)
	assert.False(t, diags.HasError())
	assert.Equal(t, armpolicy.ResourceIdentityTypeUserAssigned, *identity.Type)
	assert.Len(t, identity.UserAssignedIdentities, 1)
	assert.Contains(t, identity.UserAssignedIdentities, "id1")
}

// TestConvertPolicyAssignmentNonComplianceMessagesToSdkType tests the the conversion of policy assignment non-compliance messages from framework to Azure Go SDK types.
func TestConvertPolicyAssignmentNonComplianceMessagesToSdkType(t *testing.T) {
	src := []gen.NonComplianceMessagesValue{
		{
			Message:                     types.StringValue("message1"),
			PolicyDefinitionReferenceId: types.StringValue("policy1"),
		},
		{
			Message: types.StringValue("message2"),
		},
	}

	expected := []*armpolicy.NonComplianceMessage{
		{
			Message:                     to.Ptr("message1"),
			PolicyDefinitionReferenceID: to.Ptr("policy1"),
		},
		{
			Message: to.Ptr("message2"),
		},
	}

	result := convertPolicyAssignmentNonComplianceMessagesToSdkType(src)
	assert.Equal(t, expected, result)
}

// TestConvertPolicyAssignmentEnforcementModeToSdkType tests the conversion of policy assignment enforcement mode from framework to Azure Go SDK types.
func TestConvertPolicyAssignmentEnforcementModeToSdkType(t *testing.T) {
	// Test with unknown enforcement mode
	src := types.StringValue("Unknown")
	res := convertPolicyAssignmentEnforcementModeToSdkType(src)
	assert.Nil(t, res)

	// Test with DoNotEnforce enforcement mode
	src = types.StringValue("DoNotEnforce")
	res = convertPolicyAssignmentEnforcementModeToSdkType(src)
	assert.NotNil(t, res)
	assert.Equal(t, armpolicy.EnforcementModeDoNotEnforce, *res)

	// Test with Default enforcement mode
	src = types.StringValue("Default")
	res = convertPolicyAssignmentEnforcementModeToSdkType(src)
	assert.NotNil(t, res)
	assert.Equal(t, armpolicy.EnforcementModeDefault, *res)
}

// TestConvertPolicyAssignmentParametersToSdkType tests the convertPolicyAssignmentParametersToSdkType function.
func TestConvertPolicyAssignmentParametersToSdkType(t *testing.T) {
	// Test with nil input
	var src types.Map
	var res map[string]*armpolicy.ParameterValuesValue
	res, diags := convertPolicyAssignmentParametersMapToSdkType(src)
	assert.False(t, diags.HasError())
	assert.Nil(t, res)

	// Test with empty input
	src = types.MapNull(types.StringType)
	res, diags = convertPolicyAssignmentParametersMapToSdkType(src)
	assert.False(t, diags.HasError())
	assert.Nil(t, res)

	param1 := armpolicy.ParameterValuesValue{
		Value: to.Ptr("value1"),
	}
	param2 := armpolicy.ParameterValuesValue{
		Value: to.Ptr(123),
	}
	param3 := armpolicy.ParameterValuesValue{
		Value: to.Ptr(true),
	}
	param1Json, _ := param1.MarshalJSON()
	param2Json, _ := param2.MarshalJSON()
	param3Json, _ := param3.MarshalJSON()
	src, _ = types.MapValueFrom(context.Background(), types.StringType, map[string]string{
		"param1": string(param1Json),
		"param2": string(param2Json),
		"param3": string(param3Json),
	})

	res, diags = convertPolicyAssignmentParametersMapToSdkType(src)
	assert.False(t, diags.HasError())
	assert.NotNil(t, res)
	assert.Len(t, res, 3)
	assert.Equal(t, "value1", res["param1"].Value)
	assert.Equal(t, float64(123), res["param2"].Value)
	assert.Equal(t, true, res["param3"].Value)
}

func TestPolicyAssignmentType2ArmPolicyValues(t *testing.T) {
	ctx := context.Background()
	param1 := armpolicy.ParameterValuesValue{
		Value: to.Ptr("value1"),
	}
	param2 := armpolicy.ParameterValuesValue{
		Value: to.Ptr(123),
	}
	param3 := armpolicy.ParameterValuesValue{
		Value: to.Ptr(true),
	}
	param1Json, _ := param1.MarshalJSON()
	param2Json, _ := param2.MarshalJSON()
	param3Json, _ := param3.MarshalJSON()
	params, _ := types.MapValueFrom(ctx, types.StringType, map[string]string{
		"param1": string(param1Json),
		"param2": string(param2Json),
		"param3": string(param3Json),
	})
	pa := gen.PolicyAssignmentsValue{ //nolint:forcetypeassert
		EnforcementMode: types.StringValue("DoNotEnforce"),
		NonComplianceMessages: types.SetValueMust(
			gen.NewNonComplianceMessagesValueNull().Type(ctx),
			[]attr.Value{
				gen.NonComplianceMessagesValue{
					Message:                     types.StringValue("Non-compliance message 1"),
					PolicyDefinitionReferenceId: types.StringValue("PolicyDefinition1"),
				},
				gen.NonComplianceMessagesValue{
					Message:                     types.StringValue("Non-compliance message 2"),
					PolicyDefinitionReferenceId: types.StringValue("PolicyDefinition2"),
				},
			}),
		Parameters: params,
	}

	enforcementMode, identity, nonComplianceMessages, parameters, _, _, diags := policyAssignmentType2ArmPolicyValues(ctx, pa)

	assert.False(t, diags.HasError())
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

func TestPolicyRoleAssignmentsSetToProviderType(t *testing.T) {
	ctx := context.Background()
	// Test with nil input
	res, diags := policyRoleAssignmentsSetToProviderType(ctx, nil)
	assert.False(t, diags.HasError())
	assert.Empty(t, len(res.Elements()))

	// Test with empty input
	res, diags = policyRoleAssignmentsSetToProviderType(ctx, make([]deployment.PolicyRoleAssignment, 0))
	assert.False(t, diags.HasError())
	assert.Empty(t, len(res.Elements()))

	// Test with non-empty input
	src := mapset.NewThreadUnsafeSet[deployment.PolicyRoleAssignment](
		deployment.PolicyRoleAssignment{
			RoleDefinitionId: "test1",
			Scope:            "test1",
			AssignmentName:   "test1",
		},
	)
	res, _ = policyRoleAssignmentsSetToProviderType(ctx, src.ToSlice())
	assert.NotNil(t, res)
	assert.Len(t, res.Elements(), src.Cardinality())
	for _, v := range res.Elements() {
		praval := v.(gen.PolicyRoleAssignmentsValue) //nolint:forcetypeassert
		setMember := deployment.PolicyRoleAssignment{
			RoleDefinitionId: praval.RoleDefinitionId.ValueString(),
			Scope:            praval.Scope.ValueString(),
			AssignmentName:   praval.PolicyAssignmentName.ValueString(),
		}
		assert.True(t, src.Contains(setMember))
	}
}
