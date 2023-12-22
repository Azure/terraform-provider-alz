// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package provider

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"regexp"
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
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

const (
	userAgentBase = "AzureTerraformAlzProvider"
)

// Ensure ScaffoldingProvider satisfies various provider interfaces.
var _ provider.Provider = &AlzProvider{}

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

// AlzProviderModel describes the provider data model.
type AlzProviderModel struct {
	AllowLibOverwrite         types.Bool   `tfsdk:"allow_lib_overwrite"`
	AuxiliaryTenantIds        types.List   `tfsdk:"auxiliary_tenant_ids"`
	ClientCertificatePassword types.String `tfsdk:"client_certificate_password"`
	ClientCertificatePath     types.String `tfsdk:"client_certificate_path"`
	ClientId                  types.String `tfsdk:"client_id"`
	ClientSecret              types.String `tfsdk:"client_secret"`
	Environment               types.String `tfsdk:"environment"`
	LibDirs                   types.List   `tfsdk:"lib_dirs"`
	OidcRequestToken          types.String `tfsdk:"oidc_request_token"`
	OidcRequestUrl            types.String `tfsdk:"oidc_request_url"`
	OidcToken                 types.String `tfsdk:"oidc_token"`
	OidcTokenFilePath         types.String `tfsdk:"oidc_token_file_path"`
	SkipProviderRegistration  types.Bool   `tfsdk:"skip_provider_registration"`
	TenantId                  types.String `tfsdk:"tenant_id"`
	UseAlzLib                 types.Bool   `tfsdk:"use_alz_lib"`
	UseCli                    types.Bool   `tfsdk:"use_cli"`
	UseMsi                    types.Bool   `tfsdk:"use_msi"`
	UseOidc                   types.Bool   `tfsdk:"use_oidc"`
}

func (p *AlzProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "alz"
	resp.Version = p.version
}

func (p *AlzProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "ALZ provider to generate archetype data for use with the ALZ Terraform module.",

		Attributes: map[string]schema.Attribute{
			"allow_lib_overwrite": schema.BoolAttribute{
				MarkdownDescription: "Whether to allow overwriting of the library by other lib directories. Default is `false`.",
				Optional:            true,
			},

			"auxiliary_tenant_ids": schema.ListAttribute{
				MarkdownDescription: "A list of auxiliary tenant ids which should be used. If not specified, value will be attempted to be read from the `ARM_AUXILIARY_TENANT_IDS` environment variable. When configuring from the environment, use a semicolon as a delimiter.",
				ElementType:         types.StringType,
				Optional:            true,
				Validators: []validator.List{
					listvalidator.ValueStringsAre(
						stringvalidator.RegexMatches(regexp.MustCompile(`^[0-9a-f]{8}-([0-9a-f]{4}-){3}[0-9a-f]{12}$`), "The client id must be a valid lowercase UUID."),
					),
				},
			},

			"client_certificate_password": schema.StringAttribute{
				MarkdownDescription: "The password associated with the client certificate. For use when authenticating as a service principal using a client certificate. If not specified, value will be attempted to be read from the `ARM_CLIENT_CERTIFICATE_PASSWORD` environment variable.",
				Optional:            true,
				Sensitive:           true,
			},

			"client_certificate_path": schema.StringAttribute{
				MarkdownDescription: "The path to the client certificate associated with the service principal for use when authenticating as a service principal using a client certificate. If not specified, value will be attempted to be read from the `ARM_CLIENT_CERTIFICATE_PATH` environment variable.",
				Optional:            true,
			},

			"client_id": schema.StringAttribute{
				MarkdownDescription: "The client id which should be used. For use when authenticating as a service principal. If not specified, value will be attempted to be read from the `ARM_CLIENT_ID` environment variable.",
				Optional:            true,
				Validators: []validator.String{
					stringvalidator.RegexMatches(regexp.MustCompile(`^[0-9a-f]{8}-([0-9a-f]{4}-){3}[0-9a-f]{12}$`), "The client id must be a valid lowercase UUID."),
				},
			},

			"client_secret": schema.StringAttribute{
				MarkdownDescription: "The client secret which should be used. For use when authenticating as a service principal using a client secret. If not specified, value will be attempted to be read from the `ARM_CLIENT_SECRET` environment variable.",
				Optional:            true,
				Sensitive:           true,
			},

			"environment": schema.StringAttribute{
				MarkdownDescription: "The cloud environment which should be used. Possible values are `public`, `usgovernment` and `china`. Defaults to `public`. If not specified, value will be attempted to be read from the `ARM_ENVIRONMENT` environment variable.",
				Optional:            true,
				Validators: []validator.String{
					stringvalidator.OneOf("public", "usgovernment", "china"),
				},
			},

			"lib_dirs": schema.ListAttribute{
				MarkdownDescription: "A list of directories to search for ALZ artefacts. The directories will be processed in order.",
				ElementType:         types.StringType,
				Optional:            true,
				Validators: []validator.List{
					listvalidator.UniqueValues(),
				},
			},

			"oidc_request_token": schema.StringAttribute{
				MarkdownDescription: "The bearer token for the request to the OIDC provider. For use when authenticating using OpenID Connect. If not specified, value will be attempted to be read from the first non-empty value of the `ARM_OIDC_REQUEST_TOKEN` and `ACTIONS_ID_TOKEN_REQUEST_TOKEN` environment variables.",
				Optional:            true,
				Sensitive:           true,
			},

			"oidc_request_url": schema.StringAttribute{
				MarkdownDescription: "The URL for the OIDC provider from which to request an id token. For use when authenticating as a service principal using OpenID Connect. If not specified, value will be attempted to be read from the first non-empty value of the `ARM_OIDC_REQUEST_URL` and `ACTIONS_ID_TOKEN_REQUEST_URL` environment variables.",
				Optional:            true,
			},

			"oidc_token": schema.StringAttribute{
				MarkdownDescription: "The OIDC id token for use when authenticating as a service principal using OpenID Connect. If not specified, value will be attempted to be read from the `ARM_OIDC_TOKEN` environment variable.",
				Optional:            true,
				Sensitive:           true,
			},

			"oidc_token_file_path": schema.StringAttribute{
				MarkdownDescription: "The path to a file containing an OIDC id token for use when authenticating using OpenID Connect. If not specified, value will be attempted to be read from the `ARM_OIDC_TOKEN_FILE_PATH` environment variable.",
				Optional:            true,
			},

			"skip_provider_registration": schema.BoolAttribute{
				MarkdownDescription: "Should the provider skip registering all of the resource providers that it supports, if they're not already registered? Default is `false`. If not specified, value will be attempted to be read from the `ARM_SKIP_PROVIDER_REGISTRATION` environment variable.",
				Optional:            true,
			},

			"tenant_id": schema.StringAttribute{
				MarkdownDescription: "The Tenant ID which should be used. If not specified, value will be attempted to be read from the `ARM_TENANT_ID` environment variable.",
				Optional:            true,
				Validators: []validator.String{
					stringvalidator.RegexMatches(regexp.MustCompile(`^[0-9a-f]{8}-([0-9a-f]{4}-){3}[0-9a-f]{12}$`), "The tenant id must be a valid lowercase UUID."),
				},
			},

			"use_alz_lib": schema.BoolAttribute{
				MarkdownDescription: "Use the built-in ALZ library to resolve archetypes. Default is `true`.",
				Optional:            true,
			},

			"use_cli": schema.BoolAttribute{
				MarkdownDescription: "Allow Azure CLI to be used for authentication. Default is `true`. If not specified, value will be attempted to be read from the `ARM_USE_CLI` environment variable.",
				Optional:            true,
			},

			"use_msi": schema.BoolAttribute{
				MarkdownDescription: "Allow managed service identity to be used for authentication. Default is `false`. If not specified, value will be attempted to be read from the `ARM_USE_MSI` environment variable.",
				Optional:            true,
			},

			"use_oidc": schema.BoolAttribute{
				MarkdownDescription: "Allow OpenID Connect to be used for authentication. Default is `false`. If not specified, value will be attempted to be read from the `ARM_USE_OIDC` environment variable.",
				Optional:            true,
			},
		},
	}
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

	var data AlzProviderModel

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
	configureDefaults(&data)

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

	// Configure clients

	// Create the fs.FS library file systems based on the configuration.
	libdirfs := make([]fs.FS, 0)
	if data.UseAlzLib.ValueBool() {
		libdirfs = append(libdirfs, alzlib.Lib)
	}
	if len(data.LibDirs.Elements()) != 0 {
		// We turn the list of elements into a list of strings,
		// if we use the Elements() method, we get a list of *attr.Value and the .String() method
		// results in a string wrapped in double quotes.
		dirs := make([]string, 0, len(data.LibDirs.Elements()))
		if diags := data.LibDirs.ElementsAs(ctx, &dirs, false); diags.HasError() {
			resp.Diagnostics = append(resp.Diagnostics, diags...)
			return
		}
		for _, v := range dirs {
			libdirfs = append(libdirfs, os.DirFS(v))
		}
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()
	if err := alz.Init(ctx, libdirfs...); err != nil {
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
		NewPolicyRoleAssignmentResource,
	}
}

func (p *AlzProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewArchetypeDataSource,
		NewArchetypeKeysDataSource,
	}
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &AlzProvider{
			version: version,
		}
	}
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
func configureAuxTenants(ctx context.Context, data *AlzProviderModel) diag.Diagnostics {
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
func configureFromEnvironment(data *AlzProviderModel) {
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
func configureAzIdentityEnvironment(data *AlzProviderModel) {
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
func configureAlzLib(token *azidentity.ChainedTokenCredential, data AlzProviderModel, userAgent string) (*alzlib.AlzLib, diag.Diagnostics) {
	var diags diag.Diagnostics
	popts := new(policy.ClientOptions)
	popts.DisableRPRegistration = data.SkipProviderRegistration.ValueBool()
	popts.PerRetryPolicies = append(popts.PerRetryPolicies, withUserAgent(userAgent))

	alz := alzlib.NewAlzLib()
	cf, err := armpolicy.NewClientFactory("", token, popts)
	if err != nil {
		diags.AddError("failed to create Azure Policy client factory: %v", err.Error())
		return nil, diags
	}

	alz.AddPolicyClient(cf)

	alz.Options.AllowOverwrite = data.AllowLibOverwrite.ValueBool()

	return alz, diags
}

func getClients(token *azidentity.ChainedTokenCredential, data AlzProviderModel, userAgent string) (*AlzProviderClients, diag.Diagnostics) {
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
func getTokenCredential(data AlzProviderModel) (*azidentity.ChainedTokenCredential, diag.Diagnostics) {
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
func configureDefaults(data *AlzProviderModel) {
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

	// Use internal AlzLib reference library by default.
	if data.UseAlzLib.IsNull() {
		data.UseAlzLib = types.BoolValue(true)
	}

	// Do not allow library overwrite by default.
	if data.AllowLibOverwrite.IsNull() {
		data.AllowLibOverwrite = types.BoolValue(false)
	}
}

func newDefaultAzureCredential(data AlzProviderModel, options *azidentity.DefaultAzureCredentialOptions) (*azidentity.ChainedTokenCredential, diag.Diagnostics) {
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
