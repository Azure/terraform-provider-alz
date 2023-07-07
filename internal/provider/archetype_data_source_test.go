// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAlzArchetypeDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Read testing
			{
				Config: testAccExampleDataSourceConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.alz_archetype.test", "name", "example"),
					resource.TestCheckResourceAttr("data.alz_archetype.test", "alz_policy_assignments.%", "1"),
				),
			},
		},
	})
}

const testAccExampleDataSourceConfig = `
provider "alz" {
	use_alz_lib = false
	lib_dirs = [
		"./testdata/tfacc_lib",
	]
}



data "alz_archetype" "test" {
	name           = "example"
  parent_id      = "test"
	base_archetype = "root"
	defaults = {
		location = "westeurope"
		log_analytics_workspace_id = "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg/providers/Microsoft.OperationalInsights/workspaces/la"
	}
}
`
