package test

import (
	"bytes"
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
	"github.com/sammcj/go-a2a/pkg/config"
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
		{
			name: "ValidDefaultConfig",
			opts: []server.Option{
				server.WithAgentCard(&a2a.AgentCard{ID: "test-agent"}),
				server.WithTaskHandler(func(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) { return nil, nil }),
			},
			expectedError: nil,
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
	}
}

// TestNewGollmOptionsFromConfig tests the NewGollmOptionsFromConfig function.
func TestNewGollmOptionsFromConfig(t *testing.T) {
	testCases := []struct {
		name          string
		llmConfig     config.LLMConfig
		expectedError error
	}{
		{
			name:          "ValidConfig",
			llmConfig:     config.LLMConfig{Provider: "openai", Model: "gpt-3.5-turbo", ApiKey: "test-api-key"},
			expectedError: nil,
		},
		{
			name:          "NoProvider",
			llmConfig:     config.LLMConfig{},
			expectedError: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := server.NewGollmOptionsFromConfig(tc.llmConfig)
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

// TestDefaultGollmConfig tests the DefaultGollmConfig function.
func TestDefaultGollmConfig(t *testing.T) {
	defaultConfig := server.DefaultGollmConfig()

	if defaultConfig.Provider != "openai" {
		t.Errorf("Expected default provider to be 'openai', but got: %s", defaultConfig.Provider)
	}
	if defaultConfig.Options == nil {
		t.Fatalf("Expected default options to not be nil")
	}
	if defaultConfig.Options["temperature"] != 0.7 {
		t.Errorf("Expected default temperature to be 0.7, but got: %v", defaultConfig.Options["temperature"])
	}
}

// TestNewGollmOptions tests the NewGollmOptions function.
func TestNewGollmOptions(t *testing.T) {
	testCases := []struct {
		name          string
		llmConfig     server.GollmConfig
		expectedError error
		checkOpts     func(opts *gollm.Options) error
	}{
		{
			name:          "ValidConfig",
			llmConfig:     server.GollmConfig{Provider: "openai", Model: "gpt-3.5-turbo", APIKey: "test-api-key", BaseUrl: "http://test.com", SystemPrompt: "test-prompt", Options: map[string]interface{}{"temperature": 0.8}},
			expectedError: nil,
			checkOpts: func(opts *gollm.Options) error {
				if opts.Provider != "openai" {
					return fmt.Errorf("expected provider to be 'openai', but got: %s", opts.Provider)
				}
				if opts.Model != "gpt-3.5-turbo" {
					return fmt.Errorf("expected model to be 'gpt-3.5-turbo', but got: %s", opts.Model)
				}
				if opts.APIKey != "test-api-key" {
					return fmt.Errorf("expected api key to be 'test-api-key', but got: %s", opts.APIKey)
				}
				if opts.BaseURL != "http://test.com" {
					return fmt.Errorf("expected base url to be 'http://test.com', but got: %s", opts.BaseURL)
				}
				if opts.SystemPrompt != "test-prompt" {
					return fmt.Errorf("expected system prompt to be 'test-prompt', but got: %s", opts.SystemPrompt)
				}
				if opts.Options["temperature"] != 0.8 {
					return fmt.Errorf("expected temperature to be 0.8, but got: %v", opts.Options["temperature"])
				}
				return nil
			},
		},
		{
			name:          "NoProvider",
			llmConfig:     server.GollmConfig{},
			expectedError: nil,
			checkOpts: func(opts *gollm.Options) error {
				if opts.Provider != "" {
					return fmt.Errorf("expected provider to be '', but got: %s", opts.Provider)
				}
				return nil
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			opts, err := server.NewGollmOptions(tc.llmConfig)
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
			if tc.checkOpts != nil {
				if err := tc.checkOpts(opts); err != nil {
					t.Fatalf("Check options failed: %v", err)
				}
			}
		})
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := server.DefaultConfig()

	if cfg.ListenAddress != ":8080" {
		t.Errorf("Expected ListenAddress to be :8080, got %s", cfg.ListenAddress)
	}
}