package test

import (
	"fmt"
	"testing"
	"time"

	"github.com/bowtieworks/terraform-provider-bowtie/internal/bowtie/provider"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
)

func TestAccPolicyEngineResources(t *testing.T) {
	suffix := time.Now().UnixNano()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: provider.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: policyEngineConfig(suffix, false),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("bowtie_device_group.laptops", "name", fmt.Sprintf("tf functional laptops %d", suffix)),
					resource.TestCheckResourceAttr("bowtie_collection.endpoints", "members.#", "3"),
					resource.TestCheckResourceAttr("bowtie_resource.app", "location.cidr", "10.42.0.0/16"),
					resource.TestCheckResourceAttr("bowtie_resource_group.apps", "resources.#", "1"),
					resource.TestCheckResourceAttr("bowtie_policy.allow_laptops", "action", "Accept"),
					resource.TestCheckResourceAttr("bowtie_policy.allow_laptops", "status", "Enabled"),
					resource.TestCheckResourceAttrSet("bowtie_policy.allow_laptops", "id"),
				),
			},
			{
				Config: policyEngineConfig(suffix, true),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("bowtie_device_group.laptops", plancheck.ResourceActionUpdate),
						plancheck.ExpectResourceAction("bowtie_collection.endpoints", plancheck.ResourceActionUpdate),
						plancheck.ExpectResourceAction("bowtie_resource.app", plancheck.ResourceActionUpdate),
						plancheck.ExpectResourceAction("bowtie_policy.allow_laptops", plancheck.ResourceActionUpdate),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("bowtie_device_group.laptops", "description", "updated by terraform functional test"),
					resource.TestCheckResourceAttr("bowtie_collection.endpoints", "members.#", "2"),
					resource.TestCheckResourceAttr("bowtie_resource.app", "protocol", "https"),
					resource.TestCheckResourceAttr("bowtie_policy.allow_laptops", "action", "Reject"),
					resource.TestCheckResourceAttr("bowtie_policy.allow_laptops", "status", "Disabled"),
				),
			},
			{
				ResourceName:      "bowtie_device_group.laptops",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				ResourceName:      "bowtie_collection.endpoints",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				ResourceName:      "bowtie_resource.app",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				ResourceName:      "bowtie_resource_group.apps",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				ResourceName:      "bowtie_policy.allow_laptops",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func policyEngineConfig(suffix int64, updated bool) string {
	deviceDescription := "created by terraform functional test"
	collectionDescription := "private endpoints for functional test"
	resourceProtocol := "tcp"
	ports := "collection = [443, 8443]"
	resourceCIDR := "10.42.0.0/16"
	policySource := "or = [{ device_group = bowtie_device_group.laptops.id }, { authenticated_user = true }]"
	policyAction := "Accept"
	policyStatus := "Enabled"
	members := `
    {
      name     = "wiki"
      location = { dns = "wiki.functional.example.com" }
    },
    {
      name     = "private cidr"
      comment  = "initial cidr member"
      location = { cidr = "10.42.0.0/16" }
    },
    {
      name     = "single host"
      location = { ip = "203.0.113.42" }
    },`

	if updated {
		deviceDescription = "updated by terraform functional test"
		collectionDescription = "updated private endpoints for functional test"
		resourceProtocol = "https"
		ports = "range = [443, 443]"
		resourceCIDR = "10.43.0.0/16"
		policySource = "always = true"
		policyAction = "Reject"
		policyStatus = "Disabled"
		members = `
    {
      name     = "wiki"
      location = { dns = "wiki-updated.functional.example.com" }
    },
    {
      name     = "updated cidr"
      comment  = "updated cidr member"
      location = { cidr = "10.43.0.0/16" }
    },`
	}

	return fmt.Sprintf(provider.ProviderConfig+`
resource "bowtie_device_group" "laptops" {
  name        = "tf functional laptops %[1]d"
  description = %[2]q
}

resource "bowtie_collection" "endpoints" {
  name        = "tf functional endpoints %[1]d"
  description = %[3]q

  members = [%[4]s
  ]
}

resource "bowtie_resource" "app" {
  name     = "tf functional app %[1]d"
	  protocol = %[5]q

	  location = {
	    cidr = %[10]q
	  }

  ports = {
    %[6]s
  }
}

resource "bowtie_resource_group" "apps" {
  name      = "tf functional apps %[1]d"
  resources = [bowtie_resource.app.id]
  inherited = []
}

resource "bowtie_policy" "allow_laptops" {
  source = {
    %[7]s
  }
  dest   = bowtie_resource_group.apps.id
  action = %[8]q
  status = %[9]q
}
	`, suffix, deviceDescription, collectionDescription, members, resourceProtocol, ports, policySource, policyAction, policyStatus, resourceCIDR)
}
