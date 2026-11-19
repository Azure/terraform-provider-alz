package clients

import (
	"context"
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
	cacheFileName                        string
	cacheFileSaveEnabled                 bool
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

// WithCacheFile sets the built-in cache file and whether SaveBuiltInCache writes it.
func WithCacheFile(name string, saveEnabled bool) Option {
	return func(c *Client) {
		c.cacheFileName = name
		c.cacheFileSaveEnabled = saveEnabled
	}
}

// SaveBuiltInCache writes the built-in policy and policy set definitions loaded into the AlzLib to
// the cache file. It does nothing unless a cache file is set and saving is enabled.
// Built-in definitions are loaded when an architecture is built, so call it after that.
// Calls are serialised, because several alz_architecture data sources can be read concurrently.
func (s *Client) SaveBuiltInCache(ctx context.Context) error {
	if s.AlzLib == nil || s.cacheFileName == "" || !s.cacheFileSaveEnabled {
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	return SaveCacheFile(ctx, s.AlzLib, s.cacheFileName)
}
