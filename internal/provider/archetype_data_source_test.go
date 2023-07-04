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
				),
			},
		},
	})
}

const testAccExampleDataSourceConfig = `
data "alz_archetype" "test" {
	name           = "example"
  parent_id      = "test"
	base_archetype = "root"
	defaults = {
		location = "westeurope"
	}
}
`
