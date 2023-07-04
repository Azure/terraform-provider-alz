package alztypes

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// Ensure the implementation satisfies the expected interfaces
var _ basetypes.StringTypable = PolicyParameterType{}

type PolicyParameterType struct {
	basetypes.StringType
	// ... potentially other fields ...
}

// PolicyParameterMap is a map of string to any
// and is used to represent ARM policy parameter values
type PolicyParameterMap map[string]any

func (t PolicyParameterType) Equal(o attr.Type) bool {
	other, ok := o.(PolicyParameterType)

	if !ok {
		return false
	}

	return t.StringType.Equal(other.StringType)
}

func (t PolicyParameterType) String() string {
	return "PolicyParameterType"
}

func (t PolicyParameterType) ValueFromString(ctx context.Context, in basetypes.StringValue) (basetypes.StringValuable, diag.Diagnostics) {
	// CustomStringValue defined in the value type section
	value := PolicyParameterValue{
		StringValue: in,
	}

	return value, nil
}

func (t PolicyParameterType) ValueFromTerraform(ctx context.Context, in tftypes.Value) (attr.Value, error) {
	attrValue, err := t.StringType.ValueFromTerraform(ctx, in)

	if err != nil {
		return nil, err
	}

	stringValue, ok := attrValue.(basetypes.StringValue)

	if !ok {
		return nil, fmt.Errorf("unexpected value type of %T", attrValue)
	}

	stringValuable, diags := t.ValueFromString(ctx, stringValue)

	if diags.HasError() {
		return nil, fmt.Errorf("unexpected error converting StringValue to StringValuable: %v", diags)
	}

	return stringValuable, nil
}

func (t PolicyParameterType) ValueType(ctx context.Context) attr.Value {
	// PolicyParameterType defined in the value type section
	return PolicyParameterValue{}
}

// PolicyParameterType defined in the schema type section
func (t PolicyParameterType) Validate(ctx context.Context, value tftypes.Value, valuePath path.Path) diag.Diagnostics {
	if value.IsNull() || !value.IsKnown() {
		return nil
	}

	var diags diag.Diagnostics
	var valueString string

	if err := value.As(&valueString); err != nil {
		diags.AddAttributeError(
			valuePath,
			"Invalid Terraform Value",
			"An unexpected error occurred while attempting to convert a Terraform value to a string. "+
				"This generally is an issue with the provider schema implementation. "+
				"Please contact the provider developers.\n\n"+
				"Path: "+valuePath.String()+"\n"+
				"Error: "+err.Error(),
		)

		return diags
	}

	if !json.Valid([]byte(valueString)) {
		diags.AddAttributeError(
			valuePath,
			"Invalid policy parameter value",
			"An unexpected error occurred while attempting to convert a Terraform value to JSON. "+
				"This generally is an issue with the provider schema implementation. "+
				"Please contact the provider developers.\n\n"+
				"Path: "+valuePath.String()+"\n"+
				"Error: Invalid JSON",
		)

		return diags
	}

	paramMap := new(PolicyParameterMap)

	if err := json.Unmarshal([]byte(valueString), paramMap); err != nil {
		diags.AddAttributeError(
			valuePath,
			"Invalid policy parameter JSON",
			"An unexpected error occurred while converting a string value that was expected to be a JSON representation of policy parameters. "+
				"The string value was expected to unmarshal to a map[string]any value, but it did not.\n\n"+
				"Path: "+valuePath.String()+"\n"+
				"Given Value: "+valueString+"\n"+
				"Error: "+err.Error(),
		)

		return diags
	}

	return diags
}
