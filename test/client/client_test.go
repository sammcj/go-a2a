package client

import (
	"io"
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"github.com/sammcj/go-a2a/a2a"
)

func TestNewClient(t *testing.T) {
	// Test with valid base URL
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

	// Test with invalid base URL
	invalidBaseURL := ":invalid:"
	client, err = NewClient(WithBaseURL(invalidBaseURL))
	if err == nil {
		t.Fatalf("NewClient with invalid base URL should have failed")
	}

	// Test with missing base URL
	client, err = NewClient()
	if err == nil {
		t.Fatalf("NewClient with missing base URL should have failed")
	}

	// Test with base URL without trailing slash
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

	// Test with custom HTTP client
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
	
	// Test with auth headers
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

func TestFetchAgentCard(t *testing.T) {
	// Define a sample agent card
	sampleAgentCard := &a2a.AgentCard{
		A2AVersion: "1.0",
		ID:         "test-agent",
		Name:       "Test Agent",
	}

	// Marshal the agent card to JSON
	sampleAgentCardJSON, err := json.Marshal(sampleAgentCard)
	if err != nil {
		t.Fatalf("Failed to marshal sample agent card: %v", err)
	}

	// Create a mock HTTP server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check for correct path
		if r.URL.Path != "/.well-known/agent.json" {
			t.Errorf("Expected request to /.well-known/agent.json, got %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		// Check for correct method
		if r.Method != http.MethodGet {
			t.Errorf("Expected GET request, got %s", r.Method)
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		// Send the sample agent card
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(sampleAgentCardJSON)
	}))
	defer server.Close()

	// Create a client using the mock server's URL
	client, err := NewClient(WithBaseURL(server.URL))
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// Fetch the agent card
	fetchedCard, err := client.FetchAgentCard(context.Background())
	if err != nil {
		t.Fatalf("Failed to fetch agent card: %v", err)
	}

	// Compare the fetched card with the sample card
	if fetchedCard.ID != sampleAgentCard.ID {
		t.Errorf("Expected agent card ID %s, got %s", sampleAgentCard.ID, fetchedCard.ID)
	}
}

func TestSendTask(t *testing.T) {
	// Define sample task send params
	sampleTaskSendParams := &a2a.TaskSendParams{
		SkillID: "test-skill",
		Input:   map[string]interface{}{"message": "test"},
	}

	// Define sample JSON-RPC response for SendTask
	sampleResponse := a2a.JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      "client-request-1",
		Result: map[string]interface{}{
			"id":      "test-task",
			"status":  "pending",
			"skillId": "test-skill",
		},
	}
	sampleResponseJSON, err := json.Marshal(sampleResponse)
	if err != nil {
		t.Fatalf("Failed to marshal sample response: %v", err)
	}

	// Create a mock HTTP server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check for correct method
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST request, got %s", r.Method)
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		// Send the sample response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(sampleResponseJSON)
	}))
	defer server.Close()

	// Create a client using the mock server's URL
	client, err := NewClient(WithBaseURL(server.URL))
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// Send the task
	task, err := client.SendTask(context.Background(), sampleTaskSendParams)
	if err != nil {
		t.Fatalf("Failed to send task: %v", err)
	}

	// Check the response
	if task.ID != "test-task" {
		t.Errorf("Expected task ID 'test-task', got %s", task.ID)
	}
}

func TestGetTask(t *testing.T) {
    // Define sample task ID
    sampleTaskID := "test-task"

    // Define sample JSON-RPC response for GetTask
    sampleResponse := a2a.JSONRPCResponse{
        JSONRPC: "2.0",
        ID:      "client-request-1",
        Result: map[string]interface{}{
            "id":      sampleTaskID,
            "status":  "pending",
            "skillId": "test-skill",
        },
    }
    sampleResponseJSON, err := json.Marshal(sampleResponse)
    if err != nil {
        t.Fatalf("Failed to marshal sample response: %v", err)
    }

    // Create a mock HTTP server
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Check for correct method
        if r.Method != http.MethodPost {
            t.Errorf("Expected POST request, got %s", r.Method)
            w.WriteHeader(http.StatusMethodNotAllowed)
            return
        }

        // Send the sample response
        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(http.StatusOK)
        w.Write(sampleResponseJSON)
    }))
    defer server.Close()

    // Create a client using the mock server's URL
    client, err := NewClient(WithBaseURL(server.URL))
    if err != nil {
        t.Fatalf("Failed to create client: %v", err)
    }

    // Get the task
    task, err := client.GetTask(context.Background(), sampleTaskID)
    if err != nil {
        t.Fatalf("Failed to get task: %v", err)
    }

    // Check the response
    if task.ID != sampleTaskID {
        t.Errorf("Expected task ID '%s', got %s", sampleTaskID, task.ID)
    }
}

func TestCancelTask(t *testing.T) {
	// Define sample task ID
	sampleTaskID := "test-task"

	// Define sample JSON-RPC response for CancelTask
	sampleResponse := a2a.JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      "client-request-1",
		Result: map[string]interface{}{
			"id":     sampleTaskID,
			"status": "canceled",
		},
	}
	sampleResponseJSON, err := json.Marshal(sampleResponse)
	if err != nil {
		t.Fatalf("Failed to marshal sample response: %v", err)
	}

	// Create a mock HTTP server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check for correct method
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST request, got %s", r.Method)
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		// Send the sample response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(sampleResponseJSON)
	}))
	defer server.Close()

	// Create a client using the mock server's URL
	client, err := NewClient(WithBaseURL(server.URL))
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// Cancel the task
	task, err := client.CancelTask(context.Background(), sampleTaskID)
	if err != nil {
		t.Fatalf("Failed to cancel task: %v", err)
	}

	// Check the response
	if task.ID != sampleTaskID {
		t.Errorf("Expected task ID '%s', got %s", sampleTaskID, task.ID)
	}
    if task.Status != "canceled" {
        t.Errorf("Expected task status to be canceled, got %s", task.Status)
    }
}

func TestSetTaskPushNotification(t *testing.T) {
	// Define sample task push notification config params
	sampleTaskID := "test-task"
	samplePushNotificationConfig := &a2a.TaskPushNotificationConfigParams{
		TaskID:        sampleTaskID,
		CallbackURL:   "http://test.com/callback",
	}

	// Define sample JSON-RPC response for SetTaskPushNotification
	sampleResponse := a2a.JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      "client-request-1",
		Result: map[string]interface{}{
			"taskId": sampleTaskID,
			"callbackUrl": "http://test.com/callback",
		},
	}
	sampleResponseJSON, err := json.Marshal(sampleResponse)
	if err != nil {
		t.Fatalf("Failed to marshal sample response: %v", err)
	}

	// Create a mock HTTP server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check for correct method
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST request, got %s", r.Method)
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		// Send the sample response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(sampleResponseJSON)
	}))
	defer server.Close()

	// Create a client using the mock server's URL
	client, err := NewClient(WithBaseURL(server.URL))
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// Set the task push notification config
	config, err := client.SetTaskPushNotification(context.Background(), samplePushNotificationConfig)
	if err != nil {
		t.Fatalf("Failed to set task push notification config: %v", err)
	}

	// Check the response
	if config.TaskID != sampleTaskID {
		t.Errorf("Expected task ID '%s', got %s", sampleTaskID, config.TaskID)
	}
    if config.CallbackURL != "http://test.com/callback" {
        t.Errorf("Expected callback url 'http://test.com/callback', got %s", config.CallbackURL)
    }
}

func TestGetTaskPushNotification(t *testing.T){
	// Define sample task ID
	sampleTaskID := "test-task"

	// Define sample JSON-RPC response for GetTaskPushNotification
	sampleResponse := a2a.JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      "client-request-1",
		Result: map[string]interface{}{
			"taskId":      sampleTaskID,
			"callbackUrl": "http://test.com/callback",
		},
	}
	sampleResponseJSON, err := json.Marshal(sampleResponse)
	if err != nil {
		t.Fatalf("Failed to marshal sample response: %v", err)
	}

	// Create a mock HTTP server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check for correct method
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST request, got %s", r.Method)
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		// Read request body
        body, err := io.ReadAll(r.Body)
        if err != nil {
            t.Fatalf("Failed to read request body: %v", err)
        }
        // Check for correct request
        if !bytes.Contains(body, []byte(`"method":"tasks/pushNotification/get"`)) {
            t.Errorf("Expected request to contain 'method:tasks/pushNotification/get', got %s", body)
        }

		// Send the sample response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(sampleResponseJSON)
	}))
	defer server.Close()

	// Create a client using the mock server's URL
	client, err := NewClient(WithBaseURL(server.URL))
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// Get the task push notification config
	config, err := client.GetTaskPushNotification(context.Background(), sampleTaskID)
	if err != nil {
		t.Fatalf("Failed to get task push notification config: %v", err)
	}

	// Check the response
	if config.TaskID != sampleTaskID {
		t.Errorf("Expected task ID '%s', got %s", sampleTaskID, config.TaskID)
	}
    if config.CallbackURL != "http://test.com/callback" {
        t.Errorf("Expected callback url 'http://test.com/callback', got %s", config.CallbackURL)
    }
}

func TestSendJSONRPCRequest(t *testing.T) {
    // Define sample JSON-RPC request and response
    sampleRequest := a2a.JSONRPCRequest{
        JSONRPC: "2.0",
        Method:  "test/method",
        ID:      "test-request",
    }

    sampleResponse := a2a.JSONRPCResponse{
        JSONRPC: "2.0",
        ID:      "test-request",
        Result:  map[string]string{"status": "success"},
    }

    // Marshal the sample response to JSON
    sampleResponseJSON, err := json.Marshal(sampleResponse)
    if err != nil {
        t.Fatalf("Failed to marshal sample response: %v", err)
    }

    // Create a mock HTTP server
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Check for correct method
        if r.Method != http.MethodPost {
            t.Errorf("Expected POST request, got %s", r.Method)
            w.WriteHeader(http.StatusMethodNotAllowed)
            return
        }
        
        // Check for content type
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Expected Content-Type: application/json, got %s", r.Header.Get("Content-Type"))
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if r.Header.Get("Accept") != "application/json" {
			t.Errorf("Expected Accept: application/json, got %s", r.Header.Get("Accept"))
			w.WriteHeader(http.StatusBadRequest)
			return
		}

        // Read request body
        body, err := io.ReadAll(r.Body)
        if err != nil {
            t.Fatalf("Failed to read request body: %v", err)
        }

        // Check for correct request
        if !bytes.Contains(body, []byte(`"method":"test/method"`)) {
            t.Errorf("Expected request to contain 'method:test/method', got %s", body)
        }

        // Send the sample response
        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(http.StatusOK)
        w.Write(sampleResponseJSON)
    }))
    defer server.Close()

    // Create a client using the mock server's URL
    client, err := NewClient(WithBaseURL(server.URL))
    if err != nil {
        t.Fatalf("Failed to create client: %v", err)
    }

    // Send the JSON-RPC request
    var response map[string]string
    err = client.sendJSONRPCRequest(context.Background(), sampleRequest, &response)
    if err != nil {
        t.Fatalf("Failed to send JSON-RPC request: %v", err)
    }
}