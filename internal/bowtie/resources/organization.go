package resources

import (
	"context"
	"time"

	"github.com/bowtieworks/terraform-provider-bowtie/internal/bowtie/client"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &organizationResource{}
var _ resource.ResourceWithImportState = &organizationResource{}

type organizationResource struct {
	client *client.Client
}

type organizationResourceModel struct {
	ID          types.String `tfsdk:"id"`
	LastUpdated types.String `tfsdk:"last_updated"`
	Name        types.String `tfsdk:"name"`
	Domain      types.String `tfsdk:"domain"`
}

func NewOrganizationResource() resource.Resource {
	return &organizationResource{}
}

func (org *organizationResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_organization"
}

func (org *organizationResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `
Manage organization information.

**Note**: Organizations are always represented in the Bowtie API and cannot be created or destroyed.
This resource will **fail** with an error if any Terraform actions try to delete or create organizations.
Instead, you should use an [import](https://developer.hashicorp.com/terraform/language/import) block (or ` + "`terraform import <target>`" + ` command) to import the already-existing organization which you can then configure normally.
If you need to remove an organization from your Terraform state, you may remove it with the ` + "`terraform state rm <target>`" + ` command.
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
				MarkdownDescription: "The last time this object was change by Terraform. This field is _not part of the Bowtie API_ but rather extra provider metadata.",
				Computed:            true,
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The human readable name of the organization.",
			},
			"domain": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Domain to associate with this organization.",
			},
		},
	}
}

func (org *organizationResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Incorrect provider data",
			"The provider data was not appropriate and failed to resolve as *client.Client",
		)
	}

	org.client = client
}

func (org *organizationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	resp.Diagnostics.AddError(
		"Organization creation is not supported.",
		"Please instead use an import block or import command if you would like to manage the existing organization.",
	)
}

func (org *organizationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state organizationResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	org_response, err := org.client.GetOrganization()
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed retrieving organization information.",
			err.Error(),
		)
		return
	}

	state.Name = types.StringValue(org_response.Name)
	state.Domain = types.StringValue(org_response.Domain)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (org *organizationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan organizationResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := org.client.UpsertOrganization(
		plan.Name.ValueString(),
		plan.Domain.ValueString(),
	)

	if err != nil {
		resp.Diagnostics.AddError("Failed updating organization", err.Error())
		return
	}

	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (org *organizationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	resp.Diagnostics.AddError(
		"Organization destruction is not supported.",
		"Please instead remove the resource from your terraform state as Bowtie organizations cannot be removed.",
	)
}

func (org *organizationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
