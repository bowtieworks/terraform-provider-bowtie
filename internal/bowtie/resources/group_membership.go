package resources

import (
	"context"

	"github.com/bowtieworks/terraform-provider-bowtie/internal/bowtie/client"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &GroupMembershipResource{}
var _ resource.ResourceWithImportState = &GroupMembershipResource{}

type GroupMembershipResource struct {
	client *client.Client
}

type groupMembershipResourceModel struct {
	GroupID types.String `tfsdk:"group_id"`
	Users   types.Set    `tfsdk:"users"`
}

func NewGroupMembershipResource() resource.Resource {
	return &GroupMembershipResource{}
}

func (g *GroupMembershipResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_group_membership"
}

func (g *GroupMembershipResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Used to set the membership of a group. Will remove any users not represented in the users array. Each group can only be associated with a single membership resource.",
		Attributes: map[string]schema.Attribute{
			"group_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Internal resource ID.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"users": schema.SetAttribute{
				ElementType:         types.StringType,
				Required:            true,
				MarkdownDescription: "The list of users to grant membership to the group. This resource accepts both `user_ids` and emails. Will completely overwrite membership on apply.",
			},
		},
	}
}

func (g *GroupMembershipResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Incorrect provider data",
			"The provider data was not appropiate and failed to resolve as *client.Client",
		)
	}

	g.client = client
}

func (g *GroupMembershipResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan groupMembershipResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var users []string
	resp.Diagnostics.Append(plan.Users.ElementsAs(ctx, &users, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := g.client.SetGroupMembership(plan.GroupID.ValueString(), users)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to set group membership",
			"Unexpected error setting group membership: "+plan.GroupID.ValueString()+" err: "+err.Error(),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (g *GroupMembershipResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var plan groupMembershipResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	groupInfo, err := g.client.ListUsersInGroup(plan.GroupID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed listing users in group",
			"Unexpected error listing users in group: "+plan.GroupID.ValueString()+" err: "+err.Error(),
		)
		return
	}

	stateUsers, diags := types.SetValueFrom(ctx, types.StringType, groupInfo.Users)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	plan.Users = stateUsers

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (g *GroupMembershipResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan groupMembershipResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var users []string
	resp.Diagnostics.Append(plan.Users.ElementsAs(ctx, &users, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := g.client.SetGroupMembership(plan.GroupID.ValueString(), users)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to set group membership",
			"Unexpected error setting group membership: "+plan.GroupID.ValueString()+" err: "+err.Error(),
		)
		return
	}

	stateUsers, diags := types.SetValueFrom(ctx, types.StringType, users)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	plan.Users = stateUsers
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)

}

func (g *GroupMembershipResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var plan groupMembershipResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := g.client.SetGroupMembership(plan.GroupID.ValueString(), []string{})
	if err != nil {
		resp.Diagnostics.AddError(
			"failed to remove all users from the group",
			"Unexpected error removing users from group: "+plan.GroupID.ValueString()+" err: "+err.Error(),
		)
	}
}

func (g *GroupMembershipResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("group_id"), req, resp)
}
