package common

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/sammcj/go-a2a/a2a"
	"github.com/sammcj/go-a2a/pkg/config"
	"gopkg.in/yaml.v3"
)

// ServerConfig represents the configuration for a server.
type ServerConfig struct {
	ListenAddress string          `json:"listenAddress" yaml:"listenAddress"`
	LogLevel      string          `json:"logLevel" yaml:"logLevel"`
	AgentCardPath string          `json:"agentCardPath" yaml:"agentCardPath"`
	A2APathPrefix string          `json:"a2aPathPrefix" yaml:"a2aPathPrefix"`
	PluginPath    string          `json:"pluginPath" yaml:"pluginPath"`
	LLMConfig     *LLMConfig      `json:"llmConfig,omitempty" yaml:"llmConfig,omitempty"`
	AgentCard     AgentCardConfig `json:"agentCard" yaml:"agentCard"`
}

// ClientConfig represents the configuration for a client.
type ClientConfig struct {
	DefaultAgentURL string                 `json:"defaultAgentUrl" yaml:"defaultAgentUrl"`
	OutputFormat    string                 `json:"outputFormat" yaml:"outputFormat"`
	Authentication  map[string]interface{} `json:"authentication,omitempty" yaml:"authentication,omitempty"`
}

// LLMConfig represents the configuration for a language model.
type LLMConfig = config.LLMConfig

// AgentCardConfig represents the configuration for an agent card.
type AgentCardConfig struct {
	A2AVersion       string              `json:"a2aVersion" yaml:"a2aVersion"`
	ID               string              `json:"id" yaml:"id"`
	Name             string              `json:"name" yaml:"name"`
	Description      string              `json:"description" yaml:"description"`
	IconURI          string              `json:"iconUri" yaml:"iconUri"`
	Provider         *ProviderConfig     `json:"provider,omitempty" yaml:"provider,omitempty"`
	Skills           []SkillConfig       `json:"skills" yaml:"skills"`
	Capabilities     *CapabilitiesConfig `json:"capabilities,omitempty" yaml:"capabilities,omitempty"`
	Authentication   []AuthConfig        `json:"authentication,omitempty" yaml:"authentication,omitempty"`
	ContactEmail     string              `json:"contactEmail" yaml:"contactEmail"`
	LegalInfoURI     string              `json:"legalInfoUri" yaml:"legalInfoUri"`
	HomepageURI      string              `json:"homepageUri" yaml:"homepageUri"`
	DocumentationURI string              `json:"documentationUri" yaml:"documentationUri"`
}

// ProviderConfig represents the configuration for a provider.
type ProviderConfig struct {
	Name string `json:"name" yaml:"name"`
	URI  string `json:"uri" yaml:"uri"`
}

// SkillConfig represents the configuration for a skill.
type SkillConfig struct {
	ID          string `json:"id" yaml:"id"`
	Name        string `json:"name" yaml:"name"`
	Description string `json:"description" yaml:"description"`
}

// CapabilitiesConfig represents the capabilities configuration.
type CapabilitiesConfig struct {
	SupportsStreaming        bool `json:"supportsStreaming" yaml:"supportsStreaming"`
	SupportsSessions         bool `json:"supportsSessions" yaml:"supportsSessions"`
	SupportsPushNotification bool `json:"supportsPushNotification" yaml:"supportsPushNotification"`
}

// AuthConfig represents the configuration for authentication.
type AuthConfig struct {
	Type          string                 `json:"type" yaml:"type"`
	Scheme        string                 `json:"scheme" yaml:"scheme"`
	Configuration map[string]interface{} `json:"configuration" yaml:"configuration"`
}

// LoadConfig loads a configuration file of type T.
func LoadConfig[T any](path string) (*T, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config T
	ext := filepath.Ext(path)
	switch ext {
	case ".json":
		if err := json.Unmarshal(data, &config); err != nil {
			return nil, fmt.Errorf("failed to parse JSON config: %w", err)
		}
	case ".yaml", ".yml":
		if err := yaml.Unmarshal(data, &config); err != nil {
			return nil, fmt.Errorf("failed to parse YAML config: %w", err)
		}
	default:
		return nil, fmt.Errorf("unsupported config file format: %s", ext)
	}

	return &config, nil
}

// SaveConfig saves a configuration of type T to a file.
func SaveConfig[T any](config T, path string) error {
	var data []byte
	var err error

	ext := filepath.Ext(path)
	switch ext {
	case ".json":
		data, err = json.MarshalIndent(config, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON config: %w", err)
		}
	case ".yaml", ".yml":
		data, err = yaml.Marshal(config)
		if err != nil {
			return fmt.Errorf("failed to marshal YAML config: %w", err)
		}
	default:
		return fmt.Errorf("unsupported config file format: %s", ext)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// DefaultClientConfig returns a default client configuration.
func DefaultClientConfig() ClientConfig {
	return ClientConfig{
		DefaultAgentURL: "http://localhost:8080",
		OutputFormat:    "pretty",
	}
}

// DefaultLLMConfig returns a default language model configuration.
func DefaultLLMConfig() LLMConfig {
	return LLMConfig{
		Provider:     "openai",
		Model:        "",
		APIKey:       "",
		SystemPrompt: "",
		BaseUrl:      "",
		Options:      map[string]interface{}{"temperature": 0.7},
	}
}

// ConvertToAgentCard converts an AgentCardConfig to an AgentCard.
func ConvertToAgentCard(cfg *AgentCardConfig) *a2a.AgentCard {
	card := &a2a.AgentCard{
		A2AVersion: cfg.A2AVersion,
		ID:         cfg.ID,
		Name:       cfg.Name,
	}

	// Convert optional fields
	if cfg.Description != "" {
		card.Description = &cfg.Description
	}
	if cfg.IconURI != "" {
		card.IconURI = &cfg.IconURI
	}
	if cfg.Provider != nil {
		card.Provider = &a2a.AgentProvider{
			Name: cfg.Provider.Name,
		}
		if cfg.Provider.URI != "" {
			card.Provider.URI = &cfg.Provider.URI
		}
	}

	// Convert skills
	card.Skills = make([]a2a.AgentSkill, len(cfg.Skills))
	for i, skill := range cfg.Skills {
		desc := skill.Description
		card.Skills[i] = a2a.AgentSkill{
			ID:          skill.ID,
			Name:        skill.Name,
			Description: &desc,
		}
	}

	// Convert capabilities
	if cfg.Capabilities != nil {
		card.Capabilities = &a2a.AgentCapabilities{
			SupportsStreaming:        cfg.Capabilities.SupportsStreaming,
			SupportsSessions:         cfg.Capabilities.SupportsSessions,
			SupportsPushNotification: cfg.Capabilities.SupportsPushNotification,
		}
	}

	// Convert authentication
	if len(cfg.Authentication) > 0 {
		card.Authentication = make([]a2a.AgentAuthentication, len(cfg.Authentication))
		for i, auth := range cfg.Authentication {
			scheme := auth.Scheme
			card.Authentication[i] = a2a.AgentAuthentication{
				Type:          auth.Type,
				Scheme:        &scheme,
				Configuration: auth.Configuration,
			}
		}
	}

	// Convert optional URIs and contact
	if cfg.ContactEmail != "" {
		card.ContactEmail = &cfg.ContactEmail
	}
	if cfg.LegalInfoURI != "" {
		card.LegalInfoURI = &cfg.LegalInfoURI
	}
	if cfg.HomepageURI != "" {
		card.HomepageURI = &cfg.HomepageURI
	}
	if cfg.DocumentationURI != "" {
		card.DocumentationURI = &cfg.DocumentationURI
	}

	return card
}

// DefaultServerConfig returns a default server configuration.
func DefaultServerConfig() ServerConfig {
	return ServerConfig{
		ListenAddress: ":8080",
		AgentCardPath: "/.well-known/agent.json",
		A2APathPrefix: "/a2a",
		LogLevel:      "info",
		AgentCard: AgentCardConfig{
			A2AVersion:  "1.0",
			ID:          "go-a2a-server",
			Name:        "Go A2A Server",
			Description: "A standalone A2A server implemented in Go",
			Skills: []SkillConfig{
				{
					ID:          "echo",
					Name:        "Echo",
					Description: "Echoes back the input message",
				},
			},
			Capabilities: &CapabilitiesConfig{
				SupportsStreaming:        true,
				SupportsSessions:         true,
				SupportsPushNotification: true,
			},
		},
	}
}
