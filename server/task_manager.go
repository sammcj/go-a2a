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
	pushNotifier *PushNotifier                            // Push notification sender
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
		tasks:        make(map[string]*a2a.Task),
		pushConfigs:  make(map[string]*a2a.PushNotificationConfig),
		taskHandler:  handler,
		pushNotifier: NewPushNotifier(10 * time.Second), // Default 10 second timeout
	}
}

// OnSendTask implements TaskManager.OnSendTask.
func (tm *InMemoryTaskManager) OnSendTask(ctx context.Context, params *a2a.TaskSendParams) (*a2a.Task, error) {
	// Check if this is a resume (taskId provided)
	if params.TaskID != nil {
		tm.mu.RLock()
		task, exists := tm.tasks[*params.TaskID]
		tm.mu.RUnlock()

		if !exists {
			return nil, a2a.ErrTaskNotFound(*params.TaskID)
		}

		// TODO: Validate session ID if provided

		// Create a task context
		taskCtx := TaskContext{
			Task:        *task,
			UserMessage: params.Message,
			History:     append(task.History, params.Message),
		}

		// Start a goroutine to handle the task
		go func() {
			// Call the task handler
			handlerUpdateChan, err := tm.taskHandler(ctx, taskCtx)
			if err != nil {
				// Update task status to failed
				tm.mu.Lock()
				task.Status = a2a.TaskStatus{
					State:     a2a.TaskStateFailed,
					Timestamp: time.Now(),
					Message: &a2a.Message{
						Role:      a2a.RoleSystem,
						Timestamp: time.Now(),
						Parts: []a2a.Part{
							a2a.TextPart{
								Type: "text",
								Text: fmt.Sprintf("Task failed: %v", err),
							},
						},
					},
				}

				// Get push notification config (if any)
				config, hasPushConfig := tm.pushConfigs[*params.TaskID]
				tm.mu.Unlock()

				// Send push notification if configured
				if hasPushConfig && tm.pushNotifier != nil {
					if err := tm.pushNotifier.SendStatusUpdate(context.Background(), task, config); err != nil {
						// Just log the error for now
						fmt.Printf("Failed to send push notification for task %s: %v\n", *params.TaskID, err)
					}
				}

				return
			}

			// Process updates from the handler
			for update := range handlerUpdateChan {
				// Update task state in memory and send push notifications if configured
				switch u := update.(type) {
				case StatusUpdate:
					tm.mu.Lock()
					task.Status = a2a.TaskStatus{
						State:     u.State,
						Timestamp: time.Now(),
						Message:   u.Message,
					}
					if u.Message != nil {
						task.History = append(task.History, *u.Message)
					}

					// Get push notification config (if any)
					config, hasPushConfig := tm.pushConfigs[*params.TaskID]
					tm.mu.Unlock()

					// Send push notification if configured
					if hasPushConfig && tm.pushNotifier != nil {
						if err := tm.pushNotifier.SendStatusUpdate(context.Background(), task, config); err != nil {
							// Just log the error for now
							fmt.Printf("Failed to send push notification for task %s: %v\n", *params.TaskID, err)
						}
					}

				case ArtifactUpdate:
					artifact := a2a.Artifact{
						ID:        fmt.Sprintf("artifact_%d", time.Now().UnixNano()),
						TaskID:    *params.TaskID,
						Timestamp: time.Now(),
						Part:      u.Part,
						Metadata:  u.Metadata,
					}

					tm.mu.Lock()
					task.Artifacts = append(task.Artifacts, artifact)

					// Get push notification config (if any)
					config, hasPushConfig := tm.pushConfigs[*params.TaskID]
					tm.mu.Unlock()

					// Send push notification if configured
					if hasPushConfig && tm.pushNotifier != nil {
						if err := tm.pushNotifier.SendArtifactUpdate(context.Background(), task, artifact, config); err != nil {
							// Just log the error for now
							fmt.Printf("Failed to send push notification for artifact %s: %v\n", artifact.ID, err)
						}
					}
				}
			}
		}()

		return task, nil
	}

	// Create a new task
	taskID := generateTaskID()
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

	// Create a task context
	taskCtx := TaskContext{
		Task:        *task,
		UserMessage: params.Message,
		History:     task.History,
	}

	// Start a goroutine to handle the task
	go func() {
		// Update task status to working
		tm.mu.Lock()
		task.Status = a2a.TaskStatus{
			State:     a2a.TaskStateWorking,
			Timestamp: time.Now(),
		}
		tm.mu.Unlock()

		// Call the task handler
		handlerUpdateChan, err := tm.taskHandler(ctx, taskCtx)
		if err != nil {
			// Update task status to failed
			tm.mu.Lock()
			task.Status = a2a.TaskStatus{
				State:     a2a.TaskStateFailed,
				Timestamp: time.Now(),
				Message: &a2a.Message{
					Role:      a2a.RoleSystem,
					Timestamp: time.Now(),
					Parts: []a2a.Part{
						a2a.TextPart{
							Type: "text",
							Text: fmt.Sprintf("Task failed: %v", err),
						},
					},
				},
			}

			// Get push notification config (if any)
			config, hasPushConfig := tm.pushConfigs[taskID]
			tm.mu.Unlock()

			// Send push notification if configured
			if hasPushConfig && tm.pushNotifier != nil {
				if err := tm.pushNotifier.SendStatusUpdate(context.Background(), task, config); err != nil {
					// Just log the error for now
					fmt.Printf("Failed to send push notification for task %s: %v\n", taskID, err)
				}
			}

			return
		}

		// Process updates from the handler
		for update := range handlerUpdateChan {
			// Update task state in memory and send push notifications if configured
			switch u := update.(type) {
			case StatusUpdate:
				tm.mu.Lock()
				task.Status = a2a.TaskStatus{
					State:     u.State,
					Timestamp: time.Now(),
					Message:   u.Message,
				}
				if u.Message != nil {
					task.History = append(task.History, *u.Message)
				}

				// Get push notification config (if any)
				config, hasPushConfig := tm.pushConfigs[taskID]
				tm.mu.Unlock()

				// Send push notification if configured
				if hasPushConfig && tm.pushNotifier != nil {
					if err := tm.pushNotifier.SendStatusUpdate(context.Background(), task, config); err != nil {
						// Just log the error for now
						fmt.Printf("Failed to send push notification for task %s: %v\n", taskID, err)
					}
				}

			case ArtifactUpdate:
				artifact := a2a.Artifact{
					ID:        fmt.Sprintf("artifact_%d", time.Now().UnixNano()),
					TaskID:    taskID,
					Timestamp: time.Now(),
					Part:      u.Part,
					Metadata:  u.Metadata,
				}

				tm.mu.Lock()
				task.Artifacts = append(task.Artifacts, artifact)

				// Get push notification config (if any)
				config, hasPushConfig := tm.pushConfigs[taskID]
				tm.mu.Unlock()

				// Send push notification if configured
				if hasPushConfig && tm.pushNotifier != nil {
					if err := tm.pushNotifier.SendArtifactUpdate(context.Background(), task, artifact, config); err != nil {
						// Just log the error for now
						fmt.Printf("Failed to send push notification for artifact %s: %v\n", artifact.ID, err)
					}
				}
			}
		}
	}()

	return task, nil
}

// OnSendTaskSubscribe implements TaskManager.OnSendTaskSubscribe.
func (tm *InMemoryTaskManager) OnSendTaskSubscribe(ctx context.Context, params *a2a.TaskSendParams) (<-chan TaskYieldUpdate, error) {
	// Create a channel for updates
	updateChan := make(chan TaskYieldUpdate)

	// Check if this is a resume (taskId provided)
	if params.TaskID != nil {
		tm.mu.RLock()
		task, exists := tm.tasks[*params.TaskID]
		tm.mu.RUnlock()

		if !exists {
			return nil, a2a.ErrTaskNotFound(*params.TaskID)
		}

		// TODO: Validate session ID if provided

		// Start a goroutine to handle the task
		go func() {
			defer close(updateChan)

			// Send the current status as the first update
			updateChan <- StatusUpdate{
				State:   task.Status.State,
				Message: task.Status.Message,
			}

			// Create a task context
			taskCtx := TaskContext{
				Task:        *task,
				UserMessage: params.Message,
				History:     append(task.History, params.Message),
			}

			// Call the task handler
			handlerUpdateChan, err := tm.taskHandler(ctx, taskCtx)
			if err != nil {
				// Update task status to failed
				tm.mu.Lock()
				task.Status = a2a.TaskStatus{
					State:     a2a.TaskStateFailed,
					Timestamp: time.Now(),
					Message: &a2a.Message{
						Role:      a2a.RoleSystem,
						Timestamp: time.Now(),
						Parts: []a2a.Part{
							a2a.TextPart{
								Type: "text",
								Text: fmt.Sprintf("Task failed: %v", err),
							},
						},
					},
				}
				tm.mu.Unlock()

				// Send a failed status update
				updateChan <- StatusUpdate{
					State: a2a.TaskStateFailed,
					Message: &a2a.Message{
						Role:      a2a.RoleSystem,
						Timestamp: time.Now(),
						Parts: []a2a.Part{
							a2a.TextPart{
								Type: "text",
								Text: fmt.Sprintf("Task failed: %v", err),
							},
						},
					},
				}
				return
			}

			// Forward updates from the handler
			for update := range handlerUpdateChan {
				// Update task state in memory and send push notifications if configured
				switch u := update.(type) {
				case StatusUpdate:
					tm.mu.Lock()
					task.Status = a2a.TaskStatus{
						State:     u.State,
						Timestamp: time.Now(),
						Message:   u.Message,
					}
					if u.Message != nil {
						task.History = append(task.History, *u.Message)
					}

					// Get push notification config (if any)
					config, hasPushConfig := tm.pushConfigs[*params.TaskID]
					tm.mu.Unlock()

					// Send push notification if configured
					if hasPushConfig && tm.pushNotifier != nil {
						// Send in a goroutine to avoid blocking
						go func() {
							if err := tm.pushNotifier.SendStatusUpdate(context.Background(), task, config); err != nil {
								// Just log the error for now
								fmt.Printf("Failed to send push notification for task %s: %v\n", *params.TaskID, err)
							}
						}()
					}

				case ArtifactUpdate:
					artifact := a2a.Artifact{
						ID:        fmt.Sprintf("artifact_%d", time.Now().UnixNano()),
						TaskID:    *params.TaskID,
						Timestamp: time.Now(),
						Part:      u.Part,
						Metadata:  u.Metadata,
					}

					tm.mu.Lock()
					task.Artifacts = append(task.Artifacts, artifact)

					// Get push notification config (if any)
					config, hasPushConfig := tm.pushConfigs[*params.TaskID]
					tm.mu.Unlock()

					// Send push notification if configured
					if hasPushConfig && tm.pushNotifier != nil {
						// Send in a goroutine to avoid blocking
						go func(a a2a.Artifact) {
							if err := tm.pushNotifier.SendArtifactUpdate(context.Background(), task, a, config); err != nil {
								// Just log the error for now
								fmt.Printf("Failed to send push notification for artifact %s: %v\n", a.ID, err)
							}
						}(artifact)
					}
				}

				// Forward the update
				updateChan <- update
			}
		}()

		return updateChan, nil
	}

	// Create a new task
	taskID := generateTaskID()
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

	// Start a goroutine to handle the task
	go func() {
		defer close(updateChan)

		// Send the initial status update
		updateChan <- StatusUpdate{
			State: a2a.TaskStateSubmitted,
		}

		// Update task status to working
		tm.mu.Lock()
		task.Status = a2a.TaskStatus{
			State:     a2a.TaskStateWorking,
			Timestamp: time.Now(),
		}
		tm.mu.Unlock()

		// Send a working status update
		updateChan <- StatusUpdate{
			State: a2a.TaskStateWorking,
		}

		// Create a task context
		taskCtx := TaskContext{
			Task:        *task,
			UserMessage: params.Message,
			History:     task.History,
		}

		// Call the task handler
		handlerUpdateChan, err := tm.taskHandler(ctx, taskCtx)
		if err != nil {
			// Update task status to failed
			tm.mu.Lock()
			task.Status = a2a.TaskStatus{
				State:     a2a.TaskStateFailed,
				Timestamp: time.Now(),
				Message: &a2a.Message{
					Role:      a2a.RoleSystem,
					Timestamp: time.Now(),
					Parts: []a2a.Part{
						a2a.TextPart{
							Type: "text",
							Text: fmt.Sprintf("Task failed: %v", err),
						},
					},
				},
			}
			tm.mu.Unlock()

			// Send a failed status update
			updateChan <- StatusUpdate{
				State: a2a.TaskStateFailed,
				Message: &a2a.Message{
					Role:      a2a.RoleSystem,
					Timestamp: time.Now(),
					Parts: []a2a.Part{
						a2a.TextPart{
							Type: "text",
							Text: fmt.Sprintf("Task failed: %v", err),
						},
					},
				},
			}
			return
		}

		// Forward updates from the handler
		for update := range handlerUpdateChan {
			// Update task state in memory
			switch u := update.(type) {
			case StatusUpdate:
				tm.mu.Lock()
				task.Status = a2a.TaskStatus{
					State:     u.State,
					Timestamp: time.Now(),
					Message:   u.Message,
				}
				if u.Message != nil {
					task.History = append(task.History, *u.Message)
				}
				tm.mu.Unlock()
			case ArtifactUpdate:
				artifact := a2a.Artifact{
					ID:        fmt.Sprintf("artifact_%d", time.Now().UnixNano()),
					TaskID:    taskID,
					Timestamp: time.Now(),
					Part:      u.Part,
					Metadata:  u.Metadata,
				}
				tm.mu.Lock()
				task.Artifacts = append(task.Artifacts, artifact)
				tm.mu.Unlock()
			}

			// Forward the update
			updateChan <- update
		}
	}()

	return updateChan, nil
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

	task, exists := tm.tasks[params.TaskID]
	if !exists {
		tm.mu.Unlock()
		return nil, a2a.ErrTaskNotFound(params.TaskID)
	}

	// Update task status to cancelled
	task.Status = a2a.TaskStatus{
		State:     a2a.TaskStateCancelled,
		Timestamp: time.Now(),
		Message: &a2a.Message{
			Role:      a2a.RoleSystem,
			Timestamp: time.Now(),
			Parts: []a2a.Part{
				a2a.TextPart{
					Type: "text",
					Text: "Task cancelled by client",
				},
			},
		},
	}

	// Get push notification config (if any)
	config, hasPushConfig := tm.pushConfigs[params.TaskID]
	tm.mu.Unlock()

	// Send push notification if configured
	if hasPushConfig && tm.pushNotifier != nil {
		// Send in a goroutine to avoid blocking
		go func() {
			if err := tm.pushNotifier.SendStatusUpdate(context.Background(), task, config); err != nil {
				// Just log the error for now
				fmt.Printf("Failed to send push notification for cancelled task %s: %v\n", params.TaskID, err)
			}
		}()
	}

	// TODO: Cancel any active goroutines for this task

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
	// Create a channel for updates
	updateChan := make(chan TaskYieldUpdate)

	// Check if the task exists
	tm.mu.RLock()
	task, exists := tm.tasks[params.TaskID]
	tm.mu.RUnlock()

	if !exists {
		return nil, a2a.ErrTaskNotFound(params.TaskID)
	}

	// Start a goroutine to handle the resubscription
	go func() {
		defer close(updateChan)

		// Send the current status as the first update
		updateChan <- StatusUpdate{
			State:   task.Status.State,
			Message: task.Status.Message,
		}

		// If the task is already completed, failed, or cancelled, just return
		if task.Status.State == a2a.TaskStateCompleted ||
			task.Status.State == a2a.TaskStateFailed ||
			task.Status.State == a2a.TaskStateCancelled {
			return
		}

		// For tasks that are still in progress, we need to monitor them
		// This is a simplified implementation that just waits for the task to complete
		// In a real implementation, we would need to hook into the task's execution
		ticker := time.NewTicker(500 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				// Context cancelled, stop monitoring
				return
			case <-ticker.C:
				// Check the task status
				tm.mu.RLock()
				currentStatus := task.Status
				tm.mu.RUnlock()

				// Send an update if the status has changed
				updateChan <- StatusUpdate{
					State:   currentStatus.State,
					Message: currentStatus.Message,
				}

				// If the task is now completed, failed, or cancelled, stop monitoring
				if currentStatus.State == a2a.TaskStateCompleted ||
					currentStatus.State == a2a.TaskStateFailed ||
					currentStatus.State == a2a.TaskStateCancelled {
					return
				}
			}
		}
	}()

	return updateChan, nil
}

// --- Helper Functions ---

// generateTaskID generates a unique task ID.
// TODO: Implement a proper ID generation strategy.
func generateTaskID() string {
	return fmt.Sprintf("task_%d", time.Now().UnixNano())
}
