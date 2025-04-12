package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/sammcj/go-a2a/a2a"
	"github.com/sammcj/go-a2a/server"
)

// This example demonstrates how to set up an A2A server with authentication
// and push notifications.

func main() {
	// Create a simple task handler that echoes the user's message
	taskHandler := func(ctx context.Context, taskCtx server.TaskContext) (<-chan server.TaskYieldUpdate, error) {
		updateChan := make(chan server.TaskYieldUpdate)

		go func() {
			defer close(updateChan)

			// Simulate some processing time
			time.Sleep(1 * time.Second)

			// Get the user's message
			userMessage := taskCtx.UserMessage
			var userText string
			for _, part := range userMessage.Parts {
				if textPart, ok := part.(a2a.TextPart); ok {
					userText = textPart.Text
					break
				}
			}

			// Create a response message
			responseMessage := a2a.Message{
				Role:      a2a.RoleAgent,
				Timestamp: time.Now(),
				Parts: []a2a.Part{
					a2a.TextPart{
						Type: "text",
						Text: fmt.Sprintf("Echo: %s", userText),
					},
				},
			}

			// Send a working status update
			updateChan <- server.StatusUpdate{
				State:   a2a.TaskStateWorking,
				Message: &responseMessage,
			}

			// Simulate more processing time
			time.Sleep(1 * time.Second)

			// Create an artifact
			updateChan <- server.ArtifactUpdate{
				Part: a2a.TextPart{
					Type: "text",
					Text: fmt.Sprintf("Processed text: %s", userText),
				},
				Metadata: map[string]interface{}{
					"processingTime": "2s",
				},
			}

			// Simulate final processing time
			time.Sleep(1 * time.Second)

			// Send a completed status update
			updateChan <- server.StatusUpdate{
				State: a2a.TaskStateCompleted,
			}
		}()

		return updateChan, nil
	}

	// Create an agent card with authentication
	agentCard := &a2a.AgentCard{
		A2AVersion: "1.0",
		ID:         "echo-agent",
		Name:       "Echo Agent",
		Description: func() *string {
			s := "An agent that echoes back your messages"
			return &s
		}(),
		Skills: []a2a.AgentSkill{
			{
				ID:          "echo",
				Name:        "Echo",
				Description: func() *string { s := "Echoes back your message"; return &s }(),
			},
		},
		Capabilities: &a2a.AgentCapabilities{
			SupportsStreaming:       true,
			SupportsSessions:        true,
			SupportsPushNotification: true,
		},
		Authentication: []a2a.AgentAuthentication{
			{
				Type: "bearer",
			},
		},
	}

	// Create a simple token validator
	authValidator := func(w http.ResponseWriter, r *http.Request, next http.Handler, card *a2a.AgentCard) {
		// Get the Authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			// No auth header, return 401
			http.Error(w, "Authentication required", http.StatusUnauthorized)
			return
		}

		// Check if it's a Bearer token
		if len(authHeader) < 7 || authHeader[:7] != "Bearer " {
			http.Error(w, "Invalid authentication format", http.StatusUnauthorized)
			return
		}

		// Get the token
		token := authHeader[7:]

		// Check if it's the expected token (in a real app, you'd use a more secure method)
		if token != "secret-token" {
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}

		// Token is valid, proceed to the next handler
		next.ServeHTTP(w, r)
	}

	// Create a push notification handler
	http.HandleFunc("/push-notifications", func(w http.ResponseWriter, r *http.Request) {
		// Log the push notification
		fmt.Println("Received push notification:")

		// Parse the request body
		var payload map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			fmt.Printf("Error parsing push notification: %v\n", err)
			http.Error(w, "Bad request", http.StatusBadRequest)
			return
		}

		// Print the payload
		jsonData, _ := json.MarshalIndent(payload, "", "  ")
		fmt.Println(string(jsonData))

		// Return a success response
		w.WriteHeader(http.StatusOK)
	})

	// Start the push notification server on a different port
	go func() {
		fmt.Println("Starting push notification server on :8081...")
		if err := http.ListenAndServe(":8081", nil); err != nil {
			log.Fatalf("Failed to start push notification server: %v", err)
		}
	}()

	// Create and start the A2A server
	a2aServer, err := server.NewServer(
		server.WithAgentCard(agentCard),
		server.WithTaskHandler(taskHandler),
		server.WithListenAddress(":8080"),
		server.WithAuthValidator(authValidator),
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

	fmt.Println("A2A server started on :8080")
	fmt.Println("Use the following curl commands to test:")
	fmt.Println("1. Get the agent card:")
	fmt.Println("   curl http://localhost:8080/.well-known/agent.json")
	fmt.Println("2. Send a task (requires authentication):")
	fmt.Println("   curl -X POST http://localhost:8080/a2a -H \"Authorization: Bearer secret-token\" -H \"Content-Type: application/json\" -d '{\"jsonrpc\":\"2.0\",\"method\":\"tasks/send\",\"id\":1,\"params\":{\"message\":{\"role\":\"user\",\"timestamp\":\"2023-01-01T00:00:00Z\",\"parts\":[{\"type\":\"text\",\"text\":\"Hello, world!\"}]}}}'")
	fmt.Println("3. Set up push notifications:")
	fmt.Println("   curl -X POST http://localhost:8080/a2a -H \"Authorization: Bearer secret-token\" -H \"Content-Type: application/json\" -d '{\"jsonrpc\":\"2.0\",\"method\":\"tasks/pushNotification/set\",\"id\":2,\"params\":{\"taskId\":\"TASK_ID_FROM_PREVIOUS_RESPONSE\",\"url\":\"http://localhost:8081/push-notifications\",\"includeTaskData\":true,\"includeArtifacts\":true}}'")

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	// Gracefully shut down the server
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := a2aServer.Stop(ctx); err != nil {
		log.Fatalf("Failed to stop A2A server: %v", err)
	}
	fmt.Println("Server stopped")
}
