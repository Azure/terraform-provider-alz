package gotype

import (
	"context"
	"testing"

	"github.com/Azure/alzlib/to"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
)

func TestSliceOfPrimitiveToFramework(t *testing.T) {
	ctx := context.Background()

	t.Run("EmptySlice", func(t *testing.T) {
		input := []*string{}
		want := []attr.Value{}
		got := SliceOfPrimitiveToFramework[string](ctx, input)
		assert.Equal(t, want, got)
	})

	t.Run("NonEmptySlice", func(t *testing.T) {
		input := []*string{
			to.Ptr("foo"),
			to.Ptr("bar"),
			to.Ptr("baz"),
		}
		want := []attr.Value{
			types.StringValue("foo"),
			types.StringValue("bar"),
			types.StringValue("baz"),
		}
		got := SliceOfPrimitiveToFramework[string](ctx, input)
		assert.Equal(t, want, got)
	})

	t.Run("SliceWithNilValue", func(t *testing.T) {
		input := []*string{
			to.Ptr("foo"),
			nil,
			to.Ptr("baz"),
		}
		want := []attr.Value{
			types.StringValue("foo"),
			types.StringNull(),
			types.StringValue("baz"),
		}
		got := SliceOfPrimitiveToFramework[string](ctx, input)
		assert.Equal(t, want, got)
	})
}
