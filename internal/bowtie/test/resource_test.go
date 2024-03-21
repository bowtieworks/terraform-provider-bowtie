package test

import (
	"strings"
	"testing"
	"text/template"

	"github.com/bowtieworks/terraform-provider-bowtie/internal/bowtie/provider"
	"github.com/bowtieworks/terraform-provider-bowtie/internal/bowtie/utils"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
)

func TestAccBowtieResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: provider.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: getBowtieConfig(
					"Internal web",
					"http",
					map[string]string{
						"cidr": "10.0.0.0/16",
					},
					map[string][]int{
						"collection": {80, 443},
					},
				),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{},
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("bowtie_resource.test", "name", "Internal web"),
					resource.TestCheckResourceAttr("bowtie_resource.test", "protocol", "http"),
					resource.TestCheckResourceAttr("bowtie_resource.test", "location.cidr", "10.0.0.0/16"),
					resource.TestCheckResourceAttr("bowtie_resource.test", "ports.collection.0", "80"),
					resource.TestCheckResourceAttr("bowtie_resource.test", "ports.collection.1", "443"),
				),
			},
			{
				Config: getBowtieConfig(
					"Internal web changed",
					"all",
					map[string]string{
						"dns": "test.example.com",
					},
					map[string][]int{
						"range": {0, 65535},
					},
				),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectNonEmptyPlan(),
						plancheck.ExpectResourceAction("bowtie_resource.test", plancheck.ResourceActionUpdate),
					},
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("bowtie_resource.test", "name", "Internal web changed"),
					resource.TestCheckResourceAttr("bowtie_resource.test", "protocol", "all"),
					resource.TestCheckResourceAttr("bowtie_resource.test", "location.dns", "test.example.com"),
					resource.TestCheckResourceAttr("bowtie_resource.test", "ports.range.0", "0"),
					resource.TestCheckResourceAttr("bowtie_resource.test", "ports.range.1", "65535"),
				),
			},
		},
	})
}

func TestAccBowtieResourceRecreation(t *testing.T) {
	utils.RecreationTest(
		t,
		"bowtie_resource.test",
		getBowtieConfig(
			"Testing",
			"all",
			map[string]string{
				"ip": "192.168.1.1",
			},
			map[string][]int{
				"collection": {80},
			},
		),
		deleteBowtieResources,
	)
}

// Delete all Bowtie resources from the API.
func deleteBowtieResources() {
	client, _ := utils.NewEnvClient()

	// Pretty simple blanket statement to just remove everything.
	resources, _ := client.GetResources()
	for id := range resources {
		_ = client.DeleteResource(id)
	}
}

func getBowtieConfig(name string, protocol string, locations map[string]string, ports map[string][]int) string {
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
	err = tmpl.ExecuteTemplate(output, "resource.tmpl", map[string]interface{}{
		"provider":  provider.ProviderConfig,
		"name":      name,
		"protocol":  protocol,
		"locations": locations,
		"ports":     ports,
	})
	if err != nil {
		panic("Failed to render template")
	}

	return output.String()
}
