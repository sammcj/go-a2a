// Package llm provides interfaces and utilities for integrating Large Language Models (LLMs)
// with the go-a2a library.
package llm

import (
	"context"
)

// LLMInterface defines the interface for LLM interactions.
// Implementations of this interface provide a standard way to interact with
// different LLM providers.
type LLMInterface interface {
	// Generate generates text from a prompt.
	// It takes a context for cancellation, a prompt string, and optional LLMOptions.
	// It returns the generated text and any error that occurred.
	Generate(ctx context.Context, prompt string, options ...LLMOption) (string, error)

	// GenerateStream streams text generation from a prompt.
	// It takes a context for cancellation, a prompt string, and optional LLMOptions.
	// It returns a channel for receiving chunks of generated text and a channel for errors.
	// The text channel will be closed when generation is complete or an error occurs.
	// The error channel will receive any errors that occur during generation.
	GenerateStream(ctx context.Context, prompt string, options ...LLMOption) (<-chan LLMChunk, <-chan error)

	// GetModelInfo returns information about the LLM model.
	GetModelInfo() LLMModelInfo
}

// LLMChunk represents a chunk of text from streaming generation.
type LLMChunk struct {
	// Text is the generated text chunk.
	Text string

	// Completed indicates whether this is the final chunk.
	Completed bool
}

// LLMModelInfo contains information about an LLM model.
type LLMModelInfo struct {
	// Name is the name of the model (e.g., "gpt-4o", "llama3").
	Name string

	// Provider is the name of the provider (e.g., "openai", "ollama").
	Provider string

	// MaxContextSize is the maximum number of tokens the model can process.
	MaxContextSize int

	// Capabilities is a list of capabilities the model supports (e.g., "text-generation", "image-understanding").
	Capabilities []string

	// InputModalities is a list of input modalities the model supports (e.g., "text/plain", "image/png").
	InputModalities []string

	// OutputModalities is a list of output modalities the model supports (e.g., "text/plain", "image/png").
	OutputModalities []string
}

// LLMOption defines options for LLM generation.
type LLMOption func(*LLMOptions)

// LLMOptions contains options for LLM generation.
type LLMOptions struct {
	// Temperature controls the randomness of the generation.
	// Higher values (e.g., 0.8) make the output more random,
	// while lower values (e.g., 0.2) make it more deterministic.
	// Default is typically 0.7.
	Temperature float64

	// MaxTokens is the maximum number of tokens to generate.
	// Default is typically model-dependent.
	MaxTokens int

	// StopSequences are sequences that will stop generation when encountered.
	StopSequences []string

	// SystemPrompt is a prompt that provides context or instructions to the model.
	SystemPrompt string

	// StructuredOutput contains options for generating structured output.
	StructuredOutput *StructuredOutputOptions
}

// StructuredOutputOptions contains options for generating structured output.
type StructuredOutputOptions struct {
	// Format is a description of the expected output format (e.g., "JSON", "YAML").
	Format string

	// Schema is a JSON schema for validating the output.
	Schema map[string]interface{}
}

// DefaultLLMOptions returns the default LLM options.
func DefaultLLMOptions() *LLMOptions {
	return &LLMOptions{
		Temperature: 0.7,
		MaxTokens:   1000,
	}
}

// WithTemperature sets the temperature for generation.
func WithTemperature(temperature float64) LLMOption {
	return func(o *LLMOptions) {
		o.Temperature = temperature
	}
}

// WithMaxTokens sets the maximum number of tokens to generate.
func WithMaxTokens(maxTokens int) LLMOption {
	return func(o *LLMOptions) {
		o.MaxTokens = maxTokens
	}
}

// WithStopSequences sets the stop sequences for generation.
func WithStopSequences(stopSequences ...string) LLMOption {
	return func(o *LLMOptions) {
		o.StopSequences = stopSequences
	}
}

// WithSystemPrompt sets the system prompt for generation.
func WithSystemPrompt(systemPrompt string) LLMOption {
	return func(o *LLMOptions) {
		o.SystemPrompt = systemPrompt
	}
}

// WithStructuredOutput sets options for generating structured output.
func WithStructuredOutput(format string, schema map[string]interface{}) LLMOption {
	return func(o *LLMOptions) {
		o.StructuredOutput = &StructuredOutputOptions{
			Format: format,
			Schema: schema,
		}
	}
}
