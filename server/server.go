package server

import (
	"context"
	"fmt"
	"net/http"
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

	// TODO: Validate other config options (e.g., address)

	s := &Server{
		config:      cfg,
		taskManager: cfg.TaskManager,
		sseManager:  NewSSEManager(),
	}

	// Setup HTTP routing (basic for now)
	mux := http.NewServeMux()

	// Register Agent Card handler
	RegisterAgentCardHandler(mux, cfg.AgentCard, cfg.AgentCardPath)

	// Register main A2A endpoint
	mux.HandleFunc(cfg.A2APathPrefix, s.handleA2ARequest)

	// Register SSE endpoint
	mux.HandleFunc(cfg.A2APathPrefix+"/sse", s.handleSSERequest)

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

// Note: The handleA2ARequest method is now implemented in handler.go
