// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package alztypes_test

import (
	"context"
	"testing"

	"github.com/Azure/terraform-provider-alz/internal/alztypes"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/stretchr/testify/assert"
)

func TestStringSemanticEquals(t *testing.T) {
	str := `{"param": "value"}`
	ppv := alztypes.PolicyParameterValue{
		basetypes.NewStringValue(str),
	}

	var ppt alztypes.PolicyParameterType
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	str2 := `{"param": "value"}`
	sv := basetypes.NewStringValue(str2)
	strv2, diags := ppt.ValueFromString(ctx, sv)
	assert.False(t, diags.HasError())

	equal, diags := ppv.StringSemanticEquals(ctx, strv2)
	assert.False(t, diags.HasError())
	assert.True(t, equal)
}

func TestStringSemanticEqualsOutOfOrder(t *testing.T) {
	got := `{"param2": "value2", "param1": 1}`
	ppv := alztypes.PolicyParameterValue{
		basetypes.NewStringValue(got),
	}

	var ppt alztypes.PolicyParameterType
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	want := `{"param1": 1, "param2": "value2"}`
	sv := basetypes.NewStringValue(want)
	strv2, diags := ppt.ValueFromString(ctx, sv)
	assert.False(t, diags.HasError())

	equal, diags := ppv.StringSemanticEquals(ctx, strv2)
	assert.False(t, diags.HasError())
	assert.True(t, equal)
}
