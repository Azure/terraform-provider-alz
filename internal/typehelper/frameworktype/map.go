package frameworktype

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/attr"
)

func MapOfPrimitiveToGo[T ToGoPrimitive](ctx context.Context, input map[string]attr.Value) map[string]*T {
	res := make(map[string]*T, len(input))
	for k, v := range input {
		res[k] = PrimitiveToGo[T](ctx, v)
	}
	return res
}
