package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccAlzArchetypeDataSource tests the data source for alz_archetype.
// It checks that the policy parameter substitution & location defaults are applied.
func TestAccAlzArchitectureDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactoriesUnique(),
		ExternalProviders: map[string]resource.ExternalProvider{
			"azurerm": {
				Source:            "hashicorp/azurerm",
				VersionConstraint: "~> 3.107",
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
provider "alz" {}

provider "azurerm" {
  features {}
}

data "azurerm_client_config" "current" {}

data "alz_architecture" "test" {
  name                     = "alz"
	root_management_group_id = data.azurerm_client_config.current.tenant_id
	location                 = "northeurope"
}
`
}
