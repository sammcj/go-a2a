// Package gollm provides an implementation of the LLM interface using the gollm library.
package gollm

// options contains the configuration options for the gollm adapter.
type options struct {
	// Provider is the LLM provider to use (e.g., "ollama", "openai").
	Provider string

	// Model is the model to use (e.g., "llama3", "gpt-4o").
	Model string

	// APIKey is the API key to use for authentication (not needed for Ollama).
	APIKey string

	// MaxTokens is the maximum number of tokens to generate.
	MaxTokens int

	// Memory is the size of the memory buffer for context retention.
	Memory int

	// MaxContextSize is the maximum number of tokens the model can process.
	MaxContextSize int

	// Capabilities is a list of capabilities the model supports.
	Capabilities []string

	// InputModalities is a list of input modalities the model supports.
	InputModalities []string

	// OutputModalities is a list of output modalities the model supports.
	OutputModalities []string
}

// Option configures the gollm adapter.
type Option func(*options)

// defaultOptions returns the default options for the gollm adapter.
func defaultOptions() *options {
	return &options{
		Provider:         "ollama",
		Model:            "llama3",
		MaxTokens:        1000,
		MaxContextSize:   8192,
		Capabilities:     []string{"text-generation"},
		InputModalities:  []string{"text/plain"},
		OutputModalities: []string{"text/plain"},
	}
}

// WithProvider sets the LLM provider.
func WithProvider(provider string) Option {
	return func(o *options) {
		o.Provider = provider
	}
}

// WithModel sets the LLM model.
func WithModel(model string) Option {
	return func(o *options) {
		o.Model = model
	}
}

// WithAPIKey sets the API key for authentication.
func WithAPIKey(apiKey string) Option {
	return func(o *options) {
		o.APIKey = apiKey
	}
}

// WithMaxTokens sets the maximum number of tokens to generate.
func WithMaxTokens(maxTokens int) Option {
	return func(o *options) {
		o.MaxTokens = maxTokens
	}
}

// WithMemory sets the memory size for context retention.
func WithMemory(memory int) Option {
	return func(o *options) {
		o.Memory = memory
	}
}

// WithMaxContextSize sets the maximum context size for the model.
func WithMaxContextSize(maxContextSize int) Option {
	return func(o *options) {
		o.MaxContextSize = maxContextSize
	}
}

// WithCapabilities sets the capabilities of the model.
func WithCapabilities(capabilities ...string) Option {
	return func(o *options) {
		o.Capabilities = capabilities
	}
}

// WithInputModalities sets the input modalities of the model.
func WithInputModalities(inputModalities ...string) Option {
	return func(o *options) {
		o.InputModalities = inputModalities
	}
}

// WithOutputModalities sets the output modalities of the model.
func WithOutputModalities(outputModalities ...string) Option {
	return func(o *options) {
		o.OutputModalities = outputModalities
	}
}
