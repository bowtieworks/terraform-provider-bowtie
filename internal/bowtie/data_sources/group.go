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
	_ datasource.DataSource              = &groupDataSource{}
	_ datasource.DataSourceWithConfigure = &groupDataSource{}
)

func NewGroupDataSource() datasource.DataSource {
	return &groupDataSource{}
}

type groupDataSource struct {
	client *client.Client
}

type groupDataSourceModel struct {
	ID   types.String `tfsdk:"id"`
	Name types.String `tfsdk:"name"`
}

func (d *groupDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_group"
}

func (d *groupDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Look up an existing user group by name, for example to reference its ID in a policy source (`user_group`).",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Internal user group ID.",
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The name of the user group to look up.",
			},
		},
	}
}

func (d *groupDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *groupDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state groupDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	groups, err := d.client.GetGroups()
	if err != nil {
		resp.Diagnostics.AddError("Failed to read user groups", err.Error())
		return
	}

	name := state.Name.ValueString()
	var match *client.Group
	for _, group := range groups {
		group := group
		if group.Name == name {
			if match != nil {
				resp.Diagnostics.AddError("Ambiguous user group", fmt.Sprintf("More than one user group is named %q.", name))
				return
			}
			match = &group
		}
	}
	if match == nil {
		resp.Diagnostics.AddError("User group not found", fmt.Sprintf("No user group is named %q.", name))
		return
	}

	state.ID = types.StringValue(match.ID)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
