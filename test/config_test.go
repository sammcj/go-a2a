package client

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/sammcj/go-a2a/cmd/common"
)

func TestLoadConfig(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "config_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	jsonFile := filepath.Join(tempDir, "test.json")
	yamlFile := filepath.Join(tempDir, "test.yaml")
	invalidFile := filepath.Join(tempDir, "invalid.txt")

	// Create valid JSON config file
	jsonContent := []byte(`{"listenAddress": ":8081", "logLevel": "debug"}`)
	if err := os.WriteFile(jsonFile, jsonContent, 0644); err != nil {
		t.Fatalf("Failed to write JSON config file: %v", err)
	}

	// Create valid YAML config file
	yamlContent := []byte(`listenAddress: ":8082"\nlogLevel: "info"`)
	if err := os.WriteFile(yamlFile, yamlContent, 0644); err != nil {
		t.Fatalf("Failed to write YAML config file: %v", err)
	}

	// Create invalid file
	if err := os.WriteFile(invalidFile, []byte("invalid content"), 0644); err != nil {
		t.Fatalf("Failed to write invalid file: %v", err)
	}

	tests := []struct {
		name        string
		filePath    string
		expected    common.ServerConfig
		expectedErr bool
	}{
		{
			name:        "Load JSON config",
			filePath:    jsonFile,
			expected:    common.ServerConfig{ListenAddress: ":8081", LogLevel: "debug"},
			expectedErr: false,
		},
		{
			name:        "Load YAML config",
			filePath:    yamlFile,
			expected:    common.ServerConfig{ListenAddress: ":8082", LogLevel: "info"},
			expectedErr: false,
		},
		{
			name:        "File not found",
			filePath:    filepath.Join(tempDir, "notfound.json"),
			expected:    common.ServerConfig{},
			expectedErr: true,
		},
		{
			name:        "Invalid file content",
			filePath:    invalidFile,
			expected:    common.ServerConfig{},
			expectedErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := common.LoadConfig[common.ServerConfig](tt.filePath)
			if (err != nil) != tt.expectedErr {
				t.Errorf("LoadConfig() error = %v, expectedErr %v", err, tt.expectedErr)
				return
			}
			if err == nil && !reflect.DeepEqual(*got, tt.expected) {
				t.Errorf("LoadConfig() got = %v, want %v", *got, tt.expected)
			}
		})
	}
}

func TestSaveConfig(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "config_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	jsonFile := filepath.Join(tempDir, "save.json")
	yamlFile := filepath.Join(tempDir, "save.yaml")

	tests := []struct {
		name        string
		filePath    string
		config      common.ServerConfig
		expectedErr bool
	}{
		{
			name:        "Save JSON config",
			filePath:    jsonFile,
			config:      common.ServerConfig{ListenAddress: ":8083", LogLevel: "warn"},
			expectedErr: false,
		},
		{
			name:        "Save YAML config",
			filePath:    yamlFile,
			config:      common.ServerConfig{ListenAddress: ":8084", LogLevel: "error"},
			expectedErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := common.SaveConfig(tt.config, tt.filePath)
			if (err != nil) != tt.expectedErr {
				t.Errorf("SaveConfig() error = %v, expectedErr %v", err, tt.expectedErr)
				return
			}
			if err == nil {
				loadedConfig, loadErr := common.LoadConfig[common.ServerConfig](tt.filePath)
				if loadErr != nil {
					t.Errorf("Failed to load saved config: %v", loadErr)
					return
				}
				if !reflect.DeepEqual(*loadedConfig, tt.config) {
					t.Errorf("Saved config = %v, expected = %v", *loadedConfig, tt.config)
				}
			}
		})
	}
}

func TestConvertToAgentCard(t *testing.T) {
	skill := common.SkillConfig{ID: "skill1", Name: "Skill 1", Description: "Test skill"}
	provider := common.ProviderConfig{Name: "Provider 1", URI: "http://provider.com"}
	capabilities := common.CapabilitiesConfig{SupportsStreaming: true, SupportsSessions: false, SupportsPushNotification: true}

	authConfig := common.AuthConfig{Type: "test", Scheme: "testScheme", Configuration: map[string]interface{}{"key": "value"}}
	testCard := common.AgentCardConfig{
		A2AVersion:       "1.0",
		ID:               "test-agent",
		Name:             "Test Agent",
		Description:      "This is a test agent.",
		IconURI:          "http://test.com/icon.png",
		Provider:         &provider,
		Skills:           []common.SkillConfig{skill},
		Capabilities:     &capabilities,
		Authentication:   []common.AuthConfig{authConfig},
		ContactEmail:     "test@test.com",
		LegalInfoURI:     "http://test.com/legal",
		HomepageURI:      "http://test.com/home",
		DocumentationURI: "http://test.com/docs",
	}

	a2aCard := common.ConvertToAgentCard(&testCard)

	if a2aCard.A2AVersion != testCard.A2AVersion {
		t.Errorf("ConvertToAgentCard() A2AVersion = %v, expected %v", a2aCard.A2AVersion, testCard.A2AVersion)
	}
	if a2aCard.ID != testCard.ID {
		t.Errorf("ConvertToAgentCard() ID = %v, expected %v", a2aCard.ID, testCard.ID)
	}
	if a2aCard.Name != testCard.Name {
		t.Errorf("ConvertToAgentCard() Name = %v, expected %v", a2aCard.Name, testCard.Name)
	}
	if *a2aCard.Description != testCard.Description {
		t.Errorf("ConvertToAgentCard() Description = %v, expected %v", *a2aCard.Description, testCard.Description)
	}
	if *a2aCard.IconURI != testCard.IconURI {
		t.Errorf("ConvertToAgentCard() IconURI = %v, expected %v", *a2aCard.IconURI, testCard.IconURI)
	}
	if a2aCard.Provider.Name != testCard.Provider.Name {
		t.Errorf("ConvertToAgentCard() Provider Name = %v, expected %v", a2aCard.Provider.Name, testCard.Provider.Name)
	}
	if *a2aCard.Provider.URI != testCard.Provider.URI {
		t.Errorf("ConvertToAgentCard() Provider URI = %v, expected %v", *a2aCard.Provider.URI, testCard.Provider.URI)
	}
	if len(a2aCard.Skills) != len(testCard.Skills) {
		t.Errorf("ConvertToAgentCard() Skills length = %v, expected %v", len(a2aCard.Skills), len(testCard.Skills))
	}
	if len(a2aCard.Authentication) != len(testCard.Authentication) {
		t.Errorf("ConvertToAgentCard() Authentication length = %v, expected %v", len(a2aCard.Authentication), len(testCard.Authentication))
	}
	if a2aCard.Capabilities.SupportsStreaming != testCard.Capabilities.SupportsStreaming {
		t.Errorf("ConvertToAgentCard() SupportsStreaming = %v, expected %v", a2aCard.Capabilities.SupportsStreaming, testCard.Capabilities.SupportsStreaming)
	}
	if a2aCard.Capabilities.SupportsSessions != testCard.Capabilities.SupportsSessions {
		t.Errorf("ConvertToAgentCard() SupportsSessions = %v, expected %v", a2aCard.Capabilities.SupportsSessions, testCard.Capabilities.SupportsSessions)
	}
	if a2aCard.Capabilities.SupportsPushNotification != testCard.Capabilities.SupportsPushNotification {
		t.Errorf("ConvertToAgentCard() SupportsPushNotification = %v, expected %v", a2aCard.Capabilities.SupportsPushNotification, testCard.Capabilities.SupportsPushNotification)
	}
	if *a2aCard.ContactEmail != testCard.ContactEmail {
		t.Errorf("ConvertToAgentCard() ContactEmail = %v, expected %v", *a2aCard.ContactEmail, testCard.ContactEmail)
	}
	if *a2aCard.LegalInfoURI != testCard.LegalInfoURI {
		t.Errorf("ConvertToAgentCard() LegalInfoURI = %v, expected %v", *a2aCard.LegalInfoURI, testCard.LegalInfoURI)
	}
	if *a2aCard.HomepageURI != testCard.HomepageURI {
		t.Errorf("ConvertToAgentCard() HomepageURI = %v, expected %v", *a2aCard.HomepageURI, testCard.HomepageURI)
	}
	if *a2aCard.DocumentationURI != testCard.DocumentationURI {
		t.Errorf("ConvertToAgentCard() DocumentationURI = %v, expected %v", *a2aCard.DocumentationURI, testCard.DocumentationURI)
	}
	for i, auth := range testCard.Authentication {
		if a2aCard.Authentication[i].Type != auth.Type {
			t.Errorf("ConvertToAgentCard() Authentication Type = %v, expected %v", a2aCard.Authentication[i].Type, auth.Type)
		}
		if *a2aCard.Authentication[i].Scheme != auth.Scheme {
			t.Errorf("ConvertToAgentCard() Authentication Scheme = %v, expected %v", *a2aCard.Authentication[i].Scheme, auth.Scheme)
		}
		if !reflect.DeepEqual(a2aCard.Authentication[i].Configuration, auth.Configuration) {
			t.Errorf("ConvertToAgentCard() Authentication Configuration = %v, expected %v", a2aCard.Authentication[i].Configuration, auth.Configuration)
		}
	}
}

func TestDefaultServerConfig(t *testing.T) {
	config := common.DefaultServerConfig()

	expectedConfig := common.ServerConfig{
		ListenAddress: ":8080",
		AgentCardPath: "/.well-known/agent.json",
		A2APathPrefix: "/a2a",
		LogLevel:      "info",
		AgentCard: common.AgentCardConfig{
			A2AVersion:  "1.0",
			ID:          "go-a2a-server",
			Name:        "Go A2A Server",
			Description: "A standalone A2A server implemented in Go",
			Skills: []common.SkillConfig{
				{
					ID:          "echo",
					Name:        "Echo",
					Description: "Echoes back the input message",
				},
			},
			Capabilities: &common.CapabilitiesConfig{
				SupportsStreaming:        true,
				SupportsSessions:         true,
				SupportsPushNotification: true,
			},
		},
	}

	if config.ListenAddress != expectedConfig.ListenAddress {
		t.Errorf("DefaultServerConfig() ListenAddress = %v, expected %v", config.ListenAddress, expectedConfig.ListenAddress)
	}
	if config.AgentCardPath != expectedConfig.AgentCardPath {
		t.Errorf("DefaultServerConfig() AgentCardPath = %v, expected %v", config.AgentCardPath, expectedConfig.AgentCardPath)
	}
	if config.A2APathPrefix != expectedConfig.A2APathPrefix {
		t.Errorf("DefaultServerConfig() A2APathPrefix = %v, expected %v", config.A2APathPrefix, expectedConfig.A2APathPrefix)
	}
	if config.LogLevel != expectedConfig.LogLevel {
		t.Errorf("DefaultServerConfig() LogLevel = %v, expected %v", config.LogLevel, expectedConfig.LogLevel)
	}

	if config.AgentCard.A2AVersion != expectedConfig.AgentCard.A2AVersion {
		t.Errorf("DefaultServerConfig() AgentCard.A2AVersion = %v, expected %v", config.AgentCard.A2AVersion, expectedConfig.AgentCard.A2AVersion)
	}
	if config.AgentCard.ID != expectedConfig.AgentCard.ID {
		t.Errorf("DefaultServerConfig() AgentCard.ID = %v, expected %v", config.AgentCard.ID, expectedConfig.AgentCard.ID)
	}
	if config.AgentCard.Name != expectedConfig.AgentCard.Name {
		t.Errorf("DefaultServerConfig() AgentCard.Name = %v, expected %v", config.AgentCard.Name, expectedConfig.AgentCard.Name)
	}
	if config.AgentCard.Description != expectedConfig.AgentCard.Description {
		t.Errorf("DefaultServerConfig() AgentCard.Description = %v, expected %v", config.AgentCard.Description, expectedConfig.AgentCard.Description)
	}
	if len(config.AgentCard.Skills) != len(expectedConfig.AgentCard.Skills) {
		t.Errorf("DefaultServerConfig() AgentCard.Skills length = %v, expected %v", len(config.AgentCard.Skills), len(expectedConfig.AgentCard.Skills))
	}
	if config.AgentCard.Skills[0].ID != expectedConfig.AgentCard.Skills[0].ID {
		t.Errorf("DefaultServerConfig() AgentCard.Skills[0].ID = %v, expected %v", config.AgentCard.Skills[0].ID, expectedConfig.AgentCard.Skills[0].ID)
	}
	if config.AgentCard.Skills[0].Name != expectedConfig.AgentCard.Skills[0].Name {
		t.Errorf("DefaultServerConfig() AgentCard.Skills[0].Name = %v, expected %v", config.AgentCard.Skills[0].Name, expectedConfig.AgentCard.Skills[0].Name)
	}
	if config.AgentCard.Skills[0].Description != expectedConfig.AgentCard.Skills[0].Description {
		t.Errorf("DefaultServerConfig() AgentCard.Skills[0].Description = %v, expected %v", config.AgentCard.Skills[0].Description, expectedConfig.AgentCard.Skills[0].Description)
	}
	if config.AgentCard.Capabilities.SupportsStreaming != expectedConfig.AgentCard.Capabilities.SupportsStreaming {
		t.Errorf("DefaultServerConfig() AgentCard.Capabilities.SupportsStreaming = %v, expected %v", config.AgentCard.Capabilities.SupportsStreaming, expectedConfig.AgentCard.Capabilities.SupportsStreaming)
	}
	if config.AgentCard.Capabilities.SupportsSessions != expectedConfig.AgentCard.Capabilities.SupportsSessions {
		t.Errorf("DefaultServerConfig() AgentCard.Capabilities.SupportsSessions = %v, expected %v", config.AgentCard.Capabilities.SupportsSessions, expectedConfig.AgentCard.Capabilities.SupportsSessions)
	}
	if config.AgentCard.Capabilities.SupportsPushNotification != expectedConfig.AgentCard.Capabilities.SupportsPushNotification {
		t.Errorf("DefaultServerConfig() AgentCard.Capabilities.SupportsPushNotification = %v, expected %v", config.AgentCard.Capabilities.SupportsPushNotification, expectedConfig.AgentCard.Capabilities.SupportsPushNotification)
	}
}

func TestDefaultClientConfig(t *testing.T) {
	config := common.DefaultClientConfig()

	expectedConfig := common.ClientConfig{
		DefaultAgentURL: "http://localhost:8080",
		OutputFormat:    "pretty",
	}

	if config.DefaultAgentURL != expectedConfig.DefaultAgentURL {
		t.Errorf("DefaultClientConfig() DefaultAgentURL = %v, expected %v", config.DefaultAgentURL, expectedConfig.DefaultAgentURL)
	}
	if config.OutputFormat != expectedConfig.OutputFormat {
		t.Errorf("DefaultClientConfig() OutputFormat = %v, expected %v", config.OutputFormat, expectedConfig.OutputFormat)
	}
}

func TestDefaultLLMConfig(t *testing.T) {
	config := common.DefaultLLMConfig()

	expectedConfig := common.LLMConfig{
		Provider:     "openai",
		Model:        "",
		APIKey:       "",
		SystemPrompt: "",
		BaseUrl:      "",
		Options:      map[string]interface{}{"temperature": 0.7},
	}
	if config.Provider != expectedConfig.Provider {
		t.Errorf("DefaultLLMConfig() Provider = %v, expected %v", config.Provider, expectedConfig.Provider)
	}
	if config.Model != expectedConfig.Model {
		t.Errorf("DefaultLLMConfig() Model = %v, expected %v", config.Model, expectedConfig.Model)
	}
	if config.APIKey != expectedConfig.APIKey {
		t.Errorf("DefaultLLMConfig() APIKey = %v, expected %v", config.APIKey, expectedConfig.APIKey)
	}
	if config.SystemPrompt != expectedConfig.SystemPrompt {
		t.Errorf("DefaultLLMConfig() SystemPrompt = %v, expected %v", config.SystemPrompt, expectedConfig.SystemPrompt)
	}
	if config.BaseUrl != expectedConfig.BaseUrl {
		t.Errorf("DefaultLLMConfig() BaseUrl = %v, expected %v", config.BaseUrl, expectedConfig.BaseUrl)
	}
	if len(config.Options) != len(expectedConfig.Options) {
		t.Errorf("DefaultLLMConfig() Options length = %v, expected %v", len(config.Options), len(expectedConfig.Options))
	}
	if fmt.Sprintf("%v", config.Options["temperature"]) != fmt.Sprintf("%v", expectedConfig.Options["temperature"]) {
		t.Errorf("DefaultLLMConfig() Options[\"temperature\"] = %v, expected %v", config.Options["temperature"], expectedConfig.Options["temperature"])
	}
}
