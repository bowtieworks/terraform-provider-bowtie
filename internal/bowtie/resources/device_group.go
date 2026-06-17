package resources

import (
	"context"
	"fmt"

	"github.com/bowtieworks/terraform-provider-bowtie/internal/bowtie/client"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &deviceGroupResource{}
var _ resource.ResourceWithImportState = &deviceGroupResource{}

type deviceGroupResource struct {
	client *client.Client
}

type deviceGroupResourceModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
}

func NewDeviceGroupResource() resource.Resource {
	return &deviceGroupResource{}
}

func (d *deviceGroupResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_device_group"
}

func (d *deviceGroupResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "A named group of devices. Device groups are referenced by policy sources (`device_group`) to grant or deny access based on device membership rather than per-device.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Internal device group ID.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The human readable name of the device group.",
				Required:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "An optional description of the device group.",
				Optional:            true,
			},
		},
	}
}

func (d *deviceGroupResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

	d.client = c
}

func (d *deviceGroupResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan deviceGroupResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if plan.ID.ValueString() == "" {
		plan.ID = types.StringValue(uuid.NewString())
	}

	err := d.client.UpsertDeviceGroup(plan.ID.ValueString(), plan.Name.ValueString(), descriptionPointer(plan.Description))
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to create device group",
			"Unexpected error creating device group: "+err.Error(),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (d *deviceGroupResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state deviceGroupResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	groups, err := d.client.GetDeviceGroups()
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to read device group",
			"Unexpected error reading device group "+state.ID.ValueString()+": "+err.Error(),
		)
		return
	}

	group, present := groups[state.ID.ValueString()]
	if !present {
		resp.Diagnostics.AddAttributeWarning(
			path.Root("id"),
			"Device group not found, removing from state",
			state.ID.ValueString(),
		)
		resp.State.RemoveResource(ctx)
		return
	}

	state.Name = types.StringValue(group.Name)
	if group.Description != nil && *group.Description != "" {
		state.Description = types.StringValue(*group.Description)
	} else {
		state.Description = types.StringNull()
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (d *deviceGroupResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan deviceGroupResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := d.client.UpsertDeviceGroup(plan.ID.ValueString(), plan.Name.ValueString(), descriptionPointer(plan.Description))
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to update device group",
			"Unexpected error updating device group "+plan.ID.ValueString()+": "+err.Error(),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (d *deviceGroupResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state deviceGroupResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := d.client.DeleteDeviceGroup(state.ID.ValueString()); err != nil {
		resp.Diagnostics.AddError(
			"Failed to delete device group",
			"Unexpected error deleting device group "+state.ID.ValueString()+": "+err.Error(),
		)
	}
}

func (d *deviceGroupResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// descriptionPointer converts an optional string attribute into the pointer the
// API expects: a null attribute clears the field on the Controller.
func descriptionPointer(value types.String) *string {
	if value.IsNull() || value.IsUnknown() {
		return nil
	}
	s := value.ValueString()
	return &s
}

// stringFromPointer converts an optional server string into an attribute value.
func stringFromPointer(value *string) types.String {
	if value == nil {
		return types.StringNull()
	}
	return types.StringValue(*value)
}
