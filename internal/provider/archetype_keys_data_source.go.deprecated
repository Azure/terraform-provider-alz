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
	alz *alzProviderData
}

// ArchetypeKeysDataSourceModel describes the data source data model.
type ArchetypeKeysDataSourceModel struct {
	Id                         types.String `tfsdk:"id"`                             // string
	BaseArchetype              types.String `tfsdk:"base_archetype"`                 // string
	AlzPolicyAssignmentKeys    types.Set    `tfsdk:"alz_policy_assignment_keys"`     // set of string
	AlzPolicyDefinitionKeys    types.Set    `tfsdk:"alz_policy_definition_keys"`     // set of string
	AlzPolicySetDefinitionKeys types.Set    `tfsdk:"alz_policy_set_definition_keys"` // set of string
	AlzRoleDefinitionKeys      types.Set    `tfsdk:"alz_role_definition_keys"`       // set of string
}

func (d *ArchetypeKeysDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_archetype_keys"
}

func (d *ArchetypeKeysDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "Archetype keys data source. Produces sets of strings to be used in `for_each` loops for Terraform resources, without a dependency on any data that is only known after apply." +
			"The values are the keys to the data maps produced by the `alz_archetype` resource. You can use this to create a local map, combining the keys with the data from the `alz_archetype` resource.",

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
		},
	}
}

func (d *ArchetypeKeysDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

	// Make a copy of the archetype.
	arch, err := d.alz.CopyArchetype(data.BaseArchetype.ValueString(), nil)
	if err != nil {
		resp.Diagnostics.AddError("Archetype not found", fmt.Sprintf("Unable to find archetype %s", data.BaseArchetype.ValueString()))
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
}
