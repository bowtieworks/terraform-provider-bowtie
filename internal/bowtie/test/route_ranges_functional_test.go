package test

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/bowtieworks/terraform-provider-bowtie/internal/bowtie/provider"
	"github.com/bowtieworks/terraform-provider-bowtie/internal/bowtie/utils"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccRouteExclusionResource(t *testing.T) {
	suffix := time.Now().UnixNano()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: provider.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: routeExclusionConfig(suffix, false),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("bowtie_route_exclusion.office_lan", "apply_strategy", "percentage_user_match"),
					resource.TestCheckResourceAttr("bowtie_route_exclusion.office_lan", "apply_strategy_percentage", "64"),
					resource.TestCheckResourceAttrSet("bowtie_route_exclusion.office_lan", "version"),
				),
			},
			{
				Config: routeExclusionConfig(suffix, true),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("bowtie_route_exclusion.office_lan", plancheck.ResourceActionUpdate),
					},
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("bowtie_route_exclusion.office_lan", "apply_strategy", "always"),
					resource.TestCheckNoResourceAttr("bowtie_route_exclusion.office_lan", "apply_strategy_percentage"),
				),
			},
			{
				ResourceName:      "bowtie_route_exclusion.office_lan",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccOrgRangeResources(t *testing.T) {
	suffix := time.Now().UnixNano()
	v4ThirdOctet := 100 + suffix%100

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: provider.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: orgRangesConfig(v4ThirdOctet, false),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("bowtie_ipv4_range.functional", "assign_addresses_from_here", "never"),
					resource.TestCheckResourceAttr("bowtie_ipv4_range.functional", "skip_first_n_addresses", "0"),
					resource.TestCheckResourceAttr("bowtie_ipv6_range.functional", "assign_addresses_from_here", "false"),
					resource.TestCheckResourceAttrSet("bowtie_ipv4_range.functional", "id"),
					resource.TestCheckResourceAttrSet("bowtie_ipv6_range.functional", "id"),
				),
			},
			{
				Config: orgRangesConfig(v4ThirdOctet, true),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("bowtie_ipv4_range.functional", plancheck.ResourceActionUpdate),
						plancheck.ExpectResourceAction("bowtie_ipv6_range.functional", plancheck.ResourceActionUpdate),
					},
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("bowtie_ipv4_range.functional", "assign_addresses_from_here", "on-demand-assign-random"),
					resource.TestCheckResourceAttr("bowtie_ipv4_range.functional", "skip_first_n_addresses", "2"),
					resource.TestCheckResourceAttr("bowtie_ipv6_range.functional", "assign_addresses_from_here", "true"),
				),
			},
			{
				ResourceName:            "bowtie_ipv4_range.functional",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"last_updated"},
			},
			{
				ResourceName:            "bowtie_ipv6_range.functional",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"last_updated"},
			},
		},
	})
}

func TestAccIPv4RangeOutOfBandDeleteRecreates(t *testing.T) {
	suffix := time.Now().UnixNano()
	v4ThirdOctet := 100 + suffix%100
	config := orgRangesConfig(v4ThirdOctet, false)
	var ipv4RangeID string

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: provider.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
				Check: captureResourceID("bowtie_ipv4_range.functional", &ipv4RangeID),
			},
			{
				PreConfig: func() {
					if ipv4RangeID == "" {
						t.Fatal("missing IPv4 range ID from first apply")
					}
					c, err := utils.NewEnvClient()
					if err != nil {
						t.Fatalf("new env client: %v", err)
					}
					if err := c.DeleteIPv4Range(ipv4RangeID); err != nil {
						t.Fatalf("delete IPv4 range out of band: %v", err)
					}
				},
				Config: config,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("bowtie_ipv4_range.functional", plancheck.ResourceActionCreate),
					},
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
			},
		},
	})
}

func captureResourceID(address string, target *string) resource.TestCheckFunc {
	return func(state *terraform.State) error {
		rs, ok := state.RootModule().Resources[address]
		if !ok {
			return fmt.Errorf("resource %s not found in state", address)
		}
		if rs.Primary == nil || rs.Primary.ID == "" {
			return errors.New("resource has no primary ID")
		}
		*target = rs.Primary.ID
		return nil
	}
}

func routeExclusionConfig(suffix int64, updated bool) string {
	applyStrategy := `apply_strategy = "percentage_user_match"
  apply_strategy_percentage = 64
  only_if_wan_matches_cidrs = ["203.0.113.0/24"]
  match_only_device_groups   = [bowtie_device_group.laptops.id]`

	if updated {
		applyStrategy = `apply_strategy = "always"
  only_if_wan_matches_cidrs = ["198.51.100.0/24"]`
	}

	return fmt.Sprintf(provider.ProviderConfig+`
resource "bowtie_device_group" "laptops" {
  name        = "tf route laptops %[1]d"
  description = "route exclusion selector"
}

resource "bowtie_collection" "office_lan" {
  name        = "tf route office lan %[1]d"
  description = "office lan route exclusion targets"

  members = [
    {
      name     = "office lan"
      location = { cidr = "10.55.0.0/16" }
    },
  ]
}

resource "bowtie_route_exclusion" "office_lan" {
  name          = "tf functional office lan %[1]d"
  collection_id = bowtie_collection.office_lan.id

  %[2]s
}
`, suffix, applyStrategy)
}

func orgRangesConfig(v4ThirdOctet int64, updated bool) string {
	assignV4 := "never"
	skipFirst := int64(0)
	assignV6 := "false"

	if updated {
		assignV4 = "on-demand-assign-random"
		skipFirst = 2
		assignV6 = "true"
	}

	return fmt.Sprintf(provider.ProviderConfig+`
resource "bowtie_ipv4_range" "functional" {
  range                      = "198.18.%[1]d.0/24"
  assign_addresses_from_here = %[2]q
  skip_first_n_addresses     = %[3]d
}

resource "bowtie_ipv6_range" "functional" {
  range                      = "fd00:1:1:%[1]x::/64"
  assign_addresses_from_here = %[4]s
}
`, v4ThirdOctet, assignV4, skipFirst, assignV6)
}
