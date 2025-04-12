package server

import (
	"context"
	"fmt"
	"time"

	"github.com/sammcj/go-a2a/a2a"
	"github.com/sammcj/go-a2a/pkg/task"
)

// newMockHandler creates a new mock task handler
func newMockHandler() task.Handler {
	return func(ctx context.Context, taskCtx task.Context) (<-chan task.YieldUpdate, error) {
		updates := make(chan task.YieldUpdate)

		go func() {
			defer close(updates)

			// Send working status
			updates <- task.StatusUpdate{
				State: a2a.TaskStateWorking,
				Message: &a2a.Message{
					Role:      "assistant",
					Timestamp: time.Now(),
					Parts: []a2a.Part{
						a2a.TextPart{
							Type: "text",
							Text: "Working on task...",
						},
					},
				},
			}

			// Send completed status
			updates <- task.StatusUpdate{
				State: a2a.TaskStateCompleted,
				Message: &a2a.Message{
					Role:      "assistant",
					Timestamp: time.Now(),
					Parts: []a2a.Part{
						a2a.TextPart{
							Type: "text",
							Text: fmt.Sprintf("Completed task: %s", taskCtx.TaskID),
						},
					},
				},
			}
		}()
		return updates, nil
	}
}
