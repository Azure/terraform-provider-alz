// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package provider

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Azure/alzlib"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/cloud"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/authorization/armauthorization"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armpolicy"
	"github.com/Azure/terraform-provider-alz/internal/provider/gen"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Run go generate to automatically generate provider, data source and resource types
// from the intermediate representation JSON file `ir.json`.
//go:generate tfplugingen-framework generate provider --package gen --output ./gen
//go:generate tfplugingen-framework generate data-sources --package gen --output ./gen
//go:generate tfplugingen-framework generate resources --package gen --output ./gen

const (
	userAgentBase = "AzureTerraformAlzProvider"
	alzLibDirBase = ".alzlib"
	alzLibRef     = "2024.07.01"
	alzLibPath    = "platform/alz"
)

// Ensure ScaffoldingProvider satisfies various provider interfaces.
var (
	_ provider.Provider              = &AlzProvider{}
	_ provider.ProviderWithFunctions = &AlzProvider{}
)

// AlzProvider defines the provider implementation.
type AlzProvider struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.
	version string
	alz     *alzProviderData
}

type AlzProviderClients struct {
	RoleAssignmentsClient *armauthorization.RoleAssignmentsClient
}

type alzProviderData struct {
	*alzlib.AlzLib
	mu      *sync.Mutex
	clients *AlzProviderClients
}

func (p *AlzProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "alz"
	resp.Version = p.version
}

func (p *AlzProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = gen.AlzProviderSchema(ctx)
}

func (p *AlzProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	tflog.Debug(ctx, "Provider configuration started")

	if p.alz != nil {
		tflog.Debug(ctx, "Provider AlzLib already present, skipping configuration")
		resp.DataSourceData = p.alz
		resp.ResourceData = p.alz
		return
	}

	tflog.Debug(ctx, "Provider AlzLib not present, beginning configuration")

	var data gen.AlzModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}
	// Read the environment variables and set in data
	// if the data is not already set and the environment variable is set.
	configureFromEnvironment(&data)

	// Set the go sdk's azidentity specific environment variables
	configureAzIdentityEnvironment(&data)

	// Configure aux tenant ids from config and environment.
	if resp.Diagnostics = append(resp.Diagnostics, configureAuxTenants(ctx, &data)...); resp.Diagnostics.HasError() {
		return
	}

	// Set the default values if not already set in the config or by environment.
	configureDefaults(ctx, &data)

	// Get a token credential.
	cred, diags := getTokenCredential(data)
	resp.Diagnostics = append(resp.Diagnostics, diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Create the clients
	clients, diags := getClients(cred, data, fmt.Sprintf("%s/%s", userAgentBase, p.version))
	resp.Diagnostics = append(resp.Diagnostics, diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Create the AlzLib.
	alz, diags := configureAlzLib(cred, data, fmt.Sprintf("%s/%s", userAgentBase, p.version))
	resp.Diagnostics = append(resp.Diagnostics, diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Configure lib references
	libDirFs, diags := downloadLibs(ctx, &data)
	resp.Diagnostics = append(resp.Diagnostics, diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Init alzlib
	if err := alz.Init(ctx, libDirFs...); err != nil {
		resp.Diagnostics.AddError("Failed to initialize AlzLib", err.Error())
		return
	}

	// Store the alz pointer in the provider struct so we don't have to do all this work every time `.Configure` is called.
	// Due to fetch from Azure, it takes approx 30 seconds each time and is called 4-5 time during a single acceptance test.
	p.alz = &alzProviderData{
		AlzLib:  alz,
		mu:      &sync.Mutex{},
		clients: clients,
	}
	resp.DataSourceData = p.alz
	resp.ResourceData = p.alz
	tflog.Debug(ctx, "Provider configuration finished")
}

func (p *AlzProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewPolicyRoleAssignmentsResource,
	}
}

func (p *AlzProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewArchitectureDataSource,
	}
}

func (p *AlzProvider) Functions(ctx context.Context) []func() function.Function {
	return []func() function.Function{}
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &AlzProvider{
			version: version,
		}
	}
}

func downloadLibs(ctx context.Context, data *gen.AlzModel) ([]fs.FS, diag.Diagnostics) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	var diags diag.Diagnostics

	alzLibRefs := make([]gen.AlzLibraryReferencesValue, len(data.AlzLibraryReferences.Elements()))
	diags = data.AlzLibraryReferences.ElementsAs(ctx, &alzLibRefs, false)
	if diags.HasError() {
		return nil, diags
	}

	libDirFs := make([]fs.FS, len(alzLibRefs))
	for i, ref := range alzLibRefs {
		if !ref.CustomUrl.IsNull() {
			ldfs, err := alzlib.FetchLibraryByGetterString(ctx, ref.CustomUrl.ValueString(), strconv.Itoa(i))
			if err != nil {
				diags.AddError("Failed to fetch library", err.Error())
				return nil, diags
			}
			libDirFs[i] = ldfs
		}
		ldfs, err := alzlib.FetchAzureLandingZonesLibraryMember(ctx, ref.Path.ValueString(), ref.Ref.ValueString(), strconv.Itoa(i))
		if err != nil {
			diags.AddError("Failed to fetch library", err.Error())
			return nil, diags
		}
		libDirFs[i] = ldfs
	}
	return libDirFs, nil
}

func getFirstSetEnvVar(envVars ...string) string {
	for _, envVar := range envVars {
		if val := os.Getenv(envVar); val != "" {
			return val
		}
	}
	return ""
}

// configureAuxTenants gets a slice of the auxiliary tenant IDs from the provider data,
// or the environment variable `ARM_AUXILIARY_TENANT_IDS` if the provider data is not set.
func configureAuxTenants(ctx context.Context, data *gen.AlzModel) diag.Diagnostics {
	var auxTenants []string
	if data.AuxiliaryTenantIds.IsNull() {
		if v := os.Getenv("ARM_AUXILIARY_TENANT_IDS"); v != "" {
			auxTenants = strings.Split(v, ";")
		}
		var diags diag.Diagnostics
		data.AuxiliaryTenantIds, diags = types.ListValueFrom(ctx, types.StringType, auxTenants)
		return diags
	}
	return nil
}

// configureFromEnvironment sets the provider data from environment variables.
func configureFromEnvironment(data *gen.AlzModel) {
	if val := getFirstSetEnvVar("ARM_CLIENT_CERTIFICATE_PASSWORD"); val != "" && data.ClientCertificatePassword.IsNull() {
		data.ClientCertificatePassword = types.StringValue(val)
	}

	if val := getFirstSetEnvVar("ARM_CLIENT_CERTIFICATE_PATH"); val != "" && data.ClientCertificatePath.IsNull() {
		data.ClientCertificatePath = types.StringValue(val)
	}

	if val := getFirstSetEnvVar("ARM_CLIENT_ID"); val != "" && data.ClientId.IsNull() {
		data.ClientId = types.StringValue(val)
	}

	if val := getFirstSetEnvVar("ARM_CLIENT_SECRET"); val != "" && data.ClientSecret.IsNull() {
		data.ClientSecret = types.StringValue(val)
	}

	if val := getFirstSetEnvVar("ARM_ENVIRONMENT"); val != "" && data.Environment.IsNull() {
		data.Environment = types.StringValue(val)
	}

	if val := getFirstSetEnvVar("ARM_OIDC_REQUEST_TOKEN", "ACTIONS_ID_TOKEN_REQUEST_TOKEN"); val != "" && data.OidcRequestToken.IsNull() {
		data.OidcRequestToken = types.StringValue(val)
	}

	if val := getFirstSetEnvVar("ARM_OIDC_REQUEST_URL", "ACTIONS_ID_TOKEN_REQUEST_URL"); val != "" && data.OidcRequestUrl.IsNull() {
		data.OidcRequestUrl = types.StringValue(val)
	}

	if val := getFirstSetEnvVar("ARM_OIDC_TOKEN"); val != "" && data.OidcToken.IsNull() {
		data.OidcToken = types.StringValue(val)
	}

	if val := getFirstSetEnvVar("ARM_OIDC_TOKEN_FILE_PATH"); val != "" && data.OidcTokenFilePath.IsNull() {
		data.OidcTokenFilePath = types.StringValue(val)
	}

	if val := getFirstSetEnvVar("ARM_TENANT_ID"); val != "" && data.TenantId.IsNull() {
		data.TenantId = types.StringValue(val)
	}

	if val := getFirstSetEnvVar("ARM_USE_CLI"); val != "" && data.UseCli.IsNull() {
		data.UseCli = types.BoolValue(str2Bool(val))
	}

	if val := getFirstSetEnvVar("ARM_USE_MSI"); val != "" && data.UseMsi.IsNull() {
		data.UseMsi = types.BoolValue(str2Bool(val))
	}

	if val := getFirstSetEnvVar("ARM_USE_OIDC"); val != "" && data.UseOidc.IsNull() {
		data.UseOidc = types.BoolValue(str2Bool(val))
	}

	if val := getFirstSetEnvVar("ARM_SKIP_PROVIDER_REGISTRATION"); val != "" && data.SkipProviderRegistration.IsNull() {
		data.SkipProviderRegistration = types.BoolValue(str2Bool(val))
	}
}

// str2Bool converts a string to a bool, returning false if the string is not a valid bool.
func str2Bool(val string) bool {
	b, err := strconv.ParseBool(val)
	if err != nil {
		b = false
	}
	return b
}

// configureAzIdentityEnvironment sets the environment variables used by go Azure sdk's azidentity package.
func configureAzIdentityEnvironment(data *gen.AlzModel) {
	// Maps the auth related environment variables used in the provider to what azidentity honors.
	if !data.TenantId.IsNull() {
		// #nosec G104
		os.Setenv("AZURE_TENANT_ID", data.TenantId.ValueString())
	}
	if !data.ClientId.IsNull() {
		// #nosec G104
		os.Setenv("AZURE_CLIENT_ID", data.ClientId.ValueString())
	}
	if !data.ClientSecret.IsNull() {
		// #nosec G104
		os.Setenv("AZURE_CLIENT_SECRET", data.ClientSecret.ValueString())
	}
	if !data.ClientCertificatePath.IsNull() {
		// #nosec G104
		os.Setenv("AZURE_CLIENT_CERTIFICATE_PATH", data.ClientCertificatePath.ValueString())
	}
	if !data.ClientCertificatePassword.IsNull() {
		// #nosec G104
		os.Setenv("AZURE_CLIENT_CERTIFICATE_PASSWORD", data.ClientCertificatePassword.ValueString())
	}
	if len(data.AuxiliaryTenantIds.Elements()) != 0 {
		auxTenants := listElementsToStrings(data.AuxiliaryTenantIds.Elements())
		// #nosec G104
		os.Setenv("AZURE_ADDITIONALLY_ALLOWED_TENANTS", strings.Join(auxTenants, ";"))
	}
}

// listElementsToStrings converts a list of attr.Value to a list of strings.
func listElementsToStrings(list []attr.Value) []string {
	if len(list) == 0 {
		return nil
	}
	strings := make([]string, len(list))
	for i, v := range list {
		sv, ok := v.(basetypes.StringValue)
		if !ok {
			return nil
		}
		strings[i] = sv.ValueString()
	}
	return strings
}

// configureAlzLib configures the alzlib for use by the provider.
func configureAlzLib(token *azidentity.ChainedTokenCredential, data gen.AlzModel, userAgent string) (*alzlib.AlzLib, diag.Diagnostics) {
	var diags diag.Diagnostics
	popts := new(policy.ClientOptions)
	popts.DisableRPRegistration = data.SkipProviderRegistration.ValueBool()
	popts.PerRetryPolicies = append(popts.PerRetryPolicies, withUserAgent(userAgent))

	opts := &alzlib.AlzLibOptions{
		AllowOverwrite: data.LibOverwriteEnabled.ValueBool(),
		Parallelism:    10,
	}
	alz := alzlib.NewAlzLib(opts)
	cf, err := armpolicy.NewClientFactory("", token, popts)
	if err != nil {
		diags.AddError("failed to create Azure Policy client factory: %v", err.Error())
		return nil, diags
	}

	alz.AddPolicyClient(cf)

	return alz, diags
}

func getClients(token *azidentity.ChainedTokenCredential, data gen.AlzModel, userAgent string) (*AlzProviderClients, diag.Diagnostics) {
	var diags diag.Diagnostics
	clients := new(AlzProviderClients)

	popts := new(policy.ClientOptions)
	popts.DisableRPRegistration = data.SkipProviderRegistration.ValueBool()
	popts.PerRetryPolicies = append(popts.PerRetryPolicies, withUserAgent(userAgent))

	client, err := armauthorization.NewRoleAssignmentsClient("", token, popts)

	// Create the clients
	//roleAssignmentsClient, err := newRoleAssignmentsClient(data)
	if err != nil {
		diags.AddError("failed to create Azure Role Assignments client: %v", err.Error())
		return clients, diags
	}

	clients.RoleAssignmentsClient = client

	return clients, diags
}

// getTokenCredential gets a token credential based on the provider data.
func getTokenCredential(data gen.AlzModel) (*azidentity.ChainedTokenCredential, diag.Diagnostics) {
	var diags diag.Diagnostics
	var cloudConfig cloud.Configuration
	env := data.Environment.ValueString()
	switch strings.ToLower(env) {
	case "public":
		cloudConfig = cloud.AzurePublic
	case "usgovernment":
		cloudConfig = cloud.AzureGovernment
	case "china":
		cloudConfig = cloud.AzureChina
	default:
		diags.AddError("Could not determine cloud configuration", "Valid values are 'public', 'usgovernment', or 'china'")
		return nil, diags
	}

	auxTenants := listElementsToStrings(data.AuxiliaryTenantIds.Elements())

	option := &azidentity.DefaultAzureCredentialOptions{
		AdditionallyAllowedTenants: auxTenants,
		ClientOptions: azcore.ClientOptions{
			Cloud: cloudConfig,
		},
		TenantID: data.TenantId.ValueString(),
	}

	return newDefaultAzureCredential(data, option)
}

// configureDefaults sets default values if they aren't already set.
func configureDefaults(ctx context.Context, data *gen.AlzModel) {
	// Use azure public cloud by default.
	if data.Environment.IsNull() {
		data.Environment = types.StringValue("public")
	}

	// Do not skip provider registration by default.
	if data.SkipProviderRegistration.IsNull() {
		data.SkipProviderRegistration = types.BoolValue(false)
	}

	// Do not use OIDC auth by default.
	if data.UseOidc.IsNull() {
		data.UseOidc = types.BoolValue(false)
	}

	// Do not use MSI auth by default.
	if data.UseMsi.IsNull() {
		data.UseMsi = types.BoolValue(false)
	}

	// Use CLI auth by default.
	if data.UseCli.IsNull() {
		data.UseCli = types.BoolValue(true)
	}

	// Do not allow library overwrite by default.
	if data.LibOverwriteEnabled.IsNull() {
		data.LibOverwriteEnabled = types.BoolValue(false)
	}

	// Set alz library references to the default value if not already set.
	if data.AlzLibraryReferences.IsNull() {
		element := gen.NewAlzLibraryReferencesValueMust(
			gen.NewAlzLibraryReferencesValueNull().AttributeTypes(ctx),
			map[string]attr.Value{
				"ref":        types.StringValue(alzLibRef),
				"path":       types.StringValue(alzLibPath),
				"custom_url": types.StringNull(),
			},
		)
		data.AlzLibraryReferences = types.ListValueMust(element.Type(ctx), []attr.Value{element})
	}
}

func newDefaultAzureCredential(data gen.AlzModel, options *azidentity.DefaultAzureCredentialOptions) (*azidentity.ChainedTokenCredential, diag.Diagnostics) {
	var creds []azcore.TokenCredential
	var diags diag.Diagnostics

	if options == nil {
		options = &azidentity.DefaultAzureCredentialOptions{}
	}

	if data.UseOidc.ValueBool() {
		oidcCred, err := NewOidcCredential(&OidcCredentialOptions{
			ClientOptions: azcore.ClientOptions{
				Cloud: options.Cloud,
			},
			AdditionallyAllowedTenants: options.AdditionallyAllowedTenants,
			TenantID:                   data.TenantId.ValueString(),
			ClientID:                   data.ClientId.ValueString(),
			RequestToken:               data.OidcRequestToken.ValueString(),
			RequestUrl:                 data.OidcRequestUrl.ValueString(),
			Token:                      data.OidcToken.ValueString(),
			TokenFilePath:              data.OidcTokenFilePath.ValueString(),
		})

		if err == nil {
			creds = append(creds, oidcCred)
		} else {
			diags.AddWarning("newDefaultAzureCredential failed to initialize oidc credential:\n\t%s", err.Error())
		}
	}

	envCred, err := azidentity.NewEnvironmentCredential(&azidentity.EnvironmentCredentialOptions{
		ClientOptions:            options.ClientOptions,
		DisableInstanceDiscovery: options.DisableInstanceDiscovery,
	})
	if err == nil {
		creds = append(creds, envCred)
	} else {
		diags.AddWarning("newDefaultAzureCredential failed to initialize environment credential:\n\t%s", err.Error())
	}

	if data.UseMsi.ValueBool() {
		o := &azidentity.ManagedIdentityCredentialOptions{ClientOptions: options.ClientOptions}
		if ID, ok := os.LookupEnv("AZURE_CLIENT_ID"); ok {
			o.ID = azidentity.ClientID(ID)
		}
		miCred, err := newManagedIdentityCredential(o)
		if err == nil {
			creds = append(creds, miCred)
		} else {
			diags.AddWarning("newDefaultAzureCredential failed to initialize msi credential:\n\t%s", err.Error())
		}
	}

	if data.UseCli.ValueBool() {
		cliCred, err := azidentity.NewAzureCLICredential(&azidentity.AzureCLICredentialOptions{
			AdditionallyAllowedTenants: options.AdditionallyAllowedTenants,
			TenantID:                   options.TenantID})
		if err == nil {
			creds = append(creds, cliCred)
		} else {
			diags.AddWarning("newDefaultAzureCredential failed to initialize cli credential:\n\t%s", err.Error())
		}
	}

	if len(creds) == 0 {
		diags.AddError("newDefaultAzureCredential failed to initialize any credential", "None of the credentials were initialized")
		return nil, diags
	}

	chain, err := azidentity.NewChainedTokenCredential(creds, nil)
	if err != nil {
		diags.AddError("newDefaultAzureCredential failed to initialize chained credential:\n\t%s", err.Error())
		return nil, diags
	}

	return chain, nil
}
