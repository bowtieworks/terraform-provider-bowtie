package test

import (
	"strings"
	"testing"
	"text/template"

	"github.com/bowtieworks/terraform-provider-bowtie/internal/bowtie/provider"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
)

func TestAccGroupMembershipResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: provider.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: getGroupMembershipConfig(true),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{},
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("bowtie_group_membership.admin_memberships", "users.#", "3"),
					resource.TestCheckResourceAttrSet("bowtie_group_membership.admin_memberships", "group_id"),
				),
			},
			{
				Config: getGroupMembershipConfig(false),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{},
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
			},
		},
	})
}

func getGroupMembershipConfig(enabled bool) string {
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
	err = tmpl.ExecuteTemplate(output, "group_membership.tmpl", map[string]any{
		"enabled":  enabled,
		"provider": provider.ProviderConfig,
	})
	if err != nil {
		panic("Failed to render template")
	}

	return output.String()
}
