package clients

import (
	"sync"

	"github.com/Azure/alzlib"
)

// Client is the data struct passed to services via Configure.
type Client struct {
	*alzlib.AlzLib
	mu                                   *sync.Mutex
	suppressWarningPolicyRoleAssignments bool
	ncmPlaceholder                       string
	ncmEnforcedReplacement               string
	ncmNotEnforcedReplacement            string
}

func (s *Client) SuppressWarningPolicyRoleAssignments() bool {
	return s.suppressWarningPolicyRoleAssignments
}

// NonComplianceMessagePlaceholder returns the configured placeholder string.
func (s *Client) NonComplianceMessagePlaceholder() string {
	return s.ncmPlaceholder
}

// NonComplianceMessageEnforcedReplacement returns the replacement for enforced assignments.
func (s *Client) NonComplianceMessageEnforcedReplacement() string {
	return s.ncmEnforcedReplacement
}

// NonComplianceMessageNotEnforcedReplacement returns the replacement for not-enforced assignments.
func (s *Client) NonComplianceMessageNotEnforcedReplacement() string {
	return s.ncmNotEnforcedReplacement
}

// Option is a functional option for configuring the Client.
type Option func(*Client)

// NewClient creates a new Client with the given options.
func NewClient(opts ...Option) *Client {
	client := &Client{
		AlzLib:                               nil,
		mu:                                   &sync.Mutex{},
		suppressWarningPolicyRoleAssignments: false,
		ncmPlaceholder:                       "",
		ncmEnforcedReplacement:               "",
		ncmNotEnforcedReplacement:            "",
	}

	for _, opt := range opts {
		opt(client)
	}

	return client
}

// WithSuppressWarningPolicyRoleAssignments sets the suppressWarningPolicyRoleAssignments field.
func WithSuppressWarningPolicyRoleAssignments(suppress bool) Option {
	return func(c *Client) {
		c.suppressWarningPolicyRoleAssignments = suppress
	}
}

// WithAlzLib sets the AlzLib field.
func WithAlzLib(alzLib *alzlib.AlzLib) Option {
	return func(c *Client) {
		c.AlzLib = alzLib
	}
}

// WithNonComplianceMessageSubstitutionSettings configures the non-compliance message substitution settings.
func WithNonComplianceMessageSubstitutionSettings(placeholder, enforcedReplacement, notEnforcedReplacement string) Option {
	return func(c *Client) {
		c.ncmPlaceholder = placeholder
		c.ncmEnforcedReplacement = enforcedReplacement
		c.ncmNotEnforcedReplacement = notEnforcedReplacement
	}
}
