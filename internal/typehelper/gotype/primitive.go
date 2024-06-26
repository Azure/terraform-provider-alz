package gotype

import (
	"context"
	"math/big"
	"reflect"

	"github.com/Azure/alzlib/to"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type ToFrameworkPrimitive interface {
	int64 | float64 | string | bool
}

func PrimitiveToFramework[T ToFrameworkPrimitive](ctx context.Context, input *T) attr.Value {
	switch {
	case reflect.TypeOf(input) == reflect.TypeOf(to.Ptr(int64(0))):
		if input == nil {
			return types.NumberNull()
		}
		i, _ := reflect.ValueOf(*input).Interface().(int64)
		return types.NumberValue(big.NewFloat(float64(i)))
	case reflect.TypeOf(input) == reflect.TypeOf(to.Ptr(float64(0))):
		if input == nil {
			return types.NumberNull()
		}
		f, _ := reflect.ValueOf(*input).Interface().(float64)
		return types.NumberValue(big.NewFloat(f))
	case reflect.TypeOf(input) == reflect.TypeOf(to.Ptr("")):
		if input == nil {
			return types.StringNull()
		}
		s, _ := reflect.ValueOf(*input).Interface().(string)
		return types.StringValue(s)
	case reflect.TypeOf(input) == reflect.TypeOf(to.Ptr(true)):
		if input == nil {
			return types.BoolNull()
		}
		b, _ := reflect.ValueOf(*input).Interface().(bool)
		return types.BoolValue(b)
	}
	return nil
}
