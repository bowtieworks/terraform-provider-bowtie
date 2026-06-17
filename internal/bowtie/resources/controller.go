package resources

import (
	"context"
	"fmt"
	"time"

	"github.com/bowtieworks/terraform-provider-bowtie/internal/bowtie/client"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
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

var _ resource.Resource = &controllerResource{}
var _ resource.ResourceWithImportState = &controllerResource{}
var _ resource.ResourceWithValidateConfig = &controllerResource{}

type controllerResource struct {
	client *client.Client
}

type controllerResourceModel struct {
	ID                              types.String `tfsdk:"id"`
	LastUpdated                     types.String `tfsdk:"last_updated"`
	ClearOverrides                  types.Set    `tfsdk:"clear_overrides"`
	SiteID                          types.String `tfsdk:"site_id"`
	PublicAddress                   types.String `tfsdk:"public_address"`
	WireguardPort                   types.Int64  `tfsdk:"wireguard_port"`
	PersistentKeepalive             types.Int64  `tfsdk:"persistent_keepalive"`
	WireguardStrategy               types.String `tfsdk:"wireguard_strategy"`
	VersionStrategyType             types.String `tfsdk:"version_strategy_type"`
	VersionStrategyValue            types.String `tfsdk:"version_strategy_value"`
	VersionStrategySplayType        types.String `tfsdk:"version_strategy_splay_type"`
	VersionStrategySplayValue       types.String `tfsdk:"version_strategy_splay_value"`
	VersionIncludePrereleases       types.Bool   `tfsdk:"version_include_prereleases"`
	VersionMinimumAge               types.Int64  `tfsdk:"version_minimum_age"`
	SSHListener                     types.String `tfsdk:"ssh_listener"`
	WebFilterTrustedProxyCollection types.String `tfsdk:"web_filter_trusted_proxy_collection"`
	CanUseVanityDomain              types.Bool   `tfsdk:"can_use_vanity_domain"`
	CanUsePublicHTTPS               types.Bool   `tfsdk:"can_use_public_https"`
	CanUseIDP                       types.Bool   `tfsdk:"can_use_idp"`
	TrackPolicyVerdictMetrics       types.Bool   `tfsdk:"track_policy_verdict_metrics"`
	TrackPolicyVerdictLogs          types.Bool   `tfsdk:"track_policy_verdict_logs"`
	AllowTemporaryConsoleUsers      types.Bool   `tfsdk:"allow_temporary_console_users"`
	PublicKey                       types.String `tfsdk:"public_key"`
	HTTPSEndpoint                   types.String `tfsdk:"https_endpoint"`
	CurrentVersion                  types.String `tfsdk:"current_version"`
}

var controllerClearableOverrides = map[string]struct{}{
	"version_strategy_splay":              {},
	"version_include_prereleases":         {},
	"version_minimum_age":                 {},
	"ssh_listener":                        {},
	"web_filter_trusted_proxy_collection": {},
}

var controllerVersionStrategyValueVariants = map[string]bool{
	"specific":           true,
	"newest-at-interval": true,
	"newest-at-calendar": true,
}

var controllerVersionStrategySplayValueVariants = map[string]bool{
	"randomized-delay":            true,
	"consistent-randomized-delay": true,
}

func NewControllerResource() resource.Resource {
	return &controllerResource{}
}

func (r *controllerResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_controller"
}

func (r *controllerResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `
Manage the lifecycle settings of an existing Bowtie Controller: its update
(version) strategy, update stagger, minimum release age, and other per-Controller options.

**Note**: Controllers register themselves with the control plane when they boot, so this
resource **cannot create or destroy** a Controller. Create will fail; instead, import an
existing Controller with ` + "`terraform import bowtie_controller.<name> <controller-id>`" + `
and then manage its settings. Destroying the resource removes it from Terraform state only and
leaves the Controller running.

Optional attributes that inherit an organization default keep the Controller's current value when
they are omitted from configuration. To actively clear one of those per-Controller overrides, list
it in ` + "`clear_overrides`" + `.
`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The Controller's unique identifier. Set by importing an existing Controller.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"last_updated": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The last time Terraform changed this object. Provider metadata, not part of the Bowtie API.",
			},
			"clear_overrides": schema.SetAttribute{
				Optional:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "Per-Controller overrides to clear on update. Supported values are `version_strategy_splay`, `version_include_prereleases`, `version_minimum_age`, `ssh_listener`, and `web_filter_trusted_proxy_collection`. Use `version_strategy_type = \"org-default\"` to make the Controller inherit the organization update strategy.",
			},
			"site_id": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "The site this Controller belongs to.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"public_address": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "The publicly reachable address clients and other Controllers use to reach this Controller.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"wireguard_port": schema.Int64Attribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "The UDP port the Controller listens on for VPN connections.",
				Validators:          []validator.Int64{int64validator.Between(0, 65535)},
				PlanModifiers:       []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
			},
			"persistent_keepalive": schema.Int64Attribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Keepalive interval, in seconds, advertised to peers.",
				Validators:          []validator.Int64{int64validator.Between(0, 65535)},
				PlanModifiers:       []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
			},
			"wireguard_strategy": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "How the VPN port is reached: `static` (publicly reachable) or `dynamic` (privately reachable or NAT-punched).",
				Validators:          []validator.String{stringvalidator.OneOf("static", "dynamic")},
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"version_strategy_type": schema.StringAttribute{
				Optional: true,
				Computed: true,
				MarkdownDescription: "How this Controller chooses update versions. One of `org-default`, `manual`, " +
					"`specific`, `newest-at-interval`, or `newest-at-calendar`. This is the upgrade orchestration control.",
				Validators: []validator.String{
					stringvalidator.OneOf("org-default", "manual", "specific", "newest-at-interval", "newest-at-calendar"),
				},
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"version_strategy_value": schema.StringAttribute{
				Optional: true,
				Computed: true,
				MarkdownDescription: "The value for `version_strategy_type` when it requires one: a version string for `specific`, " +
					"a systemd time-span for `newest-at-interval`, or a systemd calendar expression for `newest-at-calendar`. " +
					"Leave unset for `org-default` and `manual`.",
				PlanModifiers: []planmodifier.String{
					clearWhenTaggedTypeHasNoValue(path.Root("version_strategy_type"), controllerVersionStrategyValueVariants),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"version_strategy_splay_type": schema.StringAttribute{
				Optional: true,
				Computed: true,
				MarkdownDescription: "Spreads update times across a fleet. One of `no-delay`, `randomized-delay`, or " +
					"`consistent-randomized-delay`.",
				Validators: []validator.String{
					stringvalidator.OneOf("no-delay", "randomized-delay", "consistent-randomized-delay"),
				},
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"version_strategy_splay_value": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "A systemd time-span used by `randomized-delay` and `consistent-randomized-delay`.",
				PlanModifiers: []planmodifier.String{
					clearWhenTaggedTypeHasNoValue(path.Root("version_strategy_splay_type"), controllerVersionStrategySplayValueVariants),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"version_include_prereleases": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Consider pre-release versions when selecting updates. Unset falls back to the organization default.",
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"version_minimum_age": schema.Int64Attribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Minimum age, in days, a release must have before this Controller adopts it under a latest-version strategy. Unset falls back to the organization default.",
				PlanModifiers:       []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
			},
			"ssh_listener": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Where the Controller accepts SSH: `any`, `local-only`, or `tunnel-only`.",
				Validators:          []validator.String{stringvalidator.OneOf("any", "local-only", "tunnel-only")},
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"web_filter_trusted_proxy_collection": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "A collection ID used as the web-filter trusted proxy list for this Controller.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"can_use_vanity_domain": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Whether this Controller may use a vanity domain.",
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"can_use_public_https": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Whether this Controller may serve public HTTPS.",
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"can_use_idp": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Whether this Controller may use the identity provider.",
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"track_policy_verdict_metrics": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Record per-packet policy verdict metrics.",
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"track_policy_verdict_logs": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Emit per-packet policy verdict logs.",
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"allow_temporary_console_users": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Whether temporary console users may be created on this Controller.",
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"public_key": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The Controller's VPN public key.",
			},
			"https_endpoint": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The Controller's HTTPS endpoint.",
			},
			"current_version": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The software version the Controller is currently running.",
			},
		},
	}
}

func (r *controllerResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *controllerResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	resp.Diagnostics.AddError(
		"Controller creation is not supported.",
		"Controllers register themselves at boot. Import an existing Controller with `terraform import bowtie_controller.<name> <controller-id>` instead of creating one.",
	)
}

func (r *controllerResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state controllerResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	controller, err := r.client.GetController(state.ID.ValueString())
	if err != nil {
		if isNotFoundError(err) {
			resp.Diagnostics.AddAttributeWarning(
				path.Root("id"),
				"Controller not found, removing from state",
				state.ID.ValueString(),
			)
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Failed reading Controller", err.Error())
		return
	}

	r.mapToState(controller, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *controllerResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan controllerResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Read-modify-write: the update endpoint overlays the posted payload onto
	// the existing record, so start from the current representation and change
	// only the managed fields.
	current, err := r.client.GetController(plan.ID.ValueString())
	if err != nil {
		if isNotFoundError(err) {
			resp.Diagnostics.AddAttributeWarning(
				path.Root("id"),
				"Controller not found, removing from state",
				plan.ID.ValueString(),
			)
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Failed reading Controller before update", err.Error())
		return
	}

	if !plan.SiteID.IsNull() && !plan.SiteID.IsUnknown() {
		v := plan.SiteID.ValueString()
		current.SiteID = &v
	}
	if !plan.PublicAddress.IsNull() && !plan.PublicAddress.IsUnknown() {
		current.PublicAddress = plan.PublicAddress.ValueString()
	}
	if !plan.WireguardPort.IsNull() && !plan.WireguardPort.IsUnknown() {
		current.WireguardPort = int(plan.WireguardPort.ValueInt64())
	}
	if !plan.PersistentKeepalive.IsNull() && !plan.PersistentKeepalive.IsUnknown() {
		current.PersistentKeepalive = int(plan.PersistentKeepalive.ValueInt64())
	}
	if !plan.WireguardStrategy.IsNull() && !plan.WireguardStrategy.IsUnknown() {
		current.WireguardStrategy = client.TaggedValue{Type: plan.WireguardStrategy.ValueString()}
	}
	if !plan.VersionStrategyType.IsNull() && !plan.VersionStrategyType.IsUnknown() {
		current.VersionStrategy = taggedValueFromPlan(plan.VersionStrategyType, plan.VersionStrategyValue, controllerVersionStrategyValueVariants)
	}
	if !plan.VersionStrategySplayType.IsNull() && !plan.VersionStrategySplayType.IsUnknown() {
		splay := taggedValueFromPlan(plan.VersionStrategySplayType, plan.VersionStrategySplayValue, controllerVersionStrategySplayValueVariants)
		current.VersionStrategySplay = &splay
	}
	if !plan.VersionIncludePrereleases.IsNull() && !plan.VersionIncludePrereleases.IsUnknown() {
		v := plan.VersionIncludePrereleases.ValueBool()
		current.VersionIncludePrereleases = &v
	}
	if !plan.VersionMinimumAge.IsNull() && !plan.VersionMinimumAge.IsUnknown() {
		v := int(plan.VersionMinimumAge.ValueInt64())
		current.VersionMinimumAge = &v
	}
	if !plan.SSHListener.IsNull() && !plan.SSHListener.IsUnknown() {
		v := plan.SSHListener.ValueString()
		current.SSHListener = &v
	}
	if !plan.WebFilterTrustedProxyCollection.IsNull() && !plan.WebFilterTrustedProxyCollection.IsUnknown() {
		v := plan.WebFilterTrustedProxyCollection.ValueString()
		current.WebFilterTrustedProxyCollection = &v
	}
	if !plan.CanUseVanityDomain.IsNull() && !plan.CanUseVanityDomain.IsUnknown() {
		current.CanUseVanityDomain = plan.CanUseVanityDomain.ValueBool()
	}
	if !plan.CanUsePublicHTTPS.IsNull() && !plan.CanUsePublicHTTPS.IsUnknown() {
		current.CanUsePublicHTTPS = plan.CanUsePublicHTTPS.ValueBool()
	}
	if !plan.CanUseIDP.IsNull() && !plan.CanUseIDP.IsUnknown() {
		current.CanUseIDP = plan.CanUseIDP.ValueBool()
	}
	if !plan.TrackPolicyVerdictMetrics.IsNull() && !plan.TrackPolicyVerdictMetrics.IsUnknown() {
		v := plan.TrackPolicyVerdictMetrics.ValueBool()
		current.TrackPolicyVerdictMetrics = &v
	}
	if !plan.TrackPolicyVerdictLogs.IsNull() && !plan.TrackPolicyVerdictLogs.IsUnknown() {
		v := plan.TrackPolicyVerdictLogs.ValueBool()
		current.TrackPolicyVerdictLogs = &v
	}
	if !plan.AllowTemporaryConsoleUsers.IsNull() && !plan.AllowTemporaryConsoleUsers.IsUnknown() {
		current.AllowTemporaryConsoleUsers = plan.AllowTemporaryConsoleUsers.ValueBool()
	}

	clearOverrides := stringSetToMap(ctx, plan.ClearOverrides, path.Root("clear_overrides"), controllerClearableOverrides, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	if clearOverrides["version_strategy_splay"] {
		current.VersionStrategySplay = nil
	}
	if clearOverrides["version_include_prereleases"] {
		current.VersionIncludePrereleases = nil
	}
	if clearOverrides["version_minimum_age"] {
		current.VersionMinimumAge = nil
	}
	if clearOverrides["ssh_listener"] {
		current.SSHListener = nil
	}
	if clearOverrides["web_filter_trusted_proxy_collection"] {
		current.WebFilterTrustedProxyCollection = nil
	}

	updated, err := r.client.UpdateController(current)
	if err != nil {
		resp.Diagnostics.AddError("Failed updating Controller", err.Error())
		return
	}

	r.mapToState(updated, &plan)
	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *controllerResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	resp.Diagnostics.AddWarning(
		"Controller not removed from Bowtie",
		"Removing bowtie_controller from Terraform state stops managing it but does not remove the Controller from the control plane. Remove a Controller through the Control Plane if that is intended.",
	)
}

func (r *controllerResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *controllerResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var config controllerResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	validateTaggedValue(
		config.VersionStrategyType, config.VersionStrategyValue,
		path.Root("version_strategy_value"), "version_strategy",
		controllerVersionStrategyValueVariants,
		&resp.Diagnostics,
	)
	validateTaggedValue(
		config.VersionStrategySplayType, config.VersionStrategySplayValue,
		path.Root("version_strategy_splay_value"), "version_strategy_splay",
		controllerVersionStrategySplayValueVariants,
		&resp.Diagnostics,
	)
	validateClearConflicts(
		ctx,
		config.ClearOverrides,
		path.Root("clear_overrides"),
		controllerClearableOverrides,
		map[string]bool{
			"version_strategy_splay":              isSet(config.VersionStrategySplayType) || isSet(config.VersionStrategySplayValue),
			"version_include_prereleases":         isSetBool(config.VersionIncludePrereleases),
			"version_minimum_age":                 isSetInt(config.VersionMinimumAge),
			"ssh_listener":                        isSet(config.SSHListener),
			"web_filter_trusted_proxy_collection": isSet(config.WebFilterTrustedProxyCollection),
		},
		&resp.Diagnostics,
	)
}

// mapToState copies the server representation into the Terraform model.
func (r *controllerResource) mapToState(c *client.ControllerSettings, state *controllerResourceModel) {
	state.ID = types.StringValue(c.ID)
	state.SiteID = stringFromPtr(c.SiteID)
	state.PublicAddress = types.StringValue(c.PublicAddress)
	state.WireguardPort = types.Int64Value(int64(c.WireguardPort))
	state.PersistentKeepalive = types.Int64Value(int64(c.PersistentKeepalive))
	state.WireguardStrategy = types.StringValue(c.WireguardStrategy.Type)
	state.VersionStrategyType, state.VersionStrategyValue = taggedValueToState(c.VersionStrategy, controllerVersionStrategyValueVariants)
	if c.VersionStrategySplay != nil {
		state.VersionStrategySplayType, state.VersionStrategySplayValue = taggedValueToState(*c.VersionStrategySplay, controllerVersionStrategySplayValueVariants)
	} else {
		state.VersionStrategySplayType = types.StringNull()
		state.VersionStrategySplayValue = types.StringNull()
	}
	state.VersionIncludePrereleases = boolFromPtr(c.VersionIncludePrereleases)
	state.VersionMinimumAge = int64FromIntPtr(c.VersionMinimumAge)
	state.SSHListener = stringFromPtr(c.SSHListener)
	state.WebFilterTrustedProxyCollection = stringFromPtr(c.WebFilterTrustedProxyCollection)
	state.CanUseVanityDomain = types.BoolValue(c.CanUseVanityDomain)
	state.CanUsePublicHTTPS = types.BoolValue(c.CanUsePublicHTTPS)
	state.CanUseIDP = types.BoolValue(c.CanUseIDP)
	state.TrackPolicyVerdictMetrics = boolFromPtr(c.TrackPolicyVerdictMetrics)
	state.TrackPolicyVerdictLogs = boolFromPtr(c.TrackPolicyVerdictLogs)
	state.AllowTemporaryConsoleUsers = types.BoolValue(c.AllowTemporaryConsoleUsers)
	state.PublicKey = types.StringValue(c.PublicKey)
	state.HTTPSEndpoint = types.StringValue(c.HTTPSEndpoint)
	state.CurrentVersion = stringFromPtr(c.CurrentVersion)
}

func optionalString(v types.String) *string {
	if v.IsNull() || v.IsUnknown() {
		return nil
	}
	s := v.ValueString()
	return &s
}

func taggedValueFromPlan(typeAttr, valueAttr types.String, requiresValue map[string]bool) client.TaggedValue {
	tagged := client.TaggedValue{Type: typeAttr.ValueString()}
	if requiresValue[tagged.Type] {
		tagged.Value = optionalString(valueAttr)
	}
	return tagged
}

func taggedValueToState(tagged client.TaggedValue, requiresValue map[string]bool) (types.String, types.String) {
	if tagged.Type == "" {
		return types.StringNull(), types.StringNull()
	}
	if !requiresValue[tagged.Type] {
		return types.StringValue(tagged.Type), types.StringNull()
	}
	return types.StringValue(tagged.Type), stringFromPtr(tagged.Value)
}

// validateTaggedValue enforces that an internally-tagged enum's value attribute
// is set exactly when the chosen type variant carries content.
func validateTaggedValue(typeAttr, valueAttr types.String, valuePath path.Path, name string, requiresValue map[string]bool, diags *diag.Diagnostics) {
	if typeAttr.IsNull() || typeAttr.IsUnknown() {
		return
	}
	needs := requiresValue[typeAttr.ValueString()]
	hasValue := !valueAttr.IsNull() && !valueAttr.IsUnknown()
	if needs && !hasValue {
		diags.AddAttributeError(valuePath, "Missing "+name+"_value",
			fmt.Sprintf("%s_type = %q requires %s_value to be set.", name, typeAttr.ValueString(), name))
	}
	if !needs && hasValue {
		diags.AddAttributeError(valuePath, "Unexpected "+name+"_value",
			fmt.Sprintf("%s_value must not be set when %s_type = %q.", name, name, typeAttr.ValueString()))
	}
}

func stringFromPtr(p *string) types.String {
	if p == nil {
		return types.StringNull()
	}
	return types.StringValue(*p)
}

func boolFromPtr(p *bool) types.Bool {
	if p == nil {
		return types.BoolNull()
	}
	return types.BoolValue(*p)
}

func int64FromIntPtr(p *int) types.Int64 {
	if p == nil {
		return types.Int64Null()
	}
	return types.Int64Value(int64(*p))
}
