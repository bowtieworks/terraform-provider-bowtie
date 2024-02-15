package test

import (
	"testing"

	"github.com/bowtieworks/terraform-provider-bowtie/internal/bowtie/provider"
	"github.com/bowtieworks/terraform-provider-bowtie/internal/bowtie/utils"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

const siteConfig = provider.ProviderConfig + `
resource "bowtie_site" "test" {
  name = "Test Site"
}
`

func TestAccSiteResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: provider.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: siteConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("bowtie_site.test", "name", "Test Site"),
					resource.TestCheckResourceAttrSet("bowtie_site.test", "id"),
					resource.TestCheckResourceAttrSet("bowtie_site.test", "last_updated"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "bowtie_site.test",
				ImportState:       true,
				ImportStateVerify: true,
				// The last_updated attribute does not exist in the HashiCups
				// API, therefore there is no value for it during import.
				ImportStateVerifyIgnore: []string{"last_updated"},
			},
		},
	})
}

func TestAccSiteRecreation(t *testing.T) {
	utils.RecreationTest(
		t,
		"bowtie_site.test",
		siteConfig,
		deleteSiteResources,
	)
}

// Delete all site resources from the API.
func deleteSiteResources() {
	client, _ := utils.NewEnvClient()

	// Pretty simple blanket statement to just remove everything.
	sites, _ := client.GetSites()
	for _, site := range sites {
		client.DeleteSite(site.ID)
	}
}
