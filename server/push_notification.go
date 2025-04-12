package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/sammcj/go-a2a/a2a"
)

// PushNotifier handles sending push notifications for task updates.
type PushNotifier struct {
	httpClient *http.Client
}

// NewPushNotifier creates a new PushNotifier.
func NewPushNotifier(timeout time.Duration) *PushNotifier {
	if timeout == 0 {
		timeout = 10 * time.Second // Default timeout
	}

	return &PushNotifier{
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

// PushNotificationPayload represents the payload sent in a push notification.
type PushNotificationPayload struct {
	TaskID    string          `json:"taskId"`
	EventType string          `json:"eventType"` // "status" or "artifact"
	Status    *a2a.TaskStatus `json:"status,omitempty"`
	Artifact  *a2a.Artifact   `json:"artifact,omitempty"`
	Task      *a2a.Task       `json:"task,omitempty"` // Full task data, included if IncludeTaskData is true
}

// SendStatusUpdate sends a push notification for a task status update.
func (p *PushNotifier) SendStatusUpdate(ctx context.Context, task *a2a.Task, config *a2a.PushNotificationConfig) error {
	if config == nil || config.URL == "" {
		return nil // No push notification configured
	}

	// Create payload
	payload := PushNotificationPayload{
		TaskID:    task.ID,
		EventType: "status",
		Status:    &task.Status,
	}

	// Include full task data if requested
	if config.IncludeTaskData != nil && *config.IncludeTaskData {
		payload.Task = task
	}

	// Send notification
	return p.sendNotification(ctx, config, payload)
}

// SendArtifactUpdate sends a push notification for a task artifact update.
func (p *PushNotifier) SendArtifactUpdate(ctx context.Context, task *a2a.Task, artifact a2a.Artifact, config *a2a.PushNotificationConfig) error {
	if config == nil || config.URL == "" {
		return nil // No push notification configured
	}

	// Skip if artifacts are not included in push notifications
	if config.IncludeArtifacts != nil && !*config.IncludeArtifacts {
		return nil
	}

	// Create payload
	payload := PushNotificationPayload{
		TaskID:    task.ID,
		EventType: "artifact",
		Artifact:  &artifact,
	}

	// Include full task data if requested
	if config.IncludeTaskData != nil && *config.IncludeTaskData {
		payload.Task = task
	}

	// Send notification
	return p.sendNotification(ctx, config, payload)
}

// sendNotification sends a push notification to the configured URL.
func (p *PushNotifier) sendNotification(ctx context.Context, config *a2a.PushNotificationConfig, payload interface{}) error {
	// Marshal payload to JSON
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal push notification payload: %w", err)
	}

	// Create request
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, config.URL, bytes.NewReader(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create push notification request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "go-a2a-push-notifier")

	// Add authentication if configured
	if config.Authentication != nil {
		if err := addAuthenticationToRequest(req, config.Authentication); err != nil {
			return fmt.Errorf("failed to add authentication to push notification request: %w", err)
		}
	}

	// Send request
	resp, err := p.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send push notification: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("push notification failed with status code %d", resp.StatusCode)
	}

	return nil
}

// addAuthenticationToRequest adds authentication information to a request.
func addAuthenticationToRequest(req *http.Request, auth *a2a.AuthenticationInfo) error {
	switch auth.Type {
	case "bearer":
		// Extract token from configuration
		config, ok := auth.Configuration.(map[string]interface{})
		if !ok {
			return fmt.Errorf("invalid bearer authentication configuration")
		}
		token, ok := config["token"].(string)
		if !ok {
			return fmt.Errorf("bearer token not found in authentication configuration")
		}
		req.Header.Set("Authorization", "Bearer "+token)
	case "header":
		// Extract header name and value from configuration
		config, ok := auth.Configuration.(map[string]interface{})
		if !ok {
			return fmt.Errorf("invalid header authentication configuration")
		}
		headerName, ok := config["headerName"].(string)
		if !ok {
			return fmt.Errorf("header name not found in authentication configuration")
		}
		headerValue, ok := config["value"].(string)
		if !ok {
			return fmt.Errorf("header value not found in authentication configuration")
		}
		req.Header.Set(headerName, headerValue)
	default:
		return fmt.Errorf("unsupported authentication type: %s", auth.Type)
	}
	return nil
}
