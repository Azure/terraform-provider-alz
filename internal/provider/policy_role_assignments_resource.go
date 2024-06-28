// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package provider

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/Azure/alzlib/to"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/authorization/armauthorization"
	"github.com/Azure/terraform-provider-alz/internal/provider/gen"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &PolicyRoleAssignmentsResource{}
var _ resource.ResourceWithImportState = &PolicyRoleAssignmentsResource{}
var _ resource.ResourceWithConfigure = &PolicyRoleAssignmentsResource{}

var respErr *azcore.ResponseError

func NewPolicyRoleAssignmentResource() resource.Resource {
	return &PolicyRoleAssignmentsResource{}
}

// PolicyRoleAssignmentsResource defines the resource implementation.
type PolicyRoleAssignmentsResource struct {
	alz *alzProviderData
}

func (r PolicyRoleAssignmentsResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_policy_role_assignments"
}

func (r *PolicyRoleAssignmentsResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = gen.PolicyRoleAssignmentsResourceSchema(ctx)
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
	var data gen.PolicyRoleAssignmentsModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	for k, v := range data.Assignments.Elements() {
		assignment, err := createPolicyRoleAssignment(ctx, r.alz.clients.RoleAssignmentsClient, k, v)
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create role assignment, got error: %s", err))
			return
		}
		data.Assignments[k] = *assignment
	}

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *PolicyRoleAssignmentsResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data PolicyRoleAssignmentsResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	for k, v := range data.Assignments {
		tflog.Info(ctx, fmt.Sprintf("reading role assignment: %s", v.ResourceID.ValueString()))
		if v.ResourceID.IsNull() || v.RoleDefinitionID.IsUnknown() {
			continue
		}
		assignment, err := readPolicyRoleAssignment(ctx, r.alz.clients.RoleAssignmentsClient, v.ResourceID.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read role assignment, got error: %s", err))
			return
		}
		data.Assignments[k] = *assignment
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *PolicyRoleAssignmentsResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var planned, current PolicyRoleAssignmentsResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &planned)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &current)...)
	if resp.Diagnostics.HasError() {
		return
	}

	for k, v := range planned.Assignments {
		// If the assignment is planned to be created, create it
		asis, ok := current.Assignments[k]
		if !ok {
			tflog.Info(ctx, fmt.Sprintf("creating role assignment %s at scope %s", k, v.Scope.ValueString()))
			assignment, err := createPolicyRoleAssignment(ctx, r.alz.clients.RoleAssignmentsClient, k, v)
			if err != nil {
				resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create role assignment, got error: %s", err))
				return
			}
			planned.Assignments[k] = *assignment
		}
		if ok {
			// This shouldn't happen as the map key is deterministic and based on these values, however, if it does, update the assignment.
			if asis.PrincipalId != v.PrincipalId || asis.RoleDefinitionID != v.RoleDefinitionID || asis.Scope != v.Scope {
				tflog.Info(ctx, fmt.Sprintf("updating role assignment: %s at scope %s", k, v.Scope.ValueString()))
				assignment, err := createPolicyRoleAssignment(ctx, r.alz.clients.RoleAssignmentsClient, k, v)
				if err != nil {
					resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create role assignment, got error: %s", err))
					return
				}
				planned.Assignments[k] = *assignment
			} else {
				// Ok, then just read it
				tflog.Info(ctx, fmt.Sprintf("reading role assignment: %s", asis.ResourceID.ValueString()))
				assignment, err := readPolicyRoleAssignment(ctx, r.alz.clients.RoleAssignmentsClient, asis.ResourceID.ValueString())
				if err != nil {
					resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read role assignment, got error: %s", err))
					return
				}
				planned.Assignments[k] = *assignment
			}
		}

	}

	// If the assignment is planned to be deleted, delete it
	for k, v := range current.Assignments {
		if _, ok := planned.Assignments[k]; !ok {
			tflog.Info(ctx, fmt.Sprintf("deleting role assignment: %s (%s at scope %s)", v.ResourceID.ValueString(), k, v.Scope.ValueString()))
			if err := deletePolicyRoleAssignment(ctx, r.alz.clients.RoleAssignmentsClient, v.ResourceID.ValueString()); err != nil {
				resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete role assignment, got error: %s", err))
				return
			}
		}
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &planned)...)
}

func (r *PolicyRoleAssignmentsResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data PolicyRoleAssignmentsResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	for k, v := range data.Assignments {
		tflog.Info(ctx, fmt.Sprintf("deleting role assignment: %s", v.ResourceID.ValueString()))
		if err := deletePolicyRoleAssignment(ctx, r.alz.clients.RoleAssignmentsClient, v.ResourceID.ValueString()); err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete role assignment, got error: %s", err))
		}
		delete(data.Assignments, k)
	}

	data.Id = types.StringNull()
	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *PolicyRoleAssignmentsResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func standardizeRoleAssignmentRoleDefinititionId(id string) string {
	split := strings.Split(id, "/")
	if len(split) == 7 {
		return "/" + strings.Join(split[3:], "/")
	}
	return id
}

func readPolicyRoleAssignment(ctx context.Context, client *armauthorization.RoleAssignmentsClient, resourceId string) (*PolicyRoleAssignmentsAssignmentResourceModel, error) {
	ra, err := client.GetByID(ctx, resourceId, nil)
	if err != nil {
		if errors.As(err, &respErr) {
			e, _ := err.(*azcore.ResponseError)
			if e.StatusCode != 404 {
				return nil, err
			}
			assignment := PolicyRoleAssignmentsAssignmentResourceModel{
				PrincipalId:      types.StringNull(),
				RoleDefinitionID: types.StringNull(),
				Scope:            types.StringNull(),
				ResourceID:       types.StringNull(),
			}
			return &assignment, nil
		}
	}
	assignment := PolicyRoleAssignmentsAssignmentResourceModel{
		PrincipalId:      types.StringValue(*ra.Properties.PrincipalID),
		RoleDefinitionID: types.StringValue(standardizeRoleAssignmentRoleDefinititionId(*ra.Properties.RoleDefinitionID)),
		Scope:            types.StringValue(*ra.Properties.Scope),
		ResourceID:       types.StringValue(*ra.ID),
	}

	return &assignment, nil
}

func createPolicyRoleAssignment(ctx context.Context, client *armauthorization.RoleAssignmentsClient, id string, data *gen.AssignmentsValue) error {
	params := armauthorization.RoleAssignmentCreateParameters{
		Properties: &armauthorization.RoleAssignmentProperties{
			PrincipalID:      to.Ptr(data.PrincipalId.ValueString()),
			RoleDefinitionID: to.Ptr(data.RoleDefinitionId.ValueString()),
		},
	}
	ra, err := client.Create(ctx, data.Scope.ValueString(), id, params, nil)
	if err != nil {
		return fmt.Errorf("createPolicyRoleAssignment: unable to create role assignment, got error: %w", err)
	}

	data.PrincipalId = types.StringValue(*ra.Properties.PrincipalID)
	data.RoleDefinitionId = types.StringValue(standardizeRoleAssignmentRoleDefinititionId(*ra.Properties.RoleDefinitionID))
	data.Scope = types.StringValue(*ra.Properties.Scope)
	data.ResourceId = types.StringValue(*ra.ID)

	return nil
}

func deletePolicyRoleAssignment(ctx context.Context, client *armauthorization.RoleAssignmentsClient, resourceId string) error {
	_, err := client.DeleteByID(ctx, resourceId, nil)
	if err != nil {
		if errors.As(err, &respErr) {
			e, _ := err.(*azcore.ResponseError)
			if e.StatusCode != 404 {
				return err
			}
		}
	}
	return nil
}
