package clients

import (
	"sync"

	"github.com/Azure/alzlib"
)

// Client is the data struct passed to services via Configure
type Client struct {
	*alzlib.AlzLib
	mu                                   *sync.Mutex
	suppressWarningPolicyRoleAssignments bool
}

func (s *Client) SuppressWarningPolicyRoleAssignments() bool {
	return s.suppressWarningPolicyRoleAssignments
}

// Option is a functional option for configuring the Client.
type Option func(*Client)

// NewClient creates a new Client with the given options.
func NewClient(opts ...Option) *Client {
	client := &Client{
		AlzLib:                               nil,
		mu:                                   &sync.Mutex{},
		suppressWarningPolicyRoleAssignments: false,
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
