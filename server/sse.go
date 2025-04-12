package server

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/sammcj/go-a2a/a2a"
)

// SSEManager manages Server-Sent Events (SSE) connections for A2A tasks.
type SSEManager struct {
	// Map of task ID to a map of connection IDs to SSE connections
	connections map[string]map[string]*sseConnection
	mu          sync.RWMutex
}

// sseConnection represents a single SSE connection.
type sseConnection struct {
	taskID       string
	connectionID string
	w            http.ResponseWriter
	flusher      http.Flusher
	done         chan struct{}
	lastEventID  string
}

// NewSSEManager creates a new SSE manager.
func NewSSEManager() *SSEManager {
	return &SSEManager{
		connections: make(map[string]map[string]*sseConnection),
	}
}

// HandleSSE handles an SSE connection for a task.
func (sm *SSEManager) HandleSSE(w http.ResponseWriter, r *http.Request, taskID string, lastEventID string) {
	// Check if the client supports SSE
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// Create a unique connection ID
	connectionID := fmt.Sprintf("conn_%d", time.Now().UnixNano())

	// Create a channel to signal when the connection is closed
	done := make(chan struct{})

	// Create the SSE connection
	conn := &sseConnection{
		taskID:       taskID,
		connectionID: connectionID,
		w:            w,
		flusher:      flusher,
		done:         done,
		lastEventID:  lastEventID,
	}

	// Register the connection
	sm.registerConnection(taskID, connectionID, conn)

	// Remove the connection when the handler returns
	defer sm.removeConnection(taskID, connectionID)

	// Send a comment to establish the connection
	fmt.Fprint(w, ": connected\n\n")
	flusher.Flush()

	// Wait for the connection to be closed
	select {
	case <-r.Context().Done():
		// Request context was cancelled (client disconnected)
		return
	case <-done:
		// Connection was closed by the server
		return
	}
}

// registerConnection registers a new SSE connection.
func (sm *SSEManager) registerConnection(taskID, connectionID string, conn *sseConnection) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	// Create the map for this task if it doesn't exist
	if _, exists := sm.connections[taskID]; !exists {
		sm.connections[taskID] = make(map[string]*sseConnection)
	}

	// Add the connection
	sm.connections[taskID][connectionID] = conn
}

// removeConnection removes an SSE connection.
func (sm *SSEManager) removeConnection(taskID, connectionID string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	// Remove the connection
	if conns, exists := sm.connections[taskID]; exists {
		delete(conns, connectionID)

		// Remove the task entry if there are no more connections
		if len(conns) == 0 {
			delete(sm.connections, taskID)
		}
	}
}

// closeConnection closes an SSE connection.
func (sm *SSEManager) closeConnection(taskID, connectionID string) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	// Find the connection
	if conns, exists := sm.connections[taskID]; exists {
		if conn, exists := conns[connectionID]; exists {
			// Signal that the connection is closed
			close(conn.done)
		}
	}
}

// SendTaskStatusUpdate sends a task status update to all connected clients for a task.
func (sm *SSEManager) SendTaskStatusUpdate(taskID string, status a2a.TaskStatus) {
	event := a2a.TaskStatusUpdateEvent{
		TaskID: taskID,
		Status: status,
	}

	sm.sendEvent(taskID, "taskStatusUpdate", event, taskID+":status:"+string(status.State))
}

// SendTaskArtifactUpdate sends a task artifact update to all connected clients for a task.
func (sm *SSEManager) SendTaskArtifactUpdate(taskID string, artifact a2a.Artifact) {
	event := a2a.TaskArtifactUpdateEvent{
		TaskID:   taskID,
		Artifact: artifact,
	}

	sm.sendEvent(taskID, "taskArtifactUpdate", event, taskID+":artifact:"+artifact.ID)
}

// sendEvent sends an SSE event to all connected clients for a task.
func (sm *SSEManager) sendEvent(taskID, eventType string, data interface{}, eventID string) {
	// Marshal the data to JSON
	jsonData, err := json.Marshal(data)
	if err != nil {
		// Log the error
		fmt.Printf("Error marshalling SSE event data: %v\n", err)
		return
	}

	// Get all connections for this task
	sm.mu.RLock()
	conns := make([]*sseConnection, 0)
	if taskConns, exists := sm.connections[taskID]; exists {
		for _, conn := range taskConns {
			conns = append(conns, conn)
		}
	}
	sm.mu.RUnlock()

	// Send the event to all connections
	for _, conn := range conns {
		// Skip if the connection already received this event
		if conn.lastEventID >= eventID {
			continue
		}

		// Send the event
		fmt.Fprintf(conn.w, "event: %s\n", eventType)
		fmt.Fprintf(conn.w, "id: %s\n", eventID)
		fmt.Fprintf(conn.w, "data: %s\n\n", jsonData)
		conn.flusher.Flush()

		// Update the last event ID
		conn.lastEventID = eventID
	}
}

// HandleTaskSendSubscribe handles the tasks/sendSubscribe method.
func (s *Server) handleTaskSendSubscribe(ctx context.Context, w http.ResponseWriter, r *http.Request, request *a2a.JSONRPCRequest) {
	// Parse params
	var params a2a.TaskSendParams
	if err := json.Unmarshal(request.Params, &params); err != nil {
		writeJSONRPCError(w, r, a2a.ErrInvalidParams(err.Error()), request.ID)
		return
	}

	// Get the Last-Event-ID header if present
	lastEventID := r.Header.Get("Last-Event-ID")

	// Call TaskManager to start the task
	updateChan, err := s.taskManager.OnSendTaskSubscribe(ctx, &params)
	if err != nil {
		// Convert error to JSON-RPC error
		var a2aErr *a2a.Error
		if e, ok := err.(*a2a.Error); ok {
			a2aErr = e
		} else {
			a2aErr = a2a.ErrInternalError(err)
		}
		writeJSONRPCError(w, r, a2aErr, request.ID)
		return
	}

	// Get the task ID from the task manager
	var taskID string
	if params.TaskID != nil {
		taskID = *params.TaskID
	} else {
		// For new tasks, we need to get the task ID from the first update
		select {
		case update := <-updateChan:
			if statusUpdate, ok := update.(StatusUpdate); ok && statusUpdate.State == a2a.TaskStateSubmitted {
				// Get the task from the task manager
				task, err := s.taskManager.OnGetTask(ctx, &a2a.TaskQueryParams{TaskID: taskID})
				if err != nil {
					writeJSONRPCError(w, r, a2a.ErrInternalError(err), request.ID)
					return
				}
				taskID = task.ID
			} else {
				writeJSONRPCError(w, r, a2a.ErrInternalError(fmt.Errorf("unexpected first update")), request.ID)
				return
			}
		case <-ctx.Done():
			writeJSONRPCError(w, r, a2a.ErrInternalError(ctx.Err()), request.ID)
			return
		}
	}

	// Start a goroutine to process updates from the task manager
	go func() {
		for update := range updateChan {
			switch u := update.(type) {
			case StatusUpdate:
				s.sseManager.SendTaskStatusUpdate(taskID, a2a.TaskStatus{
					State:     u.State,
					Timestamp: time.Now(),
					Message:   u.Message,
				})
			case ArtifactUpdate:
				artifact := a2a.Artifact{
					ID:        fmt.Sprintf("artifact_%d", time.Now().UnixNano()),
					TaskID:    taskID,
					Timestamp: time.Now(),
					Part:      u.Part,
					Metadata:  u.Metadata,
				}
				s.sseManager.SendTaskArtifactUpdate(taskID, artifact)
			}
		}
	}()

	// Handle the SSE connection
	s.sseManager.HandleSSE(w, r, taskID, lastEventID)
}

// handleSSERequest handles SSE requests.
func (s *Server) handleSSERequest(w http.ResponseWriter, r *http.Request) {
	// Only accept POST requests
	if r.Method != http.MethodPost {
		writeJSONRPCError(w, r, a2a.ErrInvalidRequest("Method not allowed"), nil)
		return
	}

	// Read request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeJSONRPCError(w, r, a2a.ErrParseError(err), nil)
		return
	}

	// Parse JSON-RPC request
	var request a2a.JSONRPCRequest
	if err := json.Unmarshal(body, &request); err != nil {
		writeJSONRPCError(w, r, a2a.ErrParseError(err), nil)
		return
	}

	// Validate JSON-RPC version
	if request.JSONRPC != "2.0" {
		writeJSONRPCError(w, r, a2a.ErrInvalidRequest("Invalid JSON-RPC version"), request.ID)
		return
	}

	// Create context with timeout
	// TODO: Make timeout configurable
	ctx := r.Context()

	// Route request to appropriate handler based on method
	switch request.Method {
	case "tasks/sendSubscribe":
		s.handleTaskSendSubscribe(ctx, w, r, &request)
	case "tasks/resubscribe":
		s.handleTaskResubscribe(ctx, w, r, &request)
	default:
		writeJSONRPCError(w, r, a2a.ErrMethodNotFound(request.Method), request.ID)
	}
}

// HandleTaskResubscribe handles the tasks/resubscribe method.
func (s *Server) handleTaskResubscribe(ctx context.Context, w http.ResponseWriter, r *http.Request, request *a2a.JSONRPCRequest) {
	// Parse params
	var params a2a.TaskIdParams
	if err := json.Unmarshal(request.Params, &params); err != nil {
		writeJSONRPCError(w, r, a2a.ErrInvalidParams(err.Error()), request.ID)
		return
	}

	// Get the Last-Event-ID header if present
	lastEventID := r.Header.Get("Last-Event-ID")

	// Call TaskManager to resubscribe to the task
	updateChan, err := s.taskManager.OnResubscribeToTask(ctx, &params)
	if err != nil {
		// Convert error to JSON-RPC error
		var a2aErr *a2a.Error
		if e, ok := err.(*a2a.Error); ok {
			a2aErr = e
		} else {
			a2aErr = a2a.ErrInternalError(err)
		}
		writeJSONRPCError(w, r, a2aErr, request.ID)
		return
	}

	// Start a goroutine to process updates from the task manager
	go func() {
		for update := range updateChan {
			switch u := update.(type) {
			case StatusUpdate:
				s.sseManager.SendTaskStatusUpdate(params.TaskID, a2a.TaskStatus{
					State:     u.State,
					Timestamp: time.Now(),
					Message:   u.Message,
				})
			case ArtifactUpdate:
				artifact := a2a.Artifact{
					ID:        fmt.Sprintf("artifact_%d", time.Now().UnixNano()),
					TaskID:    params.TaskID,
					Timestamp: time.Now(),
					Part:      u.Part,
					Metadata:  u.Metadata,
				}
				s.sseManager.SendTaskArtifactUpdate(params.TaskID, artifact)
			}
		}
	}()

	// Handle the SSE connection
	s.sseManager.HandleSSE(w, r, params.TaskID, lastEventID)
}
