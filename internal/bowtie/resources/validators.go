package resources

import (
	"context"
	"fmt"
	"net/netip"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type cidrPrefixValidator struct {
	version int
}

func (v cidrPrefixValidator) Description(ctx context.Context) string {
	if v.version == 4 || v.version == 6 {
		return fmt.Sprintf("value must be an IPv%d CIDR prefix", v.version)
	}
	return "value must be a CIDR prefix"
}

func (v cidrPrefixValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v cidrPrefixValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	prefix, err := netip.ParsePrefix(req.ConfigValue.ValueString())
	if err != nil {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid CIDR prefix",
			"Value must be a valid CIDR prefix: "+err.Error(),
		)
		return
	}

	switch v.version {
	case 4:
		if !prefix.Addr().Is4() {
			resp.Diagnostics.AddAttributeError(req.Path, "Invalid IPv4 CIDR prefix", "Value must be an IPv4 CIDR prefix.")
		}
	case 6:
		if !prefix.Addr().Is6() {
			resp.Diagnostics.AddAttributeError(req.Path, "Invalid IPv6 CIDR prefix", "Value must be an IPv6 CIDR prefix.")
		}
	}
}

func stringSetToMap(ctx context.Context, value types.Set, attrPath path.Path, allowed map[string]struct{}, diags *diag.Diagnostics) map[string]bool {
	out := map[string]bool{}
	if value.IsNull() || value.IsUnknown() {
		return out
	}

	var values []string
	diags.Append(value.ElementsAs(ctx, &values, false)...)
	if diags.HasError() {
		return out
	}

	for _, item := range values {
		if _, ok := allowed[item]; !ok {
			diags.AddAttributeError(
				attrPath,
				"Unsupported clear field",
				fmt.Sprintf("%q is not a supported clear field.", item),
			)
			continue
		}
		out[item] = true
	}
	return out
}

func validateClearConflicts(ctx context.Context, value types.Set, attrPath path.Path, allowed map[string]struct{}, configured map[string]bool, diags *diag.Diagnostics) {
	clear := stringSetToMap(ctx, value, attrPath, allowed, diags)
	for field := range clear {
		if configured[field] {
			diags.AddAttributeError(
				attrPath,
				"Clear field conflicts with configured value",
				fmt.Sprintf("%q is listed for clearing and is also configured. Remove one of the two settings.", field),
			)
		}
	}
}

func isNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "http 404") || strings.Contains(msg, "not found")
}

type taggedValueValueModifier struct {
	typePath      path.Path
	requiresValue map[string]bool
}

func clearWhenTaggedTypeHasNoValue(typePath path.Path, requiresValue map[string]bool) planmodifier.String {
	return taggedValueValueModifier{
		typePath:      typePath,
		requiresValue: requiresValue,
	}
}

func (m taggedValueValueModifier) Description(ctx context.Context) string {
	return "Clears the value when the selected tagged enum variant does not carry one."
}

func (m taggedValueValueModifier) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (m taggedValueValueModifier) PlanModifyString(ctx context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	var typeValue types.String
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, m.typePath, &typeValue)...)
	if resp.Diagnostics.HasError() || typeValue.IsNull() || typeValue.IsUnknown() {
		resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, m.typePath, &typeValue)...)
	}
	if resp.Diagnostics.HasError() || typeValue.IsNull() || typeValue.IsUnknown() {
		resp.Diagnostics.Append(req.State.GetAttribute(ctx, m.typePath, &typeValue)...)
	}
	if resp.Diagnostics.HasError() || typeValue.IsNull() || typeValue.IsUnknown() {
		return
	}
	if !m.requiresValue[typeValue.ValueString()] {
		resp.PlanValue = types.StringNull()
	}
}
