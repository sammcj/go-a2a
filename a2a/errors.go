package a2a

import (
	"fmt"
)

// JSONRPCError represents a JSON-RPC error object.
// Defined here as it's closely related to the error handling logic.
type JSONRPCError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// --- Standard JSON-RPC Error Codes ---
// See: https://www.jsonrpc.org/specification#error_object

const (
	// JSON-RPC Defined Codes
	CodeParseError     = -32700
	CodeInvalidRequest = -32600
	CodeMethodNotFound = -32601
	CodeInvalidParams  = -32602
	CodeInternalError  = -32603
	// -32000 to -32099: Server error (Implementation-defined)

	// A2A Specific Error Codes (within server error range)
	CodeTaskNotFound           = -32000
	CodeSkillNotFound          = -32001
	CodeSessionNotFound        = -32002 // If sessions are strictly enforced
	CodeAuthenticationRequired = -32010
	CodeAuthenticationFailed   = -32011
	CodeOperationNotSupported  = -32020 // e.g., streaming requested but not supported
	CodeTaskCancelled          = -32030 // Explicitly cancelled by client
	CodeTaskFailed             = -32031 // Task execution failed internally
	CodePushNotificationFailed = -32040
	CodeRateLimitExceeded      = -32050
	// Add more as needed
)

// Error represents an A2A error with a corresponding JSON-RPC code.
type Error struct {
	Code    int         // The JSON-RPC error code.
	Message string      // A human-readable error message.
	Data    interface{} // Optional additional data.
	cause   error       // Optional underlying error.
}

// Error implements the standard Go error interface.
func (e *Error) Error() string {
	if e.cause != nil {
		return fmt.Sprintf("a2a error: code=%d, message=%s, cause=%v", e.Code, e.Message, e.cause)
	}
	return fmt.Sprintf("a2a error: code=%d, message=%s", e.Code, e.Message)
}

// Unwrap returns the underlying cause of the error, if any.
func (e *Error) Unwrap() error {
	return e.cause
}

// ToJSONRPCError converts an A2A Error to a JSONRPCError struct.
func (e *Error) ToJSONRPCError() *JSONRPCError {
	return &JSONRPCError{
		Code:    e.Code,
		Message: e.Message,
		Data:    e.Data,
	}
}

// NewError creates a new A2A Error.
func NewError(code int, message string) *Error {
	return &Error{Code: code, Message: message}
}

// NewErrorf creates a new A2A Error with formatted message.
func NewErrorf(code int, format string, args ...interface{}) *Error {
	return &Error{Code: code, Message: fmt.Sprintf(format, args...)}
}

// WrapError creates a new A2A Error that wraps an existing error.
func WrapError(cause error, code int, message string) *Error {
	return &Error{Code: code, Message: message, cause: cause}
}

// WrapErrorf creates a new A2A Error that wraps an existing error with a formatted message.
func WrapErrorf(cause error, code int, format string, args ...interface{}) *Error {
	return &Error{Code: code, Message: fmt.Sprintf(format, args...), cause: cause}
}

// --- Predefined Errors ---

func ErrParseError(cause error) *Error {
	return WrapError(cause, CodeParseError, "Parse error")
}

func ErrInvalidRequest(message string) *Error {
	if message == "" {
		message = "Invalid Request"
	}
	return NewError(CodeInvalidRequest, message)
}

func ErrMethodNotFound(method string) *Error {
	return NewErrorf(CodeMethodNotFound, "Method not found: %s", method)
}

func ErrInvalidParams(message string) *Error {
	if message == "" {
		message = "Invalid params"
	}
	return NewError(CodeInvalidParams, message)
}

func ErrInternalError(cause error) *Error {
	return WrapError(cause, CodeInternalError, "Internal error")
}

func ErrTaskNotFound(taskId string) *Error {
	return NewErrorf(CodeTaskNotFound, "Task not found: %s", taskId)
}

func ErrSkillNotFound(skillId string) *Error {
	return NewErrorf(CodeSkillNotFound, "Skill not found: %s", skillId)
}

func ErrAuthenticationRequired() *Error {
	return NewError(CodeAuthenticationRequired, "Authentication required")
}

func ErrAuthenticationFailed(message string) *Error {
	if message == "" {
		message = "Authentication failed"
	}
	return NewError(CodeAuthenticationFailed, message)
}

func ErrOperationNotSupported(operation string) *Error {
	return NewErrorf(CodeOperationNotSupported, "Operation not supported: %s", operation)
}

func ErrTaskCancelled(taskId string) *Error {
	return NewErrorf(CodeTaskCancelled, "Task cancelled: %s", taskId)
}

func ErrTaskFailed(taskId string, cause error) *Error {
	return WrapErrorf(cause, CodeTaskFailed, "Task failed: %s", taskId)
}

func ErrPushNotificationFailed(taskId string, cause error) *Error {
	return WrapErrorf(cause, CodePushNotificationFailed, "Push notification failed for task: %s", taskId)
}
