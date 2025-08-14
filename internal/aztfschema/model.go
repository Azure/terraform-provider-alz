package aztfschema

import (
	"context"
	"os"
	"reflect"
	"strconv"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/cloud"
	"github.com/Azure/entrauth/aztfauth"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

// AuthModel represents the base model for the authentication attributes.
// Embed this struct in your own model to include the authentication attributes.
type AuthModel struct {
	ClientID                     types.String `tfsdk:"client_id" fromenv:"ARM_CLIENT_ID,AZURE_CLIENT_ID"`
	ClientIDFilePath             types.String `tfsdk:"client_id_file_path" fromenv:"ARM_CLIENT_ID_FILE_PATH"`
	TenantID                     types.String `tfsdk:"tenant_id" fromenv:"ARM_TENANT_ID,AZURE_TENANT_ID"`
	AuxiliaryTenantIDs           types.List   `tfsdk:"auxiliary_tenant_ids" fromenv:"ARM_AUXILIARY_TENANT_IDS"`
	Environment                  types.String `tfsdk:"environment" fromenv:"ARM_ENVIRONMENT,AZURE_ENVIRONMENT"`
	ClientCertificate            types.String `tfsdk:"client_certificate" fromenv:"ARM_CLIENT_CERTIFICATE"`
	ClientCertificatePath        types.String `tfsdk:"client_certificate_path" fromenv:"ARM_CLIENT_CERTIFICATE_PATH"`
	ClientCertificatePassword    types.String `tfsdk:"client_certificate_password" fromenv:"ARM_CLIENT_CERTIFICATE_PASSWORD"`
	ClientSecret                 types.String `tfsdk:"client_secret" fromenv:"ARM_CLIENT_SECRET,AZURE_CLIENT_SECRET"`
	ClientSecretFilePath         types.String `tfsdk:"client_secret_file_path" fromenv:"ARM_CLIENT_SECRET_FILE_PATH"`
	OIDCRequestToken             types.String `tfsdk:"oidc_request_token" fromenv:"ARM_OIDC_REQUEST_TOKEN,ACTIONS_ID_TOKEN_REQUEST_TOKEN,SYSTEM_ACCESSTOKEN"`
	OIDCRequestURL               types.String `tfsdk:"oidc_request_url" fromenv:"ARM_OIDC_REQUEST_URL,ACTIONS_ID_TOKEN_REQUEST_URL,SYSTEM_OIDCREQUESTURI"`
	OIDCToken                    types.String `tfsdk:"oidc_token" fromenv:"ARM_OIDC_TOKEN"`
	OIDCTokenFilePath            types.String `tfsdk:"oidc_token_file_path" fromenv:"ARM_OIDC_TOKEN_FILE_PATH,AZURE_FEDERATED_TOKEN_FILE"`
	OIDCAzureServiceConnectionID types.String `tfsdk:"oidc_azure_service_connection_id" fromenv:"ARM_ADO_PIPELINE_SERVICE_CONNECTION_ID,ARM_OIDC_AZURE_SERVICE_CONNECTION_ID,AZURESUBSCRIPTION_SERVICE_CONNECTION_ID"`
	UseAKSWorkloadIdentity       types.Bool   `tfsdk:"use_aks_workload_identity" fromenv:"ARM_USE_AKS_WORKLOAD_IDENTITY" defaultvalue:"false"`
	UseOIDC                      types.Bool   `tfsdk:"use_oidc" fromenv:"ARM_USE_OIDC" defaultvalue:"false"`
	UseCLI                       types.Bool   `tfsdk:"use_cli" fromenv:"ARM_USE_CLI" defaultvalue:"true"`
	UseMSI                       types.Bool   `tfsdk:"use_msi" fromenv:"ARM_USE_MSI" defaultvalue:"false"`
}

// AuthModelWithSubscriptionID is a model that includes the subscription ID.
// It embeds the base AuthModel struct and adds a SubscriptionID field.
// Embed this struct in your own model to include the subscription ID.
type AuthModelWithSubscriptionID struct {
	AuthModel
	SubscriptionID types.String `tfsdk:"subscription_id" fromenv:"ARM_SUBSCRIPTION_ID,AZURE_SUBSCRIPTION_ID"`
}

// environmentToCloud maps environment names to their corresponding cloud configurations.
var environmentToCloud = map[string]cloud.Configuration{
	"public":       cloud.AzurePublic,
	"usgovernment": cloud.AzureGovernment,
	"china":        cloud.AzureChina,
}

// SetOpinionatedDefaults sets default values for the model, if the values are null. The values are based on the defaults in the struct tags.
// Typically this is run after ConfigureFromEnv.
func (m *AuthModel) SetOpinionatedDefaults() {
	setDefaultValueFromStructTags(m)
}

// SetOpinionatedDefaults sets default values for the model, if the values are null. The values are based on the defaults in the struct tags.
// Typically this is run after ConfigureFromEnv.
func (m *AuthModelWithSubscriptionID) SetOpinionatedDefaults() {
	m.AuthModel.SetOpinionatedDefaults()
	setDefaultValueFromStructTags(m)
}

// ConfigureFromEnv sets default values from environment variables for the model.
func (m *AuthModel) ConfigureFromEnv() {
	setFieldDefaultsFromEnv(m)
}

// ConfigureFromEnv sets default values from environment variables for the model.
func (m *AuthModelWithSubscriptionID) ConfigureFromEnv() {
	m.AuthModel.ConfigureFromEnv()
	setFieldDefaultsFromEnv(m)
}

// AuthOption returns the authentication options for the model.
// To be used by the aztfauth package.
// This function doesn't set the Logger field, so it must be set separately.
func (m *AuthModel) AuthOption(opts azcore.ClientOptions) aztfauth.Option {
	if cloudConfig, ok := environmentToCloud[m.Environment.ValueString()]; ok {
		opts.Cloud = cloudConfig
	}

	auxTenantIDs := make([]string, len(m.AuxiliaryTenantIDs.Elements()))
	m.AuxiliaryTenantIDs.ElementsAs(context.Background(), &auxTenantIDs, false)

	return aztfauth.Option{
		AdditionallyAllowedTenants: auxTenantIDs,
		ADOServiceConnectionId:     m.OIDCAzureServiceConnectionID.ValueString(),
		ClientCertBase64:           m.ClientCertificate.ValueString(),
		ClientCertPassword:         []byte(m.ClientCertificatePassword.ValueString()),
		ClientCertPfxFile:          m.ClientCertificatePath.ValueString(),
		ClientId:                   m.ClientID.ValueString(),
		ClientIdFile:               m.ClientIDFilePath.ValueString(),
		ClientOptions:              opts,
		ClientSecret:               m.ClientSecret.ValueString(),
		ClientSecretFile:           m.ClientSecretFilePath.ValueString(),
		OIDCRequestToken:           m.OIDCRequestToken.ValueString(),
		OIDCRequestURL:             m.OIDCRequestURL.ValueString(),
		OIDCToken:                  m.OIDCToken.ValueString(),
		OIDCTokenFile:              m.OIDCTokenFilePath.ValueString(),
		TenantId:                   m.TenantID.ValueString(),
		UseAzureCLI:                m.UseCLI.ValueBool(),
		UseClientCert:              true,
		UseClientSecret:            true,
		UseOIDCToken:               m.UseOIDC.ValueBool(),
		UseOIDCTokenFile:           m.UseOIDC.ValueBool() || m.UseAKSWorkloadIdentity.ValueBool(),
		UseOIDCTokenRequest:        m.UseOIDC.ValueBool(),
	}
}

// setFieldDefaultsFromEnv iterates through the model and set default values from environment if the .IsNull() method is true.
// It uses the `fromenv` struct tag to find the corresponding environment variables (comma separated).
func setFieldDefaultsFromEnv(a any) {
	val := reflect.ValueOf(a).Elem()
	typ := val.Type()

	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		realValInt := val.Field(i).Interface()
		realAttrVal, ok := realValInt.(attr.Value)

		// Only apply defaults to string-typed fields that are currently null.
		if !ok || !realAttrVal.IsNull() {
			continue
		}

		envVar := field.Tag.Get("fromenv")
		if envVar == "" {
			continue
		}

		// Get the environment variable value and set it
		envVars := strings.Split(envVar, ",")

		for _, envVar := range envVars {
			envValue := os.Getenv(envVar)
			if envValue == "" {
				continue
			}

			switch realValInt.(type) {
			case types.String:
				val.Field(i).Set(reflect.ValueOf(types.StringValue(envValue)))

			case types.List:
				// Split on semicolon and create []attr.Value
				var listValues []attr.Value
				for _, item := range strings.Split(envValue, ";") {
					listValues = append(listValues, types.StringValue(item))
				}
				val.Field(i).Set(reflect.ValueOf(types.ListValueMust(
					basetypes.StringType{},
					listValues,
				)))

			case types.Bool:
				b, err := strconv.ParseBool(envValue)
				if err != nil {
					continue // Skip if conversion fails
				}
				val.Field(i).Set(reflect.ValueOf(types.BoolValue(b)))
			}

			// First non-empty env var wins
			break
		}
	}
}

// setDefaultValueFromStructTags sets default values (not already set) for the model based on struct tag `defaultvalue`.
func setDefaultValueFromStructTags(a any) {
	val := reflect.ValueOf(a).Elem()
	typ := val.Type()

	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		realValInt := val.Field(i).Interface()
		realAttrVal, ok := realValInt.(attr.Value)

		// Only apply defaults to string-typed fields that are currently null.
		if !ok || !realAttrVal.IsNull() {
			continue
		}

		defaultValue := field.Tag.Get("defaultvalue")
		if defaultValue == "" {
			continue
		}

		switch realValInt.(type) {
		case types.String:
			val.Field(i).Set(reflect.ValueOf(types.StringValue(defaultValue)))

		case types.List:
			// Split on comma and create []attr.Value
			var listValues []attr.Value
			for _, item := range strings.Split(defaultValue, ",") {
				listValues = append(listValues, types.StringValue(item))
			}
			val.Field(i).Set(reflect.ValueOf(types.ListValueMust(
				basetypes.StringType{},
				listValues,
			)))

		case types.Bool:
			b, err := strconv.ParseBool(defaultValue)
			if err != nil {
				continue // Skip if conversion fails
			}
			val.Field(i).Set(reflect.ValueOf(types.BoolValue(b)))
		}
	}
}
