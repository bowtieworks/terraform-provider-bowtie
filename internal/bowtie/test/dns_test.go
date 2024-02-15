package test

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"text/template"

	"github.com/bowtieworks/terraform-provider-bowtie/internal/bowtie/client"
	"github.com/bowtieworks/terraform-provider-bowtie/internal/bowtie/provider"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
)

func TestAccDNSResource(t *testing.T) {
	getDNSConfig("chrisk-test.example.com", []string{"1.1.1.1", "4.4.4.4"}, []string{"wrong.example.com"}, nil)
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: provider.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: getDNSConfig("chrisk-test.example.com", []string{"1.1.1.1", "4.4.4.4"}, []string{"wrong.example.com"}, nil),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{},
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("bowtie_dns.test", "name", "chrisk-test.example.com"),
					resource.TestCheckResourceAttr("bowtie_dns.test", "servers.0.addr", "1.1.1.1"),
					resource.TestCheckResourceAttr("bowtie_dns.test", "servers.1.addr", "4.4.4.4"),
					resource.TestCheckResourceAttr("bowtie_dns.test", "is_dns64", "true"),
					resource.TestCheckResourceAttrSet("bowtie_dns.test", "servers.0.id"),
					resource.TestCheckResourceAttrSet("bowtie_dns.test", "servers.0.order"),
					resource.TestCheckResourceAttr("bowtie_dns.test", "excludes.0.name", "wrong.example.com"),
					resource.TestCheckResourceAttrSet("bowtie_dns.test", "excludes.0.id"),
					resource.TestCheckResourceAttrSet("bowtie_dns.test", "excludes.0.order"),
					resource.TestCheckResourceAttrSet("bowtie_dns.test", "id"),
					resource.TestCheckResourceAttrSet("bowtie_dns.test", "last_updated"),
				),
			},
			{
				Config: getDNSConfig("chrisk-test.example.com", []string{"1.1.1.1"}, []string{"wrong.example.com"}, nil),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectNonEmptyPlan(),
						plancheck.ExpectResourceAction("bowtie_dns.test", plancheck.ResourceActionUpdate),
					},
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("bowtie_dns.test", "name", "chrisk-test.example.com"),
					resource.TestCheckResourceAttr("bowtie_dns.test", "servers.0.addr", "1.1.1.1"),
					resource.TestCheckResourceAttr("bowtie_dns.test", "is_dns64", "true"),
					resource.TestCheckResourceAttrSet("bowtie_dns.test", "servers.0.id"),
					resource.TestCheckResourceAttrSet("bowtie_dns.test", "servers.0.order"),
					resource.TestCheckNoResourceAttr("bowtie_dns.test", "servers.1.addr"),
					resource.TestCheckNoResourceAttr("bowtie_dns.test", "servers.1.id"),
					resource.TestCheckNoResourceAttr("bowtie_dns.test", "servers.1.order"),
					resource.TestCheckResourceAttr("bowtie_dns.test", "excludes.0.name", "wrong.example.com"),
					resource.TestCheckResourceAttrSet("bowtie_dns.test", "excludes.0.id"),
					resource.TestCheckResourceAttrSet("bowtie_dns.test", "excludes.0.order"),
					resource.TestCheckResourceAttrSet("bowtie_dns.test", "id"),
					resource.TestCheckResourceAttrSet("bowtie_dns.test", "last_updated"),
				),
			},
			{
				Config: getDNSConfig("chrisk-test.example.com", []string{"1.1.1.1"}, []string{"wrong.example.com", "ignore.example.com"}, nil),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectNonEmptyPlan(),
						plancheck.ExpectResourceAction("bowtie_dns.test", plancheck.ResourceActionUpdate),
					},
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("bowtie_dns.test", "name", "chrisk-test.example.com"),
					resource.TestCheckResourceAttr("bowtie_dns.test", "servers.0.addr", "1.1.1.1"),
					resource.TestCheckResourceAttr("bowtie_dns.test", "is_dns64", "true"),
					resource.TestCheckResourceAttrSet("bowtie_dns.test", "servers.0.id"),
					resource.TestCheckResourceAttrSet("bowtie_dns.test", "servers.0.order"),
					resource.TestCheckNoResourceAttr("bowtie_dns.test", "servers.1.addr"),
					resource.TestCheckNoResourceAttr("bowtie_dns.test", "servers.1.id"),
					resource.TestCheckNoResourceAttr("bowtie_dns.test", "servers.1.order"),
					resource.TestCheckResourceAttr("bowtie_dns.test", "excludes.0.name", "wrong.example.com"),
					resource.TestCheckResourceAttrSet("bowtie_dns.test", "excludes.0.id"),
					resource.TestCheckResourceAttrSet("bowtie_dns.test", "excludes.0.order"),
					resource.TestCheckResourceAttrSet("bowtie_dns.test", "id"),
					resource.TestCheckResourceAttrSet("bowtie_dns.test", "last_updated"),
				),
			},
			{
				Config: getDNSConfig("chrisk-test.example.com", []string{"1.1.1.1"}, []string{"wrong.example.com"}, nil),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectNonEmptyPlan(),
						plancheck.ExpectResourceAction("bowtie_dns.test", plancheck.ResourceActionUpdate),
					},
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("bowtie_dns.test", "name", "chrisk-test.example.com"),
					resource.TestCheckResourceAttr("bowtie_dns.test", "servers.0.addr", "1.1.1.1"),
					resource.TestCheckResourceAttr("bowtie_dns.test", "is_dns64", "true"),
					resource.TestCheckResourceAttrSet("bowtie_dns.test", "servers.0.id"),
					resource.TestCheckResourceAttrSet("bowtie_dns.test", "servers.0.order"),
					resource.TestCheckNoResourceAttr("bowtie_dns.test", "servers.1.addr"),
					resource.TestCheckNoResourceAttr("bowtie_dns.test", "servers.1.id"),
					resource.TestCheckNoResourceAttr("bowtie_dns.test", "servers.1.order"),
					resource.TestCheckResourceAttr("bowtie_dns.test", "excludes.0.name", "wrong.example.com"),
					resource.TestCheckResourceAttrSet("bowtie_dns.test", "excludes.0.id"),
					resource.TestCheckResourceAttrSet("bowtie_dns.test", "excludes.0.order"),
					resource.TestCheckResourceAttrSet("bowtie_dns.test", "id"),
					resource.TestCheckResourceAttrSet("bowtie_dns.test", "last_updated"),
				),
			},
			{
				Config: getDNSConfig("chrisk-test.example.com", []string{"1.1.1.1"}, []string{"wrong.example.com"}, []string{"542b94ed-2866-4ff0-8b32-4ec1616039e9"}),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectNonEmptyPlan(),
						plancheck.ExpectResourceAction("bowtie_dns.test", plancheck.ResourceActionUpdate),
					},
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("bowtie_dns.test", "name", "chrisk-test.example.com"),
					resource.TestCheckResourceAttr("bowtie_dns.test", "servers.0.addr", "1.1.1.1"),
					resource.TestCheckResourceAttr("bowtie_dns.test", "is_dns64", "true"),
					resource.TestCheckResourceAttrSet("bowtie_dns.test", "servers.0.id"),
					resource.TestCheckResourceAttrSet("bowtie_dns.test", "servers.0.order"),
					resource.TestCheckNoResourceAttr("bowtie_dns.test", "servers.1.addr"),
					resource.TestCheckNoResourceAttr("bowtie_dns.test", "servers.1.id"),
					resource.TestCheckNoResourceAttr("bowtie_dns.test", "servers.1.order"),
					resource.TestCheckResourceAttr("bowtie_dns.test", "excludes.0.name", "wrong.example.com"),
					resource.TestCheckResourceAttrSet("bowtie_dns.test", "excludes.0.id"),
					resource.TestCheckResourceAttrSet("bowtie_dns.test", "excludes.0.order"),
					resource.TestCheckResourceAttrSet("bowtie_dns.test", "id"),
					resource.TestCheckResourceAttrSet("bowtie_dns.test", "last_updated"),
					resource.TestCheckResourceAttr("bowtie_dns.test", "include_only_sites.#", "1"),
					resource.TestCheckResourceAttr("bowtie_dns.test", "include_only_sites.0", "542b94ed-2866-4ff0-8b32-4ec1616039e9"),
				),
			},
			{
				Config: getDNSConfig("chrisk-test.example.com", []string{"1.1.1.1"}, []string{"wrong.example.com"}, []string{"542b94ed-2866-4ff0-8b32-4ec1616039e9", "86661a74-c408-4dde-a2c6-027d7f64da59"}),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectNonEmptyPlan(),
						plancheck.ExpectResourceAction("bowtie_dns.test", plancheck.ResourceActionUpdate),
					},
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("bowtie_dns.test", "name", "chrisk-test.example.com"),
					resource.TestCheckResourceAttr("bowtie_dns.test", "servers.0.addr", "1.1.1.1"),
					resource.TestCheckResourceAttr("bowtie_dns.test", "is_dns64", "true"),
					resource.TestCheckResourceAttrSet("bowtie_dns.test", "servers.0.id"),
					resource.TestCheckResourceAttrSet("bowtie_dns.test", "servers.0.order"),
					resource.TestCheckNoResourceAttr("bowtie_dns.test", "servers.1.addr"),
					resource.TestCheckNoResourceAttr("bowtie_dns.test", "servers.1.id"),
					resource.TestCheckNoResourceAttr("bowtie_dns.test", "servers.1.order"),
					resource.TestCheckResourceAttr("bowtie_dns.test", "excludes.0.name", "wrong.example.com"),
					resource.TestCheckResourceAttrSet("bowtie_dns.test", "excludes.0.id"),
					resource.TestCheckResourceAttrSet("bowtie_dns.test", "excludes.0.order"),
					resource.TestCheckResourceAttrSet("bowtie_dns.test", "id"),
					resource.TestCheckResourceAttrSet("bowtie_dns.test", "last_updated"),
					resource.TestCheckResourceAttr("bowtie_dns.test", "include_only_sites.#", "2"),
					resource.TestCheckResourceAttr("bowtie_dns.test", "include_only_sites.0", "542b94ed-2866-4ff0-8b32-4ec1616039e9"),
				),
			},
			{
				Config: getDNSConfig("chrisk-test.example.com", []string{"1.1.1.1"}, []string{"wrong.example.com"}, []string{"86661a74-c408-4dde-a2c6-027d7f64da59"}),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectNonEmptyPlan(),
						plancheck.ExpectResourceAction("bowtie_dns.test", plancheck.ResourceActionUpdate),
					},
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("bowtie_dns.test", "name", "chrisk-test.example.com"),
					resource.TestCheckResourceAttr("bowtie_dns.test", "servers.0.addr", "1.1.1.1"),
					resource.TestCheckResourceAttr("bowtie_dns.test", "is_dns64", "true"),
					resource.TestCheckResourceAttrSet("bowtie_dns.test", "servers.0.id"),
					resource.TestCheckResourceAttrSet("bowtie_dns.test", "servers.0.order"),
					resource.TestCheckNoResourceAttr("bowtie_dns.test", "servers.1.addr"),
					resource.TestCheckNoResourceAttr("bowtie_dns.test", "servers.1.id"),
					resource.TestCheckNoResourceAttr("bowtie_dns.test", "servers.1.order"),
					resource.TestCheckResourceAttr("bowtie_dns.test", "excludes.0.name", "wrong.example.com"),
					resource.TestCheckResourceAttrSet("bowtie_dns.test", "excludes.0.id"),
					resource.TestCheckResourceAttrSet("bowtie_dns.test", "excludes.0.order"),
					resource.TestCheckResourceAttrSet("bowtie_dns.test", "id"),
					resource.TestCheckResourceAttrSet("bowtie_dns.test", "last_updated"),
					resource.TestCheckResourceAttr("bowtie_dns.test", "include_only_sites.#", "1"),
					resource.TestCheckResourceAttr("bowtie_dns.test", "include_only_sites.0", "86661a74-c408-4dde-a2c6-027d7f64da59"),
				),
			},
		},
	})
}

// An acceptance test to confirm that the resource can be created,
// then deleted out-of-band underneath the provider, then re-applied
// and successfully recreate the resource without erroring out.
func TestAccDNSResourceRecreation(t *testing.T) {
	// Re-use this step later:
	create := resource.TestStep{
		Config: getDNSConfig("example.com", []string{"1.1.1.1"}, []string{}, nil),
		ConfigPlanChecks: resource.ConfigPlanChecks{
			PreApply: []plancheck.PlanCheck{
				plancheck.ExpectResourceAction("bowtie_dns.test", plancheck.ResourceActionCreate),
			},
			PostApplyPostRefresh: []plancheck.PlanCheck{
				plancheck.ExpectEmptyPlan(),
			},
		},
		Check: resource.ComposeAggregateTestCheckFunc(
			resource.TestCheckResourceAttr("bowtie_dns.test", "name", "example.com"),
			resource.TestCheckResourceAttr("bowtie_dns.test", "servers.0.addr", "1.1.1.1"),
			resource.TestCheckResourceAttrSet("bowtie_dns.test", "id"),
			resource.TestCheckResourceAttrSet("bowtie_dns.test", "last_updated"),
		),
	}

	recreate := create
	recreate.PreConfig = deleteDNSResources

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: provider.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// First, create the resource normally and confirm that no
			// pending changes remain and that the resource is consistent:
			create,
			// Then re-run the same configuration after deleting the
			// resource from underneath terrform:
			recreate,
		},
	})
}

// Delete all DNS resources from the API.
func deleteDNSResources() {
	username := os.Getenv("BOWTIE_USERNAME")
	password := os.Getenv("BOWTIE_PASSWORD")

	c, err := client.NewClient("http://127.0.0.1:3000", username, password, false)
	if err != nil {
		fmt.Println("Couldn't create Bowtie client")
	}

	// Pretty simple blanket statement to just remove everything.
	dnss, _ := c.GetDNS()
	for id, _ := range dnss {
		c.DeleteDNS(id)
	}
}

func getDNSConfig(name string, servers, excludes, sites []string) string {
	funcMap := template.FuncMap{
		"notNil": func(val any) bool {
			return val != nil
		},
	}

	tmpl, err := template.New("").Funcs(funcMap).ParseGlob("testdata/*.tmpl")
	if err != nil {
		return ""
	}

	var output *strings.Builder = &strings.Builder{}
	err = tmpl.ExecuteTemplate(output, "dns.tmpl", map[string]interface{}{
		"provider": provider.ProviderConfig,
		"name":     name,
		"servers":  servers,
		"excludes": excludes,
		"sites":    sites,
	})
	if err != nil {
		panic("Failed to render template")
	}

	return output.String()
}
