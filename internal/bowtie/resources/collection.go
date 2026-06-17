package resources

import (
	"context"
	"fmt"

	"github.com/bowtieworks/terraform-provider-bowtie/internal/bowtie/client"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &collectionResource{}
var _ resource.ResourceWithImportState = &collectionResource{}

type collectionResource struct {
	client *client.Client
}

type collectionResourceModel struct {
	ID          types.String            `tfsdk:"id"`
	Name        types.String            `tfsdk:"name"`
	Description types.String            `tfsdk:"description"`
	Members     []collectionMemberModel `tfsdk:"members"`
}

type collectionMemberModel struct {
	Name     types.String            `tfsdk:"name"`
	Comment  types.String            `tfsdk:"comment"`
	Expires  types.String            `tfsdk:"expires"`
	Location collectionLocationModel `tfsdk:"location"`
}

type collectionLocationModel struct {
	IP         types.String `tfsdk:"ip"`
	CIDR       types.String `tfsdk:"cidr"`
	DNS        types.String `tfsdk:"dns"`
	Collection types.String `tfsdk:"collection"`
}

func NewCollectionResource() resource.Resource {
	return &collectionResource{}
}

func (c *collectionResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_collection"
}

func (c *collectionResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "A *collection* is a named, reusable set of network locations - IP addresses, CIDR ranges, DNS names, or nested collections. Collections can be targeted by a resource's `location` (type `collection`) and are used throughout the policy engine and web filtering configuration.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Internal collection ID.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The human readable name of the collection.",
				Required:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "An optional description of the collection.",
				Optional:            true,
			},
			"members": schema.SetNestedAttribute{
				MarkdownDescription: "The locations contained in this collection.",
				Optional:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							MarkdownDescription: "A human readable name for this member.",
							Required:            true,
						},
						"comment": schema.StringAttribute{
							MarkdownDescription: "An optional comment describing this member.",
							Optional:            true,
						},
						"expires": schema.StringAttribute{
							MarkdownDescription: "An optional RFC 3339 timestamp after which the Controller automatically removes this member.",
							Optional:            true,
						},
						"location": schema.SingleNestedAttribute{
							MarkdownDescription: "The location this member matches. Set exactly one of `ip`, `cidr`, `dns`, or `collection`.",
							Required:            true,
							Attributes: map[string]schema.Attribute{
								"ip": schema.StringAttribute{
									MarkdownDescription: "A single IP address.",
									Optional:            true,
									Validators: []validator.String{
										stringvalidator.ExactlyOneOf(path.Expressions{
											path.MatchRelative().AtParent().AtName("cidr"),
											path.MatchRelative().AtParent().AtName("dns"),
											path.MatchRelative().AtParent().AtName("collection"),
										}...),
									},
								},
								"cidr": schema.StringAttribute{
									MarkdownDescription: "A CIDR range.",
									Optional:            true,
									Validators: []validator.String{
										stringvalidator.ExactlyOneOf(path.Expressions{
											path.MatchRelative().AtParent().AtName("ip"),
											path.MatchRelative().AtParent().AtName("dns"),
											path.MatchRelative().AtParent().AtName("collection"),
										}...),
									},
								},
								"dns": schema.StringAttribute{
									MarkdownDescription: "A DNS name.",
									Optional:            true,
									Validators: []validator.String{
										stringvalidator.ExactlyOneOf(path.Expressions{
											path.MatchRelative().AtParent().AtName("ip"),
											path.MatchRelative().AtParent().AtName("cidr"),
											path.MatchRelative().AtParent().AtName("collection"),
										}...),
									},
								},
								"collection": schema.StringAttribute{
									MarkdownDescription: "The ID of another collection to nest inside this one.",
									Optional:            true,
									Validators: []validator.String{
										stringvalidator.ExactlyOneOf(path.Expressions{
											path.MatchRelative().AtParent().AtName("ip"),
											path.MatchRelative().AtParent().AtName("cidr"),
											path.MatchRelative().AtParent().AtName("dns"),
										}...),
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func (c *collectionResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	cl, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configuration Type",
			fmt.Sprintf("Expected *client.Client, got: %T, please report this to the provider.", req.ProviderData),
		)
		return
	}

	c.client = cl
}

func (c *collectionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan collectionResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if plan.ID.ValueString() == "" {
		plan.ID = types.StringValue(uuid.NewString())
	}

	members, diags := membersToAPI(plan.Members)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := c.client.UpsertCollection(plan.ID.ValueString(), plan.Name.ValueString(), descriptionString(plan.Description)); err != nil {
		resp.Diagnostics.AddError("Failed to create collection", "Unexpected error creating collection: "+err.Error())
		return
	}

	if err := c.client.AddCollectionMembers(plan.ID.ValueString(), members); err != nil {
		resp.Diagnostics.AddError("Failed to add collection members", "Unexpected error adding collection members: "+err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (c *collectionResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state collectionResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	collections, err := c.client.GetCollections()
	if err != nil {
		resp.Diagnostics.AddError("Failed to read collection", "Unexpected error reading collection "+state.ID.ValueString()+": "+err.Error())
		return
	}

	collection, present := collections[state.ID.ValueString()]
	if !present {
		resp.Diagnostics.AddAttributeWarning(
			path.Root("id"),
			"Collection not found, removing from state",
			state.ID.ValueString(),
		)
		resp.State.RemoveResource(ctx)
		return
	}

	state.Name = types.StringValue(collection.Name)
	if collection.Description != "" {
		state.Description = types.StringValue(collection.Description)
	} else {
		state.Description = types.StringNull()
	}

	members, diags := membersFromAPI(collection.Members)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	state.Members = members

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (c *collectionResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan collectionResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	desired, diags := membersToAPI(plan.Members)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := c.client.UpsertCollection(plan.ID.ValueString(), plan.Name.ValueString(), descriptionString(plan.Description)); err != nil {
		resp.Diagnostics.AddError("Failed to update collection", "Unexpected error updating collection "+plan.ID.ValueString()+": "+err.Error())
		return
	}

	// Members can only be mutated through add/remove, so replace the full set:
	// drop everything currently stored, then add the desired members back.
	collections, err := c.client.GetCollections()
	if err != nil {
		resp.Diagnostics.AddError("Failed to read collection members", "Unexpected error reading collection "+plan.ID.ValueString()+": "+err.Error())
		return
	}
	if current, present := collections[plan.ID.ValueString()]; present {
		existingIDs := collectionMemberIDs(current.Members)
		if err := c.client.RemoveCollectionMembers(plan.ID.ValueString(), existingIDs); err != nil {
			resp.Diagnostics.AddError("Failed to remove collection members", "Unexpected error removing collection members: "+err.Error())
			return
		}
	}

	if err := c.client.AddCollectionMembers(plan.ID.ValueString(), desired); err != nil {
		resp.Diagnostics.AddError("Failed to add collection members", "Unexpected error adding collection members: "+err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (c *collectionResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state collectionResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := c.client.DeleteCollection(state.ID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Failed to delete collection", "Unexpected error deleting collection "+state.ID.ValueString()+": "+err.Error())
	}
}

func (c *collectionResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func membersToAPI(members []collectionMemberModel) ([]client.BowtieCollectionMember, diag.Diagnostics) {
	var diags diag.Diagnostics
	out := make([]client.BowtieCollectionMember, 0, len(members))
	for _, member := range members {
		location, err := member.Location.toAPI()
		if err != nil {
			diags.AddAttributeError(path.Root("members"), "Invalid collection member location", err.Error())
			return nil, diags
		}

		apiMember := client.BowtieCollectionMember{
			ID:       uuid.NewString(),
			Name:     member.Name.ValueString(),
			Comment:  member.Comment.ValueString(),
			Location: location,
		}
		if !member.Expires.IsNull() && member.Expires.ValueString() != "" {
			expires := member.Expires.ValueString()
			apiMember.Expires = &expires
		}
		out = append(out, apiMember)
	}
	return out, diags
}

func collectionMemberIDs(members map[string]client.BowtieCollectionMember) []string {
	ids := make([]string, 0, len(members))
	for key := range members {
		ids = append(ids, key)
	}
	return ids
}

func membersFromAPI(members map[string]client.BowtieCollectionMember) ([]collectionMemberModel, diag.Diagnostics) {
	var diags diag.Diagnostics
	// Return null (not an empty set) when there are no members so the attribute
	// matches a configuration that simply omits members.
	if len(members) == 0 {
		return nil, diags
	}
	out := make([]collectionMemberModel, 0, len(members))
	for _, member := range members {
		location, err := locationFromAPI(member.Location)
		if err != nil {
			diags.AddAttributeError(path.Root("members"), "Unsupported collection member location", err.Error())
			return nil, diags
		}

		model := collectionMemberModel{
			Name:     types.StringValue(member.Name),
			Comment:  types.StringNull(),
			Expires:  types.StringNull(),
			Location: location,
		}
		if member.Comment != "" {
			model.Comment = types.StringValue(member.Comment)
		}
		if member.Expires != nil && *member.Expires != "" {
			model.Expires = types.StringValue(*member.Expires)
		}
		out = append(out, model)
	}
	return out, diags
}

func (l collectionLocationModel) toAPI() (client.BowtieCollectionLocation, error) {
	set := 0
	for _, value := range []types.String{l.IP, l.CIDR, l.DNS, l.Collection} {
		if isSet(value) {
			set++
		}
	}
	if set != 1 {
		return client.BowtieCollectionLocation{}, fmt.Errorf("a member location must set exactly one of ip, cidr, dns, or collection; %d were configured", set)
	}

	switch {
	case isSet(l.IP):
		return client.BowtieCollectionLocation{Type: "ip", Value: l.IP.ValueString()}, nil
	case isSet(l.CIDR):
		return client.BowtieCollectionLocation{Type: "cidr", Value: l.CIDR.ValueString()}, nil
	case isSet(l.DNS):
		return client.BowtieCollectionLocation{Type: "dns", Value: l.DNS.ValueString()}, nil
	case isSet(l.Collection):
		return client.BowtieCollectionLocation{Type: "collection", Value: l.Collection.ValueString()}, nil
	default:
		return client.BowtieCollectionLocation{}, fmt.Errorf("a member location must set one of ip, cidr, dns, or collection")
	}
}

func locationFromAPI(location client.BowtieCollectionLocation) (collectionLocationModel, error) {
	model := collectionLocationModel{
		IP:         types.StringNull(),
		CIDR:       types.StringNull(),
		DNS:        types.StringNull(),
		Collection: types.StringNull(),
	}
	switch location.Type {
	case "ip":
		model.IP = types.StringValue(location.Value)
	case "cidr":
		model.CIDR = types.StringValue(location.Value)
	case "dns":
		model.DNS = types.StringValue(location.Value)
	case "collection":
		model.Collection = types.StringValue(location.Value)
	default:
		return model, fmt.Errorf("unknown location type %q", location.Type)
	}
	return model, nil
}

// descriptionString returns the value of an optional string attribute, or the
// empty string when it is unset, for endpoints that take a bare string.
func descriptionString(value types.String) string {
	if value.IsNull() || value.IsUnknown() {
		return ""
	}
	return value.ValueString()
}
