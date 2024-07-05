package frameworktype

import (
	"context"
	"testing"

	"github.com/Azure/alzlib/to"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
)

func TestSliceOfPrimitiveToGo(t *testing.T) {
	ctx := context.Background()

	t.Run("EmptySlice", func(t *testing.T) {
		input := []attr.Value{}
		want := []*string{}
		got, _ := SliceOfPrimitiveToGo[string](ctx, input)
		assert.Equal(t, want, got)
	})

	t.Run("NonEmptySlice", func(t *testing.T) {
		input := []attr.Value{
			types.StringValue("foo"),
			types.StringValue("bar"),
			types.StringValue("baz"),
		}
		want := []*string{
			to.Ptr("foo"),
			to.Ptr("bar"),
			to.Ptr("baz"),
		}
		got, _ := SliceOfPrimitiveToGo[string](ctx, input)
		assert.Equal(t, want, got)
	})

	t.Run("SliceWithNullValue", func(t *testing.T) {
		input := []attr.Value{
			types.StringValue("foo"),
			types.StringNull(),
			types.StringValue("baz"),
		}
		want := []*string{
			to.Ptr("foo"),
			nil,
			to.Ptr("baz"),
		}
		got, _ := SliceOfPrimitiveToGo[string](ctx, input)
		assert.Equal(t, want, got)
	})
}
