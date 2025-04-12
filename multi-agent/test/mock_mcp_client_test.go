package test

import (
	"context"
	"testing"

	"github.com/sammcj/go-a2a/server"
)

// MockMCPClient is a mock implementation of the server.MCPClient interface for testing.
type MockMCPClient struct{}

// CallTool mocks calling an MCP tool.
func (m *MockMCPClient) CallTool(ctx context.Context, toolName string, params map[string]interface{}) (interface{}, error) {
	// Mock responses for different tools
	switch toolName {
	case "fetch":
		return map[string]interface{}{
			"content": "This is mock content from a web page",
		}, nil
	case "brave_web_search":
		return map[string]interface{}{
			"results": []map[string]interface{}{
				{
					"title":       "Mock Search Result 1",
					"description": "This is a mock search result",
					"url":         "https://example.com/1",
				},
				{
					"title":       "Mock Search Result 2",
					"description": "This is another mock search result",
					"url":         "https://example.com/2",
				},
			},
		}, nil
	default:
		return nil, nil
	}
}

// ReadResource mocks reading an MCP resource.
func (m *MockMCPClient) ReadResource(ctx context.Context, uri string) (string, string, error) {
	return "Mock resource content", "text/plain", nil
}

// GetAvailableTools mocks getting available MCP tools.
func (m *MockMCPClient) GetAvailableTools(ctx context.Context) ([]server.MCPToolInfo, error) {
	return []server.MCPToolInfo{
		{
			Name:        "fetch",
			Description: "Fetch a URL from the internet",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"url": map[string]interface{}{
						"type": "string",
					},
				},
				"required": []string{"url"},
			},
		},
		{
			Name:        "brave_web_search",
			Description: "Search the web using Brave Search",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"query": map[string]interface{}{
						"type": "string",
					},
				},
				"required": []string{"query"},
			},
		},
	}, nil
}

// GetAvailableResources mocks getting available MCP resources.
func (m *MockMCPClient) GetAvailableResources(ctx context.Context) ([]server.MCPResourceInfo, error) {
	return []server.MCPResourceInfo{}, nil
}

// TestMockMCPClient tests the mock MCP client.
func TestMockMCPClient(t *testing.T) {
	// Create a mock MCP client
	client := &MockMCPClient{}

	// Test CallTool with fetch
	result, err := client.CallTool(context.Background(), "fetch", map[string]interface{}{
		"url": "https://example.com",
	})
	if err != nil {
		t.Fatalf("Failed to call fetch tool: %v", err)
	}
	content, ok := result.(map[string]interface{})["content"].(string)
	if !ok {
		t.Fatalf("Expected content to be a string")
	}
	if content != "This is mock content from a web page" {
		t.Errorf("Expected content to be 'This is mock content from a web page', got '%s'", content)
	}

	// Test CallTool with brave_web_search
	result, err = client.CallTool(context.Background(), "brave_web_search", map[string]interface{}{
		"query": "test",
	})
	if err != nil {
		t.Fatalf("Failed to call brave_web_search tool: %v", err)
	}
	results, ok := result.(map[string]interface{})["results"].([]map[string]interface{})
	if !ok {
		t.Fatalf("Expected results to be an array of maps")
	}
	if len(results) != 2 {
		t.Errorf("Expected 2 results, got %d", len(results))
	}
	if results[0]["title"] != "Mock Search Result 1" {
		t.Errorf("Expected first result title to be 'Mock Search Result 1', got '%s'", results[0]["title"])
	}

	// Test ReadResource
	content, contentType, err := client.ReadResource(context.Background(), "test://resource")
	if err != nil {
		t.Fatalf("Failed to read resource: %v", err)
	}
	if content != "Mock resource content" {
		t.Errorf("Expected resource content to be 'Mock resource content', got '%s'", content)
	}
	if contentType != "text/plain" {
		t.Errorf("Expected content type to be 'text/plain', got '%s'", contentType)
	}

	// Test GetAvailableTools
	tools, err := client.GetAvailableTools(context.Background())
	if err != nil {
		t.Fatalf("Failed to get available tools: %v", err)
	}
	if len(tools) != 2 {
		t.Errorf("Expected 2 tools, got %d", len(tools))
	}
	if tools[0].Name != "fetch" {
		t.Errorf("Expected first tool name to be 'fetch', got '%s'", tools[0].Name)
	}
	if tools[1].Name != "brave_web_search" {
		t.Errorf("Expected second tool name to be 'brave_web_search', got '%s'", tools[1].Name)
	}

	// Test GetAvailableResources
	resources, err := client.GetAvailableResources(context.Background())
	if err != nil {
		t.Fatalf("Failed to get available resources: %v", err)
	}
	if len(resources) != 0 {
		t.Errorf("Expected 0 resources, got %d", len(resources))
	}
}
