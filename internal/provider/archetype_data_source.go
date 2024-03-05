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
	mapset "github.com/deckarep/golang-set/v2"
	"github.com/google/uuid"
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
	PolicyAssignmentsToModify map[string]PolicyAssignmentType        `tfsdk:"policy_assignments_to_modify"` // map of PolicyAssignmentType
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
}

// PolicyAssignmentNonComplianceMessage describes non-compliance message in a policy assignment.
type PolicyAssignmentNonComplianceMessage struct {
	Message                     types.String `tfsdk:"message"`
	PolicyDefinitionReferenceId types.String `tfsdk:"policy_definition_reference_id"`
}

// type RoleAssignmentType struct {
// 	DefinitionName types.String `tfsdk:"definition_name"`
// 	DefinitionId   types.String `tfsdk:"definition_id"`
// 	ObjectId       types.String `tfsdk:"object_id"`
// }

func (d *ArchetypeDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_archetype"
}

func (d *ArchetypeDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
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

						"parameters": schema.StringAttribute{
							MarkdownDescription: "The parameters to use for the policy assignment. " +
								"**Note:** This is a JSON string, and not a map. This is because the parameter values have different types, which confuses the type system used by the provider sdk. " +
								"Use `jsonencode()` to construct the map. " +
								"The map keys must be strings, the values are `any` type.\n\n" +
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
		enf, ident, noncompl, params, err := policyAssignmentType2ArmPolicyValues(v)
		if err != nil {
			resp.Diagnostics.AddError(fmt.Sprintf("Unable to convert supplied policy assignment modifications to SDK values for policy assignment %s", k), err.Error())
			return
		}
		if err := mg.ModifyPolicyAssignment(k, params, enf, noncompl, ident); err != nil {
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
	err error) {
	// Set enforcement mode.
	if isKnown(pa.EnforcementMode) {
		switch pa.EnforcementMode.ValueString() {
		case "DoNotEnforce":
			enforcementMode = to.Ptr(armpolicy.EnforcementModeDoNotEnforce)
		case "Default":
			enforcementMode = to.Ptr(armpolicy.EnforcementModeDefault)
		}
	}

	// set non-compliance message
	if len(pa.NonComplianceMessage) > 0 {
		nonComplianceMessages = make([]*armpolicy.NonComplianceMessage, len(pa.NonComplianceMessage))
		for i, msg := range pa.NonComplianceMessage {
			nonComplianceMessages[i] = &armpolicy.NonComplianceMessage{
				Message: to.Ptr(msg.Message.ValueString()),
			}
			if isKnown(msg.PolicyDefinitionReferenceId) {
				nonComplianceMessages[i].PolicyDefinitionReferenceID = to.Ptr(msg.PolicyDefinitionReferenceId.ValueString())
			}
		}
	}

	// set parameters
	if isKnown(pa.Parameters) {
		params := make(map[string]any)
		if err := json.Unmarshal([]byte(pa.Parameters.ValueString()), &params); err != nil {
			return nil, nil, nil, nil, fmt.Errorf("unable to unmarshal policy parameters: %w", err)
		}
		parameters = convertPolicyAssignmentParametersToSdkType(params)
	}

	return enforcementMode, identity, nonComplianceMessages, parameters, nil
}

// convertPolicyAssignmentParametersToSdkType converts a map[string]any to a map[string]*armpolicy.ParameterValuesValue.
func convertPolicyAssignmentParametersToSdkType(src map[string]any) map[string]*armpolicy.ParameterValuesValue {
	if src == nil {
		return nil
	}
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

func genPolicyRoleAssignmentId(pra alzlib.PolicyRoleAssignment) string {
	u := uuid.NewSHA1(uuid.NameSpaceURL, []byte(pra.AssignmentName+pra.RoleDefinitionId+pra.Scope))
	return u.String()
}
