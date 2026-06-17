package resources

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
)

func TestOrgConfigComputedBoolKeepsStateForUnknown(t *testing.T) {
	attr := computedBool("test bool")
	if len(attr.PlanModifiers) == 0 {
		t.Fatal("computed bool attributes must use UseStateForUnknown to avoid perpetual known-after-apply plans")
	}
}

func TestOrgConfigBoolAttributesHavePlanModifiers(t *testing.T) {
	ctx := context.Background()
	res := &orgConfigResource{}
	resp := &resource.SchemaResponse{}
	res.Schema(ctx, resource.SchemaRequest{}, resp)

	for _, name := range []string{
		"controller_version_include_prereleases",
		"allow_device_approval_on_user_auth",
		"allow_controller_approval_with_psk_only",
		"disable_peers_require_quorum",
	} {
		attr, ok := resp.Schema.Attributes[name].(schema.BoolAttribute)
		if !ok {
			t.Fatalf("%s is %T, want schema.BoolAttribute", name, resp.Schema.Attributes[name])
		}
		if len(attr.PlanModifiers) == 0 {
			t.Fatalf("%s is missing UseStateForUnknown plan modifier", name)
		}
	}
}
