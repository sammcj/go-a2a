package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"

	"github.com/sammcj/go-a2a/a2a"
)

// Client is an A2A client for interacting with A2A servers.
type Client struct {
	config Config
}

// NewClient creates a new A2A client.
func NewClient(opts ...Option) (*Client, error) {
	cfg := DefaultConfig()
	for _, opt := range opts {
		opt(&cfg)
	}

	// Validate configuration
	if cfg.BaseURL == "" {
		return nil, fmt.Errorf("base URL is required")
	}

	// Ensure base URL ends with a slash
	if !strings.HasSuffix(cfg.BaseURL, "/") {
		cfg.BaseURL += "/"
	}

	// Validate base URL format
	_, err := url.Parse(cfg.BaseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid base URL: %w", err)
	}

	return &Client{
		config: cfg,
	}, nil
}

// FetchAgentCard fetches the agent card from the server.
func (c *Client) FetchAgentCard(ctx context.Context) (*a2a.AgentCard, error) {
	// If we already have a cached agent card, return it
	if c.config.AgentCard != nil {
		return c.config.AgentCard, nil
	}

	// Construct the URL for the agent card
	baseURL, err := url.Parse(c.config.BaseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid base URL: %w", err)
	}

	// Default path for agent card
	cardURL := *baseURL
	cardURL.Path = path.Join(cardURL.Path, ".well-known/agent.json")

	// Create request
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, cardURL.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add headers
	for name, value := range c.config.AuthHeaders {
		req.Header.Set(name, value)
	}
	req.Header.Set("Accept", "application/json")

	// Send request
	resp, err := c.config.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch agent card: %w", err)
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch agent card: status code %d", resp.StatusCode)
	}

	// Parse response
	var card a2a.AgentCard
	if err := json.NewDecoder(resp.Body).Decode(&card); err != nil {
		return nil, fmt.Errorf("failed to parse agent card: %w", err)
	}

	// Cache the agent card
	c.config.AgentCard = &card

	return &card, nil
}

// SendTask sends a task to the A2A server.
func (c *Client) SendTask(ctx context.Context, params *a2a.TaskSendParams) (*a2a.Task, error) {
	// Create JSON-RPC request
	request := a2a.JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  "tasks/send",
		ID:      generateRequestID(),
	}

	// Marshal params
	paramsJSON, err := json.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal params: %w", err)
	}
	request.Params = paramsJSON

	// Send request
	var task a2a.Task
	if err := c.sendJSONRPCRequest(ctx, request, &task); err != nil {
		return nil, err
	}

	return &task, nil
}

// GetTask retrieves a task from the A2A server.
func (c *Client) GetTask(ctx context.Context, taskID string) (*a2a.Task, error) {
	// Create params
	params := a2a.TaskQueryParams{
		TaskID: taskID,
	}

	// Create JSON-RPC request
	request := a2a.JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  "tasks/get",
		ID:      generateRequestID(),
	}

	// Marshal params
	paramsJSON, err := json.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal params: %w", err)
	}
	request.Params = paramsJSON

	// Send request
	var task a2a.Task
	if err := c.sendJSONRPCRequest(ctx, request, &task); err != nil {
		return nil, err
	}

	return &task, nil
}

// CancelTask cancels a task on the A2A server.
func (c *Client) CancelTask(ctx context.Context, taskID string) (*a2a.Task, error) {
	// Create params
	params := a2a.TaskIdParams{
		TaskID: taskID,
	}

	// Create JSON-RPC request
	request := a2a.JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  "tasks/cancel",
		ID:      generateRequestID(),
	}

	// Marshal params
	paramsJSON, err := json.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal params: %w", err)
	}
	request.Params = paramsJSON

	// Send request
	var task a2a.Task
	if err := c.sendJSONRPCRequest(ctx, request, &task); err != nil {
		return nil, err
	}

	return &task, nil
}

// SetTaskPushNotification sets the push notification configuration for a task.
func (c *Client) SetTaskPushNotification(ctx context.Context, params *a2a.TaskPushNotificationConfigParams) (*a2a.PushNotificationConfig, error) {
	// Create JSON-RPC request
	request := a2a.JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  "tasks/pushNotification/set",
		ID:      generateRequestID(),
	}

	// Marshal params
	paramsJSON, err := json.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal params: %w", err)
	}
	request.Params = paramsJSON

	// Send request
	var config a2a.PushNotificationConfig
	if err := c.sendJSONRPCRequest(ctx, request, &config); err != nil {
		return nil, err
	}

	return &config, nil
}

// GetTaskPushNotification gets the push notification configuration for a task.
func (c *Client) GetTaskPushNotification(ctx context.Context, taskID string) (*a2a.PushNotificationConfig, error) {
	// Create params
	params := a2a.TaskIdParams{
		TaskID: taskID,
	}

	// Create JSON-RPC request
	request := a2a.JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  "tasks/pushNotification/get",
		ID:      generateRequestID(),
	}

	// Marshal params
	paramsJSON, err := json.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal params: %w", err)
	}
	request.Params = paramsJSON

	// Send request
	var config a2a.PushNotificationConfig
	if err := c.sendJSONRPCRequest(ctx, request, &config); err != nil {
		return nil, err
	}

	return &config, nil
}

// sendJSONRPCRequest sends a JSON-RPC request to the A2A server and unmarshals the result.
func (c *Client) sendJSONRPCRequest(ctx context.Context, request a2a.JSONRPCRequest, result interface{}) error {
	// Marshal request
	requestJSON, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.config.BaseURL, bytes.NewReader(requestJSON))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Add headers
	for name, value := range c.config.AuthHeaders {
		req.Header.Set(name, value)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	// Send request
	resp, err := c.config.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	// Parse JSON-RPC response
	var jsonRPCResponse a2a.JSONRPCResponse
	if err := json.Unmarshal(body, &jsonRPCResponse); err != nil {
		return fmt.Errorf("failed to parse JSON-RPC response: %w", err)
	}

	// Check for JSON-RPC error
	if jsonRPCResponse.Error != nil {
		return fmt.Errorf("JSON-RPC error: code=%d, message=%s", jsonRPCResponse.Error.Code, jsonRPCResponse.Error.Message)
	}

	// Unmarshal result
	resultJSON, err := json.Marshal(jsonRPCResponse.Result)
	if err != nil {
		return fmt.Errorf("failed to marshal result: %w", err)
	}

	if err := json.Unmarshal(resultJSON, result); err != nil {
		return fmt.Errorf("failed to unmarshal result: %w", err)
	}

	return nil
}

// generateRequestID generates a unique ID for a JSON-RPC request.
func generateRequestID() string {
	// For now, just use a simple string. In a real implementation, we might use a UUID.
	return "client-request-1"
}
