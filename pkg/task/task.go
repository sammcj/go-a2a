// Package task provides shared types and interfaces for the A2A task handling system.
package task

import (
	"context"

	"github.com/sammcj/go-a2a/a2a"
)

// Context contains the context for a task to be processed.
type Context struct {
	TaskID      string      // The ID of the task
	UserMessage a2a.Message // The user's message
}

// YieldUpdate represents an update to a task being processed.
type YieldUpdate interface {
	isTaskYieldUpdate()
}

// StatusUpdate represents a status update for a task.
type StatusUpdate struct {
	State   a2a.TaskState // The new state of the task
	Message *a2a.Message  // An optional message to include with the update
}

// Ensure StatusUpdate implements YieldUpdate.
func (s StatusUpdate) isTaskYieldUpdate() {}

// ArtifactUpdate represents an artifact being added to a task.
type ArtifactUpdate struct {
	Part     a2a.Part                // The artifact part
	Metadata map[string]interface{}  // Optional metadata for the artifact
}

// Ensure ArtifactUpdate implements YieldUpdate.
func (a ArtifactUpdate) isTaskYieldUpdate() {}

// Handler is a function that processes a task and returns a channel for updates.
type Handler func(ctx context.Context, taskCtx Context) (<-chan YieldUpdate, error)
