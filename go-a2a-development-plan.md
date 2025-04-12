# Go A2A Implementation - Development Plan

## 1. Introduction

**Goal:** Develop a Go library (`go-a2a`) for the Agent-to-Agent (A2A) protocol. This library should provide both server and client implementations, enabling Go applications to act as A2A agents or interact with them. A key objective is to ensure the design facilitates easy integration with the existing `mcp-go` library, allowing agents to potentially leverage MCP tools and resources.

**Design Principles:**

*   **Clean & Idiomatic Go:** Follow standard Go practices and conventions.
*   **Efficient:** Optimise for performance where appropriate, especially in transport and concurrency.
*   **Easy to Maintain:** Modular design, clear separation of concerns, good documentation and testing.
*   **Extensible:** Use interfaces and patterns that allow users to customise behaviour (e.g., task storage, authentication).
*   **Lightweight:** Minimise external dependencies.
*   **Practical:** Focus on usability for developers building A2A agents or clients.
*   **Specification Compliant:** Adhere strictly to the A2A JSON specification.

## 2. Core Concepts Mapping

*   **A2A Server/Client:** Implemented as distinct components within the `go-a2a` library. The server will handle incoming HTTP/SSE requests, and the client will make outgoing requests.
*   **Agent Card:** A Go struct (`a2a.AgentCard`) representing the card. The server will have functionality to define and serve this (e.g., via a configurable HTTP handler for `/.well-known/agent.json`). The client will have helpers to fetch and parse it.
*   **A2A Methods (`tasks/send`, etc.):** These will be mapped to specific handler functions on the server and dedicated methods on the client struct. JSON-RPC 2.0 request/response formats will be strictly followed.
*   **A2A Objects (Task, Artifact, Message, Part):** Defined as Go structs (`a2a.Task`, `a2a.Artifact`, etc.) in a core types package, with appropriate `json` tags for marshalling/unmarshalling.
*   **Skills:** Represented within the `AgentCard` struct. The server logic will route requests based on the implicit or explicit skill targeted by the client's task message. Skill implementation logic resides within the server's task handlers.
*   **Transport (HTTP/SSE):** The server will use Go's standard `net/http` library to handle JSON-RPC requests. SSE support for `tasks/sendSubscribe` and `tasks/resubscribe` will require dedicated SSE handling logic, potentially inspired by `mcp-go`'s approach but adapted for A2A's specific event types (`TaskStatusUpdateEvent`, `TaskArtifactUpdateEvent`).
*   **Authentication:** Handled primarily at the HTTP layer using middleware. The library will provide hooks or interfaces to integrate custom authentication logic based on schemes declared in the `AgentCard`.
*   **Push Notifications:** The server will need logic to store `PushNotificationConfig` per task and make outbound HTTP requests to the configured notification URL when required.

## 3. Proposed Package Structure

```
go-a2a/
├── a2a.go             # Core types (Task, Artifact, Message, Part, AgentCard, etc.) & constants
├── client/
│   ├── client.go      # Main client implementation (methods for SendTask, GetTask, etc.)
│   ├── options.go     # Client configuration options
│   └── sse.go         # Client-side SSE handling
├── server/
│   ├── server.go      # Main server implementation (HTTP listener, routing)
│   ├── handler.go     # JSON-RPC request handling logic
│   ├── task_manager.go# Interface and default implementation for managing task state
│   ├── sse.go         # Server-side SSE handling (sendSubscribe, resubscribe)
│   ├── agent_card.go  # Agent Card serving logic
│   ├── middleware/    # HTTP middleware (auth, logging, etc.)
│   │   └── auth.go
│   └── options.go     # Server configuration options
├── errors.go          # A2A specific error types (mapping to JSON-RPC codes)
├── go.mod
├── go.sum
└── README.md
```

## 4. Data Structures (Types)

Located primarily in `a2a.go`. Key structs will include:

*   `AgentCard`, `AgentProvider`, `AgentSkill`, `AgentCapabilities`, `AgentAuthentication`
*   `Task`, `TaskStatus`, `TaskState` (enum/const)
*   `Artifact`
*   `Message`, `Role` (enum/const)
*   `Part` (interface or tagged union), `TextPart`, `FilePart`, `DataPart`, `FileContent`
*   `PushNotificationConfig`, `AuthenticationInfo`
*   Structs for each JSON-RPC request parameter (`TaskSendParams`, `TaskQueryParams`, etc.)
*   Structs for JSON-RPC request/response envelopes (`JSONRPCRequest`, `JSONRPCResponse`, specific method request/response types like `SendTaskRequest`, `SendTaskResponse`).
*   Structs for SSE events (`TaskStatusUpdateEvent`, `TaskArtifactUpdateEvent`).

All structs will have `json:"..."` tags matching the A2A specification precisely.

## 5. Server Implementation (`server/`)

*   **Transport:** Use `net/http` to create an HTTP server. A central handler will parse incoming requests, identify the JSON-RPC method, and route to specific method handlers.
*   **JSON-RPC Handling & Delegation:** The server's central HTTP handler will decode incoming JSON-RPC requests, perform basic validation (JSON structure, method name), and then delegate the request to the corresponding method on the configured `TaskManager` interface (e.g., `taskManager.OnGetTask(ctx, params)`). The server is responsible for encoding the `TaskManager`'s response (or error) back into the appropriate JSON-RPC format (standard JSON or SSE).
*   **Task Management (`TaskManager` Interface):** This interface becomes the core logic hub. It should define methods for each A2A operation:
    ```go
    type TaskManager interface {
        // Handles non-streaming task send/resume.
        OnSendTask(ctx context.Context, params *a2a.TaskSendParams) (*a2a.Task, error)

        // Handles streaming task send/resume. Returns a channel for updates.
        OnSendTaskSubscribe(ctx context.Context, params *a2a.TaskSendParams) (<-chan TaskYieldUpdate, error)

        // Handles task retrieval.
        OnGetTask(ctx context.Context, params *a2a.TaskQueryParams) (*a2a.Task, error)

        // Handles task cancellation.
        OnCancelTask(ctx context.Context, params *a2a.TaskIdParams) (*a2a.Task, error)

        // Handles setting push notification config.
        OnSetTaskPushNotification(ctx context.Context, params *a2a.TaskPushNotificationConfig) (*a2a.TaskPushNotificationConfig, error)

        // Handles getting push notification config.
        OnGetTaskPushNotification(ctx context.Context, params *a2a.TaskIdParams) (*a2a.TaskPushNotificationConfig, error)

        // Handles resubscribing to a task stream.
        OnResubscribeToTask(ctx context.Context, params *a2a.TaskIdParams) (<-chan TaskYieldUpdate, error)

        // (Potentially other internal methods for state management)
    }
    ```
    *   The *implementation* of this interface (e.g., `InMemoryTaskManager`) will contain the detailed logic for managing task state, history, artifacts, invoking the application-specific `TaskHandler` function for `OnSendTask`/`OnSendTaskSubscribe`, managing SSE channels, and handling persistence.
*   **Application Logic (`TaskHandler` Function):** The application developer provides a function matching the `TaskHandler` signature, which is used by the `TaskManager` implementation to execute the core agent/tool logic.
    *   **Refined Go Types (within `server` package):**
        ```go
        // TaskContext provides context to the TaskHandler.
        type TaskContext struct {
            Task         a2a.Task          // Snapshot of task state (pass a copy)
            UserMessage  a2a.Message       // Triggering message
            History      []a2a.Message     // Snapshot of history (pass a copy)
            // Cancellation is checked via the context.Context passed to the handler
        }

        // TaskYieldUpdate represents status or artifact updates yielded by a handler.
        type TaskYieldUpdate interface {
            isTaskYieldUpdate() // Marker method
        }

        // StatusUpdate represents a status change yielded by the handler.
        // The server will add the timestamp.
        type StatusUpdate struct {
            State   a2a.TaskState
            Message *a2a.Message // Optional agent message accompanying the status
        }
        func (s StatusUpdate) isTaskYieldUpdate() {}

        // For yielding artifacts, the handler can yield the a2a.Artifact struct directly.
        // The server will check the type received from the channel.
        // Example: yieldChannel <- *myNewArtifact

        // TaskHandler defines the function signature for application-specific task execution logic.
        // It's invoked by the TaskManager implementation.
        // It receives context and returns a channel for yielding updates and an error.
        // Closing the channel indicates completion.
        type TaskHandler func(ctx context.Context, taskContext TaskContext) (<-chan TaskYieldUpdate, error)
        ```
*   **Task Management (`TaskManager` Implementation):**
    *   The default `InMemoryTaskManager` implementation will manage task state (maps, mutexes), history, artifacts, and persistence.
    *   It will invoke the user-provided `TaskHandler` when processing `OnSendTask` or `OnSendTaskSubscribe`.
    *   Provide a default in-memory implementation using maps and mutexes/channels for concurrency control.
    *   Manage task state transitions (`submitted` -> `working` -> `input-required` -> `completed`/`failed`/`canceled`).
    *   Store task history and artifacts.
    *   Handle task persistence (initially in-memory, extensible via the interface).
*   **SSE Handling (A2A):**
    *   Specific endpoint (e.g., `/a2a/sse`) for `tasks/sendSubscribe` and `tasks/resubscribe`. Requires distinct logic from any potential MCP SSE handling.
    *   Manage active A2A SSE connections per task ID.
    *   Push A2A-specific `TaskStatusUpdateEvent` and `TaskArtifactUpdateEvent` to connected clients when task state or artifacts change. Use Go channels for communication between task management logic and SSE writers.
*   **Agent Card Serving:** A configurable HTTP handler (defaulting to `/.well-known/agent.json`) that serves the server's configured `AgentCard` as JSON.
*   **Authentication:** Implement HTTP middleware that checks request headers (e.g., `Authorization`) against the schemes defined in the `AgentCard`. Provide hooks/interfaces for integrating specific auth logic (e.g., validating OAuth tokens).
*   **Push Notifications:** When a task updates and has a `PushNotificationConfig`, the `TaskManager` should trigger an asynchronous function to make an authenticated POST request to the configured `url` with the task update details.
*   **Configuration:** Use a builder pattern or functional options (`server.Option`) for configuring the server (Agent Card details, address, task manager implementation, auth middleware, etc.).

## 6. Client Implementation (`client/`)

*   **Client Struct:** Holds the target agent's base URL, HTTP client, and potentially parsed `AgentCard`.
*   **Core Methods:** Functions like `SendTask`, `GetTask`, `CancelTask`, `SetPushNotification`, `GetPushNotification`, `SendSubscribe`, `Resubscribe`.
*   **HTTP Interaction:** Use Go's `net/http` client to make requests. Handle marshalling Go structs into JSON-RPC request bodies and unmarshalling responses.
*   **SSE Handling (A2A):** The `SendSubscribe` and `Resubscribe` methods will establish an SSE connection to an A2A server and return channels for receiving A2A-specific `TaskStatusUpdateEvent` and `TaskArtifactUpdateEvent` objects, along with an error channel.
*   **Agent Card:** Method to fetch and parse the `AgentCard` from the server's well-known endpoint.
*   **Authentication:** Provide ways to configure authentication credentials (e.g., tokens) that the client will automatically add to outgoing request headers.
*   **Configuration:** Functional options (`client.Option`) for configuration (base URL, HTTP client, timeout, auth credentials).

## 7. LLM Integration Architecture

To provide an all-in-one solution while maintaining flexibility and loose coupling, the go-a2a library will include direct LLM integration capabilities through a modular architecture:

### Core Components

*   **LLM Interface:** A Go interface that defines standard methods for interacting with LLMs, such as generating text and streaming responses.
*   **Agent Engine:** A Go interface that defines how an agent processes tasks, potentially using LLMs.
*   **gollm Adapter:** The primary implementation of the LLM interface using the [gollm](https://github.com/teilomillet/gollm) library, which provides a unified API for interacting with various LLM providers.

### Package Structure

```
go-a2a/
├── llm/
│   ├── interface.go       # LLM interface definition
│   ├── options.go         # Common LLM options
│   └── gollm/
│       ├── adapter.go     # gollm adapter implementation
│       └── options.go     # gollm-specific options
├── server/
│   ├── agent_engine.go    # Agent engine interface and implementations
│   └── ...                # Other server components
```

### Key Interfaces

```go
// LLMInterface defines the interface for LLM interactions
type LLMInterface interface {
    // Generate generates text from a prompt
    Generate(ctx context.Context, prompt string, options ...LLMOption) (string, error)

    // GenerateStream streams text generation from a prompt
    GenerateStream(ctx context.Context, prompt string, options ...LLMOption) (<-chan LLMChunk, <-chan error)

    // GetModelInfo returns information about the LLM model
    GetModelInfo() LLMModelInfo
}

// AgentEngine defines the interface for agent intelligence
type AgentEngine interface {
    // ProcessTask processes a task and returns a channel for updates
    ProcessTask(ctx context.Context, taskCtx TaskContext) (<-chan TaskYieldUpdate, error)

    // GetCapabilities returns the agent's capabilities
    GetCapabilities() AgentCapabilities
}
```

### gollm Adapter

The gollm adapter will be the primary implementation of the LLM interface, focusing on support for:

1. **Ollama**: For local model deployment and inference
2. **OpenAI-compatible APIs**: For compatibility with various hosted services

The adapter will provide a simple, unified interface for interacting with these providers while handling the complexities of different APIs, authentication methods, and response formats.

### Agent Implementations

The library will include default agent implementations that use the LLM interface:

1. **BasicLLMAgent**: A simple agent that processes tasks using an LLM
2. **ToolAugmentedAgent**: An agent that can use tools (e.g., weather API, calculator) in addition to an LLM

### Configuration Options

The server package will include configuration options for easily setting up LLM-powered agents:

```go
// Create a server with a basic gollm agent
server.WithBasicGollmAgent(
    "ollama",                 // provider
    "llama3",                 // model
    "",                       // API key (not needed for Ollama)
    "You are a helpful assistant that responds to user queries.",
)

// Create a server with a tool-augmented gollm agent
server.WithToolAugmentedGollmAgent(
    "openai",                 // provider
    "gpt-4o",                 // model
    os.Getenv("OPENAI_API_KEY"),
    []server.Tool{weatherTool, calculatorTool},
)
```

This architecture provides an all-in-one solution while maintaining flexibility and loose coupling, allowing users to easily integrate LLMs into their A2A agents.

## 8. Integration with `mcp-go`

*   **Primary Integration:** An A2A agent (built with `go-a2a`) acting as an MCP client. The A2A task handling logic within the `go-a2a` server can instantiate and use an MCP client (potentially using `mcp-go` if it offers client capabilities, or another library) to call tools or read resources from MCP servers as part of fulfilling an A2A task. `go-a2a` itself will *not* depend directly on `mcp-go`.
*   **Secondary (Future):** An application could potentially run both an `mcp-go` server and a `go-a2a` server in the same process, sharing underlying logic. An A2A `tasks/send` could potentially be mapped to trigger an `mcp-go` tool call internally. This is more complex and considered a future enhancement.

## 9. Key Considerations

*   **Error Handling:** Define Go error types that map clearly to the standard A2A JSON-RPC error codes (`TaskNotFound`, `InvalidParams`, etc.). Ensure handlers return these errors appropriately.
*   **Concurrency:** Use goroutines for handling concurrent requests, SSE streams, and background task processing. Ensure thread safety in shared components like the `TaskManager` (e.g., using `sync.Mutex` or channels).
*   **State Management:** Robustly handle task state transitions and persistence (even if initially in-memory).
*   **Testing:** Implement comprehensive unit tests for core types, handlers, client methods, and task management. Include integration tests simulating client-server interactions over HTTP and SSE.
*   **Extensibility:** Design with interfaces (`TaskManager`, potentially auth handlers) to allow users to swap out default implementations.
*   **Dependencies:** Rely primarily on the Go standard library (`net/http`, `encoding/json`, `sync`, `context`). Avoid unnecessary third-party dependencies.
*   **Concurrent SSE:** An application needs to manage concurrent SSE connections acting as an A2A client, A2A server, and potentially MCP clients/servers simultaneously. The library must handle these distinct protocol streams (A2A vs. MCP) and connection lifecycles correctly, potentially requiring separate endpoints and handlers if serving both protocols.
*   **Discovery Mechanisms:** Acknowledge the different discovery approaches. A2A uses pre-interaction discovery via fetching the `AgentCard` (typically from `/.well-known/agent.json`), which `go-a2a` must support (serving and fetching). MCP uses post-connection discovery via the `initialize` handshake, which is handled by the MCP client library (e.g., `mcp-go`) used *within* the A2A application, not directly by `go-a2a`.

## 10. Roadmap / Next Steps

1.  **Phase 1: Core Types & Basic Client/Server:** ✅ COMPLETED
    *   ✅ Define all core Go structs (`a2a.go`).
    *   ✅ Define A2A specific error types (`errors.go`).
    *   ✅ Implement basic HTTP server with JSON-RPC request/response handling (`server/server.go`, `server/handler.go`).
    *   ✅ Implement in-memory `TaskManager` (`server/task_manager.go`).
    *   ✅ Implement Agent Card serving (`server/agent_card.go`).
    *   ✅ Implement basic client methods (`SendTask`, `GetTask`, `CancelTask`) without SSE/Push/Auth (`client/client.go`, `client/options.go`).
    *   ✅ Basic unit tests.
2.  **Phase 2: SSE Implementation:** ✅ COMPLETED
    *   ✅ Add server-side SSE handling for `sendSubscribe`/`resubscribe`.
    *   ✅ Add client-side SSE handling.
    *   ✅ Integrate SSE with `TaskManager` updates.
    *   ✅ SSE-specific tests.
3.  **Phase 3: Authentication & Push Notifications:** ✅ COMPLETED
    *   ✅ Implement server-side auth middleware hooks/interfaces (`server/middleware/auth.go`).
    *   ✅ Implement client-side auth configuration (`client/options.go`).
    *   ✅ Implement server logic for sending push notifications (`server/push_notification.go`).
    *   ✅ Implement client/server methods for managing push notification config (`tasks/pushNotification/set`, `tasks/pushNotification/get`).
    *   ✅ Auth and push notification tests (`server/middleware/auth_test.go`, `server/push_notification_test.go`).
    *   ✅ Example demonstrating auth and push notifications (`examples/auth_and_push_example.go`).
4.  **Phase 4: LLM Integration:** 🔄 IN PROGRESS
    *   ⬜ Define LLM interface (`llm/interface.go`).
    *   ⬜ Implement gollm adapter (`llm/gollm/adapter.go`).
    *   ⬜ Define Agent Engine interface (`server/agent_engine.go`).
    *   ⬜ Implement BasicLLMAgent.
    *   ⬜ Implement ToolAugmentedAgent.
    *   ⬜ Add server configuration options for LLM-powered agents.
    *   ⬜ Create examples demonstrating LLM integration.
    *   ⬜ Add tests for LLM components.
5.  **Phase 5: Standalone Client & Server Applications:** ⬜ PLANNED
    *   **Server Application (`cmd/a2a-server`):**
        *   ⬜ Create command-line interface with flags for configuration:
            *   ⬜ Listen address and port
            *   ⬜ Agent card file path
            *   ⬜ Authentication settings
            *   ⬜ Task handler plugin path
        *   ⬜ Implement configuration file support (YAML/JSON):
            *   ⬜ Server settings
            *   ⬜ Agent card configuration
            *   ⬜ Authentication settings
        *   ⬜ Develop plugin system for task handlers:
            *   ⬜ Define plugin interface
            *   ⬜ Implement dynamic loading of plugins
            *   ⬜ Create sample plugins (echo, file processor, etc.)
        *   ⬜ Add logging:
            *   ⬜ Configurable log levels
            *   ⬜ Optional Request/response logging
            *   ⬜ Optional Task execution logging
        *   ⬜ Add graceful shutdown handling
        *   ⬜ Create Dockerfile and docker-compose examples
        *   ⬜ Implement monitoring endpoints:
            *   ⬜ Health check endpoint
            *   ⬜ Basic Metrics endpoint (Prometheus compatible)
            *   ⬜ Basic Task status dashboard (but don't implement a full JS/TS web UI as a standalone app)
    *   **Client Application (`cmd/a2a-client`):**
        *   ⬜ Create command-line interface with subcommands:
            *   ⬜ `send` - Send a task to an agent
            *   ⬜ `get` - Get task status
            *   ⬜ `cancel` - Cancel a task
            *   ⬜ `subscribe` - Subscribe to task updates
            *   ⬜ `push` - Configure push notifications
            *   ⬜ `card` - Get agent card information
        *   ⬜ Add configuration file support:
            *   ⬜ Default agent URLs
            *   ⬜ Authentication settings
            *   ⬜ Output formatting preferences
        *   ⬜ Implement interactive mode:
            *   ⬜ TUI (Terminal User Interface) for task interaction
            *   ⬜ History
            *   ⬜ Live task status updates
        *   ⬜ Implement various output formats:
            *   ⬜ JSON
            *   ⬜ Pretty-printed
    *   **Common Infrastructure:**
        *   ⬜ Shared configuration handling
        *   ⬜ Authentication utilities
        *   ⬜ Error handling and reporting
        *   ⬜ Documentation and examples
        *   ⬜ Installation scripts and packages
6.  **Phase 6: Refinement & Documentation:** 🔄 IN PROGRESS
    *   ✅ Create comprehensive README with architecture overview and usage examples.
    *   ⬜ Write detailed package documentation (godoc).
    *   ⬜ Refine APIs based on usage feedback.
    *   ⬜ Improve test coverage.
    *   ⬜ Add helper utilities (e.g., validating `AgentCard`s).
    *   ⬜ Create Github Actions CI/CD pipeline for testing and releases with semver versioning.

## 11. Example: Web UI Integration (Conceptual)

This section outlines a *conceptual* approach for how a web-based administrative or monitoring UI could interact with an application built using the `go-a2a` library. **Note:** This administrative API is *not* part of the A2A protocol specification and would be implemented by the application developer *alongside* the core A2A server functionality provided by the `go-a2a` library. This would not be part of this project/repository codebase and the following information only serves as an example and something to keep in mind when developing the go-a2a project.

**Challenge:** The A2A protocol itself is designed for agent-to-agent communication focused on task execution, not for direct querying of server state or configuration by a standard web UI.

**Proposed Solution:** The application hosting the `go-a2a` server could expose a separate, standard HTTP API (e.g., REST or GraphQL) specifically for administrative/monitoring purposes. This API would run on a different port or path prefix than the main A2A endpoint.

**Potential API Endpoints (REST Example):**

*   `GET /admin/config`: Returns the server's configuration, potentially including the loaded Agent Card details.
*   `GET /admin/tasks`: Returns a list of active or recent tasks managed by the `TaskManager`. Could support pagination and filtering (e.g., by status).
    *   Response might include basic task info: `[{ "id": "...", "sessionId": "...", "status": "working", "startTime": "..." }, ...]`.
*   `GET /admin/tasks/{taskId}`: Returns detailed information about a specific task, including its current status, history (if stored), and associated artifacts. This endpoint would query the `TaskManager`.
*   `GET /admin/stats`: Returns operational statistics (e.g., total tasks processed, active SSE connections, error counts).

**Web UI Interaction (Pseudo-JavaScript):**

```javascript
async function fetchServerConfig() {
  const response = await fetch('/admin/config');
  const config = await response.json();
  displayConfig(config); // Function to render config in the UI
}

async function fetchTasks(statusFilter = 'active') {
  const response = await fetch(`/admin/tasks?status=${statusFilter}`);
  const tasks = await response.json();
  displayTaskList(tasks); // Function to render task list in the UI
}

async function fetchTaskDetails(taskId) {
  const response = await fetch(`/admin/tasks/${taskId}`);
  const taskDetails = await response.json();
  displayTaskDetails(taskDetails); // Function to render task details
}

// Initial load or periodic refresh
fetchServerConfig();
fetchTasks();
```

**Integration with `go-a2a`:**

*   The application developer would use a web framework (like Go's standard `net/http`, Gin, Echo, etc.) to build this admin API.
*   The API handlers would interact with the configured `TaskManager` instance (accessible via the interface defined in `go-a2a`) to retrieve task data.
*   Server configuration details could be read from the same source used to configure the `go-a2a` server instance.

This approach keeps the A2A protocol implementation clean and focused, while allowing developers to add standard monitoring and administration capabilities as needed using familiar web technologies. The `go-a2a` library facilitates this by providing access to underlying components like the `TaskManager` via interfaces.
