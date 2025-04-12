# go-a2a: Agent-to-Agent Protocol Implementation in Go

[![Go Reference](https://pkg.go.dev/badge/github.com/sammcj/go-a2a.svg)](https://pkg.go.dev/github.com/sammcj/go-a2a)

A Go implementation of the Agent-to-Agent (A2A) protocol, enabling Go applications to act as A2A agents or interact with them.

## Overview

The Agent-to-Agent (A2A) protocol is designed to facilitate communication between AI agents. This library provides both server and client implementations of the protocol in Go, allowing developers to:

- Create A2A agents that can receive and process tasks
- Build client applications that can interact with A2A agents
- Implement authentication and push notification mechanisms
- Handle streaming task updates via Server-Sent Events (SSE)

## Architecture

The library is structured into several key components:

### Core Types (`a2a` package)

Contains the fundamental data structures defined by the A2A protocol:

- `AgentCard`: Describes an agent's capabilities, skills, and authentication requirements
- `Task`: Represents a task being processed by an agent
- `Message`: Represents a message within a task's history
- `Part`: Interface for different types of content (text, files, data)
- Error types that map to JSON-RPC error codes

### Server (`server` package)

Implements the server-side of the A2A protocol:

- `Server`: Main server implementation that handles HTTP requests
- `TaskManager`: Interface for managing task state and processing
- `InMemoryTaskManager`: Default implementation that stores tasks in memory
- SSE handling for streaming task updates
- Push notification support for task status and artifact updates
- Authentication middleware

### Client (`client` package)

Implements the client-side of the A2A protocol:

- `Client`: Main client implementation for interacting with A2A servers
- Methods for sending tasks, getting task status, cancelling tasks, etc.
- SSE client for receiving streaming task updates
- Authentication configuration

## Getting Started

### Installation

```bash
go get github.com/sammcj/go-a2a
```

### Creating an A2A Server

Here's a simple example of creating an A2A server:

```go
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/sammcj/go-a2a/a2a"
	"github.com/sammcj/go-a2a/server"
)

func main() {
	// Create a task handler function
	taskHandler := func(ctx context.Context, taskCtx server.TaskContext) (<-chan server.TaskYieldUpdate, error) {
		updateChan := make(chan server.TaskYieldUpdate)

		go func() {
			defer close(updateChan)

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

			// Send a completed status update
			updateChan <- server.StatusUpdate{
				State: a2a.TaskStateCompleted,
			}
		}()

		return updateChan, nil
	}

	// Create an agent card
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
				ID:   "echo",
				Name: "Echo",
			},
		},
		Capabilities: &a2a.AgentCapabilities{
			SupportsStreaming: true,
		},
	}

	// Create and start the server
	a2aServer, err := server.NewServer(
		server.WithAgentCard(agentCard),
		server.WithTaskHandler(taskHandler),
		server.WithListenAddress(":8080"),
	)
	if err != nil {
		log.Fatalf("Failed to create A2A server: %v", err)
	}

	// Start the server
	if err := a2aServer.Start(); err != nil {
		log.Fatalf("Failed to start A2A server: %v", err)
	}
}
```

### Using the A2A Client

Here's how to use the client to interact with an A2A server:

```go
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/sammcj/go-a2a/a2a"
	"github.com/sammcj/go-a2a/client"
)

func main() {
	// Create a client
	a2aClient, err := client.NewClient(
		client.WithBaseURL("http://localhost:8080"),
	)
	if err != nil {
		log.Fatalf("Failed to create A2A client: %v", err)
	}

	// Create a message
	message := a2a.Message{
		Role:      a2a.RoleUser,
		Timestamp: time.Now(),
		Parts: []a2a.Part{
			a2a.TextPart{
				Type: "text",
				Text: "Hello, world!",
			},
		},
	}

	// Send a task
	task, err := a2aClient.SendTask(context.Background(), &a2a.TaskSendParams{
		Message: message,
	})
	if err != nil {
		log.Fatalf("Failed to send task: %v", err)
	}

	fmt.Printf("Task created with ID: %s\n", task.ID)
	fmt.Printf("Task status: %s\n", task.Status.State)

	// Get task updates via streaming
	updateChan, errChan := a2aClient.SendSubscribe(context.Background(), &a2a.TaskSendParams{
		Message: message,
	})

	// Process updates
	for {
		select {
		case update, ok := <-updateChan:
			if !ok {
				// Channel closed, all updates received
				return
			}
			if update.Type == "status" {
				fmt.Printf("Status update: %s\n", update.Status.State)
			} else if update.Type == "artifact" {
				fmt.Printf("Artifact update: %s\n", update.Artifact.ID)
			}
		case err := <-errChan:
			log.Fatalf("Error receiving updates: %v", err)
		}
	}
}
```

## Authentication

The library supports various authentication methods as defined in the A2A protocol:

### Server-side Authentication

```go
// Create an authentication validator
authValidator := func(w http.ResponseWriter, r *http.Request, next http.Handler, card *a2a.AgentCard) {
	// Get the Authorization header
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
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

	// Check if it's the expected token
	if token != "secret-token" {
		http.Error(w, "Invalid token", http.StatusUnauthorized)
		return
	}

	// Token is valid, proceed to the next handler
	next.ServeHTTP(w, r)
}

// Add authentication to the server
a2aServer, err := server.NewServer(
	server.WithAgentCard(agentCard),
	server.WithTaskHandler(taskHandler),
	server.WithAuthValidator(authValidator),
)
```

### Client-side Authentication

```go
// Create a client with authentication
a2aClient, err := client.NewClient(
	client.WithBaseURL("http://localhost:8080"),
	client.WithBearerToken("secret-token"),
)
```

## Push Notifications

The library supports push notifications for task updates:

### Setting Up Push Notifications (Client)

```go
// Set up push notifications for a task
config, err := a2aClient.SetTaskPushNotification(context.Background(), &a2a.TaskPushNotificationConfigParams{
	TaskID:           task.ID,
	URL:              "https://your-server.com/push-notifications",
	IncludeTaskData:  &includeTaskData,   // true/false
	IncludeArtifacts: &includeArtifacts,  // true/false
	Authentication: &a2a.AuthenticationInfo{
		Type: "bearer",
		Configuration: map[string]interface{}{
			"token": "your-push-notification-token",
		},
	},
})
```

### Receiving Push Notifications (Server)

```go
// Set up an HTTP handler to receive push notifications
http.HandleFunc("/push-notifications", func(w http.ResponseWriter, r *http.Request) {
	// Parse the request body
	var payload map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	// Process the notification
	fmt.Printf("Received push notification: %+v\n", payload)

	// Return a success response
	w.WriteHeader(http.StatusOK)
})
```

## Server-Sent Events (SSE)

The library supports SSE for streaming task updates:

### Streaming Task Updates (Server)

The server automatically handles SSE connections for the `tasks/sendSubscribe` and `tasks/resubscribe` methods.

### Receiving Streaming Updates (Client)

```go
// Subscribe to task updates
updateChan, errChan := a2aClient.SendSubscribe(context.Background(), &a2a.TaskSendParams{
	Message: message,
})

// Process updates
for {
	select {
	case update, ok := <-updateChan:
		if !ok {
			// Channel closed, all updates received
			return
		}
		if update.Type == "status" {
			fmt.Printf("Status update: %s\n", update.Status.State)
		} else if update.Type == "artifact" {
			fmt.Printf("Artifact update: %s\n", update.Artifact.ID)
		}
	case err := <-errChan:
		log.Fatalf("Error receiving updates: %v", err)
	}
}
```

## Examples

See the `examples` directory for more detailed examples:

- `examples/simple_server.go`: A basic A2A server implementation
- `examples/simple_client.go`: A basic A2A client implementation
- `examples/auth_and_push_example.go`: An example demonstrating authentication and push notifications

## Development Status

The library is currently in active development. The following features have been implemented:

- ✅ Core A2A types and constants
- ✅ Basic HTTP server with JSON-RPC request/response handling
- ✅ In-memory task management
- ✅ Agent card serving
- ✅ Basic client methods (SendTask, GetTask, CancelTask)
- ✅ SSE support for streaming task updates
- ✅ Authentication middleware
- ✅ Push notification support

Future enhancements may include:

- More comprehensive examples
- Improved documentation
- Additional helper utilities
- Integration with other libraries and frameworks

## License

This project is licensed under the MIT License - see the LICENSE file for details.
