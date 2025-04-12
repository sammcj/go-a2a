package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/sammcj/go-a2a/a2a"
)

func TestPushNotifier_SendStatusUpdate(t *testing.T) {
	// Create a test server to receive push notifications
	var receivedPayload PushNotificationPayload
	var receivedHeaders http.Header

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Store the headers for later inspection
		receivedHeaders = r.Header.Clone()

		// Parse the request body
		decoder := json.NewDecoder(r.Body)
		err := decoder.Decode(&receivedPayload)
		if err != nil {
			t.Errorf("Failed to decode request body: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		// Return a success response
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Create a push notifier
	notifier := NewPushNotifier(5 * time.Second)

	// Create a task
	task := &a2a.Task{
		ID: "test-task-123",
		Status: a2a.TaskStatus{
			State:     a2a.TaskStateCompleted,
			Timestamp: time.Now(),
			Message: &a2a.Message{
				Role:      a2a.RoleAgent,
				Timestamp: time.Now(),
				Parts: []a2a.Part{
					a2a.TextPart{
						Type: "text",
						Text: "Task completed successfully",
					},
				},
			},
		},
		History: []a2a.Message{
			{
				Role:      a2a.RoleUser,
				Timestamp: time.Now(),
				Parts: []a2a.Part{
					a2a.TextPart{
						Type: "text",
						Text: "Hello",
					},
				},
			},
		},
	}

	// Create push notification config
	includeTaskData := true
	config := &a2a.PushNotificationConfig{
		TaskID:          task.ID,
		URL:             server.URL,
		IncludeTaskData: &includeTaskData,
		Authentication: &a2a.AuthenticationInfo{
			Type: "bearer",
			Configuration: map[string]interface{}{
				"token": "test-token",
			},
		},
	}

	// Send the notification
	err := notifier.SendStatusUpdate(context.Background(), task, config)
	if err != nil {
		t.Fatalf("Failed to send push notification: %v", err)
	}

	// Verify the received payload
	if receivedPayload.TaskID != task.ID {
		t.Errorf("Expected task ID %s, got %s", task.ID, receivedPayload.TaskID)
	}

	if receivedPayload.EventType != "status" {
		t.Errorf("Expected event type 'status', got %s", receivedPayload.EventType)
	}

	if receivedPayload.Status == nil {
		t.Error("Expected status to be present, got nil")
	} else if receivedPayload.Status.State != a2a.TaskStateCompleted {
		t.Errorf("Expected status state %s, got %s", a2a.TaskStateCompleted, receivedPayload.Status.State)
	}

	if receivedPayload.Task == nil {
		t.Error("Expected task to be included, got nil")
	}

	// Verify the authentication header
	authHeader := receivedHeaders.Get("Authorization")
	expectedAuthHeader := "Bearer test-token"
	if authHeader != expectedAuthHeader {
		t.Errorf("Expected Authorization header %q, got %q", expectedAuthHeader, authHeader)
	}
}

func TestPushNotifier_SendArtifactUpdate(t *testing.T) {
	// Create a test server to receive push notifications
	var receivedPayload PushNotificationPayload

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Parse the request body
		decoder := json.NewDecoder(r.Body)
		err := decoder.Decode(&receivedPayload)
		if err != nil {
			t.Errorf("Failed to decode request body: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		// Return a success response
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Create a push notifier
	notifier := NewPushNotifier(5 * time.Second)

	// Create a task
	task := &a2a.Task{
		ID: "test-task-456",
	}

	// Create an artifact
	artifact := a2a.Artifact{
		ID:        "artifact-123",
		TaskID:    task.ID,
		Timestamp: time.Now(),
		Part: a2a.TextPart{
			Type: "text",
			Text: "This is an artifact",
		},
	}

	// Create push notification config with artifacts included
	includeArtifacts := true
	config := &a2a.PushNotificationConfig{
		TaskID:           task.ID,
		URL:              server.URL,
		IncludeArtifacts: &includeArtifacts,
	}

	// Send the notification
	err := notifier.SendArtifactUpdate(context.Background(), task, artifact, config)
	if err != nil {
		t.Fatalf("Failed to send push notification: %v", err)
	}

	// Verify the received payload
	if receivedPayload.TaskID != task.ID {
		t.Errorf("Expected task ID %s, got %s", task.ID, receivedPayload.TaskID)
	}

	if receivedPayload.EventType != "artifact" {
		t.Errorf("Expected event type 'artifact', got %s", receivedPayload.EventType)
	}

	if receivedPayload.Artifact == nil {
		t.Error("Expected artifact to be present, got nil")
	} else if receivedPayload.Artifact.ID != artifact.ID {
		t.Errorf("Expected artifact ID %s, got %s", artifact.ID, receivedPayload.Artifact.ID)
	}
}

func TestPushNotifier_SkipArtifactNotification(t *testing.T) {
	// Create a test server that fails if it receives any requests
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("Server should not have received a request")
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	// Create a push notifier
	notifier := NewPushNotifier(5 * time.Second)

	// Create a task
	task := &a2a.Task{
		ID: "test-task-789",
	}

	// Create an artifact
	artifact := a2a.Artifact{
		ID:        "artifact-456",
		TaskID:    task.ID,
		Timestamp: time.Now(),
		Part: a2a.TextPart{
			Type: "text",
			Text: "This is an artifact",
		},
	}

	// Create push notification config with artifacts excluded
	includeArtifacts := false
	config := &a2a.PushNotificationConfig{
		TaskID:           task.ID,
		URL:              server.URL,
		IncludeArtifacts: &includeArtifacts,
	}

	// Send the notification - this should be a no-op
	err := notifier.SendArtifactUpdate(context.Background(), task, artifact, config)
	if err != nil {
		t.Fatalf("Failed to send push notification: %v", err)
	}

	// No assertions needed - the test server will fail if it receives a request
}
