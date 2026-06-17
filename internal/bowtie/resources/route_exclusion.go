package resources

import (
	"context"
	"fmt"

	"github.com/bowtieworks/terraform-provider-bowtie/internal/bowtie/client"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &routeExclusionResource{}
var _ resource.ResourceWithImportState = &routeExclusionResource{}

type routeExclusionResource struct {
	client *client.Client
}

type routeExclusionResourceModel struct {
	ID                      types.String `tfsdk:"id"`
	Name                    types.String `tfsdk:"name"`
	CollectionID            types.String `tfsdk:"collection_id"`
	Sites                   types.List   `tfsdk:"sites"`
	ApplyStrategy           types.String `tfsdk:"apply_strategy"`
	ApplyStrategyPercentage types.Int64  `tfsdk:"apply_strategy_percentage"`
	OnlyIfWANMatchesCIDRs   types.List   `tfsdk:"only_if_wan_matches_cidrs"`
	MatchOnlyDeviceOS       types.String `tfsdk:"match_only_device_os"`
	MatchOnlyDeviceType     types.String `tfsdk:"match_only_device_type"`
	MatchOnlyOwnership      types.String `tfsdk:"match_only_ownership"`
	MatchOnlyDeviceGroups   types.List   `tfsdk:"match_only_device_groups"`
	MatchOnlyUserGroups     types.List   `tfsdk:"match_only_user_groups"`
	Version                 types.String `tfsdk:"version"`
}

const (
	strategyAlways                = "always"
	strategyNever                 = "never"
	strategyPercentageUserMatch   = "percentage_user_match"
	strategyPercentageDeviceMatch = "percentage_device_match"
)

func NewRouteExclusionResource() resource.Resource {
	return &routeExclusionResource{}
}

func (r *routeExclusionResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_route_exclusion"
}

func (r *routeExclusionResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "A *route exclusion* keeps a set of network destinations (a collection of CIDRs) out of the Bowtie tunnel for split-tunnel routing, optionally scoped to specific sites, WAN networks, and device or user attributes.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Internal route exclusion ID.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Human-readable name for the exclusion.",
				Required:            true,
			},
			"collection_id": schema.StringAttribute{
				MarkdownDescription: "The ID of the collection whose CIDR members are excluded from the tunnel. The collection must already exist.",
				Required:            true,
			},
			"sites": schema.ListAttribute{
				MarkdownDescription: "The site IDs this exclusion applies to. Omit to apply to all sites.",
				ElementType:         types.StringType,
				Optional:            true,
			},
			"apply_strategy": schema.StringAttribute{
				MarkdownDescription: "Rollout strategy: `always`, `never` (staged but inactive), `percentage_user_match`, or `percentage_device_match`. Defaults to `always`.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(strategyAlways),
				Validators: []validator.String{
					stringvalidator.OneOf(strategyAlways, strategyNever, strategyPercentageUserMatch, strategyPercentageDeviceMatch),
				},
			},
			"apply_strategy_percentage": schema.Int64Attribute{
				MarkdownDescription: "The match percentage, from 0 through 255 in ~0.5% increments, used when `apply_strategy` is `percentage_user_match` or `percentage_device_match`.",
				Optional:            true,
				Validators: []validator.Int64{
					int64validator.Between(0, 255),
				},
			},
			"only_if_wan_matches_cidrs": schema.ListAttribute{
				MarkdownDescription: "If set, the exclusion only applies when the device's WAN IP falls within one of these CIDRs (for example, an office network).",
				ElementType:         types.StringType,
				Optional:            true,
			},
			"match_only_device_os": schema.StringAttribute{
				MarkdownDescription: "Apply only to devices with this OS: `windows`, `macos`, `linux`, `ios`, `android`, `chromeos`, `bowtie_controller`, or `unknown`. Changing or removing this recreates the exclusion.",
				Optional:            true,
				Validators: []validator.String{
					stringvalidator.OneOf("windows", "macos", "linux", "ios", "android", "chromeos", "bowtie_controller", "unknown"),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"match_only_device_type": schema.StringAttribute{
				MarkdownDescription: "Apply only to devices of this type: `laptop`, `desktop`, `mobile`, `server`, or `other`. Changing or removing this recreates the exclusion.",
				Optional:            true,
				Validators: []validator.String{
					stringvalidator.OneOf("laptop", "desktop", "mobile", "server", "other"),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"match_only_ownership": schema.StringAttribute{
				MarkdownDescription: "Apply only to devices owned by this organization ID. Changing or removing this recreates the exclusion.",
				Optional:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"match_only_device_groups": schema.ListAttribute{
				MarkdownDescription: "Apply only to devices in these device group IDs.",
				ElementType:         types.StringType,
				Optional:            true,
			},
			"match_only_user_groups": schema.ListAttribute{
				MarkdownDescription: "Apply only to devices whose user is in these user group IDs.",
				ElementType:         types.StringType,
				Optional:            true,
			},
			"version": schema.StringAttribute{
				MarkdownDescription: "A content hash of the exclusion, computed by the Controller.",
				Computed:            true,
			},
		},
	}
}

func (r *routeExclusionResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configuration Type",
			fmt.Sprintf("Expected *client.Client, got: %T, please report this to the provider.", req.ProviderData),
		)
		return
	}

	r.client = c
}

func (r *routeExclusionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan routeExclusionResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if plan.ID.ValueString() == "" {
		plan.ID = types.StringValue(uuid.NewString())
	}

	r.upsert(ctx, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *routeExclusionResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state routeExclusionResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	exclusions, err := r.client.GetRouteExclusions()
	if err != nil {
		resp.Diagnostics.AddError("Failed to read route exclusion", "Unexpected error reading route exclusion "+state.ID.ValueString()+": "+err.Error())
		return
	}

	exclusion, present := exclusions[state.ID.ValueString()]
	if !present {
		resp.Diagnostics.AddAttributeWarning(
			path.Root("id"),
			"Route exclusion not found, removing from state",
			state.ID.ValueString(),
		)
		resp.State.RemoveResource(ctx)
		return
	}

	model, diags := exclusionToModel(ctx, exclusion)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}

func (r *routeExclusionResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan routeExclusionResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	r.upsert(ctx, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *routeExclusionResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state routeExclusionResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.DeleteRouteExclusion(state.ID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Failed to delete route exclusion", "Unexpected error deleting route exclusion "+state.ID.ValueString()+": "+err.Error())
	}
}

func (r *routeExclusionResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// upsert builds the API object from the plan, writes it, and copies back the
// server-computed version.
func (r *routeExclusionResource) upsert(ctx context.Context, plan *routeExclusionResourceModel, diags *diag.Diagnostics) {
	strategy, strategyDiags := applyStrategyToClient(plan.ApplyStrategy.ValueString(), plan.ApplyStrategyPercentage)
	diags.Append(strategyDiags...)
	if diags.HasError() {
		return
	}

	exclusion := client.BowtieRouteExclusion{
		ID:                    plan.ID.ValueString(),
		Name:                  plan.Name.ValueString(),
		CollectionID:          plan.CollectionID.ValueString(),
		Sites:                 sitesToClient(ctx, plan.Sites, diags),
		ApplyStrategy:         strategy,
		OnlyIfWANMatchesCIDRs: listToStrings(ctx, plan.OnlyIfWANMatchesCIDRs, diags),
		MatchOnlyDeviceOS:     descriptionPointer(plan.MatchOnlyDeviceOS),
		MatchOnlyDeviceType:   descriptionPointer(plan.MatchOnlyDeviceType),
		MatchOnlyOwnership:    descriptionPointer(plan.MatchOnlyOwnership),
		MatchOnlyDeviceGroups: listToStrings(ctx, plan.MatchOnlyDeviceGroups, diags),
		MatchOnlyUserGroups:   listToStrings(ctx, plan.MatchOnlyUserGroups, diags),
	}
	if diags.HasError() {
		return
	}

	saved, err := r.client.UpsertRouteExclusion(exclusion)
	if err != nil {
		diags.AddError("Failed to write route exclusion", "Unexpected error writing route exclusion "+plan.ID.ValueString()+": "+err.Error())
		return
	}

	plan.Version = types.StringValue(saved.Version)
}

func exclusionToModel(ctx context.Context, exclusion client.BowtieRouteExclusion) (routeExclusionResourceModel, diag.Diagnostics) {
	var diags diag.Diagnostics

	strategy, percentage := applyStrategyToModel(exclusion.ApplyStrategy)

	model := routeExclusionResourceModel{
		ID:                      types.StringValue(exclusion.ID),
		Name:                    types.StringValue(exclusion.Name),
		CollectionID:            types.StringValue(exclusion.CollectionID),
		Sites:                   sitesToModel(ctx, exclusion.Sites, &diags),
		ApplyStrategy:           types.StringValue(strategy),
		ApplyStrategyPercentage: percentage,
		OnlyIfWANMatchesCIDRs:   stringsToList(ctx, exclusion.OnlyIfWANMatchesCIDRs, &diags),
		MatchOnlyDeviceOS:       stringFromPointer(exclusion.MatchOnlyDeviceOS),
		MatchOnlyDeviceType:     stringFromPointer(exclusion.MatchOnlyDeviceType),
		MatchOnlyOwnership:      stringFromPointer(exclusion.MatchOnlyOwnership),
		MatchOnlyDeviceGroups:   stringsToList(ctx, exclusion.MatchOnlyDeviceGroups, &diags),
		MatchOnlyUserGroups:     stringsToList(ctx, exclusion.MatchOnlyUserGroups, &diags),
		Version:                 types.StringValue(exclusion.Version),
	}

	return model, diags
}

func sitesToClient(ctx context.Context, sites types.List, diags *diag.Diagnostics) client.BowtieSiteDefinition {
	if sites.IsNull() || sites.IsUnknown() {
		return client.BowtieSiteDefinition{Type: "all"}
	}
	return client.BowtieSiteDefinition{Type: "specific", Value: listToStrings(ctx, sites, diags)}
}

func sitesToModel(ctx context.Context, sites client.BowtieSiteDefinition, diags *diag.Diagnostics) types.List {
	if sites.Type == "specific" {
		return stringsToList(ctx, sites.Value, diags)
	}
	return types.ListNull(types.StringType)
}

func applyStrategyToClient(strategy string, percentage types.Int64) (client.BowtieApplyStrategy, diag.Diagnostics) {
	var diags diag.Diagnostics
	isPercentage := strategy == strategyPercentageUserMatch || strategy == strategyPercentageDeviceMatch

	if isPercentage && (percentage.IsNull() || percentage.IsUnknown()) {
		diags.AddAttributeError(
			path.Root("apply_strategy_percentage"),
			"Missing percentage",
			"apply_strategy_percentage is required when apply_strategy is a percentage strategy.",
		)
		return client.BowtieApplyStrategy{}, diags
	}
	if !isPercentage && !percentage.IsNull() {
		diags.AddAttributeError(
			path.Root("apply_strategy_percentage"),
			"Unexpected percentage",
			"apply_strategy_percentage is only valid when apply_strategy is a percentage strategy.",
		)
		return client.BowtieApplyStrategy{}, diags
	}

	switch strategy {
	case strategyAlways:
		return client.BowtieApplyStrategy{Type: "always"}, diags
	case strategyNever:
		return client.BowtieApplyStrategy{Type: "never"}, diags
	case strategyPercentageUserMatch:
		value := int(percentage.ValueInt64())
		return client.BowtieApplyStrategy{Type: "percentage-user-match", Value: &value}, diags
	case strategyPercentageDeviceMatch:
		value := int(percentage.ValueInt64())
		return client.BowtieApplyStrategy{Type: "percentage-device-match", Value: &value}, diags
	default:
		diags.AddAttributeError(path.Root("apply_strategy"), "Unknown strategy", fmt.Sprintf("Unknown apply_strategy %q.", strategy))
		return client.BowtieApplyStrategy{}, diags
	}
}

func applyStrategyToModel(strategy client.BowtieApplyStrategy) (string, types.Int64) {
	switch strategy.Type {
	case "never":
		return strategyNever, types.Int64Null()
	case "percentage-user-match":
		return strategyPercentageUserMatch, percentageValue(strategy.Value)
	case "percentage-device-match":
		return strategyPercentageDeviceMatch, percentageValue(strategy.Value)
	default:
		return strategyAlways, types.Int64Null()
	}
}

func percentageValue(value *int) types.Int64 {
	if value == nil {
		return types.Int64Null()
	}
	return types.Int64Value(int64(*value))
}

// listToStrings converts a list attribute into a non-nil slice. A null list
// becomes an empty slice so the upsert clears the field rather than preserving
// the previous value.
func listToStrings(ctx context.Context, list types.List, diags *diag.Diagnostics) []string {
	out := []string{}
	if list.IsNull() || list.IsUnknown() {
		return out
	}
	diags.Append(list.ElementsAs(ctx, &out, false)...)
	return out
}

// stringsToList converts a server slice back into a list attribute, mapping an
// empty slice to null so it matches an omitted configuration.
func stringsToList(ctx context.Context, values []string, diags *diag.Diagnostics) types.List {
	if len(values) == 0 {
		return types.ListNull(types.StringType)
	}
	list, d := types.ListValueFrom(ctx, types.StringType, values)
	diags.Append(d...)
	return list
}
