package frameworktype

// func TestObjectToGo(t *testing.T) {
// 	ctx := context.Background()

// 	t.Run("ValidObject", func(t *testing.T) {
// 		nestedInput, diags := types.ObjectValue(
// 			map[string]attr.Type{
// 				"nestedKey1": types.StringType,
// 			},
// 			map[string]attr.Value{
// 				"nestedKey1": types.StringValue("nestedValue1"),
// 			},
// 		)
// 		input, d := types.ObjectValue(
// 			map[string]attr.Type{
// 				"key1":   types.StringType,
// 				"key2":   types.NumberType,
// 				"key3":   types.BoolType,
// 				"nested": nestedInput.Type(ctx),
// 			},
// 			map[string]attr.Value{
// 				"key1":   types.StringValue("value1"),
// 				"key2":   types.NumberValue(big.NewFloat(3.14)),
// 				"key3":   types.BoolValue(true),
// 				"nested": nestedInput,
// 			},
// 		)
// 		diags.Append(d...)

// 		if diags.ErrorsCount() > 0 {
// 			t.Fatalf("unexpected diags: %v", diags)
// 		}

// 		type outputTypeNested struct {
// 			NestedKey1 string `tfsdk:"nestedKey1"`
// 		}

// 		type outputType struct {
// 			Key1   string           `tfsdk:"key1"`
// 			Key2   float64          `tfsdk:"key2"`
// 			Key3   bool             `tfsdk:"key3"`
// 			Nested outputTypeNested `tfsdk:"nested"`
// 		}

// 		var output outputType

// 		diags.Append(ObjectToGo(ctx, input, &output)...)

// 		if diags.ErrorsCount() > 0 {
// 			t.Fatalf("unexpected diags: %v", diags)
// 		}

// 		assert.Empty(t, diags)
// 		assert.Equal(t, outputType{
// 			Key1: "value1",
// 			Key2: 3.14,
// 			Key3: true,
// 			Nested: outputTypeNested{
// 				NestedKey1: "nestedValue1",
// 			},
// 		}, output)
// 	})

// 	t.Run("InvalidObject", func(t *testing.T) {
// 		input := types.StringValue("not an object")

// 		var output map[string]interface{}
// 		diags := ObjectToGo(ctx, input, &output)

// 		assert.Equal(t, diag.Diagnostics{
// 			diag.NewErrorDiagnostic("expected object value", ""),
// 		}, diags)
// 		assert.Nil(t, output)
// 	})
// }
