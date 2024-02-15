package test

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"text/template"

	"github.com/bowtieworks/terraform-provider-bowtie/internal/bowtie/provider"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
)

func init() {
	resource.AddTestSweepers("user", &resource.Sweeper{
		Name: "user",
		F: func(host string) error {
			ctx := context.Background()
			client, err := getBowtieClient(host)
			if err != nil {
				return err
			}

			users, err := client.GetUsers()
			if err != nil {
				return err
			}

			for _, user := range users {
				if user.Email == "admin@example.com" {
					continue
				}

				if user.Role == "Owner" {
					_, err := client.UpsertUser(user.ID, "", "", "User", false, false, false, false, false)
					if err != nil {
						fmt.Println("[Error] Failed to demote user")
						continue
					}
				}

				err = client.DisableUser(ctx, user.ID)
				if err != nil {
					fmt.Println("[Error] failed to disble user")
					continue
				}

				err = client.DeleteUser(ctx, user.ID)
				if err != nil {
					fmt.Println("[Error] failed to delete user")
					continue
				}
			}
			return nil
		},
	})
}

func TestAccUserResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: provider.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: getUserConfig("Jane Doe", "jane.doe@example.com", "", false, false, false, false, false),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{},
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("bowtie_user.user", "name", "Jane Doe"),
					resource.TestCheckResourceAttr("bowtie_user.user", "email", "jane.doe@example.com"),
					resource.TestCheckResourceAttr("bowtie_user.user", "role", "User"),
					resource.TestCheckResourceAttr("bowtie_user.user", "enabled", "true"),
					resource.TestCheckResourceAttr("bowtie_user.user", "authz_devices", "false"),
					resource.TestCheckResourceAttr("bowtie_user.user", "authz_policies", "false"),
					resource.TestCheckResourceAttr("bowtie_user.user", "authz_control_plane", "false"),
					resource.TestCheckResourceAttr("bowtie_user.user", "authz_users", "false"),
					resource.TestCheckResourceAttrSet("bowtie_user.user", "id"),
				),
			},
			{
				Config: getUserConfig("Jane Doe", "jane.doe@example.com", "Owner", true, true, true, true, true),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{},
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("bowtie_user.user", "name", "Jane Doe"),
					resource.TestCheckResourceAttr("bowtie_user.user", "email", "jane.doe@example.com"),
					resource.TestCheckResourceAttr("bowtie_user.user", "role", "Owner"),
					resource.TestCheckResourceAttr("bowtie_user.user", "enabled", "true"),
					resource.TestCheckResourceAttr("bowtie_user.user", "authz_devices", "true"),
					resource.TestCheckResourceAttr("bowtie_user.user", "authz_policies", "true"),
					resource.TestCheckResourceAttr("bowtie_user.user", "authz_control_plane", "true"),
					resource.TestCheckResourceAttr("bowtie_user.user", "authz_users", "true"),
					resource.TestCheckResourceAttrSet("bowtie_user.user", "id"),
				),
			},
			{
				Config: getUserConfig("Jane Doe", "jane.doe@example.com", "User", true, false, false, false, false),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{},
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("bowtie_user.user", "name", "Jane Doe"),
					resource.TestCheckResourceAttr("bowtie_user.user", "email", "jane.doe@example.com"),
					resource.TestCheckResourceAttr("bowtie_user.user", "role", "User"),
					resource.TestCheckResourceAttr("bowtie_user.user", "enabled", "true"),
					resource.TestCheckResourceAttr("bowtie_user.user", "authz_devices", "false"),
					resource.TestCheckResourceAttr("bowtie_user.user", "authz_policies", "false"),
					resource.TestCheckResourceAttr("bowtie_user.user", "authz_control_plane", "false"),
					resource.TestCheckResourceAttr("bowtie_user.user", "authz_users", "false"),
					resource.TestCheckResourceAttrSet("bowtie_user.user", "id"),
				),
			},
		},
	})
}

func getUserConfig(name, email, role string, authz, authz_users, authz_devices, authz_policies, authz_control_plane bool) string {
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
	err = tmpl.ExecuteTemplate(output, "user.tmpl", map[string]interface{}{
		"provider":            provider.ProviderConfig,
		"name":                name,
		"email":               email,
		"role":                role,
		"authz":               authz,
		"authz_devices":       authz_devices,
		"authz_policies":      authz_policies,
		"authz_users":         authz_users,
		"authz_control_plane": authz_control_plane,
	})
	if err != nil {
		panic("Failed to render template")
	}

	return output.String()
}
