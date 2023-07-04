// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package alzvalidators_test

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/matt-FFFFFF/terraform-provider-alz/internal/alzvalidators"
)

func TestArmTypeResourceIds(t *testing.T) {
	t.Parallel()

	type testCase struct {
		rid       types.String
		validator validator.String
		expErrors int
	}

	testCases := map[string]testCase{
		"simple-match": {
			rid: types.StringValue("/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/foo"),
			validator: alzvalidators.ArmTypeResourceId(
				"Microsoft.Resources",
				"resourceGroups",
			),
			expErrors: 0,
		},
		"mg-match": {
			rid: types.StringValue("/providers/Microsoft.Management/managementGroups/foo/providers/Microsoft.Authorization/policyDefinitions/foo"),
			validator: alzvalidators.ArmTypeResourceId(
				"Microsoft.Authorization",
				"policyDefinitions",
			),
			expErrors: 0,
		},
		"subresource-match": {
			rid: types.StringValue("/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/foo/providers/Microsoft.Network/virtualNetworks/bar/subnets/baz"),
			validator: alzvalidators.ArmTypeResourceId(
				"Microsoft.Network",
				"virtualNetworks/subnets",
			),
			expErrors: 0,
		},
	}

	for name, test := range testCases {
		name, test := name, test
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			req := validator.StringRequest{
				ConfigValue: test.rid,
			}
			res := validator.StringResponse{}
			test.validator.ValidateString(context.TODO(), req, &res)

			if test.expErrors > 0 && !res.Diagnostics.HasError() {
				t.Fatalf("expected %d error(s), got none", test.expErrors)
			}

			if test.expErrors > 0 && test.expErrors != res.Diagnostics.ErrorsCount() {
				t.Fatalf("expected %d error(s), got %d: %v", test.expErrors, res.Diagnostics.ErrorsCount(), res.Diagnostics)
			}

			if test.expErrors == 0 && res.Diagnostics.HasError() {
				t.Fatalf("expected no error(s), got %d: %v", res.Diagnostics.ErrorsCount(), res.Diagnostics)
			}
		})
	}
}
