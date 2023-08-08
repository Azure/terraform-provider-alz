// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package provider

import (
	"context"
	"math/big"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/stretchr/testify/assert"
)

// testAccProtoV6ProviderFactories are used to instantiate a provider during
// acceptance testing. The factory function will be invoked for every Terraform
// CLI command executed to create a provider server to which the CLI can
// reattach.
// var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
// 	"alz": providerserver.NewProtocol6WithError(New("test")()),
// }

// testAccProtoV6ProviderFactoriesUnique is used to ensure that the provider instance used for
// each acceptance test is unique.
// This is necessary because this provider make use of state stored in the provider instance.
// See type AlzProvider.
func testAccProtoV6ProviderFactoriesUnique() map[string]func() (tfprotov6.ProviderServer, error) {
	return map[string]func() (tfprotov6.ProviderServer, error){
		"alz": providerserver.NewProtocol6WithError(New("test")()),
	}

}

// testAccPreCheck ensures that the environment is properly configured for acceptance testing.
func testAccPreCheck(t *testing.T) {
	// You can add code here to run prior to any test case execution, for example assertions
	// about the appropriate environment variables being set are common to see in a pre-check
	// function.
}

func TestGetFirstSetEnvVar(t *testing.T) {
	// Test when no environment variable is set
	_ = os.Unsetenv("VAR1")
	_ = os.Unsetenv("VAR2")
	_ = os.Unsetenv("VAR3")
	result := getFirstSetEnvVar("VAR1", "VAR2", "VAR3")
	assert.Equal(t, "", result)

	// Test when the first environment variable is set
	t.Setenv("VAR1", "value1")
	result = getFirstSetEnvVar("VAR1", "VAR2", "VAR3")
	assert.Equal(t, "value1", result)
	_ = os.Unsetenv("VAR1")

	// Test when the second environment variable is set
	t.Setenv("VAR2", "value2")
	result = getFirstSetEnvVar("VAR1", "VAR2", "VAR3")
	assert.Equal(t, "value2", result)
	os.Unsetenv("VAR2")

	// Test when the third environment variable is set
	t.Setenv("VAR3", "value3")
	result = getFirstSetEnvVar("VAR1", "VAR2", "VAR3")
	assert.Equal(t, "value3", result)
	os.Unsetenv("VAR3")

	// Test when multiple environment variables are set
	t.Setenv("VAR1", "value1")
	t.Setenv("VAR2", "value2")
	t.Setenv("VAR3", "value3")
	result = getFirstSetEnvVar("VAR1", "VAR2", "VAR3")
	assert.Equal(t, "value1", result)
	os.Unsetenv("VAR1")
	os.Unsetenv("VAR2")
	os.Unsetenv("VAR3")
}

func TestConfigureFromEnvironment(t *testing.T) {
	// Unset all environment variables
	os.Unsetenv("ARM_CLIENT_CERTIFICATE_PASSWORD")
	os.Unsetenv("ARM_CLIENT_CERTIFICATE_PATH")
	os.Unsetenv("ARM_CLIENT_ID")
	os.Unsetenv("ARM_CLIENT_SECRET")
	os.Unsetenv("ARM_ENVIRONMENT")
	os.Unsetenv("ARM_OIDC_REQUEST_TOKEN")
	os.Unsetenv("ARM_OIDC_REQUEST_URL")
	os.Unsetenv("ARM_OIDC_TOKEN")
	os.Unsetenv("ARM_OIDC_TOKEN_FILE_PATH")
	os.Unsetenv("ARM_TENANT_ID")
	os.Unsetenv("ARM_USE_CLI")
	os.Unsetenv("ARM_USE_MSI")
	os.Unsetenv("ARM_USE_OIDC")
	os.Unsetenv("ARM_SKIP_PROVIDER_REGISTRATION")
	os.Unsetenv("ACTIONS_ID_TOKEN_REQUEST_TOKEN")
	os.Unsetenv("ACTIONS_ID_TOKEN_REQUEST_URL")

	// Test when no environment variable is set
	data := &AlzProviderModel{}
	configureFromEnvironment(data)
	assert.True(t, data.ClientCertificatePassword.IsNull())
	assert.True(t, data.ClientCertificatePath.IsNull())
	assert.True(t, data.ClientId.IsNull())
	assert.True(t, data.ClientSecret.IsNull())
	assert.True(t, data.Environment.IsNull())
	assert.True(t, data.OidcRequestToken.IsNull())
	assert.True(t, data.OidcRequestUrl.IsNull())
	assert.True(t, data.OidcToken.IsNull())
	assert.True(t, data.OidcTokenFilePath.IsNull())
	assert.True(t, data.TenantId.IsNull())
	assert.True(t, data.UseCli.IsNull())
	assert.True(t, data.UseMsi.IsNull())
	assert.True(t, data.UseOidc.IsNull())
	assert.True(t, data.SkipProviderRegistration.IsNull())

	// Test when some environment variables are set
	t.Setenv("ARM_CLIENT_ID", "client_id")
	t.Setenv("ARM_CLIENT_SECRET", "client_secret")
	t.Setenv("ARM_TENANT_ID", "tenant_id")
	data = &AlzProviderModel{}
	configureFromEnvironment(data)
	assert.Equal(t, "", data.ClientCertificatePassword.ValueString())
	assert.Equal(t, "", data.ClientCertificatePath.ValueString())
	assert.Equal(t, "client_id", data.ClientId.ValueString())
	assert.Equal(t, "client_secret", data.ClientSecret.ValueString())
	assert.Equal(t, "", data.Environment.ValueString())
	assert.Equal(t, "", data.OidcRequestToken.ValueString())
	assert.Equal(t, "", data.OidcRequestUrl.ValueString())
	assert.Equal(t, "", data.OidcToken.ValueString())
	assert.Equal(t, "", data.OidcTokenFilePath.ValueString())
	assert.Equal(t, "tenant_id", data.TenantId.ValueString())
	assert.Equal(t, false, data.UseCli.ValueBool())
	assert.Equal(t, false, data.UseMsi.ValueBool())
	assert.Equal(t, false, data.UseOidc.ValueBool())
	assert.Equal(t, false, data.SkipProviderRegistration.ValueBool())
	os.Unsetenv("ARM_CLIENT_ID")
	os.Unsetenv("ARM_CLIENT_SECRET")
	os.Unsetenv("ARM_TENANT_ID")

	// Test when all environment variables are set
	t.Setenv("ARM_CLIENT_CERTIFICATE_PASSWORD", "password")
	t.Setenv("ARM_CLIENT_CERTIFICATE_PATH", "path")
	t.Setenv("ARM_CLIENT_ID", "client_id")
	t.Setenv("ARM_CLIENT_SECRET", "client_secret")
	t.Setenv("ARM_ENVIRONMENT", "environment")
	t.Setenv("ARM_OIDC_REQUEST_TOKEN", "request_token")
	t.Setenv("ARM_OIDC_REQUEST_URL", "request_url")
	t.Setenv("ARM_OIDC_TOKEN", "token")
	t.Setenv("ARM_OIDC_TOKEN_FILE_PATH", "token_file_path")
	t.Setenv("ARM_TENANT_ID", "tenant_id")
	t.Setenv("ARM_USE_CLI", "true")
	t.Setenv("ARM_USE_MSI", "true")
	t.Setenv("ARM_USE_OIDC", "true")
	t.Setenv("ARM_SKIP_PROVIDER_REGISTRATION", "true")
	data = &AlzProviderModel{}
	configureFromEnvironment(data)
	assert.Equal(t, "password", data.ClientCertificatePassword.ValueString())
	assert.Equal(t, "path", data.ClientCertificatePath.ValueString())
	assert.Equal(t, "client_id", data.ClientId.ValueString())
	assert.Equal(t, "client_secret", data.ClientSecret.ValueString())
	assert.Equal(t, "environment", data.Environment.ValueString())
	assert.Equal(t, "request_token", data.OidcRequestToken.ValueString())
	assert.Equal(t, "request_url", data.OidcRequestUrl.ValueString())
	assert.Equal(t, "token", data.OidcToken.ValueString())
	assert.Equal(t, "token_file_path", data.OidcTokenFilePath.ValueString())
	assert.Equal(t, "tenant_id", data.TenantId.ValueString())
	assert.Equal(t, true, data.UseCli.ValueBool())
	assert.Equal(t, true, data.UseMsi.ValueBool())
	assert.Equal(t, true, data.UseOidc.ValueBool())
	assert.Equal(t, true, data.SkipProviderRegistration.ValueBool())
	os.Unsetenv("ARM_CLIENT_CERTIFICATE_PASSWORD")
	os.Unsetenv("ARM_CLIENT_CERTIFICATE_PATH")
	os.Unsetenv("ARM_CLIENT_ID")
	os.Unsetenv("ARM_CLIENT_SECRET")
	os.Unsetenv("ARM_ENVIRONMENT")
	os.Unsetenv("ARM_OIDC_REQUEST_TOKEN")
	os.Unsetenv("ARM_OIDC_REQUEST_URL")
	os.Unsetenv("ARM_OIDC_TOKEN")
	os.Unsetenv("ARM_OIDC_TOKEN_FILE_PATH")
	os.Unsetenv("ARM_TENANT_ID")
	os.Unsetenv("ARM_USE_CLI")
	os.Unsetenv("ARM_USE_MSI")
	os.Unsetenv("ARM_USE_OIDC")
	os.Unsetenv("ARM_SKIP_PROVIDER_REGISTRATION")
}

func TestConfigureAuxTenants(t *testing.T) {
	// Test when no environment variable is set and data.AuxiliaryTenantIds is null
	ctx := context.Background()
	data := &AlzProviderModel{}
	diags := configureAuxTenants(ctx, data)
	assert.True(t, data.AuxiliaryTenantIds.IsNull())
	assert.Empty(t, diags)

	// Test when no environment variable is set and data.AuxiliaryTenantIds is not null
	auxTenants := []string{"tenant1", "tenant2"}
	lv, _ := types.ListValueFrom(ctx, types.StringType, auxTenants)
	data = &AlzProviderModel{AuxiliaryTenantIds: lv}
	diags = configureAuxTenants(context.Background(), data)
	assert.Truef(t, data.AuxiliaryTenantIds.Equal(lv), "Expected %v, got %v", lv, data.AuxiliaryTenantIds)
	assert.Empty(t, diags)

	// Test when ARM_AUXILIARY_TENANT_IDS environment variable is set and data.AuxiliaryTenantIds is null
	t.Setenv("ARM_AUXILIARY_TENANT_IDS", "tenant1;tenant2")
	data = &AlzProviderModel{}
	diags = configureAuxTenants(context.Background(), data)
	assert.True(t, data.AuxiliaryTenantIds.Equal(lv))
	assert.Empty(t, diags)
	_ = os.Unsetenv("ARM_AUXILIARY_TENANT_IDS")

	// Test when ARM_AUXILIARY_TENANT_IDS environment variable is set and data.AuxiliaryTenantIds is not null
	t.Setenv("ARM_AUXILIARY_TENANT_IDS", "tenant3;tenant4")
	data = &AlzProviderModel{AuxiliaryTenantIds: lv}
	diags = configureAuxTenants(context.Background(), data)
	assert.True(t, data.AuxiliaryTenantIds.Equal(lv))
	assert.Empty(t, diags)
	_ = os.Unsetenv("ARM_AUXILIARY_TENANT_IDS")
}

func TestConfigureAzIdentityEnvironment(t *testing.T) {
	// Test when no data fields are set
	data := &AlzProviderModel{}
	configureAzIdentityEnvironment(data)
	t.Setenv("AZURE_TENANT_ID", "")
	t.Setenv("AZURE_CLIENT_ID", "")
	t.Setenv("AZURE_CLIENT_SECRET", "")
	t.Setenv("AZURE_CLIENT_CERTIFICATE_PATH", "")
	t.Setenv("AZURE_CLIENT_CERTIFICATE_PASSWORD", "")
	t.Setenv("AZURE_ADDITIONALLY_ALLOWED_TENANTS", "")
	assert.Empty(t, os.Getenv("AZURE_TENANT_ID"))
	assert.Empty(t, os.Getenv("AZURE_CLIENT_ID"))
	assert.Empty(t, os.Getenv("AZURE_CLIENT_SECRET"))
	assert.Empty(t, os.Getenv("AZURE_CLIENT_CERTIFICATE_PATH"))
	assert.Empty(t, os.Getenv("AZURE_CLIENT_CERTIFICATE_PASSWORD"))
	assert.Empty(t, os.Getenv("AZURE_ADDITIONALLY_ALLOWED_TENANTS"))

	lv, _ := types.ListValue(types.StringType, []attr.Value{
		types.StringValue("tenant2"),
		types.StringValue("tenant3"),
	})
	// Test when all data fields are set
	data = &AlzProviderModel{
		TenantId:                  types.StringValue("tenant1"),
		ClientId:                  types.StringValue("client1"),
		ClientSecret:              types.StringValue("secret1"),
		ClientCertificatePath:     types.StringValue("/path/to/cert"),
		ClientCertificatePassword: types.StringValue("password1"),
		AuxiliaryTenantIds:        lv,
	}
	configureAzIdentityEnvironment(data)
	assert.Equal(t, "tenant1", os.Getenv("AZURE_TENANT_ID"))
	assert.Equal(t, "client1", os.Getenv("AZURE_CLIENT_ID"))
	assert.Equal(t, "secret1", os.Getenv("AZURE_CLIENT_SECRET"))
	assert.Equal(t, "/path/to/cert", os.Getenv("AZURE_CLIENT_CERTIFICATE_PATH"))
	assert.Equal(t, "password1", os.Getenv("AZURE_CLIENT_CERTIFICATE_PASSWORD"))
	assert.Equal(t, "tenant2;tenant3", os.Getenv("AZURE_ADDITIONALLY_ALLOWED_TENANTS"))
}

func TestStr2Bool(t *testing.T) {
	// Test when input is "true"
	result := str2Bool("true")
	assert.Equal(t, true, result)

	// Test when input is "false"
	result = str2Bool("false")
	assert.Equal(t, false, result)

	// Test when input is "TRUE"
	result = str2Bool("TRUE")
	assert.Equal(t, true, result)

	// Test when input is "FALSE"
	result = str2Bool("FALSE")
	assert.Equal(t, false, result)

	// Test when input is "1"
	result = str2Bool("1")
	assert.Equal(t, true, result)

	// Test when input is "0"
	result = str2Bool("0")
	assert.Equal(t, false, result)

	// Test when input is "invalid"
	result = str2Bool("invalid")
	assert.Equal(t, false, result)
}

func TestListElementsToStrings(t *testing.T) {
	// Test when list is empty
	list := []attr.Value{}
	result := listElementsToStrings(list)
	assert.Nil(t, result)

	// Test when list contains only string values
	list = []attr.Value{basetypes.NewStringValue("value1"), types.StringValue("value2")}
	result = listElementsToStrings(list)
	assert.Equal(t, []string{"value1", "value2"}, result)

	// Test when list contains non-string values
	list = []attr.Value{basetypes.NewStringValue("value1"), types.NumberValue(big.NewFloat(1.0))}
	result = listElementsToStrings(list)
	assert.Nil(t, result)
}
