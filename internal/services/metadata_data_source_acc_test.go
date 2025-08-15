package services_test

import (
	"testing"

	"github.com/Azure/terraform-provider-alz/internal/acceptance"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccAlzMetadataDataSource tests the data source for alz_metadata.
func TestAccAlzMetadataDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.AccTestPreCheck(t) },
		ProtoV6ProviderFactories: acceptance.AccTestProtoV6ProviderFactoriesUnique(),
		Steps: []resource.TestStep{
			{
				Config: testAccMetadataDataSourceConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.alz_metadata.test", "alz_library_references.0", "platform/alz@2024.07.5"),
				),
			},
		},
	})
}

// TestAccAlzMetadataDataSource tests the data source for alz_metadata doesn't return custom url libraries.
// when using a remote lib.
func TestAccAlzMetadataDataSourceCustomLib(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acceptance.AccTestPreCheck(t) },
		ProtoV6ProviderFactories: acceptance.AccTestProtoV6ProviderFactoriesUnique(),
		Steps: []resource.TestStep{
			{
				Config: testAccMetadataDataSourceConfigCustomUrl(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.alz_metadata.test", "alz_library_references.#", "0"),
				),
			},
		},
	})
}

// testAccMetadataDataSourceConfig returns a test configuration for .
func testAccMetadataDataSourceConfig() string {
	return `
provider "alz" {
  library_references = [
  {
	  path = "platform/alz"
		ref  = "2024.07.5"
	}
	]
}

data "alz_metadata" "test" {}
`
}

// testAccMetadataDataSourceConfig returns a test configuration for .
func testAccMetadataDataSourceConfigCustomUrl() string {
	return `
provider "alz" {
  library_references = [
  {
		custom_url = "testdata/testacc_lib"
	}
	]
}

data "alz_metadata" "test" {}
`
}
