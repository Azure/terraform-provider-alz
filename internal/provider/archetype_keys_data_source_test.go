// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package provider

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccAlzArchetypeDataSource tests the data source for alz_archetype.
// It checks that the policy parameter substitution & location defaults are applied.
func TestAccAlzArchetypeKeysDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactoriesUnique(),
		ExternalProviders:        map[string]resource.ExternalProvider{},
		Steps: []resource.TestStep{
			{
				Config: testAccExampleDataSourceKeysConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.alz_archetype_keys.test", "alz_policy_definition_keys.#", "1"),
					resource.TestCheckResourceAttr("data.alz_archetype_keys.test", "alz_policy_assignment_keys.#", "1"),
				),
			},
		},
	})
}

// testAccExampleDataSourceConfig returns a test configuration for TestAccAlzArchetypeDataSource.
func testAccExampleDataSourceKeysConfig() string {
	cwd, _ := os.Getwd()
	libPath := filepath.Join(cwd, "testdata/testacc_lib")

	return fmt.Sprintf(`
provider "alz" {
  use_alz_lib = false
  lib_urls = [
    "%s",
  ]
}

data "alz_archetype_keys" "test" {
  base_archetype = "test"
}
`, libPath)
}
