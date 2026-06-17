package resources

import (
	"context"
	"encoding/json"
	"time"

	"github.com/bowtieworks/terraform-provider-bowtie/internal/bowtie/client"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

const orgConfigID = "organization-config"

var _ resource.Resource = &orgConfigResource{}
var _ resource.ResourceWithImportState = &orgConfigResource{}
var _ resource.ResourceWithValidateConfig = &orgConfigResource{}

type orgConfigResource struct {
	client *client.Client
}

type orgConfigResourceModel struct {
	ID                                     types.String `tfsdk:"id"`
	LastUpdated                            types.String `tfsdk:"last_updated"`
	ClearFields                            types.Set    `tfsdk:"clear_fields"`
	ControllerVersionStrategyType          types.String `tfsdk:"controller_version_strategy_type"`
	ControllerVersionStrategyValue         types.String `tfsdk:"controller_version_strategy_value"`
	ControllerVersionSplayType             types.String `tfsdk:"controller_version_splay_type"`
	ControllerVersionSplayValue            types.String `tfsdk:"controller_version_splay_value"`
	ControllerVersionIncludePrereleases    types.Bool   `tfsdk:"controller_version_include_prereleases"`
	ControllerVersionMinimumAge            types.Int64  `tfsdk:"controller_version_minimum_age"`
	ControllerSSHListener                  types.String `tfsdk:"controller_ssh_listener"`
	AllowDeviceApprovalOnUserAuth          types.Bool   `tfsdk:"allow_device_approval_on_user_auth"`
	AllowControllerApprovalWithPSKOnly     types.Bool   `tfsdk:"allow_controller_approval_with_psk_only"`
	DisablePeersRequireQuorum              types.Bool   `tfsdk:"disable_peers_require_quorum"`
	PeersRequireQuorumPercentage           types.Int64  `tfsdk:"peers_require_quorum_percentage"`
	DeleteDevicesMoreStaleThanDays         types.Int64  `tfsdk:"delete_devices_more_stale_than_days"`
	RemoveApprovalDevicesMoreStaleThanDays types.Int64  `tfsdk:"remove_approval_devices_more_stale_than_days"`
	UserDeviceDisassociationMinutes        types.Int64  `tfsdk:"user_device_disassociation_minutes"`
}

var orgConfigClearableFields = map[string]struct{}{
	"controller_version_strategy":                  {},
	"controller_version_splay":                     {},
	"controller_version_include_prereleases":       {},
	"controller_version_minimum_age":               {},
	"controller_ssh_listener":                      {},
	"allow_device_approval_on_user_auth":           {},
	"allow_controller_approval_with_psk_only":      {},
	"disable_peers_require_quorum":                 {},
	"peers_require_quorum_percentage":              {},
	"delete_devices_more_stale_than_days":          {},
	"remove_approval_devices_more_stale_than_days": {},
	"user_device_disassociation_minutes":           {},
}

func NewOrgConfigResource() resource.Resource { return &orgConfigResource{} }

func (r *orgConfigResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_org_config"
}

func computedString(desc string) schema.StringAttribute {
	return schema.StringAttribute{Optional: true, Computed: true, MarkdownDescription: desc,
		PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}}
}
func computedBool(desc string) schema.BoolAttribute {
	return schema.BoolAttribute{Optional: true, Computed: true, MarkdownDescription: desc,
		PlanModifiers: []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()}}
}
func computedInt(desc string) schema.Int64Attribute {
	return schema.Int64Attribute{Optional: true, Computed: true, MarkdownDescription: desc,
		PlanModifiers: []planmodifier.Int64{int64planmodifier.UseStateForUnknown()}}
}
func computedTaggedValueString(desc string, typeAttr string, requiresValue map[string]bool) schema.StringAttribute {
	return schema.StringAttribute{Optional: true, Computed: true, MarkdownDescription: desc,
		PlanModifiers: []planmodifier.String{
			clearWhenTaggedTypeHasNoValue(path.Root(typeAttr), requiresValue),
			stringplanmodifier.UseStateForUnknown(),
		}}
}
func computedStringOneOf(desc string, options ...string) schema.StringAttribute {
	attr := computedString(desc)
	attr.Validators = []validator.String{stringvalidator.OneOf(options...)}
	return attr
}
func computedIntAtLeast(desc string, min int64) schema.Int64Attribute {
	attr := computedInt(desc)
	attr.Validators = []validator.Int64{int64validator.AtLeast(min)}
	return attr
}

func (r *orgConfigResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `
Manage organization-wide configuration, including the default Controller update strategy that
Controllers inherit when their own version strategy is ` + "`org-default`" + `.

**Note**: the organization configuration is a singleton and cannot be created or destroyed.
Import it with ` + "`terraform import bowtie_org_config.this organization-config`" + ` and then manage it.
Fields the provider does not surface are preserved on update. Surfaced fields are managed only
when they are configured or listed in ` + "`clear_fields`" + `; omitting a surfaced field leaves the
current Control Plane value in place.
`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"last_updated": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Provider metadata: the last time Terraform changed this object.",
			},
			"clear_fields": schema.SetAttribute{
				Optional:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "Organization configuration keys to clear from the organization config document on update. Supported values are `controller_version_strategy`, `controller_version_splay`, `controller_version_include_prereleases`, `controller_version_minimum_age`, `controller_ssh_listener`, `allow_device_approval_on_user_auth`, `allow_controller_approval_with_psk_only`, `disable_peers_require_quorum`, `peers_require_quorum_percentage`, `delete_devices_more_stale_than_days`, `remove_approval_devices_more_stale_than_days`, and `user_device_disassociation_minutes`.",
			},
			"controller_version_strategy_type":             computedStringOneOf("Organization default update strategy: `manual`, `specific`, `newest-at-interval`, or `newest-at-calendar`. (`org-default` is not valid here; the organization default cannot itself defer to the organization default.)", "manual", "specific", "newest-at-interval", "newest-at-calendar"),
			"controller_version_strategy_value":            computedTaggedValueString("Value for the default update strategy when it requires one (a version, time-span, or calendar expression).", "controller_version_strategy_type", controllerVersionStrategyValueVariants),
			"controller_version_splay_type":                computedStringOneOf("Organization default update stagger: `no-delay`, `randomized-delay`, or `consistent-randomized-delay`.", "no-delay", "randomized-delay", "consistent-randomized-delay"),
			"controller_version_splay_value":               computedTaggedValueString("A systemd time-span for `randomized-delay` and `consistent-randomized-delay`.", "controller_version_splay_type", controllerVersionStrategySplayValueVariants),
			"controller_version_include_prereleases":       computedBool("Default for considering pre-release versions when Controllers do not specify their own value."),
			"controller_version_minimum_age":               computedInt("Default minimum age, in days, a release must have before adoption under a latest-version strategy."),
			"controller_ssh_listener":                      computedStringOneOf("Organization default for how Controllers listen on the SSH port: `any`, `local-only`, or `tunnel-only`.", "any", "local-only", "tunnel-only"),
			"allow_device_approval_on_user_auth":           computedBool("Automatically approve a device when its user authenticates after installing the client."),
			"allow_controller_approval_with_psk_only":      computedBool("Automatically admit a Controller that presents a valid pre-shared key, for zero-touch cluster scale-out."),
			"disable_peers_require_quorum":                 computedBool("When false, destructive device cleanup only proceeds when a quorum of Controller peers is connected."),
			"peers_require_quorum_percentage":              computedInt("Percentage, from 0 through 100, of Controller peers required for quorum. Default 51."),
			"delete_devices_more_stale_than_days":          computedIntAtLeast("Delete devices that have not checked in for more than this many days. Must be greater than 7.", 8),
			"remove_approval_devices_more_stale_than_days": computedIntAtLeast("Revoke approval for devices that have not checked in for more than this many days. Must be greater than 7.", 8),
			"user_device_disassociation_minutes":           computedInt("Disassociate a device from its user after this many minutes."),
		},
	}
}

func (r *orgConfigResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError("Incorrect provider data", "The provider data did not resolve as *client.Client")
		return
	}
	r.client = c
}

func (r *orgConfigResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	resp.Diagnostics.AddError(
		"Organization configuration creation is not supported.",
		"The organization configuration is a singleton. Import it with `terraform import bowtie_org_config.<name> organization-config` instead.",
	)
}

func (r *orgConfigResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state orgConfigResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	cfg, err := r.client.GetOrgConfig()
	if err != nil {
		resp.Diagnostics.AddError("Failed reading organization configuration", err.Error())
		return
	}
	r.mapToState(cfg, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *orgConfigResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan orgConfigResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	cfg, err := r.client.GetOrgConfig()
	if err != nil {
		resp.Diagnostics.AddError("Failed reading organization configuration before update", err.Error())
		return
	}

	if isSet(plan.ControllerVersionStrategyType) {
		ocSet(cfg, "controller_version_strategy", taggedValueFromPlan(plan.ControllerVersionStrategyType, plan.ControllerVersionStrategyValue, controllerVersionStrategyValueVariants))
	}
	if isSet(plan.ControllerVersionSplayType) {
		ocSet(cfg, "controller_version_splay", taggedValueFromPlan(plan.ControllerVersionSplayType, plan.ControllerVersionSplayValue, controllerVersionStrategySplayValueVariants))
	}
	if isSetBool(plan.ControllerVersionIncludePrereleases) {
		ocSet(cfg, "controller_version_include_prereleases", plan.ControllerVersionIncludePrereleases.ValueBool())
	}
	if isSetInt(plan.ControllerVersionMinimumAge) {
		ocSet(cfg, "controller_version_minimum_age", plan.ControllerVersionMinimumAge.ValueInt64())
	}
	if isSet(plan.ControllerSSHListener) {
		ocSet(cfg, "controller_ssh_listener", plan.ControllerSSHListener.ValueString())
	}
	if isSetBool(plan.AllowDeviceApprovalOnUserAuth) {
		ocSet(cfg, "allow_device_approval_on_user_auth", plan.AllowDeviceApprovalOnUserAuth.ValueBool())
	}
	if isSetBool(plan.AllowControllerApprovalWithPSKOnly) {
		ocSet(cfg, "allow_controller_approval_with_psk_only", plan.AllowControllerApprovalWithPSKOnly.ValueBool())
	}
	if isSetBool(plan.DisablePeersRequireQuorum) {
		ocSet(cfg, "disable_peers_require_quorum", plan.DisablePeersRequireQuorum.ValueBool())
	}
	if isSetInt(plan.PeersRequireQuorumPercentage) {
		ocSet(cfg, "peers_require_quorum_percentage", plan.PeersRequireQuorumPercentage.ValueInt64())
	}
	if isSetInt(plan.DeleteDevicesMoreStaleThanDays) {
		ocSet(cfg, "delete_devices_more_stale_than_days", plan.DeleteDevicesMoreStaleThanDays.ValueInt64())
	}
	if isSetInt(plan.RemoveApprovalDevicesMoreStaleThanDays) {
		ocSet(cfg, "remove_approval_devices_more_stale_than_days", plan.RemoveApprovalDevicesMoreStaleThanDays.ValueInt64())
	}
	if isSetInt(plan.UserDeviceDisassociationMinutes) {
		ocSet(cfg, "user_device_disassociation_minutes", plan.UserDeviceDisassociationMinutes.ValueInt64())
	}

	clearFields := stringSetToMap(ctx, plan.ClearFields, path.Root("clear_fields"), orgConfigClearableFields, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	for field := range clearFields {
		delete(cfg, field)
	}

	if err := r.client.UpdateOrgConfig(cfg); err != nil {
		resp.Diagnostics.AddError("Failed updating organization configuration", err.Error())
		return
	}

	r.mapToState(cfg, &plan)
	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *orgConfigResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	resp.Diagnostics.AddWarning(
		"Organization configuration not removed",
		"The organization configuration is a singleton and was not reset. Removing the resource only stops Terraform from managing it.",
	)
}

func (r *orgConfigResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *orgConfigResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var config orgConfigResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	validateTaggedValue(
		config.ControllerVersionStrategyType, config.ControllerVersionStrategyValue,
		path.Root("controller_version_strategy_value"), "controller_version_strategy",
		controllerVersionStrategyValueVariants,
		&resp.Diagnostics,
	)
	validateTaggedValue(
		config.ControllerVersionSplayType, config.ControllerVersionSplayValue,
		path.Root("controller_version_splay_value"), "controller_version_splay",
		controllerVersionStrategySplayValueVariants,
		&resp.Diagnostics,
	)
	validateClearConflicts(
		ctx,
		config.ClearFields,
		path.Root("clear_fields"),
		orgConfigClearableFields,
		map[string]bool{
			"controller_version_strategy":                  isSet(config.ControllerVersionStrategyType) || isSet(config.ControllerVersionStrategyValue),
			"controller_version_splay":                     isSet(config.ControllerVersionSplayType) || isSet(config.ControllerVersionSplayValue),
			"controller_version_include_prereleases":       isSetBool(config.ControllerVersionIncludePrereleases),
			"controller_version_minimum_age":               isSetInt(config.ControllerVersionMinimumAge),
			"controller_ssh_listener":                      isSet(config.ControllerSSHListener),
			"allow_device_approval_on_user_auth":           isSetBool(config.AllowDeviceApprovalOnUserAuth),
			"allow_controller_approval_with_psk_only":      isSetBool(config.AllowControllerApprovalWithPSKOnly),
			"disable_peers_require_quorum":                 isSetBool(config.DisablePeersRequireQuorum),
			"peers_require_quorum_percentage":              isSetInt(config.PeersRequireQuorumPercentage),
			"delete_devices_more_stale_than_days":          isSetInt(config.DeleteDevicesMoreStaleThanDays),
			"remove_approval_devices_more_stale_than_days": isSetInt(config.RemoveApprovalDevicesMoreStaleThanDays),
			"user_device_disassociation_minutes":           isSetInt(config.UserDeviceDisassociationMinutes),
		},
		&resp.Diagnostics,
	)
}

func (r *orgConfigResource) mapToState(cfg client.OrgConfig, state *orgConfigResourceModel) {
	state.ID = types.StringValue(orgConfigID)
	t, v := ocGetTagged(cfg, "controller_version_strategy", controllerVersionStrategyValueVariants)
	state.ControllerVersionStrategyType = t
	state.ControllerVersionStrategyValue = v
	st, sv := ocGetTagged(cfg, "controller_version_splay", controllerVersionStrategySplayValueVariants)
	state.ControllerVersionSplayType = st
	state.ControllerVersionSplayValue = sv
	state.ControllerVersionIncludePrereleases = ocGetBool(cfg, "controller_version_include_prereleases")
	state.ControllerVersionMinimumAge = ocGetInt(cfg, "controller_version_minimum_age")
	state.ControllerSSHListener = ocGetString(cfg, "controller_ssh_listener")
	state.AllowDeviceApprovalOnUserAuth = ocGetBool(cfg, "allow_device_approval_on_user_auth")
	state.AllowControllerApprovalWithPSKOnly = ocGetBool(cfg, "allow_controller_approval_with_psk_only")
	state.DisablePeersRequireQuorum = ocGetBool(cfg, "disable_peers_require_quorum")
	state.PeersRequireQuorumPercentage = ocGetInt(cfg, "peers_require_quorum_percentage")
	state.DeleteDevicesMoreStaleThanDays = ocGetInt(cfg, "delete_devices_more_stale_than_days")
	state.RemoveApprovalDevicesMoreStaleThanDays = ocGetInt(cfg, "remove_approval_devices_more_stale_than_days")
	state.UserDeviceDisassociationMinutes = ocGetInt(cfg, "user_device_disassociation_minutes")
}

func isSet(v types.String) bool   { return !v.IsNull() && !v.IsUnknown() }
func isSetBool(v types.Bool) bool { return !v.IsNull() && !v.IsUnknown() }
func isSetInt(v types.Int64) bool { return !v.IsNull() && !v.IsUnknown() }

func ocSet(cfg client.OrgConfig, key string, value any) {
	if b, err := json.Marshal(value); err == nil {
		cfg[key] = b
	}
}

func ocGetTagged(cfg client.OrgConfig, key string, requiresValue map[string]bool) (types.String, types.String) {
	raw, ok := cfg[key]
	if !ok {
		return types.StringNull(), types.StringNull()
	}
	var tv client.TaggedValue
	if err := json.Unmarshal(raw, &tv); err != nil || tv.Type == "" {
		return types.StringNull(), types.StringNull()
	}
	return taggedValueToState(tv, requiresValue)
}

func ocGetString(cfg client.OrgConfig, key string) types.String {
	raw, ok := cfg[key]
	if !ok {
		return types.StringNull()
	}
	var s *string
	if err := json.Unmarshal(raw, &s); err != nil {
		return types.StringNull()
	}
	return stringFromPtr(s)
}

func ocGetBool(cfg client.OrgConfig, key string) types.Bool {
	raw, ok := cfg[key]
	if !ok {
		return types.BoolNull()
	}
	var b *bool
	if err := json.Unmarshal(raw, &b); err != nil {
		return types.BoolNull()
	}
	return boolFromPtr(b)
}

func ocGetInt(cfg client.OrgConfig, key string) types.Int64 {
	raw, ok := cfg[key]
	if !ok {
		return types.Int64Null()
	}
	var n *int64
	if err := json.Unmarshal(raw, &n); err != nil {
		return types.Int64Null()
	}
	if n == nil {
		return types.Int64Null()
	}
	return types.Int64Value(*n)
}
