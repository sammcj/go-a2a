package test

import (
	"context"
	"testing"

	"github.com/sammcj/go-a2a/a2a"
)

// MockClient is a mock implementation of a client for testing.
type MockClient struct {
	Tasks map[string]*a2a.Task
}

// NewMockClient creates a new MockClient.
func NewMockClient() *MockClient {
	return &MockClient{
		Tasks: make(map[string]*a2a.Task),
	}
}

// SendTask mocks sending a task.
func (c *MockClient) SendTask(ctx context.Context, params *a2a.TaskSendParams) (*a2a.Task, error) {
	task := &a2a.Task{
		ID: "mock-task-id",
		Status: a2a.TaskStatus{
			State: a2a.TaskStateWorking,
		},
	}
	c.Tasks[task.ID] = task
	return task, nil
}

// GetTask mocks getting a task.
func (c *MockClient) GetTask(ctx context.Context, taskID string) (*a2a.Task, error) {
	task, ok := c.Tasks[taskID]
	if !ok {
		return nil, nil
	}
	return task, nil
}

// TestMockClient tests the mock client.
func TestMockClient(t *testing.T) {
	// Create a mock client
	client := NewMockClient()

	// Create a message
	message := a2a.Message{
		Role: a2a.RoleUser,
		Parts: []a2a.Part{
			a2a.TextPart{
				Type: "text",
				Text: "Hello, world!",
			},
		},
	}

	// Send a task
	task, err := client.SendTask(context.Background(), &a2a.TaskSendParams{
		Message: message,
	})
	if err != nil {
		t.Fatalf("Failed to send task: %v", err)
	}

	// Check the task
	if task.ID == "" {
		t.Errorf("Expected task ID to be non-empty")
	}
	if task.Status.State != a2a.TaskStateWorking {
		t.Errorf("Expected task state to be 'working', got '%s'", task.Status.State)
	}

	// Update the task
	task.Status.State = a2a.TaskStateCompleted
	task.Status.Message = &a2a.Message{
		Role: a2a.RoleAgent,
		Parts: []a2a.Part{
			a2a.TextPart{
				Type: "text",
				Text: "Echo: Hello, world!",
			},
		},
	}
	client.Tasks[task.ID] = task

	// Get the task
	task, err = client.GetTask(context.Background(), task.ID)
	if err != nil {
		t.Fatalf("Failed to get task: %v", err)
	}

	// Check the task
	if task.Status.State != a2a.TaskStateCompleted {
		t.Errorf("Expected task state to be 'completed', got '%s'", task.Status.State)
	}
	if task.Status.Message == nil {
		t.Fatalf("Expected task message to be non-nil")
	}
	if len(task.Status.Message.Parts) != 1 {
		t.Fatalf("Expected task message to have one part")
	}
	textPart, ok := task.Status.Message.Parts[0].(a2a.TextPart)
	if !ok {
		t.Fatalf("Expected task message part to be a TextPart")
	}
	if textPart.Text != "Echo: Hello, world!" {
		t.Errorf("Expected task message text to be 'Echo: Hello, world!', got '%s'", textPart.Text)
	}
}
