package provider

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStandardizeRoleAssignmentRoleDefinititionId(t *testing.T) {
	// Test a valid input.
	input := "/subscriptions/dabf9763-fbbb-435c-921b-61f5ed59b3d1/providers/Microsoft.Authorization/roleDefinitions/92aaf0da-9dab-42b6-94a3-d43ce8d16293"
	expectedOutput := "/providers/Microsoft.Authorization/roleDefinitions/92aaf0da-9dab-42b6-94a3-d43ce8d16293"
	output := standardizeRoleAssignmentRoleDefinititionId(input)
	assert.Equal(t, expectedOutput, output)

	// Test an invalid input.
	input = "/subscriptions/dabf9763-fbbb-435c-921b-61f5ed59b3d1/providers/Microsoft.Authorization/roleDefinitions"
	expectedOutput = "/subscriptions/dabf9763-fbbb-435c-921b-61f5ed59b3d1/providers/Microsoft.Authorization/roleDefinitions"
	output = standardizeRoleAssignmentRoleDefinititionId(input)
	assert.Equal(t, expectedOutput, output)
}
