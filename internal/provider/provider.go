// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"regexp"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/cloud"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armpolicy"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/matt-FFFFFF/alzlib"
)

// Ensure ScaffoldingProvider satisfies various provider interfaces.
var _ provider.Provider = &AlzProvider{}

// AlzProvider defines the provider implementation.
type AlzProvider struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.
	version string
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
	OidcRequestUrl            types.String `tfsdk:"oidc_request_token"`
	OidcToken                 types.String `tfsdk:"oidc_token"`
	OidcTokenFilePath         types.String `tfsdk:"oidc_token_file_path"`
	SkipProviderRegistration  types.Bool   `tfsdk:"skip_provider_registration"`
	TenantId                  types.String `tfsdk:"tenant_id"`
	UseAlzLib                 types.Bool   `tfsdk:"use_alzlib"`
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
		Attributes: map[string]schema.Attribute{
			"allow_lib_overwrite": schema.BoolAttribute{
				MarkdownDescription: "Whether to allow overwriting of the library by other lib directories.",
				Optional:            true,
			},

			"auxiliary_tenant_ids": schema.ListAttribute{
				MarkdownDescription: "A list of auxiliary tenant ids which should be used.",
				ElementType:         types.StringType,
				Optional:            true,
				Validators: []validator.List{
					listvalidator.ValueStringsAre(
						stringvalidator.RegexMatches(regexp.MustCompile(`^[0-9a-f]{8}-([0-9a-f]{4}-){3}[0-9a-f]{12}$`), "The client id must be a valid lowercase UUID."),
					),
				},
			},

			"client_certificate_password": schema.StringAttribute{
				MarkdownDescription: "The password associated with the client certificate. For use when authenticating as a service principal using a client certificate",
				Optional:            true,
				Sensitive:           true,
			},

			"client_certificate_path": schema.StringAttribute{
				MarkdownDescription: "The path to the client certificate associated with the service principal for use when authenticating as a service principal using a client certificate.",
				Optional:            true,
			},

			"client_id": schema.StringAttribute{
				MarkdownDescription: "The client id which should be used.",
				Optional:            true,
				Validators: []validator.String{
					stringvalidator.RegexMatches(regexp.MustCompile(`^[0-9a-f]{8}-([0-9a-f]{4}-){3}[0-9a-f]{12}$`), "The client id must be a valid lowercase UUID."),
				},
			},

			"client_secret": schema.StringAttribute{
				MarkdownDescription: "The client secret which should be used. For use when authenticating as a service principal using a client secret.",
				Optional:            true,
				Sensitive:           true,
			},

			"environment": schema.StringAttribute{
				MarkdownDescription: "The cloud environment which should be used. Possible values are `public`, `usgovernment` and `china`. Defaults to public.",
				Optional:            true,
				Validators: []validator.String{
					stringvalidator.OneOf("public", "usgovernment", "china"),
				},
			},

			"lib_dirs": schema.ListAttribute{
				MarkdownDescription: "A list of directories to search for ALZ artefacts. The directories will be processed in order.",
				ElementType:         types.StringType,
				Optional:            true,
			},

			"oidc_request_token": schema.StringAttribute{
				MarkdownDescription: "The bearer token for the request to the OIDC provider. For use when authenticating using OpenID Connect.",
				Optional:            true,
				Sensitive:           true,
			},

			"oidc_request_url": schema.StringAttribute{
				MarkdownDescription: "The URL for the OIDC provider from which to request an id token. For use when authenticating as a service principal using OpenID Connect.",
				Optional:            true,
			},

			"oidc_token": schema.StringAttribute{
				MarkdownDescription: "The OIDC id token for use when authenticating as a service principal using OpenID Connect.",
				Optional:            true,
				Sensitive:           true,
			},

			"oidc_token_file_path": schema.StringAttribute{
				MarkdownDescription: "The path to a file containing an OIDC id token for use when authenticating using OpenID Connect.",
				Optional:            true,
			},

			"skip_provider_registration": schema.BoolAttribute{
				MarkdownDescription: "Should the provider skip registering all of the resource providers that it supports, if they're not already registered?",
				Optional:            true,
			},

			"tenant_id": schema.StringAttribute{
				MarkdownDescription: "The Tenant ID which should be used.",
				Optional:            true,
				Validators: []validator.String{
					stringvalidator.RegexMatches(regexp.MustCompile(`^[0-9a-f]{8}-([0-9a-f]{4}-){3}[0-9a-f]{12}$`), "The tenant id must be a valid lowercase UUID."),
				},
			},

			"use_alz_lib": schema.BoolAttribute{
				MarkdownDescription: "Use the built-in ALZ library to resolve archetypes.",
				Optional:            true,
			},

			"use_cli": schema.StringAttribute{
				MarkdownDescription: "Allow Azure CLI to be used for authentication.",
				Optional:            true,
			},

			"use_msi": schema.StringAttribute{
				MarkdownDescription: "Allow managed service identity to be used for authentication.",
				Optional:            true,
			},

			"use_oidc": schema.BoolAttribute{
				MarkdownDescription: "Allow OpenID Connect to be used for authentication.",
				Optional:            true,
			},
		},
	}
}

func (p *AlzProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var data AlzProviderModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	var auxTenants []string
	if !data.AuxiliaryTenantIds.IsNull() {
		if avs := data.AuxiliaryTenantIds.Elements(); len(avs) > 0 {
			auxTenants = make([]string, len(avs))
			for i, v := range avs {
				auxTenants[i] = v.String()
			}
		} else if v := os.Getenv("ARM_AUXILIARY_TENANT_IDS"); v != "" {
			auxTenants = strings.Split(v, ";")
		}
	}

	type env2type struct {
		tp  types.String
		env []string
	}

	// Read the environment variables and set in data
	// if the data is not already set and the environment variable is set
	e2t := []env2type{
		{data.ClientCertificatePassword, []string{"ARM_CLIENT_CERTIFICATE_PASSWORD"}},
		{data.ClientCertificatePath, []string{"ARM_CLIENT_CERTIFICATE_PATH"}},
		{data.ClientId, []string{"ARM_CLIENT_ID"}},
		{data.ClientSecret, []string{"ARM_CLIENT_SECRET"}},
		{data.Environment, []string{"ARM_ENVIRONMENT"}},
		{data.OidcRequestToken, []string{"ARM_OIDC_REQUEST_TOKEN", "ACTIONS_ID_TOKEN_REQUEST_TOKEN"}},
		{data.OidcRequestUrl, []string{"ARM_OIDC_REQUEST_URL", "ACTIONS_ID_TOKEN_REQUEST_URL"}},
		{data.OidcToken, []string{"ARM_OIDC_TOKEN"}},
		{data.OidcTokenFilePath, []string{"ARM_OIDC_TOKEN_FILE_PATH"}},
		{data.TenantId, []string{"ARM_TENANT_ID"}},
	}

	for _, e := range e2t {
		if e.tp.IsNull() {
			v := getFirstSetEnvVar(e.env...)
			if v == "" {
				continue
			}
			e.tp = types.StringValue(v)
		}
	}

	var cloudConfig cloud.Configuration
	env := data.Environment.String()
	switch strings.ToLower(env) {
	case "public":
		cloudConfig = cloud.AzurePublic
	case "usgovernment":
		cloudConfig = cloud.AzureGovernment
	case "china":
		cloudConfig = cloud.AzureChina
	default:
		resp.Diagnostics.AddError("unknown `environment` specified: %q", env)
		return
	}

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
		os.Setenv("AZURE_CLIENT_CERTIFICATE_PATH", data.ClientCertificatePath.String())
	}
	if !data.ClientCertificatePassword.IsNull() {
		// #nosec G104
		os.Setenv("AZURE_CLIENT_CERTIFICATE_PASSWORD", data.ClientCertificatePassword.String())
	}
	if len(auxTenants) != 0 {
		// #nosec G104
		os.Setenv("AZURE_ADDITIONALLY_ALLOWED_TENANTS", strings.Join(auxTenants, ";"))
	}

	option := &azidentity.DefaultAzureCredentialOptions{
		AdditionallyAllowedTenants: auxTenants,
		ClientOptions: azcore.ClientOptions{
			Cloud: cloudConfig,
		},
		TenantID: data.TenantId.ValueString(),
	}

	cred, d := newDefaultAzureCredential(data, option)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}

	popts := new(policy.ClientOptions)
	popts.PerRetryPolicies = append(popts.PerRetryPolicies, withUserAgent(fmt.Sprintf("Terraform Azure/AlzLib provider: %s", p.version)))

	alz := alzlib.NewAlzLib()
	cf, err := armpolicy.NewClientFactory("", cred, popts)
	if err != nil {
		resp.Diagnostics.AddError("failed to create Azure Policy client factory: %v", err.Error())
		return
	}
	alz.AddPolicyClient(cf)

	if data.UseAlzLib.IsNull() {
		data.UseAlzLib = types.BoolValue(true)
	}

	if data.AllowLibOverwrite.IsNull() {
		data.AllowLibOverwrite = types.BoolValue(false)
	}

	alz.Options = &alzlib.AlzLibOptions{
		AllowOverwrite: data.AllowLibOverwrite.ValueBool(),
	}

	libdirfs := make([]fs.FS, 2)
	if data.UseAlzLib.ValueBool() {
		libdirfs[0] = alzlib.Lib
	}
	if !data.LibDirs.IsNull() {
		for _, v := range data.LibDirs.Elements() {
			libdirfs = append(libdirfs, os.DirFS(v.String()))
		}
	}

	if err := alz.Init(ctx, libdirfs...); err != nil {
		resp.Diagnostics.AddError("failed to initialize AlzLib: %v", err.Error())
		return
	}

	resp.DataSourceData = alz
	resp.ResourceData = alz
}

func (p *AlzProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewExampleResource,
	}
}

func (p *AlzProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewExampleDataSource,
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
			RequestUrl:                 data.ClientCertificatePassword.ValueString(),
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
