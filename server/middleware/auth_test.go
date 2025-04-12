package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sammcj/go-a2a/a2a"
)

func TestAuthMiddleware(t *testing.T) {
	// Create a simple agent card with authentication schemes
	card := &a2a.AgentCard{
		ID:   "test-agent",
		Name: "Test Agent",
		Authentication: []a2a.AgentAuthentication{
			{
				Type: "bearer",
			},
			{
				Type: "header",
				Configuration: map[string]interface{}{
					"headerName": "X-API-Key",
				},
			},
		},
	}

	// Create a simple validator that accepts a specific token
	validator := func(ctx context.Context, info AuthInfo) (bool, error) {
		if info.Type == "bearer" && info.Value == "valid-token" {
			return true, nil
		}
		if info.Type == "header" && info.Scheme == "X-API-Key" && info.Value == "valid-api-key" {
			return true, nil
		}
		return false, nil
	}

	// Create a simple handler that returns 200 OK
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if auth info was added to context
		authInfo, ok := r.Context().Value(AuthKey{}).(*AuthInfo)
		if !ok {
			t.Error("Auth info not found in context")
		} else {
			// Write the auth type to the response for testing
			w.Write([]byte(authInfo.Type))
		}
	})

	// Create the middleware
	middleware := AuthMiddleware(card, validator)
	handler := middleware(nextHandler)

	tests := []struct {
		name           string
		path           string
		headers        map[string]string
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "Agent card path should bypass auth",
			path:           "/.well-known/agent.json",
			headers:        map[string]string{},
			expectedStatus: http.StatusOK,
			expectedBody:   "",
		},
		{
			name:           "Missing auth should return 401",
			path:           "/a2a",
			headers:        map[string]string{},
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   "",
		},
		{
			name:           "Invalid bearer token should return 401",
			path:           "/a2a",
			headers:        map[string]string{"Authorization": "Bearer invalid-token"},
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   "",
		},
		{
			name:           "Valid bearer token should pass",
			path:           "/a2a",
			headers:        map[string]string{"Authorization": "Bearer valid-token"},
			expectedStatus: http.StatusOK,
			expectedBody:   "bearer",
		},
		{
			name:           "Invalid API key should return 401",
			path:           "/a2a",
			headers:        map[string]string{"X-API-Key": "invalid-api-key"},
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   "",
		},
		{
			name:           "Valid API key should pass",
			path:           "/a2a",
			headers:        map[string]string{"X-API-Key": "valid-api-key"},
			expectedStatus: http.StatusOK,
			expectedBody:   "header",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a request with the specified headers
			req := httptest.NewRequest("GET", tt.path, nil)
			for k, v := range tt.headers {
				req.Header.Set(k, v)
			}

			// Create a response recorder
			rr := httptest.NewRecorder()

			// Serve the request
			handler.ServeHTTP(rr, req)

			// Check the status code
			if rr.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, rr.Code)
			}

			// Check the response body if expected
			if tt.expectedBody != "" && rr.Body.String() != tt.expectedBody {
				t.Errorf("Expected body %q, got %q", tt.expectedBody, rr.Body.String())
			}
		})
	}
}

// Additional test cases can be added here as needed
