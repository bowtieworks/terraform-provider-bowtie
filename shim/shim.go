// Package shim exposes the Bowtie Terraform provider's constructor outside the
// internal/ tree so the Pulumi bridge (a separate Go module) can wrap it.
//
// The Pulumi bridge cannot import internal/bowtie/provider directly because Go
// forbids importing another module's internal/ packages. This thin re-export is
// the supported way to hand the plugin-framework provider to the bridge.
package shim

import (
	pfprovider "github.com/hashicorp/terraform-plugin-framework/provider"

	"github.com/bowtieworks/terraform-provider-bowtie/internal/bowtie/provider"
)

// NewProvider returns the terraform-plugin-framework provider that powers the
// bowtieworks/bowtie Terraform provider.
func NewProvider() pfprovider.Provider {
	return provider.New()
}
