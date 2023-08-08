// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &ArchetypeKeysDataSource{}

func NewArchetypeKeysDataSource() datasource.DataSource {
	return &ArchetypeKeysDataSource{}
}

// ArchetypeKeysDataSource defines the data source implementation.
type ArchetypeKeysDataSource struct {
	alz *alzlibWithMutex
}

// ArchetypeKeysDataSourceModel describes the data source data model.
type ArchetypeKeysDataSourceModel struct {
	Id                           types.String `tfsdk:"id"`                               // string
	AlzPolicyAssignmentKeys      types.Set    `tfsdk:"alz_policy_assignment_keys"`       // set of string
	AlzPolicyDefinitionKeys      types.Set    `tfsdk:"alz_policy_definition_keys"`       // set of string
	AlzPolicySetDefinitionKeys   types.Set    `tfsdk:"alz_policy_set_definition_keys"`   // set of string
	AlzRoleDefinitionKeys        types.Set    `tfsdk:"alz_role_definition_keys"`         // set of string
	PolicyAssignmentsToAdd       types.Set    `tfsdk:"policy_assignments_to_add"`        // set of string
	PolicyAssignmentsToRemove    types.Set    `tfsdk:"policy_assignments_to_remove"`     // set of string
	PolicyDefinitionsToAdd       types.Set    `tfsdk:"policy_definitions_to_add"`        // set of string
	PolicyDefinitionsToRemove    types.Set    `tfsdk:"policy_definitions_to_remove"`     // set of string
	PolicySetDefinitionsToAdd    types.Set    `tfsdk:"policy_set_definitions_to_add"`    // set of string
	PolicySetDefinitionsToRemove types.Set    `tfsdk:"policy_set_definitions_to_remove"` // set of string
	RoleDefinitionsToAdd         types.Set    `tfsdk:"role_definitions_to_add"`          // set of string
	RoleDefinitionsToRemove      types.Set    `tfsdk:"role_definitions_to_remove"`       // set of string
	BaseArchetype                types.String `tfsdk:"base_archetype"`                   // string
}

func (d *ArchetypeKeysDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_archetype_keys"
}

func (d *ArchetypeKeysDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "Archetype keys data source. Used to generate used in `for_each` loops for Terraform resources, without a dependency on any data that is only known after apply.",

		Attributes: map[string]schema.Attribute{
			"base_archetype": schema.StringAttribute{
				MarkdownDescription: "The base archetype name to use. This has been generated from the provider lib directories.",
				Required:            true,
			},

			"id": schema.StringAttribute{
				MarkdownDescription: "A an id used for acceptance testing.",
				Computed:            true,
			},

			"alz_policy_assignment_keys": schema.SetAttribute{
				MarkdownDescription: "A set of policy assignment names belonging to the archetype.",
				Computed:            true,
				ElementType:         types.StringType,
			},

			"alz_policy_definition_keys": schema.SetAttribute{
				MarkdownDescription: "A set of policy definition names belonging to the archetype.",
				Computed:            true,
				ElementType:         types.StringType,
			},

			"alz_policy_set_definition_keys": schema.SetAttribute{
				MarkdownDescription: "A set of policy set definition names belonging to the archetype.",
				Computed:            true,
				ElementType:         types.StringType,
			},

			"alz_role_definition_keys": schema.SetAttribute{
				MarkdownDescription: "A set of role definition names belonging to the archetype.",
				Computed:            true,
				ElementType:         types.StringType,
			},

			"role_definitions_to_add": schema.SetAttribute{
				MarkdownDescription: "A list of role definition names to add to the archetype.",
				Optional:            true,
				ElementType:         types.StringType,
			},

			"role_definitions_to_remove": schema.SetAttribute{
				MarkdownDescription: "A list of role definition names to remove from the archetype.",
				Optional:            true,
				ElementType:         types.StringType,
			},

			"policy_assignments_to_add": schema.SetAttribute{
				MarkdownDescription: "A list of policy assignment names to add to the archetype.",
				Optional:            true,
				ElementType:         types.StringType,
			},

			"policy_assignments_to_remove": schema.SetAttribute{
				MarkdownDescription: "A list of policy assignment names to remove from the archetype.",
				Optional:            true,
				ElementType:         types.StringType,
			},

			"policy_definitions_to_add": schema.SetAttribute{
				MarkdownDescription: "A list of policy definition names to add to the archetype.",
				Optional:            true,
				ElementType:         types.StringType,
			},

			"policy_definitions_to_remove": schema.SetAttribute{
				MarkdownDescription: "A list of policy definition names to remove from the archetype.",
				Optional:            true,
				ElementType:         types.StringType,
			},

			"policy_set_definitions_to_add": schema.SetAttribute{
				MarkdownDescription: "A list of policy set definition names to add to the archetype.",
				Optional:            true,
				ElementType:         types.StringType,
			},

			"policy_set_definitions_to_remove": schema.SetAttribute{
				MarkdownDescription: "A list of policy set definition names to remove from the archetype.",
				Optional:            true,
				ElementType:         types.StringType,
			},
		},
	}
}

func (d *ArchetypeKeysDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *ArchetypeKeysDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data ArchetypeKeysDataSourceModel

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

	if diags := resp.State.SetAttribute(ctx, path.Root("id"), data.BaseArchetype.ValueString()); diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	// Make a copy of the archetype so we can customize it.
	arch, err := d.alz.CopyArchetype(data.BaseArchetype.ValueString(), nil)
	if err != nil {
		resp.Diagnostics.AddError("Archetype not found", fmt.Sprintf("Unable to find archetype %s", data.BaseArchetype.ValueString()))
		return
	}

	// Add/remove policy definiitons from archetype before adding the management group.
	if err := addAttrStringElementsToSet(arch.PolicyDefinitions, data.PolicyDefinitionsToAdd.Elements()); err != nil {
		resp.Diagnostics.AddError("Unable to add policy definitions", err.Error())
		return
	}
	if err := deleteAttrStringElementsFromSet(arch.PolicyDefinitions, data.PolicyDefinitionsToRemove.Elements()); err != nil {
		resp.Diagnostics.AddError("Unable to remove policy definitions", err.Error())
		return
	}

	// Add/remove policy set definiitons from archetype before adding the management group.
	if err := addAttrStringElementsToSet(arch.PolicySetDefinitions, data.PolicySetDefinitionsToAdd.Elements()); err != nil {
		resp.Diagnostics.AddError("Unable to add policy set definitions", err.Error())
		return
	}
	if err := deleteAttrStringElementsFromSet(arch.PolicySetDefinitions, data.PolicySetDefinitionsToRemove.Elements()); err != nil {
		resp.Diagnostics.AddError("Unable to remove policy set definitions", err.Error())
		return
	}

	// Add/remove role definiitons from archetype before adding the management group.
	if err := addAttrStringElementsToSet(arch.RoleDefinitions, data.RoleDefinitionsToAdd.Elements()); err != nil {
		resp.Diagnostics.AddError("Unable to add role definitions", err.Error())
		return
	}
	if err := deleteAttrStringElementsFromSet(arch.RoleDefinitions, data.RoleDefinitionsToRemove.Elements()); err != nil {
		resp.Diagnostics.AddError("Unable to remove role definitions", err.Error())
		return
	}

	// Add/remove policy assignments from archetype before adding the management group.
	if err := addAttrStringElementsToSet(arch.PolicyAssignments, data.PolicyAssignmentsToAdd.Elements()); err != nil {
		resp.Diagnostics.AddError("Unable to remove policy assignments", err.Error())
		return
	}
	if err := deleteAttrStringElementsFromSet(arch.PolicyAssignments, data.PolicyAssignmentsToRemove.Elements()); err != nil {
		resp.Diagnostics.AddError("Unable to remove policy assignments", err.Error())
		return
	}

	if diags := resp.State.SetAttribute(ctx, path.Root("alz_policy_assignment_keys"), arch.PolicyAssignments.ToSlice()); diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}
	if diags := resp.State.SetAttribute(ctx, path.Root("alz_policy_definition_keys"), arch.PolicyDefinitions.ToSlice()); diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}
	if diags := resp.State.SetAttribute(ctx, path.Root("alz_policy_set_definition_keys"), arch.PolicySetDefinitions.ToSlice()); diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}
	if diags := resp.State.SetAttribute(ctx, path.Root("alz_role_definition_keys"), arch.RoleDefinitions.ToSlice()); diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}
	types.SetValueFrom(ctx, types.StringType, arch.PolicyAssignments.ToSlice())

	// //Save data into Terraform state
	// resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
