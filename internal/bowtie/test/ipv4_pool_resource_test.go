package test

import (
	"embed"
	"fmt"
	"strings"
	"testing"
	"text/template"

	"github.com/bowtieworks/terraform-provider-bowtie/internal/bowtie/provider"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// Embed all *.tmpl files under testdata so we don't depend on CWD.
//go:embed testdata/*.tmpl
var templatesFS embed.FS

// Renders testdata/ipv4_pool.tmpl with the provided values.
func getIPv4PoolConfig(resourceName, ipRange, assign string, skipFirst int) string {
	funcMap := template.FuncMap{
		"notNil": func(v any) bool { return v != nil },
	}

	tmpl, err := template.New("").Funcs(funcMap).ParseFS(templatesFS, "testdata/*.tmpl")
	if err != nil {
		panic(fmt.Errorf("parse templates: %w", err))
	}

	var out strings.Builder
	err = tmpl.ExecuteTemplate(&out, "ipv4_pool.tmpl", map[string]any{
		"provider":                   provider.ProviderConfig,
		"resource":                   strings.Split(resourceName, ".")[1], // "test" from "bowtie_ipv4_pool.test"
		"range":                      ipRange,
		"assign_addresses_from_here": assign,
		"skip_first_n_addresses":     skipFirst,
	})
	if err != nil {
		panic(fmt.Errorf("render ipv4_pool.tmpl: %w", err))
	}
	return out.String()
}

func TestIPv4PoolResource(t *testing.T) {
	const rn = "bowtie_ipv4_pool.test"

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: provider.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create
			{
				Config: getIPv4PoolConfig(rn, "10.10.0.0/24", "on-demand-assign-random", 10),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(rn, "range", "10.10.0.0/24"),
					resource.TestCheckResourceAttr(rn, "assign_addresses_from_here", "on-demand-assign-random"),
					resource.TestCheckResourceAttr(rn, "skip_first_n_addresses", "10"),
					resource.TestCheckResourceAttrSet(rn, "id"),
					resource.TestCheckResourceAttrSet(rn, "last_updated"),
				),
			},
			// Import
			{
				ResourceName:            rn,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"last_updated"},
			},
			// Update (no site_strategies to avoid UUID key requirement)
			{
				Config: getIPv4PoolConfig(rn, "10.20.0.0/24", "always-assign-random", 5),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(rn, "range", "10.20.0.0/24"),
					resource.TestCheckResourceAttr(rn, "assign_addresses_from_here", "always-assign-random"),
					resource.TestCheckResourceAttr(rn, "skip_first_n_addresses", "5"),
					resource.TestCheckResourceAttrSet(rn, "id"),
					resource.TestCheckResourceAttrSet(rn, "last_updated"),
				),
			},
		},
	})
}
