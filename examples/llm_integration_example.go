// Package main provides examples of using the go-a2a library.
// This file demonstrates LLM integration with the A2A protocol.
//
// To avoid conflicts with other examples, you can build this file specifically with:
// go build -o llm_example examples/llm_integration_example.go
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/sammcj/go-a2a/a2a"
	"github.com/sammcj/go-a2a/client"
	"github.com/sammcj/go-a2a/server"
)

// WeatherTool is a simple tool that provides weather information
type WeatherTool struct{}

// Name returns the name of the tool
func (t *WeatherTool) Name() string {
	return "weather"
}

// Description returns a description of the tool
func (t *WeatherTool) Description() string {
	return "Get weather information for a location"
}

// Execute executes the tool with the given parameters
func (t *WeatherTool) Execute(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	// In a real implementation, this would call a weather API
	location, ok := params["location"].(string)
	if !ok {
		return nil, fmt.Errorf("location parameter is required")
	}

	// Mock weather data
	weatherData := map[string]interface{}{
		"location":    location,
		"temperature": 22,
		"condition":   "Sunny",
		"humidity":    65,
		"wind":        "10 km/h",
	}

	return weatherData, nil
}

// CalculatorTool is a simple tool that performs calculations
type CalculatorTool struct{}

// Name returns the name of the tool
func (t *CalculatorTool) Name() string {
	return "calculator"
}

// Description returns a description of the tool
func (t *CalculatorTool) Description() string {
	return "Perform mathematical calculations"
}

// Execute executes the tool with the given parameters
func (t *CalculatorTool) Execute(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	expression, ok := params["expression"].(string)
	if !ok {
		return nil, fmt.Errorf("expression parameter is required")
	}

	// This is a very simple calculator that only handles addition
	// In a real implementation, you would use a proper expression parser
	parts := strings.Split(expression, "+")
	if len(parts) != 2 {
		return nil, fmt.Errorf("only addition is supported")
	}

	// Parse the numbers
	var a, b float64
	_, err := fmt.Sscanf(strings.TrimSpace(parts[0]), "%f", &a)
	if err != nil {
		return nil, fmt.Errorf("invalid first number: %w", err)
	}
	_, err = fmt.Sscanf(strings.TrimSpace(parts[1]), "%f", &b)
	if err != nil {
		return nil, fmt.Errorf("invalid second number: %w", err)
	}

	// Calculate the result
	result := a + b

	return map[string]interface{}{
		"expression": expression,
		"result":     result,
	}, nil
}

func main() {
	// Create an agent card
	agentCard := &a2a.AgentCard{
		A2AVersion: "1.0",
		ID:         "llm-agent",
		Name:       "LLM Agent",
		Description: func() *string {
			s := "An intelligent agent powered by an LLM with tool capabilities"
			return &s
		}(),
		Skills: []a2a.AgentSkill{
			{
				ID:   "general-assistance",
				Name: "General Assistance",
				Description: func() *string {
					s := "Provides helpful responses to general questions and requests"
					return &s
				}(),
			},
			{
				ID:   "weather-info",
				Name: "Weather Information",
				Description: func() *string {
					s := "Provides weather information for a location"
					return &s
				}(),
			},
			{
				ID:   "calculator",
				Name: "Calculator",
				Description: func() *string {
					s := "Performs mathematical calculations"
					return &s
				}(),
			},
		},
		Capabilities: &a2a.AgentCapabilities{
			SupportsStreaming: true,
		},
	}

	// Define tools
	tools := []server.Tool{
		&WeatherTool{},
		&CalculatorTool{},
	}

	// Create and start the server with a tool-augmented gollm agent
	a2aServer, err := server.NewServer(
		server.WithAgentCard(agentCard),
		server.WithToolAugmentedGollmAgent(
			"ollama",                // provider
			"deepcoder:14b-preview-q4_K_M", // model
			"",                      // API key (not needed for Ollama)
			tools,                   // tools
		),
		server.WithListenAddress(":8080"),
	)
	if err != nil {
		log.Fatalf("Failed to create A2A server: %v", err)
	}

	// Start the server in a goroutine
	go func() {
		if err := a2aServer.Start(); err != nil {
			log.Fatalf("Failed to start A2A server: %v", err)
		}
	}()

	// Wait for the server to start
	time.Sleep(1 * time.Second)

	// Create a client
	a2aClient, err := client.NewClient(
		client.WithBaseURL("http://localhost:8080"),
	)
	if err != nil {
		log.Fatalf("Failed to create A2A client: %v", err)
	}

	// Example 1: Ask about Go for backend development
	runExample(a2aClient, "What are the benefits of using Go for backend development?")

	// Example 2: Ask about the weather (should use the weather tool)
	runExample(a2aClient, "What's the weather like in London?")

	// Example 3: Ask for a calculation (should use the calculator tool)
	runExample(a2aClient, "Calculate 123.45 + 67.89")
}

func runExample(a2aClient *client.Client, query string) {
	fmt.Printf("\n\n--- Example: %s ---\n\n", query)

	// Create a message
	message := a2a.Message{
		Role: a2a.RoleUser,
		Parts: []a2a.Part{
			a2a.TextPart{
				Type: "text",
				Text: query,
			},
		},
	}

	fmt.Println("Sending task to LLM agent...")

	// Send a task with streaming
	updateChan, errChan := a2aClient.SendSubscribe(context.Background(), &a2a.TaskSendParams{
		Message: message,
	})

	// Process updates
	var taskID string
	for {
		select {
		case update, ok := <-updateChan:
			if !ok {
				// Channel closed, all updates received
				fmt.Println("\nTask completed.")
				return
			}

			if update.Type == "status" {
				// Extract task ID from the status update
				if taskID == "" && update.Status != nil {
					taskID = update.Status.TaskID
				}

				// Check if we have a message to display
				if update.Status != nil && update.Status.State == a2a.TaskStateWorking && update.Status.Message != nil {
					// Print agent's response
					for _, part := range update.Status.Message.Parts {
						if textPart, ok := part.(a2a.TextPart); ok {
							fmt.Print(textPart.Text)
						}
					}
				}
			} else if update.Type == "artifact" {
				// Print artifact content if it's a text part
				if update.Artifact != nil && update.Artifact.Part != nil {
					if textPart, ok := update.Artifact.Part.(a2a.TextPart); ok {
						fmt.Print(textPart.Text)
					}
				}
			}

		case err := <-errChan:
			log.Fatalf("Error receiving updates: %v", err)
		case <-time.After(30 * time.Second):
			// Timeout after 30 seconds
			fmt.Println("\nTimeout waiting for response.")

			// Cancel the task
			if taskID != "" {
				_, err := a2aClient.CancelTask(context.Background(), taskID)
				if err != nil {
					log.Printf("Failed to cancel task: %v", err)
				}
			}

			os.Exit(1)
		}
	}
}
