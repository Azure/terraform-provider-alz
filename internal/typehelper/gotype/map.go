package gotype

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/attr"
)

func MapOfPrimitiveToFramework[T ToFrameworkPrimitive](ctx context.Context, input map[string]*T) map[string]attr.Value {
	res := make(map[string]attr.Value, len(input))
	for k, v := range input {
		res[k] = PrimitiveToFramework(ctx, v)
	}
	return res
}
