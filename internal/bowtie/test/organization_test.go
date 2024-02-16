package test

import (
	"strings"
	"testing"
	"text/template"

	"github.com/bowtieworks/terraform-provider-bowtie/internal/bowtie/provider"
	"github.com/bowtieworks/terraform-provider-bowtie/internal/bowtie/utils"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

const (
	resourceOrg = "bowtie_organization.org"
	orgName     = "Demonstration"
	orgDomain   = "example.com"
)

func TestAccOrganizationResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: provider.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Import testing. Note that organizations cannot be
			// created or destroyed, so we rely purely on import to create
			// the initial resource.
			{
				Config:       getOrganizationConfig(resourceOrg, orgName, orgDomain),
				ResourceName: resourceOrg,
				ImportState:  true,
				// We can’t control what Id the API comes up with for the
				// organization ID, so derive it dynamically:
				ImportStateIdFunc: getOrgId(),
				// The last_updated attribute does not exist in the HashiCups
				// API, therefore there is no value for it during import.
				ImportStateVerifyIgnore: []string{"last_updated"},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceOrg, "name"),
					resource.TestCheckResourceAttrSet(resourceOrg, "domain"),
					resource.TestCheckResourceAttrSet(resourceOrg, "id"),
					resource.TestCheckResourceAttrSet(resourceOrg, "last_updated"),
				),
			},
		},
	})
}

func getOrganizationConfig(resource string, name string, domain string) string {
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
	err = tmpl.ExecuteTemplate(output, "organization.tmpl", map[string]interface{}{
		"provider":      provider.ProviderConfig,
		"resource_type": strings.Split(resource, ".")[0],
		"resource":      strings.Split(resource, ".")[1],
		"name":          name,
		"domain":        domain,
	})

	if err != nil {
		panic("Failed to render template")
	}

	return output.String()
}

func getOrgId() resource.ImportStateIdFunc {
	return func(state *terraform.State) (string, error) {
		client, err := utils.NewEnvClient()

		if err != nil {
			return "", err
		}

		org, err := client.GetOrganization()
		if err != nil {
			return "", err
		}

		return org.ID, nil
	}
}
