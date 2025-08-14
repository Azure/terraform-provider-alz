package aztfschema

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Test setFieldDefaultsFromEnv using a custom struct with a single string field and fromenv tag
func Test_setFieldDefaultsFromEnv_SingleStringField(t *testing.T) {
	tests := []struct {
		name              string
		initial           types.String
		envValue          string
		wantValue         string
		wantIsNull        bool
		expectValueChange bool
	}{
		{
			name:              "sets value from env when field is null",
			initial:           types.StringNull(),
			envValue:          "s3cr3t",
			wantValue:         "s3cr3t",
			wantIsNull:        false,
			expectValueChange: true,
		},
		{
			name:              "does nothing when env var is not set",
			initial:           types.StringNull(),
			envValue:          "", // not set
			wantValue:         "",
			wantIsNull:        true,
			expectValueChange: false,
		},
		{
			name:              "does not override when field already set",
			initial:           types.StringValue("preset"),
			envValue:          "new-secret",
			wantValue:         "preset",
			wantIsNull:        false,
			expectValueChange: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Local struct under test
			type Secret struct {
				P types.String `fromenv:"ARM_CLIENT_CERTIFICATE_PASSWORD"`
			}

			// Isolate env per test
			if tt.envValue == "" {
				t.Setenv("ARM_CLIENT_CERTIFICATE_PASSWORD", "")
			} else {
				t.Setenv("ARM_CLIENT_CERTIFICATE_PASSWORD", tt.envValue)
			}

			m := &Secret{P: tt.initial}
			before := m.P

			setFieldDefaultsFromEnv(m)

			got := m.P

			if tt.wantIsNull != got.IsNull() {
				t.Fatalf("IsNull mismatch: want %v, got %v", tt.wantIsNull, got.IsNull())
			}

			if !got.IsNull() {
				if got.ValueString() != tt.wantValue {
					t.Fatalf("value mismatch: want %q, got %q", tt.wantValue, got.ValueString())
				}
			}

			changed := !before.Equal(got)
			if tt.expectValueChange != changed {
				t.Fatalf("change mismatch: want changed=%v, got %v (before=%v, after=%v)", tt.expectValueChange, changed, before, got)
			}
		})
	}
}

// Test precedence and fallback across multiple env vars listed in tag
func Test_setFieldDefaultsFromEnv_SubscriptionIDPrecedence(t *testing.T) {
	type Sub struct {
		SubscriptionID types.String `fromenv:"ARM_SUBSCRIPTION_ID,AZURE_SUBSCRIPTION_ID"`
	}

	// First non-empty should win
	t.Setenv("ARM_SUBSCRIPTION_ID", "11111111-1111-1111-1111-111111111111")
	t.Setenv("AZURE_SUBSCRIPTION_ID", "22222222-2222-2222-2222-222222222222")

	m := &Sub{SubscriptionID: types.StringNull()}
	setFieldDefaultsFromEnv(m)
	if m.SubscriptionID.IsNull() {
		t.Fatalf("expected subscription id to be set from env")
	}
	if got := m.SubscriptionID.ValueString(); got != "11111111-1111-1111-1111-111111111111" {
		t.Fatalf("expected first env var value to win, got %q", got)
	}

	// Fallback when first is empty
	t.Setenv("ARM_SUBSCRIPTION_ID", "")
	t.Setenv("AZURE_SUBSCRIPTION_ID", "22222222-2222-2222-2222-222222222222")

	m2 := &Sub{SubscriptionID: types.StringNull()}
	setFieldDefaultsFromEnv(m2)
	if m2.SubscriptionID.IsNull() || m2.SubscriptionID.ValueString() != "22222222-2222-2222-2222-222222222222" {
		t.Fatalf("expected fallback to second env var when first unset, got %v", m2.SubscriptionID)
	}
}

// Test that bool fields are now properly supported
func Test_setFieldDefaultsFromEnv_MixedTypesAndNoTag(t *testing.T) {
	type Mixed struct {
		A types.String `fromenv:"A_VAR"`
		B types.Bool   `fromenv:"B_VAR"`
		C types.String // no tag
	}

	t.Setenv("A_VAR", "foo")
	t.Setenv("B_VAR", "true")

	m := &Mixed{
		A: types.StringNull(),
		B: types.BoolNull(),
		C: types.StringNull(),
	}

	setFieldDefaultsFromEnv(m)

	if m.A.IsNull() || m.A.ValueString() != "foo" {
		t.Fatalf("expected A to be set from env, got %v", m.A)
	}
	// B should now be set because bool fields are supported
	if m.B.IsNull() || !m.B.ValueBool() {
		t.Fatalf("expected B to be set to true from env, got %v", m.B)
	}
	// C should remain null because there's no fromenv tag
	if !m.C.IsNull() {
		t.Fatalf("expected C to remain null (no tag), got %v", m.C)
	}
}

// Test setFieldDefaultsFromEnv with bool fields
func Test_setFieldDefaultsFromEnv_BoolField(t *testing.T) {
	tests := []struct {
		name              string
		initial           types.Bool
		envValue          string
		wantValue         bool
		wantIsNull        bool
		expectValueChange bool
	}{
		{
			name:              "sets true from env when field is null",
			initial:           types.BoolNull(),
			envValue:          "true",
			wantValue:         true,
			wantIsNull:        false,
			expectValueChange: true,
		},
		{
			name:              "sets false from env when field is null",
			initial:           types.BoolNull(),
			envValue:          "false",
			wantValue:         false,
			wantIsNull:        false,
			expectValueChange: true,
		},
		{
			name:              "sets true from 1 when field is null",
			initial:           types.BoolNull(),
			envValue:          "1",
			wantValue:         true,
			wantIsNull:        false,
			expectValueChange: true,
		},
		{
			name:              "sets false from 0 when field is null",
			initial:           types.BoolNull(),
			envValue:          "0",
			wantValue:         false,
			wantIsNull:        false,
			expectValueChange: true,
		},
		{
			name:              "does nothing when env var is not set",
			initial:           types.BoolNull(),
			envValue:          "", // not set
			wantValue:         false,
			wantIsNull:        true,
			expectValueChange: false,
		},
		{
			name:              "does not override when field already set",
			initial:           types.BoolValue(true),
			envValue:          "false",
			wantValue:         true,
			wantIsNull:        false,
			expectValueChange: false,
		},
		{
			name:              "ignores invalid bool values",
			initial:           types.BoolNull(),
			envValue:          "invalid",
			wantValue:         false,
			wantIsNull:        true,
			expectValueChange: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Local struct under test
			type BoolConfig struct {
				Enabled types.Bool `fromenv:"BOOL_ENABLED"`
			}

			// Isolate env per test
			if tt.envValue == "" {
				t.Setenv("BOOL_ENABLED", "")
			} else {
				t.Setenv("BOOL_ENABLED", tt.envValue)
			}

			m := &BoolConfig{Enabled: tt.initial}
			before := m.Enabled

			setFieldDefaultsFromEnv(m)

			got := m.Enabled

			if tt.wantIsNull != got.IsNull() {
				t.Fatalf("IsNull mismatch: want %v, got %v", tt.wantIsNull, got.IsNull())
			}

			if !got.IsNull() {
				if got.ValueBool() != tt.wantValue {
					t.Fatalf("value mismatch: want %v, got %v", tt.wantValue, got.ValueBool())
				}
			}

			changed := !before.Equal(got)
			if tt.expectValueChange != changed {
				t.Fatalf("change mismatch: want changed=%v, got %v (before=%v, after=%v)", tt.expectValueChange, changed, before, got)
			}
		})
	}
}

// Helper to convert a types.List of strings into []string for assertions.
func listToStrings(t *testing.T, l types.List) []string {
	t.Helper()
	if l.IsNull() {
		return nil
	}
	elems := l.Elements()
	out := make([]string, 0, len(elems))
	for _, ev := range elems {
		sv, ok := ev.(types.String)
		if !ok {
			t.Fatalf("list element is not types.String: %T", ev)
		}
		out = append(out, sv.ValueString())
	}
	return out
}

// Tests for list-typed fields populated from env
func Test_setFieldDefaultsFromEnv_ListField(t *testing.T) {
	type L struct {
		IDs types.List `fromenv:"A_IDS"`
	}

	tests := []struct {
		name              string
		initial           types.List
		envValue          string
		wantIsNull        bool
		wantValues        []string
		expectValueChange bool
	}{
		{
			name:              "sets list from env when field is null",
			initial:           types.ListNull(types.StringType),
			envValue:          "one;two;three",
			wantIsNull:        false,
			wantValues:        []string{"one", "two", "three"},
			expectValueChange: true,
		},
		{
			name:              "does nothing when env var is not set",
			initial:           types.ListNull(types.StringType),
			envValue:          "", // not set
			wantIsNull:        true,
			wantValues:        nil,
			expectValueChange: false,
		},
		{
			name:     "does not override when field already set",
			initial:  types.ListValueMust(types.StringType, []attr.Value{types.StringValue("preset")}),
			envValue: "one;two",
			// remains the preset list
			wantIsNull:        false,
			wantValues:        []string{"preset"},
			expectValueChange: false,
		},
		{
			name:              "preserves whitespace and empty items by design",
			initial:           types.ListNull(types.StringType),
			envValue:          "a; b ;c;;d",
			wantIsNull:        false,
			wantValues:        []string{"a", " b ", "c", "", "d"},
			expectValueChange: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envValue == "" {
				t.Setenv("A_IDS", "")
			} else {
				t.Setenv("A_IDS", tt.envValue)
			}

			m := &L{IDs: tt.initial}
			before := m.IDs

			setFieldDefaultsFromEnv(m)

			got := m.IDs

			if got.IsNull() != tt.wantIsNull {
				t.Fatalf("IsNull mismatch: want %v, got %v", tt.wantIsNull, got.IsNull())
			}

			if !got.IsNull() {
				if values := listToStrings(t, got); len(values) != len(tt.wantValues) {
					t.Fatalf("list length mismatch: want %d, got %d (%v)", len(tt.wantValues), len(values), values)
				} else {
					for i, v := range values {
						if v != tt.wantValues[i] {
							t.Fatalf("value[%d] mismatch: want %q, got %q", i, tt.wantValues[i], v)
						}
					}
				}
			}

			changed := !before.Equal(got)
			if tt.expectValueChange != changed {
				t.Fatalf("change mismatch: want changed=%v, got %v (before=%v, after=%v)", tt.expectValueChange, changed, before, got)
			}
		})
	}
}

// Test precedence across multiple env vars for a list field
func Test_setFieldDefaultsFromEnv_ListFieldPrecedence(t *testing.T) {
	type L2 struct {
		IDs types.List `fromenv:"A_IDS,B_IDS"`
	}

	// First non-empty should win
	t.Setenv("A_IDS", "a1;a2")
	t.Setenv("B_IDS", "b1;b2")

	m := &L2{IDs: types.ListNull(types.StringType)}
	setFieldDefaultsFromEnv(m)
	got := listToStrings(t, m.IDs)
	if len(got) != 2 || got[0] != "a1" || got[1] != "a2" {
		t.Fatalf("expected values from A_IDS, got %v", got)
	}

	// Fallback when first is empty
	t.Setenv("A_IDS", "")
	t.Setenv("B_IDS", "b1;b2")
	m2 := &L2{IDs: types.ListNull(types.StringType)}
	setFieldDefaultsFromEnv(m2)
	got2 := listToStrings(t, m2.IDs)
	if len(got2) != 2 || got2[0] != "b1" || got2[1] != "b2" {
		t.Fatalf("expected values from B_IDS when A_IDS empty, got %v", got2)
	}
}

// String default should populate when null and not override when already set.
func Test_setDefaultValueFromStructTags_StringField(t *testing.T) {
	type S struct {
		Name  types.String `defaultvalue:"foo"`
		NoTag types.String
	}

	// populates when null
	m := &S{Name: types.StringNull(), NoTag: types.StringNull()}
	setDefaultValueFromStructTags(m)
	if m.Name.IsNull() || m.Name.ValueString() != "foo" {
		t.Fatalf("expected Name to default to 'foo', got %v", m.Name)
	}
	if !m.NoTag.IsNull() {
		t.Fatalf("expected NoTag (no defaultvalue tag) to remain null, got %v", m.NoTag)
	}

	// does not override preset
	m2 := &S{Name: types.StringValue("preset")}
	setDefaultValueFromStructTags(m2)
	if m2.Name.IsNull() || m2.Name.ValueString() != "preset" {
		t.Fatalf("expected preset value to remain, got %v", m2.Name)
	}
}

// List default should split on comma and preserve whitespace/empties by design.
func Test_setDefaultValueFromStructTags_ListField(t *testing.T) {
	type L struct {
		IDs types.List `defaultvalue:"a,b, c,,d"`
	}

	m := &L{IDs: types.ListNull(types.StringType)}
	setDefaultValueFromStructTags(m)
	if m.IDs.IsNull() {
		t.Fatalf("expected IDs to be set from defaults")
	}
	got := listToStrings(t, m.IDs)
	want := []string{"a", "b", " c", "", "d"}
	if len(got) != len(want) {
		t.Fatalf("length mismatch: want %d, got %d (%v)", len(want), len(got), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("value[%d] mismatch: want %q, got %q", i, want[i], got[i])
		}
	}

	// does not override preset
	m2 := &L{IDs: types.ListValueMust(types.StringType, []attr.Value{types.StringValue("preset")})}
	setDefaultValueFromStructTags(m2)
	got2 := listToStrings(t, m2.IDs)
	if len(got2) != 1 || got2[0] != "preset" {
		t.Fatalf("expected preset list to remain, got %v", got2)
	}
}

// Bool default should parse valid values and skip invalid ones.
func Test_setDefaultValueFromStructTags_BoolField(t *testing.T) {
	t.Run("true", func(t *testing.T) {
		type B struct {
			Enabled types.Bool `defaultvalue:"true"`
		}
		m := &B{Enabled: types.BoolNull()}
		setDefaultValueFromStructTags(m)
		if m.Enabled.IsNull() || !m.Enabled.ValueBool() {
			t.Fatalf("expected Enabled=true, got %v", m.Enabled)
		}
	})

	t.Run("one-as-true", func(t *testing.T) {
		type B struct {
			Enabled types.Bool `defaultvalue:"1"`
		}
		m := &B{Enabled: types.BoolNull()}
		setDefaultValueFromStructTags(m)
		if m.Enabled.IsNull() || !m.Enabled.ValueBool() {
			t.Fatalf("expected Enabled=true from '1', got %v", m.Enabled)
		}
	})

	t.Run("invalid-skips", func(t *testing.T) {
		type B struct {
			Enabled types.Bool `defaultvalue:"invalid"`
		}
		m := &B{Enabled: types.BoolNull()}
		setDefaultValueFromStructTags(m)
		if !m.Enabled.IsNull() {
			t.Fatalf("expected Enabled to remain null on invalid default, got %v", m.Enabled)
		}
	})

	t.Run("does-not-override", func(t *testing.T) {
		type B struct {
			Enabled types.Bool `defaultvalue:"false"`
		}
		m := &B{Enabled: types.BoolValue(true)}
		setDefaultValueFromStructTags(m)
		if m.Enabled.IsNull() || !m.Enabled.ValueBool() {
			t.Fatalf("expected preset true to remain, got %v", m.Enabled)
		}
	})
}
