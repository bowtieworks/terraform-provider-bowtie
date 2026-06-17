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
	_ datasource.DataSource              = &collectionDataSource{}
	_ datasource.DataSourceWithConfigure = &collectionDataSource{}
)

func NewCollectionDataSource() datasource.DataSource {
	return &collectionDataSource{}
}

type collectionDataSource struct {
	client *client.Client
}

type collectionDataSourceModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
}

func (d *collectionDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_collection"
}

func (d *collectionDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Look up an existing collection by name, for example to reference its ID from a resource location of type `collection`.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Internal collection ID.",
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The name of the collection to look up.",
			},
			"description": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The description of the collection.",
			},
		},
	}
}

func (d *collectionDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *collectionDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state collectionDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	collections, err := d.client.GetCollections()
	if err != nil {
		resp.Diagnostics.AddError("Failed to read collections", err.Error())
		return
	}

	name := state.Name.ValueString()
	var match *client.BowtieCollection
	for _, collection := range collections {
		collection := collection
		if collection.Name == name {
			if match != nil {
				resp.Diagnostics.AddError("Ambiguous collection", fmt.Sprintf("More than one collection is named %q.", name))
				return
			}
			match = &collection
		}
	}
	if match == nil {
		resp.Diagnostics.AddError("Collection not found", fmt.Sprintf("No collection is named %q.", name))
		return
	}

	state.ID = types.StringValue(match.ID)
	if match.Description != "" {
		state.Description = types.StringValue(match.Description)
	} else {
		state.Description = types.StringNull()
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
