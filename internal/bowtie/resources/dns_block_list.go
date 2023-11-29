package resources

import (
	"context"
	"net/url"
	"time"

	"github.com/bowtieworks/terraform-provider-bowtie/internal/bowtie/client"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &dnsBlockListResource{}
var _ resource.ResourceWithImportState = &dnsBlockListResource{}

type dnsBlockListResource struct {
	client *client.Client
}

type dnsBlockListResourceModel struct {
	ID              types.String `tfsdk:"id"`
	Name            types.String `tfsdk:"name"`
	LastUpdated     types.String `tfsdk:"last_updated"`
	Upstream        types.String `tfsdk:"upstream"`
	OverrideToAllow types.String `tfsdk:"override_to_allow"`
}

func NewDNSBlockListResource() resource.Resource {
	return &dnsBlockListResource{}
}

func (bl *dnsBlockListResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dns_block_list"
}

type urlValidator struct{}

func (v urlValidator) Description(ctx context.Context) string {
	return "Ensures that the given string forms a proper URL"
}

func (v urlValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v urlValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	_, err := url.Parse(req.ConfigValue.ValueString())
	if err != nil {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Malformed URL",
			"Value is not a valid URL: "+req.ConfigValue.String()+": "+err.Error(),
		)
	}
	return
}

func (bl *dnsBlockListResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `
Manage lists of DNS names that Controllers will reference to perform DNS-level blocking.

Names may be given as upstream URLs which will be retrieved periodically.
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
				MarkdownDescription: "The last time this object was change by Terraform. This field is _not part of the Bowtie API_ but rather additional provider metadata.",
				Computed:            true,
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The human readable name of the block list.",
			},
			"upstream": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "An upstream URL that returns a DNS block list.",
				Validators: []validator.String{
					&urlValidator{},
				},
			},
			"override_to_allow": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Optional list of DNS names to exclude from any retrieved DNS block lists.",
			},
		},
	}
}

func (bl *dnsBlockListResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

	bl.client = client
	// When creating block lists, they are fetched to confirm validity,
	// and so we increase the timeout to ensure that the server has time
	// to perform any requisite GETs for our blocklist URL.
	bl.client.HTTPClient.Timeout = 30 * time.Second
}

func (bl *dnsBlockListResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan dnsBlockListResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := bl.client.CreateDNSBlockList(
		plan.Name.ValueString(),
		plan.Upstream.ValueString(),
		plan.OverrideToAllow.ValueString(),
	)

	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to create DNS block list",
			"Unexpected error calling the Bowtie API: "+err.Error(),
		)
		return
	}

	plan.ID = types.StringValue(id)
	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (bl *dnsBlockListResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state dnsBlockListResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	blocklist, err := bl.client.GetDNSBlockList(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed retrieving DNS block list",
			"Unexpected error retrieving DNS block list from Bowtie API: "+err.Error(),
		)
		return
	}

	state.Name = types.StringValue(blocklist.Name)
	state.Upstream = types.StringValue(blocklist.Upstream)
	state.OverrideToAllow = types.StringValue(blocklist.OverrideToAllow)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (bl *dnsBlockListResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan dnsBlockListResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := bl.client.UpsertDNSBlockList(
		plan.ID.ValueString(),
		plan.Name.ValueString(),
		plan.Upstream.ValueString(),
		plan.OverrideToAllow.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed updating the DNS block list",
			"Unexpected error communicating with the Bowtie API: "+err.Error(),
		)
		return
	}

	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (bl *dnsBlockListResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state dnsBlockListResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := bl.client.DeleteDNSBlockList(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed deleting DNS block list",
			"Unexpected failure deleting DNS block list "+state.ID.ValueString()+": error: "+err.Error(),
		)
	}
}

func (bl *dnsBlockListResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
