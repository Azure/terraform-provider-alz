// Code generated by terraform-plugin-framework-generator DO NOT EDIT.

package gen

import (
	"context"
	"fmt"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
)

func AlzProviderSchema(ctx context.Context) schema.Schema {
	return schema.Schema{
		Attributes: map[string]schema.Attribute{
			"auxiliary_tenant_ids": schema.ListAttribute{
				ElementType:         types.StringType,
				Optional:            true,
				Description:         "A list of auxiliary tenant ids which should be used. If not specified, value will be attempted to be read from the `ARM_AUXILIARY_TENANT_IDS` environment variable. When configuring from the environment, use a semicolon as a delimiter.",
				MarkdownDescription: "A list of auxiliary tenant ids which should be used. If not specified, value will be attempted to be read from the `ARM_AUXILIARY_TENANT_IDS` environment variable. When configuring from the environment, use a semicolon as a delimiter.",
			},
			"client_certificate_password": schema.StringAttribute{
				Optional:            true,
				Sensitive:           true,
				Description:         "The password associated with the client certificate. For use when authenticating as a service principal using a client certificate. If not specified, value will be attempted to be read from the `ARM_CLIENT_CERTIFICATE_PASSWORD` environment variable.",
				MarkdownDescription: "The password associated with the client certificate. For use when authenticating as a service principal using a client certificate. If not specified, value will be attempted to be read from the `ARM_CLIENT_CERTIFICATE_PASSWORD` environment variable.",
			},
			"client_certificate_path": schema.StringAttribute{
				Optional:            true,
				Description:         "The path to the client certificate associated with the service principal for use when authenticating as a service principal using a client certificate. If not specified, value will be attempted to be read from the `ARM_CLIENT_CERTIFICATE_PATH` environment variable.",
				MarkdownDescription: "The path to the client certificate associated with the service principal for use when authenticating as a service principal using a client certificate. If not specified, value will be attempted to be read from the `ARM_CLIENT_CERTIFICATE_PATH` environment variable.",
			},
			"client_id": schema.StringAttribute{
				Optional:            true,
				Description:         "The client id which should be used. For use when authenticating as a service principal. If not specified, value will be attempted to be read from the `ARM_CLIENT_ID` environment variable.",
				MarkdownDescription: "The client id which should be used. For use when authenticating as a service principal. If not specified, value will be attempted to be read from the `ARM_CLIENT_ID` environment variable.",
			},
			"client_secret": schema.StringAttribute{
				Optional:            true,
				Sensitive:           true,
				Description:         "The client secret which should be used. For use when authenticating as a service principal using a client secret. If not specified, value will be attempted to be read from the `ARM_CLIENT_SECRET` environment variable.",
				MarkdownDescription: "The client secret which should be used. For use when authenticating as a service principal using a client secret. If not specified, value will be attempted to be read from the `ARM_CLIENT_SECRET` environment variable.",
			},
			"environment": schema.StringAttribute{
				Optional: true,
				Validators: []validator.String{
					stringvalidator.OneOf("public", "usgovernment", "china"),
				},
			},
			"library_fetch_dependencies": schema.BoolAttribute{
				Optional:            true,
				Description:         "Whether to automatically fetch dependencies for the library. This option reads the `alz_library_metadata.json` file in any supplied library and will recursively download dependent libraries. Default is `true`.",
				MarkdownDescription: "Whether to automatically fetch dependencies for the library. This option reads the `alz_library_metadata.json` file in any supplied library and will recursively download dependent libraries. Default is `true`.",
			},
			"library_overwrite_enabled": schema.BoolAttribute{
				Optional:            true,
				Description:         "Whether to allow overwriting of the library by other lib directories. Default is `false`.",
				MarkdownDescription: "Whether to allow overwriting of the library by other lib directories. Default is `false`.",
			},
			"library_references": schema.ListNestedAttribute{
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"custom_url": schema.StringAttribute{
							Optional:            true,
							Sensitive:           true,
							Description:         "A custom path/URL to the library to use. Conflicts with `path` and `ref`. For supported protocols, see [go-getter](https://pkg.go.dev/github.com/hashicorp/go-getter/v2). Value is marked sensitive as may contain secrets.",
							MarkdownDescription: "A custom path/URL to the library to use. Conflicts with `path` and `ref`. For supported protocols, see [go-getter](https://pkg.go.dev/github.com/hashicorp/go-getter/v2). Value is marked sensitive as may contain secrets.",
							Validators: []validator.String{
								stringvalidator.ConflictsWith(path.MatchRelative().AtParent().AtName("path")),
								stringvalidator.ConflictsWith(path.MatchRelative().AtParent().AtName("ref")),
							},
						},
						"path": schema.StringAttribute{
							Optional:            true,
							Description:         "The path in the ALZ Library, e.g. `platform/alz`. Also requires `ref`. Conflicts with `custom_url`.",
							MarkdownDescription: "The path in the ALZ Library, e.g. `platform/alz`. Also requires `ref`. Conflicts with `custom_url`.",
							Validators: []validator.String{
								stringvalidator.ConflictsWith(path.MatchRelative().AtParent().AtName("custom_url")),
								stringvalidator.AlsoRequires(path.MatchRelative().AtParent().AtName("ref")),
							},
						},
						"ref": schema.StringAttribute{
							Optional:            true,
							Description:         "This is the version of the library to use, e.g. `2024.07.5`. Also requires `path`. Conflicts with `custom_url`.",
							MarkdownDescription: "This is the version of the library to use, e.g. `2024.07.5`. Also requires `path`. Conflicts with `custom_url`.",
							Validators: []validator.String{
								stringvalidator.ConflictsWith(path.MatchRelative().AtParent().AtName("custom_url")),
								stringvalidator.AlsoRequires(path.MatchRelative().AtParent().AtName("path")),
							},
						},
					},
					CustomType: LibraryReferencesType{
						ObjectType: types.ObjectType{
							AttrTypes: LibraryReferencesValue{}.AttributeTypes(ctx),
						},
					},
				},
				Optional:            true,
				Description:         "A list of references to the [ALZ library](https://aka.ms/alz/library) to use. Each reference should either contain the `path` (e.g. `platform/alz`) and the `ref` (e.g. `2024.03.5`), or a `custom_url` to be supplied to go-getter.\nIf this value is not specified, the default value will be used, which is:\n\n```terraform\nalz_library_references = [\n  { path = \"platform/alz\", tag = \"2024.10.1\" },\n]\n```\n\n",
				MarkdownDescription: "A list of references to the [ALZ library](https://aka.ms/alz/library) to use. Each reference should either contain the `path` (e.g. `platform/alz`) and the `ref` (e.g. `2024.03.5`), or a `custom_url` to be supplied to go-getter.\nIf this value is not specified, the default value will be used, which is:\n\n```terraform\nalz_library_references = [\n  { path = \"platform/alz\", tag = \"2024.10.1\" },\n]\n```\n\n",
				Validators: []validator.List{
					listvalidator.UniqueValues(),
				},
			},
			"oidc_request_token": schema.StringAttribute{
				Optional:            true,
				Sensitive:           true,
				Description:         "The bearer token for the request to the OIDC provider. For use when authenticating using OpenID Connect. If not specified, value will be attempted to be read from the first non-empty value of the `ARM_OIDC_REQUEST_TOKEN` and `ACTIONS_ID_TOKEN_REQUEST_TOKEN` environment variables.",
				MarkdownDescription: "The bearer token for the request to the OIDC provider. For use when authenticating using OpenID Connect. If not specified, value will be attempted to be read from the first non-empty value of the `ARM_OIDC_REQUEST_TOKEN` and `ACTIONS_ID_TOKEN_REQUEST_TOKEN` environment variables.",
			},
			"oidc_request_url": schema.StringAttribute{
				Optional:            true,
				Description:         "The URL for the OIDC provider from which to request an id token. For use when authenticating as a service principal using OpenID Connect. If not specified, value will be attempted to be read from the first non-empty value of the `ARM_OIDC_REQUEST_URL` and `ACTIONS_ID_TOKEN_REQUEST_URL` environment variables.",
				MarkdownDescription: "The URL for the OIDC provider from which to request an id token. For use when authenticating as a service principal using OpenID Connect. If not specified, value will be attempted to be read from the first non-empty value of the `ARM_OIDC_REQUEST_URL` and `ACTIONS_ID_TOKEN_REQUEST_URL` environment variables.",
			},
			"oidc_token": schema.StringAttribute{
				Optional:            true,
				Sensitive:           true,
				Description:         "The OIDC id token for use when authenticating as a service principal using OpenID Connect. If not specified, value will be attempted to be read from the `ARM_OIDC_TOKEN` environment variable.",
				MarkdownDescription: "The OIDC id token for use when authenticating as a service principal using OpenID Connect. If not specified, value will be attempted to be read from the `ARM_OIDC_TOKEN` environment variable.",
			},
			"oidc_token_file_path": schema.StringAttribute{
				Optional:            true,
				Description:         "The path to a file containing an OIDC id token for use when authenticating using OpenID Connect. If not specified, value will be attempted to be read from the `ARM_OIDC_TOKEN_FILE_PATH` environment variable.",
				MarkdownDescription: "The path to a file containing an OIDC id token for use when authenticating using OpenID Connect. If not specified, value will be attempted to be read from the `ARM_OIDC_TOKEN_FILE_PATH` environment variable.",
			},
			"skip_provider_registration": schema.BoolAttribute{
				Optional:            true,
				Description:         "Should the provider skip registering all of the resource providers that it supports, if they're not already registered? Default is `false`. If not specified, value will be attempted to be read from the `ARM_SKIP_PROVIDER_REGISTRATION` environment variable.",
				MarkdownDescription: "Should the provider skip registering all of the resource providers that it supports, if they're not already registered? Default is `false`. If not specified, value will be attempted to be read from the `ARM_SKIP_PROVIDER_REGISTRATION` environment variable.",
			},
			"tenant_id": schema.StringAttribute{
				Optional:            true,
				Description:         "The Tenant ID which should be used. If not specified, value will be attempted to be read from the `ARM_TENANT_ID` environment variable.",
				MarkdownDescription: "The Tenant ID which should be used. If not specified, value will be attempted to be read from the `ARM_TENANT_ID` environment variable.",
			},
			"use_cli": schema.BoolAttribute{
				Optional:            true,
				Description:         "Allow Azure CLI to be used for authentication. Default is `true`. If not specified, value will be attempted to be read from the `ARM_USE_CLI` environment variable.",
				MarkdownDescription: "Allow Azure CLI to be used for authentication. Default is `true`. If not specified, value will be attempted to be read from the `ARM_USE_CLI` environment variable.",
			},
			"use_msi": schema.BoolAttribute{
				Optional:            true,
				Description:         "Allow managed service identity to be used for authentication. Default is `false`. If not specified, value will be attempted to be read from the `ARM_USE_MSI` environment variable.",
				MarkdownDescription: "Allow managed service identity to be used for authentication. Default is `false`. If not specified, value will be attempted to be read from the `ARM_USE_MSI` environment variable.",
			},
			"use_oidc": schema.BoolAttribute{
				Optional:            true,
				Description:         "Allow OpenID Connect to be used for authentication. Default is `false`. If not specified, value will be attempted to be read from the `ARM_USE_OIDC` environment variable.",
				MarkdownDescription: "Allow OpenID Connect to be used for authentication. Default is `false`. If not specified, value will be attempted to be read from the `ARM_USE_OIDC` environment variable.",
			},
		},
		Description:         "ALZ provider to generate archetype data for use with the ALZ Terraform module.",
		MarkdownDescription: "ALZ provider to generate archetype data for use with the ALZ Terraform module.",
	}
}

type AlzModel struct {
	AuxiliaryTenantIds        types.List   `tfsdk:"auxiliary_tenant_ids"`
	ClientCertificatePassword types.String `tfsdk:"client_certificate_password"`
	ClientCertificatePath     types.String `tfsdk:"client_certificate_path"`
	ClientId                  types.String `tfsdk:"client_id"`
	ClientSecret              types.String `tfsdk:"client_secret"`
	Environment               types.String `tfsdk:"environment"`
	LibraryFetchDependencies  types.Bool   `tfsdk:"library_fetch_dependencies"`
	LibraryOverwriteEnabled   types.Bool   `tfsdk:"library_overwrite_enabled"`
	LibraryReferences         types.List   `tfsdk:"library_references"`
	OidcRequestToken          types.String `tfsdk:"oidc_request_token"`
	OidcRequestUrl            types.String `tfsdk:"oidc_request_url"`
	OidcToken                 types.String `tfsdk:"oidc_token"`
	OidcTokenFilePath         types.String `tfsdk:"oidc_token_file_path"`
	SkipProviderRegistration  types.Bool   `tfsdk:"skip_provider_registration"`
	TenantId                  types.String `tfsdk:"tenant_id"`
	UseCli                    types.Bool   `tfsdk:"use_cli"`
	UseMsi                    types.Bool   `tfsdk:"use_msi"`
	UseOidc                   types.Bool   `tfsdk:"use_oidc"`
}

var _ basetypes.ObjectTypable = LibraryReferencesType{}

type LibraryReferencesType struct {
	basetypes.ObjectType
}

func (t LibraryReferencesType) Equal(o attr.Type) bool {
	other, ok := o.(LibraryReferencesType)

	if !ok {
		return false
	}

	return t.ObjectType.Equal(other.ObjectType)
}

func (t LibraryReferencesType) String() string {
	return "LibraryReferencesType"
}

func (t LibraryReferencesType) ValueFromObject(ctx context.Context, in basetypes.ObjectValue) (basetypes.ObjectValuable, diag.Diagnostics) {
	var diags diag.Diagnostics

	attributes := in.Attributes()

	customUrlAttribute, ok := attributes["custom_url"]

	if !ok {
		diags.AddError(
			"Attribute Missing",
			`custom_url is missing from object`)

		return nil, diags
	}

	customUrlVal, ok := customUrlAttribute.(basetypes.StringValue)

	if !ok {
		diags.AddError(
			"Attribute Wrong Type",
			fmt.Sprintf(`custom_url expected to be basetypes.StringValue, was: %T`, customUrlAttribute))
	}

	pathAttribute, ok := attributes["path"]

	if !ok {
		diags.AddError(
			"Attribute Missing",
			`path is missing from object`)

		return nil, diags
	}

	pathVal, ok := pathAttribute.(basetypes.StringValue)

	if !ok {
		diags.AddError(
			"Attribute Wrong Type",
			fmt.Sprintf(`path expected to be basetypes.StringValue, was: %T`, pathAttribute))
	}

	refAttribute, ok := attributes["ref"]

	if !ok {
		diags.AddError(
			"Attribute Missing",
			`ref is missing from object`)

		return nil, diags
	}

	refVal, ok := refAttribute.(basetypes.StringValue)

	if !ok {
		diags.AddError(
			"Attribute Wrong Type",
			fmt.Sprintf(`ref expected to be basetypes.StringValue, was: %T`, refAttribute))
	}

	if diags.HasError() {
		return nil, diags
	}

	return LibraryReferencesValue{
		CustomUrl: customUrlVal,
		Path:      pathVal,
		Ref:       refVal,
		state:     attr.ValueStateKnown,
	}, diags
}

func NewLibraryReferencesValueNull() LibraryReferencesValue {
	return LibraryReferencesValue{
		state: attr.ValueStateNull,
	}
}

func NewLibraryReferencesValueUnknown() LibraryReferencesValue {
	return LibraryReferencesValue{
		state: attr.ValueStateUnknown,
	}
}

func NewLibraryReferencesValue(attributeTypes map[string]attr.Type, attributes map[string]attr.Value) (LibraryReferencesValue, diag.Diagnostics) {
	var diags diag.Diagnostics

	// Reference: https://github.com/hashicorp/terraform-plugin-framework/issues/521
	ctx := context.Background()

	for name, attributeType := range attributeTypes {
		attribute, ok := attributes[name]

		if !ok {
			diags.AddError(
				"Missing LibraryReferencesValue Attribute Value",
				"While creating a LibraryReferencesValue value, a missing attribute value was detected. "+
					"A LibraryReferencesValue must contain values for all attributes, even if null or unknown. "+
					"This is always an issue with the provider and should be reported to the provider developers.\n\n"+
					fmt.Sprintf("LibraryReferencesValue Attribute Name (%s) Expected Type: %s", name, attributeType.String()),
			)

			continue
		}

		if !attributeType.Equal(attribute.Type(ctx)) {
			diags.AddError(
				"Invalid LibraryReferencesValue Attribute Type",
				"While creating a LibraryReferencesValue value, an invalid attribute value was detected. "+
					"A LibraryReferencesValue must use a matching attribute type for the value. "+
					"This is always an issue with the provider and should be reported to the provider developers.\n\n"+
					fmt.Sprintf("LibraryReferencesValue Attribute Name (%s) Expected Type: %s\n", name, attributeType.String())+
					fmt.Sprintf("LibraryReferencesValue Attribute Name (%s) Given Type: %s", name, attribute.Type(ctx)),
			)
		}
	}

	for name := range attributes {
		_, ok := attributeTypes[name]

		if !ok {
			diags.AddError(
				"Extra LibraryReferencesValue Attribute Value",
				"While creating a LibraryReferencesValue value, an extra attribute value was detected. "+
					"A LibraryReferencesValue must not contain values beyond the expected attribute types. "+
					"This is always an issue with the provider and should be reported to the provider developers.\n\n"+
					fmt.Sprintf("Extra LibraryReferencesValue Attribute Name: %s", name),
			)
		}
	}

	if diags.HasError() {
		return NewLibraryReferencesValueUnknown(), diags
	}

	customUrlAttribute, ok := attributes["custom_url"]

	if !ok {
		diags.AddError(
			"Attribute Missing",
			`custom_url is missing from object`)

		return NewLibraryReferencesValueUnknown(), diags
	}

	customUrlVal, ok := customUrlAttribute.(basetypes.StringValue)

	if !ok {
		diags.AddError(
			"Attribute Wrong Type",
			fmt.Sprintf(`custom_url expected to be basetypes.StringValue, was: %T`, customUrlAttribute))
	}

	pathAttribute, ok := attributes["path"]

	if !ok {
		diags.AddError(
			"Attribute Missing",
			`path is missing from object`)

		return NewLibraryReferencesValueUnknown(), diags
	}

	pathVal, ok := pathAttribute.(basetypes.StringValue)

	if !ok {
		diags.AddError(
			"Attribute Wrong Type",
			fmt.Sprintf(`path expected to be basetypes.StringValue, was: %T`, pathAttribute))
	}

	refAttribute, ok := attributes["ref"]

	if !ok {
		diags.AddError(
			"Attribute Missing",
			`ref is missing from object`)

		return NewLibraryReferencesValueUnknown(), diags
	}

	refVal, ok := refAttribute.(basetypes.StringValue)

	if !ok {
		diags.AddError(
			"Attribute Wrong Type",
			fmt.Sprintf(`ref expected to be basetypes.StringValue, was: %T`, refAttribute))
	}

	if diags.HasError() {
		return NewLibraryReferencesValueUnknown(), diags
	}

	return LibraryReferencesValue{
		CustomUrl: customUrlVal,
		Path:      pathVal,
		Ref:       refVal,
		state:     attr.ValueStateKnown,
	}, diags
}

func NewLibraryReferencesValueMust(attributeTypes map[string]attr.Type, attributes map[string]attr.Value) LibraryReferencesValue {
	object, diags := NewLibraryReferencesValue(attributeTypes, attributes)

	if diags.HasError() {
		// This could potentially be added to the diag package.
		diagsStrings := make([]string, 0, len(diags))

		for _, diagnostic := range diags {
			diagsStrings = append(diagsStrings, fmt.Sprintf(
				"%s | %s | %s",
				diagnostic.Severity(),
				diagnostic.Summary(),
				diagnostic.Detail()))
		}

		panic("NewLibraryReferencesValueMust received error(s): " + strings.Join(diagsStrings, "\n"))
	}

	return object
}

func (t LibraryReferencesType) ValueFromTerraform(ctx context.Context, in tftypes.Value) (attr.Value, error) {
	if in.Type() == nil {
		return NewLibraryReferencesValueNull(), nil
	}

	if !in.Type().Equal(t.TerraformType(ctx)) {
		return nil, fmt.Errorf("expected %s, got %s", t.TerraformType(ctx), in.Type())
	}

	if !in.IsKnown() {
		return NewLibraryReferencesValueUnknown(), nil
	}

	if in.IsNull() {
		return NewLibraryReferencesValueNull(), nil
	}

	attributes := map[string]attr.Value{}

	val := map[string]tftypes.Value{}

	err := in.As(&val)

	if err != nil {
		return nil, err
	}

	for k, v := range val {
		a, err := t.AttrTypes[k].ValueFromTerraform(ctx, v)

		if err != nil {
			return nil, err
		}

		attributes[k] = a
	}

	return NewLibraryReferencesValueMust(LibraryReferencesValue{}.AttributeTypes(ctx), attributes), nil
}

func (t LibraryReferencesType) ValueType(ctx context.Context) attr.Value {
	return LibraryReferencesValue{}
}

var _ basetypes.ObjectValuable = LibraryReferencesValue{}

type LibraryReferencesValue struct {
	CustomUrl basetypes.StringValue `tfsdk:"custom_url"`
	Path      basetypes.StringValue `tfsdk:"path"`
	Ref       basetypes.StringValue `tfsdk:"ref"`
	state     attr.ValueState
}

func (v LibraryReferencesValue) ToTerraformValue(ctx context.Context) (tftypes.Value, error) {
	attrTypes := make(map[string]tftypes.Type, 3)

	var val tftypes.Value
	var err error

	attrTypes["custom_url"] = basetypes.StringType{}.TerraformType(ctx)
	attrTypes["path"] = basetypes.StringType{}.TerraformType(ctx)
	attrTypes["ref"] = basetypes.StringType{}.TerraformType(ctx)

	objectType := tftypes.Object{AttributeTypes: attrTypes}

	switch v.state {
	case attr.ValueStateKnown:
		vals := make(map[string]tftypes.Value, 3)

		val, err = v.CustomUrl.ToTerraformValue(ctx)

		if err != nil {
			return tftypes.NewValue(objectType, tftypes.UnknownValue), err
		}

		vals["custom_url"] = val

		val, err = v.Path.ToTerraformValue(ctx)

		if err != nil {
			return tftypes.NewValue(objectType, tftypes.UnknownValue), err
		}

		vals["path"] = val

		val, err = v.Ref.ToTerraformValue(ctx)

		if err != nil {
			return tftypes.NewValue(objectType, tftypes.UnknownValue), err
		}

		vals["ref"] = val

		if err := tftypes.ValidateValue(objectType, vals); err != nil {
			return tftypes.NewValue(objectType, tftypes.UnknownValue), err
		}

		return tftypes.NewValue(objectType, vals), nil
	case attr.ValueStateNull:
		return tftypes.NewValue(objectType, nil), nil
	case attr.ValueStateUnknown:
		return tftypes.NewValue(objectType, tftypes.UnknownValue), nil
	default:
		panic(fmt.Sprintf("unhandled Object state in ToTerraformValue: %s", v.state))
	}
}

func (v LibraryReferencesValue) IsNull() bool {
	return v.state == attr.ValueStateNull
}

func (v LibraryReferencesValue) IsUnknown() bool {
	return v.state == attr.ValueStateUnknown
}

func (v LibraryReferencesValue) String() string {
	return "LibraryReferencesValue"
}

func (v LibraryReferencesValue) ToObjectValue(ctx context.Context) (basetypes.ObjectValue, diag.Diagnostics) {
	var diags diag.Diagnostics

	attributeTypes := map[string]attr.Type{
		"custom_url": basetypes.StringType{},
		"path":       basetypes.StringType{},
		"ref":        basetypes.StringType{},
	}

	if v.IsNull() {
		return types.ObjectNull(attributeTypes), diags
	}

	if v.IsUnknown() {
		return types.ObjectUnknown(attributeTypes), diags
	}

	objVal, diags := types.ObjectValue(
		attributeTypes,
		map[string]attr.Value{
			"custom_url": v.CustomUrl,
			"path":       v.Path,
			"ref":        v.Ref,
		})

	return objVal, diags
}

func (v LibraryReferencesValue) Equal(o attr.Value) bool {
	other, ok := o.(LibraryReferencesValue)

	if !ok {
		return false
	}

	if v.state != other.state {
		return false
	}

	if v.state != attr.ValueStateKnown {
		return true
	}

	if !v.CustomUrl.Equal(other.CustomUrl) {
		return false
	}

	if !v.Path.Equal(other.Path) {
		return false
	}

	if !v.Ref.Equal(other.Ref) {
		return false
	}

	return true
}

func (v LibraryReferencesValue) Type(ctx context.Context) attr.Type {
	return LibraryReferencesType{
		basetypes.ObjectType{
			AttrTypes: v.AttributeTypes(ctx),
		},
	}
}

func (v LibraryReferencesValue) AttributeTypes(ctx context.Context) map[string]attr.Type {
	return map[string]attr.Type{
		"custom_url": basetypes.StringType{},
		"path":       basetypes.StringType{},
		"ref":        basetypes.StringType{},
	}
}
