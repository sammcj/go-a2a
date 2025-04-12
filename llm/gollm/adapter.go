// Package gollm provides an implementation of the LLM interface using the gollm library.
package gollm

import (
	"context"
	"fmt"
	"time"

	"github.com/sammcj/go-a2a/llm"
	"github.com/teilomillet/gollm"
)

// Adapter implements the LLM interface using the gollm library.
type Adapter struct {
	llmClient *gollm.LLM
	modelInfo llm.LLMModelInfo
}

// NewAdapter creates a new gollm adapter.
func NewAdapter(opts ...Option) (*Adapter, error) {
	options := defaultOptions()
	for _, opt := range opts {
		opt(options)
	}

	// Create gollm client with the specified options
	gollmOpts := []gollm.Option{
		gollm.SetProvider(options.Provider),
		gollm.SetModel(options.Model),
		gollm.SetMaxTokens(options.MaxTokens),
	}

	// Add API key if provided (not needed for Ollama)
	if options.APIKey != "" {
		gollmOpts = append(gollmOpts, gollm.SetAPIKey(options.APIKey))
	}

	// Add memory if specified
	if options.Memory > 0 {
		gollmOpts = append(gollmOpts, gollm.SetMemory(options.Memory))
	}

	// Add retry options
	gollmOpts = append(gollmOpts,
		gollm.SetMaxRetries(3),
		gollm.SetRetryDelay(time.Second*2),
	)

	// Create the LLM client
	llmClient, err := gollm.NewLLM(gollmOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create gollm client: %w", err)
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

	// Create gollm prompt
	gollmPrompt := prompt

	// Apply system prompt if provided
	if opts.SystemPrompt != "" {
		gollmPrompt = gollm.NewPrompt(prompt,
			gollm.WithDirectives(opts.SystemPrompt),
		)
	} else {
		gollmPrompt = gollm.NewPrompt(prompt)
	}

	// Apply structured output if requested
	if opts.StructuredOutput != nil {
		gollmPrompt = gollm.NewPrompt(prompt,
			gollm.WithOutput(opts.StructuredOutput.Format),
		)
	}

	// Generate response
	var response string
	var err error

	if opts.StructuredOutput != nil && opts.StructuredOutput.Schema != nil {
		// Use JSON schema validation if provided
		response, err = a.llmClient.Generate(ctx, gollmPrompt, gollm.WithJSONSchemaValidation())
	} else {
		response, err = a.llmClient.Generate(ctx, gollmPrompt)
	}

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

	// Create gollm prompt
	var gollmPrompt interface{}

	// Apply system prompt if provided
	if opts.SystemPrompt != "" {
		gollmPrompt = gollm.NewPrompt(prompt,
			gollm.WithDirectives(opts.SystemPrompt),
		)
	} else {
		gollmPrompt = gollm.NewPrompt(prompt)
	}

	// Create channels
	chunkChan := make(chan llm.LLMChunk)
	errChan := make(chan error, 1)

	// Start streaming in a goroutine
	go func() {
		defer close(chunkChan)
		defer close(errChan)

		// Use gollm's streaming capability
		stream, err := a.llmClient.GenerateStream(ctx, gollmPrompt)
		if err != nil {
			errChan <- fmt.Errorf("failed to start gollm stream: %w", err)
			return
		}

		// Process the stream
		for {
			chunk, err := stream.Recv()
			if err != nil {
				if err.Error() == "EOF" {
					// Stream completed successfully
					chunkChan <- llm.LLMChunk{
						Text:      "",
						Completed: true,
					}
					return
				}

				// Error occurred
				errChan <- fmt.Errorf("gollm stream error: %w", err)
				return
			}

			// Send the chunk
			chunkChan <- llm.LLMChunk{
				Text:      chunk,
				Completed: false,
			}
		}
	}()

	return chunkChan, errChan
}

// GetModelInfo implements the LLM interface GetModelInfo method.
func (a *Adapter) GetModelInfo() llm.LLMModelInfo {
	return a.modelInfo
}
