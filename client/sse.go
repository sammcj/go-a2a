package client

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/sammcj/go-a2a/a2a"
)

// SSEEvent represents an event received from an SSE stream.
type SSEEvent struct {
	ID    string
	Event string
	Data  string
}

// TaskUpdate represents an update to a task (either status or artifact).
type TaskUpdate struct {
	Type     string // "status" or "artifact"
	Status   *a2a.TaskStatus
	Artifact *a2a.Artifact
}

// SSEClient handles Server-Sent Events (SSE) connections for A2A tasks.
type SSEClient struct {
	httpClient *http.Client
	baseURL    string
	authHeaders map[string]string
}

// NewSSEClient creates a new SSE client.
func NewSSEClient(httpClient *http.Client, baseURL string, authHeaders map[string]string) *SSEClient {
	return &SSEClient{
		httpClient: httpClient,
		baseURL:    baseURL,
		authHeaders: authHeaders,
	}
}

// SubscribeToTask subscribes to task updates via SSE.
// It returns a channel for receiving task updates and an error channel.
func (c *SSEClient) SubscribeToTask(ctx context.Context, params *a2a.TaskSendParams) (<-chan TaskUpdate, <-chan error) {
	updateChan := make(chan TaskUpdate)
	errChan := make(chan error, 1)

	// Create JSON-RPC request
	request := a2a.JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  "tasks/sendSubscribe",
		ID:      generateRequestID(),
	}

	// Marshal params
	paramsJSON, err := json.Marshal(params)
	if err != nil {
		errChan <- fmt.Errorf("failed to marshal params: %w", err)
		close(updateChan)
		close(errChan)
		return updateChan, errChan
	}
	request.Params = paramsJSON

	// Marshal request
	requestJSON, err := json.Marshal(request)
	if err != nil {
		errChan <- fmt.Errorf("failed to marshal request: %w", err)
		close(updateChan)
		close(errChan)
		return updateChan, errChan
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/sse", strings.NewReader(string(requestJSON)))
	if err != nil {
		errChan <- fmt.Errorf("failed to create request: %w", err)
		close(updateChan)
		close(errChan)
		return updateChan, errChan
	}

	// Add headers
	for name, value := range c.authHeaders {
		req.Header.Set(name, value)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("Connection", "keep-alive")

	// Send request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		errChan <- fmt.Errorf("failed to send request: %w", err)
		close(updateChan)
		close(errChan)
		return updateChan, errChan
	}

	// Check status code
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		errChan <- fmt.Errorf("unexpected status code: %d", resp.StatusCode)
		close(updateChan)
		close(errChan)
		return updateChan, errChan
	}

	// Start a goroutine to read the SSE stream
	go func() {
		defer resp.Body.Close()
		defer close(updateChan)
		defer close(errChan)

		scanner := bufio.NewScanner(resp.Body)
		var event SSEEvent

		for scanner.Scan() {
			line := scanner.Text()

			// Skip empty lines
			if line == "" {
				// End of event, process it
				if event.Event != "" && event.Data != "" {
					c.processEvent(event, updateChan, errChan)
					event = SSEEvent{} // Reset event
				}
				continue
			}

			// Parse the line
			if strings.HasPrefix(line, "id:") {
				event.ID = strings.TrimSpace(line[3:])
			} else if strings.HasPrefix(line, "event:") {
				event.Event = strings.TrimSpace(line[6:])
			} else if strings.HasPrefix(line, "data:") {
				event.Data = strings.TrimSpace(line[5:])
			}
		}

		if err := scanner.Err(); err != nil {
			errChan <- fmt.Errorf("error reading SSE stream: %w", err)
		}
	}()

	return updateChan, errChan
}

// ResubscribeToTask resubscribes to task updates via SSE.
// It returns a channel for receiving task updates and an error channel.
func (c *SSEClient) ResubscribeToTask(ctx context.Context, taskID string, lastEventID string) (<-chan TaskUpdate, <-chan error) {
	updateChan := make(chan TaskUpdate)
	errChan := make(chan error, 1)

	// Create JSON-RPC request
	request := a2a.JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  "tasks/resubscribe",
		ID:      generateRequestID(),
	}

	// Create params
	params := a2a.TaskIdParams{
		TaskID: taskID,
	}

	// Marshal params
	paramsJSON, err := json.Marshal(params)
	if err != nil {
		errChan <- fmt.Errorf("failed to marshal params: %w", err)
		close(updateChan)
		close(errChan)
		return updateChan, errChan
	}
	request.Params = paramsJSON

	// Marshal request
	requestJSON, err := json.Marshal(request)
	if err != nil {
		errChan <- fmt.Errorf("failed to marshal request: %w", err)
		close(updateChan)
		close(errChan)
		return updateChan, errChan
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/sse", strings.NewReader(string(requestJSON)))
	if err != nil {
		errChan <- fmt.Errorf("failed to create request: %w", err)
		close(updateChan)
		close(errChan)
		return updateChan, errChan
	}

	// Add headers
	for name, value := range c.authHeaders {
		req.Header.Set(name, value)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("Connection", "keep-alive")

	// Add Last-Event-ID header if provided
	if lastEventID != "" {
		req.Header.Set("Last-Event-ID", lastEventID)
	}

	// Send request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		errChan <- fmt.Errorf("failed to send request: %w", err)
		close(updateChan)
		close(errChan)
		return updateChan, errChan
	}

	// Check status code
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		errChan <- fmt.Errorf("unexpected status code: %d", resp.StatusCode)
		close(updateChan)
		close(errChan)
		return updateChan, errChan
	}

	// Start a goroutine to read the SSE stream
	go func() {
		defer resp.Body.Close()
		defer close(updateChan)
		defer close(errChan)

		scanner := bufio.NewScanner(resp.Body)
		var event SSEEvent

		for scanner.Scan() {
			line := scanner.Text()

			// Skip empty lines
			if line == "" {
				// End of event, process it
				if event.Event != "" && event.Data != "" {
					c.processEvent(event, updateChan, errChan)
					event = SSEEvent{} // Reset event
				}
				continue
			}

			// Parse the line
			if strings.HasPrefix(line, "id:") {
				event.ID = strings.TrimSpace(line[3:])
			} else if strings.HasPrefix(line, "event:") {
				event.Event = strings.TrimSpace(line[6:])
			} else if strings.HasPrefix(line, "data:") {
				event.Data = strings.TrimSpace(line[5:])
			}
		}

		if err := scanner.Err(); err != nil {
			errChan <- fmt.Errorf("error reading SSE stream: %w", err)
		}
	}()

	return updateChan, errChan
}

// processEvent processes an SSE event and sends it to the appropriate channel.
func (c *SSEClient) processEvent(event SSEEvent, updateChan chan<- TaskUpdate, errChan chan<- error) {
	switch event.Event {
	case "taskStatusUpdate":
		var statusEvent a2a.TaskStatusUpdateEvent
		if err := json.Unmarshal([]byte(event.Data), &statusEvent); err != nil {
			errChan <- fmt.Errorf("failed to unmarshal status update: %w", err)
			return
		}
		updateChan <- TaskUpdate{
			Type:   "status",
			Status: &statusEvent.Status,
		}
	case "taskArtifactUpdate":
		var artifactEvent a2a.TaskArtifactUpdateEvent
		if err := json.Unmarshal([]byte(event.Data), &artifactEvent); err != nil {
			errChan <- fmt.Errorf("failed to unmarshal artifact update: %w", err)
			return
		}
		updateChan <- TaskUpdate{
			Type:     "artifact",
			Artifact: &artifactEvent.Artifact,
		}
	default:
		errChan <- fmt.Errorf("unknown event type: %s", event.Event)
	}
}
