package typehelper

import (
	"fmt"

	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/stretchr/testify/assert"
)

func TestAttrSlice2StringSlice(t *testing.T) {
	// Test case 1: Empty attribute slice
	attrSlice := []attr.Value{}
	expectedResult := []string{}
	result, err := AttrSlice2StringSlice(attrSlice)
	assert.NoError(t, err)
	assert.Equal(t, expectedResult, result)

	// Test case 2: Attribute slice with string values
	attrSlice = []attr.Value{
		basetypes.NewStringValue("value1"),
		basetypes.NewStringValue("value2"),
		basetypes.NewStringValue("value3"),
	}
	expectedResult = []string{"value1", "value2", "value3"}
	result, err = AttrSlice2StringSlice(attrSlice)
	assert.NoError(t, err)
	assert.Equal(t, expectedResult, result)

	// Test case 3: Attribute slice with non-string values
	attrSlice = []attr.Value{
		basetypes.NewStringValue("value1"),
		basetypes.NewInt64Value(1),
		basetypes.NewBoolValue(true),
	}
	_, err = AttrSlice2StringSlice(attrSlice)
	expectedError := fmt.Errorf("expected string, got basetypes.Int64Value")
	assert.EqualError(t, err, expectedError.Error())
}
