// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package alztypes_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/Azure/terraform-provider-alz/internal/alztypes"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/stretchr/testify/assert"
)

func TestValueFromString(t *testing.T) {
	var ppt alztypes.PolicyParameterType
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	str := `{"param": "value"}`
	sv := basetypes.NewStringValue(str)
	_, diags := ppt.ValueFromString(ctx, sv)
	assert.False(t, diags.HasError())
}

func TestValidate(t *testing.T) {
	var ppt alztypes.PolicyParameterType
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pa := path.Root("test")
	str := `{"param": "value"}`
	tfval := tftypes.NewValue(tftypes.String, str)
	diags := ppt.Validate(ctx, tfval, pa)
	assert.Falsef(t, diags.HasError(), "diags: %v", diags)
}

func TestValidateInvalidJson(t *testing.T) {
	var ppt alztypes.PolicyParameterType
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pa := path.Root("test")
	str := `{"param": "value"`
	tfval := tftypes.NewValue(tftypes.String, str)
	diags := ppt.Validate(ctx, tfval, pa)
	assert.True(t, diags.HasError())
	assert.Contains(t, fmt.Sprintf("%v", diags), "An unexpected error occurred while attempting to convert a Terraform value to JSON")
}

func TestValidateInvalidStringConversion(t *testing.T) {
	var ppt alztypes.PolicyParameterType
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	pa := path.Root("test")
	tfval := tftypes.NewValue(tftypes.Bool, true)
	diags := ppt.Validate(ctx, tfval, pa)
	assert.True(t, diags.HasError())
	assert.Contains(t, fmt.Sprintf("%v", diags), "An unexpected error occurred while attempting to convert a Terraform value to a string")
}

func TestValidateInvalidJsonSchema(t *testing.T) {
	var ppt alztypes.PolicyParameterType
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	pa := path.Root("test")
	str := `["value1", "value2"]`
	tfval := tftypes.NewValue(tftypes.String, str)
	diags := ppt.Validate(ctx, tfval, pa)
	assert.True(t, diags.HasError(), "diags: %v", diags)
	assert.Contains(t, fmt.Sprintf("%v", diags), "An unexpected error occurred while converting a string value that was expected to be a JSON representation of policy parameters")
}
