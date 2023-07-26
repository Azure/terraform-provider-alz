// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package provider

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/matt-FFFFFF/alzlib"
	"github.com/matt-FFFFFF/alzlib/sets"
	"github.com/stretchr/testify/assert"
)

func TestAccAlzArchetypeDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccExampleDataSourceConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.alz_archetype.test", "name", "example"),
					resource.TestCheckResourceAttr("data.alz_archetype.test", "alz_policy_assignments.%", "1"),
				),
			},
		},
	})
}

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

	data "alz_archetype" "test" {
		name           = "example"
		parent_id      = "test"
		base_archetype = "test"
		defaults = {
			location = "westeurope"
			log_analytics_workspace_id = "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg/providers/Microsoft.OperationalInsights/workspaces/la"
		}
	}
	`, libPath)
}

// TestAddAttrStringElementsToSet tests that addAttrStringElementsToSet adds a value to a set
func TestAddAttrStringElementsToSet(t *testing.T) {
	arch := &alzlib.Archetype{
		PolicyDefinitions: sets.NewSet[string]("a", "b", "c"),
	}
	vals := []attr.Value{
		basetypes.NewStringValue("d"),
	}
	addAttrStringElementsToSet(arch.PolicyDefinitions, vals)
	assert.True(t, arch.PolicyDefinitions.Contains("d"))
}

// TestDeleteAttrStringElementsToSet tests that deleteAttrStringElementsToSet removes a value from a set
func TestDeleteAttrStringElementsToSet(t *testing.T) {
	arch := &alzlib.Archetype{
		PolicyDefinitions: sets.NewSet[string]("a", "b", "c"),
	}
	vals := []attr.Value{
		basetypes.NewStringValue("c"),
	}
	deleteAttrStringElementsFromSet(arch.PolicyDefinitions, vals)
	assert.True(t, !arch.PolicyDefinitions.Contains("c"))
}
