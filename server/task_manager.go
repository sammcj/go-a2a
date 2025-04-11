package server

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/sammcj/go-a2a/a2a"
)

// TaskContext provides context to the TaskHandler.
type TaskContext struct {
	Task        a2a.Task     // Snapshot of task state (pass a copy)
	UserMessage a2a.Message  // Triggering message
	History     []a2a.Message // Snapshot of history (pass a copy)
	// Cancellation is checked via the context.Context passed to the handler
}

// TaskYieldUpdate represents status or artifact updates yielded by a handler.
type TaskYieldUpdate interface {
	isTaskYieldUpdate() // Marker method
}

// StatusUpdate represents a status change yielded by the handler.
// The server will add the timestamp.
type StatusUpdate struct {
	State   a2a.TaskState
	Message *a2a.Message // Optional agent message accompanying the status
}

func (s StatusUpdate) isTaskYieldUpdate() {}

// ArtifactUpdate represents an artifact yielded by the handler.
// The server will add the timestamp and ID.
type ArtifactUpdate struct {
	Part     a2a.Part
	Metadata interface{}
}

func (a ArtifactUpdate) isTaskYieldUpdate() {}

// TaskHandler defines the function signature for application-specific task execution logic.
// It's invoked by the TaskManager implementation.
// It receives context and returns a channel for yielding updates and an error.
// Closing the channel indicates completion.
type TaskHandler func(ctx context.Context, taskContext TaskContext) (<-chan TaskYieldUpdate, error)

// TaskManager defines the interface for task management operations.
type TaskManager interface {
	// Handles non-streaming task send/resume.
	OnSendTask(ctx context.Context, params *a2a.TaskSendParams) (*a2a.Task, error)

	// Handles streaming task send/resume. Returns a channel for updates.
	OnSendTaskSubscribe(ctx context.Context, params *a2a.TaskSendParams) (<-chan TaskYieldUpdate, error)

	// Handles task retrieval.
	OnGetTask(ctx context.Context, params *a2a.TaskQueryParams) (*a2a.Task, error)

	// Handles task cancellation.
	OnCancelTask(ctx context.Context, params *a2a.TaskIdParams) (*a2a.Task, error)

	// Handles setting push notification config.
	OnSetTaskPushNotification(ctx context.Context, params *a2a.TaskPushNotificationConfigParams) (*a2a.PushNotificationConfig, error)

	// Handles getting push notification config.
	OnGetTaskPushNotification(ctx context.Context, params *a2a.TaskIdParams) (*a2a.PushNotificationConfig, error)

	// Handles resubscribing to a task stream.
	OnResubscribeToTask(ctx context.Context, params *a2a.TaskIdParams) (<-chan TaskYieldUpdate, error)

	// (Potentially other internal methods for state management)
}

// InMemoryTaskManager is a basic implementation of TaskManager that stores tasks in memory.
type InMemoryTaskManager struct {
	tasks       map[string]*a2a.Task                      // Map of task ID to task
	pushConfigs map[string]*a2a.PushNotificationConfig    // Map of task ID to push notification config
	taskHandler TaskHandler                               // Application-specific task handler
	mu          sync.RWMutex                              // Mutex for thread safety
	// TODO: Add fields for SSE connections, active tasks, etc.
}

// NewInMemoryTaskManager creates a new InMemoryTaskManager.
func NewInMemoryTaskManager(handler TaskHandler) *InMemoryTaskManager {
	if handler == nil {
		// This should be caught earlier, but just in case
		panic("TaskHandler is required for InMemoryTaskManager")
	}

	return &InMemoryTaskManager{
		tasks:       make(map[string]*a2a.Task),
		pushConfigs: make(map[string]*a2a.PushNotificationConfig),
		taskHandler: handler,
	}
}

// OnSendTask implements TaskManager.OnSendTask.
func (tm *InMemoryTaskManager) OnSendTask(ctx context.Context, params *a2a.TaskSendParams) (*a2a.Task, error) {
	// TODO: Implement full task handling logic
	// For now, just a placeholder implementation

	// Check if this is a resume (taskId provided)
	if params.TaskID != nil {
		tm.mu.RLock()
		task, exists := tm.tasks[*params.TaskID]
		tm.mu.RUnlock()

		if !exists {
			return nil, a2a.ErrTaskNotFound(*params.TaskID)
		}

		// TODO: Validate session ID if provided
		// TODO: Handle task resumption logic
		return task, nil
	}

	// Create a new task
	taskID := generateTaskID() // TODO: Implement this helper
	now := time.Now()

	task := &a2a.Task{
		ID:        taskID,
		SessionID: params.SessionID, // May be nil
		Status: a2a.TaskStatus{
			State:     a2a.TaskStateSubmitted,
			Timestamp: now,
		},
		History:   []a2a.Message{params.Message}, // Start with the user message
		Artifacts: []a2a.Artifact{},              // Empty initially
	}

	// Store the task
	tm.mu.Lock()
	tm.tasks[taskID] = task
	tm.mu.Unlock()

	// TODO: Execute the task handler in a goroutine
	// TODO: Handle task state transitions
	// TODO: Send push notifications if configured

	return task, nil
}

// OnSendTaskSubscribe implements TaskManager.OnSendTaskSubscribe.
func (tm *InMemoryTaskManager) OnSendTaskSubscribe(ctx context.Context, params *a2a.TaskSendParams) (<-chan TaskYieldUpdate, error) {
	// TODO: Implement streaming task handling
	return nil, a2a.ErrOperationNotSupported("streaming tasks")
}

// OnGetTask implements TaskManager.OnGetTask.
func (tm *InMemoryTaskManager) OnGetTask(ctx context.Context, params *a2a.TaskQueryParams) (*a2a.Task, error) {
	tm.mu.RLock()
	task, exists := tm.tasks[params.TaskID]
	tm.mu.RUnlock()

	if !exists {
		return nil, a2a.ErrTaskNotFound(params.TaskID)
	}

	return task, nil
}

// OnCancelTask implements TaskManager.OnCancelTask.
func (tm *InMemoryTaskManager) OnCancelTask(ctx context.Context, params *a2a.TaskIdParams) (*a2a.Task, error) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	task, exists := tm.tasks[params.TaskID]
	if !exists {
		return nil, a2a.ErrTaskNotFound(params.TaskID)
	}

	// Update task status to cancelled
	task.Status = a2a.TaskStatus{
		State:     a2a.TaskStateCancelled,
		Timestamp: time.Now(),
		// Optionally add a system message about cancellation
	}

	// TODO: Cancel any active goroutines for this task
	// TODO: Send push notification if configured

	return task, nil
}

// OnSetTaskPushNotification implements TaskManager.OnSetTaskPushNotification.
func (tm *InMemoryTaskManager) OnSetTaskPushNotification(ctx context.Context, params *a2a.TaskPushNotificationConfigParams) (*a2a.PushNotificationConfig, error) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	// Check if task exists
	if _, exists := tm.tasks[params.TaskID]; !exists {
		return nil, a2a.ErrTaskNotFound(params.TaskID)
	}

	// Create push notification config
	config := &a2a.PushNotificationConfig{
		TaskID:           params.TaskID,
		URL:              params.URL,
		Authentication:   params.Authentication,
		IncludeTaskData:  params.IncludeTaskData,
		IncludeArtifacts: params.IncludeArtifacts,
	}

	// Store the config
	tm.pushConfigs[params.TaskID] = config

	return config, nil
}

// OnGetTaskPushNotification implements TaskManager.OnGetTaskPushNotification.
func (tm *InMemoryTaskManager) OnGetTaskPushNotification(ctx context.Context, params *a2a.TaskIdParams) (*a2a.PushNotificationConfig, error) {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	// Check if task exists
	if _, exists := tm.tasks[params.TaskID]; !exists {
		return nil, a2a.ErrTaskNotFound(params.TaskID)
	}

	// Get push notification config
	config, exists := tm.pushConfigs[params.TaskID]
	if !exists {
		return nil, fmt.Errorf("no push notification configuration for task %s", params.TaskID)
	}

	return config, nil
}

// OnResubscribeToTask implements TaskManager.OnResubscribeToTask.
func (tm *InMemoryTaskManager) OnResubscribeToTask(ctx context.Context, params *a2a.TaskIdParams) (<-chan TaskYieldUpdate, error) {
	// TODO: Implement resubscription logic
	return nil, a2a.ErrOperationNotSupported("task resubscription")
}

// --- Helper Functions ---

// generateTaskID generates a unique task ID.
// TODO: Implement a proper ID generation strategy.
func generateTaskID() string {
	return fmt.Sprintf("task_%d", time.Now().UnixNano())
}
