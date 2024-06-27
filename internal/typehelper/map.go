package typehelper

import (
	"encoding/json"

	"github.com/Azure/alzlib/assets"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

// AlzMapTypes is used for the generic functions that operate on certain map types.
type AlzMapTypes interface {
	*assets.PolicyAssignment |
		*assets.PolicyDefinition |
		*assets.PolicySetDefinition |
		*assets.RoleDefinition
}

// ConvertAlzMapToFrameworkType converts a map[string]armTypes to a map[string]attr.Value, using types.StringType as the value type.
func ConvertAlzMapToFrameworkType[T AlzMapTypes](m map[string]T) (basetypes.MapValue, diag.Diagnostics) {
	result := make(map[string]attr.Value, len(m))
	for k, v := range m {
		b, err := json.Marshal(v)
		if err != nil {
			var diags diag.Diagnostics
			diags.AddError("ConvertMapOfStringToMapValue: Unable to marshal ARM object", err.Error())
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
