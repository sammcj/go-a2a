package main

import (
	"context"
	"fmt"
	"time"

	"github.com/sammcj/go-a2a/a2a"
	"github.com/sammcj/go-a2a/server"
)

// CustomEchoPlugin is an example plugin that echoes back the user's message.
type CustomEchoPlugin struct{}

// GetTaskHandler returns the task handler function for the echo plugin.
func (p *CustomEchoPlugin) GetTaskHandler() func(context.Context, server.TaskContext) (<-chan server.TaskYieldUpdate, error) {
	return func(ctx context.Context, taskCtx server.TaskContext) (<-chan server.TaskYieldUpdate, error) {
		// Create a channel for updates
		updateChan := make(chan server.TaskYieldUpdate)

		// Start a goroutine to handle the task
		go func() {
			defer close(updateChan)

			// Extract the user's message
			userMessage := taskCtx.UserMessage
			var userText string
			for _, part := range userMessage.Parts {
				if textPart, ok := part.(a2a.TextPart); ok {
					userText = textPart.Text
					break
				}
			}

			// Create an agent message
			agentMessage := &a2a.Message{
				Role:      a2a.RoleAgent,
				Timestamp: time.Now(),
				Parts: []a2a.Part{
					a2a.TextPart{
						Type: "text",
						Text: fmt.Sprintf("Custom Echo: %s", userText),
					},
				},
			}

			// Send a status update with the agent message
			updateChan <- server.StatusUpdate{
				State:   a2a.TaskStateCompleted,
				Message: agentMessage,
			}
		}()

		return updateChan, nil
	}
}

// GetSkills returns the skills provided by the echo plugin.
func (p *CustomEchoPlugin) GetSkills() []a2a.AgentSkill {
	return []a2a.AgentSkill{
		{
			ID:   "echo",
			Name: "Echo",
			Description: func() *string {
				desc := "Echoes back the user's message"
				return &desc
			}(),
		},
	}
}

// Plugin is the exported symbol that the server will look for.
// var Plugin common.TaskHandlerPlugin = &CustomEchoPlugin{} // TODO: fix this?
