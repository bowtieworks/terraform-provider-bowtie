package resources

import (
	"context"
	"fmt"

	"github.com/bowtieworks/terraform-provider-bowtie/internal/bowtie/client"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &TemplateResource{}
var _ resource.ResourceWithImportState = &TemplateResource{}

type resourceResource struct {
	client *client.Client
}

type resourceResourceModel struct {
	ID       types.String           `tfsdk:"id"`
	Name     types.String           `tfsdk:"name"`
	Protocol types.String           `tfsdk:"protocol"`
	Location *resourceLocationModel `tfsdk:"location"`
	Ports    *resourcePortsModel    `tfsdk:"ports"`
}

type resourceLocationModel struct {
	IP   types.String `tfsdk:"ip"`
	CIDR types.String `tfsdk:"cidr"`
	DNS  types.String `tfsdk:"dns"`
}

type resourcePortsModel struct {
	Range      types.List `tfsdk:"range"`
	Collection types.List `tfsdk:"collection"`
}

func NewResourceResource() resource.Resource {
	return &resourceResource{}
}

func (r *resourceResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_resource"
}

func (r *resourceResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `
Bowtie *resources* represent network properties like address ranges that may be targeted by *policies*.

Note that defining these resources does not implicitly grant or deny access to them - resources must be collected into resource groups and then referenced by policies.`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Internal resource ID.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Human readable name of the resource.",
				Required:            true,
			},
			"protocol": schema.StringAttribute{
				MarkdownDescription: "Matching connection protocol.",
				Validators: []validator.String{
					stringvalidator.OneOf("all", "tcp", "udp", "http", "https", "icmp4", "icmp6"),
				},
				Required: true,
			},
			"location": schema.SingleNestedAttribute{
				MarkdownDescription: "The address of the resource. May be a CIDR address, single IP, or DNS name.",
				Required:            true,
				Attributes: map[string]schema.Attribute{
					"ip": schema.StringAttribute{
						MarkdownDescription: "The IP address of a resource reachable from behind your Bowtie Controller.",
						Optional:            true,
						Validators: []validator.String{
							stringvalidator.ExactlyOneOf(path.Expressions{
								path.MatchRelative().AtParent().AtName("cidr"),
								path.MatchRelative().AtParent().AtName("dns"),
							}...),
						},
					},
					"cidr": schema.StringAttribute{
						MarkdownDescription: "A CIDR address reachable from behind your Bowtie Controller.",
						Optional:            true,
					},
					"dns": schema.StringAttribute{
						MarkdownDescription: "A DNS name pointing to a resource reachable from behind your Bowtie Controller.",
						Optional:            true,
					},
				},
			},
			"ports": schema.SingleNestedAttribute{
				MarkdownDescription: "Which ports to include in this resource.",
				Required:            true,
				Attributes: map[string]schema.Attribute{
					"range": schema.ListAttribute{
						MarkdownDescription: "First element is the low port and second is the high port (range is inclusive).",
						ElementType:         types.Int64Type,
						Validators: []validator.List{
							listvalidator.SizeAtMost(2),
							listvalidator.SizeAtLeast(2),
							listvalidator.ExactlyOneOf(path.Expressions{
								path.MatchRelative().AtParent().AtName("collection"),
							}...),
						},
						Optional: true,
					},
					"collection": schema.ListAttribute{
						MarkdownDescription: "List of allowed ports.",
						ElementType:         types.Int64Type,
						Validators: []validator.List{
							listvalidator.UniqueValues(),
						},
						Optional: true,
					},
				},
			},
		},
	}
}

func (r *resourceResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configuration Type",
			fmt.Sprintf("Expected *client.Client, got: %T, please report this to the provider.", req.ProviderData),
		)
	}

	r.client = client
}

func (r *resourceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan resourceResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var portsRange []int64
	var portsCollection []int64
	if !plan.Ports.Range.IsNull() {
		portsRange = []int64{}
		plan.Ports.Range.ElementsAs(ctx, &portsRange, true)
	} else if !plan.Ports.Collection.IsNull() {
		portsCollection = []int64{}
		plan.Ports.Collection.ElementsAs(ctx, &portsCollection, true)
	} else {
		resp.Diagnostics.AddAttributeError(
			path.Root("ports"),
			"Ports subkeys are both unset",
			"Please ensure that either Range or Collection subkeys are set",
		)
		return
	}

	if plan.ID.ValueString() == "" {
		plan.ID = types.StringValue(uuid.NewString())
	}

	_, err := r.client.UpsertResource(
		plan.ID.ValueString(),
		plan.Name.ValueString(),
		plan.Protocol.ValueString(),
		plan.Location.IP.ValueString(),
		plan.Location.CIDR.ValueString(),
		plan.Location.DNS.ValueString(),
		portsRange,
		portsCollection,
	)

	if err != nil {
		resp.Diagnostics.AddError(
			"Unexpected error from bowtie API",
			"Failed to create resource error from the bowtie API: "+err.Error(),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *resourceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state resourceResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resources, err := r.client.GetResources()
	if err != nil {
		resp.Diagnostics.AddError(
			"Unexpected error retrieving the resource",
			"Failed to retrieve resource: "+state.ID.ValueString()+" error: "+err.Error(),
		)
		return
	}

	resource, present := resources[state.ID.ValueString()]
	if !present {
		resp.Diagnostics.AddAttributeWarning(
			path.Root("id"),
			"resource not found, removing from state",
			state.ID.ValueString(),
		)
		resp.State.RemoveResource(ctx)
		return
	}

	state.Name = types.StringValue(resource.Name)
	state.Protocol = types.StringValue(resource.Protocol)
	state.Location = &resourceLocationModel{}

	if resource.Location.CIDR != "" {
		state.Location.CIDR = types.StringValue(resource.Location.CIDR)
		state.Location.DNS = types.StringNull()
		state.Location.IP = types.StringNull()
	} else if resource.Location.IP != "" {
		state.Location.IP = types.StringValue(resource.Location.IP)
		state.Location.DNS = types.StringNull()
		state.Location.CIDR = types.StringNull()
	} else if resource.Location.DNS != "" {
		state.Location.DNS = types.StringValue(resource.Location.DNS)
		state.Location.IP = types.StringNull()
		state.Location.CIDR = types.StringNull()
	} else {
		resp.Diagnostics.AddAttributeError(
			path.Root("location"),
			"Invalid resource returned from bowtie api",
			"Unexpected location key. either wasn't set or an unexpected key was found",
		)
		return
	}

	state.Ports = &resourcePortsModel{}
	if resource.Ports.Collection != nil {
		state.Ports.Range = types.ListNull(types.Int64Type)
		collection, diags := types.ListValueFrom(ctx, types.Int64Type, resource.Ports.Collection.Ports)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		state.Ports.Collection = collection
	} else if len(resource.Ports.Range) > 0 {
		state.Ports.Collection = types.ListNull(types.Int64Type)
		val, diags := types.ListValueFrom(ctx, types.Int64Type, resource.Ports.Range)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		state.Ports.Range = val
	} else {
		resp.Diagnostics.AddAttributeError(
			path.Root("ports"),
			"Invalid resource returned from the bowtie api",
			"Unexpected ports key. either expected key was set or an unexpected key was set",
		)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *resourceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan resourceResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var portsRange []int64
	var portsCollection []int64
	if !plan.Ports.Range.IsNull() {
		portsRange = []int64{}
		plan.Ports.Range.ElementsAs(ctx, &portsRange, true)
	} else if !plan.Ports.Collection.IsNull() {
		portsCollection = []int64{}
		plan.Ports.Collection.ElementsAs(ctx, &portsCollection, true)
	} else {
		resp.Diagnostics.AddAttributeError(
			path.Root("ports"),
			"Ports subkeys are both unset",
			"Please ensure that either Range or Collection subkeys are set",
		)
		return
	}

	_, err := r.client.UpsertResource(
		plan.ID.ValueString(),
		plan.Name.ValueString(),
		plan.Protocol.ValueString(),
		plan.Location.IP.ValueString(),
		plan.Location.CIDR.ValueString(),
		plan.Location.DNS.ValueString(),
		portsRange,
		portsCollection,
	)

	if err != nil {
		resp.Diagnostics.AddError(
			"Failed updating resource",
			"Unexpected error updating resource: "+plan.ID.ValueString()+" error: "+err.Error(),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *resourceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var plan resourceResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteResource(plan.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"deleting resource failed",
			"Unexpected error calling bowtie api to delete resource: "+plan.ID.ValueString()+" error: "+err.Error(),
		)
	}
}

func (r *resourceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
