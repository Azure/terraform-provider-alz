// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package provider

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/Azure/alzlib"
	"github.com/Azure/alzlib/cache"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/cloud"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
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

	// Defaults for non-compliance message placeholder substitution settings.
	// These are applied at the provider level when the user does not set them
	// in the `non_compliance_message_substitution_settings` block.
	defaultEnforcementModePlaceholder = "{enforcementMode}"
	defaultEnforcedReplacement        = "must"
	defaultNotEnforcedReplacement     = "should"
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

	authOptions, err := data.AuthOption(ctx, azcore.ClientOptions{
		Retry: policy.RetryOptions{
			MaxRetries: math.MaxInt16,
		},
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to configure cloud environment", err.Error())
		return
	}
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

	// If a cache file was supplied and exists, load it and inject into AlzLib so
	// that built-in policy and policy set definitions can be served from the cache
	// without making Azure API calls during Init.
	cacheFileName := data.CacheFileName.ValueString()
	if cacheFileName != "" {
		if err := loadCacheFile(ctx, alz, cacheFileName); err != nil {
			resp.Diagnostics.AddError("Failed to load cache file", err.Error())
			return
		}
	}

	// Init alzlib
	if err := alz.Init(ctx, libRefs...); err != nil {
		resp.Diagnostics.AddError("Failed to initialize AlzLib", err.Error())
		return
	}

	// If requested, persist the built-in cache to disk so that subsequent runs
	// can use it. This must happen after Init so that the AlzLib has been
	// populated with the built-in definitions referenced by the library.
	if cacheFileName != "" && data.CacheFileSaveEnabled.ValueBool() {
		if err := saveCacheFile(ctx, alz, cacheFileName); err != nil {
			resp.Diagnostics.AddError("Failed to save cache file", err.Error())
			return
		}
	}

	// Drop the cache to free up RAM now that the AlzLib has been populated.
	if cacheFileName != "" {
		alz.AddCache(nil)
		tflog.Debug(ctx, "Dropped AlzLib built-in cache to free memory")
	}

	// Store the alz pointer in the provider struct so we don't have to do all this work every time `.Configure` is called.
	// Due to fetch from Azure, it takes approx 30 seconds each time and is called 4-5 time during a single acceptance test.
	clientOpts := []clients.Option{
		clients.WithAlzLib(alz),
		clients.WithSuppressWarningPolicyRoleAssignments(data.SuppressWarningPolicyRoleAssignments.ValueBool()),
	}

	// Parse non-compliance message substitution settings, applying provider-level
	// defaults when the block (or any individual attribute) is not configured.
	placeholder := defaultEnforcementModePlaceholder
	enforcedRepl := defaultEnforcedReplacement
	notEnforcedRepl := defaultNotEnforcedReplacement
	ncmSubSettings := data.NonComplianceMessageSubstitutionSettings
	if !ncmSubSettings.IsNull() && !ncmSubSettings.IsUnknown() {
		if v := ncmSubSettings.EnforcementModePlaceholder; !v.IsNull() && !v.IsUnknown() && v.ValueString() != "" {
			placeholder = v.ValueString()
		}
		if v := ncmSubSettings.EnforcedReplacement; !v.IsNull() && !v.IsUnknown() && v.ValueString() != "" {
			enforcedRepl = v.ValueString()
		}
		if v := ncmSubSettings.NotEnforcedReplacement; !v.IsNull() && !v.IsUnknown() && v.ValueString() != "" {
			notEnforcedRepl = v.ValueString()
		}
	}
	clientOpts = append(clientOpts, clients.WithNonComplianceMessageSubstitutionSettings(placeholder, enforcedRepl, notEnforcedRepl))

	p.data = clients.NewClient(clientOpts...)
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

	// Do not save the cache file by default.
	if data.CacheFileSaveEnabled.IsNull() {
		data.CacheFileSaveEnabled = types.BoolValue(false)
	}
}

// loadCacheFile loads the gzipped cache file at the given path and injects it
// into the AlzLib. If the file does not exist, this is treated as a no-op so
// that the cache file can be created on first run when used with
// `cache_file_save_enabled = true`.
func loadCacheFile(ctx context.Context, alz *alzlib.AlzLib, path string) error {
	f, err := os.Open(path) // #nosec G304 -- path is provided by the operator via provider config.
	if err != nil {
		if os.IsNotExist(err) {
			tflog.Debug(ctx, "Cache file does not exist, skipping load", map[string]interface{}{
				"cache_file_name": path,
			})
			return nil
		}
		return fmt.Errorf("opening cache file %q: %w", path, err)
	}
	defer f.Close() // #nosec G307 -- read-only.

	c, err := cache.NewCache(f)
	if err != nil {
		return fmt.Errorf("reading cache file %q: %w", path, err)
	}
	alz.AddCache(c)
	tflog.Debug(ctx, "Loaded AlzLib built-in cache from file", map[string]interface{}{
		"cache_file_name": path,
	})
	return nil
}

// saveCacheFile exports the built-in policy and policy set definitions from the
// AlzLib and writes them to the given path as a gzipped JSON file. The write is
// performed via a temporary file in the same directory and renamed atomically
// to avoid leaving a corrupt cache file if the process is interrupted.
func saveCacheFile(ctx context.Context, alz *alzlib.AlzLib, path string) error {
	c := alz.ExportBuiltInCache()

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating cache directory %q: %w", dir, err)
	}

	tmp, err := os.CreateTemp(dir, filepath.Base(path)+".tmp-*")
	if err != nil {
		return fmt.Errorf("creating temporary cache file in %q: %w", dir, err)
	}
	tmpName := tmp.Name()
	// Best-effort cleanup if we don't make it to the rename.
	defer func() { _ = os.Remove(tmpName) }()

	if err := c.Save(tmp); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("writing cache file %q: %w", path, err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("closing cache file %q: %w", tmpName, err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("renaming cache file %q to %q: %w", tmpName, path, err)
	}
	tflog.Debug(ctx, "Saved AlzLib built-in cache to file", map[string]interface{}{
		"cache_file_name": path,
	})
	return nil
}
