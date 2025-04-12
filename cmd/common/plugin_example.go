package common

import (
	"context"
	"fmt"
	"time"

	"github.com/sammcj/go-a2a/a2a"
	"github.com/sammcj/go-a2a/server"
)

// EchoPlugin is an example plugin that echoes back the user's message.
type EchoPlugin struct{}

// GetTaskHandler returns the task handler function for the echo plugin.
func (p *EchoPlugin) GetTaskHandler() server.TaskHandler {
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
						Text: fmt.Sprintf("Echo: %s", userText),
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
func (p *EchoPlugin) GetSkills() []a2a.AgentSkill {
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

// FileProcessorPlugin is an example plugin that processes files.
type FileProcessorPlugin struct{}

// GetTaskHandler returns the task handler function for the file processor plugin.
func (p *FileProcessorPlugin) GetTaskHandler() server.TaskHandler {
	return func(ctx context.Context, taskCtx server.TaskContext) (<-chan server.TaskYieldUpdate, error) {
		// Create a channel for updates
		updateChan := make(chan server.TaskYieldUpdate)

		// Start a goroutine to handle the task
		go func() {
			defer close(updateChan)

			// Extract the user's message
			userMessage := taskCtx.UserMessage
			var fileName string
			var fileContent string

			// Look for file parts in the message
			for _, part := range userMessage.Parts {
				if filePart, ok := part.(a2a.FilePart); ok {
					fileName = filePart.Filename
					if filePart.Content != nil {
						fileContent = filePart.Content.Data
					}
					break
				}
			}

			// If no file was found, send an error message
			if fileName == "" {
				// Create an agent message
				agentMessage := &a2a.Message{
					Role:      a2a.RoleAgent,
					Timestamp: time.Now(),
					Parts: []a2a.Part{
						a2a.TextPart{
							Type: "text",
							Text: "No file found in the message. Please provide a file to process.",
						},
					},
				}

				// Send a status update with the agent message
				updateChan <- server.StatusUpdate{
					State:   a2a.TaskStateInputRequired,
					Message: agentMessage,
				}
				return
			}

			// Process the file (in this example, just count characters)
			charCount := len(fileContent)

			// Create an agent message
			agentMessage := &a2a.Message{
				Role:      a2a.RoleAgent,
				Timestamp: time.Now(),
				Parts: []a2a.Part{
					a2a.TextPart{
						Type: "text",
						Text: fmt.Sprintf("Processed file: %s\nCharacter count: %d", fileName, charCount),
					},
				},
			}

			// Send a status update with the agent message
			updateChan <- server.StatusUpdate{
				State:   a2a.TaskStateCompleted,
				Message: agentMessage,
			}

			// Create an artifact with the results
			updateChan <- server.ArtifactUpdate{
				Part: a2a.DataPart{
					Type:     "data",
					MimeType: "application/json",
					Data: map[string]interface{}{
						"fileName":    fileName,
						"charCount":   charCount,
						"processedAt": time.Now().Format(time.RFC3339),
					},
				},
				Metadata: map[string]interface{}{
					"type": "file-analysis",
				},
			}
		}()

		return updateChan, nil
	}
}

// GetSkills returns the skills provided by the file processor plugin.
func (p *FileProcessorPlugin) GetSkills() []a2a.AgentSkill {
	return []a2a.AgentSkill{
		{
			ID:   "file-processor",
			Name: "File Processor",
			Description: func() *string {
				desc := "Processes files and returns information about them"
				return &desc
			}(),
		},
	}
}

// NewEchoPlugin creates a new echo plugin.
func NewEchoPlugin() *EchoPlugin {
	return &EchoPlugin{}
}

// NewFileProcessorPlugin creates a new file processor plugin.
func NewFileProcessorPlugin() *FileProcessorPlugin {
	return &FileProcessorPlugin{}
}
