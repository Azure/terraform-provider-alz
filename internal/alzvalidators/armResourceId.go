// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package alzvalidators

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	"github.com/hashicorp/terraform-plugin-framework-validators/helpers/validatordiag"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

var _ validator.String = armResourceIdValidator{}

// armResourceIdValidator validates that a string Attribute's value matches the specified regular expression.
type armResourceIdValidator struct{}

// Description describes the validation in plain text formatting.
func (validator armResourceIdValidator) Description(_ context.Context) string {
	return "Value must be an ARM resource id"
}

// MarkdownDescription describes the validation in Markdown formatting.
func (validator armResourceIdValidator) MarkdownDescription(ctx context.Context) string {
	return validator.Description(ctx)
}

// Validate performs the validation.
func (v armResourceIdValidator) ValidateString(ctx context.Context, request validator.StringRequest, response *validator.StringResponse) {
	if request.ConfigValue.IsNull() || request.ConfigValue.IsUnknown() {
		return
	}

	value := request.ConfigValue.ValueString()
	_, err := arm.ParseResourceID(value)
	if err != nil {
		response.Diagnostics.Append(validatordiag.InvalidAttributeValueMatchDiagnostic(
			request.Path,
			v.Description(ctx),
			value,
		))
	}
}

// ArmResourceId returns an AttributeValidator which ensures that any configured
// attribute value:
//
//   - Is a valid ARM resource id
//
// Null (unconfigured) and unknown (known after apply) values are skipped.
func ArmResourceId() validator.String {
	return armResourceIdValidator{}
}
