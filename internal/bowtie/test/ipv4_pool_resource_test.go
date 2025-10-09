package test

import (
	"strings"
	"testing"
	"text/template"

	"github.com/bowtieworks/terraform-provider-bowtie/internal/bowtie/provider"
	"github.com/bowtieworks/terraform-provider-bowtie/internal/bowtie/utils"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

const (
	ipv4ResourceName = "bowtie_ipv4_pool.test"

	// Create step values
	ipv4Range         = "10.10.0.0/24"
	ipv4Assign        = "on-demand-assign-random"
	ipv4SkipFirst     = 10
	ipv4SiteStrategiesCreate = "" // none on create

	// Update step values
	ipv4RangeChange    = "10.20.0.0/24"
	ipv4AssignChange   = "always-assign-random"
	ipv4SkipFirstChange = 5

	// Provide some JSON for update (tests parsing + read-back existence)
	// Note: order of keys may vary server-side; we only assert it is set.
	ipv4SiteStrategiesUpdate = `{
  "site-a": {"type": "nat"},
  "site-b": {"type": "static", "value": "10.20.0.5"}
}`
)

func TestIPv4PoolResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: provider.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create
			{
				Config: getIPv4PoolConfig(
					ipv4ResourceName,
					ipv4Range,
					ipv4Assign,
					ipv4SkipFirst,
					ipv4SiteStrategiesCreate,
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(ipv4ResourceName, "range", ipv4Range),
					resource.TestCheckResourceAttr(ipv4ResourceName, "assign_addresses_from_here", ipv4Assign),
					resource.TestCheckResourceAttr(ipv4ResourceName, "skip_first_n_addresses", "10"),
					// Not provided on create
					resource.TestCheckNoResourceAttr(ipv4ResourceName, "site_strategies"),
					resource.TestCheckResourceAttrSet(ipv4ResourceName, "id"),
					resource.TestCheckResourceAttrSet(ipv4ResourceName, "last_updated"),
				),
			},
			// Import
			{
				ResourceName:            ipv4ResourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"last_updated"},
			},
			// Update
			{
				Config: getIPv4PoolConfig(
					ipv4ResourceName,
					ipv4RangeChange,
					ipv4AssignChange,
					ipv4SkipFirstChange,
					ipv4SiteStrategiesUpdate,
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(ipv4ResourceName, "range", ipv4RangeChange),
					resource.TestCheckResourceAttr(ipv4ResourceName, "assign_addresses_from_here", ipv4AssignChange),
					resource.TestCheckResourceAttr(ipv4ResourceName, "skip_first_n_addresses", "5"),
					// We don't assert exact JSON (ordering/spacing can differ),
					// only that the attribute is present after update.
					resource.TestCheckResourceAttrSet(ipv4ResourceName, "site_strategies"),
					resource.TestCheckResourceAttrSet(ipv4ResourceName, "id"),
					resource.TestCheckResourceAttrSet(ipv4ResourceName, "last_updated"),
				),
			},
		},
	})
}

func TestAccIPv4PoolResourceRecreation(t *testing.T) {
	utils.RecreationTest(
		t,
		ipv4ResourceName,
		getIPv4PoolConfig(ipv4ResourceName, ipv4Range, ipv4Assign, ipv4SkipFirst, ipv4SiteStrategiesCreate),
		deleteAllIPv4Pools,
	)
}

// Blanket delete of all IPv4 pools via API for clean re-creation tests.
func deleteAllIPv4Pools() {
	client, _ := utils.NewEnvClient()

	pools, _ := client.GetIPv4Pools()
	for id := range pools {
		_ = client.DeleteIPv4Pool(id)
	}
}

func getIPv4PoolConfig(resourceName, ipRange, assign string, skipFirst int, siteStrategies string) string {
	funcMap := template.FuncMap{
		"notNil": func(val any) bool { return val != nil },
	}

	tmpl, err := template.New("").Funcs(funcMap).ParseGlob("testdata/*.tmpl")
	if err != nil {
		return ""
	}

	var out strings.Builder
	err = tmpl.ExecuteTemplate(&out, "ipv4_pool.tmpl", map[string]any{
		"provider":                   provider.ProviderConfig,
		"resource":                   strings.Split(resourceName, ".")[1],
		"range":                      ipRange,
		"assign_addresses_from_here": assign,
		"skip_first_n_addresses":     skipFirst,
		"site_strategies":            siteStrategies, // empty string => omitted block
	})
	if err != nil {
		panic("failed to render template")
	}

	return out.String()
}
