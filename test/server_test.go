package server

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/sammcj/go-a2a/a2a"
)

// MockTaskManager is a mock implementation of TaskManager for testing.
type MockTaskManager struct {
	sync.Mutex
	Tasks       map[string]*Task
	AddTaskFunc func(task *Task) error
}

func (m *MockTaskManager) AddTask(task *Task) error {
	m.Lock()
	defer m.Unlock()
	if m.AddTaskFunc != nil {
		return m.AddTaskFunc(task)
	}
	m.Tasks[task.ID] = task
	return nil
}

func (m *MockTaskManager) GetTask(taskID string) (*Task, error) {
	m.Lock()
	defer m.Unlock()
	task, ok := m.Tasks[taskID]
	if !ok {
		return nil, errors.New("task not found")
	}
	return task, nil
}

func (m *MockTaskManager) UpdateTask(task *Task) error {
	m.Lock()
	defer m.Unlock()
	if _, ok := m.Tasks[task.ID]; !ok {
		return errors.New("task not found")
	}
	m.Tasks[task.ID] = task
	return nil
}

func (m *MockTaskManager) RemoveTask(taskID string) error {
	m.Lock()
	defer m.Unlock()
	if _, ok := m.Tasks[taskID]; !ok {
		return errors.New("task not found")
	}
	delete(m.Tasks, taskID)
	return nil
}

// MockTaskHandler is a mock implementation of TaskHandler for testing.
type MockTaskHandler struct {
	HandleTaskFunc func(task *Task) error
}

func (m *MockTaskHandler) HandleTask(task *Task) error {
	if m.HandleTaskFunc != nil {
		return m.HandleTaskFunc(task)
	}
	return nil
}

// MockAgentEngine is a mock implementation of AgentEngine for testing.
type MockAgentEngine struct {
	HandleRequestFunc func(w http.ResponseWriter, r *http.Request)
}

func (m *MockAgentEngine) HandleRequest(w http.ResponseWriter, r *http.Request) {
	if m.HandleRequestFunc != nil {
		m.HandleRequestFunc(w, r)
	}
}

// Helper function to create a test server with custom options
func newTestServer(opts ...Option) (*Server, error) {
	return NewServer(opts...)
}

// Helper function to create a basic test agent card
func newTestAgentCard() *a2a.AgentCard {
	id := "test-agent"
	name := "Test Agent"
	return &a2a.AgentCard{
		A2AVersion: "1.0",
		ID:         id,
		Name:       name,
	}
}

func TestNewServer(t *testing.T) {
	agentCard := newTestAgentCard()

	tests := []struct {
		name        string
		opts        []Option
		expectedErr bool
	}{
		{
			name:        "Valid Configuration",
			opts:        []Option{WithAgentCard(agentCard)},
			expectedErr: false,
		},
		{
			name:        "Missing Agent Card",
			opts:        []Option{},
			expectedErr: true,
		},
		{
			name:        "Missing Gollm options",
			opts:        []Option{WithAgentCard(agentCard)},
			expectedErr: true,
		},
		{
			name: "With Gollm options",
			opts: []Option{
				WithAgentCard(agentCard),
				WithGollmOptions(&testGollmOptions)},
			expectedErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := newTestServer(tt.opts...)
			if (err != nil) != tt.expectedErr {
				t.Errorf("NewServer() error = %v, expectedErr %v", err, tt.expectedErr)
			}
		})
	}
}

func TestServerStartAndStop(t *testing.T) {
	agentCard := newTestAgentCard()
	mockTaskManager := &MockTaskManager{Tasks: make(map[string]*Task)}
	mockTaskHandler := &MockTaskHandler{}
	testGollmOptions := testGollmOptions
	srv, err := newTestServer(
		WithAgentCard(agentCard),
		WithTaskManager(mockTaskManager),
		WithTaskHandler(mockTaskHandler),
		WithGollmOptions(&testGollmOptions),
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

func TestServerHandleAgentEngineRequest(t *testing.T) {
	agentCard := newTestAgentCard()
	mockAgentEngine := &MockAgentEngine{}

	srv, err := newTestServer(
		WithAgentCard(agentCard),
		WithAgentEngine(mockAgentEngine),
	)
	if err != nil {
		t.Fatalf("Failed to create test server: %v", err)
	}

	// Create a test HTTP request
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	// Call handleAgentEngineRequest
	srv.handleAgentEngineRequest(w, req)
}

var testGollmOptions = gollm.Options{
	Provider: "test",
	Model:    "test",
}