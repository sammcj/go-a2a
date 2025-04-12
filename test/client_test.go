package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/sammcj/go-a2a/a2a"
	"github.com/sammcj/go-a2a/client"
)

// Function aliases
var (
	NewClient      = client.NewClient
	WithBaseURL    = client.WithBaseURL
	WithHTTPClient = client.WithHTTPClient
)

// TestNewClient tests the NewClient function.
func TestNewClient(t *testing.T) {
	t.Run("valid client", func(t *testing.T) {
		// Set up test server to verify URL is set correctly
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !strings.HasPrefix(r.URL.String(), "/") {
				t.Errorf("Expected URL to start with /, got %s", r.URL.String())
			}
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(a2a.AgentCard{ID: "test"})
		}))
		defer server.Close()

		client, err := NewClient(WithBaseURL(server.URL))
		if err != nil {
			t.Errorf("NewClient() with valid URL returned error: %v", err)
		}

		// Test client by making a request
		_, err = client.FetchAgentCard(context.Background())
		if err != nil {
			t.Errorf("Failed to make request with configured client: %v", err)
		}
	})

	t.Run("invalid base url", func(t *testing.T) {
		_, err := NewClient(WithBaseURL("::invalid::url"))
		if err == nil {
			t.Errorf("NewClient() did not return an error for invalid base URL")
		}
	})

	t.Run("missing base url", func(t *testing.T) {
		_, err := NewClient()
		if err == nil {
			t.Errorf("NewClient() did not return an error for missing base URL")
		}
	})
}

// TestFetchAgentCard tests the FetchAgentCard function.
func TestFetchAgentCard(t *testing.T) {
	testCases := []struct {
		name         string
		handlerFunc  func(*testing.T) http.Handler
		expectedCard *a2a.AgentCard
		expectError  bool
	}{
		{
			name: "successful fetch",
			handlerFunc: func(t *testing.T) http.Handler {
				return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					// Check request
					if r.Method != http.MethodGet {
						t.Errorf("Expected method to be GET, got %s", r.Method)
					}
					if !strings.HasSuffix(r.URL.Path, "/.well-known/agent.json") {
						t.Errorf("Expected path to end with /.well-known/agent.json, got %s", r.URL.Path)
					}

					card := a2a.AgentCard{ID: "test-agent"}
					json.NewEncoder(w).Encode(card)
				})
			},
			expectedCard: &a2a.AgentCard{ID: "test-agent"},
			expectError:  false,
		},
		{
			name: "non-200 status code",
			handlerFunc: func(t *testing.T) http.Handler {
				return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusNotFound)
				})
			},
			expectedCard: nil,
			expectError:  true,
		},
		{
			name: "invalid json response",
			handlerFunc: func(t *testing.T) http.Handler {
				return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Write([]byte("invalid json"))
				})
			},
			expectedCard: nil,
			expectError:  true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(tc.handlerFunc(t))
			defer server.Close()

			client, err := NewClient(WithBaseURL(server.URL))
			if err != nil {
				t.Fatalf("Failed to create client: %v", err)
			}

			card, err := client.FetchAgentCard(context.Background())

			if tc.expectError {
				if err == nil {
					t.Errorf("Expected an error, but got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Did not expect an error, but got %v", err)
				}
				if card.ID != tc.expectedCard.ID {
					t.Errorf("Expected agent card ID to be %s, but got %s", tc.expectedCard.ID, card.ID)
				}
			}
		})
	}
}

// TestSendTask tests the SendTask function.
func TestSendTask(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST request, got %s", r.Method)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Expected Content-Type: application/json, got %s", r.Header.Get("Content-Type"))
		}

		// Return successful response
		w.WriteHeader(http.StatusOK)
		response := a2a.JSONRPCResponse{
			JSONRPC: "2.0",
			Result:  a2a.Task{ID: "test-task"},
			ID:      "client-request-1",
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client, err := NewClient(WithBaseURL(server.URL))
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	params := &a2a.TaskSendParams{
		Message: a2a.Message{
			Role: a2a.RoleUser,
			Parts: []a2a.Part{
				a2a.TextPart{
					Type: "text",
					Text: "Hello",
				},
			},
		},
	}

	task, err := client.SendTask(context.Background(), params)
	if err != nil {
		t.Errorf("SendTask() returned error: %v", err)
	}
	if task.ID != "test-task" {
		t.Errorf("Expected task ID to be test-task, got %s", task.ID)
	}
}
