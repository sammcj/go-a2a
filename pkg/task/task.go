package task

import (
	"context"

	"github.com/sammcj/go-a2a/a2a"
)

// Context represents the context for a task execution.
type Context struct {
	TaskID      string
	UserMessage a2a.Message
}

// YieldUpdate represents an update from a task execution.
type YieldUpdate interface {
	isYieldUpdate()
}

// StatusUpdate represents a status update from a task.
type StatusUpdate struct {
	State   a2a.TaskState
	Message *a2a.Message
}

func (StatusUpdate) isYieldUpdate() {}

// ArtifactUpdate represents an artifact update from a task.
type ArtifactUpdate struct {
	Part     a2a.Part
	Metadata interface{}
}

func (ArtifactUpdate) isYieldUpdate() {}

// Handler is a function type that processes a task and returns a channel of updates.
type Handler func(context.Context, Context) (<-chan YieldUpdate, error)
