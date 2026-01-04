// Package bravozero provides the official Go SDK for Bravo Zero Breaking the Limits APIs.
package bravozero

import (
	"context"
	"fmt"
	"os"
)

// Environment constants
const (
	EnvProduction  = "production"
	EnvStaging     = "staging"
	EnvDevelopment = "development"
)

// ClientConfig holds configuration for the Bravo Zero client.
type ClientConfig struct {
	// APIKey for authentication
	APIKey string
	// AgentID is the PERSONA agent identifier
	AgentID string
	// PrivateKeyPath is the path to Ed25519 private key for signing
	PrivateKeyPath string
	// BaseURL overrides the default API base URL
	BaseURL string
	// Environment (production, staging, development)
	Environment string
	// TimeoutSeconds is the request timeout
	TimeoutSeconds int
}

// ClientOption is a function that configures a Client
type ClientOption func(*ClientConfig)

// WithAPIKey sets the API key
func WithAPIKey(key string) ClientOption {
	return func(c *ClientConfig) {
		c.APIKey = key
	}
}

// WithAgentID sets the agent ID
func WithAgentID(id string) ClientOption {
	return func(c *ClientConfig) {
		c.AgentID = id
	}
}

// WithPrivateKeyPath sets the path to the private key
func WithPrivateKeyPath(path string) ClientOption {
	return func(c *ClientConfig) {
		c.PrivateKeyPath = path
	}
}

// WithBaseURL sets the base URL
func WithBaseURL(url string) ClientOption {
	return func(c *ClientConfig) {
		c.BaseURL = url
	}
}

// WithEnvironment sets the environment
func WithEnvironment(env string) ClientOption {
	return func(c *ClientConfig) {
		c.Environment = env
	}
}

// WithTimeout sets the timeout in seconds
func WithTimeout(seconds int) ClientOption {
	return func(c *ClientConfig) {
		c.TimeoutSeconds = seconds
	}
}

// Client is the main Bravo Zero client providing access to all services.
type Client struct {
	config        ClientConfig
	authenticator *PersonaAuthenticator
	constitution  *ConstitutionClient
	memory        *MemoryClient
	bridge        *BridgeClient
}

// NewClient creates a new Bravo Zero client with the given options.
func NewClient(opts ...ClientOption) (*Client, error) {
	config := ClientConfig{
		Environment:    EnvProduction,
		TimeoutSeconds: 30,
	}

	// Apply options
	for _, opt := range opts {
		opt(&config)
	}

	// Check environment variables
	if config.APIKey == "" {
		config.APIKey = os.Getenv("BRAVOZERO_API_KEY")
	}
	if config.AgentID == "" {
		config.AgentID = os.Getenv("BRAVOZERO_AGENT_ID")
	}
	if config.PrivateKeyPath == "" {
		config.PrivateKeyPath = os.Getenv("BRAVOZERO_PRIVATE_KEY_PATH")
	}

	// Validate required fields
	if config.APIKey == "" {
		return nil, fmt.Errorf("API key required: set BRAVOZERO_API_KEY or use WithAPIKey")
	}
	if config.AgentID == "" {
		return nil, fmt.Errorf("Agent ID required: set BRAVOZERO_AGENT_ID or use WithAgentID")
	}

	// Set base URL
	if config.BaseURL == "" {
		config.BaseURL = getBaseURL(config.Environment)
	}

	// Initialize authenticator
	var auth *PersonaAuthenticator
	if config.PrivateKeyPath != "" {
		var err error
		auth, err = NewPersonaAuthenticator(config.AgentID, config.PrivateKeyPath)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize authenticator: %w", err)
		}
	}

	return &Client{
		config:        config,
		authenticator: auth,
	}, nil
}

func getBaseURL(env string) string {
	switch env {
	case EnvStaging:
		return "https://api.staging.bravozero.ai"
	case EnvDevelopment:
		return "http://localhost:8080"
	default:
		return "https://api.bravozero.ai"
	}
}

// Constitution returns the Constitution Agent client.
func (c *Client) Constitution() *ConstitutionClient {
	if c.constitution == nil {
		c.constitution = NewConstitutionClient(
			c.config.BaseURL,
			c.config.APIKey,
			c.config.AgentID,
			c.authenticator,
			c.config.TimeoutSeconds,
		)
	}
	return c.constitution
}

// Memory returns the Memory Service client.
func (c *Client) Memory() *MemoryClient {
	if c.memory == nil {
		c.memory = NewMemoryClient(
			c.config.BaseURL,
			c.config.APIKey,
			c.config.AgentID,
			c.authenticator,
			c.config.TimeoutSeconds,
		)
	}
	return c.memory
}

// Bridge returns the Forge Bridge client.
func (c *Client) Bridge() *BridgeClient {
	if c.bridge == nil {
		c.bridge = NewBridgeClient(
			c.config.BaseURL,
			c.config.APIKey,
			c.config.AgentID,
			c.authenticator,
			c.config.TimeoutSeconds,
		)
	}
	return c.bridge
}

// Close closes any open connections.
func (c *Client) Close() error {
	// Close any gRPC connections if applicable
	return nil
}

// Context helper for operations
func (c *Client) contextWithTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, c.config.timeout())
}

func (c *ClientConfig) timeout() interface{} {
	return c.TimeoutSeconds
}
