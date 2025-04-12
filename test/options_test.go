package test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/sammcj/go-a2a/a2a"
	"github.com/sammcj/go-a2a/llm/gollm"
	"github.com/sammcj/go-a2a/server"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockTaskManager is a mock implementation of TaskManager for testing.
type MockTaskManager struct {
	mock.Mock
}

func (m *MockTaskManager) Start() error {
	args := m.Called()
	return args.Error(0)
}
func (m *MockTaskManager) Stop() error {
	args := m.Called()
	return args.Error(0)
}
func (m *MockTaskManager) AddTask(task *a2a.AgentTask) error {
	args := m.Called(task)
	return args.Error(0)
}
func (m *MockTaskManager) GetTask(taskID string) (*a2a.AgentTask, error) {
	args := m.Called(taskID)
	return args.Get(0).(*a2a.AgentTask), args.Error(1)
}
func (m *MockTaskManager) GetAllTasks() ([]*a2a.AgentTask, error) {
	args := m.Called()
	return args.Get(0).([]*a2a.AgentTask), args.Error(1)
}
func (m *MockTaskManager) GetTasks(taskIDs []string) ([]*a2a.AgentTask, error) {
	args := m.Called(taskIDs)
	return args.Get(0).([]*a2a.AgentTask), args.Error(1)
}
func (m *MockTaskManager) DeleteTask(taskID string) error {
	args := m.Called(taskID)
	return args.Error(0)
}
func (m *MockTaskManager) DeleteAllTasks() error {
	args := m.Called()
	return args.Error(0)
}

// MockTaskHandler is a mock implementation of TaskHandler for testing.
type MockTaskHandler struct {
	mock.Mock
}

func (m *MockTaskHandler) HandleTask(task *a2a.AgentTask) error {
	args := m.Called(task)
	return args.Error(0)
}

func (m *MockTaskHandler) GetSkill(id string) (*a2a.AgentSkill, error) {
	args := m.Called(id)
	return args.Get(0).(*a2a.AgentSkill), args.Error(1)
}

func TestNewServer(t *testing.T) {
	agentCard := &a2a.AgentCard{
		A2AVersion: "1.0",
		ID:         "test-agent",
		Name:       "Test Agent",
	}
	tests := []struct {
		name        string
		opts        []server.Option
		expectedErr bool
	}{
		{
			name:        "Valid configuration",
			opts:        []server.Option{server.WithAgentCard(agentCard)},
			expectedErr: false,
		},
		{
			name:        "Missing AgentCard",
			opts:        []server.Option{},
			expectedErr: true,
		},
		{
			name:        "Valid Configuration with gollm options",
			opts:        []server.Option{server.WithAgentCard(agentCard), server.WithGollmOptions(&gollm.Options{Provider: "openai"})},
			expectedErr: false,
		},
		{
			name:        "Missing gollm options",
			opts:        []server.Option{server.WithAgentCard(agentCard)},
			expectedErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := server.NewServer(tt.opts...)
			if tt.expectedErr {
				assert.Error(t, err)
			}