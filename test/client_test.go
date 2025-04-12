package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/sammcj/go-a2a/a2a"
)

// MockHTTPClient is a mock implementation of the HTTPClient interface.
type MockHTTPClient struct {
	DoFunc func(req *http.Request) (*http.Response, error)
}

// Do mocks the HTTPClient's Do method.
func (m *MockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	return m.DoFunc(req)
}

// TestNewClient tests the NewClient function.
func TestNewClient(t *testing.T) {
	t.Run("valid client", func(t *testing.T) {
		c, err := NewClient(WithDefaultAgentURL("http://localhost:8080"))
		if err != nil {
			t.Errorf("NewClient() returned an error: %v", err)
		}
		if c.config.DefaultAgentURL != "http://localhost:8080" {
			t.Errorf("NewClient() did not set default agent URL correctly")
		}
	})

	t.Run("invalid agent url", func(t *testing.T) {
		_, err := NewClient(WithDefaultAgentURL("::invalid::url"))
		if err == nil {
			t.Errorf("NewClient() did not return an error for invalid agent URL")
		}
	})
}

// TestFetchAgentCard tests the FetchAgentCard function.
func TestFetchAgentCard(t *testing.T) {
	testCases := []struct {
		name           string
		mockHTTPClient *MockHTTPClient
		expectedCard   *a2a.AgentCard
		expectError    bool
	}{
		{
			name: "successful fetch",
			mockHTTPClient: &MockHTTPClient{
				DoFunc: func(req *http.Request) (*http.Response, error) {
					// Check request
					if req.Method != http.MethodGet {
						t.Errorf("Expected method to be GET, got %s", req.Method)
					}

					card := a2a.AgentCard{ID: "test-agent"}
					cardBytes, _ := json.Marshal(card)

					return &http.Response{
						StatusCode: http.StatusOK,
						Body:       io.NopCloser(bytes.NewReader(cardBytes)),
					}, nil
				},
			},
			expectedCard: &a2a.AgentCard{ID: "test-agent"},
			expectError:  false,
		},
		{
			name: "non-200 status code",
			mockHTTPClient: &MockHTTPClient{
				DoFunc: func(req *http.Request) (*http.Response, error) {
					return &http.Response{
						StatusCode: http.StatusNotFound,
						Body:       io.NopCloser(bytes.NewReader([]byte{})),
					}, nil
				},
			},
			expectedCard: nil,
			expectError:  true,
		},
		{
			name: "http error",
			mockHTTPClient: &MockHTTPClient{
				DoFunc: func(req *http.Request) (*http.Response, error) {
					return nil, errors.New("http error")
				},
			},
			expectedCard: nil,
			expectError:  true,
		},
		{
			name: "invalid json response",
			mockHTTPClient: &MockHTTPClient{
				DoFunc: func(req *http.Request) (*http.Response, error) {
					return &http.Response{
						StatusCode: http.StatusOK,
						Body:       io.NopCloser(strings.NewReader("invalid json")),
					}, nil
				},
			},
			expectedCard: nil,
			expectError:  true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			client := &Client{
				httpClient: tc.mockHTTPClient,
				config:     &Config{DefaultAgentURL: "http://localhost:8080"},
			}

			card, err := client.FetchAgentCard(context.Background(), "http://localhost:8080")

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

// Helper function to create a test server with custom handler
func createTestServer(handlerFunc func(w http.ResponseWriter, r *http.Request)) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(handlerFunc))
}
func TestDoRequest(t *testing.T) {
	testCases := []struct {
		name          string
		handlerFunc   func(w http.ResponseWriter, r *http.Request)
		expectedError bool
	}{
		{
			name: "successful request",
			handlerFunc: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"message": "success"}`))
			},
			expectedError: false,
		},
		{
			name: "non-200 status code",
			handlerFunc: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusNotFound)
				w.Write([]byte(`{"message": "not found"}`))
			},
			expectedError: true,
		},
		{
			name: "invalid json response",
			handlerFunc: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`invalid json`))
			},
			expectedError: true,
		},
		{
			name: "timeout",
			handlerFunc: func(w http.ResponseWriter, r *http.Request) {
				time.Sleep(2 * time.Second) // Simulate a timeout
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"message": "timeout"}`))
			},
			expectedError: true,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			server := createTestServer(tc.handlerFunc)
			defer server.Close()
			client, _ := NewClient(WithDefaultAgentURL(server.URL))

			ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
			defer cancel()
			_, err := client.DoRequest(ctx, http.MethodGet, server.URL, nil)

			if tc.expectedError {
				if err == nil {
					t.Errorf("Expected an error, but got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Did not expect an error, but got %v", err)
				}
			}
		})
	}
}