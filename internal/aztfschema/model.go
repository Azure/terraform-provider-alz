package aztfschema

import (
	"context"
	"fmt"
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

// EndpointModel represents the custom endpoint configuration for Azure environments.
type EndpointModel struct {
	ResourceManagerEndpoint      types.String `tfsdk:"resource_manager_endpoint" fromenv:"ARM_RESOURCE_MANAGER_ENDPOINT"`
	ActiveDirectoryAuthorityHost types.String `tfsdk:"active_directory_authority_host" fromenv:"ARM_ACTIVE_DIRECTORY_AUTHORITY_HOST"`
	ResourceManagerAudience      types.String `tfsdk:"resource_manager_audience" fromenv:"ARM_RESOURCE_MANAGER_AUDIENCE"`
}

// AuthModel represents the base model for the authentication attributes.
// Embed this struct in your own model to include the authentication attributes.
type AuthModel struct {
	ClientID                     types.String `tfsdk:"client_id" fromenv:"ARM_CLIENT_ID,AZURE_CLIENT_ID"`
	ClientIDFilePath             types.String `tfsdk:"client_id_file_path" fromenv:"ARM_CLIENT_ID_FILE_PATH"`
	TenantID                     types.String `tfsdk:"tenant_id" fromenv:"ARM_TENANT_ID,AZURE_TENANT_ID"`
	AuxiliaryTenantIDs           types.List   `tfsdk:"auxiliary_tenant_ids" fromenv:"ARM_AUXILIARY_TENANT_IDS"`
	Environment                  types.String `tfsdk:"environment" fromenv:"ARM_ENVIRONMENT,AZURE_ENVIRONMENT"`
	Endpoint                     types.List   `tfsdk:"endpoint"`
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
	// Also configure endpoint model from environment if endpoint list is empty
	if (m.Endpoint.IsNull() || len(m.Endpoint.Elements()) == 0) && strings.ToLower(m.Environment.ValueString()) == "custom" {
		m.configureEndpointFromEnv()
	}
}

// ConfigureFromEnv sets default values from environment variables for the model.
func (m *AuthModelWithSubscriptionID) ConfigureFromEnv() {
	m.AuthModel.ConfigureFromEnv()
	setFieldDefaultsFromEnv(m)
}

// getCloudConfiguration returns the cloud configuration based on environment and endpoint settings.
// It supports both predefined environments (public, usgovernment, china) and custom environments.
func (m *AuthModel) getCloudConfiguration(ctx context.Context) (cloud.Configuration, error) {
	envName := strings.ToLower(strings.TrimSpace(m.Environment.ValueString()))

	// Handle predefined environments
	if cloudConfig, ok := environmentToCloud[envName]; ok {
		return cloudConfig, nil
	}

	// Handle custom environment
	if envName == "custom" {
		return m.buildCustomCloudConfiguration(ctx)
	}

	// Default to public cloud if not specified or empty
	if envName == "" {
		return cloud.AzurePublic, nil
	}

	return cloud.Configuration{}, fmt.Errorf("unsupported environment '%s': must be one of 'public', 'usgovernment', 'china', or 'custom'", envName)
}

// buildCustomCloudConfiguration builds a custom cloud configuration from the endpoint block.
func (m *AuthModel) buildCustomCloudConfiguration(ctx context.Context) (cloud.Configuration, error) {
	// Check if endpoint configuration is provided
	if m.Endpoint.IsNull() || len(m.Endpoint.Elements()) == 0 {
		return cloud.Configuration{}, fmt.Errorf("endpoint configuration is required when environment is set to 'custom'")
	}

	// Parse endpoint configuration
	var endpoints []EndpointModel
	diags := m.Endpoint.ElementsAs(ctx, &endpoints, false)
	if diags.HasError() {
		return cloud.Configuration{}, fmt.Errorf("failed to parse endpoint configuration: %v", diags.Errors())
	}

	endpoint := endpoints[0]

	// Validate required fields
	var missingFields []string
	if endpoint.ResourceManagerEndpoint.IsNull() || endpoint.ResourceManagerEndpoint.ValueString() == "" {
		missingFields = append(missingFields, "resource_manager_endpoint")
	}
	if endpoint.ActiveDirectoryAuthorityHost.IsNull() || endpoint.ActiveDirectoryAuthorityHost.ValueString() == "" {
		missingFields = append(missingFields, "active_directory_authority_host")
	}
	if endpoint.ResourceManagerAudience.IsNull() || endpoint.ResourceManagerAudience.ValueString() == "" {
		missingFields = append(missingFields, "resource_manager_audience")
	}

	if len(missingFields) > 0 {
		return cloud.Configuration{}, fmt.Errorf("required endpoint fields missing for custom environment: %s", strings.Join(missingFields, ", "))
	}

	// Build custom cloud configuration
	return cloud.Configuration{
		ActiveDirectoryAuthorityHost: endpoint.ActiveDirectoryAuthorityHost.ValueString(),
		Services: map[cloud.ServiceName]cloud.ServiceConfiguration{
			cloud.ResourceManager: {
				Endpoint: endpoint.ResourceManagerEndpoint.ValueString(),
				Audience: endpoint.ResourceManagerAudience.ValueString(),
			},
		},
	}, nil
}

// configureEndpointFromEnv configures the endpoint from environment variables.
func (m *AuthModel) configureEndpointFromEnv() {
	var endpoint EndpointModel
	setFieldDefaultsFromEnv(&endpoint)

	// Only create endpoint list if at least one endpoint field is set
	if !endpoint.ResourceManagerEndpoint.IsNull() ||
		!endpoint.ActiveDirectoryAuthorityHost.IsNull() ||
		!endpoint.ResourceManagerAudience.IsNull() {

		ctx := context.Background()
		endpointValue, diags := types.ObjectValueFrom(ctx, map[string]attr.Type{
			"resource_manager_endpoint":       types.StringType,
			"active_directory_authority_host": types.StringType,
			"resource_manager_audience":       types.StringType,
		}, endpoint)
		if diags.HasError() {
			// Failed to convert endpoint struct to Terraform object; do not set m.Endpoint from env.
			// Errors will be surfaced during later validation of the configuration.
			return
		}

		m.Endpoint = types.ListValueMust(
			types.ObjectType{
				AttrTypes: map[string]attr.Type{
					"resource_manager_endpoint":       types.StringType,
					"active_directory_authority_host": types.StringType,
					"resource_manager_audience":       types.StringType,
				},
			},
			[]attr.Value{endpointValue},
		)
	}
}

// AuthOption returns the authentication options for the model.
// To be used by the aztfauth package.
// This function doesn't set the Logger field, so it must be set separately.
// Returns an error if cloud configuration is invalid.
func (m *AuthModel) AuthOption(ctx context.Context, opts azcore.ClientOptions) (aztfauth.Option, error) {
	cloudConfig, err := m.getCloudConfiguration(ctx)
	if err != nil {
		return aztfauth.Option{}, err
	}
	opts.Cloud = cloudConfig

	auxTenantIDs := make([]string, len(m.AuxiliaryTenantIDs.Elements()))
	m.AuxiliaryTenantIDs.ElementsAs(ctx, &auxTenantIDs, false)

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
	}, nil
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
