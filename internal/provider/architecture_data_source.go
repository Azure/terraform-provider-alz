package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"time"

	"github.com/Azure/alzlib/deployment"
	"github.com/Azure/alzlib/to"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armpolicy"
	"github.com/Azure/terraform-provider-alz/internal/alztypes"
	"github.com/Azure/terraform-provider-alz/internal/provider/gen"
	"github.com/Azure/terraform-provider-alz/internal/typehelper"
	"github.com/Azure/terraform-provider-alz/internal/typehelper/frameworktype"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

var _ datasource.DataSource = (*architectureDataSource)(nil)
var _ datasource.DataSourceWithConfigure = (*architectureDataSource)(nil)

func NewArchitectureDataSource() datasource.DataSource {
	return &architectureDataSource{}
}

type architectureDataSource struct {
	alz *alzProviderData
}

func (d *architectureDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_architecture"
}

func (d *architectureDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = gen.ArchitectureDataSourceSchema(ctx)
}

func (d *architectureDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	data, ok := req.ProviderData.(*alzProviderData)

	if !ok {
		resp.Diagnostics.AddError(
			"architectureDataSource.Configure() Unexpected type",
			fmt.Sprintf("Expected *alzProviderData, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	d.alz = data
}

func (d *architectureDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data gen.ArchitectureModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	readTimeout, diags := data.Timeouts.Read(ctx, 10*time.Minute)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, readTimeout)
	defer cancel()

	if d.alz == nil {
		resp.Diagnostics.AddError(
			"architectureDataSource.Read() Provider not configured",
			"The provider has not been configured. Please see the provider documentation for configuration instructions.",
		)
		return
	}

	// Use alzlib to create the hierarchy from the supplied architecture
	depl := deployment.NewHierarchy(d.alz.AlzLib)
	if err := depl.FromArchitecture(ctx, data.Name.ValueString(), data.RootManagementGroupId.ValueString(), data.Location.ValueString()); err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("architectureDataSource.Read() Error creating architecture %s", data.Name.ValueString()),
			err.Error(),
		)
		return
	}

	// Modify policy assignments
	mg2paModMap := make(map[string]gen.PolicyAssignmentsToModifyValue)
	diags = data.PolicyAssignmentsToModify.ElementsAs(ctx, &mg2paModMap, false)
	resp.Diagnostics.Append(diags...)
	for mgName, pa2mod := range mg2paModMap {
		pa2modMap := make(map[string]gen.PolicyAssignmentsValue)
		diags = pa2mod.PolicyAssignments.ElementsAs(ctx, &pa2modMap, false)
		resp.Diagnostics.Append(diags...)
		for paName, mod := range pa2modMap {
			enf, ident, noncompl, params, resourceSel, overrides, diags := policyAssignmentType2ArmPolicyValues(ctx, mod)
			resp.Diagnostics.Append(diags...)
			if diags.HasError() {
				resp.Diagnostics.AddError(
					"architectureDataSource.Read() Error converting policy assignment values to Azure SDK types",
					fmt.Sprintf("Error modifying policy assignment values for `%s` at mg `%s`", paName, mgName),
				)
				return
			}
			if err := depl.ManagementGroup(mgName).ModifyPolicyAssignment(paName, params, enf, noncompl, ident, resourceSel, overrides); err != nil {
				resp.Diagnostics.AddError(
					"architectureDataSource.Read() Error modifying policy assignment values in alzlib",
					fmt.Sprintf("Error modifying policy assignment values for `%s` at mg `%s`: %s", paName, mgName, err.Error()),
				)
				return
			}
		}
	}

	// Generate policy role assignments
	policyRoleAssignments, err := depl.PolicyRoleAssignments(ctx)
	if err != nil {
		resp.Diagnostics.AddError(
			"architectureDataSource.Read() Error generating policy role assignments",
			err.Error(),
		)
		return
	}
	policyRoleAssignmentsVal, diags := policyRoleAssignmentsSetToProviderType(ctx, policyRoleAssignments.ToSlice())
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.PolicyRoleAssignments = policyRoleAssignmentsVal

	// Set computed values
	mgNames := depl.ManagementGroupNames()
	mgVals := make([]gen.ManagementGroupsValue, len(mgNames))
	for i, mgName := range mgNames {
		mgVal, diags := alzMgToProviderType(ctx, depl.ManagementGroup(mgName))
		resp.Diagnostics.Append(diags...)
		mgVals[i] = mgVal
	}
	mgs, diags := types.ListValueFrom(ctx, gen.NewManagementGroupsValueNull().Type(ctx), &mgVals)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.ManagementGroups = mgs

	// Set the id to keep ACC tests happy
	data.Id = data.Name

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func policyRoleAssignmentsSetToProviderType(ctx context.Context, input []deployment.PolicyRoleAssignment) (basetypes.SetValue, diag.Diagnostics) {
	var diags diag.Diagnostics
	praSlice := make([]gen.PolicyRoleAssignmentsValue, 0, len(input))
	for _, v := range input {
		pra, diag := policyRoleAssignmentToProviderType(ctx, v)
		diags.Append(diag...)
		praSlice = append(praSlice, pra)
	}
	if diags.HasError() {
		return types.SetNull(gen.NewPolicyRoleAssignmentsValueNull().Type(ctx)), diags
	}
	return types.SetValueFrom(ctx, gen.NewPolicyRoleAssignmentsValueNull().Type(ctx), &praSlice)
}

func policyRoleAssignmentToProviderType(ctx context.Context, input deployment.PolicyRoleAssignment) (gen.PolicyRoleAssignmentsValue, diag.Diagnostics) {
	return gen.NewPolicyRoleAssignmentsValue(
		gen.NewPolicyRoleAssignmentsValueNull().AttributeTypes(ctx),
		map[string]attr.Value{
			"role_definition_id":     types.StringValue(input.RoleDefinitionId),
			"scope":                  types.StringValue(input.Scope),
			"policy_assignment_name": types.StringValue(input.AssignmentName),
			"management_group_id":    types.StringValue(input.ManagementGroupId),
		},
	)
}

func alzMgToProviderType(ctx context.Context, mg *deployment.HierarchyManagementGroup) (gen.ManagementGroupsValue, diag.Diagnostics) {
	var respDiags diag.Diagnostics
	policyAssignments, diags := typehelper.ConvertAlzMapToFrameworkType(mg.PolicyAssignmentMap())
	respDiags.Append(diags...)
	policyDefinitions, diags := typehelper.ConvertAlzMapToFrameworkType(mg.PolicyDefinitionsMap())
	respDiags.Append(diags...)
	policySetDefinitions, diags := typehelper.ConvertAlzMapToFrameworkType(mg.PolicySetDefinitionsMap())
	respDiags.Append(diags...)
	roleDefinitions, diags := typehelper.ConvertAlzMapToFrameworkType(mg.RoleDefinitionsMap())
	respDiags.Append(diags...)
	if respDiags.HasError() {
		return gen.NewManagementGroupsValueNull(), respDiags
	}
	return gen.NewManagementGroupsValue(
		gen.NewManagementGroupsValueNull().AttributeTypes(ctx),
		map[string]attr.Value{
			"id":                     types.StringValue(mg.Name()),
			"parent_id":              types.StringValue(mg.ParentId()),
			"display_name":           types.StringValue(mg.DisplayName()),
			"exists":                 types.BoolValue(mg.Exists()),
			"level":                  types.NumberValue(big.NewFloat(float64(mg.Level()))),
			"policy_assignments":     policyAssignments,
			"policy_definitions":     policyDefinitions,
			"policy_set_definitions": policySetDefinitions,
			"role_definitions":       roleDefinitions,
		},
	)
}

// policyAssignmentType2ArmPolicyValues returns a set of Azure Go SDK values from a PolicyAssignmentType.
// This is used to modify existing policy assignments.
func policyAssignmentType2ArmPolicyValues(ctx context.Context, pa gen.PolicyAssignmentsValue) (
	enforcementMode *armpolicy.EnforcementMode,
	identity *armpolicy.Identity,
	nonComplianceMessages []*armpolicy.NonComplianceMessage,
	parameters map[string]*armpolicy.ParameterValuesValue,
	resourceSelectors []*armpolicy.ResourceSelector,
	overrides []*armpolicy.Override,
	diags diag.Diagnostics) {
	var diag diag.Diagnostics
	// Set enforcement mode.
	enforcementMode = convertPolicyAssignmentEnforcementModeToSdkType(pa.EnforcementMode)

	// set identity
	identity, diag = convertPolicyAssignmentIdentityToSdkType(pa.Identity, pa.IdentityIds)
	diags.Append(diag...)
	if diags.HasError() {
		return nil, nil, nil, nil, nil, nil, diags
	}

	// set non-compliance message
	if isKnown(pa.NonComplianceMessages) {
		nonCompl := make([]gen.NonComplianceMessagesValue, len(pa.NonComplianceMessages.Elements()))
		for i, msg := range pa.NonComplianceMessages.Elements() {
			frameworkMsg, ok := msg.(gen.NonComplianceMessagesValue)
			if !ok {
				diags.AddError(
					"policyAssignmentType2ArmPolicyValues: error",
					"unable to convert non-compliance message attr.Value to concrete type",
				)
			}
			nonCompl[i] = frameworkMsg
		}
		if diags.HasError() {
			return nil, nil, nil, nil, nil, nil, diags
		}
		nonComplianceMessages = convertPolicyAssignmentNonComplianceMessagesToSdkType(nonCompl)
	}

	// set parameters
	params := alztypes.PolicyParameterValue{
		StringValue: types.StringValue(pa.Parameters.ValueString()),
	}
	parameters, diag = convertPolicyAssignmentParametersToSdkType(params)
	diags.Append(diag...)
	if diag.HasError() {
		return nil, nil, nil, nil, nil, nil, diags
	}

	// set resource selectors
	if isKnown(pa.ResourceSelectors) {
		rS := make([]gen.ResourceSelectorsValue, len(pa.ResourceSelectors.Elements()))
		diag = pa.ResourceSelectors.ElementsAs(ctx, &rS, false)
		diags.Append(diag...)
		resourceSelectors, diag = convertPolicyAssignmentResourceSelectorsToSdkType(ctx, rS)
		diags.Append(diag...)
		if diags.HasError() {
			return nil, nil, nil, nil, nil, nil, diags
		}
	}

	// set overrides
	if isKnown(pa.Overrides) {
		ovr := make([]gen.OverridesValue, len(pa.Overrides.Elements()))
		diag = pa.Overrides.ElementsAs(ctx, &ovr, false)
		diags.Append(diag...)
		overrides, diag = convertPolicyAssignmentOverridesToSdkType(ctx, ovr)
		diags.Append(diag...)
		if diags.HasError() {
			return nil, nil, nil, nil, nil, nil, diags
		}
	}

	return enforcementMode, identity, nonComplianceMessages, parameters, resourceSelectors, overrides, nil
}

func convertPolicyAssignmentOverridesToSdkType(ctx context.Context, input []gen.OverridesValue) ([]*armpolicy.Override, diag.Diagnostics) {
	var diags diag.Diagnostics
	if len(input) == 0 {
		return nil, nil
	}
	res := make([]*armpolicy.Override, len(input))
	for i, o := range input {
		selectors := make([]*armpolicy.Selector, len(o.OverrideSelectors.Elements()))
		for j, s := range o.OverrideSelectors.Elements() {
			osv, ok := s.(gen.OverrideSelectorsValue)
			if !ok {
				diags.AddError(
					"convertPolicyAssignmentOverridesToSdkType: error",
					"unable to convert override selectors attr.Value to concrete type",
				)
			}
			in, err := frameworktype.SliceOfPrimitiveToGo[string](ctx, osv.In.Elements())
			if err != nil {
				diags.AddError(
					"convertPolicyAssignmentOverridesToSdkType: error",
					fmt.Sprintf("unable to convert OverrideSelctorsValue.In elements to Go slice: %s", err.Error()),
				)
				return nil, diags
			}
			notIn, err := frameworktype.SliceOfPrimitiveToGo[string](ctx, osv.NotIn.Elements())
			if err != nil {
				diags.AddError(
					"convertPolicyAssignmentOverridesToSdkType: error",
					fmt.Sprintf("unable to convert OverrideSelctorsValue.NotIn elements to Go slice: %s", err.Error()),
				)
				return nil, diags
			}
			selectors[j] = &armpolicy.Selector{
				Kind:  to.Ptr(armpolicy.SelectorKind(osv.Kind.ValueString())),
				In:    in,
				NotIn: notIn,
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

func convertPolicyAssignmentResourceSelectorsToSdkType(ctx context.Context, input []gen.ResourceSelectorsValue) ([]*armpolicy.ResourceSelector, diag.Diagnostics) {
	var diags diag.Diagnostics
	if len(input) == 0 {
		return nil, nil
	}
	res := make([]*armpolicy.ResourceSelector, len(input))
	for i, rs := range input {
		selectors := make([]*armpolicy.Selector, len(rs.ResourceSelectorSelectors.Elements()))
		for j, s := range rs.ResourceSelectorSelectors.Elements() {
			rssv, ok := s.(gen.ResourceSelectorSelectorsValue)
			if !ok {
				diags.AddError(
					"convertPolicyAssignmentResourceSelectorsToSdkType: error",
					"unable to convert resource selector selectors attr.Value to concrete type",
				)
			}
			in, err := frameworktype.SliceOfPrimitiveToGo[string](ctx, rssv.In.Elements())
			if err != nil {
				diags.AddError(
					"convertPolicyAssignmentResourceSelectorsToSdkType: error",
					fmt.Sprintf("unable to convert ResourceSelectorSelectorsValue.In elements to Go slice: %s", err.Error()),
				)
				return nil, diags
			}
			notIn, err := frameworktype.SliceOfPrimitiveToGo[string](ctx, rssv.NotIn.Elements())
			if err != nil {
				diags.AddError(
					"convertPolicyAssignmentResourceSelectorsToSdkType: error",
					fmt.Sprintf("unable to convert ResourceSelectorSelectorsValue.NotIn elements to Go slice: %s", err.Error()),
				)
				return nil, diags
			}
			selectors[j] = &armpolicy.Selector{
				Kind:  to.Ptr(armpolicy.SelectorKind(rssv.Kind.ValueString())),
				In:    in,
				NotIn: notIn,
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

func convertPolicyAssignmentNonComplianceMessagesToSdkType(src []gen.NonComplianceMessagesValue) []*armpolicy.NonComplianceMessage {
	if len(src) == 0 {
		return nil
	}
	res := make([]*armpolicy.NonComplianceMessage, len(src))

	for i, msg := range src {
		res[i] = &armpolicy.NonComplianceMessage{
			Message: msg.Message.ValueStringPointer(),
		}
		if isKnown(msg.PolicyDefinitionReferenceId) {
			res[i].PolicyDefinitionReferenceID = to.Ptr(msg.PolicyDefinitionReferenceId.ValueString())
		}
	}
	return res
}

func convertPolicyAssignmentIdentityToSdkType(typ types.String, ids types.Set) (*armpolicy.Identity, diag.Diagnostics) {
	var diags diag.Diagnostics
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
			diags.AddError(
				"convertPolicyAssignmentIdentityToSdkType: error",
				"one (and only one) identity id is required for user assigned identity",
			)
			return nil, diags
		}
		idStr, ok := ids.Elements()[0].(types.String)
		if !ok {
			diags.AddError(
				"convertPolicyAssignmentIdentityToSdkType: error",
				"unable to convert identity id to string",
			)
			return nil, diags
		}
		id = idStr.ValueString()

		identity = to.Ptr(armpolicy.Identity{
			Type:                   to.Ptr(armpolicy.ResourceIdentityTypeUserAssigned),
			UserAssignedIdentities: map[string]*armpolicy.UserAssignedIdentitiesValue{id: {}},
		})
	default:
		diags.AddError(
			"convertPolicyAssignmentIdentityToSdkType: error",
			fmt.Sprintf("unknown identity type: %s", typ.ValueString()),
		)
		return nil, diags
	}
	return identity, nil
}

// convertPolicyAssignmentParametersToSdkType converts a map[string]any to a map[string]*armpolicy.ParameterValuesValue.
func convertPolicyAssignmentParametersToSdkType(src alztypes.PolicyParameterValue) (map[string]*armpolicy.ParameterValuesValue, diag.Diagnostics) {
	var diags diag.Diagnostics
	if !isKnown(src) {
		return nil, nil
	}
	params := make(map[string]any)
	if err := json.Unmarshal([]byte(src.ValueString()), &params); err != nil {
		diags.AddError(
			"convertPolicyAssignmentParametersToSdkType: error",
			fmt.Sprintf("convertPolicyAssignmentParametersToSdkType: unable to unmarshal policy parameters: %s", err.Error()),
		)
		return nil, diags
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