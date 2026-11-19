package services_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/Azure/alzlib/assets"
	"github.com/Azure/alzlib/cache"
	"github.com/Azure/alzlib/to"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armpolicy"
	"github.com/Azure/terraform-provider-alz/internal/acceptance"
	"github.com/Azure/terraform-provider-alz/internal/clients"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/stretchr/testify/require"
)

// testAccCacheFileBuiltInName is the name of the built-in policy definition assigned by the `testdata/cachefile`
// libraries. It does not exist in Azure, so it can only be loaded from the cache file.
const testAccCacheFileBuiltInName = "00000000-0000-0000-0000-000000000001"

// TestAccAlzArchitectureDataSourceRemoteLib tests the data source for alz_architecture
// when using a remote lib.
func TestAccAlzArchitectureDataSourceRemoteLib(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.AccTestPreCheck(t) },
		ProtoV6ProviderFactories: acceptance.AccTestProtoV6ProviderFactoriesUnique(),
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
		PreCheck:                 func() { acceptance.AccTestPreCheck(t) },
		ProtoV6ProviderFactories: acceptance.AccTestProtoV6ProviderFactoriesUnique(),
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
		PreCheck:                 func() { acceptance.AccTestPreCheck(t) },
		ProtoV6ProviderFactories: acceptance.AccTestProtoV6ProviderFactoriesUnique(),
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
					resource.TestCheckOutput("policy_assignment_not_scopes", "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/test-rg-excluded"),
				),
			},
		},
	})
}

// TestAccAlzArchetypeDataSource tests the data source for alz_archetype.
// It checks that the policy default values and the modification of policy assignments are correctly applied.
func TestAccAlzArchitectureDataSourceExistingMg(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.AccTestPreCheck(t) },
		ProtoV6ProviderFactories: acceptance.AccTestProtoV6ProviderFactoriesUnique(),
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
		PreCheck:                 func() { acceptance.AccTestPreCheck(t) },
		ProtoV6ProviderFactories: acceptance.AccTestProtoV6ProviderFactoriesUnique(),
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
		PreCheck:                 func() { acceptance.AccTestPreCheck(t) },
		ProtoV6ProviderFactories: acceptance.AccTestProtoV6ProviderFactoriesUnique(),
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

func TestAccArchitectureDataSourceMultipleProviders(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.AccTestPreCheck(t) },
		ProtoV6ProviderFactories: acceptance.AccTestProtoV6ProviderFactoriesUnique(),
		ExternalProviders: map[string]resource.ExternalProvider{
			"azapi": {
				Source:            "azure/azapi",
				VersionConstraint: "~> 2.0",
			},
		},
		Steps: []resource.TestStep{
			{
				Config:            testAccArchitectureDataSourceMultipleProviders(),
				ConfigStateChecks: []statecheck.StateCheck{},
			},
		},
	})
}

// TestAccAlzArchitectureDataSourceNonComplianceMessageDefault tests the default non-compliance message feature.
func TestAccAlzArchitectureDataSourceNonComplianceMessageDefault(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.AccTestPreCheck(t) },
		ProtoV6ProviderFactories: acceptance.AccTestProtoV6ProviderFactoriesUnique(),
		ExternalProviders: map[string]resource.ExternalProvider{
			"azapi": {
				Source:            "azure/azapi",
				VersionConstraint: "~> 2.0",
			},
		},
		Steps: []resource.TestStep{
			{
				Config: testAccArchitectureDataSourceConfigNonComplianceMessageDefault(),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Policy without message should have default applied with "must" (Default mode)
					resource.TestCheckOutput("policy_without_message_nc_message", "This resource must be compliant."),
					// Policy with DoNotEnforce should have "should" in the message
					resource.TestCheckOutput("policy_donotenforce_nc_message", "This resource should be compliant."),
					// Overwrite mode (default): policy-specific message preserved, default message overwritten
					resource.TestCheckOutput("policy_with_message_nc_count", "2"),
					// The policy-specific message (with policyDefinitionReferenceId) is preserved
					resource.TestCheckOutput("policy_with_message_nc_policy_specific", "Message for specific policy definition"),
					// The default message is the new one (old default overwritten)
					resource.TestCheckOutput("policy_with_message_nc_default", "This resource must be compliant."),
					// Policy assignment referencing a resource provider mode definition should have no non-compliance messages
					resource.TestCheckOutput("policy_rp_mode_nc_empty", "true"),
				),
			},
		},
	})
}

// TestAccAlzArchitectureDataSourceNonComplianceMessagePreferExisting tests the prefer_existing merge mode.
func TestAccAlzArchitectureDataSourceNonComplianceMessagePreferExisting(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.AccTestPreCheck(t) },
		ProtoV6ProviderFactories: acceptance.AccTestProtoV6ProviderFactoriesUnique(),
		ExternalProviders: map[string]resource.ExternalProvider{
			"azapi": {
				Source:            "azure/azapi",
				VersionConstraint: "~> 2.0",
			},
		},
		Steps: []resource.TestStep{
			{
				Config: testAccArchitectureDataSourceConfigNonComplianceMessagePreferExisting(),
				Check: resource.ComposeAggregateTestCheckFunc(
					// prefer_existing: existing default message is kept, policy-specific preserved
					resource.TestCheckOutput("policy_with_message_nc_count", "2"),
					resource.TestCheckOutput("policy_with_message_nc_policy_specific", "Message for specific policy definition"),
					// Existing default is preserved (not replaced)
					resource.TestCheckOutput("policy_with_message_nc_default", "Existing non-compliance message"),
					// Policy without message should still get the configured default
					resource.TestCheckOutput("policy_without_message_nc_message", "This resource must be compliant."),
				),
			},
		},
	})
}

// TestAccAlzArchitectureDataSourceNonComplianceMessageCustomSubstitution tests custom placeholder and replacement values.
func TestAccAlzArchitectureDataSourceNonComplianceMessageCustomSubstitution(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.AccTestPreCheck(t) },
		ProtoV6ProviderFactories: acceptance.AccTestProtoV6ProviderFactoriesUnique(),
		ExternalProviders: map[string]resource.ExternalProvider{
			"azapi": {
				Source:            "azure/azapi",
				VersionConstraint: "~> 2.0",
			},
		},
		Steps: []resource.TestStep{
			{
				Config: testAccArchitectureDataSourceConfigNonComplianceMessageCustomSubstitution(),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Policy without message should have default applied with custom enforced replacement
					resource.TestCheckOutput("policy_without_message_nc_message", "This resource needs to be compliant."),
					// Policy with DoNotEnforce should have custom not-enforced replacement
					resource.TestCheckOutput("policy_donotenforce_nc_message", "This resource has to be compliant."),
					// Policy-specific message preserved, default message uses custom replacement
					resource.TestCheckOutput("policy_with_message_nc_count", "2"),
					resource.TestCheckOutput("policy_with_message_nc_policy_specific", "Message for specific policy definition"),
					resource.TestCheckOutput("policy_with_message_nc_default", "This resource needs to be compliant."),
				),
			},
		},
	})
}

// TestAccAlzArchitectureDataSourceLibraryOverwrite tests that, with `library_overwrite_enabled`, a policy definition
// and a policy set definition redefined by a later library replace those supplied by an earlier library.
// The libraries only contain custom assets, so no Azure API requests are made.
func TestAccAlzArchitectureDataSourceLibraryOverwrite(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.AccTestPreCheck(t) },
		ProtoV6ProviderFactories: acceptance.AccTestProtoV6ProviderFactoriesUnique(),
		Steps: []resource.TestStep{
			{
				Config: testAccArchitectureDataSourceConfigLibraryOverwrite(true, "override"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckOutput("policy_definition_display_name", "Overridden policy definition"),
					resource.TestCheckOutput("policy_definition_effect_default", "Deny"),
					resource.TestCheckOutput("policy_set_definition_display_name", "Overridden policy set definition"),
					resource.TestCheckOutput("policy_set_definition_version", "1.0.0"),
					resource.TestCheckOutput("policy_set_definition_effect_default", "Deny"),
				),
			},
		},
	})
}

// TestAccAlzArchitectureDataSourceLibraryOverwriteDisabled tests that redefining an existing asset in a later
// library is an error when `library_overwrite_enabled` is not set.
func TestAccAlzArchitectureDataSourceLibraryOverwriteDisabled(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.AccTestPreCheck(t) },
		ProtoV6ProviderFactories: acceptance.AccTestProtoV6ProviderFactoriesUnique(),
		Steps: []resource.TestStep{
			{
				Config:      testAccArchitectureDataSourceConfigLibraryOverwrite(false, "override"),
				ExpectError: regexp.MustCompile(`(?s)Failed to initialize AlzLib.*test-policy-definition`),
			},
		},
	})
}

// TestAccAlzArchitectureDataSourceLibraryNewPolicySetDefinitionVersion tests that a new version of a policy set
// definition supplied by a later library is retained when `library_overwrite_enabled` is not set.
func TestAccAlzArchitectureDataSourceLibraryNewPolicySetDefinitionVersion(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.AccTestPreCheck(t) },
		ProtoV6ProviderFactories: acceptance.AccTestProtoV6ProviderFactoriesUnique(),
		Steps: []resource.TestStep{
			{
				Config: testAccArchitectureDataSourceConfigLibraryOverwrite(false, "newversion"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckOutput("policy_definition_display_name", "Base policy definition"),
					resource.TestCheckOutput("policy_set_definition_display_name", "Policy set definition version 1.1.0"),
					resource.TestCheckOutput("policy_set_definition_version", "1.1.0"),
				),
			},
		},
	})
}

// TestAccAlzArchitectureDataSourceCacheFileSave tests that, with `cache_file_save_enabled`, the built-in definitions
// loaded by reading an architecture are saved to the cache file and loaded by the next run, and that a cache file
// saved for other libraries is not loaded (Azure/Azure-Landing-Zones#4328).
// The assigned built-in policy definition does not exist in Azure, so the architecture can only be read when the
// definition is loaded from the cache file. Each Terraform command configures a new provider instance, as separate
// runs do.
func TestAccAlzArchitectureDataSourceCacheFileSave(t *testing.T) {
	cacheFile := filepath.Join(t.TempDir(), "alzlib-cache.json.gz")
	var fingerprint string

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.AccTestPreCheck(t) },
		ProtoV6ProviderFactories: acceptance.AccTestProtoV6ProviderFactoriesPerCommand(),
		Steps: []resource.TestStep{
			{
				// A cache file without metadata, such as one created by alzlibtool, is loaded.
				PreConfig: func() { testAccWriteCacheFile(t, cacheFile) },
				Config:    testAccArchitectureDataSourceConfigCacheFile(cacheFile, "base"),
				Check: testAccCheckCacheFileSaved(cacheFile, func(saved string) error {
					fingerprint = saved
					return nil
				}),
			},
			{
				// The cache file saved by the previous run is loaded.
				Config: testAccArchitectureDataSourceConfigCacheFile(cacheFile, "base"),
				Check: testAccCheckCacheFileSaved(cacheFile, func(saved string) error {
					if saved != fingerprint {
						return fmt.Errorf("the libraries have not changed, but the fingerprint changed from %s to %s", fingerprint, saved)
					}
					return nil
				}),
			},
			{
				// The cache file was saved for other libraries, so it is not loaded and the definition is looked up in Azure.
				Config:      testAccArchitectureDataSourceConfigCacheFile(cacheFile, "base", "extra"),
				ExpectError: regexp.MustCompile(`(?s)Error creating architecture.*` + testAccCacheFileBuiltInName),
			},
			{
				// Without the metadata, the cache file is loaded, and it is saved for the new libraries.
				PreConfig: func() { require.NoError(t, os.Remove(clients.CacheFileMetadataPath(cacheFile))) },
				Config:    testAccArchitectureDataSourceConfigCacheFile(cacheFile, "base", "extra"),
				Check: testAccCheckCacheFileSaved(cacheFile, func(saved string) error {
					if saved == fingerprint {
						return errors.New("the libraries have changed, but the fingerprint has not")
					}
					return nil
				}),
			},
		},
	})
}

// TestAccAlzArchitectureDataSourceAssignPermissionsOverrideParameterNameCase tests that the parameter name of an
// assignPermissions override is matched case-insensitively.
func TestAccAlzArchitectureDataSourceAssignPermissionsOverrideParameterNameCase(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.AccTestPreCheck(t) },
		ProtoV6ProviderFactories: acceptance.AccTestProtoV6ProviderFactoriesUnique(),
		Steps: []resource.TestStep{
			{
				Config: testAccArchitectureDataSourceConfigOverrideAssignPermissionsParameterNameCase(),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownOutputValue("pra", knownvalue.Bool(true)),
				},
			},
		},
	})
}

// TestAccAlzArchitectureDataSourceModifyPolicyAssignmentParameterNameCase tests that a parameter in
// `policy_assignments_to_modify` whose name differs only in case updates the existing parameter instead of
// adding a second one.
func TestAccAlzArchitectureDataSourceModifyPolicyAssignmentParameterNameCase(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.AccTestPreCheck(t) },
		ProtoV6ProviderFactories: acceptance.AccTestProtoV6ProviderFactoriesUnique(),
		Steps: []resource.TestStep{
			{
				Config: testAccArchitectureDataSourceConfigModifyPolicyAssignmentParameterNameCase(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckOutput("parameter_names", "logAnalytics"),
					resource.TestCheckOutput("log_analytics", "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/test-rg/providers/Microsoft.OperationalInsights/workspaces/modified"),
				),
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
					not_scopes = [
						"/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/test-rg-excluded"
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

output "policy_assignment_not_scopes" {
	value = local.test_policy_assignment_decoded.properties.notScopes[0]
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

// testAccArchitectureDataSourceMultipleProviders returns a test configuration for TestAccArchitectureDataSourceMultipleProviders.
func testAccArchitectureDataSourceMultipleProviders() string {
	return `
provider "alz" {
  library_references = [
  {
		custom_url = "${path.root}/testdata/testacc_lib"
	}
	]
}

provider "alz" {
  alias = "secondary"
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


	timeouts {
		read = "5m"
	}
}

data "alz_architecture" "test2" {
	provider                 = alz.secondary
  name                     = "test"
	root_management_group_id = data.azapi_client_config.current.tenant_id
	location                 = "northeurope"


	timeouts {
		read = "5m"
	}
}
`
}

func testAccArchitectureDataSourceConfigNonComplianceMessageDefault() string {
	return `
provider "alz" {
  library_references = [
    {
      custom_url = "${path.root}/testdata/noncompliancemsg"
    }
  ]
}

data "azapi_client_config" "current" {}

data "alz_architecture" "test" {
  name                     = "test"
  root_management_group_id = data.azapi_client_config.current.tenant_id
  location                 = "northeurope"

  default_non_compliance_message_settings = {
    default_message = "This resource {enforcementMode} be compliant."
  }

  timeouts {
    read = "5m"
  }
}

locals {
  policy_without_message = jsondecode(data.alz_architecture.test.management_groups[0].policy_assignments["policy-without-message"])
  policy_with_message    = jsondecode(data.alz_architecture.test.management_groups[0].policy_assignments["policy-with-message"])
  policy_donotenforce    = jsondecode(data.alz_architecture.test.management_groups[0].policy_assignments["policy-donotenforce"])
  policy_rp_mode         = jsondecode(data.alz_architecture.test.management_groups[0].policy_assignments["policy-rp-mode"])
}

output "policy_without_message_nc_message" {
  value = local.policy_without_message.properties.nonComplianceMessages[0].message
}

output "policy_donotenforce_nc_message" {
  value = local.policy_donotenforce.properties.nonComplianceMessages[0].message
}

output "policy_with_message_nc_count" {
  value = tostring(length(local.policy_with_message.properties.nonComplianceMessages))
}

output "policy_with_message_nc_policy_specific" {
  value = one([for m in local.policy_with_message.properties.nonComplianceMessages : m.message if try(m.policyDefinitionReferenceId, "") != ""])
}

output "policy_with_message_nc_default" {
  value = one([for m in local.policy_with_message.properties.nonComplianceMessages : m.message if try(m.policyDefinitionReferenceId, "") == ""])
}

output "policy_rp_mode_nc_empty" {
  value = tostring(length(try(local.policy_rp_mode.properties.nonComplianceMessages, [])) == 0)
}
`
}

func testAccArchitectureDataSourceConfigNonComplianceMessagePreferExisting() string {
	return `
provider "alz" {
  library_references = [
    {
      custom_url = "${path.root}/testdata/noncompliancemsg"
    }
  ]
}

data "azapi_client_config" "current" {}

data "alz_architecture" "test" {
  name                                       = "test"
  root_management_group_id                   = data.azapi_client_config.current.tenant_id
  location                                   = "northeurope"

  default_non_compliance_message_settings = {
    default_message = "This resource {enforcementMode} be compliant."
    merge_mode      = "prefer_existing"
  }

  timeouts {
    read = "5m"
  }
}

locals {
  policy_without_message = jsondecode(data.alz_architecture.test.management_groups[0].policy_assignments["policy-without-message"])
  policy_with_message    = jsondecode(data.alz_architecture.test.management_groups[0].policy_assignments["policy-with-message"])
}

output "policy_without_message_nc_message" {
  value = local.policy_without_message.properties.nonComplianceMessages[0].message
}

output "policy_with_message_nc_count" {
  value = tostring(length(local.policy_with_message.properties.nonComplianceMessages))
}

output "policy_with_message_nc_policy_specific" {
  value = one([for m in local.policy_with_message.properties.nonComplianceMessages : m.message if try(m.policyDefinitionReferenceId, "") != ""])
}

output "policy_with_message_nc_default" {
  value = one([for m in local.policy_with_message.properties.nonComplianceMessages : m.message if try(m.policyDefinitionReferenceId, "") == ""])
}
`
}

func testAccArchitectureDataSourceConfigNonComplianceMessageCustomSubstitution() string {
	return `
provider "alz" {
  library_references = [
    {
      custom_url = "${path.root}/testdata/noncompliancemsg"
    }
  ]

  non_compliance_message_substitution_settings = {
    enforcement_mode_placeholder = "{MODE}"
    enforced_replacement         = "needs to"
    not_enforced_replacement     = "has to"
  }
}

data "azapi_client_config" "current" {}

data "alz_architecture" "test" {
  name                     = "test"
  root_management_group_id = data.azapi_client_config.current.tenant_id
  location                 = "northeurope"

  default_non_compliance_message_settings = {
    default_message = "This resource {MODE} be compliant."
  }

  timeouts {
    read = "5m"
  }
}

locals {
  policy_without_message = jsondecode(data.alz_architecture.test.management_groups[0].policy_assignments["policy-without-message"])
  policy_with_message    = jsondecode(data.alz_architecture.test.management_groups[0].policy_assignments["policy-with-message"])
  policy_donotenforce    = jsondecode(data.alz_architecture.test.management_groups[0].policy_assignments["policy-donotenforce"])
}

output "policy_without_message_nc_message" {
  value = local.policy_without_message.properties.nonComplianceMessages[0].message
}

output "policy_donotenforce_nc_message" {
  value = local.policy_donotenforce.properties.nonComplianceMessages[0].message
}

output "policy_with_message_nc_count" {
  value = tostring(length(local.policy_with_message.properties.nonComplianceMessages))
}

output "policy_with_message_nc_policy_specific" {
  value = one([for m in local.policy_with_message.properties.nonComplianceMessages : m.message if try(m.policyDefinitionReferenceId, "") != ""])
}

output "policy_with_message_nc_default" {
  value = one([for m in local.policy_with_message.properties.nonComplianceMessages : m.message if try(m.policyDefinitionReferenceId, "") == ""])
}
`
}

// testAccArchitectureDataSourceConfigLibraryOverwrite returns a test configuration that combines the `base` library
// with the supplied later library, both from `testdata/libraryoverwrite`.
func testAccArchitectureDataSourceConfigLibraryOverwrite(overwrite bool, laterLibrary string) string {
	return fmt.Sprintf(`
provider "alz" {
  library_overwrite_enabled = %t
  library_references = [
    {
      custom_url = "${path.root}/testdata/libraryoverwrite/base"
    },
    {
      custom_url = "${path.root}/testdata/libraryoverwrite/%s"
    }
  ]
}

data "alz_architecture" "test" {
  name                     = "test"
  root_management_group_id = "00000000-0000-0000-0000-000000000000"
  location                 = "northeurope"
}

locals {
  policy_definition     = jsondecode(data.alz_architecture.test.management_groups[0].policy_definitions["test-policy-definition"])
  policy_set_definition = jsondecode(data.alz_architecture.test.management_groups[0].policy_set_definitions["test-policy-set-definition"])
}

output "policy_definition_display_name" {
  value = local.policy_definition.properties.displayName
}

output "policy_definition_effect_default" {
  value = local.policy_definition.properties.parameters.effect.defaultValue
}

output "policy_set_definition_display_name" {
  value = local.policy_set_definition.properties.displayName
}

output "policy_set_definition_version" {
  value = local.policy_set_definition.properties.version
}

output "policy_set_definition_effect_default" {
  value = local.policy_set_definition.properties.parameters.effect.defaultValue
}
`, overwrite, laterLibrary)
}

// testAccArchitectureDataSourceConfigCacheFile returns a test configuration that saves the built-in definitions to
// the supplied cache file and uses the supplied libraries from `testdata/cachefile`.
func testAccArchitectureDataSourceConfigCacheFile(cacheFile string, libraries ...string) string {
	refs := make([]string, len(libraries))
	for i, library := range libraries {
		refs[i] = fmt.Sprintf(`{ custom_url = "${path.root}/testdata/cachefile/%s" }`, library)
	}

	return fmt.Sprintf(`
provider "alz" {
  cache_file_name         = %q
  cache_file_save_enabled = true
  library_references      = [%s]
}

data "alz_architecture" "test" {
  name                     = "test"
  root_management_group_id = "00000000-0000-0000-0000-000000000000"
  location                 = "northeurope"

  timeouts {
    read = "1m"
  }
}
`, filepath.ToSlash(cacheFile), strings.Join(refs, ", "))
}

// testAccWriteCacheFile writes a cache file without metadata, as alzlibtool does, that holds the built-in policy
// definition assigned by the `testdata/cachefile` libraries.
func testAccWriteCacheFile(t *testing.T, cacheFile string) {
	t.Helper()

	pdvs := assets.NewPolicyDefinitionVersions()
	require.NoError(t, pdvs.Add(&assets.PolicyDefinition{
		Definition: armpolicy.Definition{
			Name: to.Ptr(testAccCacheFileBuiltInName),
			Properties: &armpolicy.DefinitionProperties{
				Description: to.Ptr("Built-in policy definition that only exists in the cache file."),
				DisplayName: to.Ptr("Cached built-in policy definition"),
				Metadata:    map[string]any{},
				Mode:        to.Ptr("All"),
				PolicyRule:  map[string]any{"if": map[string]any{}, "then": map[string]any{}},
				PolicyType:  to.Ptr(armpolicy.PolicyTypeBuiltIn),
				Version:     to.Ptr("1.0.0"),
			},
		},
	}, false))

	f, err := os.Create(cacheFile)
	require.NoError(t, err)
	require.NoError(t, cache.NewCacheFromDefinitions(
		map[string]*assets.PolicyDefinitionVersions{testAccCacheFileBuiltInName: pdvs},
		map[string]*assets.PolicySetDefinitionVersions{},
	).Save(f))
	require.NoError(t, f.Close())
}

// testAccCheckCacheFileSaved checks that the cache file holds the assigned built-in policy definition, and passes the
// library fingerprint recorded in the metadata file to checkFingerprint.
func testAccCheckCacheFileSaved(cacheFile string, checkFingerprint func(string) error) resource.TestCheckFunc {
	return func(*terraform.State) error {
		f, err := os.Open(cacheFile)
		if err != nil {
			return err
		}
		defer f.Close()

		c, err := cache.NewCache(f)
		if err != nil {
			return err
		}
		if c.PolicyDefinitionVersionsByName(testAccCacheFileBuiltInName) == nil {
			return fmt.Errorf("the cache file does not hold the built-in policy definition %s", testAccCacheFileBuiltInName)
		}

		content, err := os.ReadFile(clients.CacheFileMetadataPath(cacheFile))
		if err != nil {
			return err
		}
		var meta struct {
			LibraryFingerprint string `json:"libraryFingerprint"`
		}
		if err := json.Unmarshal(content, &meta); err != nil {
			return err
		}
		if meta.LibraryFingerprint == "" {
			return errors.New("the cache file metadata does not record the libraries")
		}

		return checkFingerprint(meta.LibraryFingerprint)
	}
}

func testAccArchitectureDataSourceConfigOverrideAssignPermissionsParameterNameCase() string {
	return `
provider "alz" {
  library_references = [
    {
      custom_url = "${path.root}/testdata/overrideAssignPermissions"
    }
  ]
}

data "alz_architecture" "test" {
  name                     = "test"
  root_management_group_id = "00000000-0000-0000-0000-000000000000"
  location                 = "northeurope"

  override_policy_definition_parameter_assign_permissions_set = [
    {
      definition_name = "test-policy-definition"
      parameter_name  = "LOGANALYTICS"
    }
  ]
}

output "pra" {
  value = anytrue([
    for val in data.alz_architecture.test.policy_role_assignments : strcontains(val.scope, "Microsoft.OperationalInsights/workspaces/PLACEHOLDER")
  ])
}
`
}

func testAccArchitectureDataSourceConfigModifyPolicyAssignmentParameterNameCase() string {
	return `
provider "alz" {
  library_references = [
    {
      custom_url = "${path.root}/testdata/overrideAssignPermissions"
    }
  ]
}

data "alz_architecture" "test" {
  name                     = "test"
  root_management_group_id = "00000000-0000-0000-0000-000000000000"
  location                 = "northeurope"

  policy_assignments_to_modify = {
    test = {
      policy_assignments = {
        test-policy-assignment = {
          parameters = {
            LogAnalytics = jsonencode({ value = "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/test-rg/providers/Microsoft.OperationalInsights/workspaces/modified" })
          }
        }
      }
    }
  }
}

locals {
  policy_assignment = jsondecode(data.alz_architecture.test.management_groups[0].policy_assignments["test-policy-assignment"])
}

output "parameter_names" {
  value = join(",", sort(keys(local.policy_assignment.properties.parameters)))
}

output "log_analytics" {
  value = local.policy_assignment.properties.parameters.logAnalytics.value
}
`
}
