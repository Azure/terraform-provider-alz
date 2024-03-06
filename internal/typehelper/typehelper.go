package typehelper

import (
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func AttrSlice2StringSlice(attr []attr.Value) ([]string, error) {
	result := make([]string, len(attr))
	for i, a := range attr {
		sv, ok := a.(types.String)
		if !ok {
			return nil, fmt.Errorf("expected string, got %T", a)
		}
		result[i] = sv.ValueString()
	}
	return result, nil
}
