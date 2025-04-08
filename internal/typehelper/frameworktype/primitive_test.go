package frameworktype

import (
	"math/big"
	"testing"

	"github.com/Azure/alzlib/to"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
)

func TestToGo(t *testing.T) {
	ctx := t.Context()

	t.Run("NumberTypeFloat64", func(t *testing.T) {
		testCases := []struct {
			desc string
			in   attr.Value
			want *float64
		}{
			{
				desc: "zero",
				in:   types.NumberValue(big.NewFloat(0)),
				want: to.Ptr(float64(0)),
			},
			{
				desc: "non-zero",
				in:   types.NumberValue(big.NewFloat(72349234023974.12)),
				want: to.Ptr(float64(72349234023974.12)),
			},
			{
				desc: "overflow",
				in:   types.NumberValue(big.NewFloat(123456789012345678901234567890123456789.123456789012345678901234567890123456789)),
				want: to.Ptr(float64(123456789012345678901234567890123456789.123456789012345678901234567890123456789)),
			},
			{
				desc: "intAsFloat",
				in:   types.NumberValue(big.NewFloat(3)),
				want: to.Ptr(float64(3)),
			},
			{
				desc: "null",
				in:   types.NumberNull(),
				want: nil,
			},
			{
				desc: "unknown",
				in:   types.NumberUnknown(),
				want: to.Ptr(float64(0)),
			},
		}
		for _, tc := range testCases {
			t.Run(tc.desc, func(t *testing.T) {
				got, _ := PrimitiveToGo[float64](ctx, tc.in)
				assert.Equal(t, tc.want, got)
			})
		}
	})

	t.Run("NumberTypeInt64", func(t *testing.T) {
		testCases := []struct {
			desc string
			in   attr.Value
			want *int64
		}{
			{
				desc: "zero",
				in:   types.NumberValue(big.NewFloat(0)),
				want: to.Ptr(int64(0)),
			},
			{
				desc: "non-zero",
				in:   types.NumberValue(big.NewFloat(72349234023974)),
				want: to.Ptr(int64(72349234023974)),
			},
			{
				desc: "overflow",
				in:   types.NumberValue(big.NewFloat(72349233928498234823454023974)),
				want: nil,
			},
			{
				desc: "floatAsInt",
				in:   types.NumberValue(big.NewFloat(3.141)),
				want: nil,
			},
			{
				desc: "null",
				in:   types.NumberNull(),
				want: nil,
			},
			{
				desc: "unknown",
				in:   types.NumberUnknown(),
				want: to.Ptr(int64(0)),
			},
		}
		for _, tc := range testCases {
			t.Run(tc.desc, func(t *testing.T) {
				got, _ := PrimitiveToGo[int64](ctx, tc.in)
				assert.Equal(t, tc.want, got)
			})
		}
	})

	t.Run("StringType", func(t *testing.T) {
		testCases := []struct {
			desc string
			in   attr.Value
			want *string
		}{
			{
				desc: "empty",
				in:   types.StringValue(""),
				want: to.Ptr(""),
			},
			{
				desc: "non-empty",
				in:   types.StringValue("foo"),
				want: to.Ptr("foo"),
			},
			{
				desc: "null",
				in:   types.StringNull(),
				want: nil,
			},
			{
				desc: "unknown",
				in:   types.StringUnknown(),
				want: to.Ptr(""),
			},
		}
		for _, tc := range testCases {
			t.Run(tc.desc, func(t *testing.T) {
				got, _ := PrimitiveToGo[string](ctx, tc.in)
				assert.Equal(t, tc.want, got)
			})
		}
	})

	t.Run("BoolType", func(t *testing.T) {
		testCases := []struct {
			desc string
			in   attr.Value
			want *bool
		}{
			{
				desc: "true",
				in:   types.BoolValue(true),
				want: to.Ptr(true),
			},
			{
				desc: "false",
				in:   types.BoolValue(false),
				want: to.Ptr(false),
			},
			{
				desc: "null",
				in:   types.BoolNull(),
				want: nil,
			},
			{
				desc: "unknkown",
				in:   types.BoolUnknown(),
				want: to.Ptr(false),
			},
		}
		for _, tc := range testCases {
			t.Run(tc.desc, func(t *testing.T) {
				got, _ := PrimitiveToGo[bool](ctx, tc.in)
				assert.Equal(t, tc.want, got)
			})
		}
	})

	// Add more test cases for other types here
}
