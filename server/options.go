package server

import (
	"net/http"

	"github.com/sammcj/go-a2a/a2a"
	"github.com/sammcj/go-a2a/cmd/common"
	"github.com/sammcj/go-a2a/llm"
	"github.com/sammcj/go-a2a/llm/gollm"
)

// AuthValidator is a function that validates authentication for requests.
type AuthValidator func(w http.ResponseWriter, r *http.Request, next http.Handler, card *a2a.AgentCard)

// Config holds the configuration for the A2A server.
type Config struct {
	ListenAddress string         // Address to listen on (e.g., ":8080")
	A2APathPrefix string         // Path prefix for A2A endpoints (e.g., "/a2a")
	AgentCard     *a2a.AgentCard // The agent card describing this agent
	AgentCardPath string         // Path to serve the agent card (e.g., "/.well-known/agent.json")
	TaskManager   TaskManager    // The task manager implementation
	TaskHandler   TaskHandler    // The application-specific task handler logic
	AgentEngine   AgentEngine    // The agent engine implementation
	AuthValidator AuthValidator  // Optional authentication validator function
	// TODO: Add fields for optional TLS config, middleware, SSE config, etc.
	gollmOptions  *gollm.Options
}

// Option is a function that modifies the server configuration.
type Option func(*Config)

// DefaultConfig returns a Config with default values.
func DefaultConfig() Config {
	return Config{
		ListenAddress: ":8080",                   // Default listen address
		A2APathPrefix: "/a2a",                    // Default A2A path prefix
		AgentCardPath: DefaultAgentCardPath,      // Default agent card path
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

// WithAuthValidator sets the authentication validator function.
func WithAuthValidator(validator AuthValidator) Option {
	return func(c *Config) {
		c.AuthValidator = validator
	}
}

// WithAgentEngine sets a custom AgentEngine implementation.
func WithAgentEngine(engine AgentEngine) Option {
	return func(c *Config) {
		c.AgentEngine = engine
	}
}

// WithBasicLLMAgent creates a BasicLLMAgent with the provided LLM interface and system prompt.
func WithBasicLLMAgent(llmInterface llm.LLMInterface, systemPrompt string) Option {
	return func(c *Config) {
		c.AgentEngine = NewBasicLLMAgent(llmInterface, systemPrompt)
	}
}

// WithBasicGollmAgent creates a BasicLLMAgent with a gollm adapter.
func WithBasicGollmAgent(provider, model, apiKey, systemPrompt string) Option {
	return func(c *Config) {
		// Create gollm adapter
		adapter, err := gollm.NewAdapter(
			gollm.WithProvider(provider),
			gollm.WithModel(model),
			gollm.WithAPIKey(apiKey),
		)
		if err != nil {
			// Log error and return without setting the agent engine
			// TODO: Consider a better way to handle errors in options
			return
		}

		// Create agent
		c.AgentEngine = NewBasicLLMAgent(adapter, systemPrompt)
	}
}

// WithGollmOptions sets a custom gollm.Options struct
func WithGollmOptions(options *gollm.Options) Option {
	return func(c *Config) {
		c.gollmOptions = options
	}
}

// LLMConfig is a config for use with gollm
type LLMConfig struct {
	Provider     string                 `json:"provider" yaml:"provider"`
	Model        string                 `json:"model,omitempty" yaml:"model,omitempty"`
	APIKey       string                 `json:"apiKey,omitempty" yaml:"apiKey,omitempty"`
	SystemPrompt string                 `json:"systemPrompt,omitempty" yaml:"systemPrompt,omitempty"`
	BaseUrl      string                 `json:"baseUrl,omitempty" yaml:"baseUrl,omitempty"`
	Options      map[string]interface{} `json:"options,omitempty" yaml:"options,omitempty"`
}

// NewGollmOptions creates new gollm options from an LLMConfig
func NewGollmOptions(llmConfig LLMConfig) (*gollm.Options, error) {
	options := []func(*gollm.Options) error{}

	if llmConfig.Provider != "" {
		options = append(options, gollm.WithProvider(llmConfig.Provider))
	}
	if llmConfig.Model != "" {
		options = append(options, gollm.WithModel(llmConfig.Model))
	}
	if llmConfig.APIKey != "" {
		options = append(options, gollm.WithAPIKey(llmConfig.APIKey))
	}
	if llmConfig.SystemPrompt != "" {
		options = append(options, gollm.WithSystemPrompt(llmConfig.SystemPrompt))
	}
	if llmConfig.BaseUrl != "" {
		options = append(options, gollm.WithBaseURL(llmConfig.BaseUrl))
	}
	if len(llmConfig.Options) > 0 {
		options = append(options, gollm.WithOptions(llmConfig.Options))
	}

	return gollm.NewOptions(options...)
}

// DefaultLLMConfig returns a default LLM config.
func DefaultLLMConfig() LLMConfig {
	return LLMConfig{
		Provider:     "openai",
		Model:        "",
		APIKey:       "",
		SystemPrompt: "",
		BaseUrl:      "",
		Options: map[string]interface{}{
			"temperature": 0.7,
		},
	}
}

// NewGollmOptionsFromConfig creates new gollm options from a common.LLMConfig
func NewGollmOptionsFromConfig(llmConfig common.LLMConfig) (*gollm.Options, error) {
	return NewGollmOptions(LLMConfig{
		Provider:     llmConfig.Provider,
		Model:        llmConfig.Model,
		APIKey:       llmConfig.ApiKey,
		SystemPrompt: llmConfig.SystemPrompt,
		BaseUrl:      llmConfig.BaseUrl,
		Options:      llmConfig.Options,
	})
}



// WithToolAugmentedAgent creates a ToolAugmentedAgent with the provided LLM interface and tools.
func WithToolAugmentedAgent(llmInterface llm.LLMInterface, tools []Tool) Option {
	return func(c *Config) {
		c.AgentEngine = NewToolAugmentedAgent(llmInterface, tools)
	}
}

// WithToolAugmentedGollmAgent creates a ToolAugmentedAgent with a gollm adapter.
func WithToolAugmentedGollmAgent(provider, model, apiKey string, tools []Tool) Option {
	return func(c *Config) {
		// Create gollm adapter
		adapter, err := gollm.NewAdapter(
			gollm.WithProvider(provider),
			gollm.WithModel(model),
			gollm.WithAPIKey(apiKey),
		)
		if err != nil {
			// Log error and return without setting the agent engine
			// TODO: Consider a better way to handle errors in options
			return
		}

		// Create agent
		c.AgentEngine = NewToolAugmentedAgent(adapter, tools)
	}
}

// WithMCPToolAugmentedAgent creates a MCPToolAugmentedAgent with the provided LLM interface and MCP client.
func WithMCPToolAugmentedAgent(llmInterface llm.LLMInterface, mcpClient MCPClient) Option {
	return func(c *Config) {
		// Create agent
		agent, err := NewMCPToolAugmentedAgent(llmInterface, mcpClient)
		if err != nil {
			// Log error and return without setting the agent engine
			// TODO: Consider a better way to handle errors in options
			return
		}

		c.AgentEngine = agent
	}
}

// WithMCPToolAugmentedGollmAgent creates a MCPToolAugmentedAgent with a gollm adapter and MCP client.
func WithMCPToolAugmentedGollmAgent(provider, model, apiKey string, mcpClient MCPClient) Option {
	return func(c *Config) {
		// Create gollm adapter
		adapter, err := gollm.NewAdapter(
			gollm.WithProvider(provider),
			gollm.WithModel(model),
			gollm.WithAPIKey(apiKey),
		)
		if err != nil {
			// Log error and return without setting the agent engine
			// TODO: Consider a better way to handle errors in options
			return
		}

		// Create agent
		agent, err := NewMCPToolAugmentedAgent(adapter, mcpClient)
		if err != nil {
			// Log error and return without setting the agent engine
			// TODO: Consider a better way to handle errors in options
			return
		}

		c.AgentEngine = agent
	}
}

// TODO: Add options for TLS, SSE configuration, etc.
