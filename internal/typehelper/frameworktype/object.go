package frameworktype

// Removed as this does not work with nested objects yet

// func ObjectToGo(ctx context.Context, input attr.Value, output any) diag.Diagnostics {
// 	objInput, ok := input.(basetypes.ObjectValue)
// 	if !ok {
// 		return diag.Diagnostics{diag.NewErrorDiagnostic("expected object value", "")}
// 	}
// 	return objInput.As(ctx, output, basetypes.ObjectAsOptions{
// 		UnhandledNullAsEmpty:    false,
// 		UnhandledUnknownAsEmpty: false,
// 	})
// }
