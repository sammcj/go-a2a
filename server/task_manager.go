package server

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/sammcj/go-a2a/a2a"
	"github.com/sammcj/go-a2a/pkg/task"
)

// TaskManager defines the interface for task management operations.
type TaskManager interface {
	// Handles non-streaming task send/resume.
	OnSendTask(ctx context.Context, params *a2a.TaskSendParams) (*a2a.Task, error)

	// Handles streaming task send/resume. Returns a channel for updates.
	OnSendTaskSubscribe(ctx context.Context, params *a2a.TaskSendParams) (<-chan task.YieldUpdate, error)

	// Handles task retrieval.
	OnGetTask(ctx context.Context, params *a2a.TaskQueryParams) (*a2a.Task, error)

	// Handles task cancellation.
	OnCancelTask(ctx context.Context, params *a2a.TaskIdParams) (*a2a.Task, error)

	// Handles setting push notification config.
	OnSetTaskPushNotification(ctx context.Context, params *a2a.TaskPushNotificationConfigParams) (*a2a.PushNotificationConfig, error)

	// Handles getting push notification config.
	OnGetTaskPushNotification(ctx context.Context, params *a2a.TaskIdParams) (*a2a.PushNotificationConfig, error)

	// Handles resubscribing to a task stream.
	OnResubscribeToTask(ctx context.Context, params *a2a.TaskIdParams) (<-chan task.YieldUpdate, error)

	// (Potentially other internal methods for state management)
}

// InMemoryTaskManager is a basic implementation of TaskManager that stores tasks in memory.
type InMemoryTaskManager struct {
	tasks        map[string]*a2a.Task                   // Map of task ID to task
	pushConfigs  map[string]*a2a.PushNotificationConfig // Map of task ID to push notification config
	taskHandler  task.Handler                           // Application-specific task handler
	pushNotifier *PushNotifier                          // Push notification sender
	expiry       time.Duration                          // Task expiry duration
	mu           sync.RWMutex                           // Mutex for thread safety
}

// CreateTask creates a new task and returns its ID.
func (tm *InMemoryTaskManager) CreateTask(ctx context.Context, taskType string, params interface{}) (string, error) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	id := generateTaskID()
	now := time.Now()

	task := &a2a.Task{
		ID: id,
		Status: a2a.TaskStatus{
			State:     a2a.TaskStateSubmitted,
			Timestamp: now,
		},
		History:   []a2a.Message{},
		Artifacts: []a2a.Artifact{},
	}

	tm.tasks[id] = task
	return id, nil
}

// GetTask retrieves a task by ID.
func (tm *InMemoryTaskManager) GetTask(ctx context.Context, id string) (*a2a.Task, error) {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	task, exists := tm.tasks[id]
	if !exists {
		return nil, fmt.Errorf("task not found: %s", id)
	}

	return task, nil
}

// UpdateTask updates a task's status.
func (tm *InMemoryTaskManager) UpdateTask(ctx context.Context, id string, status a2a.TaskState, message *a2a.Message, err error) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	task, exists := tm.tasks[id]
	if !exists {
		return fmt.Errorf("task not found: %s", id)
	}

	task.Status = a2a.TaskStatus{
		State:     status,
		Timestamp: time.Now(),
		Message:   message,
	}

	if message != nil {
		var msg a2a.Message = *message
		task.History = append(task.History, msg)
	}

	return nil
}

// DeleteTask deletes a task.
func (tm *InMemoryTaskManager) DeleteTask(ctx context.Context, id string) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	if _, exists := tm.tasks[id]; !exists {
		return fmt.Errorf("task not found: %s", id)
	}

	delete(tm.tasks, id)
	return nil
}

// ListTasks returns all tasks.
func (tm *InMemoryTaskManager) ListTasks(ctx context.Context) ([]*a2a.Task, error) {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	tasks := make([]*a2a.Task, 0, len(tm.tasks))
	for _, task := range tm.tasks {
		// Check if task is expired
		if !time.Now().After(task.Status.Timestamp.Add(tm.expiry)) {
			tasks = append(tasks, task)
		}
	}

	return tasks, nil
}

// SetTaskExpiry sets the expiry duration for tasks.
func (tm *InMemoryTaskManager) SetTaskExpiry(duration time.Duration) {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	tm.expiry = duration
}

// NewInMemoryTaskManager creates a new InMemoryTaskManager.
func NewInMemoryTaskManager(handler task.Handler) *InMemoryTaskManager {
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
		existingTask, exists := tm.tasks[*params.TaskID]
		tm.mu.RUnlock()

		if !exists {
			return nil, a2a.ErrTaskNotFound(*params.TaskID)
		}

		// TODO: Validate session ID if provided

		// Create a task context
		taskCtx := task.Context{
			TaskID:      *params.TaskID,
			UserMessage: params.Message,
		}

		// Start a goroutine to handle the task
		go func() {
			// Call the task handler
			handlerUpdateChan, err := tm.taskHandler(ctx, taskCtx)
			if err != nil {
				// Update task status to failed
				tm.mu.Lock()
				existingTask.Status = a2a.TaskStatus{
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
					if err := tm.pushNotifier.SendStatusUpdate(context.Background(), existingTask, config); err != nil {
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
				case task.StatusUpdate:
					tm.mu.Lock()
					existingTask.Status = a2a.TaskStatus{
						State:     u.State,
						Timestamp: time.Now(),
						Message:   u.Message,
					}
					if u.Message != nil {
						existingTask.History = append(existingTask.History, *u.Message)
					}

					// Get push notification config (if any)
					config, hasPushConfig := tm.pushConfigs[*params.TaskID]
					tm.mu.Unlock()

					// Send push notification if configured
					if hasPushConfig && tm.pushNotifier != nil {
						if err := tm.pushNotifier.SendStatusUpdate(context.Background(), existingTask, config); err != nil {
							// Just log the error for now
							fmt.Printf("Failed to send push notification for task %s: %v\n", *params.TaskID, err)
						}
					}

				case task.ArtifactUpdate:
					artifact := a2a.Artifact{
						ID:        fmt.Sprintf("artifact_%d", time.Now().UnixNano()),
						TaskID:    *params.TaskID,
						Timestamp: time.Now(),
						Part:      u.Part,
						Metadata:  u.Metadata,
					}

					tm.mu.Lock()
					existingTask.Artifacts = append(existingTask.Artifacts, artifact)

					// Get push notification config (if any)
					config, hasPushConfig := tm.pushConfigs[*params.TaskID]
					tm.mu.Unlock()

					// Send push notification if configured
					if hasPushConfig && tm.pushNotifier != nil {
						if err := tm.pushNotifier.SendArtifactUpdate(context.Background(), existingTask, artifact, config); err != nil {
							// Just log the error for now
							fmt.Printf("Failed to send push notification for artifact %s: %v\n", artifact.ID, err)
						}
					}
				}
			}
		}()

		return existingTask, nil
	}

	// Create a new task
	taskID := generateTaskID()
	now := time.Now()

	newTask := &a2a.Task{
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
	tm.tasks[taskID] = newTask
	tm.mu.Unlock()

	// Create a task context
	taskCtx := task.Context{
		TaskID:      taskID,
		UserMessage: params.Message,
	}

	// Start a goroutine to handle the task
	go func() {
		// Update task status to working
		tm.mu.Lock()
		newTask.Status = a2a.TaskStatus{
			State:     a2a.TaskStateWorking,
			Timestamp: time.Now(),
		}
		tm.mu.Unlock()

		// Call the task handler
		handlerUpdateChan, err := tm.taskHandler(ctx, taskCtx)
		if err != nil {
			// Update task status to failed
			tm.mu.Lock()
			newTask.Status = a2a.TaskStatus{
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
				if err := tm.pushNotifier.SendStatusUpdate(context.Background(), newTask, config); err != nil {
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
			case task.StatusUpdate:
				tm.mu.Lock()
				newTask.Status = a2a.TaskStatus{
					State:     u.State,
					Timestamp: time.Now(),
					Message:   u.Message,
				}
				if u.Message != nil {
					newTask.History = append(newTask.History, *u.Message)
				}

				// Get push notification config (if any)
				config, hasPushConfig := tm.pushConfigs[taskID]
				tm.mu.Unlock()

				// Send push notification if configured
				if hasPushConfig && tm.pushNotifier != nil {
					if err := tm.pushNotifier.SendStatusUpdate(context.Background(), newTask, config); err != nil {
						// Just log the error for now
						fmt.Printf("Failed to send push notification for task %s: %v\n", taskID, err)
					}
				}

			case task.ArtifactUpdate:
				artifact := a2a.Artifact{
					ID:        fmt.Sprintf("artifact_%d", time.Now().UnixNano()),
					TaskID:    taskID,
					Timestamp: time.Now(),
					Part:      u.Part,
					Metadata:  u.Metadata,
				}

				tm.mu.Lock()
				newTask.Artifacts = append(newTask.Artifacts, artifact)

				// Get push notification config (if any)
				config, hasPushConfig := tm.pushConfigs[taskID]
				tm.mu.Unlock()

				// Send push notification if configured
				if hasPushConfig && tm.pushNotifier != nil {
					if err := tm.pushNotifier.SendArtifactUpdate(context.Background(), newTask, artifact, config); err != nil {
						// Just log the error for now
						fmt.Printf("Failed to send push notification for artifact %s: %v\n", artifact.ID, err)
					}
				}
			}
		}
	}()

	return newTask, nil
}

// OnSendTaskSubscribe implements TaskManager.OnSendTaskSubscribe.
func (tm *InMemoryTaskManager) OnSendTaskSubscribe(ctx context.Context, params *a2a.TaskSendParams) (<-chan task.YieldUpdate, error) {
	// Create a channel for updates
	updateChan := make(chan task.YieldUpdate)

	// Check if this is a resume (taskId provided)
	if params.TaskID != nil {
		tm.mu.RLock()
		taskObj, exists := tm.tasks[*params.TaskID]
		tm.mu.RUnlock()

		if !exists {
			return nil, a2a.ErrTaskNotFound(*params.TaskID)
		}

		// TODO: Validate session ID if provided

		// Start a goroutine to handle the task
		go func() {
			defer close(updateChan)

			// Send the current status as the first update
			updateChan <- task.StatusUpdate{
				State:   taskObj.Status.State,
				Message: taskObj.Status.Message,
			}

			// Create a task context
			taskCtx := task.Context{
				TaskID:      *params.TaskID,
				UserMessage: params.Message,
			}

			// Call the task handler
			handlerUpdateChan, err := tm.taskHandler(ctx, taskCtx)
			if err != nil {
				// Update task status to failed
				tm.mu.Lock()
				taskObj.Status = a2a.TaskStatus{
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
				updateChan <- task.StatusUpdate{
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
				case task.StatusUpdate:
					tm.mu.Lock()
					taskObj.Status = a2a.TaskStatus{
						State:     u.State,
						Timestamp: time.Now(),
						Message:   u.Message,
					}
					if u.Message != nil {
						taskObj.History = append(taskObj.History, *u.Message)
					}

					// Get push notification config (if any)
					config, hasPushConfig := tm.pushConfigs[*params.TaskID]
					tm.mu.Unlock()

					// Send push notification if configured
					if hasPushConfig && tm.pushNotifier != nil {
						// Send in a goroutine to avoid blocking
						go func() {
							if err := tm.pushNotifier.SendStatusUpdate(context.Background(), taskObj, config); err != nil {
								// Just log the error for now
								fmt.Printf("Failed to send push notification for task %s: %v\n", *params.TaskID, err)
							}
						}()
					}

				case task.ArtifactUpdate:
					artifact := a2a.Artifact{
						ID:        fmt.Sprintf("artifact_%d", time.Now().UnixNano()),
						TaskID:    *params.TaskID,
						Timestamp: time.Now(),
						Part:      u.Part,
						Metadata:  u.Metadata,
					}

					tm.mu.Lock()
					taskObj.Artifacts = append(taskObj.Artifacts, artifact)

					// Get push notification config (if any)
					config, hasPushConfig := tm.pushConfigs[*params.TaskID]
					tm.mu.Unlock()

					// Send push notification if configured
					if hasPushConfig && tm.pushNotifier != nil {
						// Send in a goroutine to avoid blocking
						go func(a a2a.Artifact) {
							if err := tm.pushNotifier.SendArtifactUpdate(context.Background(), taskObj, a, config); err != nil {
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

	taskObj := &a2a.Task{
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
	tm.tasks[taskID] = taskObj
	tm.mu.Unlock()

	// Start a goroutine to handle the task
	go func() {
		defer close(updateChan)

		// Send the initial status update
		updateChan <- task.StatusUpdate{
			State: a2a.TaskStateSubmitted,
		}

		// Update task status to working
		tm.mu.Lock()
		taskObj.Status = a2a.TaskStatus{
			State:     a2a.TaskStateWorking,
			Timestamp: time.Now(),
		}
		tm.mu.Unlock()

		// Send a working status update
		updateChan <- task.StatusUpdate{
			State: a2a.TaskStateWorking,
		}

		// Create a task context
		taskCtx := task.Context{
			TaskID:      taskID,
			UserMessage: params.Message,
		}

		// Call the task handler
		handlerUpdateChan, err := tm.taskHandler(ctx, taskCtx)
		if err != nil {
			// Update task status to failed
			tm.mu.Lock()
			taskObj.Status = a2a.TaskStatus{
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
			updateChan <- task.StatusUpdate{
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
			case task.StatusUpdate:
				tm.mu.Lock()
				taskObj.Status = a2a.TaskStatus{
					State:     u.State,
					Timestamp: time.Now(),
					Message:   u.Message,
				}
				if u.Message != nil {
					taskObj.History = append(taskObj.History, *u.Message)
				}
				tm.mu.Unlock()
			case task.ArtifactUpdate:
				artifact := a2a.Artifact{
					ID:        fmt.Sprintf("artifact_%d", time.Now().UnixNano()),
					TaskID:    taskID,
					Timestamp: time.Now(),
					Part:      u.Part,
					Metadata:  u.Metadata,
				}
				tm.mu.Lock()
				taskObj.Artifacts = append(taskObj.Artifacts, artifact)
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
	// Check if the task exists
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	taskObj, exists := tm.tasks[params.TaskID]
	if !exists {
		return nil, a2a.ErrTaskNotFound(params.TaskID)
	}

	return taskObj, nil
}

// OnSetTaskPushNotification implements TaskManager.OnSetTaskPushNotification.
func (tm *InMemoryTaskManager) OnSetTaskPushNotification(ctx context.Context, params *a2a.TaskPushNotificationConfigParams) (*a2a.PushNotificationConfig, error) {
	// Check if the task exists
	tm.mu.RLock()
	_, exists := tm.tasks[params.TaskID]
	tm.mu.RUnlock()

	if !exists {
		return nil, a2a.ErrTaskNotFound(params.TaskID)
	}

	// Create a push notification config from the params
	config := &a2a.PushNotificationConfig{
		TaskID:           params.TaskID,
		URL:              params.URL,
		Authentication:   params.Authentication,
		IncludeTaskData:  params.IncludeTaskData,
		IncludeArtifacts: params.IncludeArtifacts,
	}

	// Store the push notification config
	tm.mu.Lock()
	tm.pushConfigs[params.TaskID] = config
	tm.mu.Unlock()

	return config, nil
}

// OnGetTaskPushNotification implements TaskManager.OnGetTaskPushNotification.
func (tm *InMemoryTaskManager) OnGetTaskPushNotification(ctx context.Context, params *a2a.TaskIdParams) (*a2a.PushNotificationConfig, error) {
	// Check if the task exists
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	_, exists := tm.tasks[params.TaskID]
	if !exists {
		return nil, a2a.ErrTaskNotFound(params.TaskID)
	}

	// Get push notification config (if any)
	config, exists := tm.pushConfigs[params.TaskID]
	if !exists {
		return nil, fmt.Errorf("push notification config not found for task %s", params.TaskID)
	}

	return config, nil
}

// OnCancelTask implements TaskManager.OnCancelTask.
func (tm *InMemoryTaskManager) OnCancelTask(ctx context.Context, params *a2a.TaskIdParams) (*a2a.Task, error) {
	// Check if the task exists
	tm.mu.RLock()
	taskObj, exists := tm.tasks[params.TaskID]
	tm.mu.RUnlock()

	if !exists {
		return nil, a2a.ErrTaskNotFound(params.TaskID)
	}

	// Update task status to cancelled
	tm.mu.Lock()
	taskObj.Status = a2a.TaskStatus{
		State:     a2a.TaskStateCancelled,
		Timestamp: time.Now(),
		Message: &a2a.Message{
			Role:      a2a.RoleSystem,
			Timestamp: time.Now(),
			Parts: []a2a.Part{
				a2a.TextPart{
					Type: "text",
					Text: "Task cancelled by user",
				},
			},
		},
	}
	tm.mu.Unlock()

	// Get push notification config (if any)
	tm.mu.RLock()
	config, hasPushConfig := tm.pushConfigs[params.TaskID]
	tm.mu.RUnlock()

	// Send push notification if configured
	if hasPushConfig && tm.pushNotifier != nil {
		if err := tm.pushNotifier.SendStatusUpdate(context.Background(), taskObj, config); err != nil {
			// Just log the error for now
			fmt.Printf("Failed to send push notification for task %s: %v\n", params.TaskID, err)
		}
	}

	return taskObj, nil
}

// OnResubscribeToTask implements TaskManager.OnResubscribeToTask.
func (tm *InMemoryTaskManager) OnResubscribeToTask(ctx context.Context, params *a2a.TaskIdParams) (<-chan task.YieldUpdate, error) {
	// Create a channel for updates
	updateChan := make(chan task.YieldUpdate)

	// Check if the task exists
	tm.mu.RLock()
	taskObj, exists := tm.tasks[params.TaskID]
	tm.mu.RUnlock()

	if !exists {
		return nil, a2a.ErrTaskNotFound(params.TaskID)
	}

	// Start a goroutine to handle the resubscription
	go func() {
		defer close(updateChan)

		// Send the current status as the first update
		updateChan <- task.StatusUpdate{
			State:   taskObj.Status.State,
			Message: taskObj.Status.Message,
		}

		// If the task is already completed, failed, or cancelled, just return
		if taskObj.Status.State == a2a.TaskStateCompleted ||
			taskObj.Status.State == a2a.TaskStateFailed ||
			taskObj.Status.State == a2a.TaskStateCancelled {
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
				currentStatus := taskObj.Status
				tm.mu.RUnlock()

				// Send an update if the status has changed
				updateChan <- task.StatusUpdate{
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
