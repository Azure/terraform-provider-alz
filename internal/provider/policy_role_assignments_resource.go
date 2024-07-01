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
	"github.com/google/uuid"
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
	newAssignments := make([]gen.AssignmentsValue, len(data.Assignments.Elements()))
	for i, v := range data.Assignments.Elements() {
		pra, ok := v.(gen.AssignmentsValue)
		if !ok {
			resp.Diagnostics.AddError("Schema Error", "Unable to cast attr.Value to PolicyRoleAssignmentsValue")
			return
		}
		name := genPolicyRoleAssignmentId(pra)
		err := createPolicyRoleAssignment(ctx, r.alz.clients.RoleAssignmentsClient, name, &pra)
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create role assignment, got error: %s", err))
			return
		}
		newAssignments[i] = pra
	}

	newAssignmentsSet, diags := types.SetValueFrom(ctx, gen.NewAssignmentsValueNull().Type(ctx), newAssignments)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.Assignments = newAssignmentsSet

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *PolicyRoleAssignmentsResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data gen.PolicyRoleAssignmentsModel
	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	newAssignments := make([]gen.AssignmentsValue, 0, len(data.Assignments.Elements()))
	for _, v := range data.Assignments.Elements() {
		pra, ok := v.(gen.AssignmentsValue)
		if !ok {
			resp.Diagnostics.AddError("Schema Error", "Unable to cast attr.Value to PolicyRoleAssignmentsValue")
			return
		}
		tflog.Debug(ctx, fmt.Sprintf("reading role assignment: %s", pra.ResourceId.ValueString()))
		if pra.ResourceId.IsNull() || pra.RoleDefinitionId.IsUnknown() {
			continue
		}
		assignment, err := readPolicyRoleAssignment(ctx, r.alz.clients.RoleAssignmentsClient, pra.ResourceId.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read role assignment, got error: %s", err))
			return
		}
		newAssignments = append(newAssignments, *assignment)
	}
	newAssignmentsSet, diags := types.SetValueFrom(ctx, gen.NewAssignmentsValueNull().Type(ctx), newAssignments)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.Assignments = newAssignmentsSet
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *PolicyRoleAssignmentsResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var planned, current gen.PolicyRoleAssignmentsModel
	var plannedAssignments, currentAssignments []gen.AssignmentsValue

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &planned)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &current)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(planned.Assignments.ElementsAs(ctx, &plannedAssignments, false)...)
	resp.Diagnostics.Append(current.Assignments.ElementsAs(ctx, &currentAssignments, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	newAssignments := make([]gen.AssignmentsValue, 0, len(plannedAssignments))
	for _, v := range plannedAssignments {
		// If the assignment is already in state (comparison by scope, role def id and principal id), read it
		if pra := policyRoleAssignmentFromSlice(currentAssignments, v); pra != nil {
			// Ok, then just read it
			tflog.Debug(ctx, fmt.Sprintf("reading role assignment: %s", pra.ResourceId.ValueString()))
			assignment, err := readPolicyRoleAssignment(ctx, r.alz.clients.RoleAssignmentsClient, pra.ResourceId.ValueString())
			if err != nil {
				resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read role assignment, got error: %s", err))
				return
			}
			newAssignments = append(newAssignments, *assignment)
		}
		// If not then we create it
		name := genPolicyRoleAssignmentId(v)
		tflog.Debug(ctx, fmt.Sprintf("creating role assignment %s at scope %s", name, v.Scope.ValueString()))
		err := createPolicyRoleAssignment(ctx, r.alz.clients.RoleAssignmentsClient, name, &v)
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create role assignment, got error: %s", err))
			return
		}
		newAssignments = append(newAssignments, v)
	}

	// If the assignment is planned to be deleted, delete it
	for _, v := range currentAssignments {
		if policyRoleAssignmentFromSlice(plannedAssignments, v) != nil {
			continue
		}
		tflog.Debug(ctx, fmt.Sprintf("deleting role assignment: %s", v.ResourceId.ValueString()))
		if err := deletePolicyRoleAssignment(ctx, r.alz.clients.RoleAssignmentsClient, v.ResourceId.ValueString()); err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete role assignment, got error: %s", err))
			return
		}
	}
	newAssignmentsSet, diags := types.SetValueFrom(ctx, gen.NewAssignmentsValueNull().Type(ctx), newAssignments)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	planned.Assignments = newAssignmentsSet
	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &planned)...)
}

func (r *PolicyRoleAssignmentsResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data gen.PolicyRoleAssignmentsModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	for _, v := range data.Assignments.Elements() {
		pra, ok := v.(gen.AssignmentsValue)
		if !ok {
			resp.Diagnostics.AddError("Schema Error", "Unable to cast attr.Value to PolicyRoleAssignmentsValue")
			return
		}
		tflog.Debug(ctx, fmt.Sprintf("deleting role assignment: %s", pra.ResourceId.ValueString()))
		if err := deletePolicyRoleAssignment(ctx, r.alz.clients.RoleAssignmentsClient, pra.ResourceId.ValueString()); err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete role assignment, got error: %s", err))
		}
	}

	data.Id = types.StringNull()
	data.Assignments = types.SetNull(gen.NewAssignmentsValueNull().Type(ctx))
	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *PolicyRoleAssignmentsResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func policyRoleAssignmentFromSlice(s []gen.AssignmentsValue, want gen.AssignmentsValue) *gen.AssignmentsValue {
	for _, v := range s {
		if v.PrincipalId == want.PrincipalId && v.RoleDefinitionId == want.RoleDefinitionId && v.Scope == want.Scope {
			return &v
		}
	}
	return nil
}

func standardizeRoleAssignmentRoleDefinititionId(id string) string {
	split := strings.Split(id, "/")
	if len(split) == 7 {
		return "/" + strings.Join(split[3:], "/")
	}
	return id
}

func readPolicyRoleAssignment(ctx context.Context, client *armauthorization.RoleAssignmentsClient, resourceId string) (*gen.AssignmentsValue, error) {
	ra, err := client.GetByID(ctx, resourceId, nil)
	if err != nil {
		if errors.As(err, &respErr) {
			e, _ := err.(*azcore.ResponseError)
			if e.StatusCode != 404 {
				return nil, err
			}
			assignment := gen.AssignmentsValue{
				PrincipalId:      types.StringNull(),
				RoleDefinitionId: types.StringNull(),
				Scope:            types.StringNull(),
				ResourceId:       types.StringNull(),
			}
			return &assignment, nil
		}
	}
	assignment := gen.AssignmentsValue{
		PrincipalId:      types.StringValue(*ra.Properties.PrincipalID),
		RoleDefinitionId: types.StringValue(standardizeRoleAssignmentRoleDefinititionId(*ra.Properties.RoleDefinitionID)),
		Scope:            types.StringValue(*ra.Properties.Scope),
		ResourceId:       types.StringValue(*ra.ID),
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

func genPolicyRoleAssignmentId(pra gen.AssignmentsValue) string {
	u := uuid.NewSHA1(uuid.NameSpaceURL, []byte(pra.PrincipalId.ValueString()+pra.Scope.ValueString()+pra.RoleDefinitionId.ValueString()))
	return u.String()
}
