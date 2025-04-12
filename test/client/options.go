package client

import (
	"net/http"
	"time"

	"github.com/sammcj/go-a2a/a2a"
)

// Config holds the configuration for the A2A client.
type Config struct {
	BaseURL     string
	HTTPClient  *http.Client
	Timeout     time.Duration
	AgentCard   *a2a.AgentCard
	AuthHeaders map[string]string
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

// WithAuthHeaders sets the authentication headers.
func WithAuthHeaders(headers map[string]string) Option {
	return func(c *Config) {
		c.AuthHeaders = headers
	}
}
