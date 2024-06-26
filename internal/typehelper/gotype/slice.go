package gotype

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/attr"
)

func SliceOfPrimitiveToFramework[T ToFrameworkPrimitive](ctx context.Context, input []*T) []attr.Value {
	res := make([]attr.Value, 0, len(input))
	for _, v := range input {
		res = append(res, PrimitiveToFramework(ctx, v))
	}
	return res
}
