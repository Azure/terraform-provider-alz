// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package provider

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/Azure/alzlib"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/cloud"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armpolicy"
	"github.com/Azure/entrauth/aztfauth"
	"github.com/Azure/terraform-provider-alz/internal/aztfschema"
	"github.com/Azure/terraform-provider-alz/internal/clients"
	"github.com/Azure/terraform-provider-alz/internal/gen"
	"github.com/Azure/terraform-provider-alz/internal/services"
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

const (
	userAgentBase = "AzureTerraformAlzProvider"
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
	data    *clients.Client
}

// AlzModel is the data model for the ALZ provider.
// It embeds the generated ALZ model and the Entra authentication model.
type AlzModel struct {
	gen.AlzModel
	aztfschema.AuthModelWithSubscriptionID
}

func (p *AlzProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "alz"
	resp.Version = p.version
}

func (p *AlzProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	genSchema := gen.AlzProviderSchema(ctx)
	attrs := aztfschema.NewGenerator().WithAuthAttrs().WithSubscriptionID().Merge(genSchema.Attributes)
	genSchema.Attributes = attrs
	resp.Schema = genSchema
}

func (p *AlzProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	tflog.Debug(ctx, "Provider configuration started")

	if p.data != nil {
		tflog.Debug(ctx, "Provider AlzLib already present, skipping configuration")
		resp.DataSourceData = p.data
		resp.ResourceData = p.data
		return
	}

	tflog.Debug(ctx, "Provider AlzLib not present, beginning configuration")

	var data AlzModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Read the environment variables and set in data
	// if the data is not already set and the environment variable is set.
	data.ConfigureFromEnv()

	// Set the go sdk's azidentity specific environment variables
	configureAzIdentityEnvironment(&data)

	// For remaining null values, set opinionated defaults
	data.SetOpinionatedDefaults()
	configureDefaults(ctx, &data)

	authOptions := data.AuthOption(azcore.ClientOptions{})
	cred, err := aztfauth.NewCredential(authOptions)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create Azure token credential", err.Error())
		return
	}

	// Create the AlzLib.
	alz, diags := configureAlzLib(
		cred,
		data,
		authOptions.Cloud,
		fmt.Sprintf("%s/%s",
			userAgentBase,
			p.version),
	)
	resp.Diagnostics = append(resp.Diagnostics, diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Convert the supplied libraries to alzlib.LibraryReferences
	libRefs, diags := generateLibraryDefinitions(ctx, &data)
	resp.Diagnostics = append(resp.Diagnostics, diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	r := rand.Intn(math.MaxInt32)
	alzlib.Instance.Store(uint32(r))
	tflog.Debug(ctx, "Stored random ID for AlzLib instance", map[string]interface{}{
		"instance": r,
	})

	// Fetch the library dependencies if enabled.
	// If not, the refs passed to alzlib.Init() will be fetched on demand without dependencies.
	if data.LibraryFetchDependencies.ValueBool() {
		var err error
		tflog.Debug(ctx, "Begin fetch library dependencies", map[string]interface{}{
			"library_references": libRefs,
		})
		libRefs, err = libRefs.FetchWithDependencies(ctx)
		if err != nil {
			resp.Diagnostics.AddError("Failed to fetch library dependencies", err.Error())
			return
		}
		tflog.Debug(ctx, "End fetch library dependencies", map[string]interface{}{
			"library_references": libRefs,
		})
	}

	// Init alzlib
	if err := alz.Init(ctx, libRefs...); err != nil {
		resp.Diagnostics.AddError("Failed to initialize AlzLib", err.Error())
		return
	}

	// Store the alz pointer in the provider struct so we don't have to do all this work every time `.Configure` is called.
	// Due to fetch from Azure, it takes approx 30 seconds each time and is called 4-5 time during a single acceptance test.
	p.data = clients.NewClient(
		clients.WithAlzLib(alz),
		clients.WithSuppressWarningPolicyRoleAssignments(data.SuppressWarningPolicyRoleAssignments.ValueBool()),
	)
	resp.DataSourceData = p.data
	resp.ResourceData = p.data
	tflog.Debug(ctx, "Provider configuration finished")
}

func (p *AlzProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{}
}

func (p *AlzProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		services.NewArchitectureDataSource,
		services.NewMetadataDataSource,
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

func generateLibraryDefinitions(ctx context.Context, data *AlzModel) (alzlib.LibraryReferences, diag.Diagnostics) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	var diags diag.Diagnostics

	alzLibRefs := make([]gen.LibraryReferencesValue, len(data.LibraryReferences.Elements()))
	diags = data.LibraryReferences.ElementsAs(ctx, &alzLibRefs, false)
	if diags.HasError() {
		return nil, diags
	}

	libRefs := make(alzlib.LibraryReferences, len(alzLibRefs))
	for i, libRef := range alzLibRefs {
		if libRef.CustomUrl.IsNull() {
			libRefs[i] = alzlib.NewAlzLibraryReference(libRef.Path.ValueString(), libRef.Ref.ValueString())
			continue
		}
		libRefs[i] = alzlib.NewCustomLibraryReference(libRef.CustomUrl.ValueString())
	}
	return libRefs, nil
}

func getFirstSetEnvVar(envVars ...string) string {
	for _, envVar := range envVars {
		if val := os.Getenv(envVar); val != "" {
			return val
		}
	}
	return ""
}

// configureFromEnvironment sets the provider data from environment variables.
func configureFromEnvironment(data *AlzModel) {
	if val := getFirstSetEnvVar("ARM_SKIP_PROVIDER_REGISTRATION"); val != "" && data.SkipProviderRegistration.IsNull() {
		data.SkipProviderRegistration = types.BoolValue(str2Bool(val))
	}

	if val := getFirstSetEnvVar("ALZ_PROVIDER_SUPPRESS_WARNING_POLICY_ROLE_ASSIGNMENTS"); val != "" && data.SuppressWarningPolicyRoleAssignments.IsNull() {
		data.SuppressWarningPolicyRoleAssignments = types.BoolValue(str2Bool(val))
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
func configureAzIdentityEnvironment(data *AlzModel) {
	// Maps the auth related environment variables used in the provider to what azidentity honors.
	if !data.TenantID.IsNull() {
		// #nosec G104
		os.Setenv("AZURE_TENANT_ID", data.TenantID.ValueString())
	}
	if !data.ClientID.IsNull() {
		// #nosec G104
		os.Setenv("AZURE_CLIENT_ID", data.ClientID.ValueString())
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
	if len(data.AuxiliaryTenantIDs.Elements()) != 0 {
		auxTenants := listElementsToStrings(data.AuxiliaryTenantIDs.Elements())
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
func configureAlzLib(token azcore.TokenCredential, data AlzModel, cloudConfig cloud.Configuration, userAgent string) (*alzlib.AlzLib, diag.Diagnostics) {
	var diags diag.Diagnostics
	popts := new(arm.ClientOptions)
	popts.DisableRPRegistration = data.SkipProviderRegistration.ValueBool()
	popts.PerRetryPolicies = append(popts.PerRetryPolicies, withUserAgent(userAgent))
	popts.Cloud = cloudConfig

	opts := &alzlib.Options{
		AllowOverwrite:        data.LibraryOverwriteEnabled.ValueBool(),
		Parallelism:           10,
		UniqueRoleDefinitions: !data.RoleDefinitionsUseSuppliedNamesEnabled.ValueBool(),
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

// configureDefaults sets default values if they aren't already set.
func configureDefaults(_ context.Context, data *AlzModel) {

	// Do not skip provider registration by default.
	if data.SkipProviderRegistration.IsNull() {
		data.SkipProviderRegistration = types.BoolValue(false)
	}

	// Do not allow library overwrite by default.
	if data.LibraryOverwriteEnabled.IsNull() {
		data.LibraryOverwriteEnabled = types.BoolValue(false)
	}

	// Automatically download dependencies by default.
	if data.LibraryFetchDependencies.IsNull() {
		data.LibraryFetchDependencies = types.BoolValue(true)
	}

	// Do not skip warning policy role assignments by default.
	if data.SuppressWarningPolicyRoleAssignments.IsNull() {
		data.SuppressWarningPolicyRoleAssignments = types.BoolValue(false)
	}
}
