// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package provider

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/Azure/alzlib"
	sets "github.com/deckarep/golang-set/v2"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/stretchr/testify/assert"
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
		PolicyDefinitions: sets.NewSet[string]("a", "b", "c"),
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
		PolicyDefinitions: sets.NewSet[string]("a", "b", "c"),
	}
	vals := []attr.Value{
		basetypes.NewStringValue("c"),
	}
	assert.NoError(t, deleteAttrStringElementsFromSet(arch.PolicyDefinitions, vals))
	assert.True(t, !arch.PolicyDefinitions.Contains("c"))
}
