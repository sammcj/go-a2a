package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/sammcj/go-a2a/llm/gollm"
)

// Server implements the A2A server functionality.
type Server struct {
	config      Config
	httpServer  *http.Server
	taskManager TaskManager // Interface for task management logic
	sseManager  *SSEManager // Manager for SSE connections
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

	if cfg.AgentEngine == nil {
		if cfg.gollmOptions == nil {
			return nil, errors.New("gollm options must be set when agent engine not set")
		}
		// Create a gollm adapter with the provided options
		adapter, err := gollm.NewAdapter(cfg.gollmOptions...)
		if err != nil {
			return nil, fmt.Errorf("failed to create gollm adapter: %w", err)
		}

		// Create a basic LLM agent
		cfg.AgentEngine = NewBasicLLMAgent(adapter, "You are a helpful assistant.")
	}
	// TODO: Validate other config options (e.g., address)

	s := &Server{
		config:      cfg,
		taskManager: cfg.TaskManager,
		sseManager:  NewSSEManager(),
	}

	// Setup HTTP routing
	mux := http.NewServeMux()

	// Register Agent Card handler
	RegisterAgentCardHandler(mux, cfg.AgentCard, cfg.AgentCardPath)

	// Register main A2A endpoint
	mux.HandleFunc(cfg.A2APathPrefix, s.handleA2ARequest)

	// Register SSE endpoint
	mux.HandleFunc(cfg.A2APathPrefix+"/sse", s.handleSSERequest)

	// Create the final handler with middleware
	var handler http.Handler = mux

	// Apply authentication middleware if configured
	if cfg.AuthValidator != nil {
		// Import the middleware package locally to avoid import issues
		authMiddleware := func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Skip authentication for agent card requests
				if r.URL.Path == cfg.AgentCardPath {
					next.ServeHTTP(w, r)
					return
				}

				// Apply authentication logic
				cfg.AuthValidator(w, r, next, cfg.AgentCard)
			})
		}
		handler = authMiddleware(handler)
	}

	s.httpServer = &http.Server{
		Addr:    cfg.ListenAddress,
		Handler: handler,
		// TODO: Configure timeouts (ReadTimeout, WriteTimeout, IdleTimeout)
	}

	return s, nil
}

// Start runs the A2A server. It blocks until the server is stopped.
func (s *Server) Start() error {
	fmt.Printf("Starting A2A server for agent '%s' at %s%s\n", s.config.AgentCard.ID, s.config.ListenAddress, s.config.A2APathPrefix)
	err := s.httpServer.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("failed to start HTTP server: %w", err)
	}
	return nil
}

func (s *Server) handleAgentEngineRequest(w http.ResponseWriter, r *http.Request) {
	if handler, ok := s.config.AgentEngine.(interface {
		HandleRequest(http.ResponseWriter, *http.Request)
	}); ok {
		handler.HandleRequest(w, r)
	} else {
		http.Error(w, "AgentEngine does not support HandleRequest", http.StatusNotImplemented)
	}
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

// Note: The handleA2ARequest method is now implemented in handler.go
