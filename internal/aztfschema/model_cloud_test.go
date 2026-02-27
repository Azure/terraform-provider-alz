package aztfschema

import (
	"context"
	"os"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/cloud"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Test_getCloudConfiguration_PredefinedEnvironments tests cloud configuration resolution for predefined environments.
func Test_getCloudConfiguration_PredefinedEnvironments(t *testing.T) {
	tests := []struct {
		name        string
		environment string
		wantConfig  cloud.Configuration
		wantErr     bool
	}{
		{
			name:        "public cloud",
			environment: "public",
			wantConfig:  cloud.AzurePublic,
			wantErr:     false,
		},
		{
			name:        "public cloud uppercase",
			environment: "PUBLIC",
			wantConfig:  cloud.AzurePublic,
			wantErr:     false,
		},
		{
			name:        "us government cloud",
			environment: "usgovernment",
			wantConfig:  cloud.AzureGovernment,
			wantErr:     false,
		},
		{
			name:        "china cloud",
			environment: "china",
			wantConfig:  cloud.AzureChina,
			wantErr:     false,
		},
		{
			name:        "empty defaults to public",
			environment: "",
			wantConfig:  cloud.AzurePublic,
			wantErr:     false,
		},
		{
			name:        "whitespace defaults to public",
			environment: "  ",
			wantConfig:  cloud.AzurePublic,
			wantErr:     false,
		},
		{
			name:        "invalid environment",
			environment: "invalid",
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &AuthModel{
				Environment: types.StringValue(tt.environment),
			}

			got, err := m.getCloudConfiguration(context.Background())
			if (err != nil) != tt.wantErr {
				t.Errorf("getCloudConfiguration() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if got.ActiveDirectoryAuthorityHost != tt.wantConfig.ActiveDirectoryAuthorityHost {
					t.Errorf("getCloudConfiguration() ActiveDirectoryAuthorityHost = %v, want %v",
						got.ActiveDirectoryAuthorityHost, tt.wantConfig.ActiveDirectoryAuthorityHost)
				}
			}
		})
	}
}

// Test_buildCustomCloudConfiguration_Success tests successful custom cloud configuration.
func Test_buildCustomCloudConfiguration_Success(t *testing.T) {
	ctx := context.Background()
	endpoint := EndpointModel{
		ResourceManagerEndpoint:      types.StringValue("https://management.example.com/"),
		ActiveDirectoryAuthorityHost: types.StringValue("https://login.example.com/"),
		ResourceManagerAudience:      types.StringValue("https://management.core.example.com/"),
	}

	endpointValue, diags := types.ObjectValueFrom(ctx, map[string]attr.Type{
		"resource_manager_endpoint":        types.StringType,
		"active_directory_authority_host":  types.StringType,
		"resource_manager_audience":        types.StringType,
	}, endpoint)
	if diags.HasError() {
		t.Fatalf("Failed to create endpoint object: %v", diags.Errors())
	}

	endpointList := types.ListValueMust(
		types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"resource_manager_endpoint":        types.StringType,
				"active_directory_authority_host":  types.StringType,
				"resource_manager_audience":        types.StringType,
			},
		},
		[]attr.Value{endpointValue},
	)

	m := &AuthModel{
		Environment: types.StringValue("custom"),
		Endpoint:    endpointList,
	}

	got, err := m.buildCustomCloudConfiguration(ctx)
	if err != nil {
		t.Fatalf("buildCustomCloudConfiguration() error = %v", err)
	}

	if got.ActiveDirectoryAuthorityHost != "https://login.example.com/" {
		t.Errorf("ActiveDirectoryAuthorityHost = %v, want https://login.example.com/",
			got.ActiveDirectoryAuthorityHost)
	}

	if got.Services[cloud.ResourceManager].Endpoint != "https://management.example.com/" {
		t.Errorf("ResourceManager.Endpoint = %v, want https://management.example.com/",
			got.Services[cloud.ResourceManager].Endpoint)
	}

	if got.Services[cloud.ResourceManager].Audience != "https://management.core.example.com/" {
		t.Errorf("ResourceManager.Audience = %v, want https://management.core.example.com/",
			got.Services[cloud.ResourceManager].Audience)
	}
}

// Test_buildCustomCloudConfiguration_MissingEndpoint tests error when endpoint is not provided.
func Test_buildCustomCloudConfiguration_MissingEndpoint(t *testing.T) {
	m := &AuthModel{
		Environment: types.StringValue("custom"),
		Endpoint:    types.ListNull(types.ObjectType{}),
	}

	_, err := m.buildCustomCloudConfiguration(context.Background())
	if err == nil {
		t.Error("buildCustomCloudConfiguration() expected error when endpoint is null, got nil")
	}

	if err.Error() != "endpoint configuration is required when environment is set to 'custom'" {
		t.Errorf("buildCustomCloudConfiguration() error = %v, want 'endpoint configuration is required when environment is set to 'custom''", err)
	}
}

// Test_buildCustomCloudConfiguration_EmptyEndpoint tests error when endpoint list is empty.
func Test_buildCustomCloudConfiguration_EmptyEndpoint(t *testing.T) {
	emptyList := types.ListValueMust(
		types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"resource_manager_endpoint":        types.StringType,
				"active_directory_authority_host":  types.StringType,
				"resource_manager_audience":        types.StringType,
			},
		},
		[]attr.Value{},
	)

	m := &AuthModel{
		Environment: types.StringValue("custom"),
		Endpoint:    emptyList,
	}

	_, err := m.buildCustomCloudConfiguration(context.Background())
	if err == nil {
		t.Error("buildCustomCloudConfiguration() expected error when endpoint list is empty, got nil")
	}
}

// Test_buildCustomCloudConfiguration_MissingFields tests validation of required endpoint fields.
func Test_buildCustomCloudConfiguration_MissingFields(t *testing.T) {
	tests := []struct {
		name          string
		endpoint      EndpointModel
		wantErrSubstr string
	}{
		{
			name: "missing resource_manager_endpoint",
			endpoint: EndpointModel{
				ResourceManagerEndpoint:      types.StringNull(),
				ActiveDirectoryAuthorityHost: types.StringValue("https://login.example.com/"),
				ResourceManagerAudience:      types.StringValue("https://management.core.example.com/"),
			},
			wantErrSubstr: "resource_manager_endpoint",
		},
		{
			name: "missing active_directory_authority_host",
			endpoint: EndpointModel{
				ResourceManagerEndpoint:      types.StringValue("https://management.example.com/"),
				ActiveDirectoryAuthorityHost: types.StringNull(),
				ResourceManagerAudience:      types.StringValue("https://management.core.example.com/"),
			},
			wantErrSubstr: "active_directory_authority_host",
		},
		{
			name: "missing resource_manager_audience",
			endpoint: EndpointModel{
				ResourceManagerEndpoint:      types.StringValue("https://management.example.com/"),
				ActiveDirectoryAuthorityHost: types.StringValue("https://login.example.com/"),
				ResourceManagerAudience:      types.StringNull(),
			},
			wantErrSubstr: "resource_manager_audience",
		},
		{
			name: "empty string resource_manager_endpoint",
			endpoint: EndpointModel{
				ResourceManagerEndpoint:      types.StringValue(""),
				ActiveDirectoryAuthorityHost: types.StringValue("https://login.example.com/"),
				ResourceManagerAudience:      types.StringValue("https://management.core.example.com/"),
			},
			wantErrSubstr: "resource_manager_endpoint",
		},
		{
			name: "all fields missing",
			endpoint: EndpointModel{
				ResourceManagerEndpoint:      types.StringNull(),
				ActiveDirectoryAuthorityHost: types.StringNull(),
				ResourceManagerAudience:      types.StringNull(),
			},
			wantErrSubstr: "resource_manager_endpoint",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			endpointValue, diags := types.ObjectValueFrom(ctx, map[string]attr.Type{
				"resource_manager_endpoint":        types.StringType,
				"active_directory_authority_host":  types.StringType,
				"resource_manager_audience":        types.StringType,
			}, tt.endpoint)
			if diags.HasError() {
				t.Fatalf("Failed to create endpoint object: %v", diags.Errors())
			}

			endpointList := types.ListValueMust(
				types.ObjectType{
					AttrTypes: map[string]attr.Type{
						"resource_manager_endpoint":        types.StringType,
						"active_directory_authority_host":  types.StringType,
						"resource_manager_audience":        types.StringType,
					},
				},
				[]attr.Value{endpointValue},
			)

			m := &AuthModel{
				Environment: types.StringValue("custom"),
				Endpoint:    endpointList,
			}

			_, err := m.buildCustomCloudConfiguration(ctx)
			if err == nil {
				t.Error("buildCustomCloudConfiguration() expected error for missing fields, got nil")
				return
			}

			if err.Error() == "" || !contains(err.Error(), tt.wantErrSubstr) {
				t.Errorf("buildCustomCloudConfiguration() error = %v, want error containing %q",
					err, tt.wantErrSubstr)
			}
		})
	}
}

// Test_configureEndpointFromEnv tests environment variable configuration.
func Test_configureEndpointFromEnv(t *testing.T) {
	// Save original environment variables
	originalRM := os.Getenv("ARM_RESOURCE_MANAGER_ENDPOINT")
	originalAD := os.Getenv("ARM_ACTIVE_DIRECTORY_AUTHORITY_HOST")
	originalAudience := os.Getenv("ARM_RESOURCE_MANAGER_AUDIENCE")

	// Restore environment after test
	defer func() {
		os.Setenv("ARM_RESOURCE_MANAGER_ENDPOINT", originalRM)
		os.Setenv("ARM_ACTIVE_DIRECTORY_AUTHORITY_HOST", originalAD)
		os.Setenv("ARM_RESOURCE_MANAGER_AUDIENCE", originalAudience)
	}()

	// Set test environment variables
	os.Setenv("ARM_RESOURCE_MANAGER_ENDPOINT", "https://management.env.example.com/")
	os.Setenv("ARM_ACTIVE_DIRECTORY_AUTHORITY_HOST", "https://login.env.example.com/")
	os.Setenv("ARM_RESOURCE_MANAGER_AUDIENCE", "https://management.core.env.example.com/")

	m := &AuthModel{
		Environment: types.StringValue("custom"),
		Endpoint:    types.ListNull(types.ObjectType{}),
	}

	m.configureEndpointFromEnv()

	if m.Endpoint.IsNull() {
		t.Fatal("configureEndpointFromEnv() did not set Endpoint")
	}

	var endpoints []EndpointModel
	diags := m.Endpoint.ElementsAs(context.Background(), &endpoints, false)
	if diags.HasError() {
		t.Fatalf("Failed to extract endpoint: %v", diags.Errors())
	}

	if len(endpoints) != 1 {
		t.Fatalf("Expected 1 endpoint, got %d", len(endpoints))
	}

	endpoint := endpoints[0]
	if endpoint.ResourceManagerEndpoint.ValueString() != "https://management.env.example.com/" {
		t.Errorf("ResourceManagerEndpoint = %v, want https://management.env.example.com/",
			endpoint.ResourceManagerEndpoint.ValueString())
	}

	if endpoint.ActiveDirectoryAuthorityHost.ValueString() != "https://login.env.example.com/" {
		t.Errorf("ActiveDirectoryAuthorityHost = %v, want https://login.env.example.com/",
			endpoint.ActiveDirectoryAuthorityHost.ValueString())
	}

	if endpoint.ResourceManagerAudience.ValueString() != "https://management.core.env.example.com/" {
		t.Errorf("ResourceManagerAudience = %v, want https://management.core.env.example.com/",
			endpoint.ResourceManagerAudience.ValueString())
	}
}

// Test_configureEndpointFromEnv_PartialEnvironment tests partial environment variable configuration.
func Test_configureEndpointFromEnv_PartialEnvironment(t *testing.T) {
	// Save original environment variables
	originalRM := os.Getenv("ARM_RESOURCE_MANAGER_ENDPOINT")
	originalAD := os.Getenv("ARM_ACTIVE_DIRECTORY_AUTHORITY_HOST")
	originalAudience := os.Getenv("ARM_RESOURCE_MANAGER_AUDIENCE")

	// Restore environment after test
	defer func() {
		os.Setenv("ARM_RESOURCE_MANAGER_ENDPOINT", originalRM)
		os.Setenv("ARM_ACTIVE_DIRECTORY_AUTHORITY_HOST", originalAD)
		os.Setenv("ARM_RESOURCE_MANAGER_AUDIENCE", originalAudience)
	}()

	// Set only one environment variable
	os.Unsetenv("ARM_ACTIVE_DIRECTORY_AUTHORITY_HOST")
	os.Unsetenv("ARM_RESOURCE_MANAGER_AUDIENCE")
	os.Setenv("ARM_RESOURCE_MANAGER_ENDPOINT", "https://management.env.example.com/")

	m := &AuthModel{
		Environment: types.StringValue("custom"),
		Endpoint:    types.ListNull(types.ObjectType{}),
	}

	m.configureEndpointFromEnv()

	// Should still create endpoint object even with partial config
	if m.Endpoint.IsNull() {
		t.Fatal("configureEndpointFromEnv() did not set Endpoint with partial environment")
	}
}

// Test_configureEndpointFromEnv_NoEnvironment tests behavior when no environment variables are set.
func Test_configureEndpointFromEnv_NoEnvironment(t *testing.T) {
	// Save original environment variables
	originalRM := os.Getenv("ARM_RESOURCE_MANAGER_ENDPOINT")
	originalAD := os.Getenv("ARM_ACTIVE_DIRECTORY_AUTHORITY_HOST")
	originalAudience := os.Getenv("ARM_RESOURCE_MANAGER_AUDIENCE")

	// Restore environment after test
	defer func() {
		os.Setenv("ARM_RESOURCE_MANAGER_ENDPOINT", originalRM)
		os.Setenv("ARM_ACTIVE_DIRECTORY_AUTHORITY_HOST", originalAD)
		os.Setenv("ARM_RESOURCE_MANAGER_AUDIENCE", originalAudience)
	}()

	// Unset all environment variables
	os.Unsetenv("ARM_RESOURCE_MANAGER_ENDPOINT")
	os.Unsetenv("ARM_ACTIVE_DIRECTORY_AUTHORITY_HOST")
	os.Unsetenv("ARM_RESOURCE_MANAGER_AUDIENCE")

	m := &AuthModel{
		Environment: types.StringValue("custom"),
		Endpoint:    types.ListNull(types.ObjectType{}),
	}

	m.configureEndpointFromEnv()

	// Should not create endpoint object when no environment variables are set
	if !m.Endpoint.IsNull() {
		t.Error("configureEndpointFromEnv() should not set Endpoint when no environment variables are set")
	}
}

// Helper function to check if a string contains a substring.
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 || containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
