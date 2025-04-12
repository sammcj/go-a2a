package test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/sammcj/go-a2a/a2a"
	"github.com/sammcj/go-a2a/llm"
	"github.com/sammcj/go-a2a/llm/gollm"
	"github.com/sammcj/go-a2a/server"
)

// MockTaskManager is a mock implementation of the TaskManager interface for testing.
type MockTaskManager struct {
	StartTaskFunc func(ctx context.Context, agentID, skillID string, input map[string]interface{}, auth *a2a.AgentAuthentication, agentCapabilities *a2a.AgentCapabilities) (string, error)
	GetTaskFunc   func(id string, auth *a2a.AgentAuthentication) (*a2a.AgentTask, error)
	StopTaskFunc  func(id string) error
	GetTasksFunc  func(auth *a2a.AgentAuthentication) ([]*a2a.AgentTask, error)
}

func (m *MockTaskManager) StartTask(ctx context.Context, agentID, skillID string, input map[string]interface{}, auth *a2a.AgentAuthentication, agentCapabilities *a2a.AgentCapabilities) (string, error) {
	if m.StartTaskFunc != nil {
		return m.StartTaskFunc(ctx, agentID, skillID, input, auth, agentCapabilities)
	}
	return "", errors.New("StartTask not implemented")
}

func (m *MockTaskManager) GetTask(id string, auth *a2a.AgentAuthentication) (*a2a.AgentTask, error) {
	if m.GetTaskFunc != nil {
		return m.GetTaskFunc(id, auth)
	}
	return nil, errors.New("GetTask not implemented")
}

func (m *MockTaskManager) StopTask(id string) error {
	if m.StopTaskFunc != nil {
		return m.StopTaskFunc(id)
	}
	return errors.New("StopTask not implemented")
}
func (m *MockTaskManager) GetTasks(auth *a2a.AgentAuthentication) ([]*a2a.AgentTask, error) {
	if m.GetTasksFunc != nil {
		return m.GetTasksFunc(auth)
	}
	return nil, errors.New("GetTasks not implemented")
}

type MockAgentEngine struct {
	HandleRequestFunc func(w http.ResponseWriter, r *http.Request)
}

func (m *MockAgentEngine) HandleRequest(w http.ResponseWriter, r *http.Request) {
	if m.HandleRequestFunc != nil {
		m.HandleRequestFunc(w, r)
	}
}

// TestNewServer tests the NewServer function.
func TestNewServer(t *testing.T) {
	testCases := []struct {
		name          string
		opts          []server.Option
		expectedError error
	}{
		{
			name:          "NoAgentCard",
			opts:          []server.Option{},
			expectedError: fmt.Errorf("agent card configuration is required"),
		},
		{
			name: "ValidServer",
			opts: []server.Option{
				server.WithAgentCard(&a2a.AgentCard{ID: "test-agent"}),
				server.WithTaskHandler(func(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
					return nil, nil
				}),
			},
			expectedError: nil,
		},
		{
			name: "ValidServerWithAgent",
			opts: []server.Option{
				server.WithAgentCard(&a2a.AgentCard{ID: "test-agent"}),
				server.WithTaskHandler(func(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
					return nil, nil
				}),
				server.WithBasicGollmAgent("test","test","test","test"),
			},
			expectedError: nil,
		},
		{
			name: "ValidServerWithGollmOptions",
			opts: []server.Option{
				server.WithAgentCard(&a2a.AgentCard{ID: "test-agent"}),
				server.WithTaskHandler(func(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
					return nil, nil
				}),
				server.WithGollmOptions(&gollm.Options{Provider: "test"}),
			},
			expectedError: nil,
		},
		{
			name: "ValidServerWithGollmOptionsAndAgent",
			opts: []server.Option{
				server.WithAgentCard(&a2a.AgentCard{ID: "test-agent"}),
				server.WithTaskHandler(func(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
					return nil, nil
				}),
				server.WithGollmOptions(&gollm.Options{Provider: "test"}),
				server.WithBasicGollmAgent("test","test","test","test"),
			},
			expectedError: nil,
		},
		{
			name: "InvalidServerNoAgentNoOptions",
			opts: []server.Option{
				server.WithAgentCard(&a2a.AgentCard{ID: "test-agent"}),
				server.WithTaskHandler(func(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
					return nil, nil
				}),
			},
			expectedError: errors.New("gollm options must be set when agent engine not set"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := server.NewServer(tc.opts...)
			if tc.expectedError == nil && err != nil {
				t.Fatalf("Expected no error, but got: %v", err)
			}
			if tc.expectedError != nil {
				if err == nil {
					t.Fatalf("Expected error: %v, but got no error", tc.expectedError)
				}
				if err.Error() != tc.expectedError.Error() {
					t.Fatalf("Expected error: %v, but got: %v", tc.expectedError, err)
				}
			}
		})
	}
}

// TestServerStartAndStop tests the Start and Stop functions of the Server.
func TestServerStartAndStop(t *testing.T) {
	// Create a mock task manager
	mockTaskManager := &MockTaskManager{}

	// Create a server with the mock task manager
	srv, err := server.NewServer(
		server.WithAgentCard(&a2a.AgentCard{ID: "test-agent"}),
		server.WithListenAddress(":8082"),
		server.WithTaskManager(mockTaskManager),
		server.WithTaskHandler(func(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
			return nil, nil
		}),
		server.WithGollmOptions(&gollm.Options{}),
	)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	// Start the server in a goroutine
	var startErr error
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		startErr = srv.Start()
	}()

	// Wait for a short period to allow the server to start
	time.Sleep(100 * time.Millisecond)

	// Check if there was an error during start
	if startErr != nil && startErr != http.ErrServerClosed {
		t.Fatalf("Server start failed: %v", startErr)
	}

	// Stop the server
	stopCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	stopErr := srv.Stop(stopCtx)
	if stopErr != nil {
		t.Fatalf("Server stop failed: %v", stopErr)
	}

	// Wait for the server goroutine to finish
	wg.Wait()

	// Check if there was an error during start
	if startErr != nil && startErr != http.ErrServerClosed {
		t.Fatalf