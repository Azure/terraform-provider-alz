// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package alzvalidators

import (
	"context"
	"fmt"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	"github.com/hashicorp/terraform-plugin-framework-validators/helpers/validatordiag"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

var _ validator.String = armResourceIdNamespaceTypeValidator{}

// armResourceIdValidator validates that a string Attribute's value matches the specified regular expression.
type armResourceIdNamespaceTypeValidator struct {
	armtype   string
	namespace string
}

// Description describes the validation in plain text formatting.
func (validator armResourceIdNamespaceTypeValidator) Description(_ context.Context) string {
	return fmt.Sprintf("Value must be ARM resource id in namespace '%s', of type, '%s'", validator.namespace, validator.armtype)
}

// MarkdownDescription describes the validation in Markdown formatting.
func (validator armResourceIdNamespaceTypeValidator) MarkdownDescription(ctx context.Context) string {
	return validator.Description(ctx)
}

// Validate performs the validation.
func (v armResourceIdNamespaceTypeValidator) ValidateString(ctx context.Context, request validator.StringRequest, response *validator.StringResponse) {
	if request.ConfigValue.IsNull() || request.ConfigValue.IsUnknown() {
		return
	}

	value := request.ConfigValue.ValueString()
	rt, err := arm.ParseResourceType(value)
	if err != nil || !strings.EqualFold(rt.Namespace, v.namespace) || !strings.EqualFold(rt.Type, v.armtype) {
		response.Diagnostics.Append(validatordiag.InvalidAttributeValueMatchDiagnostic(
			request.Path,
			v.Description(ctx),
			value,
		))
	}
}

// ArmTypeResourceId returns an AttributeValidator which ensures that any configured
// attribute value:
//
//   - Is a valid ARM resource id
//   - Matches the given namespace and resource type
//
// Null (unconfigured) and unknown (known after apply) values are skipped.
func ArmResourceIdTypeNamespace(ns, t string) validator.String {
	return armResourceIdNamespaceTypeValidator{
		armtype:   t,
		namespace: ns,
	}
}
