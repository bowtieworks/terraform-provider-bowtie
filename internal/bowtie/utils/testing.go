package utils

import (
	"os"
	"testing"

	"github.com/bowtieworks/terraform-provider-bowtie/internal/bowtie/client"
	"github.com/bowtieworks/terraform-provider-bowtie/internal/bowtie/provider"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
)

func NewEnvClient() (*client.Client, error) {
	host := os.Getenv("BOWTIE_HOST")
	username := os.Getenv("BOWTIE_USERNAME")
	password := os.Getenv("BOWTIE_PASSWORD")

	c, err := client.NewClient(host, username, password, false, true)
	return c, err
}

// An acceptance test to confirm that the resource can be created,
// then deleted out-of-band underneath the provider, then re-applied
// and successfully recreate the resource without erroring out.
func RecreationTest(t *testing.T, resourceName string, config string, teardown func()) {
	// Re-use this step later:
	create := resource.TestStep{
		Config: config,
		ConfigPlanChecks: resource.ConfigPlanChecks{
			PreApply: []plancheck.PlanCheck{
				plancheck.ExpectResourceAction(resourceName, plancheck.ResourceActionCreate),
			},
			PostApplyPostRefresh: []plancheck.PlanCheck{
				plancheck.ExpectEmptyPlan(),
			},
		},
	}

	recreate := create
	recreate.PreConfig = teardown

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: provider.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// First, create the resource normally and confirm that no
			// pending changes remain and that the resource is consistent:
			create,
			// Then re-run the same configuration after deleting the
			// resource from underneath terraform:
			recreate,
		},
	})
}
