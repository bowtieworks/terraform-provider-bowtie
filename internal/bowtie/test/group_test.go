package test

import (
	"strings"
	"testing"
	"text/template"

	"github.com/bowtieworks/terraform-provider-bowtie/internal/bowtie/provider"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

const (
	groupResourceName = "bowtie_group.test"

	groupName       = "My Group"
	groupNameChange = "My Renamed Group"
)

func TestGroupResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: provider.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Basic tests for groups
			{
				Config: getGroupConfig(groupResourceName, groupName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(groupResourceName, "name", groupName),
					resource.TestCheckResourceAttrSet(groupResourceName, "id"),
					resource.TestCheckResourceAttrSet(groupResourceName, "last_updated"),
				),
			},
			// ImportState testing
			{
				ResourceName:      groupResourceName,
				ImportState:       true,
				ImportStateVerify: true,
				// The last_updated attribute does not exist in the HashiCups
				// API, therefore there is no value for it during import.
				ImportStateVerifyIgnore: []string{"last_updated"},
			},
			// Update and Read testing
			{
				Config: getGroupConfig(groupResourceName, groupNameChange),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(groupResourceName, "name", groupNameChange),
					resource.TestCheckResourceAttrSet(groupResourceName, "id"),
					resource.TestCheckResourceAttrSet(groupResourceName, "last_updated"),
				),
			},
		},
	})
}

func getGroupConfig(resource string, name string) string {
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
	err = tmpl.ExecuteTemplate(output, "group.tmpl", map[string]interface{}{
		"provider": provider.ProviderConfig,
		"resource": strings.Split(resource, ".")[1],
		"name":     name,
	})

	if err != nil {
		panic("Failed to render template")
	}

	return output.String()
}
