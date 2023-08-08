// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package alztypes

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

// Ensure the implementation satisfies the expected interfaces.
var _ basetypes.StringValuable = PolicyParameterValue{}

type PolicyParameterValue struct {
	basetypes.StringValue
	// ... potentially other fields ...
}

func (v PolicyParameterValue) Map() (PolicyParameterMap, error) {
	var policyParameterMap PolicyParameterMap

	if err := json.Unmarshal([]byte(v.StringValue.ValueString()), &policyParameterMap); err != nil {
		return nil, fmt.Errorf("unable to parse the PolicyParameterValue as a policyParameterMap JSON object: %w", err)
	}

	return policyParameterMap, nil
}

func (v PolicyParameterValue) Equal(o attr.Value) bool {
	other, ok := o.(PolicyParameterValue)

	if !ok {
		return false
	}

	return v.StringValue.Equal(other.StringValue)
}

func (v PolicyParameterValue) Type(ctx context.Context) attr.Type {
	// CustomStringType defined in the schema type section.
	return PolicyParameterType{}
}

// PolicyParameterValue defined in the value type section.
// Ensure the implementation satisfies the expected interfaces.
var _ basetypes.StringValuableWithSemanticEquals = PolicyParameterValue{}

func (v PolicyParameterValue) StringSemanticEquals(ctx context.Context, newValuable basetypes.StringValuable) (bool, diag.Diagnostics) {
	var diags diag.Diagnostics

	// The framework should always pass the correct value type, but always check.
	newValue, ok := newValuable.(PolicyParameterValue)

	if !ok {
		diags.AddError(
			"Semantic Equality Check Error",
			"An unexpected value type was received while performing semantic equality checks. "+
				"Please report this to the provider developers.\n\n"+
				"Expected Value Type: "+fmt.Sprintf("%T", v)+"\n"+
				"Got Value Type: "+fmt.Sprintf("%T", newValuable),
		)

		return false, diags
	}

	unmarshalMap := make(map[PolicyParameterValue]*PolicyParameterMap, 2)
	unmarshalMap[v] = new(PolicyParameterMap)
	unmarshalMap[newValue] = new(PolicyParameterMap)

	for ppv, ppm := range unmarshalMap {
		if err := json.Unmarshal([]byte(ppv.StringValue.ValueString()), ppm); err != nil {
			diags.AddError(
				"Semantic Equality Check Error",
				"Unable to parse the PolicyParameterValue as a policyParameterMap JSON object. "+
					"Please report this to the provider developers.\n\n"+
					"Error: "+err.Error(),
			)
			return false, diags
		}
	}

	// If the times are equivalent, keep the prior value.
	return reflect.DeepEqual(unmarshalMap[v], unmarshalMap[newValue]), diags
}
