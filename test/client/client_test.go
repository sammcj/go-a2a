package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sammcj/go-a2a/a2a"
)

func TestNewClient(t *testing.T) {
	validBaseURL := "http://localhost:8080/"
	client, err := NewClient(WithBaseURL(validBaseURL))
	if err != nil {
		t.Fatalf("NewClient with valid base URL failed: %v", err)
	}
	if client == nil {
		t.Fatalf("NewClient with valid base URL returned nil client")
	}
	if client.config.BaseURL != validBaseURL {
		t.Errorf("NewClient with valid base URL: expected base URL %s, got %s", validBaseURL, client.config.BaseURL)
	}

	invalidBaseURL := ":invalid:"
	client, err = NewClient(WithBaseURL(invalidBaseURL))
	if err == nil {
		t.Fatalf("NewClient with invalid base URL should have failed")
	}

	client, err = NewClient()
	if err == nil {
		t.Fatalf("NewClient with missing base URL should have failed")
	}

	noSlashBaseURL := "http://localhost:8080"
	client, err = NewClient(WithBaseURL(noSlashBaseURL))
	if err != nil {
		t.Fatalf("NewClient with base URL without trailing slash failed: %v", err)
	}
	if client == nil {
		t.Fatalf("NewClient with base URL without trailing slash returned nil client")
	}
	if client.config.BaseURL != noSlashBaseURL+"/" {
		t.Errorf("NewClient with base URL without trailing slash: expected base URL %s/, got %s", noSlashBaseURL, client.config.BaseURL)
	}

	customHTTPClient := &http.Client{}
	client, err = NewClient(WithBaseURL(validBaseURL), WithHTTPClient(customHTTPClient))
	if err != nil {
		t.Fatalf("NewClient with custom HTTP client failed: %v", err)
	}
	if client == nil {
		t.Fatalf("NewClient with custom HTTP client returned nil client")
	}
	if client.config.HTTPClient != customHTTPClient {
		t.Errorf("NewClient with custom HTTP client: expected custom HTTP client, got different client")
	}

	authHeaders := map[string]string{
		"Authorization": "Bearer testtoken",
	}
	client, err = NewClient(WithBaseURL(validBaseURL), WithAuthHeaders(authHeaders))
	if err != nil {
		t.Fatalf("NewClient with auth headers failed: %v", err)
	}
	if client == nil {
		t.Fatalf("NewClient with auth headers returned nil client")
	}
	if len(client.config.AuthHeaders) != len(authHeaders) {
		t.Errorf("NewClient with auth headers: expected %d auth headers, got %d", len(authHeaders), len(client.config.AuthHeaders))
	}
	for k, v := range authHeaders {
		if client.config.AuthHeaders[k] != v {
			t.Errorf("NewClient with auth headers: expected header %s with value %s, got value %s", k, v, client.config.AuthHeaders[k])
		}
	}
}

func TestSendTask(t *testing.T) {
	sampleTaskSendParams := &a2a.TaskSendParams{
		Message: a2a.Message{
			Role:  a2a.RoleUser,
			Parts: []a2a.Part{a2a.TextPart{Text: "test message"}},
		},
	}

	sampleResponse := a2a.JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      "client-request-1",
		Result: map[string]interface{}{
			"id":     "test-task",
			"status": map[string]interface{}{"state": a2a.TaskStateSubmitted},
		},
	}
	sampleResponseJSON, err := json.Marshal(sampleResponse)
	if err != nil {
		t.Fatalf("Failed to marshal sample response: %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST request, got %s", r.Method)
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(sampleResponseJSON)
	}))
	defer server.Close()

	client, err := NewClient(WithBaseURL(server.URL))
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	task, err := client.SendTask(context.Background(), sampleTaskSendParams)
	if err != nil {
		t.Fatalf("Failed to send task: %v", err)
	}

	if task.ID != "test-task" {
		t.Errorf("Expected task ID 'test-task', got %s", task.ID)
	}
}

func TestSetTaskPushNotification(t *testing.T) {
	sampleTaskID := "test-task"
	samplePushNotificationConfig := &a2a.TaskPushNotificationConfigParams{
		TaskID: sampleTaskID,
		URL:    "http://test.com/callback",
	}

	sampleResponse := a2a.JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      "client-request-1",
		Result: map[string]interface{}{
			"taskId": sampleTaskID,
			"url":    "http://test.com/callback",
		},
	}
	sampleResponseJSON, err := json.Marshal(sampleResponse)
	if err != nil {
		t.Fatalf("Failed to marshal sample response: %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST request, got %s", r.Method)
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(sampleResponseJSON)
	}))
	defer server.Close()

	client, err := NewClient(WithBaseURL(server.URL))
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	config, err := client.SetTaskPushNotification(context.Background(), samplePushNotificationConfig)
	if err != nil {
		t.Fatalf("Failed to set task push notification config: %v", err)
	}

	if config.TaskID != sampleTaskID {
		t.Errorf("Expected task ID '%s', got %s", sampleTaskID, config.TaskID)
	}
	if config.URL != "http://test.com/callback" {
		t.Errorf("Expected URL 'http://test.com/callback', got %s", config.URL)
	}
}
