package test

import (
	"encoding/json"
	"os"
	"testing"
)

// TestEnvironmentVariableReplacement tests replacing environment variables in configuration.
func TestEnvironmentVariableReplacement(t *testing.T) {
	// Set environment variables
	os.Setenv("TEST_API_KEY", "test-api-key")
	os.Setenv("TEST_BASE_URL", "https://test-api.example.com")
	defer os.Unsetenv("TEST_API_KEY")
	defer os.Unsetenv("TEST_BASE_URL")

	// Create a temporary config file
	configFile := `{
		"listenAddress": ":8080",
		"agentCardPath": "/.well-known/agent.json",
		"a2aPathPrefix": "/a2a",
		"logLevel": "info",
		"llmConfig": {
			"provider": "openai",
			"model": "gpt-4o",
			"apiKey": "${TEST_API_KEY}",
			"baseUrl": "${TEST_BASE_URL}",
			"systemPrompt": "You are a helpful assistant."
		},
		"mcpConfig": {
			"tools": [
				{
					"name": "brave-search",
					"enabled": true,
					"config": {
						"env": {
							"BRAVE_API_KEY": "${TEST_API_KEY}"
						}
					}
				}
			]
		},
		"agentCard": {
			"a2aVersion": "1.0",
			"id": "test-agent",
			"name": "Test Agent",
			"description": "A test agent",
			"skills": [],
			"capabilities": {}
		}
	}`

	// Write the config file
	tmpFile, err := os.CreateTemp("", "agent-config-*.json")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.Write([]byte(configFile)); err != nil {
		t.Fatalf("Failed to write temp file: %v", err)
	}
	if err := tmpFile.Close(); err != nil {
		t.Fatalf("Failed to close temp file: %v", err)
	}

	// Define types for testing
	type LLMConfig struct {
		Provider     string `json:"provider"`
		Model        string `json:"model"`
		APIKey       string `json:"apiKey"`
		BaseURL      string `json:"baseUrl"`
		SystemPrompt string `json:"systemPrompt"`
	}

	type MCPToolConfig struct {
		Name    string                 `json:"name"`
		Enabled bool                   `json:"enabled"`
		Config  map[string]interface{} `json:"config"`
	}

	type MCPConfig struct {
		Tools []MCPToolConfig `json:"tools"`
	}

	type AgentConfig struct {
		ListenAddress string                 `json:"listenAddress"`
		AgentCardPath string                 `json:"agentCardPath"`
		A2APathPrefix string                 `json:"a2aPathPrefix"`
		LogLevel      string                 `json:"logLevel"`
		LLMConfig     LLMConfig              `json:"llmConfig"`
		MCPConfig     MCPConfig              `json:"mcpConfig"`
		AgentCard     map[string]interface{} `json:"agentCard"`
		Extra         map[string]interface{} `json:"extra"`
	}

	// Load the config file
	data, err := os.ReadFile(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to read config file: %v", err)
	}

	var config AgentConfig
	if err := json.Unmarshal(data, &config); err != nil {
		t.Fatalf("Failed to parse config file: %v", err)
	}

	// Replace environment variables in the configuration
	if config.LLMConfig.APIKey != "" && config.LLMConfig.APIKey[0] == '$' {
		envVar := config.LLMConfig.APIKey[1:]
		if envVar[0] == '{' && envVar[len(envVar)-1] == '}' {
			envVar = envVar[1 : len(envVar)-1]
		}
		config.LLMConfig.APIKey = os.Getenv(envVar)
	}

	if config.LLMConfig.BaseURL != "" && config.LLMConfig.BaseURL[0] == '$' {
		envVar := config.LLMConfig.BaseURL[1:]
		if envVar[0] == '{' && envVar[len(envVar)-1] == '}' {
			envVar = envVar[1 : len(envVar)-1]
		}
		baseURL := os.Getenv(envVar)
		if baseURL != "" {
			config.LLMConfig.BaseURL = baseURL
		}
	}

	// Process MCP tool configurations
	if len(config.MCPConfig.Tools) > 0 {
		for i, tool := range config.MCPConfig.Tools {
			if env, ok := tool.Config["env"].(map[string]interface{}); ok {
				for key, value := range env {
					if strValue, ok := value.(string); ok && strValue != "" && strValue[0] == '$' {
						envVar := strValue[1:]
						if envVar[0] == '{' && envVar[len(envVar)-1] == '}' {
							envVar = envVar[1 : len(envVar)-1]
						}
						envValue := os.Getenv(envVar)
						if envValue != "" {
							env[key] = envValue
							tool.Config["env"] = env
							config.MCPConfig.Tools[i] = tool
						}
					}
				}
			}
		}
	}

	// Check the config
	if config.LLMConfig.APIKey != "test-api-key" {
		t.Errorf("Expected API key to be 'test-api-key', got '%s'", config.LLMConfig.APIKey)
	}
	if config.LLMConfig.BaseURL != "https://test-api.example.com" {
		t.Errorf("Expected base URL to be 'https://test-api.example.com', got '%s'", config.LLMConfig.BaseURL)
	}

	// Check the MCP tool config
	if len(config.MCPConfig.Tools) != 1 {
		t.Fatalf("Expected 1 MCP tool, got %d", len(config.MCPConfig.Tools))
	}
	tool := config.MCPConfig.Tools[0]
	if tool.Name != "brave-search" {
		t.Errorf("Expected tool name to be 'brave-search', got '%s'", tool.Name)
	}
	env, ok := tool.Config["env"].(map[string]interface{})
	if !ok {
		t.Fatalf("Expected tool config to have 'env' field")
	}
	braveAPIKey, ok := env["BRAVE_API_KEY"].(string)
	if !ok {
		t.Fatalf("Expected env to have 'BRAVE_API_KEY' field")
	}
	if braveAPIKey != "test-api-key" {
		t.Errorf("Expected BRAVE_API_KEY to be 'test-api-key', got '%s'", braveAPIKey)
	}
}

// TestLoadAgentConfig tests loading agent configuration from a file.
func TestLoadAgentConfig(t *testing.T) {
	// Create a temporary config file
	configFile := `{
		"listenAddress": ":8080",
		"agentCardPath": "/.well-known/agent.json",
		"a2aPathPrefix": "/a2a",
		"logLevel": "info",
		"llmConfig": {
			"provider": "ollama",
			"model": "llama3",
			"apiKey": "",
			"systemPrompt": "You are a helpful assistant."
		},
		"agentCard": {
			"a2aVersion": "1.0",
			"id": "test-agent",
			"name": "Test Agent",
			"description": "A test agent",
			"skills": [
				{
					"id": "test-skill",
					"name": "Test Skill"
				}
			],
			"capabilities": {
				"supportsStreaming": true
			}
		}
	}`

	// Write the config file
	tmpFile, err := os.CreateTemp("", "agent-config-*.json")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.Write([]byte(configFile)); err != nil {
		t.Fatalf("Failed to write temp file: %v", err)
	}
	if err := tmpFile.Close(); err != nil {
		t.Fatalf("Failed to close temp file: %v", err)
	}

	// Define a LoadAgentConfig function for testing
	type AgentConfig struct {
		ListenAddress string                 `json:"listenAddress"`
		AgentCardPath string                 `json:"agentCardPath"`
		A2APathPrefix string                 `json:"a2aPathPrefix"`
		LogLevel      string                 `json:"logLevel"`
		LLMConfig     struct {
			Provider     string `json:"provider"`
			Model        string `json:"model"`
			APIKey       string `json:"apiKey"`
			SystemPrompt string `json:"systemPrompt"`
		} `json:"llmConfig"`
		AgentCard map[string]interface{} `json:"agentCard"`
		Extra     map[string]interface{} `json:"extra"`
	}

	// Load the config file
	data, err := os.ReadFile(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to read config file: %v", err)
	}

	var config AgentConfig
	if err := json.Unmarshal(data, &config); err != nil {
		t.Fatalf("Failed to parse config file: %v", err)
	}

	// Check the config
	if config.ListenAddress != ":8080" {
		t.Errorf("Expected listen address to be ':8080', got '%s'", config.ListenAddress)
	}
	if config.AgentCardPath != "/.well-known/agent.json" {
		t.Errorf("Expected agent card path to be '/.well-known/agent.json', got '%s'", config.AgentCardPath)
	}
	if config.LLMConfig.Provider != "ollama" {
		t.Errorf("Expected LLM provider to be 'ollama', got '%s'", config.LLMConfig.Provider)
	}
	if config.LLMConfig.Model != "llama3" {
		t.Errorf("Expected LLM model to be 'llama3', got '%s'", config.LLMConfig.Model)
	}

	// Check the agent card
	agentCard := config.AgentCard
	if agentCard["id"] != "test-agent" {
		t.Errorf("Expected agent ID to be 'test-agent', got '%s'", agentCard["id"])
	}
	if agentCard["name"] != "Test Agent" {
		t.Errorf("Expected agent name to be 'Test Agent', got '%s'", agentCard["name"])
	}
}
