package frameworktype

import (
	"testing"

	"github.com/Azure/alzlib/to"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
)

func TestMapOfPrimitiveToGo(t *testing.T) {
	ctx := t.Context()

	t.Run("EmptyInput", func(t *testing.T) {
		input := make(map[string]attr.Value)
		want := make(map[string]*string)
		got, _ := MapOfPrimitiveToGo[string](ctx, input)
		assert.Equal(t, want, got)
	})

	t.Run("StringInput", func(t *testing.T) {
		input := map[string]attr.Value{
			"key1": types.StringValue("value1"),
			"key2": types.StringValue("value2"),
			"key3": types.StringValue("value3"),
		}
		want := map[string]*string{
			"key1": to.Ptr("value1"),
			"key2": to.Ptr("value2"),
			"key3": to.Ptr("value3"),
		}
		got, _ := MapOfPrimitiveToGo[string](ctx, input)
		assert.Equal(t, want, got)
	})
}
