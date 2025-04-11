package server

import (
	"github.com/sammcj/go-a2a"
)

// Config holds the configuration for the A2A server.
type Config struct {
	ListenAddress string       // Address to listen on (e.g., ":8080")
	A2APathPrefix string       // Path prefix for A2A endpoints (e.g., "/a2a")
	AgentCard     *a2a.AgentCard // The agent card describing this agent
	TaskManager   TaskManager  // The task manager implementation
	TaskHandler   TaskHandler  // The application-specific task handler logic
	// TODO: Add fields for TLS config, middleware, SSE config, etc.
}

// Option is a function that modifies the server configuration.
type Option func(*Config)

// DefaultConfig returns a Config with default values.
func DefaultConfig() Config {
	return Config{
		ListenAddress: ":8080", // Default listen address
		A2APathPrefix: "/a2a",  // Default A2A path prefix
		// AgentCard is required, must be provided via WithAgentCard
		// TaskManager defaults to InMemoryTaskManager if TaskHandler is provided
		// TaskHandler is required, must be provided via WithTaskHandler
	}
}

// WithListenAddress sets the listen address for the server.
func WithListenAddress(addr string) Option {
	return func(c *Config) {
		c.ListenAddress = addr
	}
}

// WithA2APathPrefix sets the path prefix for A2A endpoints.
func WithA2APathPrefix(prefix string) Option {
	return func(c *Config) {
		// TODO: Add validation for path prefix format (e.g., must start with /)
		c.A2APathPrefix = prefix
	}
}

// WithAgentCard sets the Agent Card for the server.
func WithAgentCard(card *a2a.AgentCard) Option {
	return func(c *Config) {
		c.AgentCard = card
	}
}

// WithTaskManager sets a custom TaskManager implementation.
func WithTaskManager(tm TaskManager) Option {
	return func(c *Config) {
		c.TaskManager = tm
	}
}

// WithTaskHandler sets the application-specific task handler function.
// This is required unless a custom TaskManager is provided.
func WithTaskHandler(handler TaskHandler) Option {
	return func(c *Config) {
		c.TaskHandler = handler
	}
}

// TODO: Add options for middleware, TLS, SSE configuration, etc.
// Example:
// func WithMiddleware(mw ...func(http.Handler) http.Handler) Option { ... }
