package provider

import (
	"context"
	"os"

	"github.com/bowtieworks/terraform-provider-bowtie/internal/bowtie/client"
	"github.com/bowtieworks/terraform-provider-bowtie/internal/bowtie/data_sources"
	"github.com/bowtieworks/terraform-provider-bowtie/internal/bowtie/resources"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type BowtieProvider struct{}

type bowtieProviderModel struct {
	Host               types.String `tfsdk:"host"`
	Username           types.String `tfsdk:"username"`
	Password           types.String `tfsdk:"password"`
	LazyAuthentication types.Bool   `tfsdk:"lazy_authentication"`
	TaggedLocations    types.Bool   `tfsdk:"tagged_locations"`
	Insecure           types.Bool   `tfsdk:"insecure"`
	CABundle           types.String `tfsdk:"ca_bundle"`
}

func New() provider.Provider {
	return &BowtieProvider{}
}

func (b *BowtieProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `The Bowtie provider for Terraform configures your Bowtie installation via native Terraform resources instead of the Controller web interface. Use the provider to declaratively manage API resources such as resource groups, DNS resources, user groups, and more.

Note that you must configure appropriate credentials to authenticate with the Bowtie API before you can use this provider.

For more documentation about installing and configuring Bowtie, refer to the official [Bowtie documentation](https://docs.bowtie.works/).
`,
		Attributes: map[string]schema.Attribute{
			"host": schema.StringAttribute{
				Description: "The Bowtie HTTP Controller endpoint. Honors the `BOWTIE_HOST` environment variable if set. Example: `https://bowtie.example.com`",
				Optional:    true,
			},
			"username": schema.StringAttribute{
				Description: "The login name (username or email) of the Bowtie account Terraform authenticates as. Use a dedicated service account scoped to the least privilege it needs, not a human administrator. Honors the `BOWTIE_USERNAME` environment variable, which is the recommended way to supply it.",
				Optional:    true,
			},
			"password": schema.StringAttribute{
				Description: "The service account's password. Supply it from a secrets manager via the `BOWTIE_PASSWORD` environment variable rather than in version-controlled Terraform configuration. Honors the `BOWTIE_PASSWORD` environment variable if set.",
				Sensitive:   true,
				Optional:    true,
			},
			"lazy_authentication": schema.BoolAttribute{
				Description: "By default, the provider will authenticate to the Bowtie API just in time (or lazily) which permits use cases like creating Controllers in Terraform before using their API endpoints. Set this variable to `false` if you instead want to authenticate at the time the provider is configured - for example, to catch authentication errors up-front before starting an `apply` or `plan`.",
				Optional:    true,
			},
			"tagged_locations": schema.BoolAttribute{
				Description: "Control whether the provider will send policy resource locations using the new tagged type format or legacy format.",
				Optional:    true,
			},
			"insecure": schema.BoolAttribute{
				Description: "Skip TLS certificate verification when connecting to the Controller. Honors the `BOWTIE_INSECURE` environment variable if set. Intended for development controllers with self-signed certificates; do not enable against production.",
				Optional:    true,
			},
			"ca_bundle": schema.StringAttribute{
				Description: "A PEM-encoded CA bundle (inline contents or a path to a file) used to verify the Controller's TLS certificate, for Controllers issued by a private certificate authority. Honors the `BOWTIE_CA_BUNDLE` environment variable if set.",
				Optional:    true,
			},
		},
	}
}

func (b *BowtieProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "bowtie"
}

func (b *BowtieProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var config bowtieProviderModel

	diags := req.Config.Get(ctx, &config)

	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	lazy_auth := true
	tagged_locations := true

	if config.Host.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("host"),
			"Unknown Bowtie API Host",
			"The provider cannot create the Bowtie API Client as the host value is unknown",
		)
	}

	if config.Username.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("username"),
			"Unknown Bowtie API Username",
			"The provider cannot create the Bowtie API Client as the username value is unknown",
		)
	}

	if config.Password.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("password"),
			"Unknown Bowtie API Password",
			"The provider cannot create the Bowtie API Client as the password value is unknown",
		)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	host := os.Getenv("BOWTIE_HOST")
	username := os.Getenv("BOWTIE_USERNAME")
	password := os.Getenv("BOWTIE_PASSWORD")

	if !config.Host.IsNull() {
		host = config.Host.ValueString()
	}

	if !config.Username.IsNull() {
		username = config.Username.ValueString()
	}

	if !config.Password.IsNull() {
		password = config.Password.ValueString()
	}

	if host == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("host"),
			"Missing Bowtie API Host",
			"The provider cannot create the Bowtie API client without a host being set",
		)
	}

	if username == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("username"),
			"Missing Bowtie API Username",
			"The provider cannot create the Bowtie API Client without a username",
		)
	}

	if password == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("password"),
			"Missing Bowtie API Password",
			"The provider cannot create the Bowtie API Client without a password",
		)
	}

	if !config.LazyAuthentication.IsNull() {
		lazy_auth = config.LazyAuthentication.ValueBool()
	}

	if !config.TaggedLocations.IsNull() {
		tagged_locations = config.TaggedLocations.ValueBool()
	}

	insecure := os.Getenv("BOWTIE_INSECURE") == "true" || os.Getenv("BOWTIE_INSECURE") == "1"
	if !config.Insecure.IsNull() {
		insecure = config.Insecure.ValueBool()
	}
	if insecure {
		resp.Diagnostics.AddWarning(
			"TLS verification disabled",
			"The Bowtie provider is configured with insecure = true, which disables TLS certificate verification. This is not safe for production use.",
		)
	}

	ca_bundle := os.Getenv("BOWTIE_CA_BUNDLE")
	if !config.CABundle.IsNull() {
		ca_bundle = config.CABundle.ValueString()
	}

	if resp.Diagnostics.HasError() {
		return
	}

	client, err := client.NewClient(host, username, password, lazy_auth, tagged_locations, insecure, ca_bundle)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to create Bowtie API Client",
			"An unexpected error creating the Bowtie API Client:  "+err.Error(),
		)
		return
	}

	resp.DataSourceData = client
	resp.ResourceData = client
}

func (b *BowtieProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		resources.NewDNSBlockListResource,
		resources.NewDNSResource,
		resources.NewGroupResource,
		resources.NewOrganizationResource,
		resources.NewSiteRangeResource,
		resources.NewSiteResource,
		resources.NewResourceResource,
		resources.NewResourceGroupResource,
		resources.NewGroupMembershipResource,
		resources.NewUserResource,
		resources.NewPolicyResource,
		resources.NewDeviceGroupResource,
		resources.NewCollectionResource,
		resources.NewRouteExclusionResource,
		resources.NewControllerResource,
		resources.NewIPv4RangeResource,
		resources.NewIPv6RangeResource,
		resources.NewOrgConfigResource,
	}
}

func (b *BowtieProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		data_sources.NewUserDataSource,
		data_sources.NewResourceGroupDataSource,
		data_sources.NewGroupDataSource,
		data_sources.NewDeviceGroupDataSource,
		data_sources.NewCollectionDataSource,
		data_sources.NewDeviceDataSource,
	}
}
