package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"

	"github.com/sammcj/go-a2a/server"
)

// MCPToolConfig represents the configuration for an MCP tool.
type MCPToolConfig struct {
	Name    string                 `json:"name"`
	Enabled bool                   `json:"enabled"`
	Config  map[string]interface{} `json:"config"`
}

// MCPConfig represents the configuration for MCP tools.
type MCPConfig struct {
	Tools []MCPToolConfig `json:"tools"`
}

// CustomMCPClient implements the server.MCPClient interface.
type CustomMCPClient struct {
	tools map[string]MCPToolConfig
}

// NewCustomMCPClient creates a new CustomMCPClient.
func NewCustomMCPClient(config MCPConfig) *CustomMCPClient {
	tools := make(map[string]MCPToolConfig)
	for _, tool := range config.Tools {
		if tool.Enabled {
			tools[tool.Name] = tool
		}
	}
	return &CustomMCPClient{
		tools: tools,
	}
}

// CallTool calls an MCP tool with the given name and parameters.
func (c *CustomMCPClient) CallTool(ctx context.Context, toolName string, params map[string]interface{}) (interface{}, error) {
	// Check if the tool is enabled
	tool, ok := c.tools[toolName]
	if !ok {
		return nil, fmt.Errorf("tool %q not found or not enabled", toolName)
	}

	// Get the command and args from the tool config
	command, ok := tool.Config["command"].(string)
	if !ok {
		return nil, fmt.Errorf("command not found in tool config")
	}

	args, ok := tool.Config["args"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("args not found in tool config")
	}

	// Convert args to []string
	argsStr := make([]string, len(args))
	for i, arg := range args {
		argsStr[i] = arg.(string)
	}

	// For fetch tool
	if toolName == "fetch" {
		return c.callFetchTool(ctx, command, argsStr, params)
	}

	// For brave-search tool
	if toolName == "brave-search" {
		return c.callBraveSearchTool(ctx, command, argsStr, params)
	}

	return nil, fmt.Errorf("unsupported tool: %s", toolName)
}

// callFetchTool calls the fetch MCP tool.
func (c *CustomMCPClient) callFetchTool(ctx context.Context, command string, args []string, params map[string]interface{}) (interface{}, error) {
	// Prepare the fetch tool parameters
	fetchParams := map[string]interface{}{
		"url": params["url"],
	}

	// Add optional parameters if provided
	if maxLength, ok := params["max_length"].(float64); ok {
		fetchParams["max_length"] = maxLength
	}
	if startIndex, ok := params["start_index"].(float64); ok {
		fetchParams["start_index"] = startIndex
	}
	if raw, ok := params["raw"].(bool); ok {
		fetchParams["raw"] = raw
	}

	// Convert parameters to JSON
	paramsJSON, err := json.Marshal(fetchParams)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal parameters: %w", err)
	}

	// Create the command
	cmd := exec.CommandContext(ctx, command, append(args, string(paramsJSON))...)

	// Run the command and get the output
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to execute fetch tool: %w, output: %s", err, output)
	}

	// Parse the output as JSON
	var result interface{}
	if err := json.Unmarshal(output, &result); err != nil {
		return nil, fmt.Errorf("failed to parse fetch tool output: %w", err)
	}

	return result, nil
}

// callBraveSearchTool calls the brave-search MCP tool.
func (c *CustomMCPClient) callBraveSearchTool(ctx context.Context, command string, args []string, params map[string]interface{}) (interface{}, error) {
	// Determine which brave search function to call
	var functionName string
	if _, ok := params["query"]; ok {
		functionName = "brave_web_search"
	} else {
		return nil, fmt.Errorf("unsupported brave search function")
	}

	// Prepare the brave search parameters
	braveParams := map[string]interface{}{
		"query": params["query"],
	}

	// Add optional parameters if provided
	if count, ok := params["count"].(float64); ok {
		braveParams["count"] = count
	}
	if offset, ok := params["offset"].(float64); ok {
		braveParams["offset"] = offset
	}

	// Convert parameters to JSON
	paramsJSON, err := json.Marshal(braveParams)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal parameters: %w", err)
	}

	// Create the command
	cmd := exec.CommandContext(ctx, command, append(args, functionName, string(paramsJSON))...)

	// Run the command and get the output
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to execute brave search tool: %w, output: %s", err, output)
	}

	// Parse the output as JSON
	var result interface{}
	if err := json.Unmarshal(output, &result); err != nil {
		return nil, fmt.Errorf("failed to parse brave search tool output: %w", err)
	}

	return result, nil
}

// ReadResource reads an MCP resource with the given URI.
func (c *CustomMCPClient) ReadResource(ctx context.Context, uri string) (string, string, error) {
	// This is a simplified implementation that doesn't actually read resources.
	// In a real implementation, this would make an actual call to an MCP server.
	return fmt.Sprintf("Resource content for URI: %s", uri), "text/plain", nil
}

// GetAvailableTools returns a list of available tools from the MCP server.
func (c *CustomMCPClient) GetAvailableTools(ctx context.Context) ([]server.MCPToolInfo, error) {
	tools := []server.MCPToolInfo{}

	// Add fetch tool if enabled
	if _, ok := c.tools["fetch"]; ok {
		tools = append(tools, server.MCPToolInfo{
			Name:        "fetch",
			Description: "Fetch a URL from the internet and optionally extract its contents as markdown",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"url": map[string]interface{}{
						"description": "URL to fetch",
						"format":      "uri",
						"minLength":   1,
						"title":       "Url",
						"type":        "string",
					},
					"max_length": map[string]interface{}{
						"default":          5000,
						"description":      "Maximum number of characters to return.",
						"exclusiveMaximum": 1000000,
						"exclusiveMinimum": 0,
						"title":            "Max Length",
						"type":             "integer",
					},
					"start_index": map[string]interface{}{
						"default":     0,
						"description": "On return output starting at this character index, useful if a previous fetch was truncated and more context is required.",
						"minimum":     0,
						"title":       "Start Index",
						"type":        "integer",
					},
					"raw": map[string]interface{}{
						"default":     false,
						"description": "Get the actual HTML content of the requested page, without simplification.",
						"title":       "Raw",
						"type":        "boolean",
					},
				},
				"description": "Parameters for fetching a URL.",
				"required":    []string{"url"},
				"title":       "Fetch",
			},
		})
	}

	// Add brave-search tool if enabled
	if _, ok := c.tools["brave-search"]; ok {
		tools = append(tools, server.MCPToolInfo{
			Name:        "brave_web_search",
			Description: "Performs a web search using the Brave Search API, ideal for general queries, news, articles, and online content",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"query": map[string]interface{}{
						"type":        "string",
						"description": "Search query (max 400 chars, 50 words)",
					},
					"count": map[string]interface{}{
						"type":        "number",
						"description": "Number of results (1-20, default 10)",
						"default":     10,
					},
					"offset": map[string]interface{}{
						"type":        "number",
						"description": "Pagination offset (max 9, default 0)",
						"default":     0,
					},
				},
				"required": []string{"query"},
			},
		})
	}

	return tools, nil
}

// GetAvailableResources returns a list of available resources from the MCP server.
func (c *CustomMCPClient) GetAvailableResources(ctx context.Context) ([]server.MCPResourceInfo, error) {
	// This is a simplified implementation that doesn't actually return resources.
	// In a real implementation, this would make an actual call to an MCP server.
	return []server.MCPResourceInfo{}, nil
}
