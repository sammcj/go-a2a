package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

type ToolName string

const (
	ECHO                   ToolName = "echo"
	ADD                    ToolName = "add"
	LONG_RUNNING_OPERATION ToolName = "longRunningOperation"
	SAMPLE_LLM             ToolName = "sampleLLM"
	GET_TINY_IMAGE         ToolName = "getTinyImage"
)

type PromptName string

const (
	SIMPLE  PromptName = "simple_prompt"
	COMPLEX PromptName = "complex_prompt"
)

func NewMCPServer() *server.MCPServer {

	hooks := &server.Hooks{}

	hooks.AddBeforeAny(func(ctx context.Context, id any, method mcp.MCPMethod, message any) {
		fmt.Printf("beforeAny: %s, %v, %v\n", method, id, message)
	})
	hooks.AddOnSuccess(func(ctx context.Context, id any, method mcp.MCPMethod, message any, result any) {
		fmt.Printf("onSuccess: %s, %v, %v, %v\n", method, id, message, result)
	})
	hooks.AddOnError(func(ctx context.Context, id any, method mcp.MCPMethod, message any, err error) {
		fmt.Printf("onError: %s, %v, %v, %v\n", method, id, message, err)
	})
	hooks.AddBeforeInitialize(func(ctx context.Context, id any, message *mcp.InitializeRequest) {
		fmt.Printf("beforeInitialize: %v, %v\n", id, message)
	})
	hooks.AddAfterInitialize(func(ctx context.Context, id any, message *mcp.InitializeRequest, result *mcp.InitializeResult) {
		fmt.Printf("afterInitialize: %v, %v, %v\n", id, message, result)
	})
	hooks.AddAfterCallTool(func(ctx context.Context, id any, message *mcp.CallToolRequest, result *mcp.CallToolResult) {
		fmt.Printf("afterCallTool: %v, %v, %v\n", id, message, result)
	})
	hooks.AddBeforeCallTool(func(ctx context.Context, id any, message *mcp.CallToolRequest) {
		fmt.Printf("beforeCallTool: %v, %v\n", id, message)
	})

	mcpServer := server.NewMCPServer(
		"example-servers/everything",
		"1.0.0",
		server.WithResourceCapabilities(true, true),
		server.WithPromptCapabilities(true),
		server.WithLogging(),
		server.WithHooks(hooks),
	)

	mcpServer.AddResource(mcp.NewResource("test://static/resource",
		"Static Resource",
		mcp.WithMIMEType("text/plain"),
	), handleReadResource)
	mcpServer.AddResourceTemplate(
		mcp.NewResourceTemplate(
			"test://dynamic/resource/{id}",
			"Dynamic Resource",
		),
		handleResourceTemplate,
	)
	mcpServer.AddPrompt(mcp.NewPrompt(string(SIMPLE),
		mcp.WithPromptDescription("A simple prompt"),
	), handleSimplePrompt)
	mcpServer.AddPrompt(mcp.NewPrompt(string(COMPLEX),
		mcp.WithPromptDescription("A complex prompt"),
		mcp.WithArgument("temperature",
			mcp.ArgumentDescription("The temperature parameter for generation"),
			mcp.RequiredArgument(),
		),
		mcp.WithArgument("style",
			mcp.ArgumentDescription("The style to use for the response"),
			mcp.RequiredArgument(),
		),
	), handleComplexPrompt)
	mcpServer.AddTool(mcp.NewTool(string(ECHO),
		mcp.WithDescription("Echoes back the input"),
		mcp.WithString("message",
			mcp.Description("Message to echo"),
			mcp.Required(),
		),
	), handleEchoTool)

	mcpServer.AddTool(
		mcp.NewTool("notify"),
		handleSendNotification,
	)

	mcpServer.AddTool(mcp.NewTool(string(ADD),
		mcp.WithDescription("Adds two numbers"),
		mcp.WithNumber("a",
			mcp.Description("First number"),
			mcp.Required(),
		),
		mcp.WithNumber("b",
			mcp.Description("Second number"),
			mcp.Required(),
		),
	), handleAddTool)
	mcpServer.AddTool(mcp.NewTool(
		string(LONG_RUNNING_OPERATION),
		mcp.WithDescription(
			"Demonstrates a long running operation with progress updates",
		),
		mcp.WithNumber("duration",
			mcp.Description("Duration of the operation in seconds"),
			mcp.DefaultNumber(10),
		),
		mcp.WithNumber("steps",
			mcp.Description("Number of steps in the operation"),
			mcp.DefaultNumber(5),
		),
	), handleLongRunningOperationTool)

	// s.server.AddTool(mcp.Tool{
	// 	Name:        string(SAMPLE_LLM),
	// 	Description: "Samples from an LLM using MCP's sampling feature",
	// 	InputSchema: mcp.ToolInputSchema{
	// 		Type: "object",
	// 		Properties: map[string]interface{}{
	// 			"prompt": map[string]interface{}{
	// 				"type":        "string",
	// 				"description": "The prompt to send to the LLM",
	// 			},
	// 			"maxTokens": map[string]interface{}{
	// 				"type":        "number",
	// 				"description": "Maximum number of tokens to generate",
	// 				"default":     100,
	// 			},
	// 		},
	// 	},
	// }, s.handleSampleLLMTool)
	mcpServer.AddTool(mcp.NewTool(string(GET_TINY_IMAGE),
		mcp.WithDescription("Returns the MCP_TINY_IMAGE"),
	), handleGetTinyImageTool)

	mcpServer.AddNotificationHandler("notification", handleNotification)

	return mcpServer
}

func generateResources() []mcp.Resource {
	resources := make([]mcp.Resource, 100)
	for i := 0; i < 100; i++ {
		uri := fmt.Sprintf("test://static/resource/%d", i+1)
		if i%2 == 0 {
			resources[i] = mcp.Resource{
				URI:      uri,
				Name:     fmt.Sprintf("Resource %d", i+1),
				MIMEType: "text/plain",
			}
		} else {
			resources[i] = mcp.Resource{
				URI:      uri,
				Name:     fmt.Sprintf("Resource %d", i+1),
				MIMEType: "application/octet-stream",
			}
		}
	}
	return resources
}

func runUpdateInterval() {
	// for range s.updateTicker.C {
	// 	for uri := range s.subscriptions {
	// 		s.server.HandleMessage(
	// 			context.Background(),
	// 			mcp.JSONRPCNotification{
	// 				JSONRPC: mcp.JSONRPC_VERSION,
	// 				Notification: mcp.Notification{
	// 					Method: "resources/updated",
	// 					Params: struct {
	// 						Meta map[string]interface{} `json:"_meta,omitempty"`
	// 					}{
	// 						Meta: map[string]interface{}{"uri": uri},
	// 					},
	// 				},
	// 			},
	// 		)
	// 	}
	// }
}

func handleReadResource(
	ctx context.Context,
	request mcp.ReadResourceRequest,
) ([]mcp.ResourceContents, error) {
	return []mcp.ResourceContents{
		mcp.TextResourceContents{
			URI:      "test://static/resource",
			MIMEType: "text/plain",
			Text:     "This is a sample resource",
		},
	}, nil
}

func handleResourceTemplate(
	ctx context.Context,
	request mcp.ReadResourceRequest,
) ([]mcp.ResourceContents, error) {
	return []mcp.ResourceContents{
		mcp.TextResourceContents{
			URI:      request.Params.URI,
			MIMEType: "text/plain",
			Text:     "This is a sample resource",
		},
	}, nil
}

func handleSimplePrompt(
	ctx context.Context,
	request mcp.GetPromptRequest,
) (*mcp.GetPromptResult, error) {
	return &mcp.GetPromptResult{
		Description: "A simple prompt without arguments",
		Messages: []mcp.PromptMessage{
			{
				Role: mcp.RoleUser,
				Content: mcp.TextContent{
					Type: "text",
					Text: "This is a simple prompt without arguments.",
				},
			},
		},
	}, nil
}

func handleComplexPrompt(
	ctx context.Context,
	request mcp.GetPromptRequest,
) (*mcp.GetPromptResult, error) {
	arguments := request.Params.Arguments
	return &mcp.GetPromptResult{
		Description: "A complex prompt with arguments",
		Messages: []mcp.PromptMessage{
			{
				Role: mcp.RoleUser,
				Content: mcp.TextContent{
					Type: "text",
					Text: fmt.Sprintf(
						"This is a complex prompt with arguments: temperature=%s, style=%s",
						arguments["temperature"],
						arguments["style"],
					),
				},
			},
			{
				Role: mcp.RoleAssistant,
				Content: mcp.TextContent{
					Type: "text",
					Text: "I understand. You've provided a complex prompt with temperature and style arguments. How would you like me to proceed?",
				},
			},
			{
				Role: mcp.RoleUser,
				Content: mcp.ImageContent{
					Type:     "image",
					Data:     MCP_TINY_IMAGE,
					MIMEType: "image/png",
				},
			},
		},
	}, nil
}

func handleEchoTool(
	ctx context.Context,
	request mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	arguments := request.Params.Arguments
	message, ok := arguments["message"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid message argument")
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{
				Type: "text",
				Text: fmt.Sprintf("Echo: %s", message),
			},
		},
	}, nil
}

func handleAddTool(
	ctx context.Context,
	request mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	arguments := request.Params.Arguments
	a, ok1 := arguments["a"].(float64)
	b, ok2 := arguments["b"].(float64)
	if !ok1 || !ok2 {
		return nil, fmt.Errorf("invalid number arguments")
	}
	sum := a + b
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{
				Type: "text",
				Text: fmt.Sprintf("The sum of %f and %f is %f.", a, b, sum),
			},
		},
	}, nil
}

func handleSendNotification(
	ctx context.Context,
	request mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {

	server := server.ServerFromContext(ctx)

	err := server.SendNotificationToClient(
		ctx,
		"notifications/progress",
		map[string]interface{}{
			"progress":      10,
			"total":         10,
			"progressToken": 0,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to send notification: %w", err)
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{
				Type: "text",
				Text: "notification sent successfully",
			},
		},
	}, nil
}

func handleLongRunningOperationTool(
	ctx context.Context,
	request mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	arguments := request.Params.Arguments
	progressToken := request.Params.Meta.ProgressToken
	duration, _ := arguments["duration"].(float64)
	steps, _ := arguments["steps"].(float64)
	stepDuration := duration / steps
	server := server.ServerFromContext(ctx)

	for i := 1; i < int(steps)+1; i++ {
		time.Sleep(time.Duration(stepDuration * float64(time.Second)))
		if progressToken != nil {
			server.SendNotificationToClient(
				ctx,
				"notifications/progress",
				map[string]interface{}{
					"progress":      i,
					"total":         int(steps),
					"progressToken": progressToken,
				},
			)
		}
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{
				Type: "text",
				Text: fmt.Sprintf(
					"Long running operation completed. Duration: %f seconds, Steps: %d.",
					duration,
					int(steps),
				),
			},
		},
	}, nil
}

// func (s *MCPServer) handleSampleLLMTool(arguments map[string]interface{}) (*mcp.CallToolResult, error) {
// 	prompt, _ := arguments["prompt"].(string)
// 	maxTokens, _ := arguments["maxTokens"].(float64)

// 	// This is a mock implementation. In a real scenario, you would use the server's RequestSampling method.
// 	result := fmt.Sprintf(
// 		"Sample LLM result for prompt: '%s' (max tokens: %d)",
// 		prompt,
// 		int(maxTokens),
// 	)

// 	return &mcp.CallToolResult{
// 		Content: []interface{}{
// 			mcp.TextContent{
// 				Type: "text",
// 				Text: fmt.Sprintf("LLM sampling result: %s", result),
// 			},
// 		},
// 	}, nil
// }

func handleGetTinyImageTool(
	ctx context.Context,
	request mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{
				Type: "text",
				Text: "This is a tiny image:",
			},
			mcp.ImageContent{
				Type:     "image",
				Data:     MCP_TINY_IMAGE,
				MIMEType: "image/png",
			},
			mcp.TextContent{
				Type: "text",
				Text: "The image above is the MCP tiny image.",
			},
		},
	}, nil
}

func handleNotification(
	ctx context.Context,
	notification mcp.JSONRPCNotification,
) {
	log.Printf("Received notification: %s", notification.Method)
}

func main() {
	var transport string
	flag.StringVar(&transport, "t", "stdio", "Transport type (stdio or sse)")
	flag.StringVar(&transport, "transport", "stdio", "Transport type (stdio or sse)")
	flag.Parse()

	mcpServer := NewMCPServer()

	// Only check for "sse" since stdio is the default
	if transport == "sse" {
		sseServer := server.NewSSEServer(mcpServer, server.WithBaseURL("http://localhost:8080"))
		log.Printf("SSE server listening on :8080")
		if err := sseServer.Start(":8080"); err != nil {
			log.Fatalf("Server error: %v", err)
		}
	} else {
		if err := server.ServeStdio(mcpServer); err != nil {
			log.Fatalf("Server error: %v", err)
		}
	}
}
