package resources

import (
	"context"

	"github.com/bowtieworks/terraform-provider-bowtie/internal/bowtie/client"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &UserResource{}
var _ resource.ResourceWithImportState = &TemplateResource{}

type UserResource struct {
	client *client.Client
}

type UserResourceModel struct {
	ID                types.String `tfsdk:"id"`
	Name              types.String `tfsdk:"name"`
	Email             types.String `tfsdk:"email"`
	AuthzDevices      types.Bool   `tfsdk:"authz_devices"`
	AuthzPolicies     types.Bool   `tfsdk:"authz_policies"`
	AuthzControlPlane types.Bool   `tfsdk:"authz_control_plane"`
	AuthzUsers        types.Bool   `tfsdk:"authz_users"`
	Enabled           types.Bool   `tfsdk:"enabled"`
	Role              types.String `tfsdk:"role"`
}

func NewUserResource() resource.Resource {
	return &UserResource{}
}

func (u *UserResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_user"
}

func (u *UserResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manage users, including individual user permissions and status.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Internal resource ID.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The user's name.",
			},
			"email": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The user's email.",
			},
			"authz_devices": schema.BoolAttribute{
				Computed: true,
				Optional: true,
				Default:  booldefault.StaticBool(false),
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
				MarkdownDescription: "Grants the user access to the Devices UI and API.",
			},
			"authz_policies": schema.BoolAttribute{
				Computed: true,
				Optional: true,
				Default:  booldefault.StaticBool(false),
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
				MarkdownDescription: "Grants the user access to the Policies UI and API.",
			},
			"authz_users": schema.BoolAttribute{
				Computed: true,
				Optional: true,
				Default:  booldefault.StaticBool(false),
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
				MarkdownDescription: "Grants the user access to the Users UI and API.",
			},
			"authz_control_plane": schema.BoolAttribute{
				Computed: true,
				Optional: true,
				Default:  booldefault.StaticBool(false),
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
				MarkdownDescription: "Grants the user access to the Control Plane UI and API.",
			},
			"role": schema.StringAttribute{
				Computed:            true,
				Optional:            true,
				Default:             stringdefault.StaticString("User"),
				MarkdownDescription: "What role the user is assigned. Value must be one of `Ownder`, `User`, `FullAdministrator`, or `LimitedAdministrator`.",
				Validators: []validator.String{
					stringvalidator.OneOf("Owner", "User", "LimitedAdministrator", "FullAdministrator"),
				},
			},
			"enabled": schema.BoolAttribute{
				Computed:            true,
				Optional:            true,
				Default:             booldefault.StaticBool(true),
				MarkdownDescription: "Configures if the user is `Active` or `Disabled`.",
			},
		},
	}
}

func (u *UserResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

	u.client = client
}

func (u *UserResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan UserResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if plan.ID.ValueString() == "" {
		plan.ID = types.StringValue(uuid.NewString())
	}

	_, err := u.client.UpsertUser(
		plan.ID.ValueString(),
		plan.Name.ValueString(),
		plan.Email.ValueString(),
		plan.Role.ValueString(),
		plan.AuthzPolicies.ValueBool(),
		plan.AuthzUsers.ValueBool(),
		plan.AuthzControlPlane.ValueBool(),
		plan.AuthzDevices.ValueBool(),
		plan.Enabled.ValueBool(),
	)

	if err != nil {
		resp.Diagnostics.AddError(
			"Failed creating user",
			"Unexpected error craeting the user: "+err.Error(),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (u *UserResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state UserResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	user, err := u.client.GetUser(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed reading the user: "+state.ID.ValueString(),
			"Unexpected error reading the user: "+err.Error(),
		)
	}

	state.Name = types.StringValue(user.Name)
	state.Email = types.StringValue(user.Email)
	state.Role = types.StringValue(user.Role)

	state.Enabled = types.BoolValue(user.Status == "Active")

	state.AuthzControlPlane = types.BoolValue(*user.AuthzControlPlane)
	state.AuthzDevices = types.BoolValue(*user.AuthzDevices)
	state.AuthzPolicies = types.BoolValue(*user.AuthzPolicies)
	state.AuthzUsers = types.BoolValue(*user.AuthzUsers)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (u *UserResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan UserResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := u.client.UpsertUser(
		plan.ID.ValueString(),
		plan.Name.ValueString(),
		plan.Email.ValueString(),
		plan.Role.ValueString(),
		plan.AuthzPolicies.ValueBool(),
		plan.AuthzUsers.ValueBool(),
		plan.AuthzControlPlane.ValueBool(),
		plan.AuthzDevices.ValueBool(),
		plan.Enabled.ValueBool(),
	)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to update the user: "+plan.ID.ValueString(),
			"Unexpected error updating the user"+err.Error(),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (u *UserResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var plan UserResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := u.client.DeleteUser(ctx, plan.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to delete user: "+plan.ID.ValueString(),
			"Unexpected error deleting user: "+err.Error(),
		)
	}
}

func (u *UserResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
