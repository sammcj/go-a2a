// Package server provides the server-side implementation of the A2A protocol.
package server

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/sammcj/go-a2a/a2a"
	"github.com/sammcj/go-a2a/llm"
	"github.com/sammcj/go-a2a/pkg/task"
)

// MCPClient defines the interface for interacting with MCP servers.
// This interface allows the A2A agent to use MCP tools and resources
// without directly depending on any specific MCP client implementation.
type MCPClient interface {
	// CallTool calls an MCP tool with the given name and parameters.
	CallTool(ctx context.Context, toolName string, params map[string]interface{}) (interface{}, error)

	// ReadResource reads an MCP resource with the given URI.
	ReadResource(ctx context.Context, uri string) (string, string, error)

	// GetAvailableTools returns a list of available tools from the MCP server.
	GetAvailableTools(ctx context.Context) ([]MCPToolInfo, error)

	// GetAvailableResources returns a list of available resources from the MCP server.
	GetAvailableResources(ctx context.Context) ([]MCPResourceInfo, error)
}

// MCPToolInfo represents information about an MCP tool.
type MCPToolInfo struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"inputSchema"`
}

// MCPResourceInfo represents information about an MCP resource.
type MCPResourceInfo struct {
	URI         string `json:"uri"`
	Name        string `json:"name"`
	Description string `json:"description"`
	MIMEType    string `json:"mimeType"`
}

// MCPToolAdapter adapts the Tool interface to work with MCP tools.
// It implements the Tool interface and delegates calls to an MCPClient.
type MCPToolAdapter struct {
	client    MCPClient
	toolName  string
	toolInfo  MCPToolInfo
	converter ToolParamConverter
}

// ToolParamConverter is a function that converts A2A tool parameters to MCP tool parameters.
type ToolParamConverter func(params map[string]interface{}) (map[string]interface{}, error)

// NewMCPToolAdapter creates a new MCPToolAdapter.
func NewMCPToolAdapter(client MCPClient, toolName string, converter ToolParamConverter) (*MCPToolAdapter, error) {
	// Get tool info from MCP server
	tools, err := client.GetAvailableTools(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to get available tools: %w", err)
	}

	// Find the tool with the given name
	var toolInfo MCPToolInfo
	found := false
	for _, tool := range tools {
		if tool.Name == toolName {
			toolInfo = tool
			found = true
			break
		}
	}

	if !found {
		return nil, fmt.Errorf("tool %q not found", toolName)
	}

	// If no converter is provided, use the default one
	if converter == nil {
		converter = DefaultToolParamConverter
	}

	return &MCPToolAdapter{
		client:    client,
		toolName:  toolName,
		toolInfo:  toolInfo,
		converter: converter,
	}, nil
}

// Name returns the name of the tool.
func (a *MCPToolAdapter) Name() string {
	return a.toolName
}

// Description returns a description of the tool.
func (a *MCPToolAdapter) Description() string {
	return a.toolInfo.Description
}

// Execute executes the tool with the given parameters.
func (a *MCPToolAdapter) Execute(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	// Convert parameters if needed
	mcpParams, err := a.converter(params)
	if err != nil {
		return nil, fmt.Errorf("failed to convert parameters: %w", err)
	}

	// Call the MCP tool
	result, err := a.client.CallTool(ctx, a.toolName, mcpParams)
	if err != nil {
		return nil, fmt.Errorf("failed to call MCP tool: %w", err)
	}

	return result, nil
}

// DefaultToolParamConverter is the default implementation of ToolParamConverter.
// It simply passes through the parameters without any conversion.
func DefaultToolParamConverter(params map[string]interface{}) (map[string]interface{}, error) {
	return params, nil
}

// MCPResourceAdapter adapts an MCP resource to be used as a tool.
// It implements the Tool interface and delegates calls to an MCPClient.
type MCPResourceAdapter struct {
	client       MCPClient
	resourceURI  string
	resourceInfo MCPResourceInfo
	uriTemplate  string
	uriParams    []string
}

// NewMCPResourceAdapter creates a new MCPResourceAdapter.
func NewMCPResourceAdapter(client MCPClient, resourceURI string) (*MCPResourceAdapter, error) {
	// Get resource info from MCP server
	resources, err := client.GetAvailableResources(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to get available resources: %w", err)
	}

	// Find the resource with the given URI
	var resourceInfo MCPResourceInfo
	found := false
	for _, resource := range resources {
		if resource.URI == resourceURI {
			resourceInfo = resource
			found = true
			break
		}
	}

	if !found {
		return nil, fmt.Errorf("resource %q not found", resourceURI)
	}

	// Parse URI template and extract parameters
	uriTemplate, uriParams := parseURITemplate(resourceURI)

	return &MCPResourceAdapter{
		client:       client,
		resourceURI:  resourceURI,
		resourceInfo: resourceInfo,
		uriTemplate:  uriTemplate,
		uriParams:    uriParams,
	}, nil
}

// Name returns the name of the tool.
func (a *MCPResourceAdapter) Name() string {
	return "read_resource_" + a.resourceInfo.Name
}

// Description returns a description of the tool.
func (a *MCPResourceAdapter) Description() string {
	return fmt.Sprintf("Read the %q resource (%s)", a.resourceInfo.Name, a.resourceInfo.Description)
}

// Execute executes the tool with the given parameters.
func (a *MCPResourceAdapter) Execute(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	// Build the resource URI with the provided parameters
	uri := a.resourceURI
	if len(a.uriParams) > 0 {
		uri = buildURIFromTemplate(a.uriTemplate, a.uriParams, params)
	}

	// Read the resource
	content, mimeType, err := a.client.ReadResource(ctx, uri)
	if err != nil {
		return nil, fmt.Errorf("failed to read MCP resource: %w", err)
	}

	return map[string]interface{}{
		"content":  content,
		"mimeType": mimeType,
	}, nil
}

// parseURITemplate parses a URI template and extracts the parameters.
// For example, "users/{id}/profile" would return "users/{}/profile" and ["id"].
func parseURITemplate(uri string) (string, []string) {
	// This is a simplified implementation that assumes parameters are enclosed in curly braces.
	// A more robust implementation would use a proper URI template parser.
	// For now, this is sufficient for our needs.

	// TODO: Implement a proper URI template parser
	return uri, nil
}

// buildURIFromTemplate builds a URI from a template and parameters.
// For example, "users/{}/profile", ["id"], {"id": "123"} would return "users/123/profile".
func buildURIFromTemplate(template string, params []string, values map[string]interface{}) string {
	// This is a simplified implementation that assumes parameters are enclosed in curly braces.
	// A more robust implementation would use a proper URI template parser.
	// For now, this is sufficient for our needs.

	// TODO: Implement a proper URI template builder
	return template
}

// MCPClientConfig contains configuration options for an MCP client.
type MCPClientConfig struct {
	// ServerURL is the URL of the MCP server.
	ServerURL string

	// AuthToken is the authentication token to use when connecting to the MCP server.
	AuthToken string

	// Timeout is the timeout for MCP requests.
	Timeout int
}

// NewMCPClient creates a new MCP client based on the provided configuration.
// This function should be implemented by the application developer to create
// an MCP client that implements the MCPClient interface.
func NewMCPClient(config MCPClientConfig) (MCPClient, error) {
	// This is a placeholder function that should be implemented by the application developer.
	// It should create an MCP client that implements the MCPClient interface.
	// The implementation will depend on the specific MCP client library being used.
	return nil, fmt.Errorf("NewMCPClient is not implemented")
}

// MCPToolAugmentedAgent implements AgentEngine using an LLM with MCP tools.
type MCPToolAugmentedAgent struct {
	llm          llm.LLMInterface
	mcpClient    MCPClient
	systemPrompt string
	capabilities AgentCapabilities
}

// NewMCPToolAugmentedAgent creates a new MCPToolAugmentedAgent.
func NewMCPToolAugmentedAgent(llmInterface llm.LLMInterface, mcpClient MCPClient) (*MCPToolAugmentedAgent, error) {
	// Get model info to determine capabilities
	modelInfo := llmInterface.GetModelInfo()

	// Get available tools from MCP server
	tools, err := mcpClient.GetAvailableTools(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to get available tools: %w", err)
	}

	// Create a system prompt that includes tool descriptions
	systemPrompt := "You are a helpful assistant with access to the following tools from an MCP server:\n\n"
	for _, tool := range tools {
		systemPrompt += fmt.Sprintf("- %s: %s\n", tool.Name, tool.Description)
	}
	systemPrompt += "\nWhen you need to use a tool, specify the tool name and parameters in your response in the following JSON format:\n"
	systemPrompt += "```json\n{\"tool\": \"tool_name\", \"params\": {\"param1\": \"value1\", \"param2\": \"value2\"}}\n```\n"
	systemPrompt += "I will execute the tool and return the result to you."

	return &MCPToolAugmentedAgent{
		llm:          llmInterface,
		mcpClient:    mcpClient,
		systemPrompt: systemPrompt,
		capabilities: AgentCapabilities{
			SupportsStreaming:         true,
			SupportedInputModalities:  modelInfo.InputModalities,
			SupportedOutputModalities: modelInfo.OutputModalities,
		},
	}, nil
}

// ProcessTask implements AgentEngine.ProcessTask.
func (a *MCPToolAugmentedAgent) ProcessTask(ctx context.Context, taskCtx task.Context) (<-chan task.YieldUpdate, error) {
	updateChan := make(chan task.YieldUpdate)

	go func() {
		defer close(updateChan)

		// Extract the user's message
		userMessage := taskCtx.UserMessage
		var userText string
		for _, part := range userMessage.Parts {
			if textPart, ok := part.(a2a.TextPart); ok {
				userText = textPart.Text
				break
			}
		}

		// Send a working status update
		updateChan <- task.StatusUpdate{
			State: a2a.TaskStateWorking,
		}

		// Process the message with the LLM
		chunkChan, errChan := a.llm.GenerateStream(ctx, userText, llm.WithSystemPrompt(a.systemPrompt))

		// Buffer to accumulate the response
		var responseBuffer string
		var toolCall *ToolCall

		// Process the streaming response
		for {
			select {
			case chunk, ok := <-chunkChan:
				if !ok {
					// Channel closed, all chunks received
					break
				}

				// Accumulate the response
				responseBuffer += chunk.Text

				// Check if the response contains a tool call
				if toolCall == nil {
					toolCall = extractToolCall(responseBuffer)
				}

				// Send a working status update with the chunk
				responseMessage := a2a.Message{
					Role: a2a.RoleAgent,
					Parts: []a2a.Part{
						a2a.TextPart{
							Type: "text",
							Text: chunk.Text,
						},
					},
				}
				updateChan <- task.StatusUpdate{
					State:   a2a.TaskStateWorking,
					Message: &responseMessage,
				}

				// If this is the final chunk, process any tool calls
				if chunk.Completed && toolCall != nil {
					// Execute the tool
					result, err := a.mcpClient.CallTool(ctx, toolCall.Tool, toolCall.Params)
					if err != nil {
						// Send a failed status update
						errorMessage := a2a.Message{
							Role: a2a.RoleSystem,
							Parts: []a2a.Part{
								a2a.TextPart{
									Type: "text",
									Text: fmt.Sprintf("Failed to execute tool %q: %v", toolCall.Tool, err),
								},
							},
						}
						updateChan <- task.StatusUpdate{
							State:   a2a.TaskStateFailed,
							Message: &errorMessage,
						}
						return
					}

					// Convert the result to a string
					resultStr, err := formatToolResult(result)
					if err != nil {
						// Send a failed status update
						errorMessage := a2a.Message{
							Role: a2a.RoleSystem,
							Parts: []a2a.Part{
								a2a.TextPart{
									Type: "text",
									Text: fmt.Sprintf("Failed to format tool result: %v", err),
								},
							},
						}
						updateChan <- task.StatusUpdate{
							State:   a2a.TaskStateFailed,
							Message: &errorMessage,
						}
						return
					}

					// Send the tool result as an artifact update
					updateChan <- task.ArtifactUpdate{
						Part: a2a.TextPart{
							Type: "text",
							Text: resultStr,
						},
						Metadata: map[string]interface{}{
							"tool": toolCall.Tool,
						},
					}

					// Process the tool result with the LLM
					prompt := fmt.Sprintf("I executed the tool %q with the parameters %v and got the following result:\n\n%s\n\nPlease continue helping the user based on this result.", toolCall.Tool, toolCall.Params, resultStr)
					response, err := a.llm.Generate(ctx, prompt, llm.WithSystemPrompt(a.systemPrompt))
					if err != nil {
						// Send a failed status update
						errorMessage := a2a.Message{
							Role: a2a.RoleSystem,
							Parts: []a2a.Part{
								a2a.TextPart{
									Type: "text",
									Text: fmt.Sprintf("Failed to process tool result: %v", err),
								},
							},
						}
						updateChan <- task.StatusUpdate{
							State:   a2a.TaskStateFailed,
							Message: &errorMessage,
						}
						return
					}

					// Send the response
					responseMessage := a2a.Message{
						Role: a2a.RoleAgent,
						Parts: []a2a.Part{
							a2a.TextPart{
								Type: "text",
								Text: response,
							},
						},
					}
					updateChan <- task.StatusUpdate{
						State:   a2a.TaskStateWorking,
						Message: &responseMessage,
					}
				}

			case err := <-errChan:
				// Error occurred during generation
				errorMessage := a2a.Message{
					Role: a2a.RoleSystem,
					Parts: []a2a.Part{
						a2a.TextPart{
							Type: "text",
							Text: fmt.Sprintf("Failed to generate response: %v", err),
						},
					},
				}
				updateChan <- task.StatusUpdate{
					State:   a2a.TaskStateFailed,
					Message: &errorMessage,
				}
				return

			case <-ctx.Done():
				// Context cancelled
				errorMessage := a2a.Message{
					Role: a2a.RoleSystem,
					Parts: []a2a.Part{
						a2a.TextPart{
							Type: "text",
							Text: "Task cancelled",
						},
					},
				}
				updateChan <- task.StatusUpdate{
					State:   a2a.TaskStateCancelled,
					Message: &errorMessage,
				}
				return
			}
		}

		// Send a completed status update
		updateChan <- task.StatusUpdate{
			State: a2a.TaskStateCompleted,
		}
	}()

	return updateChan, nil
}

// GetCapabilities implements AgentEngine.GetCapabilities.
func (a *MCPToolAugmentedAgent) GetCapabilities() AgentCapabilities {
	return a.capabilities
}

// ToolCall represents a tool call extracted from an LLM response.
type ToolCall struct {
	Tool   string                 `json:"tool"`
	Params map[string]interface{} `json:"params"`
}

// extractToolCall extracts a tool call from an LLM response.
// It looks for JSON objects with "tool" and "params" fields.
func extractToolCall(response string) *ToolCall {
	// Look for JSON objects in the response
	// This is a simplified implementation that assumes the JSON object is well-formed.
	// A more robust implementation would use a proper JSON parser.

	// Find the start and end of a JSON object
	start := -1
	end := -1
	braceCount := 0
	inString := false
	escapeNext := false

	for i, c := range response {
		if escapeNext {
			escapeNext = false
			continue
		}

		if c == '\\' {
			escapeNext = true
			continue
		}

		if c == '"' && !escapeNext {
			inString = !inString
			continue
		}

		if !inString {
			if c == '{' {
				if braceCount == 0 {
					start = i
				}
				braceCount++
			} else if c == '}' {
				braceCount--
				if braceCount == 0 && start != -1 {
					end = i + 1
					break
				}
			}
		}
	}

	if start == -1 || end == -1 {
		return nil
	}

	// Extract the JSON object
	jsonStr := response[start:end]

	// Parse the JSON object
	var obj map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &obj); err != nil {
		return nil
	}

	// Check if it's a tool call
	tool, ok := obj["tool"].(string)
	if !ok {
		return nil
	}

	params, ok := obj["params"].(map[string]interface{})
	if !ok {
		return nil
	}

	return &ToolCall{
		Tool:   tool,
		Params: params,
	}
}

// formatToolResult formats a tool result as a string.
func formatToolResult(result interface{}) (string, error) {
	// Convert the result to JSON
	resultJSON, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal result: %w", err)
	}

	return string(resultJSON), nil
}
