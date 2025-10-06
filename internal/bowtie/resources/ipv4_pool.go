package resources

import (
	"context"
	"encoding/json"
	"time"

	"github.com/bowtieworks/terraform-provider-bowtie/internal/bowtie/client"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &ipv4PoolResource{}
var _ resource.ResourceWithImportState = &ipv4PoolResource{}

type ipv4PoolResource struct {
	client *client.Client
}

type ipv4PoolResourceModel struct {
	ID                      types.String `tfsdk:"id"`
	Range                   types.String `tfsdk:"range"`
	AssignAddressesFromHere types.String `tfsdk:"assign_addresses_from_here"`
	SkipFirstNAddresses     types.Int64  `tfsdk:"skip_first_n_addresses"`
	SiteStrategies          types.String `tfsdk:"site_strategies"`
	LastUpdated             types.String `tfsdk:"last_updated"`
}

func NewIPv4PoolResource() resource.Resource {
	return &ipv4PoolResource{}
}

func (r *ipv4PoolResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_ipv4_pool"
}

func (r *ipv4PoolResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `
Manage IPv4 address pools for the organization.

IPv4 pools define ranges of addresses that can be assigned to devices.
`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Internal resource ID (UUID).",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"last_updated": schema.StringAttribute{
				MarkdownDescription: "The last time this object was changed by Terraform. This field is _not part of the Bowtie API_ but rather extra provider metadata.",
				Computed:            true,
			},
			"range": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The IPv4 CIDR range (e.g., '192.0.2.0/24').",
			},
			"assign_addresses_from_here": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Address assignment strategy. Valid values: 'never', 'on-demand-assign-random', 'always-assign-random'.",
			},
			"skip_first_n_addresses": schema.Int64Attribute{
				Required:            true,
				MarkdownDescription: "Number of addresses to skip from the beginning of the range (minimum: 0).",
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"site_strategies": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "JSON string of site strategies. Format: `{\"site-id\": {\"type\": \"nat\"}}`",
			},
		},
	}
}

func (r *ipv4PoolResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Incorrect provider data",
			"The provider data was not appropriate and failed to resolve as *client.Client",
		)
		return
	}

	r.client = client
}

func (r *ipv4PoolResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ipv4PoolResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Convert site_strategies from JSON string to client structure
	var siteStrategies map[string]client.SiteStrategy
	if !plan.SiteStrategies.IsNull() && !plan.SiteStrategies.IsUnknown() && plan.SiteStrategies.ValueString() != "" {
		if err := json.Unmarshal([]byte(plan.SiteStrategies.ValueString()), &siteStrategies); err != nil {
			resp.Diagnostics.AddError(
				"Invalid site_strategies JSON",
				"Failed to parse site_strategies: "+err.Error(),
			)
			return
		}
	}

	if plan.ID.ValueString() == "" {
		plan.ID = types.StringValue(uuid.NewString())
	}

	err := r.client.UpsertIPv4Pool(
		plan.ID.ValueString(),
		plan.Range.ValueString(),
		plan.AssignAddressesFromHere.ValueString(),
		int(plan.SkipFirstNAddresses.ValueInt64()),
		siteStrategies,
	)

	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to create IPv4 pool",
			"Unexpected error calling the Bowtie API: "+err.Error(),
		)
		return
	}

	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *ipv4PoolResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ipv4PoolResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Pull in the list of IPv4 pools from the API
	ipv4Pools, err := r.client.GetIPv4Pools()
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed retrieving IPv4 pools",
			"Unexpected error retrieving IPv4 pools from Bowtie API: "+err.Error(),
		)
		return
	}

	ipv4Pool, present := ipv4Pools[state.ID.ValueString()]
	if !present {
		resp.Diagnostics.AddAttributeWarning(
			path.Root("id"),
			"resource not found, removing from state",
			state.ID.ValueString(),
		)
		resp.State.RemoveResource(ctx)
		return
	}

	state.Range = types.StringValue(ipv4Pool.Range)
	state.AssignAddressesFromHere = types.StringValue(ipv4Pool.AssignAddressesFromHere)
	state.SkipFirstNAddresses = types.Int64Value(int64(ipv4Pool.SkipFirstNAddresses))

	// Convert site strategies to JSON string
	if ipv4Pool.SiteStrategies != nil && len(ipv4Pool.SiteStrategies) > 0 {
		strategiesJSON, err := json.Marshal(ipv4Pool.SiteStrategies)
		if err != nil {
			resp.Diagnostics.AddError(
				"Failed to marshal site strategies",
				"Error converting site strategies to JSON: "+err.Error(),
			)
			return
		}
		state.SiteStrategies = types.StringValue(string(strategiesJSON))
	} else {
		state.SiteStrategies = types.StringNull()
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *ipv4PoolResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan ipv4PoolResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Convert site_strategies from JSON string to client structure
	var siteStrategies map[string]client.SiteStrategy
	if !plan.SiteStrategies.IsNull() && !plan.SiteStrategies.IsUnknown() && plan.SiteStrategies.ValueString() != "" {
		if err := json.Unmarshal([]byte(plan.SiteStrategies.ValueString()), &siteStrategies); err != nil {
			resp.Diagnostics.AddError(
				"Invalid site_strategies JSON",
				"Failed to parse site_strategies: "+err.Error(),
			)
			return
		}
	}

	err := r.client.UpsertIPv4Pool(
		plan.ID.ValueString(),
		plan.Range.ValueString(),
		plan.AssignAddressesFromHere.ValueString(),
		int(plan.SkipFirstNAddresses.ValueInt64()),
		siteStrategies,
	)

	if err != nil {
		resp.Diagnostics.AddError(
			"Failed updating the IPv4 pool",
			"Unexpected error communicating with the Bowtie API: "+err.Error(),
		)
		return
	}

	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *ipv4PoolResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ipv4PoolResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteIPv4Pool(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed deleting IPv4 pool",
			"Unexpected failure deleting IPv4 pool "+state.ID.ValueString()+": error: "+err.Error(),
		)
	}
}

func (r *ipv4PoolResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}