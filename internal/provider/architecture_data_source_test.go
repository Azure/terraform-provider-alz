package provider

import (
	"testing"

	"github.com/Azure/alzlib/deployment"
	"github.com/Azure/alzlib/to"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armpolicy"
	"github.com/Azure/terraform-provider-alz/internal/provider/gen"
	mapset "github.com/deckarep/golang-set/v2"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/stretchr/testify/assert"
)

// TestAccAlzArchitectureDataSourceRemoteLib tests the data source for alz_architecture
// when using a remote lib.
func TestAccAlzArchitectureDataSourceRemoteLib(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactoriesUnique(),
		ExternalProviders: map[string]resource.ExternalProvider{
			"azapi": {
				Source:            "azure/azapi",
				VersionConstraint: "~> 2.0",
			},
		},
		Steps: []resource.TestStep{
			{
				Config: testAccArchitectureDataSourceConfigRemoteLib(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.alz_architecture.test", "id", "alz"),
				),
			},
		},
	})
}

// TestAccAlzArchitectureDataSourceRemoteLib tests the data source for alz_architecture
// when using a remote lib.
func TestAccAlzArchitectureDataSourceRetainRoleDefinitionNames(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactoriesUnique(),
		ExternalProviders: map[string]resource.ExternalProvider{
			"azapi": {
				Source:            "azure/azapi",
				VersionConstraint: "~> 2.0",
			},
		},
		Steps: []resource.TestStep{
			{
				Config: testAccArchitectureDataSourceConfigWithStaticRoleDefinitionNames(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckOutput("role_definition_name", "c9a07a05-a1fc-53fe-a565-5eed25597c03"),
					resource.TestCheckOutput("role_definition_role_name", "Application-Owners"),
				),
			},
		},
	})
}

// TestAccAlzArchetypeDataSource tests the data source for alz_archetype.
// It checks that the policy default values and the modification of policy assignments are correctly applied.
func TestAccAlzArchitectureDataSourceWithDefaultAndModify(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactoriesUnique(),
		ExternalProviders: map[string]resource.ExternalProvider{
			"azapi": {
				Source:            "azure/azapi",
				VersionConstraint: "~> 2.0",
			},
		},
		Steps: []resource.TestStep{
			{
				Config: testAccArchitectureDataSourceConfigWithDefaultAndModify(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.alz_architecture.test", "id", "test"),
					resource.TestCheckOutput("log_analytics_replaced_by_policy_default_values", "replacedByDefaults"),
					resource.TestCheckOutput("metrics_enabled_modified", "false"),
					resource.TestCheckOutput("identity_type", "UserAssigned"),
					resource.TestCheckOutput("identity_id", "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/test-rg/providers/Microsoft.ManagedIdentity/userAssignedIdentities/test-identity"),
					resource.TestCheckOutput("policy_assignment_override_kind", "policyEffect"),
					resource.TestCheckOutput("policy_assignment_override_value", "disabled"),
					resource.TestCheckOutput("policy_assignment_override_selector_kind", "policyDefinitionReferenceId"),
					resource.TestCheckOutput("policy_assignment_override_selector_in", "test-policy-definition"),
					resource.TestCheckOutput("policy_assignment_non_compliance_message", "testnoncompliancemessage"),
					resource.TestCheckOutput("policy_assignment_resource_selector_name", "test-resource-selector"),
					resource.TestCheckOutput("policy_assignment_resource_selector_kind", "resourceLocation"),
					resource.TestCheckOutput("policy_assignment_resource_selector_in", "northeurope"),
					resource.TestCheckOutput("policy_assignment_resource_selector_notin_should_be_null", "true"),
				),
			},
		},
	})
}

// TestAccAlzArchetypeDataSource tests the data source for alz_archetype.
// It checks that the policy default values and the modification of policy assignments are correctly applied.
func TestAccAlzArchitectureDataSourceExistingMg(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactoriesUnique(),
		ExternalProviders: map[string]resource.ExternalProvider{
			"azapi": {
				Source:            "azure/azapi",
				VersionConstraint: "~> 2.0",
			},
		},
		Steps: []resource.TestStep{
			{
				Config: testAccArchitectureDataSourceConfigExistingMg(),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownOutputValue("management_group_exists", knownvalue.Bool(true)),
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.alz_architecture.test", "id", "existingmg"),
				),
			},
		},
	})
}

func TestAccAlzArchitectureDataSourceModifyPolicyAssignmentNonExistent(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactoriesUnique(),
		ExternalProviders: map[string]resource.ExternalProvider{
			"azapi": {
				Source:            "azure/azapi",
				VersionConstraint: "~> 2.0",
			},
		},
		Steps: []resource.TestStep{
			{
				Config: testAccArchitectureDataSourceConfigModifyPolicyAssignmentNonExistent(),
			},
		},
	})
}

func TestAccAlzArchitectureDataSourceAssignPermissionsOverride(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactoriesUnique(),
		ExternalProviders: map[string]resource.ExternalProvider{
			"azapi": {
				Source:            "azure/azapi",
				VersionConstraint: "~> 2.0",
			},
		},
		Steps: []resource.TestStep{
			{
				Config: testAccArchitectureDataSourceConfigOverrideAssignPermissions(),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownOutputValue("pra", knownvalue.Bool(true)),
				},
			},
		},
	})
}

// testAccArchitectureDataSourceConfigRemoteLib returns a test configuration for TestAccAlzArchetypeDataSource.
func testAccArchitectureDataSourceConfigRemoteLib() string {
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

// testAccArchitectureDataSourceConfigRemoteLib returns a test configuration for TestAccAlzArchetypeDataSource.
func testAccArchitectureDataSourceConfigWithStaticRoleDefinitionNames() string {
	return `
provider "alz" {
  role_definitions_use_supplied_names_enabled = true
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

output "role_definition_name" {
  value = jsondecode(data.alz_architecture.test.management_groups[0].role_definitions["Application-Owners"]).name
}

output "role_definition_role_name" {
  value = jsondecode(data.alz_architecture.test.management_groups[0].role_definitions["Application-Owners"]).properties.roleName
}
`
}

// testAccArchitectureDataSourceConfigWithDefaultAndModify returns a test configuration for TestAccAlzArchetypeDataSource.
func testAccArchitectureDataSourceConfigWithDefaultAndModify() string {
	return `
provider "alz" {
  library_references = [
    {
	    custom_url = "${path.root}/testdata/testacc_lib"
	  }
	]
}

data "azapi_client_config" "current" {}

data "alz_architecture" "test" {
  name                     = "test"
	root_management_group_id = data.azapi_client_config.current.tenant_id
	location                 = "northeurope"
	policy_default_values    = {
	  test = jsonencode({ value = "replacedByDefaults" })
	}
	policy_assignments_to_modify = {
	  test = {
		  policy_assignments = {
			  test-policy-assignment = {
				  identity = "UserAssigned"
					identity_ids = [
					  "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/test-rg/providers/Microsoft.ManagedIdentity/userAssignedIdentities/test-identity"
					]
					non_compliance_messages = [
						{
							message = "testnoncompliancemessage"
						}
					]
					parameters = {
						metricsEnabled = jsonencode({ value = false })
					}
					resource_selectors	 = [
						{
							name = "test-resource-selector"
							resource_selector_selectors = [
							  {
							    kind = "resourceLocation"
								  in   = ["northeurope"]
							  }
							]
						}
					]
					overrides = [
						{
							kind = "policyEffect"
							value = "disabled"
							override_selectors = [
								{
									kind = "policyDefinitionReferenceId"
									in   = ["test-policy-definition"]
								}
							]
						}
					]
				}
			}
		}
	}

	timeouts {
		read = "5m"
	}
}

locals {
	test_policy_assignment_decoded = jsondecode(data.alz_architecture.test.management_groups[0].policy_assignments["test-policy-assignment"])
}

output "log_analytics_replaced_by_policy_default_values" {
	value = local.test_policy_assignment_decoded.properties.parameters.logAnalytics.value
}

output "metrics_enabled_modified" {
	value = tostring(local.test_policy_assignment_decoded.properties.parameters.metricsEnabled.value)
}

output "identity_type" {
	value = local.test_policy_assignment_decoded.identity.type
}

output "identity_id" {
	value = keys(local.test_policy_assignment_decoded.identity.userAssignedIdentities)[0]
}

output "policy_assignment_override_kind" {
	value = local.test_policy_assignment_decoded.properties.overrides[0].kind
}

output "policy_assignment_override_value" {
	value = local.test_policy_assignment_decoded.properties.overrides[0].value
}

output "policy_assignment_override_selector_kind" {
	value = local.test_policy_assignment_decoded.properties.overrides[0].selectors[0].kind
}

output "policy_assignment_override_selector_in" {
	value = local.test_policy_assignment_decoded.properties.overrides[0].selectors[0].in[0]
}

output "policy_assignment_non_compliance_message" {
	value = local.test_policy_assignment_decoded.properties.nonComplianceMessages[0].message
}

output "policy_assignment_resource_selector_name" {
	value = local.test_policy_assignment_decoded.properties.resourceSelectors[0].name
}

output "policy_assignment_resource_selector_kind" {
	value = local.test_policy_assignment_decoded.properties.resourceSelectors[0].selectors[0].kind
}

output "policy_assignment_resource_selector_in" {
	value = local.test_policy_assignment_decoded.properties.resourceSelectors[0].selectors[0].in[0]
}

output "policy_assignment_resource_selector_notin_should_be_null" {
	value = lookup(local.test_policy_assignment_decoded.properties.resourceSelectors[0].selectors[0], "notIn", null) == null
}
`
}

func testAccArchitectureDataSourceConfigExistingMg() string {
	return `
provider "alz" {
	library_references = [
		{
			custom_url = "${path.root}/testdata/existingmg"
    }
	]
}

data "azapi_client_config" "current" {}

data "alz_architecture" "test" {
	name                     = "existingmg"
	root_management_group_id = data.azapi_client_config.current.tenant_id
	location                 = "northeurope"
}

output "management_group_exists" {
	value = data.alz_architecture.test.management_groups[0].exists
}
`
}

func testAccArchitectureDataSourceConfigModifyPolicyAssignmentNonExistent() string {
	return `
provider "alz" {
  library_references = [
    {
	    custom_url = "${path.root}/testdata/testacc_lib"
	  }
	]
}

data "azapi_client_config" "current" {}

data "alz_architecture" "test" {
  name                     = "test"
  root_management_group_id = data.azapi_client_config.current.tenant_id
  location                 = "swedencentral"
  policy_assignments_to_modify = {
    not_exist = {
      policy_assignments = {
        Deploy-MDEndpoints = {
          enforcement_mode = "DoNotEnforce"
        }
      }
    }
  }
}
`
}

func testAccArchitectureDataSourceConfigOverrideAssignPermissions() string {
	return `
provider "alz" {
	library_references = [
		{
			custom_url = "${path.root}/testdata/overrideAssignPermissions"
    }
	]
}

data "azapi_client_config" "current" {}

data "alz_architecture" "test" {
	name                     = "test"
	root_management_group_id = data.azapi_client_config.current.tenant_id
	location                 = "northeurope"
	override_policy_definition_parameter_assign_permissions_set = [
		{
			definition_name = "test-policy-definition"
			parameter_name  = "logAnalytics"
		}
	]
}

locals {
	test = anytrue([
	  for val in data.alz_architecture.test.policy_role_assignments : strcontains(val.scope, "Microsoft.OperationalInsights/workspaces/PLACEHOLDER")
	])
}

output "pra" {
	value = local.test
}
`
}

// TestConvertPolicyAssignmentResourceSelectorsToSdkType tests the conversion of policy assignment resource selectors from framework to Azure Go SDK types.
func TestConvertPolicyAssignmentResourceSelectorsToSdkType(t *testing.T) {
	ctx := t.Context()

	rs1s1in, _ := basetypes.NewSetValueFrom(ctx, types.StringType, []string{"in1", "in2"})
	rs1s1notIn, _ := basetypes.NewSetValueFrom(ctx, types.StringType, []string{"notin1", "notin2"})
	rs1s2in, _ := basetypes.NewSetValueFrom(ctx, types.StringType, []string{"in3", "in4"})
	rs1s2notIn, _ := basetypes.NewSetValueFrom(ctx, types.StringType, []string{"notin3", "notin4"})
	rs2s1in, _ := basetypes.NewSetValueFrom(ctx, types.StringType, []string{"in5", "in6"})
	rs2s1notIn, _ := basetypes.NewSetValueFrom(ctx, types.StringType, []string{"notin5", "notin6"})

	notSetStringType, _ := basetypes.NewSetValueFrom(ctx, types.BoolType, []bool{true})
	t.Run("EmptyInput", func(t *testing.T) {
		resp := new(datasource.ReadResponse)
		resp.Diagnostics = diag.Diagnostics{}
		src := []gen.ResourceSelectorsValue{}
		res := convertPolicyAssignmentResourceSelectorsToSdkType(ctx, src, resp)
		assert.False(t, resp.Diagnostics.HasError())
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
		resp := new(datasource.ReadResponse)
		resp.Diagnostics = diag.Diagnostics{}
		res := convertPolicyAssignmentResourceSelectorsToSdkType(ctx, src, resp)
		assert.False(t, resp.Diagnostics.HasError())
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
		resp := new(datasource.ReadResponse)
		resp.Diagnostics = diag.Diagnostics{}
		res := convertPolicyAssignmentResourceSelectorsToSdkType(ctx, src, resp)
		assert.True(t, resp.Diagnostics.HasError())
		assert.Nil(t, res)
	})
}

// TestConvertPolicyAssignmentIdentityToSdkType tests the conversion of policy assignment identity from framework to Azure Go SDK types.
func TestConvertPolicyAssignmentIdentityToSdkType(t *testing.T) {
	// Test with unknown identity type
	typ := types.StringValue("UnknownType")
	ids := basetypes.NewSetUnknown(types.StringType)
	resp := new(datasource.ReadResponse)
	resp.Diagnostics = diag.Diagnostics{}
	identity := convertPolicyAssignmentIdentityToSdkType(typ, ids, resp)
	assert.Nil(t, identity)
	assert.True(t, resp.Diagnostics.HasError())
	resp.Diagnostics = diag.Diagnostics{}

	// Test with SystemAssigned identity type
	typ = types.StringValue("SystemAssigned")
	ids = basetypes.NewSetNull(types.StringType)
	identity = convertPolicyAssignmentIdentityToSdkType(typ, ids, resp)
	assert.NotNil(t, identity)
	assert.False(t, resp.Diagnostics.HasError())
	assert.Equal(t, armpolicy.ResourceIdentityTypeSystemAssigned, *identity.Type)

	// Test with UserAssigned identity type and empty ids
	typ = types.StringValue("UserAssigned")
	ids = basetypes.NewSetNull(types.StringType)
	identity = convertPolicyAssignmentIdentityToSdkType(typ, ids, resp)
	assert.Nil(t, identity)
	assert.True(t, resp.Diagnostics.HasError())
	resp.Diagnostics = diag.Diagnostics{}

	// Test with UserAssigned identity type and multiple ids
	typ = types.StringValue("UserAssigned")
	ids, _ = types.SetValueFrom(t.Context(), types.StringType, []string{"id1", "id2"})
	identity = convertPolicyAssignmentIdentityToSdkType(typ, ids, resp)
	assert.Nil(t, identity)
	assert.True(t, resp.Diagnostics.HasError())
	resp.Diagnostics = diag.Diagnostics{}

	// Test with UserAssigned identity type and valid id
	typ = types.StringValue("UserAssigned")
	ids, _ = types.SetValueFrom(t.Context(), types.StringType, []string{"id1"})
	identity = convertPolicyAssignmentIdentityToSdkType(typ, ids, resp)
	assert.NotNil(t, identity)
	assert.False(t, resp.Diagnostics.HasError())
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
	resp := new(datasource.ReadResponse)
	resp.Diagnostics = diag.Diagnostics{}
	res = convertPolicyAssignmentParametersMapToSdkType(src, resp)
	assert.False(t, resp.Diagnostics.HasError())
	assert.Nil(t, res)

	// Test with empty input
	src = types.MapNull(types.StringType)
	res = convertPolicyAssignmentParametersMapToSdkType(src, resp)
	assert.False(t, resp.Diagnostics.HasError())
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
	src, _ = types.MapValueFrom(t.Context(), types.StringType, map[string]string{
		"param1": string(param1Json),
		"param2": string(param2Json),
		"param3": string(param3Json),
	})

	res = convertPolicyAssignmentParametersMapToSdkType(src, resp)
	assert.False(t, resp.Diagnostics.HasError())
	assert.NotNil(t, res)
	assert.Len(t, res, 3)
	assert.Equal(t, "value1", res["param1"].Value)
	assert.Equal(t, float64(123), res["param2"].Value)
	assert.Equal(t, true, res["param3"].Value)
}

func TestPolicyAssignmentType2ArmPolicyValues(t *testing.T) {
	ctx := t.Context()
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
	resp := new(datasource.ReadResponse)
	resp.Diagnostics = diag.Diagnostics{}
	enforcementMode, identity, nonComplianceMessages, parameters, _, _ := policyAssignmentType2ArmPolicyValues(ctx, pa, resp)

	assert.False(t, resp.Diagnostics.HasError())
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
	ctx := t.Context()
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
