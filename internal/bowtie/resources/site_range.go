package resources

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/bowtieworks/terraform-provider-bowtie/internal/bowtie/client"
	"github.com/hashicorp/terraform-plugin-framework-validators/resourcevalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &siteRangeResource{}
var _ resource.ResourceWithImportState = &siteRangeResource{}

type siteRangeResource struct {
	client *client.Client
}

type siteRangeResourceModel struct {
	ID          types.String `tfsdk:"id"`
	SiteID      types.String `tfsdk:"site_id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	IPV4Range   types.String `tfsdk:"ipv4_range"`
	IPV6Range   types.String `tfsdk:"ipv6_range"`
	Weight      types.Int64  `tfsdk:"weight"`
	Metric      types.Int64  `tfsdk:"metric"`
	LastUpdated types.String `tfsdk:"last_updated"`
}

func NewSiteRangeResource() resource.Resource {
	return &siteRangeResource{}
}

func (sr *siteRangeResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_site_range"
}

func (sr *siteRangeResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `
Site *ranges* declare which addresses, if any, a given site is capable of serving.

A given site may be associated with more than one range.
`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Internal resource ID.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"last_updated": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Provider metadata for when the last update was performed via Terraform for this resource.",
			},
			"site_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The Site ID that this range should be associated with.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The human readable name of this range.",
			},
			"description": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Long-form description for this site.",
			},
			"ipv4_range": schema.StringAttribute{
				MarkdownDescription: "The IPv4 CIDR range for this site range. **Mutually exclusive with `ipv6_range`**.",
				Optional:            true,
			},
			"ipv6_range": schema.StringAttribute{
				MarkdownDescription: "The IPv6 CIDR range for this site range. **Mutually exclusive with `ipv4_range`**.",
				Optional:            true,
			},
			"weight": schema.Int64Attribute{
				MarkdownDescription: "The weight for this range. Currently unused but may be in future updates.",
				Computed:            true,
				Optional:            true,
				Default:             int64default.StaticInt64(1),
			},
			"metric": schema.Int64Attribute{
				MarkdownDescription: "The metric for this range. Currently unused but may be in future updates.",
				Computed:            true,
				Optional:            true,
				Default:             int64default.StaticInt64(255),
			},
		},
	}
}

func (sr *siteRangeResource) ConfigValidators(ctx context.Context) []resource.ConfigValidator {
	return []resource.ConfigValidator{
		resourcevalidator.ExactlyOneOf(
			path.MatchRoot("ipv4_range"),
			path.MatchRoot("ipv6_range"),
		),
	}
}

func (sr *siteRangeResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unknown provider data, not of expected type",
			"Casting the ProviderData to *client.Client failed. Please contact provider maintainers to report this bug",
		)
	}

	sr.client = client
}

func (sr *siteRangeResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan siteRangeResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var is_ipv4 bool
	var is_ipv6 bool
	var cidr string
	if !plan.IPV4Range.IsNull() {
		is_ipv4 = true
		cidr = plan.IPV4Range.ValueString()
	} else if !plan.IPV6Range.IsNull() {
		is_ipv6 = true
		cidr = plan.IPV6Range.ValueString()
	} else {
		resp.Diagnostics.AddError(
			"Failed to correctly configure requests",
			"Resource was unable to identify the cidr type",
		)
		return
	}

	id, err := sr.client.CreateSiteRange(plan.SiteID.ValueString(), plan.Name.ValueString(), plan.Description.ValueString(), cidr, is_ipv4, is_ipv6, plan.Weight.ValueInt64(), plan.Metric.ValueInt64())
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to create the site range",
			"Unexpected error calling the bowtie api: "+err.Error(),
		)
		return
	}

	plan.ID = types.StringValue(id)
	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (sr *siteRangeResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state siteRangeResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	info, err := sr.client.GetSiteRange(state.SiteID.ValueString(), state.ID.ValueString())
	if err != nil {
		/*resp.Diagnostics.AddError(
			"Failed to retrieve site range info from the bowtie server",
			"Unexpected error reading from the bowtie server error: "+err.Error(),
		)*/
		resp.State.RemoveResource(ctx)
		return
	}

	state.Name = types.StringValue(info.Name)
	state.Description = types.StringValue(info.Description)
	state.Weight = types.Int64Value(info.Weight)
	state.Metric = types.Int64Value(info.Metric)

	if info.ISV6 {
		state.IPV6Range = types.StringValue(info.Range)
	} else if info.ISV4 {
		state.IPV4Range = types.StringValue(info.Range)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (sr *siteRangeResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan siteRangeResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var is_ipv4 bool
	var is_ipv6 bool
	var cidr string
	if !(plan.IPV4Range.IsNull() && plan.IPV4Range.IsUnknown()) {
		is_ipv4 = true
		cidr = plan.IPV4Range.ValueString()
	} else if !(plan.IPV6Range.IsNull() && plan.IPV6Range.IsUnknown()) {
		is_ipv6 = true
		cidr = plan.IPV6Range.ValueString()
	}

	err := sr.client.UpsertSiteRange(plan.SiteID.ValueString(), plan.ID.ValueString(), plan.Name.ValueString(), plan.Description.ValueString(), cidr, is_ipv4, is_ipv6, plan.Weight.ValueInt64(), plan.Metric.ValueInt64())
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed updating site range info",
			"Unexpected error updating site range. error: "+err.Error(),
		)
		return
	}

	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (sr *siteRangeResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state siteRangeResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := sr.client.DeleteSiteRange(state.SiteID.ValueString(), state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed deleting site range",
			"Unexpected error communicating with bowtie during delete site range error: "+err.Error(),
		)
	}
}

func (sr *siteRangeResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	idParts := strings.Split(req.ID, ":")

	if len(idParts) != 2 || idParts[0] == "" || idParts[1] == "" {
		resp.Diagnostics.AddError(
			"Unexpected Import Identifier",
			fmt.Sprintf("Expected import identifier with format: site_id:id. Got: %q", req.ID),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("site_id"), idParts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), idParts[1])...)
}
