package common

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/sammcj/go-a2a/a2a"
	"github.com/sammcj/go-a2a/pkg/config"
	"gopkg.in/yaml.v3"
)

// ConfigFormat represents the format of a configuration file.
type ConfigFormat string
const (
	// ConfigFormatJSON represents JSON format.
	ConfigFormatJSON ConfigFormat = "json"
	// ConfigFormatYAML represents YAML format.
	ConfigFormatYAML ConfigFormat = "yaml"
)

// ProviderConfig represents the configuration for an agent provider.
type ProviderConfig struct {
	Name string `json:"name" yaml:"name"`
	URI  string `json:"uri,omitempty" yaml:"uri,omitempty"`
}

// ServerConfig represents the configuration for the A2A server.
type ServerConfig struct {
	config.ServerConfig
	LLMConfig config.LLMConfig `json:"llmConfig" yaml:"llmConfig"`
}


// ClientConfig represents the configuration for the A2A client.
type ClientConfig struct {
	DefaultAgentURL string                 `json:"defaultAgentUrl" yaml:"defaultAgentUrl"`
	Authentication  map[string]interface{} `json:"authentication,omitempty" yaml:"authentication,omitempty"`
	OutputFormat    string                 `json:"outputFormat" yaml:"outputFormat"`
}

// LoadConfig loads a configuration file.
func LoadConfig[T any](filePath string) (*T, error) {
	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("configuration file not found: %s", filePath)
	}

	// Read file
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read configuration file: %w", err)
	}

	// Determine format based on file extension
	format := ConfigFormatJSON
	if strings.HasSuffix(strings.ToLower(filePath), ".yaml") || strings.HasSuffix(strings.ToLower(filePath), ".yml") {
		format = ConfigFormatYAML
	}

	// Parse configuration
	var config T
	switch format {
	case ConfigFormatJSON:
		if err := json.Unmarshal(data, &config); err != nil {
			return nil, fmt.Errorf("failed to parse JSON configuration: %w", err)
		}
	case ConfigFormatYAML:
		if err := yaml.Unmarshal(data, &config); err != nil {
			return nil, fmt.Errorf("failed to parse YAML configuration: %w", err)
		}
	default:
		return nil, fmt.Errorf("unsupported configuration format: %s", format)
	}

	return &config, nil
}

// SaveConfig saves a configuration to a file.
func SaveConfig[T any](config T, filePath string) error {
	// Create directory if it doesn't exist
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Determine format based on file extension
	format := ConfigFormatJSON
	if strings.HasSuffix(strings.ToLower(filePath), ".yaml") || strings.HasSuffix(strings.ToLower(filePath), ".yml") {
		format = ConfigFormatYAML
	}

	// Marshal configuration
	var data []byte
	var err error
	switch format {
	case ConfigFormatJSON:
		data, err = json.MarshalIndent(config, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON configuration: %w", err)
		}
	case ConfigFormatYAML:
		data, err = yaml.Marshal(config)
		if err != nil {
			return fmt.Errorf("failed to marshal YAML configuration: %w", err)
		}
	default:
		return fmt.Errorf("unsupported configuration format: %s", format)
	}

	// Write file
	if err := ioutil.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write configuration file: %w", err)
	}

	return nil
}

// ConvertToAgentCard converts an AgentCardConfig to an a2a.AgentCard.
func ConvertToAgentCard(config config.AgentCardConfig) *a2a.AgentCard {
	card := &a2a.AgentCard{
		A2AVersion: config.A2AVersion,
		ID:         config.ID,
		Name:       config.Name,
	}

	// Set optional fields
	if config.Description != "" {
		desc := config.Description
		card.Description = &desc
	}
	if config.IconURI != "" {
		iconURI := config.IconURI
		card.IconURI = &iconURI
	}
	if config.Provider != nil {
		card.Provider = &a2a.AgentProvider{
			Name: config.Provider.Name,
		}
		if config.Provider.URI != "" {
			uri := config.Provider.URI
			card.Provider.URI = &uri
		}
	}
	if config.ContactEmail != "" {
		email := config.ContactEmail
		card.ContactEmail = &email
	}
	if config.LegalInfoURI != "" {
		legalURI := config.LegalInfoURI
		card.LegalInfoURI = &legalURI
	}
	if config.HomepageURI != "" {
		homeURI := config.HomepageURI
		card.HomepageURI = &homeURI
	}
	if config.DocumentationURI != "" {
		docURI := config.DocumentationURI
		card.DocumentationURI = &docURI
	}

	// Convert skills
	card.Skills = make([]a2a.AgentSkill, len(config.Skills))
	for i, skill := range config.Skills {
			agentSkill := a2a.AgentSkill{
			ID:   skill.ID,
			Name: skill.Name,
		}
		if skill.Description != "" {
			desc := skill.Description
			agentSkill.Description = &desc
		}
		if skill.InputSchema != nil {
			agentSkill.InputSchema = skill.InputSchema
		}
		if skill.ArtifactSchema != nil {
			agentSkill.ArtifactSchema = skill.ArtifactSchema
		}
		card.Skills[i] = agentSkill
	}

	// Convert capabilities
	if config.Capabilities != nil {
		card.Capabilities = &a2a.AgentCapabilities{
			SupportsStreaming:       config.Capabilities.SupportsStreaming,
			SupportsSessions:        config.Capabilities.SupportsSessions,
			SupportsPushNotification: config.Capabilities.SupportsPushNotification,
		}
	}

	// Convert authentication
	if len(config.Authentication) > 0 {
		card.Authentication = make([]a2a.AgentAuthentication, len(config.Authentication))
		for i, auth := range config.Authentication {
			agentAuth := a2a.AgentAuthentication{
				Type: auth.Type,
			}
			if auth.Scheme != "" {
				scheme := auth.Scheme
				agentAuth.Scheme = &scheme
			}
			if auth.Configuration != nil {
				agentAuth.Configuration = auth.Configuration
			}
			card.Authentication[i] = agentAuth
		}
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
		AgentCard: config.AgentCardConfig{
			A2AVersion: "1.0",
			ID:         "go-a2a-server",
			Name:       "Go A2A Server",
			Description: "A standalone A2A server implemented in Go",
			Skills: []config.SkillConfig{
				{
						ID:          "echo",
					Name:        "Echo",
					Description: "Echoes back the input message",
				},
			},
			Capabilities: &CapabilitiesConfig{
				SupportsStreaming:       true,
				SupportsSessions:        true,
				SupportsPushNotification: true,
			},
		},
	}
}

// DefaultLLMConfig returns a default LLM config.
func DefaultLLMConfig() config.LLMConfig {
	return config.LLMConfig{
		Provider:     "openai",
		Model:        "",
		APIKey:       "",
		SystemPrompt: "",
		BaseUrl:      "",
		Options: map[string]interface{}{"temperature": 0.7},
	}
}
// DefaultClientConfig returns a default client configuration.
func DefaultClientConfig() ClientConfig {
	return ClientConfig{
		DefaultAgentURL: "http://localhost:8080",
		OutputFormat:    "pretty",
	}
}
