package frameworktype

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
)

func SliceOfPrimitiveToGo[T ToGoPrimitive](ctx context.Context, input []attr.Value) ([]*T, error) {
	res := make([]*T, 0, len(input))
	for _, v := range input {
		val, err := PrimitiveToGo[T](ctx, v)
		if err != nil {
			return nil, fmt.Errorf("SliceOfPrimitiveToGo error converting element: %w", err)
		}
		res = append(res, val)
	}
	return res, nil
}
