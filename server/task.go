package server

import (
	"github.com/sammcj/go-a2a/a2a"
	"github.com/sammcj/go-a2a/pkg/task"
)

// TaskContext is an alias for task.Context
type TaskContext = task.Context

// TaskYieldUpdate is an alias for task.YieldUpdate
type TaskYieldUpdate = task.YieldUpdate

// StatusUpdate is an alias for task.StatusUpdate
type StatusUpdate = task.StatusUpdate

// Handler is an alias for task.Handler
type Handler = task.Handler

// Task represents a complete task with its status and history
type Task = a2a.Task

// TaskStatus constants
const (
	TaskStatusSubmitted  = a2a.TaskStateSubmitted
	TaskStatusInProgress = a2a.TaskStateWorking
	TaskStatusCompleted  = a2a.TaskStateCompleted
	TaskStatusFailed     = a2a.TaskStateFailed
	TaskStatusCancelled  = a2a.TaskStateCancelled
)
