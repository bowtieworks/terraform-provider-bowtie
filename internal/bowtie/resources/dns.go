package resources

import (
	"context"
	"fmt"
	"time"

	"github.com/bowtieworks/terraform-provider-bowtie/internal/bowtie/client"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &dnsResource{}
var _ resource.ResourceWithImportState = &dnsResource{}

type dnsResource struct {
	client *client.Client
}

type dnsResourceModel struct {
	ID               types.String              `tfsdk:"id"`
	LastUpdated      types.String              `tfsdk:"last_updated"`
	Name             types.String              `tfsdk:"name"`
	Servers          []dnsServersResourceModel `tfsdk:"servers"`
	IncludeOnlySites types.List                `tfsdk:"include_only_sites"`
	IsCounted        types.Bool                `tfsdk:"is_counted"`
	IsDNS64          types.Bool                `tfsdk:"is_dns64"`
	IsLog            types.Bool                `tfsdk:"is_log"`
	IsDropA          types.Bool                `tfsdk:"is_drop_a"`
	IsDropAll        types.Bool                `tfsdk:"is_drop_all"`
	IsSearchDomain   types.Bool                `tfsdk:"is_search_domain"`
	DNS64Exclude     []dnsExcludeResourceModel `tfsdk:"excludes"`
}

type dnsServersResourceModel struct {
	ID    types.String `tfsdk:"id"`
	Addr  types.String `tfsdk:"addr"`
	Order types.Int64  `tfsdk:"order"`
}

type dnsExcludeResourceModel struct {
	ID    types.String `tfsdk:"id"`
	Name  types.String `tfsdk:"name"`
	Order types.Int64  `tfsdk:"order"`
}

func NewDNSResource() resource.Resource {
	return &dnsResource{}
}

func (d *dnsResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dns"
}

func (d *dnsResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The ID of the dns settings",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"last_updated": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Metadata about the last time a write api was called by this provider for this resource",
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The DNS zone name you wish to target",
			},
			"servers": schema.ListNestedAttribute{
				MarkdownDescription: "Provider Metadata storing extra API data about the server settings",
				Required:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							MarkdownDescription: "The bowtie ID for this dns server",
							Computed:            true,
							PlanModifiers: []planmodifier.String{
								stringplanmodifier.UseStateForUnknown(),
							},
						},
						"addr": schema.StringAttribute{
							MarkdownDescription: "The IP address for this dns server",
							Required:            true,
						},
						"order": schema.Int64Attribute{
							MarkdownDescription: "The order for this dns server",
							Computed:            true,
							PlanModifiers: []planmodifier.Int64{
								int64planmodifier.UseStateForUnknown(),
							},
						},
					},
				},
			},
			"include_only_sites": schema.ListAttribute{
				ElementType:         types.StringType,
				Optional:            true,
				MarkdownDescription: "The sites you only want this dns to be responsible for",
			},
			"is_dns64": schema.BoolAttribute{
				Optional:            true,
				MarkdownDescription: "Is Counted var",
			},
			"is_counted": schema.BoolAttribute{
				Default:             booldefault.StaticBool(true),
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Is Counted var",
			},
			"is_log": schema.BoolAttribute{
				Default:             booldefault.StaticBool(false),
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Is Log Var",
			},
			"is_drop_a": schema.BoolAttribute{
				Default:             booldefault.StaticBool(true),
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Whether to drop the A record or not",
			},
			"is_drop_all": schema.BoolAttribute{
				Default:             booldefault.StaticBool(false),
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Should all records be dropped",
			},
			"is_search_domain": schema.BoolAttribute{
				Default:             booldefault.StaticBool(false),
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "should be treated as a search domain",
			},
			"excludes": schema.ListNestedAttribute{
				MarkdownDescription: "Provider Metadata storing extra API information about the exclude settings",
				Optional:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							MarkdownDescription: "",
							Computed:            true,
							PlanModifiers: []planmodifier.String{
								stringplanmodifier.UseStateForUnknown(),
							},
						},
						"name": schema.StringAttribute{
							MarkdownDescription: "",
							Required:            true,
						},
						"order": schema.Int64Attribute{
							MarkdownDescription: "",
							Computed:            true,
							PlanModifiers: []planmodifier.Int64{
								int64planmodifier.UseStateForUnknown(),
							},
						},
					},
				},
			},
		},
	}
}

func (d *dnsResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

	d.client = client
}

func (d *dnsResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan dnsResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	servers := []client.Server{}
	for order, server := range plan.Servers {
		servers = append(servers, client.Server{
			ID:    uuid.NewString(),
			Addr:  server.Addr.ValueString(),
			Order: int64(order),
		})
	}

	var includeSites []string
	resp.Diagnostics.Append(plan.IncludeOnlySites.ElementsAs(ctx, &includeSites, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	excludes := []client.DNSExclude{}
	for order, exclude := range plan.DNS64Exclude {
		excludes = append(excludes, client.DNSExclude{
			ID:    uuid.NewString(),
			Name:  exclude.Name.ValueString(),
			Order: int64(order),
		})
	}

	id, err := d.client.CreateDNS(plan.Name.ValueString(), servers, includeSites, plan.IsCounted.ValueBool(), plan.IsLog.ValueBool(), plan.IsDropA.ValueBool(), plan.IsDropAll.ValueBool(), plan.IsSearchDomain.ValueBool(), excludes)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed talking to bowtie server",
			"Unexpected error creating dns setting: "+err.Error(),
		)
	}

	plan.ID = types.StringValue(id)
	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

	plan.Servers = []dnsServersResourceModel{}
	for _, server := range servers {
		plan.Servers = append(plan.Servers, dnsServersResourceModel{
			ID:    types.StringValue(server.ID),
			Addr:  types.StringValue(server.Addr),
			Order: types.Int64Value(server.Order),
		})
	}

	plan.DNS64Exclude = []dnsExcludeResourceModel{}
	for _, exclude := range excludes {
		plan.DNS64Exclude = append(plan.DNS64Exclude, dnsExcludeResourceModel{
			ID:    types.StringValue(exclude.ID),
			Name:  types.StringValue(exclude.Name),
			Order: types.Int64Value(exclude.Order),
		})
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (d *dnsResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state dnsResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	dns, err := d.client.GetDNS(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed communicating with the bowtie api",
			"Unexpected error reading DNS settings: "+err.Error(),
		)
		return
	}

	state.Servers = []dnsServersResourceModel{}
	for _, v := range dns.Servers {
		state.Servers = append(state.Servers, dnsServersResourceModel{
			ID:    types.StringValue(v.ID),
			Addr:  types.StringValue(v.Addr),
			Order: types.Int64Value(v.Order),
		})
	}

	state.DNS64Exclude = []dnsExcludeResourceModel{}
	for _, v := range dns.DNS64Exclude {
		state.DNS64Exclude = append(state.DNS64Exclude, dnsExcludeResourceModel{
			ID:    types.StringValue(v.ID),
			Name:  types.StringValue(v.Name),
			Order: types.Int64Value(v.Order),
		})
	}

	includeSites, diags := types.ListValueFrom(ctx, types.StringType, dns.IncludeOnlySites)
	if diags.HasError() {
		return
	}

	state.IncludeOnlySites = includeSites

	state.Name = types.StringValue(dns.Name)

	state.IsCounted = types.BoolValue(dns.IsCounted)
	state.IsDropA = types.BoolValue(dns.IsDropA)
	state.IsDropAll = types.BoolValue(dns.IsDropAll)
	state.IsLog = types.BoolValue(dns.IsLog)
	state.IsSearchDomain = types.BoolValue(dns.IsSearchDomain)

	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (d *dnsResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan dnsResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var includes []string
	resp.Diagnostics.Append(plan.IncludeOnlySites.ElementsAs(ctx, &includes, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var servers []client.Server = []client.Server{}
	for _, server := range plan.Servers {
		id := server.ID.ValueString()
		if server.ID.IsUnknown() {
			id = uuid.NewString()
		}
		servers = append(servers, client.Server{
			ID:    id,
			Addr:  server.Addr.ValueString(),
			Order: server.Order.ValueInt64(),
		})
	}

	var excludes []client.DNSExclude = []client.DNSExclude{}
	for _, exclude := range plan.DNS64Exclude {
		excludes = append(excludes, client.DNSExclude{
			ID:    exclude.ID.ValueString(),
			Name:  exclude.Name.ValueString(),
			Order: exclude.Order.ValueInt64(),
		})
	}

	err := d.client.UpsertDNS(plan.ID.ValueString(), plan.Name.ValueString(), servers, includes, plan.IsCounted.ValueBool(), plan.IsLog.ValueBool(), plan.IsDropA.ValueBool(), plan.IsDropAll.ValueBool(), plan.IsSearchDomain.ValueBool(), excludes)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed updating the dns settings",
			"Unexpected Error updating the dns: "+err.Error(),
		)
		return
	}

	plan.Servers = []dnsServersResourceModel{}
	for _, server := range servers {
		plan.Servers = append(plan.Servers, dnsServersResourceModel{
			ID:    types.StringValue(server.ID),
			Addr:  types.StringValue(server.Addr),
			Order: types.Int64Value(server.Order),
		})
	}

	plan.DNS64Exclude = []dnsExcludeResourceModel{}
	for _, exclude := range excludes {
		plan.DNS64Exclude = append(plan.DNS64Exclude, dnsExcludeResourceModel{
			ID:    types.StringValue(exclude.ID),
			Name:  types.StringValue(exclude.Name),
			Order: types.Int64Value(exclude.Order),
		})
	}

	plan.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (d *dnsResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var plan dnsResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := d.client.DeleteDNS(plan.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to delete the dns settings",
			"Unexpected error communicating with bowtie api: "+err.Error(),
		)
	}
}

func (d *dnsResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func mergeServerDetails(serverList []types.String, serverDetails []dnsServersResourceModel) []client.Server {
	var result []client.Server = []client.Server{}
	for index, addr := range serverList {
		id := uuid.NewString()
		if len(serverDetails) >= index+1 {
			id = serverDetails[index].ID.ValueString()
		}
		result = append(result, client.Server{
			ID:    id,
			Addr:  addr.ValueString(),
			Order: int64(index),
		})
	}
	return result
}

func mergeExcludeDNSDetails(excludeList []types.String, excludeDetails []dnsExcludeResourceModel) []client.DNSExclude {
	var result []client.DNSExclude = []client.DNSExclude{}

	for index, name := range excludeList {
		id := uuid.NewString()
		if len(excludeDetails) >= index+1 {
			id = excludeDetails[index].ID.ValueString()
		}
		result = append(result, client.DNSExclude{
			ID:    id,
			Name:  name.ValueString(),
			Order: int64(index),
		})
	}
	return result
}
