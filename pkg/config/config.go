package config

// ConfigFormat represents the format of a configuration file.
type ConfigFormat string

const (
	// ConfigFormatJSON represents JSON format.
	ConfigFormatJSON ConfigFormat = "json"
	// ConfigFormatYAML represents YAML format.
	ConfigFormatYAML ConfigFormat = "yaml"
)

// AuthConfig represents authentication configuration.
type AuthConfig struct {
	Type          string                 `json:"type" yaml:"type"`
	Scheme        string                 `json:"scheme,omitempty" yaml:"scheme,omitempty"`
	Configuration map[string]interface{} `json:"configuration,omitempty" yaml:"configuration,omitempty"`
}

// AgentCardConfig represents the configuration for an agent card.
type AgentCardConfig struct {
	A2AVersion       string              `json:"a2aVersion" yaml:"a2aVersion"`
	ID               string              `json:"id" yaml:"id"`
	Name             string              `json:"name" yaml:"name"`
	Description      string              `json:"description,omitempty" yaml:"description,omitempty"`
	IconURI          string              `json:"iconUri,omitempty" yaml:"iconUri,omitempty"`
	Provider         *ProviderConfig     `json:"provider,omitempty" yaml:"provider,omitempty"`
	Skills           []SkillConfig       `json:"skills" yaml:"skills"`
	Capabilities     *CapabilitiesConfig `json:"capabilities,omitempty" yaml:"capabilities,omitempty"`
	Authentication   []AuthConfig        `json:"authentication,omitempty" yaml:"authentication,omitempty"`
	ContactEmail     string              `json:"contactEmail,omitempty" yaml:"contactEmail,omitempty"`
	LegalInfoURI     string              `json:"legalInfoUri,omitempty" yaml:"legalInfoUri,omitempty"`
	HomepageURI      string              `json:"homepageUri,omitempty" yaml:"homepageUri,omitempty"`
	DocumentationURI string              `json:"documentationUri,omitempty" yaml:"documentationUri,omitempty"`
}

// ProviderConfig represents the configuration for an agent provider.
type ProviderConfig struct {
	Name string `json:"name" yaml:"name"`
	URI  string `json:"uri,omitempty" yaml:"uri,omitempty"`
}

// SkillConfig represents the configuration for an agent skill.
type SkillConfig struct {
	ID             string      `json:"id" yaml:"id"`
	Name           string      `json:"name" yaml:"name"`
	Description    string      `json:"description,omitempty" yaml:"description,omitempty"`
	InputSchema    interface{} `json:"inputSchema,omitempty" yaml:"inputSchema,omitempty"`
	ArtifactSchema interface{} `json:"artifactSchema,omitempty" yaml:"artifactSchema,omitempty"`
}

// CapabilitiesConfig represents the configuration for agent capabilities.
type CapabilitiesConfig struct {
	SupportsStreaming        bool `json:"supportsStreaming" yaml:"supportsStreaming"`
	SupportsSessions         bool `json:"supportsSessions" yaml:"supportsSessions"`
	SupportsPushNotification bool `json:"supportsPushNotification" yaml:"supportsPushNotification"`
}

// ServerConfig represents the configuration for the A2A server.
type ServerConfig struct {
	ListenAddress string          `json:"listenAddress" yaml:"listenAddress"`
	AgentCard     AgentCardConfig `json:"agentCard" yaml:"agentCard"`
	AgentCardPath string          `json:"agentCardPath" yaml:"agentCardPath"`
	A2APathPrefix string          `json:"a2aPathPrefix" yaml:"a2aPathPrefix"`
	LogLevel      string          `json:"logLevel" yaml:"logLevel"`
	PluginPath    string          `json:"pluginPath" yaml:"pluginPath"`
	LLMConfig     LLMConfig       `json:"llmConfig" yaml:"llmConfig"`
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
