package provider

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStandardizeRoleAssignmentRoleDefinititionId(t *testing.T) {
	// Test a valid input.
	input := "/subscriptions/00000000-0000-0000-0000-000000000000/providers/Microsoft.Authorization/roleDefinitions/92aaf0da-9dab-42b6-94a3-d43ce8d16293"
	expectedOutput := "/providers/Microsoft.Authorization/roleDefinitions/92aaf0da-9dab-42b6-94a3-d43ce8d16293"
	output := standardizeRoleAssignmentRoleDefinititionId(input)
	assert.Equal(t, expectedOutput, output)

	// Test an invalid input.
	input = "/subscriptions/00000000-0000-0000-0000-000000000000/providers/Microsoft.Authorization/roleDefinitions"
	expectedOutput = "/subscriptions/00000000-0000-0000-0000-000000000000/providers/Microsoft.Authorization/roleDefinitions"
	output = standardizeRoleAssignmentRoleDefinititionId(input)
	assert.Equal(t, expectedOutput, output)
}
