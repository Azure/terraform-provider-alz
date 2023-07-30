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
	mapset "github.com/deckarep/golang-set/v2"
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
	"github.com/matt-FFFFFF/terraform-provider-alz/internal/alztypes"
	"github.com/matt-FFFFFF/terraform-provider-alz/internal/alzvalidators"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &ArchetypeDataSource{}

func NewArchetypeDataSource() datasource.DataSource {
	return &ArchetypeDataSource{}
}

// ArchetypeDataSource defines the data source implementation.
type ArchetypeDataSource struct {
	alz *alzlibWithMutex
}

// mapTypes is used for the generic functions that operate on certain map types.
type mapTypes interface {
	armpolicy.Assignment |
		armpolicy.Definition |
		armpolicy.SetDefinition |
		armauthorization.RoleAssignment |
		armauthorization.RoleDefinition |
		alzlib.PolicyAssignmentAdditionalRoleAssignments
}

// checkExistsInAlzLib is a helper struct to check if an item exists in the AlzLib.
type checkExistsInAlzLib struct {
	set mapset.Set[string]
	f   func(string) bool
}

// ArchetypeDataSourceModel describes the data source data model.
type ArchetypeDataSourceModel struct {
	AlzPolicyAssignments         types.Map                              `tfsdk:"alz_policy_assignments"`      // map of string, computed
	AlzPolicyDefinitions         types.Map                              `tfsdk:"alz_policy_definitions"`      // map of string, computed
	AlzPolicySetDefinitions      types.Map                              `tfsdk:"alz_policy_set_definitions"`  // map of string, computed
	AlzRoleAssignments           types.Map                              `tfsdk:"alz_role_assignments"`        // map of string, computed
	AlzPolicyRoleAssignments     map[string]AlzPolicyRoleAssignmentType `tfsdk:"alz_policy_role_assignments"` // map of string, computed
	AlzRoleDefinitions           types.Map                              `tfsdk:"alz_role_definitions"`        // map of string, computed
	BaseArchetype                types.String                           `tfsdk:"base_archetype"`
	Defaults                     ArchetypeDataSourceModelDefaults       `tfsdk:"defaults"`
	DisplayName                  types.String                           `tfsdk:"display_name"`
	Id                           types.String                           `tfsdk:"id"`
	ParentId                     types.String                           `tfsdk:"parent_id"`
	PolicyAssignmentsToAdd       map[string]PolicyAssignmentType        `tfsdk:"policy_assignments_to_add"`        // map of PolicyAssignmentType
	PolicyAssignmentsToRemove    types.Set                              `tfsdk:"policy_assignments_to_remove"`     // set of string
	PolicyDefinitionsToAdd       types.Set                              `tfsdk:"policy_definitions_to_add"`        // set of string
	PolicyDefinitionsToRemove    types.Set                              `tfsdk:"policy_definitions_to_remove"`     // set of string
	PolicySetDefinitionsToAdd    types.Set                              `tfsdk:"policy_set_definitions_to_add"`    // set of string
	PolicySetDefinitionsToRemove types.Set                              `tfsdk:"policy_set_definitions_to_remove"` // set of string
	RoleAssignmentsToAdd         map[string]RoleAssignmentType          `tfsdk:"role_assignments_to_add"`          // map of RoleAssignmentType
	RoleDefinitionsToAdd         types.Set                              `tfsdk:"role_definitions_to_add"`          // set of string
	RoleDefinitionsToRemove      types.Set                              `tfsdk:"role_definitions_to_remove"`       // set of string
	SubscriptionIds              types.Set                              `tfsdk:"subscription_ids"`                 // set of string
}

// AlzPolicyRoleAssignmentType is a representation of the additional policy assignments
// that must be created when assigning a given policy.
type AlzPolicyRoleAssignmentType struct {
	RoleDefinitionIds types.Set `tfsdk:"role_definition_ids"`
	AdditionalScopes  types.Set `tfsdk:"additional_scopes"`
}

// ArchetypeDataSourceModelDefaults describes the defaults used in the alz data processing.
type ArchetypeDataSourceModelDefaults struct {
	DefaultLocation      types.String `tfsdk:"location"`
	DefaultLaWorkspaceId types.String `tfsdk:"log_analytics_workspace_id"`
}

// PolicyAssignmentType describes the policy assignment data model.
type PolicyAssignmentType struct {
	DisplayName             types.String                           `tfsdk:"display_name"`
	PolicyDefinitionName    types.String                           `tfsdk:"policy_definition_name"`
	PolicySetDefinitionName types.String                           `tfsdk:"policy_set_definition_name"`
	PolicyDefinitionId      types.String                           `tfsdk:"policy_definition_id"`
	EnforcementMode         types.String                           `tfsdk:"enforcement_mode"`
	Identity                types.String                           `tfsdk:"identity"`
	IdentityIds             types.Set                              `tfsdk:"identity_ids"`           // set of string
	NonComplianceMessage    []PolicyAssignmentNonComplianceMessage `tfsdk:"non_compliance_message"` // set of PolicyAssignmentNonComplianceMessage
	Parameters              alztypes.PolicyParameterValue          `tfsdk:"parameters"`
}

// PolicyAssignmentNonComplianceMessage describes non-compliance message in a policy assignment.
type PolicyAssignmentNonComplianceMessage struct {
	Message                     types.String `tfsdk:"message"`
	PolicyDefinitionReferenceId types.String `tfsdk:"policy_definition_reference_id"`
}

type RoleAssignmentType struct {
	DefinitionName types.String `tfsdk:"definition_name"`
	DefinitionId   types.String `tfsdk:"definition_id"`
	ObjectId       types.String `tfsdk:"object_id"`
}

func (d *ArchetypeDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_archetype"
}

func (d *ArchetypeDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "Archetype data source.",

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

			"policy_assignments_to_remove": schema.SetAttribute{
				MarkdownDescription: "A list of policy assignment names to remove from the archetype.",
				Optional:            true,
				ElementType:         types.StringType,
			},

			"policy_definitions_to_remove": schema.SetAttribute{
				MarkdownDescription: "A list of policy definition names to remove from the archetype.",
				Optional:            true,
				ElementType:         types.StringType,
			},

			"policy_set_definitions_to_remove": schema.SetAttribute{
				MarkdownDescription: "A list of policy set definition names to remove from the archetype.",
				Optional:            true,
				ElementType:         types.StringType,
			},

			"role_definitions_to_remove": schema.SetAttribute{
				MarkdownDescription: "A list of role definition names to remove from the archetype.",
				Optional:            true,
				ElementType:         types.StringType,
			},

			"policy_assignments_to_add": schema.MapNestedAttribute{
				MarkdownDescription: "A map of policy assignments names to add to the archetype. The map key is the policy assignment name.",
				Optional:            true,
				NestedObject: schema.NestedAttributeObject{
					Validators: []validator.Object{},
					Attributes: map[string]schema.Attribute{
						"display_name": schema.StringAttribute{
							MarkdownDescription: "The policy assignment display name",
							Required:            true,
						},

						"policy_definition_name": schema.StringAttribute{
							MarkdownDescription: "The name of the policy definition to assign. Must be in the AlzLib, if not use `policy_definition_id` instead. Conflicts with `policy_definition_id` and `policy_set_definition_name`.",
							Optional:            true,
							Validators: []validator.String{
								stringvalidator.ConflictsWith(path.MatchRelative().AtParent().AtName("policy_definition_id")),
								stringvalidator.ConflictsWith(path.MatchRelative().AtParent().AtName("policy_set_definition_name")),
							},
						},

						"policy_set_definition_name": schema.StringAttribute{
							MarkdownDescription: "The name of the policy set definition to assign. Must be in the AlzLib, if not use `policy_definition_id` instead. Conflicts with `policy_definition_id` and `policy_definition_name`.",
							Optional:            true,
							Validators: []validator.String{
								stringvalidator.ConflictsWith(path.MatchRelative().AtParent().AtName("policy_definition_id")),
								stringvalidator.ConflictsWith(path.MatchRelative().AtParent().AtName("policy_definition_name")),
							},
						},

						"policy_definition_id": schema.StringAttribute{
							MarkdownDescription: "The resource id of the policy definition. Conflicts with `policy_definition_name` and `policy_set_definition_name`.",
							Optional:            true,
							Validators: []validator.String{
								stringvalidator.ConflictsWith(path.MatchRelative().AtParent().AtName("policy_set_definition_name")),
								stringvalidator.ConflictsWith(path.MatchRelative().AtParent().AtName("policy_definition_name")),
							},
						},

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
							MarkdownDescription: "A list of identity ids to assign to the policy assignment. Required if `identity` is `UserAssigned`.",
							Optional:            true,
							ElementType:         types.StringType,
							Validators: []validator.Set{
								setvalidator.ValueStringsAre(
									alzvalidators.ArmTypeResourceId("Microsoft.ManagedIdentity", "userAssignedIdentities"),
									stringvalidator.AlsoRequires(path.MatchRelative().AtParent().AtName("identity")),
								),
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

						"parameters": schema.StringAttribute{
							MarkdownDescription: "The parameters to use for the policy assignment. " +
								"**Note:** This is a JSON string, and not a map. This is because the parameter values have different types, which confuses the type system used by the provider sdk. " +
								"Use `jsonencode()` to construct the map. " +
								"The map keys must be strings, the values are `any` type.",
							CustomType: alztypes.PolicyParameterType{},
							Optional:   true,
						},
					},
				},
			},

			"policy_definitions_to_add": schema.SetAttribute{
				MarkdownDescription: "A list of policy definition names to add to the archetype.",
				Optional:            true,
				ElementType:         types.StringType,
			},

			"policy_set_definitions_to_add": schema.SetAttribute{
				MarkdownDescription: "A list of policy set definition names to add to the archetype.",
				Optional:            true,
				ElementType:         types.StringType,
			},

			"role_definitions_to_add": schema.SetAttribute{
				MarkdownDescription: "A list of role definition names to add to the archetype.",
				Optional:            true,
				ElementType:         types.StringType,
			},

			"role_assignments_to_add": schema.MapNestedAttribute{
				MarkdownDescription: "A list of role definition names to add to the archetype.",
				Optional:            true,
				NestedObject: schema.NestedAttributeObject{
					Validators: []validator.Object{},
					Attributes: map[string]schema.Attribute{
						"definition_id": schema.StringAttribute{
							MarkdownDescription: "The role definition name. Conflicts with `definition_name`.",
							Optional:            true,
							Validators: []validator.String{
								alzvalidators.ArmTypeResourceId("Microsoft.Authorization", "roleDefinitions"),
								stringvalidator.ConflictsWith(path.MatchRelative().AtParent().AtName("definition_name")),
							},
						},
						"definition_name": schema.StringAttribute{
							MarkdownDescription: "The role definition resource id. Conflicts with `definition_id`.",
							Optional:            true,
							Validators: []validator.String{
								stringvalidator.ConflictsWith(path.MatchRelative().AtParent().AtName("definition_id")),
							},
						},
						"object_id": schema.StringAttribute{
							MarkdownDescription: "The principal object id to assign.",
							Required:            true,
							Validators: []validator.String{
								stringvalidator.RegexMatches(regexp.MustCompile(`^[0-9a-f]{8}-([0-9a-f]{4}-){3}[0-9a-f]{12}$`), "The subscription id must be a valid lowercase UUID."),
							},
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
				},
			},

			"subscription_ids": schema.SetAttribute{
				MarkdownDescription: "A list of subscription ids to add to the management group.",
				Optional:            true,
				ElementType:         types.StringType,
				Validators: []validator.Set{
					setvalidator.ValueStringsAre(
						stringvalidator.RegexMatches(regexp.MustCompile(`^[0-9a-f]{8}-([0-9a-f]{4}-){3}[0-9a-f]{12}$`), "The subscription id must be a valid lowercase UUID."),
					),
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

			"alz_role_assignments": schema.MapAttribute{
				MarkdownDescription: "A map of generated role assignments. The values are ARM JSON role assignments.",
				Computed:            true,
				ElementType:         types.StringType,
			},

			"alz_role_definitions": schema.MapAttribute{
				MarkdownDescription: "A map of generated role assignments. The values are ARM JSON role definitions.",
				Computed:            true,
				ElementType:         types.StringType,
			},

			"alz_policy_role_assignments": schema.MapNestedAttribute{
				MarkdownDescription: "A map of role assignments by policy assignment name. The values are a nested object containing the role definition ids and any additionl scopes.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"role_definition_ids": schema.SetAttribute{
							MarkdownDescription: "A set of role definition ids to assign with the policy assignment.",
							ElementType:         types.StringType,
							Computed:            true,
						},

						"additional_scopes": schema.SetAttribute{
							MarkdownDescription: "A set of additional scopes to assign with the policy assignment.",
							ElementType:         types.StringType,
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

	data, ok := req.ProviderData.(*alzlibWithMutex)

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
	defloc := data.Defaults.DefaultLocation.ValueString()
	if defloc == "" {
		resp.Diagnostics.AddError("Default location not set", "Unable to find default location in the archetype attributes. This should have been caught by the schema validation.")
	}
	wkpv.DefaultLocation = defloc
	wkpv.DefaultLogAnalyticsWorkspaceId = data.Defaults.DefaultLaWorkspaceId.ValueString()

	// Make a copy of the archetype so we can customize it.
	arch, err := d.alz.CopyArchetype(data.BaseArchetype.ValueString(), wkpv)
	if err != nil {
		resp.Diagnostics.AddError("Archetype not found", fmt.Sprintf("Unable to find archetype %s", data.BaseArchetype.ValueString()))
		return
	}

	// Add/remove items from archetype before adding the management group.
	if err := addAttrStringElementsToSet(arch.PolicyDefinitions, data.PolicyDefinitionsToAdd.Elements()); err != nil {
		resp.Diagnostics.AddError("Unable to add policy definitions", err.Error())
		return
	}
	if err := deleteAttrStringElementsFromSet(arch.PolicyDefinitions, data.PolicyDefinitionsToRemove.Elements()); err != nil {
		resp.Diagnostics.AddError("Unable to remove policy definitions", err.Error())
		return
	}

	if err := addAttrStringElementsToSet(arch.PolicySetDefinitions, data.PolicySetDefinitionsToAdd.Elements()); err != nil {
		resp.Diagnostics.AddError("Unable to add policy set definitions", err.Error())
		return
	}
	if err := deleteAttrStringElementsFromSet(arch.PolicySetDefinitions, data.PolicySetDefinitionsToRemove.Elements()); err != nil {
		resp.Diagnostics.AddError("Unable to remove policy set definitions", err.Error())
		return
	}

	if err := addAttrStringElementsToSet(arch.RoleDefinitions, data.RoleDefinitionsToAdd.Elements()); err != nil {
		resp.Diagnostics.AddError("Unable to add role definitions", err.Error())
		return
	}
	if err := deleteAttrStringElementsFromSet(arch.RoleDefinitions, data.RoleDefinitionsToRemove.Elements()); err != nil {
		resp.Diagnostics.AddError("Unable to remove role definitions", err.Error())
		return
	}

	// TODO: implement code to create *armauthorization.RoleAssignment from RoleAssignmentsToAdd.
	// TODO: implement code to populate subscription ids
	// TODO: implement code to create *armpolicy.Assignment from PolicyAssignmentsToAdd.

	if err := deleteAttrStringElementsFromSet(arch.PolicyAssignments, data.PolicyAssignmentsToRemove.Elements()); err != nil {
		resp.Diagnostics.AddError("Unable to remove policy assignments", err.Error())
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

	// TODO: change this to compare the config to the AlzManagementGroup in the alz struct.
	// It should be identical. If it is not then error as the user has duplicate management group names.
	if mg := d.alz.Deployment.GetManagementGroup(mgname); mg == nil {
		tflog.Debug(ctx, "Add management group")
		external := false
		parent := data.ParentId.ValueString()
		if mg := d.alz.Deployment.GetManagementGroup(parent); mg == nil {
			external = true
		}

		if err := d.alz.AddManagementGroupToDeployment(mgname, data.DisplayName.ValueString(), data.ParentId.ValueString(), external, arch); err != nil {
			resp.Diagnostics.AddError("Unable to add management group", err.Error())
			return
		}
	}

	mg := d.alz.Deployment.GetManagementGroup(mgname)
	if mg == nil {
		resp.Diagnostics.AddError("Unable to find management group after adding", fmt.Sprintf("Unable to find management group %s", mgname))
		return
	}

	// check that the policy assignment referenced definition names are in alz
	for _, pa := range data.PolicyAssignmentsToAdd {
		if isKnown(pa.PolicyDefinitionName) && !d.alz.PolicyDefinitionExists(pa.PolicyDefinitionName.ValueString()) {
			resp.Diagnostics.AddError("Policy definition not found", fmt.Sprintf("Unable to find policy definition %s", pa.PolicyDefinitionName.ValueString()))
			return
		}
		if isKnown(pa.PolicySetDefinitionName) && !d.alz.PolicySetDefinitionExists(pa.PolicySetDefinitionName.ValueString()) {
			resp.Diagnostics.AddError("Policy set definition not found", fmt.Sprintf("Unable to find policy set definition %s", pa.PolicySetDefinitionName.ValueString()))
			return
		}
	}

	// add new policy assignments to deployed management group and run Update to set the correct references, etc.
	newPas, err := policyAssignmentType2ArmPolicyAssignment(data.PolicyAssignmentsToAdd, d.alz.AlzLib)
	if err != nil {
		resp.Diagnostics.AddError("Unable to create policy assignments", err.Error())
		return
	}

	if err := mg.UpsertPolicyAssignments(ctx, newPas, d.alz.AlzLib); err != nil {
		resp.Diagnostics.AddError("Unable to add policy assignments", err.Error())
		return
	}

	// if err := d.alz.Deployment.MGs[mgname].Update(d.alz, wkpv); err != nil {
	// 	resp.Diagnostics.AddError("Unable to update management group", err.Error())
	// 	return
	// }

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

	tflog.Debug(ctx, "Converting role assignments")
	m, diags = convertMapOfStringToMapValue(mg.GetRoleAssignmentsMap())
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.AlzRoleAssignments = m

	tflog.Debug(ctx, "Converting role definitions")
	m, diags = convertMapOfStringToMapValue(mg.GetRoleDefinitionsMap())
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.AlzRoleDefinitions = m

	tflog.Debug(ctx, "Converting additional role assignments")
	policyras, diags := convertAlzPolicyRoleAssignments(ctx, mg.GetAdditionalRoleAssignmentsByPolicyAssignmentMap())
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.AlzPolicyRoleAssignments = policyras

	//Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// convertAlzPolicyRoleAssignments converts a map[string]alzlib.PolicyAssignmentAdditionalRoleAssignments to a map[string]AlzPolicyRoleAssignmentType
func convertAlzPolicyRoleAssignments(ctx context.Context, m map[string]alzlib.PolicyAssignmentAdditionalRoleAssignments) (map[string]AlzPolicyRoleAssignmentType, diag.Diagnostics) {
	res := make(map[string]AlzPolicyRoleAssignmentType, len(m))
	diags := make(diag.Diagnostics, 0)
	for k, v := range m {
		raset, d := types.SetValueFrom(ctx, types.StringType, v.RoleDefinitionIds)
		diags.Append(d...)
		adscopeset, d := types.SetValueFrom(ctx, types.StringType, v.AdditionalScopes)
		diags.Append(d...)
		res[k] = AlzPolicyRoleAssignmentType{
			RoleDefinitionIds: raset,
			AdditionalScopes:  adscopeset,
		}
	}
	return res, diags
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

// addAttrStringElementsToSet adds the string values of the attr.Value elements to the set.
// It is used to add elements to the archetype.
func addAttrStringElementsToSet(set mapset.Set[string], vals []attr.Value) error {
	for _, attr := range vals {
		s, ok := attr.(types.String)
		if !ok {
			return fmt.Errorf("unable to convert %v to types.String", attr)
		}
		set.Add(s.ValueString())
	}
	return nil
}

// deleteAttrStringElementsFromSet removed the string values of the attr.Value elements to the set.
// It is used to remove elements from the archetype.
func deleteAttrStringElementsFromSet(set mapset.Set[string], vals []attr.Value) error {
	for _, attr := range vals {
		s, ok := attr.(types.String)
		if !ok {
			return fmt.Errorf("unable to convert %v to types.String", attr)
		}
		if !set.Contains(s.ValueString()) {
			continue
		}
		set.Remove(s.ValueString())
	}
	return nil
}

func policyAssignmentType2ArmPolicyAssignment(pamap map[string]PolicyAssignmentType, az *alzlib.AlzLib) (map[string]*armpolicy.Assignment, error) {
	const (
		policyAssignmentIdFmt    = "/providers/Microsoft.Management/managementGroups/placeholder/providers/Microsoft.Authorization/policyAssignments/%s"
		policyDefinitionIdFmt    = "/providers/Microsoft.Management/managementGroups/placeholder/providers/Microsoft.Authorization/policyAssignments/%s"
		policySetDefinitionIdFmt = "/providers/Microsoft.Management/managementGroups/placeholder/providers/Microsoft.Authorization/policyAssignments/%s"
		policyAssignementType    = "Microsoft.Authorization/policyAssignments"
	)
	res := make(map[string]*armpolicy.Assignment, len(pamap))
	for name, src := range pamap {
		dst := new(armpolicy.Assignment)
		dst.Properties = new(armpolicy.AssignmentProperties)
		dst.ID = to.Ptr(fmt.Sprintf(policyAssignmentIdFmt, name))
		dst.Name = to.Ptr(name)
		dst.Type = to.Ptr(policyAssignementType)
		dst.Properties.DisplayName = to.Ptr(src.DisplayName.ValueString())

		// Set policy definition id.
		if isKnown(src.PolicyDefinitionName) {
			if !az.PolicyDefinitionExists(src.PolicyDefinitionName.ValueString()) {
				return nil, fmt.Errorf("policy definition %s not found in AlzLib", src.PolicyDefinitionName.ValueString())
			}
			dst.Properties.PolicyDefinitionID = to.Ptr(fmt.Sprintf(policyDefinitionIdFmt, src.PolicyDefinitionName.ValueString()))
		}
		if isKnown(src.PolicySetDefinitionName) {
			if !az.PolicySetDefinitionExists(src.PolicySetDefinitionName.ValueString()) {
				return nil, fmt.Errorf("policy set definition %s not found in AlzLib", src.PolicyDefinitionName.ValueString())
			}
			dst.Properties.PolicyDefinitionID = to.Ptr(fmt.Sprintf(policySetDefinitionIdFmt, src.PolicyDefinitionName.ValueString()))
		}
		if isKnown(src.PolicyDefinitionId) {
			dst.Properties.PolicyDefinitionID = to.Ptr(src.PolicyDefinitionId.ValueString())
		}

		// Set enforcement mode.
		if isKnown(src.EnforcementMode) {
			switch src.EnforcementMode.ValueString() {
			case "DoNotEnforce":
				dst.Properties.EnforcementMode = to.Ptr(armpolicy.EnforcementModeDoNotEnforce)
			case "Default":
				dst.Properties.EnforcementMode = to.Ptr(armpolicy.EnforcementModeDefault)
			}
		}

		// set non-compliance message
		if len(src.NonComplianceMessage) > 0 {
			dst.Properties.NonComplianceMessages = make([]*armpolicy.NonComplianceMessage, len(src.NonComplianceMessage))
			for i, msg := range src.NonComplianceMessage {
				dst.Properties.NonComplianceMessages[i] = &armpolicy.NonComplianceMessage{
					Message: to.Ptr(msg.Message.ValueString()),
				}
				if isKnown(msg.PolicyDefinitionReferenceId) {
					dst.Properties.NonComplianceMessages[i].PolicyDefinitionReferenceID = to.Ptr(msg.PolicyDefinitionReferenceId.ValueString())
				}
			}
		}

		// set parameters
		if isKnown(src.Parameters) {
			params := make(map[string]any)
			if err := json.Unmarshal([]byte(src.Parameters.ValueString()), &params); err != nil {
				return nil, fmt.Errorf("unable to unmarshal policy parameters for policy %s: %w", name, err)
			}
			dst.Properties.Parameters = convertPolicyAssignmentParametersToSdkType(params)
		}
		res[name] = dst
	}
	return res, nil
}

func convertPolicyAssignmentParametersToSdkType(src map[string]any) map[string]*armpolicy.ParameterValuesValue {
	res := make(map[string]*armpolicy.ParameterValuesValue, len(src))
	for k, v := range src {
		val := new(armpolicy.ParameterValuesValue)
		val.Value = v
		res[k] = val
	}
	return res
}

func isKnown(val attr.Value) bool {
	return !val.IsNull() && !val.IsUnknown()
}
