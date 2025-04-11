package server

import (
	"context"
	"fmt"
	"net/http"

	"github.com/sammcj/go-a2a" // Assuming the module path will be this
)

// Server implements the A2A server functionality.
type Server struct {
	config      Config
	httpServer  *http.Server
	taskManager TaskManager // Interface for task management logic
	// TODO: Add other dependencies like AgentCard provider, SSE manager etc.
}

// NewServer creates a new A2A Server instance.
func NewServer(opts ...Option) (*Server, error) {
	cfg := DefaultConfig()
	for _, opt := range opts {
		opt(&cfg)
	}

	if cfg.AgentCard == nil {
		return nil, fmt.Errorf("agent card configuration is required")
	}
	if cfg.TaskManager == nil {
		// Use default in-memory task manager if none provided
		cfg.TaskManager = NewInMemoryTaskManager(cfg.TaskHandler) // Assuming TaskHandler is configured
		// TODO: Check if TaskHandler is nil and handle appropriately
	}

	// TODO: Validate other config options (e.g., address)

	s := &Server{
		config:      cfg,
		taskManager: cfg.TaskManager,
	}

	// Setup HTTP routing (basic for now)
	mux := http.NewServeMux()
	// TODO: Add Agent Card handler (/.well-known/agent.json)
	mux.HandleFunc(cfg.A2APathPrefix, s.handleA2ARequest) // Main A2A endpoint
	// TODO: Add SSE endpoint (/a2a/sse or similar)

	s.httpServer = &http.Server{
		Addr:    cfg.ListenAddress,
		Handler: mux, // TODO: Add middleware (logging, auth)
		// TODO: Configure timeouts (ReadTimeout, WriteTimeout, IdleTimeout)
	}

	return s, nil
}

// Start runs the A2A server. It blocks until the server is stopped.
func (s *Server) Start() error {
	// TODO: Log server start information (address, agent ID)
	fmt.Printf("Starting A2A server for agent '%s' at %s%s\n", s.config.AgentCard.ID, s.config.ListenAddress, s.config.A2APathPrefix)
	err := s.httpServer.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("failed to start HTTP server: %w", err)
	}
	return nil
}

// Stop gracefully shuts down the server.
func (s *Server) Stop(ctx context.Context) error {
	// TODO: Log server shutdown
	fmt.Println("Stopping A2A server...")
	err := s.httpServer.Shutdown(ctx)
	if err != nil {
		return fmt.Errorf("failed to gracefully shutdown HTTP server: %w", err)
	}
	// TODO: Add cleanup for TaskManager, SSE connections etc.
	fmt.Println("A2A server stopped.")
	return nil
}

// handleA2ARequest is the main entry point for incoming A2A JSON-RPC requests.
func (s *Server) handleA2ARequest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		// TODO: Use a2a.Error for proper JSON-RPC response
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	// TODO: Implement full JSON-RPC request parsing, validation, routing to TaskManager, and response generation.
	// This will involve:
	// 1. Reading request body.
	// 2. Unmarshalling into a2a.JSONRPCRequest.
	// 3. Validating JSON-RPC version, method name.
	// 4. Routing based on method (e.g., "tasks/send", "tasks/get").
	// 5. Unmarshalling params into specific struct (e.g., a2a.TaskSendParams).
	// 6. Calling the appropriate s.taskManager method (e.g., s.taskManager.OnSendTask).
	// 7. Handling errors from TaskManager (converting to a2a.Error).
	// 8. Marshalling the result or error into a2a.JSONRPCResponse.
	// 9. Writing the response with correct Content-Type header.

	// Placeholder response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotImplemented)
	fmt.Fprintln(w, `{"jsonrpc": "2.0", "error": {"code": -32601, "message": "Method not implemented yet"}, "id": null}`)
}

// --- Helper Functions (Example) ---

func writeJSONRPCError(w http.ResponseWriter, r *http.Request, err *a2a.Error, reqID interface{}) {
	w.Header().Set("Content-Type", "application/json")
	// Determine appropriate HTTP status code based on JSON-RPC error code?
	// For simplicity, often use 200 OK for valid JSON-RPC error responses,
	// or 4xx/5xx for transport-level issues.
	w.WriteHeader(http.StatusOK) // Or potentially map codes like AuthFailed to 401/403?

	resp := a2a.JSONRPCResponse{
		JSONRPC: "2.0",
		Error:   err.ToJSONRPCError(),
		ID:      reqID, // Use the request ID
	}
	// TODO: Marshal and write response, handle marshalling errors
}
