package frameworktype

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

func ObjectToGo(ctx context.Context, input attr.Value, output any) diag.Diagnostics {
	objInput, ok := input.(basetypes.ObjectValue)
	if !ok {
		return diag.Diagnostics{diag.NewErrorDiagnostic("expected object value", "")}
	}
	return objInput.As(ctx, output, basetypes.ObjectAsOptions{
		UnhandledNullAsEmpty:    false,
		UnhandledUnknownAsEmpty: false,
	})
}
