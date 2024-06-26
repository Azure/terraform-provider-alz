package frameworktype

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/attr"
)

func SliceOfPrimitiveToGo[T ToGoPrimitive](ctx context.Context, input []attr.Value) []*T {
	res := make([]*T, 0, len(input))
	for _, v := range input {
		res = append(res, PrimitiveToGo[T](ctx, v))
	}
	return res
}
