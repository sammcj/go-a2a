package client

import (
	"net/http"
	"time"

	"github.com/sammcj/go-a2a/a2a"
)

// Config holds the configuration for the A2A client.
type Config struct {
	BaseURL     string        // Base URL of the A2A server (e.g., "https://agent.example.com")
	HTTPClient  *http.Client  // HTTP client to use for requests
	Timeout     time.Duration // Timeout for requests
	AgentCard   *a2a.AgentCard // Cached agent card (if already fetched)
	AuthHeaders map[string]string // Authentication headers to include in requests
}

// Option is a function that modifies the client configuration.
type Option func(*Config)

// DefaultConfig returns a Config with default values.
func DefaultConfig() Config {
	return Config{
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		Timeout:     30 * time.Second,
		AuthHeaders: make(map[string]string),
	}
}

// WithBaseURL sets the base URL for the client.
func WithBaseURL(baseURL string) Option {
	return func(c *Config) {
		c.BaseURL = baseURL
	}
}

// WithHTTPClient sets the HTTP client for the client.
func WithHTTPClient(httpClient *http.Client) Option {
	return func(c *Config) {
		c.HTTPClient = httpClient
	}
}

// WithTimeout sets the timeout for requests.
func WithTimeout(timeout time.Duration) Option {
	return func(c *Config) {
		c.Timeout = timeout
		if c.HTTPClient != nil {
			c.HTTPClient.Timeout = timeout
		}
	}
}

// WithAgentCard sets a pre-fetched agent card.
func WithAgentCard(card *a2a.AgentCard) Option {
	return func(c *Config) {
		c.AgentCard = card
	}
}

// WithAuthHeader adds an authentication header to be included in all requests.
func WithAuthHeader(name, value string) Option {
	return func(c *Config) {
		c.AuthHeaders[name] = value
	}
}

// WithBearerToken sets the Authorization header with a Bearer token.
func WithBearerToken(token string) Option {
	return func(c *Config) {
		c.AuthHeaders["Authorization"] = "Bearer " + token
	}
}
