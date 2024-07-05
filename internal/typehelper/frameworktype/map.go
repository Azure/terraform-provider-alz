package frameworktype

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
)

func MapOfPrimitiveToGo[T ToGoPrimitive](ctx context.Context, input map[string]attr.Value) (map[string]*T, error) {
	res := make(map[string]*T, len(input))
	for k, v := range input {
		val, err := PrimitiveToGo[T](ctx, v)
		if err != nil {
			return nil, fmt.Errorf("MapOfPrimitiveToGo error converting element: %w", err)
		}
		res[k] = val
	}
	return res, nil
}
