package frameworktype

import (
	"context"
	"math/big"
	"reflect"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type ToGoPrimitive interface {
	int64 | float64 | string | bool
}

func ToGo[T ToGoPrimitive](ctx context.Context, input attr.Value) *T {
	ty := input.Type(ctx)
	switch {
	case ty == types.BoolType:
		val, _ := input.(types.Bool)
		return any(val.ValueBoolPointer()).(*T)
	case ty == types.StringType:
		val, _ := input.(types.String)
		return any(val.ValueStringPointer()).(*T)
	case ty == types.NumberType:
		if input.IsUnknown() {
			if reflect.TypeOf(new(T)) == reflect.TypeOf(new(int64)) {
				zero := int64(0)
				return any(&zero).(*T)
			}
			if reflect.TypeOf(new(T)) == reflect.TypeOf(new(float64)) {
				zero := float64(0)
				return any(&zero).(*T)
			}
		}
		val, _ := input.(types.Number)
		valBig := val.ValueBigFloat()
		if valBig == nil {
			return nil
		}
		if valBig.IsInt() && reflect.TypeOf(new(T)) == reflect.TypeOf(new(int64)) {
			valInt64, acc := valBig.Int64()
			if acc != big.Exact {
				return nil
			}
			return any(&valInt64).(*T)
		}
		if reflect.TypeOf(new(T)) == reflect.TypeOf(new(float64)) {
			valFLoat64, acc := valBig.Float64()
			if acc != big.Exact {
				return nil
			}
			return any(&valFLoat64).(*T)
		}
	}
	return nil
}
