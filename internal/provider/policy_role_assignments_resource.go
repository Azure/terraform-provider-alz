// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package provider

import (
	"context"
	"errors"
	"fmt"

	"github.com/Azure/alzlib/to"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/authorization/armauthorization"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &PolicyRoleAssignmentsResource{}
var _ resource.ResourceWithImportState = &PolicyRoleAssignmentsResource{}

func NewPolicyRoleAssignmentResource() resource.Resource {
	return &PolicyRoleAssignmentsResource{}
}

// PolicyRoleAssignmentsResource defines the resource implementation.
type PolicyRoleAssignmentsResource struct {
	alz *alzProviderData
}

// PolicyRoleAssignmentsResourceModel describes the resource data model.
type PolicyRoleAssignmentsResourceModel struct {
	Id          types.String                                            `tfsdk:"id"`
	Assignments map[string]PolicyRoleAssignmentsAssignmentResourceModel `tfsdk:"assignments"`
}

// PolicyRoleAssignmentsAssignmentResourceModel describes the resource data model.
type PolicyRoleAssignmentsAssignmentResourceModel struct {
	PrincipalId      types.String `tfsdk:"principal_id"`
	Scope            types.String `tfsdk:"scope"`
	RoleDefinitionID types.String `tfsdk:"role_definition_id"`
	ResourceID       types.String `tfsdk:"resource_id"`
}

// PolicyRoleAssignmentGoResourceModel describes the resource data model.
// type PolicyRoleAssignmentGoResourceModel struct {
// 	Id          string
// 	Assignments []PolicyRoleAssignmentGoAssignmentResourceModel
// }

// PolicyRoleAssignmentGoAssignmentResourceModel describes the go data model.
// type PolicyRoleAssignmentGoAssignmentResourceModel struct {
// 	AssignmentName   *string
// 	Scope            *string
// 	RoleDefinitionID *string
// 	ResourceID       *string
// }

// func (r PolicyRoleAssignmentResourceModel) ToGoType(ctx context.Context) (PolicyRoleAssignmentGoResourceModel, diag.Diagnostics) {
// 	rtn := PolicyRoleAssignmentGoResourceModel{}
// 	rtn.Id = r.Id.ValueString()
// 	rtn.Assignments = make([]PolicyRoleAssignmentGoAssignmentResourceModel, len(r.Assignments))
// 	if len(r.Assignments) == 0 {
// 		return rtn, nil
// 	}
// 	for i, assignment := range r.Assignments {
// 		rtn.Assignments[i] = PolicyRoleAssignmentGoAssignmentResourceModel{
// 			AssignmentName:   *assignment.AssignmentName.ValueStringPointer(),
// 			Scope:            assignment.Scope.ValueStringPointer(),
// 			RoleDefinitionID: assignment.RoleDefinitionID.ValueStringPointer(),
// 			ResourceID:       assignment.ResourceID.ValueStringPointer(),
// 		}
// 		return rtn, diags
// 	}
// }

func (r PolicyRoleAssignmentsResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_policy_role_assignments"
}

func (r *PolicyRoleAssignmentsResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "Policy role assignment resource",

		Attributes: map[string]schema.Attribute{
			"assignments": schema.MapNestedAttribute{
				Required: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"principal_id": schema.StringAttribute{
							Optional:            true,
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

func (r *PolicyRoleAssignmentsResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

	r.alz = data
}

func (r *PolicyRoleAssignmentsResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data PolicyRoleAssignmentsResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	for k, v := range data.Assignments {
		params := armauthorization.RoleAssignmentCreateParameters{
			Properties: &armauthorization.RoleAssignmentProperties{
				PrincipalID:      to.Ptr(v.PrincipalId.ValueString()),
				RoleDefinitionID: to.Ptr(v.RoleDefinitionID.ValueString()),
			},
		}
		tflog.Info(ctx, fmt.Sprintf("creating a role assignment: %s", k))
		r, err := r.alz.clients.RoleAssignmentsClient.Create(ctx, v.Scope.ValueString(), k, params, nil)
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create role assignment, got error: %s", err))
		}
		v.ResourceID = types.StringValue(*r.ID)
	}

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("assignments"), data.Assignments)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *PolicyRoleAssignmentsResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data PolicyRoleAssignmentsResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	var respErr *azcore.ResponseError
	for _, v := range data.Assignments {
		if v.ResourceID.IsNull() || v.RoleDefinitionID.IsUnknown() {
			continue
		}
		ra, err := r.alz.clients.RoleAssignmentsClient.GetByID(ctx, v.ResourceID.ValueString(), nil)
		if err != nil {
			if errors.As(err, &respErr) {
				e, _ := err.(*azcore.ResponseError)
				if e.StatusCode != 404 {
					resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read role assignment, got error: %s", err))
					return
				}
			}
			v.ResourceID = types.StringNull()
			v.RoleDefinitionID = types.StringNull()
			v.Scope = types.StringNull()
			v.PrincipalId = types.StringNull()
		}
		v.RoleDefinitionID = types.StringValue(*ra.Properties.RoleDefinitionID)
		v.Scope = types.StringValue(*ra.Properties.Scope)
		v.PrincipalId = types.StringValue(*ra.Properties.PrincipalID)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *PolicyRoleAssignmentsResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data PolicyRoleAssignmentsResourceModel

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

func (r *PolicyRoleAssignmentsResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data PolicyRoleAssignmentsResourceModel

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

func (r *PolicyRoleAssignmentsResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
