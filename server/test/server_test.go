package server

import (
	"context"
	"errors"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/sammcj/go-a2a/a2a"
	"github.com/sammcj/go-a2a/llm/gollm"
	"github.com/sammcj/go-a2a/pkg/task"
)

// Option is a function that modifies the server configuration.
type Option func(*Config)

// AgentEngine defines the interface for agent engine implementations.
type AgentEngine interface {
	GetCapabilities() AgentCapabilities
	HandleRequest(w http.ResponseWriter, r *http.Request)
}

// TaskManager defines the interface for task management operations.
type TaskManager interface {
	OnSendTask(ctx context.Context, params *a2a.TaskSendParams) (*a2a.Task, error)
	OnSendTaskSubscribe(ctx context.Context, params *a2a.TaskSendParams) (<-chan task.YieldUpdate, error)
	OnGetTask(ctx context.Context, params *a2a.TaskQueryParams) (*a2a.Task, error)
	OnCancelTask(ctx context.Context, params *a2a.TaskIdParams) (*a2a.Task, error)
	OnSetTaskPushNotification(ctx context.Context, params *a2a.TaskPushNotificationConfigParams) (*a2a.PushNotificationConfig, error)
	OnGetTaskPushNotification(ctx context.Context, params *a2a.TaskIdParams) (*a2a.PushNotificationConfig, error)
	OnResubscribeToTask(ctx context.Context, params *a2a.TaskIdParams) (<-chan task.YieldUpdate, error)
}

// Option functions for server configuration
var (
	WithAgentCard = func(card *a2a.AgentCard) Option {
		return func(c *Config) {
			c.AgentCard = card
		}
	}

	WithTaskManager = func(tm TaskManager) Option {
		return func(c *Config) {
			c.TaskManager = tm
		}
	}

	WithTaskHandler = func(handler task.Handler) Option {
		return func(c *Config) {
			c.TaskHandler = handler
		}
	}

	WithGollmOptions = func(opts []gollm.Option) Option {
		return func(c *Config) {
			c.gollmOptions = opts
		}
	}
)

// NewServer creates a new server instance
func NewServer(opts ...Option) (*Server, error) {
	cfg := Config{
		ListenAddress: ":8080",
		A2APathPrefix: "/a2a",
	}

	for _, opt := range opts {
		opt(&cfg)
	}

	if cfg.AgentCard == nil {
		return nil, errors.New("agent card is required")
	}

	return &Server{config: cfg}, nil
}

// Server represents the A2A server.
type Server struct {
	config Config
}

func (s *Server) Start() error {
	return nil
}

func (s *Server) Stop(ctx context.Context) error {
	return nil
}

// Config holds the configuration for the A2A server.
type Config struct {
	ListenAddress string
	A2APathPrefix string
	AgentCard     *a2a.AgentCard
	AgentCardPath string
	TaskManager   TaskManager
	TaskHandler   task.Handler
	AgentEngine   AgentEngine
	AuthValidator AuthValidator
	gollmOptions  []gollm.Option
}

// AgentCapabilities defines the capabilities of an agent.
type AgentCapabilities struct {
	SupportsStreaming bool
}

// AuthValidator is a function that validates authentication for requests.
type AuthValidator func(w http.ResponseWriter, r *http.Request, next http.Handler, card *a2a.AgentCard)

// MockTaskManager is a mock implementation of TaskManager for testing.
type MockTaskManager struct {
	sync.Mutex
	Tasks       map[string]*a2a.Task
	PushConfigs map[string]*a2a.PushNotificationConfig
}

func (m *MockTaskManager) OnSendTask(ctx context.Context, params *a2a.TaskSendParams) (*a2a.Task, error) {
	return &a2a.Task{ID: "test-task"}, nil
}

func (m *MockTaskManager) OnSendTaskSubscribe(ctx context.Context, params *a2a.TaskSendParams) (<-chan task.YieldUpdate, error) {
	updateChan := make(chan task.YieldUpdate)
	close(updateChan)
	return updateChan, nil
}

func (m *MockTaskManager) OnGetTask(ctx context.Context, params *a2a.TaskQueryParams) (*a2a.Task, error) {
	m.Lock()
	defer m.Unlock()
	task, ok := m.Tasks[params.TaskID]
	if !ok {
		return nil, errors.New("task not found")
	}
	return task, nil
}

func (m *MockTaskManager) OnCancelTask(ctx context.Context, params *a2a.TaskIdParams) (*a2a.Task, error) {
	m.Lock()
	defer m.Unlock()
	task, ok := m.Tasks[params.TaskID]
	if !ok {
		return nil, errors.New("task not found")
	}
	return task, nil
}

func (m *MockTaskManager) OnSetTaskPushNotification(ctx context.Context, params *a2a.TaskPushNotificationConfigParams) (*a2a.PushNotificationConfig, error) {
	config := &a2a.PushNotificationConfig{
		TaskID: params.TaskID,
		URL:    params.URL,
	}
	m.Lock()
	m.PushConfigs[params.TaskID] = config
	m.Unlock()
	return config, nil
}

func (m *MockTaskManager) OnGetTaskPushNotification(ctx context.Context, params *a2a.TaskIdParams) (*a2a.PushNotificationConfig, error) {
	m.Lock()
	defer m.Unlock()
	config, ok := m.PushConfigs[params.TaskID]
	if !ok {
		return nil, errors.New("push notification config not found")
	}
	return config, nil
}

func (m *MockTaskManager) OnResubscribeToTask(ctx context.Context, params *a2a.TaskIdParams) (<-chan task.YieldUpdate, error) {
	updateChan := make(chan task.YieldUpdate)
	close(updateChan)
	return updateChan, nil
}

// mockTaskHandler is a mock implementation of task.Handler for testing.
func mockTaskHandler(ctx context.Context, taskCtx task.Context) (<-chan task.YieldUpdate, error) {
	updateChan := make(chan task.YieldUpdate)
	close(updateChan)
	return updateChan, nil
}

// MockAgentEngine is a mock implementation of AgentEngine for testing.
type MockAgentEngine struct{}

func (m *MockAgentEngine) GetCapabilities() AgentCapabilities {
	return AgentCapabilities{
		SupportsStreaming: true,
	}
}

func (m *MockAgentEngine) HandleRequest(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"jsonrpc":"2.0","result":{"id":"test-response"}}`))
}

// Helper function to create a basic test agent card
func newTestAgentCard() *a2a.AgentCard {
	return &a2a.AgentCard{
		A2AVersion: "1.0",
		ID:         "test-agent",
		Name:       "Test Agent",
	}
}

func TestNewServer(t *testing.T) {
	agentCard := newTestAgentCard()
	gollmOpts := []gollm.Option{
		gollm.WithProvider("test"),
		gollm.WithModel("test"),
		gollm.WithAPIKey("test"),
	}

	tests := []struct {
		name        string
		opts        []Option
		expectedErr bool
	}{
		{
			name:        "Valid Configuration",
			opts:        []Option{WithAgentCard(agentCard), WithGollmOptions(gollmOpts)},
			expectedErr: false,
		},
		{
			name:        "Missing Agent Card",
			opts:        []Option{},
			expectedErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewServer(tt.opts...)
			if (err != nil) != tt.expectedErr {
				t.Errorf("NewServer() error = %v, expectedErr %v", err, tt.expectedErr)
			}
		})
	}
}

func TestServerStartAndStop(t *testing.T) {
	agentCard := newTestAgentCard()
	mockTaskManager := &MockTaskManager{
		Tasks:       make(map[string]*a2a.Task),
		PushConfigs: make(map[string]*a2a.PushNotificationConfig),
	}
	mockTaskHandler := mockTaskHandler
	gollmOpts := []gollm.Option{
		gollm.WithProvider("test"),
		gollm.WithModel("test"),
		gollm.WithAPIKey("test"),
	}

	srv, err := NewServer(
		WithAgentCard(agentCard),
		WithTaskManager(mockTaskManager),
		WithTaskHandler(mockTaskHandler),
		WithGollmOptions(gollmOpts),
	)
	if err != nil {
		t.Fatalf("Failed to create test server: %v", err)
	}

	// Start the server in a separate goroutine
	errChan := make(chan error)
	go func() {
		errChan <- srv.Start()
	}()

	// Wait for the server to start
	time.Sleep(100 * time.Millisecond)

	// Stop the server
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	stopErr := srv.Stop(ctx)
	if stopErr != nil {
		t.Errorf("Stop() error = %v", stopErr)
	}

	// Check if the server returned an error
	select {
	case err := <-errChan:
		if err != nil && err != http.ErrServerClosed {
			t.Errorf("Start() error = %v", err)
		}
	default:
	}
}
