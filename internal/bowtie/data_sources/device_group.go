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
	_ datasource.DataSource              = &deviceGroupDataSource{}
	_ datasource.DataSourceWithConfigure = &deviceGroupDataSource{}
)

func NewDeviceGroupDataSource() datasource.DataSource {
	return &deviceGroupDataSource{}
}

type deviceGroupDataSource struct {
	client *client.Client
}

type deviceGroupDataSourceModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
}

func (d *deviceGroupDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_device_group"
}

func (d *deviceGroupDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Look up an existing device group by name, for example to reference its ID in a policy source (`device_group`).",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Internal device group ID.",
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The name of the device group to look up.",
			},
			"description": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The description of the device group.",
			},
		},
	}
}

func (d *deviceGroupDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *deviceGroupDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state deviceGroupDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	groups, err := d.client.GetDeviceGroups()
	if err != nil {
		resp.Diagnostics.AddError("Failed to read device groups", err.Error())
		return
	}

	name := state.Name.ValueString()
	var match *client.BowtieDeviceGroup
	for _, group := range groups {
		group := group
		if group.Name == name {
			if match != nil {
				resp.Diagnostics.AddError("Ambiguous device group", fmt.Sprintf("More than one device group is named %q.", name))
				return
			}
			match = &group
		}
	}
	if match == nil {
		resp.Diagnostics.AddError("Device group not found", fmt.Sprintf("No device group is named %q.", name))
		return
	}

	state.ID = types.StringValue(match.ID)
	if match.Description != nil && *match.Description != "" {
		state.Description = types.StringValue(*match.Description)
	} else {
		state.Description = types.StringNull()
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
