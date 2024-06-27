package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"

	"github.com/Azure/alzlib/deployment"
	"github.com/Azure/alzlib/to"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armpolicy"
	"github.com/Azure/terraform-provider-alz/internal/alztypes"
	"github.com/Azure/terraform-provider-alz/internal/provider/gen"
	"github.com/Azure/terraform-provider-alz/internal/typehelper"
	"github.com/Azure/terraform-provider-alz/internal/typehelper/frameworktype"
	mapset "github.com/deckarep/golang-set/v2"
	"github.com/google/uuid"
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
	diags := data.PolicyAssignmentsToModify.ElementsAs(ctx, &mg2paModMap, false)
	resp.Diagnostics.Append(diags...)
	for mgName, pa2mod := range mg2paModMap {
		pa2modMap := make(map[string]gen.PolicyAssignmentsValue)
		diags = pa2mod.PolicyAssignments.ElementsAs(ctx, &pa2modMap, false)
		resp.Diagnostics.Append(diags...)
		for paName, mod := range pa2modMap {
			enf, ident, noncompl, params, resourceSel, overrides, err := policyAssignmentType2ArmPolicyValues(mod)
			if err != nil {
				resp.Diagnostics.AddError(
					"architectureDataSource.Read() Error converting policy assignment to ARM values",
					err.Error(),
				)
			}
			if err := depl.ManagementGroup(mgName).ModifyPolicyAssignment(paName, params, enf, noncompl, ident, resourceSel, overrides); err != nil {
				resp.Diagnostics.AddError(
					fmt.Sprintf("architectureDataSource.Read() Error modifying policy assignment `%s` at mg `%s`", paName, mgName),
					err.Error(),
				)
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
	policyRoleAssignmentsVal, diags := policyRoleAssignmentsSetToProviderType(ctx, policyRoleAssignments)
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

func policyRoleAssignmentsSetToProviderType(ctx context.Context, input mapset.Set[deployment.PolicyRoleAssignment]) (basetypes.SetValue, diag.Diagnostics) {
	var diags diag.Diagnostics
	praSlice := make([]gen.PolicyRoleAssignmentsValue, 0, input.Cardinality())
	for i := range input.Iter() {
		pra, diag := policyRoleAssignmentToProviderType(ctx, i)
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
	nonCompl := make([]gen.NonComplianceMessagesValue, len(pa.NonComplianceMessages.Elements()))
	diag = pa.NonComplianceMessages.ElementsAs(ctx, &nonCompl, false)
	diags.Append(diag...)
	if diags.HasError() {
		return nil, nil, nil, nil, nil, nil, diags
	}
	nonComplianceMessages = convertPolicyAssignmentNonComplianceMessagesToSdkType(nonCompl)

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
	rS := make([]gen.ResourceSelectorsValue, len(pa.ResourceSelectors.Elements()))
	diag = pa.ResourceSelectors.ElementsAs(ctx, &rS, false)
	diags.Append(diag...)
	resourceSelectors, err = convertPolicyAssignmentResourceSelectorsToSdkType(rS)
	if err != nil {
		return nil, nil, nil, nil, nil, nil, diags
	}

	// set overrides
	ovr := make([]gen.OverridesValue, len(pa.Overrides.Elements()))
	diag = pa.Overrides.ElementsAs(ctx, &ovr, false)
	overrides, err = convertPolicyAssignmentOverridesToSdkType(ovr)
	if err != nil {
		return nil, nil, nil, nil, nil, nil, diags
	}

	return enforcementMode, identity, nonComplianceMessages, parameters, resourceSelectors, overrides, nil
}

func convertPolicyAssignmentOverridesToSdkType(src []gen.OverridesValue) ([]*armpolicy.Override, error) {
	if len(src) == 0 {
		return nil, nil
	}
	res := make([]*armpolicy.Override, len(src))
	for i, o := range src {
		selectors := make([]*armpolicy.Selector, len(o.Selectors))
		for j, s := range o.Selectors {
			in, err := typehelper.AttrSlice2StringSlice(s.In.Elements())
			if err != nil {
				return nil, fmt.Errorf("unable to convert override selector `in` in value to string %w", err)
			}
			notIn, err := typehelper.AttrSlice2StringSlice(s.NotIn.Elements())
			if err != nil {
				return nil, fmt.Errorf("unable to convert override selector `not_in` in value to string %w", err)
			}
			selectors[j] = &armpolicy.Selector{
				Kind:  to.Ptr(armpolicy.SelectorKind(s.Kind.ValueString())),
				In:    to.SliceOfPtrs(in...),
				NotIn: to.SliceOfPtrs(notIn...),
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

func convertPolicyAssignmentResourceSelectorsToSdkType(ctx context.Context, src []gen.ResourceSelectorsValue) ([]*armpolicy.ResourceSelector, diag.Diagnostics) {
	var diags diag.Diagnostics
	if len(src) == 0 {
		return nil, nil
	}
	res := make([]*armpolicy.ResourceSelector, len(src))
	for i, rs := range src {
		selectors := make([]*armpolicy.Selector, len(rs.Selectors))
		for j, s := range rs.Selectors {
			in := frameworktype.SliceOfPrimitiveToGo[string](ctx, s.In.Elements())
			if err != nil {
				return nil, fmt.Errorf("unable to convert resource selector selector `in` in value to string %w", err)
			}
			notIn, err := typehelper.AttrSlice2StringSlice(s.NotIn.Elements())
			if err != nil {
				return nil, fmt.Errorf("unable to convert resource selector selector `not_in` in value to string %w", err)
			}
			selectors[j] = &armpolicy.Selector{
				Kind:  to.Ptr(armpolicy.SelectorKind(s.Kind.ValueString())),
				In:    to.SliceOfPtrs(in...),
				NotIn: to.SliceOfPtrs(notIn...),
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
				"convertPolicyAssignmentIdentityToSdkType: one (and only one) identity id is required for user assigned identity",
				"",
			)
			return nil, diags
		}
		idStr, ok := ids.Elements()[0].(types.String)
		if !ok {
			diags.AddError(
				"convertPolicyAssignmentIdentityToSdkType: error converting identity to SDK type",
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
			"convertPolicyAssignmentIdentityToSdkType: error converting identity to SDK type",
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
			fmt.Sprintf("convertPolicyAssignmentParametersToSdkType: unable to unmarshal policy parameters: %w", err),
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

func genPolicyRoleAssignmentId(pra deployment.PolicyRoleAssignment) string {
	u := uuid.NewSHA1(uuid.NameSpaceURL, []byte(pra.AssignmentName+pra.RoleDefinitionId+pra.Scope))
	return u.String()
}
