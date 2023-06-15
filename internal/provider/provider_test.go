// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/stretchr/testify/assert"
)

// testAccProtoV6ProviderFactories are used to instantiate a provider during
// acceptance testing. The factory function will be invoked for every Terraform
// CLI command executed to create a provider server to which the CLI can
// reattach.
var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"alz": providerserver.NewProtocol6WithError(New("test")()),
}

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
	os.Setenv("VAR1", "value1")
	result = getFirstSetEnvVar("VAR1", "VAR2", "VAR3")
	assert.Equal(t, "value1", result)
	_ = os.Unsetenv("VAR1")

	// Test when the second environment variable is set
	os.Setenv("VAR2", "value2")
	result = getFirstSetEnvVar("VAR1", "VAR2", "VAR3")
	assert.Equal(t, "value2", result)
	os.Unsetenv("VAR2")

	// Test when the third environment variable is set
	os.Setenv("VAR3", "value3")
	result = getFirstSetEnvVar("VAR1", "VAR2", "VAR3")
	assert.Equal(t, "value3", result)
	os.Unsetenv("VAR3")

	// Test when multiple environment variables are set
	os.Setenv("VAR1", "value1")
	os.Setenv("VAR2", "value2")
	os.Setenv("VAR3", "value3")
	result = getFirstSetEnvVar("VAR1", "VAR2", "VAR3")
	assert.Equal(t, "value1", result)
	os.Unsetenv("VAR1")
	os.Unsetenv("VAR2")
	os.Unsetenv("VAR3")
}
