// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &PolicyRoleAssignmentResource{}
var _ resource.ResourceWithImportState = &PolicyRoleAssignmentResource{}

func NewPolicyRoleAssignmentResource() resource.Resource {
	return &PolicyRoleAssignmentResource{}
}

// PolicyRoleAssignmentResource defines the resource implementation.
type PolicyRoleAssignmentResource struct {
	alz *alzlibWithMutex
}

// PolicyRoleAssignmentResourceModel describes the resource data model.
type PolicyRoleAssignmentResourceModel struct {
	Id          types.String `tfsdk:"id"`
	Assignments types.Set    `tfsdk:"assignments"`
}

// PolicyRoleAssignmentGoResourceModel describes the resource data model.
type PolicyRoleAssignmentGoResourceModel struct {
	Id          string
	Assignments []PolicyRoleAssignmentGoAssignmentResourceModel
}

type PolicyRoleAssignmentGoAssignmentResourceModel struct {
	AssignmentName   string
	Scope            string
	RoleDefinitionID string
	ResourceID       string
}

func (r PolicyRoleAssignmentResourceModel) ToGoType(ctx context.Context) (PolicyRoleAssignmentGoResourceModel, diag.Diagnostics) {
	rtn := PolicyRoleAssignmentGoResourceModel{}
	rtn.Id = r.Id.ValueString()
	rtn.Assignments = make([]PolicyRoleAssignmentGoAssignmentResourceModel, 0)
	diags := r.Assignments.ElementsAs(ctx, rtn.Assignments, false)
	return rtn, diags
}

func (r PolicyRoleAssignmentResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_policy_role_assignment"
}

func (r *PolicyRoleAssignmentResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "Policy role assignment resource",

		Attributes: map[string]schema.Attribute{
			"assignments": schema.SetNestedAttribute{
				Required: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"assignment_name": schema.StringAttribute{
							Required:            true,
							MarkdownDescription: "The name of the policy assignment.",
						},
						"scope": schema.StringAttribute{
							Required:            true,
							MarkdownDescription: "The scope of the policy assignment.",
						},
						"role_definition_id": schema.StringAttribute{
							Required:            true,
							MarkdownDescription: "The role definition ID of the policy assignment.",
						},
						"resource_id": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "The resource ID of the role assignment.",
						},
					},
				},
			},
			"id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The id of the management group, forming the last part of the resource ID.",
			},
		},
	}
}

func (r *PolicyRoleAssignmentResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

	r.alz = data
}

func (r *PolicyRoleAssignmentResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data PolicyRoleAssignmentResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// If applicable, this is a great opportunity to initialize any necessary
	// provider client data and make a call using it.
	// httpResp, err := r.client.Do(httpReq)
	// if err != nil {
	//     resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create example, got error: %s", err))
	//     return
	// }

	// For the purposes of this example code, hardcoding a response value to
	// save into the Terraform state.
	//data.Id = types.StringValue("example-id")

	// Write logs using the tflog package
	// Documentation: https://terraform.io/plugin/log
	tflog.Trace(ctx, "created a resource")

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *PolicyRoleAssignmentResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data PolicyRoleAssignmentResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// If applicable, this is a great opportunity to initialize any necessary
	// provider client data and make a call using it.
	// httpResp, err := r.client.Do(httpReq)
	// if err != nil {
	//     resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read example, got error: %s", err))
	//     return
	// }

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *PolicyRoleAssignmentResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data PolicyRoleAssignmentResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// If applicable, this is a great opportunity to initialize any necessary
	// provider client data and make a call using it.
	// httpResp, err := r.client.Do(httpReq)
	// if err != nil {
	//     resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update example, got error: %s", err))
	//     return
	// }

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *PolicyRoleAssignmentResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data PolicyRoleAssignmentResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// If applicable, this is a great opportunity to initialize any necessary
	// provider client data and make a call using it.
	// httpResp, err := r.client.Do(httpReq)
	// if err != nil {
	//     resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete example, got error: %s", err))
	//     return
	// }
}

func (r *PolicyRoleAssignmentResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
