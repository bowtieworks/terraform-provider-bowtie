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
	_ datasource.DataSource              = &deviceDataSource{}
	_ datasource.DataSourceWithConfigure = &deviceDataSource{}
)

func NewDeviceDataSource() datasource.DataSource {
	return &deviceDataSource{}
}

type deviceDataSource struct {
	client *client.Client
}

type deviceDataSourceModel struct {
	ID              types.String `tfsdk:"id"`
	Name            types.String `tfsdk:"name"`
	IPV6            types.String `tfsdk:"ipv6"`
	PublicKey       types.String `tfsdk:"public_key"`
	Serial          types.String `tfsdk:"serial"`
	State           types.String `tfsdk:"state"`
	ControllerID    types.String `tfsdk:"controller_id"`
	AssignedToUser  types.String `tfsdk:"assigned_to_user"`
	DeviceType      types.String `tfsdk:"device_type"`
	DeviceOS        types.String `tfsdk:"device_os"`
	LastSeen        types.String `tfsdk:"last_seen"`
	LastSeenVersion types.String `tfsdk:"last_seen_version"`
}

func (d *deviceDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_device"
}

func (d *deviceDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Look up an enrolled device by ID, for example to read its enrollment state or reference it in a policy source.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The device's unique identifier.",
			},
			"name":              schema.StringAttribute{Computed: true, MarkdownDescription: "The device name."},
			"ipv6":              schema.StringAttribute{Computed: true, MarkdownDescription: "The device's assigned IPv6 prefix."},
			"public_key":        schema.StringAttribute{Computed: true, MarkdownDescription: "The device's VPN public key."},
			"serial":            schema.StringAttribute{Computed: true, MarkdownDescription: "The device serial number, used for pre-approval."},
			"state":             schema.StringAttribute{Computed: true, MarkdownDescription: "Enrollment state: `pending`, `accepted`, or `rejected`."},
			"controller_id":     schema.StringAttribute{Computed: true, MarkdownDescription: "The Controller the device last contacted."},
			"assigned_to_user":  schema.StringAttribute{Computed: true, MarkdownDescription: "The user this device is assigned to, if any."},
			"device_type":       schema.StringAttribute{Computed: true, MarkdownDescription: "The device type."},
			"device_os":         schema.StringAttribute{Computed: true, MarkdownDescription: "The device operating system."},
			"last_seen":         schema.StringAttribute{Computed: true, MarkdownDescription: "When the device was last seen."},
			"last_seen_version": schema.StringAttribute{Computed: true, MarkdownDescription: "The client version last reported by the device."},
		},
	}
}

func (d *deviceDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *deviceDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state deviceDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	devices, err := d.client.ListDevices()
	if err != nil {
		resp.Diagnostics.AddError("Failed to read devices", err.Error())
		return
	}

	device, ok := devices[state.ID.ValueString()]
	if !ok {
		resp.Diagnostics.AddError("Device not found", fmt.Sprintf("No device has the ID %q.", state.ID.ValueString()))
		return
	}

	state.Name = types.StringValue(device.Name)
	state.IPV6 = types.StringValue(device.IPV6)
	state.PublicKey = types.StringValue(device.PublicKey)
	state.Serial = types.StringValue(device.Serial)
	state.State = types.StringValue(device.State)
	state.ControllerID = types.StringValue(device.ControllerID)
	state.AssignedToUser = types.StringValue(device.AssignedToUser)
	state.DeviceType = types.StringValue(device.DeviceType)
	state.DeviceOS = types.StringValue(device.DeviceOS)
	state.LastSeen = types.StringValue(device.LastSeen)
	state.LastSeenVersion = types.StringValue(device.LastSeenVersion)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
