package data_sources

import (
	"context"
	"fmt"

	"github.com/bowtieworks/terraform-provider-bowtie/internal/bowtie/client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = &resourceGroupDataSource{}
	_ datasource.DataSourceWithConfigure = &resourceGroupDataSource{}
)

func NewResourceGroupDataSource() datasource.DataSource {
	return &resourceGroupDataSource{}
}

type resourceGroupDataSource struct {
	client *client.Client
}

type resourceGroupDataSourceModel struct {
	ID        types.String `tfsdk:"id"`
	Name      types.String `tfsdk:"name"`
	Inherited types.List   `tfsdk:"inherited"`
	Resources types.List   `tfsdk:"resources"`
}

func (d *resourceGroupDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_resource_group"
}

func (d *resourceGroupDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Look up an existing resource group by name, for example to reference its ID as the `dest` of a policy.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Internal resource group ID.",
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The name of the resource group to look up.",
			},
			"inherited": schema.ListAttribute{
				Computed:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "The resource groups inherited by this resource group.",
			},
			"resources": schema.ListAttribute{
				Computed:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "The resources directly included in this resource group.",
			},
		},
	}
}

func (d *resourceGroupDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configuration Type",
			fmt.Sprintf("Expected *client.Client, got: %T, please report this to the provider.", req.ProviderData),
		)
		return
	}

	d.client = c
}

func (d *resourceGroupDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state resourceGroupDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	groups, err := d.client.GetResourceGroups()
	if err != nil {
		resp.Diagnostics.AddError("Failed to read resource groups", err.Error())
		return
	}

	name := state.Name.ValueString()
	var match *client.BowtieResourceGroup
	for _, group := range groups {
		group := group
		if group.Name == name {
			if match != nil {
				resp.Diagnostics.AddError("Ambiguous resource group", fmt.Sprintf("More than one resource group is named %q.", name))
				return
			}
			match = &group
		}
	}
	if match == nil {
		resp.Diagnostics.AddError("Resource group not found", fmt.Sprintf("No resource group is named %q.", name))
		return
	}

	state.ID = types.StringValue(match.ID)

	inherited, diags := types.ListValueFrom(ctx, types.StringType, match.Inherited)
	resp.Diagnostics.Append(diags...)
	resources, diags := types.ListValueFrom(ctx, types.StringType, match.Resources)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	state.Inherited = inherited
	state.Resources = resources

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
