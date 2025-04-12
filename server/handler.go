package server

import (
	"context"
	"encoding/json"
	"io"
	"net/http"

	"github.com/sammcj/go-a2a/a2a"
)

// handleA2ARequest is the main entry point for incoming A2A JSON-RPC requests.
// This is a more complete implementation that replaces the placeholder in server.go.
func (s *Server) handleA2ARequest(w http.ResponseWriter, r *http.Request) {
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
	case "tasks/send":
		s.handleTaskSend(ctx, w, r, &request)
	case "tasks/get":
		s.handleTaskGet(ctx, w, r, &request)
	case "tasks/cancel":
		s.handleTaskCancel(ctx, w, r, &request)
	case "tasks/pushNotification/set":
		s.handleTaskPushNotificationSet(ctx, w, r, &request)
	case "tasks/pushNotification/get":
		s.handleTaskPushNotificationGet(ctx, w, r, &request)
	case "tasks/sendSubscribe":
		// Redirect to SSE endpoint
		http.Redirect(w, r, r.URL.Path+"/sse", http.StatusTemporaryRedirect)
		return
	case "tasks/resubscribe":
		// Redirect to SSE endpoint
		http.Redirect(w, r, r.URL.Path+"/sse", http.StatusTemporaryRedirect)
		return
	default:
		writeJSONRPCError(w, r, a2a.ErrMethodNotFound(request.Method), request.ID)
	}
}

// handleTaskSend handles the tasks/send method.
func (s *Server) handleTaskSend(ctx context.Context, w http.ResponseWriter, r *http.Request, request *a2a.JSONRPCRequest) {
	// Parse params
	var params a2a.TaskSendParams
	if err := json.Unmarshal(request.Params, &params); err != nil {
		writeJSONRPCError(w, r, a2a.ErrInvalidParams(err.Error()), request.ID)
		return
	}

	// Call TaskManager
	task, err := s.taskManager.OnSendTask(ctx, &params)
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

	// Write successful response
	writeJSONRPCResponse(w, r, task, request.ID)
}

// handleTaskGet handles the tasks/get method.
func (s *Server) handleTaskGet(ctx context.Context, w http.ResponseWriter, r *http.Request, request *a2a.JSONRPCRequest) {
	// Parse params
	var params a2a.TaskQueryParams
	if err := json.Unmarshal(request.Params, &params); err != nil {
		writeJSONRPCError(w, r, a2a.ErrInvalidParams(err.Error()), request.ID)
		return
	}

	// Call TaskManager
	task, err := s.taskManager.OnGetTask(ctx, &params)
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

	// Write successful response
	writeJSONRPCResponse(w, r, task, request.ID)
}

// handleTaskCancel handles the tasks/cancel method.
func (s *Server) handleTaskCancel(ctx context.Context, w http.ResponseWriter, r *http.Request, request *a2a.JSONRPCRequest) {
	// Parse params
	var params a2a.TaskIdParams
	if err := json.Unmarshal(request.Params, &params); err != nil {
		writeJSONRPCError(w, r, a2a.ErrInvalidParams(err.Error()), request.ID)
		return
	}

	// Call TaskManager
	task, err := s.taskManager.OnCancelTask(ctx, &params)
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

	// Write successful response
	writeJSONRPCResponse(w, r, task, request.ID)
}

// handleTaskPushNotificationSet handles the tasks/pushNotification/set method.
func (s *Server) handleTaskPushNotificationSet(ctx context.Context, w http.ResponseWriter, r *http.Request, request *a2a.JSONRPCRequest) {
	// Parse params
	var params a2a.TaskPushNotificationConfigParams
	if err := json.Unmarshal(request.Params, &params); err != nil {
		writeJSONRPCError(w, r, a2a.ErrInvalidParams(err.Error()), request.ID)
		return
	}

	// Call TaskManager
	config, err := s.taskManager.OnSetTaskPushNotification(ctx, &params)
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

	// Write successful response
	writeJSONRPCResponse(w, r, config, request.ID)
}

// handleTaskPushNotificationGet handles the tasks/pushNotification/get method.
func (s *Server) handleTaskPushNotificationGet(ctx context.Context, w http.ResponseWriter, r *http.Request, request *a2a.JSONRPCRequest) {
	// Parse params
	var params a2a.TaskIdParams
	if err := json.Unmarshal(request.Params, &params); err != nil {
		writeJSONRPCError(w, r, a2a.ErrInvalidParams(err.Error()), request.ID)
		return
	}

	// Call TaskManager
	config, err := s.taskManager.OnGetTaskPushNotification(ctx, &params)
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

	// Write successful response
	writeJSONRPCResponse(w, r, config, request.ID)
}

// writeJSONRPCResponse writes a successful JSON-RPC response.
func writeJSONRPCResponse(w http.ResponseWriter, r *http.Request, result interface{}, id interface{}) {
	response := a2a.JSONRPCResponse{
		JSONRPC: "2.0",
		Result:  result,
		ID:      id,
	}

	// Marshal response
	jsonResp, err := json.Marshal(response)
	if err != nil {
		// If marshalling fails, return an internal error
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Write response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(jsonResp)
}

// writeJSONRPCError writes a JSON-RPC error response.
func writeJSONRPCError(w http.ResponseWriter, r *http.Request, err *a2a.Error, id interface{}) {
	response := a2a.JSONRPCResponse{
		JSONRPC: "2.0",
		Error:   err.ToJSONRPCError(),
		ID:      id,
	}

	// Marshal response
	jsonResp, marshalErr := json.Marshal(response)
	if marshalErr != nil {
		// If marshalling fails, return a simple error
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Determine HTTP status code based on error code
	httpStatus := http.StatusOK // Default for valid JSON-RPC errors
	if err.Code == a2a.CodeAuthenticationRequired || err.Code == a2a.CodeAuthenticationFailed {
		httpStatus = http.StatusUnauthorized
	} else if err.Code == a2a.CodeMethodNotFound {
		httpStatus = http.StatusNotFound
	} else if err.Code == a2a.CodeInvalidRequest || err.Code == a2a.CodeInvalidParams {
		httpStatus = http.StatusBadRequest
	} else if err.Code == a2a.CodeRateLimitExceeded {
		httpStatus = http.StatusTooManyRequests
	}

	// Write response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(httpStatus)
	w.Write(jsonResp)
}
