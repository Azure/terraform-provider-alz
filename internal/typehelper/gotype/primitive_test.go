package gotype

import (
	"context"
	"math/big"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestToFramework(t *testing.T) {
	ctx := context.Background()

	// Test int64 input
	var i int64 = 42
	int64Result := ToFramework(ctx, &i)
	expectedInt64Result := types.NumberValue(big.NewFloat(float64(i)))
	if !int64Result.Equal(expectedInt64Result) {
		t.Errorf("Expected int64 result to be %v, but got %v", expectedInt64Result, int64Result)
	}

	// Test float64 input
	var f float64 = 3.14
	float64Result := ToFramework(ctx, &f)
	expectedFloat64Result := types.NumberValue(big.NewFloat(f))
	if !float64Result.Equal(expectedFloat64Result) {
		t.Errorf("Expected float64 result to be %v, but got %v", expectedFloat64Result, float64Result)
	}

	// Test string input
	s := "hello"
	stringResult := ToFramework(ctx, &s)
	expectedStringResult := types.StringValue(s)
	if !stringResult.Equal(expectedStringResult) {
		t.Errorf("Expected string result to be %v, but got %v", expectedStringResult, stringResult)
	}

	// Test bool input
	b := true
	boolResult := ToFramework(ctx, &b)
	expectedBoolResult := types.BoolValue(b)
	if !boolResult.Equal(expectedBoolResult) {
		t.Errorf("Expected bool result to be %v, but got %v", expectedBoolResult, boolResult)
	}

	// Test nil input
	nilIntResult := ToFramework[int64](ctx, nil)
	if !nilIntResult.IsNull() {
		t.Errorf("Expected nil result to be nil, but got %v", nilIntResult)
	}
}
