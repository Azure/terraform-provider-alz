// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package provider

import (
	"math/big"
	"os"
	"testing"

	"github.com/Azure/terraform-provider-alz/internal/aztfschema"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/stretchr/testify/assert"
)

// testAccProtoV6ProviderFactories are used to instantiate a provider during
// acceptance testing. The factory function will be invoked for every Terraform
// CLI command executed to create a provider server to which the CLI can
// reattach.
// var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
// 	"alz": providerserver.NewProtocol6WithError(New("test")()),
// }

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
	data := &AlzModel{}
	data.ConfigureFromEnv()
	configureFromEnvironment(data)
	assert.True(t, data.ClientCertificatePassword.IsNull())
	assert.True(t, data.ClientCertificatePath.IsNull())
	assert.True(t, data.ClientID.IsNull())
	assert.True(t, data.ClientSecret.IsNull())
	assert.True(t, data.Environment.IsNull())
	assert.True(t, data.OIDCRequestToken.IsNull())
	assert.True(t, data.OIDCRequestURL.IsNull())
	assert.True(t, data.OIDCToken.IsNull())
	assert.True(t, data.OIDCTokenFilePath.IsNull())
	assert.True(t, data.TenantID.IsNull())
	assert.True(t, data.UseCLI.IsNull())
	assert.True(t, data.UseMSI.IsNull())
	assert.True(t, data.UseOIDC.IsNull())
	assert.True(t, data.SkipProviderRegistration.IsNull())

	// Test when some environment variables are set
	t.Setenv("ARM_CLIENT_ID", "client_id")
	t.Setenv("ARM_CLIENT_SECRET", "client_secret")
	t.Setenv("ARM_TENANT_ID", "tenant_id")
	data = &AlzModel{}
	data.ConfigureFromEnv()
	configureFromEnvironment(data)
	assert.Equal(t, "", data.ClientCertificatePassword.ValueString())
	assert.Equal(t, "", data.ClientCertificatePath.ValueString())
	assert.Equal(t, "client_id", data.ClientID.ValueString())
	assert.Equal(t, "client_secret", data.ClientSecret.ValueString())
	assert.Equal(t, "", data.Environment.ValueString())
	assert.Equal(t, "", data.OIDCRequestToken.ValueString())
	assert.Equal(t, "", data.OIDCRequestURL.ValueString())
	assert.Equal(t, "", data.OIDCToken.ValueString())
	assert.Equal(t, "", data.OIDCTokenFilePath.ValueString())
	assert.Equal(t, "tenant_id", data.TenantID.ValueString())
	assert.Equal(t, false, data.UseCLI.ValueBool())
	assert.Equal(t, false, data.UseMSI.ValueBool())
	assert.Equal(t, false, data.UseOIDC.ValueBool())
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
	data = &AlzModel{}
	data.ConfigureFromEnv()
	configureFromEnvironment(data)
	assert.Equal(t, "password", data.ClientCertificatePassword.ValueString())
	assert.Equal(t, "path", data.ClientCertificatePath.ValueString())
	assert.Equal(t, "client_id", data.ClientID.ValueString())
	assert.Equal(t, "client_secret", data.ClientSecret.ValueString())
	assert.Equal(t, "environment", data.Environment.ValueString())
	assert.Equal(t, "request_token", data.OIDCRequestToken.ValueString())
	assert.Equal(t, "request_url", data.OIDCRequestURL.ValueString())
	assert.Equal(t, "token", data.OIDCToken.ValueString())
	assert.Equal(t, "token_file_path", data.OIDCTokenFilePath.ValueString())
	assert.Equal(t, "tenant_id", data.TenantID.ValueString())
	assert.Equal(t, true, data.UseCLI.ValueBool())
	assert.Equal(t, true, data.UseMSI.ValueBool())
	assert.Equal(t, true, data.UseOIDC.ValueBool())
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

func TestConfigureAzIdentityEnvironment(t *testing.T) {
	// Test when no data fields are set
	data := &AlzModel{}
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
	data = &AlzModel{
		AuthModelWithSubscriptionID: aztfschema.AuthModelWithSubscriptionID{
			AuthModel: aztfschema.AuthModel{
				TenantID:                  types.StringValue("tenant1"),
				ClientID:                  types.StringValue("client1"),
				ClientSecret:              types.StringValue("secret1"),
				ClientCertificatePath:     types.StringValue("/path/to/cert"),
				ClientCertificatePassword: types.StringValue("password1"),
				AuxiliaryTenantIDs:        lv,
			},
		},
	}
	data.ConfigureFromEnv()
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
