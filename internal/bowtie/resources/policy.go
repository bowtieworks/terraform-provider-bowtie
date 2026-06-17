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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &policyResource{}
var _ resource.ResourceWithImportState = &policyResource{}

type policyResource struct {
	client *client.Client
}

type policyResourceModel struct {
	ID     types.String      `tfsdk:"id"`
	Source policySourceModel `tfsdk:"source"`
	Dest   types.String      `tfsdk:"dest"`
	Action types.String      `tfsdk:"action"`
	Status types.String      `tfsdk:"status"`
	Order  types.Int64       `tfsdk:"order"`
}

// maxSourceNestingDepth bounds how deeply and/or/nor groups may nest. The
// Terraform schema cannot be infinitely recursive, so the source predicate is
// modeled as a fixed chain: the top-level source plus this many nested operand
// levels. Predicates deeper than this are reported as unsupported on read.
// Three levels comfortably covers policies authored in the Controller, which
// wraps even simple rules a level or two deep.
const maxSourceNestingDepth = 3

// policySourceModel describes who a policy applies to. Exactly one of the leaf
// matchers or one of the and/or/nor lists must be set. The and/or/nor lists may
// themselves nest further groups, up to maxSourceNestingDepth levels deep.
type policySourceModel struct {
	ID                types.String          `tfsdk:"id"`
	Always            types.Bool            `tfsdk:"always"`
	AuthenticatedUser types.Bool            `tfsdk:"authenticated_user"`
	User              types.String          `tfsdk:"user"`
	Device            types.String          `tfsdk:"device"`
	UserGroup         types.String          `tfsdk:"user_group"`
	DeviceGroup       types.String          `tfsdk:"device_group"`
	And               []policyOperandModel1 `tfsdk:"and"`
	Or                []policyOperandModel1 `tfsdk:"or"`
	Nor               []policyOperandModel1 `tfsdk:"nor"`
}

// policyOperandModel1, policyOperandModel2, and policyOperandModel3 are the
// successive nesting levels below the top-level source. Each level is identical
// except that its and/or/nor groups hold the next level down; the deepest level
// (policyOperandModel3) holds only scalar matchers. Distinct types are required
// because the framework reflects each nested object shape onto a concrete Go
// struct, and the schema — which cannot be infinitely recursive — terminates the
// chain at policyOperandModel3.
type policyOperandModel1 struct {
	Always            types.Bool            `tfsdk:"always"`
	AuthenticatedUser types.Bool            `tfsdk:"authenticated_user"`
	User              types.String          `tfsdk:"user"`
	Device            types.String          `tfsdk:"device"`
	UserGroup         types.String          `tfsdk:"user_group"`
	DeviceGroup       types.String          `tfsdk:"device_group"`
	And               []policyOperandModel2 `tfsdk:"and"`
	Or                []policyOperandModel2 `tfsdk:"or"`
	Nor               []policyOperandModel2 `tfsdk:"nor"`
}

type policyOperandModel2 struct {
	Always            types.Bool            `tfsdk:"always"`
	AuthenticatedUser types.Bool            `tfsdk:"authenticated_user"`
	User              types.String          `tfsdk:"user"`
	Device            types.String          `tfsdk:"device"`
	UserGroup         types.String          `tfsdk:"user_group"`
	DeviceGroup       types.String          `tfsdk:"device_group"`
	And               []policyOperandModel3 `tfsdk:"and"`
	Or                []policyOperandModel3 `tfsdk:"or"`
	Nor               []policyOperandModel3 `tfsdk:"nor"`
}

type policyOperandModel3 struct {
	Always            types.Bool   `tfsdk:"always"`
	AuthenticatedUser types.Bool   `tfsdk:"authenticated_user"`
	User              types.String `tfsdk:"user"`
	Device            types.String `tfsdk:"device"`
	UserGroup         types.String `tfsdk:"user_group"`
	DeviceGroup       types.String `tfsdk:"device_group"`
}

// matcherFields bundles the scalar matchers shared by every nesting level so the
// conversion logic can be written once and reused across operand types.
type matcherFields struct {
	always            types.Bool
	authenticatedUser types.Bool
	user              types.String
	device            types.String
	userGroup         types.String
	deviceGroup       types.String
}

func (s policySourceModel) matchers() matcherFields {
	return matcherFields{s.Always, s.AuthenticatedUser, s.User, s.Device, s.UserGroup, s.DeviceGroup}
}

func (o policyOperandModel1) matchers() matcherFields {
	return matcherFields{o.Always, o.AuthenticatedUser, o.User, o.Device, o.UserGroup, o.DeviceGroup}
}

func (o policyOperandModel2) matchers() matcherFields {
	return matcherFields{o.Always, o.AuthenticatedUser, o.User, o.Device, o.UserGroup, o.DeviceGroup}
}

func (o policyOperandModel3) matchers() matcherFields {
	return matcherFields{o.Always, o.AuthenticatedUser, o.User, o.Device, o.UserGroup, o.DeviceGroup}
}

func NewPolicyResource() resource.Resource {
	return &policyResource{}
}

func (p *policyResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_policy"
}

// leafAttributes are reused for the top-level source and for each operand of a
// boolean group so that the predicate vocabulary stays consistent.
func leafAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"always": schema.BoolAttribute{
			MarkdownDescription: "Match every device, authenticated or not.",
			Optional:            true,
		},
		"authenticated_user": schema.BoolAttribute{
			MarkdownDescription: "Match any device attached to an authenticated user.",
			Optional:            true,
		},
		"user": schema.StringAttribute{
			MarkdownDescription: "Match a single user by ID.",
			Optional:            true,
		},
		"device": schema.StringAttribute{
			MarkdownDescription: "Match a single device by ID.",
			Optional:            true,
		},
		"user_group": schema.StringAttribute{
			MarkdownDescription: "Match any user in the user group with this ID.",
			Optional:            true,
		},
		"device_group": schema.StringAttribute{
			MarkdownDescription: "Match any device in the device group with this ID.",
			Optional:            true,
		},
	}
}

// operandAttributes builds the schema for a source predicate object. It always
// includes the scalar matchers and, while depth remains, the and/or/nor groups
// whose operands are themselves source predicates one level shallower. The
// recursion terminates at depth 0 (scalar matchers only), which keeps the schema
// finite while still allowing nested boolean groups.
func operandAttributes(depth int) map[string]schema.Attribute {
	attrs := leafAttributes()
	if depth > 0 {
		group := func(conjunction string) schema.Attribute {
			return schema.ListNestedAttribute{
				MarkdownDescription: fmt.Sprintf("Match if %s of the listed operands match. Each operand is a single matcher or a further and/or/nor group.", conjunction),
				Optional:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: operandAttributes(depth - 1),
				},
			}
		}
		attrs["and"] = group("all")
		attrs["or"] = group("at least one")
		attrs["nor"] = group("none")
	}
	return attrs
}

func (p *policyResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	sourceAttributes := operandAttributes(maxSourceNestingDepth)
	sourceAttributes["id"] = schema.StringAttribute{
		MarkdownDescription: "Internal identifier of the source predicate. Assigned by the provider.",
		Computed:            true,
		PlanModifiers: []planmodifier.String{
			stringplanmodifier.UseStateForUnknown(),
		},
	}

	resp.Schema = schema.Schema{
		MarkdownDescription: `A *policy* is a single rule in the Bowtie policy engine. It grants or denies a *source* - described by a predicate over users, devices, and groups - access to a destination *resource group*.

Policies are evaluated in order, so set ` + "`order`" + ` explicitly when the relative precedence of two rules matters.`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Internal policy ID.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"source": schema.SingleNestedAttribute{
				MarkdownDescription: "The set of devices this policy applies to. Set exactly one of the leaf matchers (`always`, `authenticated_user`, `user`, `device`, `user_group`, `device_group`) or exactly one of the logic groups (`and`, `or`, `nor`).",
				Required:            true,
				Attributes:          sourceAttributes,
			},
			"dest": schema.StringAttribute{
				MarkdownDescription: "The ID of the resource group this policy controls access to.",
				Required:            true,
			},
			"action": schema.StringAttribute{
				MarkdownDescription: "The action to take on matching traffic: `Accept`, `Reject` (deny with feedback), or `Drop` (deny silently).",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.OneOf("Accept", "Reject", "Drop"),
				},
			},
			"status": schema.StringAttribute{
				MarkdownDescription: "Whether the policy is `Enabled` or `Disabled`. Defaults to `Enabled`.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("Enabled"),
				Validators: []validator.String{
					stringvalidator.OneOf("Enabled", "Disabled"),
				},
			},
			"order": schema.Int64Attribute{
				MarkdownDescription: "Evaluation order of the policy. When omitted, the Controller appends the policy to the end of the list.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (p *policyResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configuration Type",
			fmt.Sprintf("Expected *client.Client, got: %T, please report this to the provider.", req.ProviderData),
		)
		return
	}

	p.client = c
}

func (p *policyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan policyResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	p.upsert(ctx, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (p *policyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	// Only the id is needed to re-fetch the policy. Decoding the whole model
	// would fail on import, where prior state has a null source object that
	// cannot be assigned to the non-nullable policySourceModel field.
	var id types.String
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("id"), &id)...)
	if resp.Diagnostics.HasError() {
		return
	}

	policies, err := p.client.GetPoliciesAndResources()
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to read policy",
			"Unexpected error reading policy "+id.ValueString()+": "+err.Error(),
		)
		return
	}

	policy, present := policies.Policies[id.ValueString()]
	if !present {
		resp.Diagnostics.AddAttributeWarning(
			path.Root("id"),
			"Policy not found, removing from state",
			id.ValueString(),
		)
		resp.State.RemoveResource(ctx)
		return
	}

	model, err := policyToModel(policy)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unsupported policy shape",
			"Policy "+id.ValueString()+" cannot be represented by this provider: "+err.Error(),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}

func (p *policyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan policyResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	p.upsert(ctx, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (p *policyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state policyResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := p.client.DeletePolicy(state.ID.ValueString()); err != nil {
		resp.Diagnostics.AddError(
			"Failed to delete policy",
			"Unexpected error deleting policy "+state.ID.ValueString()+": "+err.Error(),
		)
	}
}

func (p *policyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// upsert translates the plan into an API call and writes the server-assigned
// fields (id, source id, order, status) back into the plan in place.
func (p *policyResource) upsert(ctx context.Context, plan *policyResourceModel, diags *diag.Diagnostics) {
	predicate, err := plan.Source.toPredicate()
	if err != nil {
		diags.AddAttributeError(path.Root("source"), "Invalid policy source", err.Error())
		return
	}

	if plan.ID.ValueString() == "" {
		plan.ID = types.StringValue(uuid.NewString())
	}
	if plan.Source.ID.ValueString() == "" {
		plan.Source.ID = types.StringValue(uuid.NewString())
	}

	policy := client.BowtiePolicy{
		ID: plan.ID.ValueString(),
		Source: client.BowtiePolicySource{
			ID:        plan.Source.ID.ValueString(),
			Predicate: predicate,
		},
		Dest:   plan.Dest.ValueString(),
		Action: plan.Action.ValueString(),
		Status: plan.Status.ValueString(),
	}
	if !plan.Order.IsUnknown() && !plan.Order.IsNull() {
		order := plan.Order.ValueInt64()
		policy.Order = &order
	}

	saved, err := p.client.UpsertPolicy(policy)
	if err != nil {
		diags.AddError("Failed to write policy", "Unexpected error writing policy "+plan.ID.ValueString()+": "+err.Error())
		return
	}

	if saved.Order != nil {
		plan.Order = types.Int64Value(*saved.Order)
	} else if plan.Order.IsUnknown() {
		// The Controller always assigns an order, but guard against a response
		// that omits it: leaving order unknown would make Terraform reject the
		// apply with "inconsistent result". A null is a known value the schema
		// accepts, and the next refresh reconciles it from the server.
		plan.Order = types.Int64Null()
	}
	if saved.Status != "" {
		plan.Status = types.StringValue(saved.Status)
	}
}

// toPredicate converts the scalar matchers into an API predicate, returning the
// number of matchers that were set so callers can enforce that a source selects
// exactly one thing.
func (m matcherFields) toPredicate() (client.BowtiePredicate, int) {
	var predicate client.BowtiePredicate
	set := 0

	if m.always.ValueBool() {
		predicate.Always = true
		set++
	}
	if m.authenticatedUser.ValueBool() {
		predicate.AuthenticatedUser = true
		set++
	}
	if v := m.user.ValueString(); v != "" {
		predicate.User = v
		set++
	}
	if v := m.device.ValueString(); v != "" {
		predicate.Device = v
		set++
	}
	if v := m.userGroup.ValueString(); v != "" {
		predicate.InUserGroup = v
		set++
	}
	if v := m.deviceGroup.ValueString(); v != "" {
		predicate.InDeviceGroup = v
		set++
	}

	return predicate, set
}

// assembleSource combines the scalar matchers with already-converted and/or/nor
// operand groups, enforcing that exactly one selector is present.
func assembleSource(m matcherFields, and, or, nor []client.BowtiePolicySource) (client.BowtiePredicate, error) {
	predicate, set := m.toPredicate()
	if len(and) > 0 {
		predicate.And = and
		set++
	}
	if len(or) > 0 {
		predicate.Or = or
		set++
	}
	if len(nor) > 0 {
		predicate.Nor = nor
		set++
	}

	if set == 0 {
		return predicate, fmt.Errorf("a source must set one matcher (always, authenticated_user, user, device, user_group, device_group) or one of and/or/nor")
	}
	if set > 1 {
		return predicate, fmt.Errorf("a source must set exactly one matcher, but %d were configured", set)
	}

	return predicate, nil
}

// operandsToSources converts a slice of operands at any nesting level into API
// sources, tagging each with a fresh ID.
func operandsToSources[T interface {
	toPredicate() (client.BowtiePredicate, error)
}](items []T) ([]client.BowtiePolicySource, error) {
	if len(items) == 0 {
		return nil, nil
	}
	sources := make([]client.BowtiePolicySource, 0, len(items))
	for _, item := range items {
		predicate, err := item.toPredicate()
		if err != nil {
			return nil, err
		}
		sources = append(sources, client.BowtiePolicySource{
			ID:        uuid.NewString(),
			Predicate: predicate,
		})
	}
	return sources, nil
}

func (s policySourceModel) toPredicate() (client.BowtiePredicate, error) {
	and, err := operandsToSources(s.And)
	if err != nil {
		return client.BowtiePredicate{}, err
	}
	or, err := operandsToSources(s.Or)
	if err != nil {
		return client.BowtiePredicate{}, err
	}
	nor, err := operandsToSources(s.Nor)
	if err != nil {
		return client.BowtiePredicate{}, err
	}
	return assembleSource(s.matchers(), and, or, nor)
}

func (o policyOperandModel1) toPredicate() (client.BowtiePredicate, error) {
	and, err := operandsToSources(o.And)
	if err != nil {
		return client.BowtiePredicate{}, err
	}
	or, err := operandsToSources(o.Or)
	if err != nil {
		return client.BowtiePredicate{}, err
	}
	nor, err := operandsToSources(o.Nor)
	if err != nil {
		return client.BowtiePredicate{}, err
	}
	return assembleSource(o.matchers(), and, or, nor)
}

func (o policyOperandModel2) toPredicate() (client.BowtiePredicate, error) {
	and, err := operandsToSources(o.And)
	if err != nil {
		return client.BowtiePredicate{}, err
	}
	or, err := operandsToSources(o.Or)
	if err != nil {
		return client.BowtiePredicate{}, err
	}
	nor, err := operandsToSources(o.Nor)
	if err != nil {
		return client.BowtiePredicate{}, err
	}
	return assembleSource(o.matchers(), and, or, nor)
}

func (o policyOperandModel3) toPredicate() (client.BowtiePredicate, error) {
	return assembleSource(o.matchers(), nil, nil, nil)
}

// policyToModel maps a policy fetched from the API back into the resource model.
func policyToModel(policy client.BowtiePolicy) (policyResourceModel, error) {
	source, err := predicateToSource(policy.Source)
	if err != nil {
		return policyResourceModel{}, err
	}

	status := types.StringValue(policy.Status)
	if policy.Status == "" {
		status = types.StringValue("Enabled")
	}

	model := policyResourceModel{
		ID:     types.StringValue(policy.ID),
		Source: source,
		Dest:   types.StringValue(policy.Dest),
		Action: types.StringValue(policy.Action),
		Status: status,
		Order:  types.Int64Null(),
	}
	if policy.Order != nil {
		model.Order = types.Int64Value(*policy.Order)
	}

	return model, nil
}

// scalarMatchers maps a predicate's scalar variants onto matcher fields, all
// null unless one matched. It reports whether a scalar matcher was found so
// callers can distinguish a leaf from a boolean group.
func scalarMatchers(predicate client.BowtiePredicate) (matcherFields, bool) {
	m := matcherFields{
		always:            types.BoolNull(),
		authenticatedUser: types.BoolNull(),
		user:              types.StringNull(),
		device:            types.StringNull(),
		userGroup:         types.StringNull(),
		deviceGroup:       types.StringNull(),
	}

	switch {
	case predicate.Always:
		m.always = types.BoolValue(true)
	case predicate.AuthenticatedUser:
		m.authenticatedUser = types.BoolValue(true)
	case predicate.User != "":
		m.user = types.StringValue(predicate.User)
	case predicate.Device != "":
		m.device = types.StringValue(predicate.Device)
	case predicate.InUserGroup != "":
		m.userGroup = types.StringValue(predicate.InUserGroup)
	case predicate.InDeviceGroup != "":
		m.deviceGroup = types.StringValue(predicate.InDeviceGroup)
	default:
		return m, false
	}

	return m, true
}

// operandsFromSources converts API sources into operands of any nesting level
// using the supplied per-level constructor.
func operandsFromSources[T any](sources []client.BowtiePolicySource, conv func(client.BowtiePolicySource) (T, error)) ([]T, error) {
	operands := make([]T, 0, len(sources))
	for _, source := range sources {
		operand, err := conv(source)
		if err != nil {
			return nil, err
		}
		operands = append(operands, operand)
	}
	return operands, nil
}

// predicateToSource maps a policy's top-level source predicate into the model.
func predicateToSource(source client.BowtiePolicySource) (policySourceModel, error) {
	m, scalar := scalarMatchers(source.Predicate)
	model := policySourceModel{
		ID:                types.StringValue(source.ID),
		Always:            m.always,
		AuthenticatedUser: m.authenticatedUser,
		User:              m.user,
		Device:            m.device,
		UserGroup:         m.userGroup,
		DeviceGroup:       m.deviceGroup,
	}

	switch {
	case source.Predicate.And != nil:
		operands, err := operandsFromSources(source.Predicate.And, operand1FromSource)
		if err != nil {
			return model, err
		}
		model.And = operands
	case source.Predicate.Or != nil:
		operands, err := operandsFromSources(source.Predicate.Or, operand1FromSource)
		if err != nil {
			return model, err
		}
		model.Or = operands
	case source.Predicate.Nor != nil:
		operands, err := operandsFromSources(source.Predicate.Nor, operand1FromSource)
		if err != nil {
			return model, err
		}
		model.Nor = operands
	default:
		if !scalar {
			return model, fmt.Errorf("policy source has no recognized predicate")
		}
	}

	return model, nil
}

func operand1FromSource(source client.BowtiePolicySource) (policyOperandModel1, error) {
	m, scalar := scalarMatchers(source.Predicate)
	model := policyOperandModel1{
		Always:            m.always,
		AuthenticatedUser: m.authenticatedUser,
		User:              m.user,
		Device:            m.device,
		UserGroup:         m.userGroup,
		DeviceGroup:       m.deviceGroup,
	}

	switch {
	case source.Predicate.And != nil:
		operands, err := operandsFromSources(source.Predicate.And, operand2FromSource)
		if err != nil {
			return model, err
		}
		model.And = operands
	case source.Predicate.Or != nil:
		operands, err := operandsFromSources(source.Predicate.Or, operand2FromSource)
		if err != nil {
			return model, err
		}
		model.Or = operands
	case source.Predicate.Nor != nil:
		operands, err := operandsFromSources(source.Predicate.Nor, operand2FromSource)
		if err != nil {
			return model, err
		}
		model.Nor = operands
	default:
		if !scalar {
			return model, fmt.Errorf("policy source has no recognized predicate")
		}
	}

	return model, nil
}

func operand2FromSource(source client.BowtiePolicySource) (policyOperandModel2, error) {
	m, scalar := scalarMatchers(source.Predicate)
	model := policyOperandModel2{
		Always:            m.always,
		AuthenticatedUser: m.authenticatedUser,
		User:              m.user,
		Device:            m.device,
		UserGroup:         m.userGroup,
		DeviceGroup:       m.deviceGroup,
	}

	switch {
	case source.Predicate.And != nil:
		operands, err := operandsFromSources(source.Predicate.And, operand3FromSource)
		if err != nil {
			return model, err
		}
		model.And = operands
	case source.Predicate.Or != nil:
		operands, err := operandsFromSources(source.Predicate.Or, operand3FromSource)
		if err != nil {
			return model, err
		}
		model.Or = operands
	case source.Predicate.Nor != nil:
		operands, err := operandsFromSources(source.Predicate.Nor, operand3FromSource)
		if err != nil {
			return model, err
		}
		model.Nor = operands
	default:
		if !scalar {
			return model, fmt.Errorf("policy source has no recognized predicate")
		}
	}

	return model, nil
}

// operand3FromSource is the deepest level: it holds only scalar matchers, so a
// boolean group here means the predicate is nested deeper than the schema can
// represent.
func operand3FromSource(source client.BowtiePolicySource) (policyOperandModel3, error) {
	m, scalar := scalarMatchers(source.Predicate)
	if !scalar {
		if source.Predicate.And != nil || source.Predicate.Or != nil || source.Predicate.Nor != nil {
			return policyOperandModel3{}, fmt.Errorf("policy source nesting exceeds the maximum supported depth of %d; manage this policy outside Terraform", maxSourceNestingDepth)
		}
		return policyOperandModel3{}, fmt.Errorf("policy source has no recognized predicate")
	}

	return policyOperandModel3{
		Always:            m.always,
		AuthenticatedUser: m.authenticatedUser,
		User:              m.user,
		Device:            m.device,
		UserGroup:         m.userGroup,
		DeviceGroup:       m.deviceGroup,
	}, nil
}
