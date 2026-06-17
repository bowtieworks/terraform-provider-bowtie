package resources

import (
	"context"
	"time"

	"github.com/bowtieworks/terraform-provider-bowtie/internal/bowtie/client"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &ipv4RangeResource{}
var _ resource.ResourceWithImportState = &ipv4RangeResource{}

type ipv4RangeResource struct {
	client *client.Client
}

type ipv4RangeResourceModel struct {
	ID                      types.String `tfsdk:"id"`
	LastUpdated             types.String `tfsdk:"last_updated"`
	Range                   types.String `tfsdk:"range"`
	AssignAddressesFromHere types.String `tfsdk:"assign_addresses_from_here"`
	SkipFirstNAddresses     types.Int64  `tfsdk:"skip_first_n_addresses"`
}

func NewIPv4RangeResource() resource.Resource { return &ipv4RangeResource{} }

func (r *ipv4RangeResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_ipv4_range"
}

func (r *ipv4RangeResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `
Manage an organization IPv4 address pool.

**Note**: per-site routing strategies (` + "`site_strategies`" + `) are not yet managed by this resource. Strategies configured in the Control Plane are preserved across updates and are not cleared.

Destroying this resource deletes the range with Controller cascade behavior enabled so allocations from the pool do not block deletion.
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
			"range": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The IPv4 CIDR for the pool, for example `192.0.2.0/24`.",
				Validators: []validator.String{
					cidrPrefixValidator{version: 4},
				},
			},
			"assign_addresses_from_here": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Address assignment strategy: `never`, `on-demand-assign-random`, or `always-assign-random`.",
				Validators: []validator.String{
					stringvalidator.OneOf("never", "on-demand-assign-random", "always-assign-random"),
				},
			},
			"skip_first_n_addresses": schema.Int64Attribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Number of addresses to skip at the start of the range before assigning.",
				Validators: []validator.Int64{
					int64validator.AtLeast(0),
				},
				PlanModifiers: []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
			},
		},
	}
}

func (r *ipv4RangeResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *ipv4RangeResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ipv4RangeResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	out, err := r.client.UpsertIPv4Range(&client.OrgIPv4Range{
		Range:                   plan.Range.ValueString(),
		AssignAddressesFromHere: plan.AssignAddressesFromHere.ValueString(),
		SkipFirstNAddresses:     plan.SkipFirstNAddresses.ValueInt64(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed creating IPv4 range", err.Error())
		return
	}

	plan.ID = types.StringValue(out.ID)
	plan.SkipFirstNAddresses = types.Int64Value(out.SkipFirstNAddresses)
	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ipv4RangeResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ipv4RangeResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	out, err := r.client.GetIPv4Range(state.ID.ValueString())
	if err != nil {
		if isNotFoundError(err) {
			resp.Diagnostics.AddAttributeWarning(
				path.Root("id"),
				"IPv4 range not found, removing from state",
				state.ID.ValueString(),
			)
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Failed reading IPv4 range", err.Error())
		return
	}

	state.Range = types.StringValue(out.Range)
	state.AssignAddressesFromHere = types.StringValue(out.AssignAddressesFromHere)
	state.SkipFirstNAddresses = types.Int64Value(out.SkipFirstNAddresses)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *ipv4RangeResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan ipv4RangeResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Read-modify-write: the upsert rebuilds the range from the payload, so
	// carry the existing site_strategies forward rather than clearing them.
	current, err := r.client.GetIPv4Range(plan.ID.ValueString())
	if err != nil {
		if isNotFoundError(err) {
			resp.Diagnostics.AddAttributeWarning(
				path.Root("id"),
				"IPv4 range not found, removing from state",
				plan.ID.ValueString(),
			)
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Failed reading IPv4 range before update", err.Error())
		return
	}

	current.Range = plan.Range.ValueString()
	current.AssignAddressesFromHere = plan.AssignAddressesFromHere.ValueString()
	current.SkipFirstNAddresses = plan.SkipFirstNAddresses.ValueInt64()

	out, err := r.client.UpsertIPv4Range(current)
	if err != nil {
		resp.Diagnostics.AddError("Failed updating IPv4 range", err.Error())
		return
	}

	plan.SkipFirstNAddresses = types.Int64Value(out.SkipFirstNAddresses)
	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ipv4RangeResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ipv4RangeResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeleteIPv4Range(state.ID.ValueString()); err != nil {
		if isNotFoundError(err) {
			return
		}
		resp.Diagnostics.AddError("Failed deleting IPv4 range", err.Error())
	}
}

func (r *ipv4RangeResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
