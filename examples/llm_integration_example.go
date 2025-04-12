package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/sammcj/go-a2a/a2a"
	"github.com/sammcj/go-a2a/client"
	"github.com/sammcj/go-a2a/server"
)

func main() {
	// Create an agent card
	agentCard := &a2a.AgentCard{
		A2AVersion: "1.0",
		ID:         "llm-agent",
		Name:       "LLM Agent",
		Description: func() *string {
			s := "An intelligent agent powered by an LLM"
			return &s
		}(),
		Skills: []a2a.AgentSkill{
			{
				ID:   "general-assistance",
				Name: "General Assistance",
				Description: "Provides helpful responses to general questions and requests",
			},
		},
		Capabilities: &a2a.AgentCapabilities{
			SupportsStreaming: true,
		},
	}

	// Create and start the server with a gollm-based agent
	a2aServer, err := server.NewServer(
		server.WithAgentCard(agentCard),
		server.WithBasicGollmAgent(
			"ollama",                // provider
			"deepcoder:14b-preview-q4_K_M",                // model
			"",                      // API key (not needed for Ollama)
			"You are a helpful assistant that responds to user queries in a concise and accurate manner.",
		),
		server.WithListenAddress(":8080"),
	)
	if err != nil {
		log.Fatalf("Failed to create A2A server: %v", err)
	}

	// Start the server in a goroutine
	go func() {
		if err := a2aServer.Start(); err != nil {
			log.Fatalf("Failed to start A2A server: %v", err)
		}
	}()

	// Wait for the server to start
	time.Sleep(1 * time.Second)

	// Create a client
	a2aClient, err := client.NewClient(
		client.WithBaseURL("http://localhost:8080"),
	)
	if err != nil {
		log.Fatalf("Failed to create A2A client: %v", err)
	}

	// Create a message
	message := a2a.Message{
		Role: a2a.RoleUser,
		Parts: []a2a.Part{
			a2a.TextPart{
				Type: "text",
				Text: "What are the benefits of using Go for backend development?",
			},
		},
	}

	fmt.Println("Sending task to LLM agent...")

	// Send a task with streaming
	updateChan, errChan := a2aClient.SendSubscribe(context.Background(), &a2a.TaskSendParams{
		Message: message,
	})

	// Process updates
	var taskID string
	for {
		select {
		case update, ok := <-updateChan:
			if !ok {
				// Channel closed, all updates received
				fmt.Println("\nTask completed.")
				return
			}

			if update.Type == "status" {
				taskID = update.Status.ID
				if update.Status.Status.State == a2a.TaskStateWorking && update.Status.Status.Message != nil {
					// Print agent's response
					for _, part := range update.Status.Status.Message.Parts {
						if textPart, ok := part.(a2a.TextPart); ok {
							fmt.Print(textPart.Text)
						}
					}
				}
			} else if update.Type == "artifact" {
				// Print artifact content
				for _, part := range update.Artifact.Parts {
					if textPart, ok := part.(a2a.TextPart); ok {
						fmt.Print(textPart.Text)
					}
				}
			}

		case err := <-errChan:
			log.Fatalf("Error receiving updates: %v", err)
		case <-time.After(30 * time.Second):
			// Timeout after 30 seconds
			fmt.Println("\nTimeout waiting for response.")

			// Cancel the task
			_, err := a2aClient.CancelTask(context.Background(), &a2a.TaskIdParams{
				ID: taskID,
			})
			if err != nil {
				log.Printf("Failed to cancel task: %v", err)
			}

			os.Exit(1)
		}
	}
}
