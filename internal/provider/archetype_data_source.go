// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/authorization/armauthorization"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armpolicy"
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
	"github.com/matt-FFFFFF/alzlib"
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
	alz *alzlib.AlzLib
}

type armTypes interface {
	*armpolicy.Assignment | *armpolicy.Definition | *armpolicy.SetDefinition | *armauthorization.RoleAssignment | *armauthorization.RoleDefinition
}

// ArchetypeDataSourceModel describes the data source data model.
type ArchetypeDataSourceModel struct {
	AlzPolicyAssignments         types.Map                        `tfsdk:"alz_policy_assignments"`     // map of string
	AlzPolicyDefinitions         types.Map                        `tfsdk:"alz_policy_definitions"`     // map of string
	AlzPolicySetDefinitions      types.Map                        `tfsdk:"alz_policy_set_definitions"` // map of string
	AlzRoleAssignments           types.Map                        `tfsdk:"alz_role_assignments"`       // map of string
	AlzRoleDefinitions           types.Map                        `tfsdk:"alz_role_definitions"`       // map of string
	BaseArchetype                types.String                     `tfsdk:"base_archetype"`
	Defaults                     ArchetypeDataSourceModelDefaults `tfsdk:"defaults"`
	DisplayName                  types.String                     `tfsdk:"display_name"`
	Id                           types.String                     `tfsdk:"id"`
	Name                         types.String                     `tfsdk:"name"`
	ParentId                     types.String                     `tfsdk:"parent_id"`
	PolicyAssignmentsToAdd       types.Map                        `tfsdk:"policy_assignments_to_add"`        // map of PolicyAssignmentType
	PolicyAssignmentsToRemove    types.Set                        `tfsdk:"policy_assignments_to_remove"`     // set of string
	PolicyDefinitionsToAdd       types.Set                        `tfsdk:"policy_definitions_to_add"`        // set of string
	PolicyDefinitionsToRemove    types.Set                        `tfsdk:"policy_definitions_to_remove"`     // set of string
	PolicySetDefinitionsToAdd    types.Set                        `tfsdk:"policy_set_definitions_to_add"`    // set of string
	PolicySetDefinitionsToRemove types.Set                        `tfsdk:"policy_set_definitions_to_remove"` // set of string
	RoleAssignmentsToAdd         types.Map                        `tfsdk:"role_assignments_to_add"`          // map of RoleAssignmentType
	RoleDefinitionsToAdd         types.Set                        `tfsdk:"role_definitions_to_add"`          // set of string
	RoleDefinitionsToRemove      types.Set                        `tfsdk:"role_definitions_to_remove"`       // set of string
	SubscriptionIds              types.Set                        `tfsdk:"subscription_ids"`                 // set of string
}

type ArchetypeDataSourceModelDefaults struct {
	DefaultLocation      types.String `tfsdk:"location"`
	DefaultLaWorkspaceId types.String `tfsdk:"log_analytics_workspace_id"`
}

type PolicyAssignmentType struct {
	DisplayName          types.String                  `tfsdk:"display_name"`
	PolicyDefinitionName types.String                  `tfsdk:"policy_definition_name"`
	PolicyDefinitionId   types.String                  `tfsdk:"policy_definition_id"`
	EnforcementMode      types.String                  `tfsdk:"enforcement_mode"`
	Identity             types.String                  `tfsdk:"identity"`
	IdentityIds          types.Set                     `tfsdk:"identity_ids"`           // set of string
	NonComplianceMessage types.Set                     `tfsdk:"non_compliance_message"` // set of PolicyAssignmentNonComplianceMessage
	Parameters           alztypes.PolicyParameterValue `tfsdk:"parameters"`
}

type PolicyAssignmentNonComplianceMessage struct {
	Message                     types.String `tfsdk:"message"`
	PolicyDefinitionReferenceId types.String `tfsdk:"policy_definition_reference_id"`
}

type RoleAssignmentType struct {
	Definition types.String `tfsdk:"definition"`
	ObjectId   types.String `tfsdk:"object_id"`
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
				MarkdownDescription: "Internal id attribute required for acceptance testing. See [here](https://developer.hashicorp.com/terraform/plugin/framework/acctests#implement-id-attribute).",
				Computed:            true,
			},

			"name": schema.StringAttribute{
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
				MarkdownDescription: "A map of policy assignments names to add to the archetype. The map key is the policy assignemnt name.",
				Optional:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"display_name": schema.StringAttribute{
							MarkdownDescription: "The policy assignment display name",
							Required:            true,
						},

						"policy_definition_name": schema.StringAttribute{
							MarkdownDescription: "The name of the policy definition. Must be in the AlzLib, if it is not use `policy_definition_id` instead. Conflicts with `policy_definition_id`.",
							Optional:            true,
							Validators: []validator.String{
								stringvalidator.ConflictsWith(path.MatchRelative().AtMapKey("policy_definition_id")),
							},
						},

						"policy_definition_id": schema.StringAttribute{
							MarkdownDescription: "The resource id of the policy definition. Conflicts with `policy_definition_name`.",
							Optional:            true,
							Validators: []validator.String{
								stringvalidator.ConflictsWith(path.MatchRelative().AtMapKey("policy_definition_id")),
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

						"identity_ids": schema.ListAttribute{
							MarkdownDescription: "A list of identity ids to assign to the policy assignment. Required if `identity` is `UserAssigned`.",
							Optional:            true,
							ElementType:         types.StringType,
							Validators: []validator.List{
								listvalidator.UniqueValues(),
								listvalidator.ValueStringsAre(
									alzvalidators.ArmTypeResourceId("Microsoft.ManagedIdentity", "userAssignedIdentities"),
									stringvalidator.AlsoRequires(path.MatchRelative().AtMapKey("identity")),
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
					Attributes: map[string]schema.Attribute{
						"definition": schema.StringAttribute{
							MarkdownDescription: "The role definition name, or resource id.",
							Required:            true,
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
		},
	}
}

func (d *ArchetypeDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	data, ok := req.ProviderData.(*alzlib.AlzLib)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *alzlib.AlzLib, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	d.alz = data
}

func (d *ArchetypeDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data ArchetypeDataSourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	mgname := data.Name.ValueString()

	// Set the id to the name
	data.Id = data.Name

	// Get the archetype
	arch, ok := d.alz.Archetypes[data.BaseArchetype.ValueString()]
	if !ok {
		resp.Diagnostics.AddError("Archetype not found", fmt.Sprintf("Unable to find archetype %s", data.BaseArchetype.ValueString()))
		return
	}

	// Set well known policy values
	wkpv := new(alzlib.WellKnownPolicyValues)
	defloc := data.Defaults.DefaultLocation.ValueString()
	if defloc == "" {
		resp.Diagnostics.AddError("Default location not set", "Unable to find default location in the archetype attributes. This should have been caught by the schema validation.")
	}
	wkpv.DefaultLocation = defloc
	wkpv.DefaultLogAnalyticsWorkspaceId = data.Defaults.DefaultLaWorkspaceId.ValueString()

	// TODO: change this to compare the config to the AlzManagementGroup in the alz struct.
	// It should be identical. If it is not then error as the user has duplicate management group names.
	if _, ok := d.alz.Deployment.MGs[mgname]; !ok {
		tflog.Debug(ctx, "Add management group")
		external := false
		parent := data.ParentId.ValueString()
		if _, ok := d.alz.Deployment.MGs[parent]; !ok {
			external = true
		}

		if err := d.alz.Deployment.AddManagementGroup(mgname, data.DisplayName.ValueString(), data.ParentId.ValueString(), external, arch.WithWellKnownPolicyValues(wkpv)); err != nil {
			resp.Diagnostics.AddError("Unable to add management group", err.Error())
			return
		}
	}

	if err := d.alz.Deployment.MGs[mgname].GeneratePolicyAssignmentAdditionalRoleAssignments(d.alz); err != nil {
		resp.Diagnostics.AddError("Unable to generate additional role assignments", err.Error())
		return
	}

	tflog.Debug(ctx, "Converting maps from Go types to Framework types")
	var m basetypes.MapValue
	var diags diag.Diagnostics

	tflog.Debug(ctx, "Converting policy assignments")
	m, diags = marshallMap(d.alz.Deployment.MGs[mgname].PolicyAssignments)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.AlzPolicyAssignments = m

	tflog.Debug(ctx, "Converting policy definitions")
	m, diags = marshallMap(d.alz.Deployment.MGs[mgname].PolicyDefinitions)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.AlzPolicyDefinitions = m

	tflog.Debug(ctx, "Converting policy set definitions")
	m, diags = marshallMap(d.alz.Deployment.MGs[mgname].PolicySetDefinitions)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.AlzPolicySetDefinitions = m

	tflog.Debug(ctx, "Converting role assignments")
	m, diags = marshallMap(d.alz.Deployment.MGs[mgname].RoleAssignments)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.AlzRoleAssignments = m

	tflog.Debug(ctx, "Converting role definitions")
	m, diags = marshallMap(d.alz.Deployment.MGs[mgname].RoleDefinitions)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.AlzRoleDefinitions = m

	//Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func marshallMap[T armTypes](m map[string]T) (basetypes.MapValue, diag.Diagnostics) {
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
