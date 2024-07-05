package frameworktype

import (
	"context"
	"fmt"
	"math/big"
	"reflect"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type ToGoPrimitive interface {
	int64 | float64 | string | bool
}

func PrimitiveToGo[T ToGoPrimitive](ctx context.Context, input attr.Value) (*T, error) {
	ty := input.Type(ctx)

	switch {
	case ty == types.BoolType:
		val, ok := input.(types.Bool)
		if !ok {
			return nil, fmt.Errorf("PrimitiveToGo: unexpected type conversion, %s to %T", ty.String(), new(T))

		}
		ret, ok := any(val.ValueBoolPointer()).(*T)
		if !ok {
			return nil, fmt.Errorf("PrimitiveToGo: unexpected type conversion, %s to %T", ty.String(), new(T))

		}
		return ret, nil
	case ty == types.StringType:
		val, ok := input.(types.String)
		if !ok {
			return nil, fmt.Errorf("PrimitiveToGo: unexpected type conversion, %s to %T", ty.String(), new(T))
		}
		ret, ok := any(val.ValueStringPointer()).(*T)
		if !ok {
			return nil, fmt.Errorf("PrimitiveToGo: unexpected type conversion, %s to %T", ty.String(), new(T))
		}
		return ret, nil
	case ty == types.NumberType:
		if input.IsUnknown() {
			if reflect.TypeOf(new(T)) == reflect.TypeOf(new(int64)) {
				zero := int64(0)
				ret, ok := any(&zero).(*T)
				if !ok {
					return nil, fmt.Errorf("PrimitiveToGo: unexpected type conversion, %s to %T", ty.String(), new(T))
				}
				return ret, nil
			}
			if reflect.TypeOf(new(T)) == reflect.TypeOf(new(float64)) {
				zero := float64(0)
				ret, ok := any(&zero).(*T)
				if !ok {
					return nil, fmt.Errorf("PrimitiveToGo: unexpected type conversion, %s to %T", ty.String(), new(T))
				}
				return ret, nil
			}
		}
		val, ok := input.(types.Number)
		if !ok {
			return nil, fmt.Errorf("PrimitiveToGo: unexpected type conversion, %s to %T", ty.String(), new(T))
		}
		valBig := val.ValueBigFloat()
		if valBig == nil {
			return nil, nil
		}
		if valBig.IsInt() && reflect.TypeOf(new(T)) == reflect.TypeOf(new(int64)) {
			valInt64, acc := valBig.Int64()
			if acc != big.Exact {
				return nil, fmt.Errorf("PrimitiveToGo: number conversion to int64 resulted in insufficient accuracy: %s", valBig)
			}
			ret, ok := any(&valInt64).(*T)
			if !ok {
				return nil, fmt.Errorf("PrimitiveToGo: unexpected type conversion, %s to %T", ty.String(), new(T))
			}
			return ret, nil
		}
		if reflect.TypeOf(new(T)) == reflect.TypeOf(new(float64)) {
			valFLoat64, acc := valBig.Float64()
			if acc != big.Exact {
				return nil, fmt.Errorf("PrimitiveToGo: number conversion to float64 resulted in insufficient accuracy: %s", valBig)
			}
			ret, ok := any(&valFLoat64).(*T)
			if !ok {
				return nil, fmt.Errorf("PrimitiveToGo: unexpected type conversion, %s to %T", ty.String(), new(T))
			}
			return ret, nil
		}
	}
	return nil, fmt.Errorf("PrimitiveToGo: unexpected input type %s", ty.String())
}
