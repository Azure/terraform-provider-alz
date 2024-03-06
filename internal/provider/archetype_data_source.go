// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"

	"github.com/Azure/alzlib"
	"github.com/Azure/alzlib/to"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/authorization/armauthorization"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armpolicy"
	"github.com/Azure/terraform-provider-alz/internal/alztypes"
	"github.com/Azure/terraform-provider-alz/internal/alzvalidators"
	"github.com/Azure/terraform-provider-alz/internal/typehelper"
	mapset "github.com/deckarep/golang-set/v2"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &ArchetypeDataSource{}

func NewArchetypeDataSource() datasource.DataSource {
	return &ArchetypeDataSource{}
}

// ArchetypeDataSource defines the data source implementation.
type ArchetypeDataSource struct {
	alz *alzProviderData
}

// mapTypes is used for the generic functions that operate on certain map types.
type mapTypes interface {
	armpolicy.Assignment |
		armpolicy.Definition |
		armpolicy.SetDefinition |
		armauthorization.RoleAssignment |
		armauthorization.RoleDefinition
}

// checkExistsInAlzLib is a helper struct to check if an item exists in the AlzLib.
type checkExistsInAlzLib struct {
	set mapset.Set[string]
	f   func(string) bool
}

// ArchetypeDataSourceModel describes the data source data model.
type ArchetypeDataSourceModel struct {
	AlzPolicyAssignments      types.Map                              `tfsdk:"alz_policy_assignments"`     // map of string, computed
	AlzPolicyDefinitions      types.Map                              `tfsdk:"alz_policy_definitions"`     // map of string, computed
	AlzPolicySetDefinitions   types.Map                              `tfsdk:"alz_policy_set_definitions"` // map of string, computed
	AlzPolicyRoleAssignments  map[string]AlzPolicyRoleAssignmentType `tfsdk:"alz_policy_role_assignments"`
	AlzRoleDefinitions        types.Map                              `tfsdk:"alz_role_definitions"` // map of string, computed
	BaseArchetype             types.String                           `tfsdk:"base_archetype"`
	Defaults                  ArchetypeDataSourceModelDefaults       `tfsdk:"defaults"`
	DisplayName               types.String                           `tfsdk:"display_name"`
	Id                        types.String                           `tfsdk:"id"`
	ParentId                  types.String                           `tfsdk:"parent_id"`
	PolicyAssignmentsToModify map[string]PolicyAssignmentType        `tfsdk:"policy_assignments_to_modify"`
}

// AlzPolicyRoleAssignmentType is a representation of the policy assignments
// that must be created when assigning a given policy.
type AlzPolicyRoleAssignmentType struct {
	RoleDefinitionId types.String `tfsdk:"role_definition_id"`
	Scope            types.String `tfsdk:"scope"`
	AssignmentName   types.String `tfsdk:"assignment_name"`
}

// ArchetypeDataSourceModelDefaults describes the defaults used in the alz data processing.
type ArchetypeDataSourceModelDefaults struct {
	DefaultLocation               types.String `tfsdk:"location"`
	DefaultLaWorkspaceId          types.String `tfsdk:"log_analytics_workspace_id"`
	PrivateDnsZoneResourceGroupId types.String `tfsdk:"private_dns_zone_resource_group_id"`
}

// PolicyAssignmentType describes the policy assignment data model.
type PolicyAssignmentType struct {
	EnforcementMode      types.String                           `tfsdk:"enforcement_mode"`
	Identity             types.String                           `tfsdk:"identity"`
	IdentityIds          types.Set                              `tfsdk:"identity_ids"`           // set of string
	NonComplianceMessage []PolicyAssignmentNonComplianceMessage `tfsdk:"non_compliance_message"` // set of PolicyAssignmentNonComplianceMessage
	Parameters           alztypes.PolicyParameterValue          `tfsdk:"parameters"`
	Overrides            []PolicyAssignmentOverrideType         `tfsdk:"overrides"`
	ResourceSelectors    []ResourceSelectorType                 `tfsdk:"resource_selectors"`
}

// PolicyAssignmentNonComplianceMessage describes non-compliance message in a policy assignment.
type PolicyAssignmentNonComplianceMessage struct {
	Message                     types.String `tfsdk:"message"`
	PolicyDefinitionReferenceId types.String `tfsdk:"policy_definition_reference_id"`
}

type ResourceSelectorType struct {
	Name      types.String                   `tfsdk:"name"`
	Selectors []ResourceSelectorSelectorType `tfsdk:"selectors"`
}

type ResourceSelectorSelectorType struct {
	Kind  types.String `tfsdk:"kind"`
	In    types.Set    `tfsdk:"in"`     // set of string
	NotIn types.Set    `tfsdk:"not_in"` // set of string
}

type PolicyAssignmentOverrideType struct {
	Kind      types.String                           `tfsdk:"kind"`
	Value     types.String                           `tfsdk:"value"`
	Selectors []PolicyAssignmentOverrideSelectorType `tfsdk:"selectors"`
}

type PolicyAssignmentOverrideSelectorType struct {
	Kind  types.String `tfsdk:"kind"`
	In    types.Set    `tfsdk:"in"`     // set of string
	NotIn types.Set    `tfsdk:"not_in"` // set of string
}

func (d *ArchetypeDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_archetype"
}

func (d *ArchetypeDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Archetype data source. This provides data in order to create resources. Where possible, the data is provided in the form of ARM JSON.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The management group name, forming part of the resource id.",
				Required:            true,
			},

			"display_name": schema.StringAttribute{
				MarkdownDescription: "The display name of the management group.",
				Optional:            true,
			},

			"parent_id": schema.StringAttribute{
				MarkdownDescription: "The parent management group name.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.RegexMatches(regexp.MustCompile("^[().a-zA-Z0-9_-]{1,90}$"), "Max length is 90 characters. ID can only contain an letter, digit, -, _, (, ), ."),
					stringvalidator.RegexMatches(regexp.MustCompile("^.*[^.]$"), "ID cannot end with a period"),
				},
			},

			"base_archetype": schema.StringAttribute{
				MarkdownDescription: "The base archetype name to use. This has been generated from the provider lib directories.",
				Required:            true,
			},

			"policy_assignments_to_modify": schema.MapNestedAttribute{
				MarkdownDescription: "A map of policy assignments names to change in the archetype. The map key is the policy assignment name." +
					"The policy assignment **must** exist in the archetype." +
					"The nested attributes will be merged with the existing policy assignment so you do not need to re-declare everything.",
				Optional: true,
				NestedObject: schema.NestedAttributeObject{
					Validators: []validator.Object{},
					Attributes: map[string]schema.Attribute{
						"enforcement_mode": schema.StringAttribute{
							MarkdownDescription: "The enforcement mode of the policy assignment. Must be one of `Default`, or `DoNotEnforce`.",
							Optional:            true,
							Validators: []validator.String{
								stringvalidator.OneOf("Default", "DoNotEnforce"),
							},
						},

						"identity": schema.StringAttribute{
							MarkdownDescription: "The identity type. Must be one of `SystemAssigned` or `UserAssigned`.",
							Optional:            true,
							Validators: []validator.String{
								stringvalidator.OneOf("SystemAssigned", "UserAssigned"),
							},
						},

						"identity_ids": schema.SetAttribute{
							MarkdownDescription: "A list of zero or one identity ids to assign to the policy assignment. Required if `identity` is `UserAssigned`.",
							Optional:            true,
							ElementType:         types.StringType,
							Validators: []validator.Set{
								setvalidator.ValueStringsAre(
									alzvalidators.ArmTypeResourceId("Microsoft.ManagedIdentity", "userAssignedIdentities"),
								),
								setvalidator.AlsoRequires(path.MatchRelative().AtParent().AtName("identity")),
								setvalidator.SizeBetween(0, 1),
							},
						},

						"non_compliance_message": schema.SetNestedAttribute{
							MarkdownDescription: "The non-compliance messages to use for the policy assignment.",
							Optional:            true,
							NestedObject: schema.NestedAttributeObject{
								Attributes: map[string]schema.Attribute{
									"message": schema.StringAttribute{
										MarkdownDescription: "The non-compliance message.",
										Required:            true,
									},

									"policy_definition_reference_id": schema.StringAttribute{
										MarkdownDescription: "The policy definition reference id (not the resource id) to use for the non compliance message. This references the definition within the policy set.",
										Optional:            true,
									},
								},
							},
						},

						"overrides": schema.ListNestedAttribute{
							MarkdownDescription: "The overrides for this policy assignment. There are a maximum of 10 overrides allowed per assignment. " +
								"If specified here the overrides will replace the existing overrides." +
								"The overrides are processed in the order they are specified.",
							Optional: true,
							Validators: []validator.List{
								listvalidator.SizeAtMost(10),
								listvalidator.UniqueValues(),
							},
							NestedObject: schema.NestedAttributeObject{
								Attributes: map[string]schema.Attribute{
									"kind": schema.StringAttribute{
										MarkdownDescription: "The property the assignment will override. The supported kind is `policyEffect`.",
										Required:            true,
										Validators: []validator.String{
											stringvalidator.OneOf("policyEffect"),
										},
									},

									"value": schema.StringAttribute{
										MarkdownDescription: "The new value which will override the existing value. The supported values are: `addToNetworkGroup`, `append`, `audit`, `auditIfNotExists`, `deny`, `denyAction`, `deployIfNotExists`, `disabled`, `manual`, `modify`, `mutate`.\n\n" +
											"<https://learn.microsoft.com/azure/governance/policy/concepts/effects>",
										Required: true,
										Validators: []validator.String{
											stringvalidator.OneOf("addToNetworkGroup", "append", "audit", "auditIfNotExists", "deny", "denyAction", "deployIfNotExists", "disabled", "manual", "modify", "mutate"),
										},
									},

									"selectors": schema.ListNestedAttribute{
										MarkdownDescription: "The selectors to use for the override.",
										Optional:            true,
										NestedObject: schema.NestedAttributeObject{
											Attributes: map[string]schema.Attribute{
												"kind": schema.StringAttribute{
													MarkdownDescription: "The property of a selector that describes what characteristic will narrow down the scope of the override. Allowed value for kind: `policyEffect` is: `policyDefinitionReferenceId`.",
													Required:            true,
													Validators: []validator.String{
														stringvalidator.OneOf("policyEffect"),
													},
												},
												"in": schema.SetAttribute{
													MarkdownDescription: "The list of values that the selector will match. The values are the policy definition reference ids. Conflicts with `not_in`.",
													Optional:            true,
													ElementType:         types.StringType,
													Validators: []validator.Set{
														setvalidator.SizeAtMost(50),
														setvalidator.ConflictsWith(path.MatchRelative().AtParent().AtName("not_in")),
													},
												},
												"not_in": schema.SetAttribute{
													MarkdownDescription: "The list of values that the selector will not match. The values are the policy definition reference ids. Conflicts with `in`.",
													Optional:            true,
													ElementType:         types.StringType,
													Validators: []validator.Set{
														setvalidator.SizeAtMost(50),
														setvalidator.ConflictsWith(path.MatchRelative().AtParent().AtName("in")),
													},
												},
											},
										},
									},
								},
							},
						},

						"resource_selectors": schema.ListNestedAttribute{
							MarkdownDescription: "The resource selectors to use for the policy assignment. " +
								"A maximum of 10 resource selectors are allowed per assignment. " +
								"If specified here the resource selectors will replace the existing resource selectors.",
							Optional: true,
							Validators: []validator.List{
								listvalidator.SizeAtMost(10),
								listvalidator.UniqueValues(),
							},
							NestedObject: schema.NestedAttributeObject{
								Attributes: map[string]schema.Attribute{
									"name": schema.StringAttribute{
										MarkdownDescription: "The name of the resource selector. The name must be unique within the assignment.",
										Required:            true,
										Validators: []validator.String{
											stringvalidator.LengthAtLeast(1),
										},
									},
									"selectors": schema.ListNestedAttribute{
										MarkdownDescription: "The selectors to use for the resource selector.",
										Optional:            true,
										NestedObject: schema.NestedAttributeObject{
											Attributes: map[string]schema.Attribute{
												"kind": schema.StringAttribute{
													MarkdownDescription: "The property of a selector that describes what characteristic will narrow down the set of evaluated resources. " +
														"Each kind can only be used once in a single resource selector. Allowed values are: `resourceLocation`, `resourceType`, `resourceWithoutLocation`. " +
														"`resourceWithoutLocation` cannot be used in the same resource selector as `resourceLocation`.",
													Required: true,
													Validators: []validator.String{
														stringvalidator.OneOf("resourceLocation", "resourceType", "resourceWithoutLocation"),
													},
												},
												"in": schema.SetAttribute{
													MarkdownDescription: "The list of values that the selector will match. Conflicts with `not_in`.",
													Optional:            true,
													ElementType:         types.StringType,
													Validators: []validator.Set{
														setvalidator.SizeAtMost(50),
														setvalidator.ConflictsWith(path.MatchRelative().AtParent().AtName("not_in")),
													},
												},
												"not_in": schema.SetAttribute{
													MarkdownDescription: "The list of values that the selector will not match. Conflicts with `in`.",
													Optional:            true,
													ElementType:         types.StringType,
													Validators: []validator.Set{
														setvalidator.SizeAtMost(50),
														setvalidator.ConflictsWith(path.MatchRelative().AtParent().AtName("in")),
													},
												},
											},
										},
									},
								},
							},
						},

						"parameters": schema.StringAttribute{
							MarkdownDescription: "The parameters to use for the policy assignment. " +
								"**Note:** This is a JSON string, and not a map. This is because the parameter values have different types, which confuses the type system used by the provider sdk. " +
								"Use `jsonencode()` to construct the map. " +
								"The map keys must be strings, the values are `any` type. " +
								"Example: `jsonencode({\"param1\": \"value1\", \"param2\": 2})`",
							CustomType: alztypes.PolicyParameterType{},
							Optional:   true,
						},
					},
				},
			},

			"defaults": schema.SingleNestedAttribute{
				MarkdownDescription: "Archetype default values",
				Required:            true,
				Attributes: map[string]schema.Attribute{
					"location": schema.StringAttribute{
						MarkdownDescription: "Default location",
						Required:            true,
					},
					"log_analytics_workspace_id": schema.StringAttribute{
						MarkdownDescription: "Default Log Analytics workspace id",
						Optional:            true,
						Validators: []validator.String{
							alzvalidators.ArmTypeResourceId("Microsoft.OperationalInsights", "workspaces"),
						},
					},
					"private_dns_zone_resource_group_id": schema.StringAttribute{
						MarkdownDescription: "Resource group resource id containing private DNS zones. Used in the Deploy-Private-DNS-Zones assignment.",
						Optional:            true,
						Validators: []validator.String{
							alzvalidators.ArmTypeResourceId("Microsoft.Resources", "resourceGroups"),
						},
					},
				},
			},

			"alz_policy_assignments": schema.MapAttribute{
				MarkdownDescription: "A map of generated policy assignments. The values are ARM JSON policy assignments.",
				Computed:            true,
				ElementType:         types.StringType,
			},

			"alz_policy_definitions": schema.MapAttribute{
				MarkdownDescription: "A map of generated policy assignments. The values are ARM JSON policy definitions.",
				Computed:            true,
				ElementType:         types.StringType,
			},

			"alz_policy_set_definitions": schema.MapAttribute{
				MarkdownDescription: "A map of generated policy assignments. The values are ARM JSON policy set definitions.",
				Computed:            true,
				ElementType:         types.StringType,
			},

			"alz_role_definitions": schema.MapAttribute{
				MarkdownDescription: "A map of generated role assignments. The values are ARM JSON role definitions.",
				Computed:            true,
				ElementType:         types.StringType,
			},

			"alz_policy_role_assignments": schema.MapNestedAttribute{
				MarkdownDescription: "A map of role assignments generated from the policy assignments. The values are a nested object containing the role definition ids and any additionl scopes.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"role_definition_id": schema.StringAttribute{
							MarkdownDescription: "The role definition id to assign with the policy assignment.",
							Computed:            true,
						},

						"scope": schema.StringAttribute{
							MarkdownDescription: "The scope to assign with the policy assignment.",
							Computed:            true,
						},

						"assignment_name": schema.StringAttribute{
							MarkdownDescription: "The name of the policy assignment.",
							Computed:            true,
						},
					},
				},
			},
		},
	}
}

func (d *ArchetypeDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	data, ok := req.ProviderData.(*alzProviderData)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *alzlibWithMutex, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	d.alz = data
}

func (d *ArchetypeDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data ArchetypeDataSourceModel

	if d.alz == nil {
		resp.Diagnostics.AddError(
			"Provider not configured",
			"The provider has not been configured. Please see the provider documentation for configuration instructions.",
		)
		return
	}

	// Read Terraform configuration data into the model.
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}
	d.alz.mu.Lock()
	defer d.alz.mu.Unlock()

	mgname := data.Id.ValueString()

	// Set well known policy values.
	wkpv := new(alzlib.WellKnownPolicyValues)
	defloc := to.Ptr(data.Defaults.DefaultLocation.ValueString())
	if *defloc == "" {
		resp.Diagnostics.AddError("Default location not set", "Unable to find default location in the archetype attributes. This should have been caught by the schema validation.")
	}
	wkpv.DefaultLocation = defloc
	if isKnown(data.Defaults.DefaultLaWorkspaceId) {
		wkpv.DefaultLogAnalyticsWorkspaceId = to.Ptr(data.Defaults.DefaultLaWorkspaceId.ValueString())
	}
	if isKnown(data.Defaults.PrivateDnsZoneResourceGroupId) {
		wkpv.PrivateDnsZoneResourceGroupId = to.Ptr(data.Defaults.PrivateDnsZoneResourceGroupId.ValueString())
	}

	// Make a copy of the archetype so we can customize it.
	arch, err := d.alz.CopyArchetype(data.BaseArchetype.ValueString(), wkpv)
	if err != nil {
		resp.Diagnostics.AddError("Archetype not found", fmt.Sprintf("Unable to find archetype %s", data.BaseArchetype.ValueString()))
		return
	}

	checks := []checkExistsInAlzLib{
		{arch.PolicyDefinitions, d.alz.PolicyDefinitionExists},
		{arch.PolicySetDefinitions, d.alz.PolicySetDefinitionExists},
		{arch.RoleDefinitions, d.alz.RoleDefinitionExists},
		{arch.PolicyAssignments, d.alz.PolicyAssignmentExists},
	}

	for _, check := range checks {
		for item := range check.set.Iter() {
			if !check.f(item) {
				resp.Diagnostics.AddError("Item not found", fmt.Sprintf("Unable to find %s in the AlzLib", item))
				return
			}
		}
	}

	if mg := d.alz.Deployment.GetManagementGroup(mgname); mg == nil {
		tflog.Debug(ctx, "Add management group")
		external := false
		parent := data.ParentId.ValueString()
		if mg := d.alz.Deployment.GetManagementGroup(parent); mg == nil {
			external = true
		}
		req := alzlib.AlzManagementGroupAddRequest{
			Id:               mgname,
			DisplayName:      data.DisplayName.ValueString(),
			ParentId:         parent,
			ParentIsExternal: external,
			Archetype:        arch,
		}
		if err := d.alz.AddManagementGroupToDeployment(ctx, req); err != nil {
			resp.Diagnostics.AddError("Unable to add management group", err.Error())
			return
		}
	}

	mg := d.alz.Deployment.GetManagementGroup(mgname)
	if mg == nil {
		resp.Diagnostics.AddError("Unable to find management group after adding", fmt.Sprintf("Unable to find management group %s", mgname))
		return
	}

	for k, v := range data.PolicyAssignmentsToModify {
		enf, ident, noncompl, params, resourceSel, overrides, err := policyAssignmentType2ArmPolicyValues(v)
		if err != nil {
			resp.Diagnostics.AddError(fmt.Sprintf("Unable to convert supplied policy assignment modifications to SDK values for policy assignment %s", k), err.Error())
			return
		}
		if err := mg.ModifyPolicyAssignment(k, params, enf, noncompl, ident, resourceSel, overrides); err != nil {
			resp.Diagnostics.AddError(fmt.Sprintf("Unable to modify policy assignment %s", k), err.Error())
			return

		}
	}

	if err := mg.GeneratePolicyAssignmentAdditionalRoleAssignments(d.alz.AlzLib); err != nil {
		resp.Diagnostics.AddError("Unable to generate additional role assignments", err.Error())
		return
	}

	tflog.Debug(ctx, "Converting maps from Go types to Framework types")
	var m basetypes.MapValue
	var diags diag.Diagnostics

	tflog.Debug(ctx, "Converting policy assignments")
	m, diags = convertMapOfStringToMapValue(mg.GetPolicyAssignmentMap())
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.AlzPolicyAssignments = m

	tflog.Debug(ctx, "Converting policy definitions")
	m, diags = convertMapOfStringToMapValue(mg.GetPolicyDefinitionsMap())
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.AlzPolicyDefinitions = m

	tflog.Debug(ctx, "Converting policy set definitions")
	m, diags = convertMapOfStringToMapValue(mg.GetPolicySetDefinitionsMap())
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.AlzPolicySetDefinitions = m

	tflog.Debug(ctx, "Converting role definitions")
	m, diags = convertMapOfStringToMapValue(mg.GetRoleDefinitionsMap())
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.AlzRoleDefinitions = m

	tflog.Debug(ctx, "Converting additional role assignments")
	data.AlzPolicyRoleAssignments = convertAlzPolicyRoleAssignments(mg.GetPolicyRoleAssignments())

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// convertAlzPolicyRoleAssignments converts a map[string]alzlib.PolicyAssignmentAdditionalRoleAssignments to a map[string]AlzPolicyRoleAssignmentType.
func convertAlzPolicyRoleAssignments(src []alzlib.PolicyRoleAssignment) map[string]AlzPolicyRoleAssignmentType {
	if len(src) == 0 {
		return nil
	}
	res := make(map[string]AlzPolicyRoleAssignmentType, len(src))
	for _, v := range src {
		res[genPolicyRoleAssignmentId(v)] = AlzPolicyRoleAssignmentType{
			RoleDefinitionId: types.StringValue(v.RoleDefinitionId),
			Scope:            types.StringValue(v.Scope),
			AssignmentName:   types.StringValue(v.AssignmentName),
		}
	}
	return res
}

// convertMapOfStringToMapValue converts a map[string]armTypes to a map[string]attr.Value, using types.StringType as the value type.
func convertMapOfStringToMapValue[T mapTypes](m map[string]T) (basetypes.MapValue, diag.Diagnostics) {
	result := make(map[string]attr.Value, len(m))
	for k, v := range m {
		b, err := json.Marshal(v)
		if err != nil {
			var diags diag.Diagnostics
			diags.AddError("Unable to marshal ARM object", err.Error())
			return basetypes.NewMapNull(types.StringType), diags
		}
		result[k] = types.StringValue(string(b))
	}
	resultMapType, diags := types.MapValue(types.StringType, result)
	if diags.HasError() {
		return basetypes.NewMapNull(types.StringType), diags
	}
	return resultMapType, nil
}

// policyAssignmentType2ArmPolicyValues returns a set of Azure Go SDK values from a PolicyAssignmentType.
// This is used to modify existing policy assignments.
func policyAssignmentType2ArmPolicyValues(pa PolicyAssignmentType) (
	enforcementMode *armpolicy.EnforcementMode,
	identity *armpolicy.Identity,
	nonComplianceMessages []*armpolicy.NonComplianceMessage,
	parameters map[string]*armpolicy.ParameterValuesValue,
	resourceSelectors []*armpolicy.ResourceSelector,
	overrides []*armpolicy.Override,
	err error) {
	// Set enforcement mode.
	enforcementMode = convertPolicyAssignmentEnforcementModeToSdkType(pa.EnforcementMode)

	// set identity
	identity, err = convertPolicyAssignmentIdentityToSdkType(pa.Identity, pa.IdentityIds)
	if err != nil {
		return nil, nil, nil, nil, nil, nil, fmt.Errorf("unable to convert policy assignment to sdk type: %w", err)
	}

	// set non-compliance message
	nonComplianceMessages = convertPolicyAssignmentNonComplianceMessagesToSdkType(pa.NonComplianceMessage)

	// set parameters
	parameters, err = convertPolicyAssignmentParametersToSdkType(pa.Parameters)
	if err != nil {
		return nil, nil, nil, nil, nil, nil, fmt.Errorf("unable to convert policy assignment parameters to sdk type: %w", err)
	}

	resourceSelectors, err = convertPolicyAssignmentResourceSelectorsToSdkType(pa.ResourceSelectors)
	if err != nil {
		return nil, nil, nil, nil, nil, nil, fmt.Errorf("unable to convert policy assignment resource selectors to sdk type: %w", err)
	}

	overrides, err = convertPolicyAssignmentOverridesToSdkType(pa.Overrides)
	if err != nil {
		return nil, nil, nil, nil, nil, nil, fmt.Errorf("unable to convert policy assignment overrides to sdk type: %w", err)
	}

	return enforcementMode, identity, nonComplianceMessages, parameters, resourceSelectors, overrides, nil
}

func convertPolicyAssignmentOverridesToSdkType(src []PolicyAssignmentOverrideType) ([]*armpolicy.Override, error) {
	if len(src) == 0 {
		return nil, nil
	}
	res := make([]*armpolicy.Override, len(src))
	for i, o := range src {
		selectors := make([]*armpolicy.Selector, len(o.Selectors))
		for j, s := range o.Selectors {
			in, err := typehelper.AttrSlice2StringSlice(s.In.Elements())
			if err != nil {
				return nil, fmt.Errorf("unable to convert override selector `in` in value to string %w", err)
			}
			notIn, err := typehelper.AttrSlice2StringSlice(s.NotIn.Elements())
			if err != nil {
				return nil, fmt.Errorf("unable to convert override selector `not_in` in value to string %w", err)
			}
			selectors[j] = &armpolicy.Selector{
				Kind:  to.Ptr(armpolicy.SelectorKind(s.Kind.ValueString())),
				In:    to.SliceOfPtrs(in...),
				NotIn: to.SliceOfPtrs(notIn...),
			}
		}
		res[i] = &armpolicy.Override{
			Kind:      to.Ptr(armpolicy.OverrideKind(o.Kind.ValueString())),
			Value:     to.Ptr(o.Value.ValueString()),
			Selectors: selectors,
		}
	}
	return res, nil
}

func convertPolicyAssignmentResourceSelectorsToSdkType(src []ResourceSelectorType) ([]*armpolicy.ResourceSelector, error) {
	if len(src) == 0 {
		return nil, nil
	}
	res := make([]*armpolicy.ResourceSelector, len(src))
	for i, rs := range src {
		selectors := make([]*armpolicy.Selector, len(rs.Selectors))
		for j, s := range rs.Selectors {
			in, err := typehelper.AttrSlice2StringSlice(s.In.Elements())
			if err != nil {
				return nil, fmt.Errorf("unable to convert resource selector selector `in` in value to string %w", err)
			}
			notIn, err := typehelper.AttrSlice2StringSlice(s.NotIn.Elements())
			if err != nil {
				return nil, fmt.Errorf("unable to convert resource selector selector `not_in` in value to string %w", err)
			}
			selectors[j] = &armpolicy.Selector{
				Kind:  to.Ptr(armpolicy.SelectorKind(s.Kind.ValueString())),
				In:    to.SliceOfPtrs(in...),
				NotIn: to.SliceOfPtrs(notIn...),
			}
		}
		res[i] = &armpolicy.ResourceSelector{
			Name:      to.Ptr(rs.Name.ValueString()),
			Selectors: selectors,
		}
	}
	return res, nil
}

func convertPolicyAssignmentEnforcementModeToSdkType(src types.String) *armpolicy.EnforcementMode {
	if !isKnown(src) {
		return nil
	}
	switch src.ValueString() {
	case "DoNotEnforce":
		return to.Ptr(armpolicy.EnforcementModeDoNotEnforce)
	case "Default":
		return to.Ptr(armpolicy.EnforcementModeDefault)
	}
	return nil
}

func convertPolicyAssignmentNonComplianceMessagesToSdkType(src []PolicyAssignmentNonComplianceMessage) []*armpolicy.NonComplianceMessage {
	res := make([]*armpolicy.NonComplianceMessage, len(src))
	if len(src) > 0 {
		for i, msg := range src {
			res[i] = &armpolicy.NonComplianceMessage{
				Message: to.Ptr(msg.Message.ValueString()),
			}
			if isKnown(msg.PolicyDefinitionReferenceId) {
				res[i].PolicyDefinitionReferenceID = to.Ptr(msg.PolicyDefinitionReferenceId.ValueString())
			}
		}
	}
	return res
}

func convertPolicyAssignmentIdentityToSdkType(typ types.String, ids types.Set) (*armpolicy.Identity, error) {
	if !isKnown(typ) {
		return nil, nil
	}
	var identity *armpolicy.Identity
	switch typ.ValueString() {
	case "SystemAssigned":
		identity = to.Ptr(armpolicy.Identity{
			Type: to.Ptr(armpolicy.ResourceIdentityTypeSystemAssigned),
		})
	case "UserAssigned":
		if ids.IsUnknown() {
			return nil, nil
		}
		var id string
		if len(ids.Elements()) != 1 {
			return nil, fmt.Errorf("one (and only one) identity id is required for user assigned identity")
		}
		idStr, ok := ids.Elements()[0].(types.String)
		if !ok {
			return nil, fmt.Errorf("unable to convert identity id to string")
		}
		id = idStr.ValueString()

		identity = to.Ptr(armpolicy.Identity{
			Type:                   to.Ptr(armpolicy.ResourceIdentityTypeUserAssigned),
			UserAssignedIdentities: map[string]*armpolicy.UserAssignedIdentitiesValue{id: {}},
		})
	default:
		return nil, fmt.Errorf("unknown identity type: %s", typ.ValueString())
	}
	return identity, nil
}

// convertPolicyAssignmentParametersToSdkType converts a map[string]any to a map[string]*armpolicy.ParameterValuesValue.
func convertPolicyAssignmentParametersToSdkType(src alztypes.PolicyParameterValue) (map[string]*armpolicy.ParameterValuesValue, error) {
	if !isKnown(src) {
		return nil, nil
	}
	params := make(map[string]any)
	if err := json.Unmarshal([]byte(src.ValueString()), &params); err != nil {
		return nil, fmt.Errorf("unable to unmarshal policy parameters: %w", err)
	}
	if len(params) == 0 {
		return nil, nil
	}
	res := make(map[string]*armpolicy.ParameterValuesValue, len(params))
	for k, v := range params {
		val := new(armpolicy.ParameterValuesValue)
		val.Value = v
		res[k] = val
	}
	return res, nil
}

func isKnown(val attr.Value) bool {
	return !val.IsNull() && !val.IsUnknown()
}

func genPolicyRoleAssignmentId(pra alzlib.PolicyRoleAssignment) string {
	u := uuid.NewSHA1(uuid.NameSpaceURL, []byte(pra.AssignmentName+pra.RoleDefinitionId+pra.Scope))
	return u.String()
}
