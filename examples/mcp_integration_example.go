// Package main provides examples of using the go-a2a library.
// This file demonstrates MCP integration with the A2A protocol.
//
// To avoid conflicts with other examples, you can build this file specifically with:
// go build -o mcp_example examples/mcp_integration_example.go
//go:build mcp_example
// +build mcp_example

package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/sammcj/go-a2a/a2a"
	"github.com/sammcj/go-a2a/client"
	"github.com/sammcj/go-a2a/server"
)

// SimpleMCPClient is a simple implementation of the server.MCPClient interface.
type SimpleMCPClient struct {
	// In a real implementation, this would contain the necessary fields to connect to an MCP server.
}

// CallTool calls an MCP tool with the given name and parameters.
func (c *SimpleMCPClient) CallTool(ctx context.Context, toolName string, params map[string]interface{}) (interface{}, error) {
	// In a real implementation, this would make an actual call to an MCP server.
	// For this example, we'll just return a mock response.
	switch toolName {
	case "weather":
		location, ok := params["location"].(string)
		if !ok {
			return nil, fmt.Errorf("location parameter is required")
		}
		return map[string]interface{}{
			"location":    location,
			"temperature": 22,
			"condition":   "Sunny",
			"humidity":    65,
			"wind":        "10 km/h",
		}, nil
	case "calculator":
		operation, ok := params["operation"].(string)
		if !ok {
			return nil, fmt.Errorf("operation parameter is required")
		}
		a, ok := params["a"].(float64)
		if !ok {
			return nil, fmt.Errorf("a parameter is required")
		}
		b, ok := params["b"].(float64)
		if !ok {
			return nil, fmt.Errorf("b parameter is required")
		}

		var result float64
		switch operation {
		case "add":
			result = a + b
		case "subtract":
			result = a - b
		case "multiply":
			result = a * b
		case "divide":
			if b == 0 {
				return nil, fmt.Errorf("division by zero")
			}
			result = a / b
		default:
			return nil, fmt.Errorf("unsupported operation: %s", operation)
		}

		return map[string]interface{}{
			"operation": operation,
			"a":         a,
			"b":         b,
			"result":    result,
		}, nil
	default:
		return nil, fmt.Errorf("unknown tool: %s", toolName)
	}
}

// ReadResource reads an MCP resource with the given URI.
func (c *SimpleMCPClient) ReadResource(ctx context.Context, uri string) (string, string, error) {
	// In a real implementation, this would make an actual call to an MCP server.
	// For this example, we'll just return a mock response.
	return "This is a sample resource", "text/plain", nil
}

// GetAvailableTools returns a list of available tools from the MCP server.
func (c *SimpleMCPClient) GetAvailableTools(ctx context.Context) ([]server.MCPToolInfo, error) {
	// In a real implementation, this would make an actual call to an MCP server.
	// For this example, we'll just return a mock response.
	return []server.MCPToolInfo{
		{
			Name:        "weather",
			Description: "Get weather information for a location",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"location": map[string]interface{}{
						"type":        "string",
						"description": "The location to get weather for",
					},
				},
				"required": []string{"location"},
			},
		},
		{
			Name:        "calculator",
			Description: "Perform mathematical calculations",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"operation": map[string]interface{}{
						"type":        "string",
						"description": "The operation to perform (add, subtract, multiply, divide)",
						"enum":        []string{"add", "subtract", "multiply", "divide"},
					},
					"a": map[string]interface{}{
						"type":        "number",
						"description": "First number",
					},
					"b": map[string]interface{}{
						"type":        "number",
						"description": "Second number",
					},
				},
				"required": []string{"operation", "a", "b"},
			},
		},
	}, nil
}

// GetAvailableResources returns a list of available resources from the MCP server.
func (c *SimpleMCPClient) GetAvailableResources(ctx context.Context) ([]server.MCPResourceInfo, error) {
	// In a real implementation, this would make an actual call to an MCP server.
	// For this example, we'll just return a mock response.
	return []server.MCPResourceInfo{
		{
			URI:         "sample://resource",
			Name:        "Sample Resource",
			Description: "A sample resource",
			MIMEType:    "text/plain",
		},
	}, nil
}

func main() {
	// Create an agent card
	agentCard := &a2a.AgentCard{
		A2AVersion: "1.0",
		ID:         "mcp-agent",
		Name:       "MCP Agent",
		Description: func() *string {
			s := "An intelligent agent powered by an LLM with MCP tools"
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

	// Create a simple MCP client
	mcpClient := &SimpleMCPClient{}

	// Create and start the server with an MCP-augmented gollm agent
	a2aServer, err := server.NewServer(
		server.WithAgentCard(agentCard),
		server.WithMCPToolAugmentedGollmAgent(
			"ollama",                       // provider
			"deepcoder:14b-preview-q4_K_M", // model
			"",                             // API key (not needed for Ollama)
			mcpClient,                      // MCP client
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

	fmt.Println("Sending task to MCP agent...")

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
					// The TaskID is in the TaskStatusUpdateEvent, not in the TaskStatus
					// For this example, we'll just use a placeholder
					taskID = "task_id"
				}

				if update.Status != nil && update.Status.State == a2a.TaskStateWorking && update.Status.Message != nil {
					// Print agent's response
					for _, part := range update.Status.Message.Parts {
						if textPart, ok := part.(a2a.TextPart); ok {
							fmt.Print(textPart.Text)
						}
					}
				}
			} else if update.Type == "artifact" {
				// Print artifact content
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
