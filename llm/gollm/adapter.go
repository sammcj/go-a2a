// Package gollm provides an implementation of the LLM interface using the gollm library.
package gollm

import (
	"context"
	"fmt"

	"github.com/sammcj/go-a2a/llm"
	"github.com/teilomillet/gollm"
)

// Adapter implements the LLM interface using the gollm library.
type Adapter struct {
	llmClient gollm.LLM
	modelInfo llm.LLMModelInfo
}

// NewAdapter creates a new gollm adapter.
func NewAdapter(opts ...Option) (*Adapter, error) {
	options := defaultOptions()
	for _, opt := range opts {
		opt(options)
	}

	// Create gollm client with the specified options
	var gollmOpts []func(*gollm.LLM)

	// Set provider and model
	gollmOpts = append(gollmOpts, func(l *gollm.LLM) {
		l.Provider = options.Provider
		l.Model = options.Model
		l.MaxTokens = options.MaxTokens
	})

	// Add API key if provided (not needed for Ollama)
	if options.APIKey != "" {
		gollmOpts = append(gollmOpts, func(l *gollm.LLM) {
			l.APIKey = options.APIKey
		})
	}

	// Add memory if specified
	if options.Memory > 0 {
		gollmOpts = append(gollmOpts, func(l *gollm.LLM) {
			l.Memory = options.Memory
		})
	}

	// Create a new gollm LLM instance
	var llmClient gollm.LLM

	// Apply options
	for _, opt := range gollmOpts {
		opt(&llmClient)
	}

	// Initialize the LLM
	if err := llmClient.Init(); err != nil {
		return nil, fmt.Errorf("failed to initialize gollm client: %w", err)
	}

	// Create model info
	modelInfo := llm.LLMModelInfo{
		Name:            options.Model,
		Provider:        options.Provider,
		MaxContextSize:  options.MaxContextSize,
		Capabilities:    options.Capabilities,
		InputModalities: options.InputModalities,
		OutputModalities: options.OutputModalities,
	}

	return &Adapter{
		llmClient: llmClient,
		modelInfo: modelInfo,
	}, nil
}

// Generate implements the LLM interface Generate method.
func (a *Adapter) Generate(ctx context.Context, prompt string, options ...llm.LLMOption) (string, error) {
	// Apply options
	opts := llm.DefaultLLMOptions()
	for _, opt := range options {
		opt(opts)
	}

	// Create a prompt with system instructions if provided
	fullPrompt := prompt
	if opts.SystemPrompt != "" {
		fullPrompt = opts.SystemPrompt + "\n\n" + prompt
	}

	// Set temperature if specified
	if opts.Temperature > 0 {
		a.llmClient.Temperature = opts.Temperature
	}

	// Set max tokens if specified
	if opts.MaxTokens > 0 {
		a.llmClient.MaxTokens = opts.MaxTokens
	}

	// Generate response
	response, err := a.llmClient.Completion(fullPrompt)
	if err != nil {
		return "", fmt.Errorf("gollm generation failed: %w", err)
	}

	return response, nil
}

// GenerateStream implements the LLM interface GenerateStream method.
func (a *Adapter) GenerateStream(ctx context.Context, prompt string, options ...llm.LLMOption) (<-chan llm.LLMChunk, <-chan error) {
	// Apply options
	opts := llm.DefaultLLMOptions()
	for _, opt := range options {
		opt(opts)
	}

	// Create a prompt with system instructions if provided
	fullPrompt := prompt
	if opts.SystemPrompt != "" {
		fullPrompt = opts.SystemPrompt + "\n\n" + prompt
	}

	// Set temperature if specified
	if opts.Temperature > 0 {
		a.llmClient.Temperature = opts.Temperature
	}

	// Set max tokens if specified
	if opts.MaxTokens > 0 {
		a.llmClient.MaxTokens = opts.MaxTokens
	}

	// Create channels
	chunkChan := make(chan llm.LLMChunk)
	errChan := make(chan error, 1)

	// Start streaming in a goroutine
	go func() {
		defer close(chunkChan)
		defer close(errChan)

		// Use gollm's streaming capability
		streamChan := make(chan string)
		errStreamChan := make(chan error)

		go func() {
			defer close(streamChan)
			defer close(errStreamChan)

			err := a.llmClient.CompletionStream(fullPrompt, streamChan)
			if err != nil {
				errStreamChan <- err
			}
		}()

		// Process the stream
		for {
			select {
			case chunk, ok := <-streamChan:
				if !ok {
					// Stream completed successfully
					chunkChan <- llm.LLMChunk{
						Text:      "",
						Completed: true,
					}
					return
				}

				// Send the chunk
				chunkChan <- llm.LLMChunk{
					Text:      chunk,
					Completed: false,
				}

			case err := <-errStreamChan:
				// Error occurred
				errChan <- fmt.Errorf("gollm stream error: %w", err)
				return

			case <-ctx.Done():
				// Context cancelled
				errChan <- ctx.Err()
				return
			}
		}
	}()

	return chunkChan, errChan
}

// GetModelInfo implements the LLM interface GetModelInfo method.
func (a *Adapter) GetModelInfo() llm.LLMModelInfo {
	return a.modelInfo
}
