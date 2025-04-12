package a2a

import (
	"encoding/json"
	"time"
)

// --- Enums / Constants ---

// TaskState represents the state of a task.
type TaskState string

const (
	TaskStateSubmitted     TaskState = "submitted"
	TaskStateWorking       TaskState = "working"
	TaskStateInputRequired TaskState = "input-required"
	TaskStateCompleted     TaskState = "completed"
	TaskStateFailed        TaskState = "failed"
	TaskStateCancelled     TaskState = "cancelled" // Corrected spelling
)

// Role represents the role of a message sender.
type Role string

const (
	RoleSystem Role = "system"
	RoleUser   Role = "user"
	RoleAgent  Role = "agent"
)

// --- Core A2A Objects ---

// Task represents an A2A task.
type Task struct {
	ID          string       `json:"id"`
	SessionID   *string      `json:"sessionId,omitempty"` // Optional session ID
	Status      TaskStatus   `json:"status"`
	History     []Message    `json:"history"` // Chronological order
	Artifacts   []Artifact   `json:"artifacts"`
	InputSchema *interface{} `json:"inputSchema,omitempty"` // Optional JSON schema for input
	// TODO: Add other potential fields if needed based on spec refinement
}

// TaskStatus represents the status details of a task.
type TaskStatus struct {
	State     TaskState `json:"state"`
	Timestamp time.Time `json:"timestamp"`
	Message   *Message  `json:"message,omitempty"` // Optional message associated with the status change
}

// Artifact represents an artifact associated with a task.
type Artifact struct {
	ID        string      `json:"id"`
	TaskID    string      `json:"taskId"`
	Timestamp time.Time   `json:"timestamp"`
	Part      Part        `json:"part"` // The actual content of the artifact
	Metadata  interface{} `json:"metadata,omitempty"`
}

// Message represents a message within a task's history.
type Message struct {
	Role      Role        `json:"role"`
	Timestamp time.Time   `json:"timestamp"`
	Parts     []Part      `json:"parts"`
	Metadata  interface{} `json:"metadata,omitempty"`
}

// Part represents a piece of content within a message or artifact.
// Using an interface{} for now; might refine to a specific interface or tagged union struct later.
// For marshalling/unmarshalling, custom logic might be needed depending on the final approach.
type Part interface {
	// isPart is a marker method for the Part interface (or use type embedding)
	isPart()
}

// TextPart represents a plain text part.
type TextPart struct {
	Type string `json:"type"` // Should always be "text"
	Text string `json:"text"`
}

func (TextPart) isPart() {}

// FilePart represents a reference to a file, potentially with content.
type FilePart struct {
	Type        string       `json:"type"` // Should always be "file"
	Filename    string       `json:"filename"`
	MimeType    string       `json:"mimeType"`
	URI         *string      `json:"uri,omitempty"` // Optional URI to fetch the file
	Description *string      `json:"description,omitempty"`
	Content     *FileContent `json:"content,omitempty"` // Optional inline content
}

func (FilePart) isPart() {}

// FileContent holds the actual file content, typically base64 encoded.
type FileContent struct {
	Encoding string `json:"encoding"` // e.g., "base64"
	Data     string `json:"data"`
}

// DataPart represents structured data.
type DataPart struct {
	Type     string      `json:"type"`     // Should always be "data"
	MimeType string      `json:"mimeType"` // e.g., "application/json"
	Data     interface{} `json:"data"`     // The actual structured data
}

func (DataPart) isPart() {}

// --- Agent Card ---

// AgentCard describes an A2A agent.
type AgentCard struct {
	A2AVersion       string                `json:"a2aVersion"` // e.g., "1.0"
	ID               string                `json:"id"`
	Name             string                `json:"name"`
	Description      *string               `json:"description,omitempty"`
	IconURI          *string               `json:"iconUri,omitempty"`
	Provider         *AgentProvider        `json:"provider,omitempty"`
	Skills           []AgentSkill          `json:"skills"`
	Capabilities     *AgentCapabilities    `json:"capabilities,omitempty"`
	Authentication   []AgentAuthentication `json:"authentication,omitempty"`
	ContactEmail     *string               `json:"contactEmail,omitempty"`
	LegalInfoURI     *string               `json:"legalInfoUri,omitempty"`
	HomepageURI      *string               `json:"homepageUri,omitempty"`
	DocumentationURI *string               `json:"documentationUri,omitempty"`
}

// AgentProvider describes the provider of the agent.
type AgentProvider struct {
	Name string  `json:"name"`
	URI  *string `json:"uri,omitempty"`
}

// AgentSkill describes a skill the agent possesses.
type AgentSkill struct {
	ID             string      `json:"id"` // Unique ID for the skill within the agent
	Name           string      `json:"name"`
	Description    *string     `json:"description,omitempty"`
	InputSchema    interface{} `json:"inputSchema,omitempty"`    // JSON Schema for task input
	ArtifactSchema interface{} `json:"artifactSchema,omitempty"` // JSON Schema for artifacts produced
}

// AgentCapabilities describes the capabilities of the agent.
type AgentCapabilities struct {
	SupportsStreaming        bool `json:"supportsStreaming"`
	SupportsSessions         bool `json:"supportsSessions"`
	SupportsPushNotification bool `json:"supportsPushNotification"`
	// Add other capabilities as defined in the spec
}

// AgentAuthentication describes an authentication method supported by the agent.
type AgentAuthentication struct {
	Type          string      `json:"type"` // e.g., "bearer", "oauth2"
	Scheme        *string     `json:"scheme,omitempty"`
	Configuration interface{} `json:"configuration,omitempty"` // Specific details for the auth type
}

// --- Push Notifications ---

// PushNotificationConfig holds the configuration for push notifications for a task.
type PushNotificationConfig struct {
	TaskID           string              `json:"taskId"`
	URL              string              `json:"url"`
	Authentication   *AuthenticationInfo `json:"authentication,omitempty"`
	IncludeTaskData  *bool               `json:"includeTaskData,omitempty"`  // Default true
	IncludeArtifacts *bool               `json:"includeArtifacts,omitempty"` // Default false
}

// AuthenticationInfo provides details for authenticating push notification requests.
type AuthenticationInfo struct {
	Type          string      `json:"type"`          // e.g., "bearer", "header"
	Configuration interface{} `json:"configuration"` // e.g., {"token": "..."} or {"headerName": "X-API-Key", "value": "..."}
}

// --- JSON-RPC Structures ---

// JSONRPCRequest represents a JSON-RPC request object.
type JSONRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"` // Should be "2.0"
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"` // Use RawMessage to delay parsing
	ID      interface{}     `json:"id,omitempty"`     // Request ID (string, number, or null)
}

// JSONRPCResponse represents a JSON-RPC response object.
type JSONRPCResponse struct {
	JSONRPC string        `json:"jsonrpc"` // Should be "2.0"
	Result  interface{}   `json:"result,omitempty"`
	Error   *JSONRPCError `json:"error,omitempty"`
	ID      interface{}   `json:"id"` // Should match the request ID
}

// --- Specific Method Params/Results (Examples - To be expanded) ---

// TaskSendParams represents the parameters for the tasks/send method.
type TaskSendParams struct {
	TaskID      *string     `json:"taskId,omitempty"` // For resuming
	SessionID   *string     `json:"sessionId,omitempty"`
	SkillID     *string     `json:"skillId,omitempty"`
	Message     Message     `json:"message"`
	InputSchema interface{} `json:"inputSchema,omitempty"` // Optional override/validation
	// Add other params like stream preference if needed
}

// TaskQueryParams represents the parameters for the tasks/get method.
type TaskQueryParams struct {
	TaskID string `json:"taskId"`
}

// TaskIdParams represents parameters containing just a taskId.
type TaskIdParams struct {
	TaskID string `json:"taskId"`
}

// TaskPushNotificationConfigParams represents parameters for setting push config.
type TaskPushNotificationConfigParams struct {
	TaskID           string              `json:"taskId"`
	URL              string              `json:"url"`
	Authentication   *AuthenticationInfo `json:"authentication,omitempty"`
	IncludeTaskData  *bool               `json:"includeTaskData,omitempty"`
	IncludeArtifacts *bool               `json:"includeArtifacts,omitempty"`
}

// --- SSE Event Structures ---

// SSEEvent is a helper struct for marshalling SSE events.
type SSEEvent struct {
	Event string      `json:"-"` // Event type (e.g., "taskStatusUpdate") - not part of JSON data
	Data  interface{} `json:"-"` // The actual data payload (will be marshalled to JSON)
	ID    *string     `json:"-"` // Optional event ID
	Retry *int        `json:"-"` // Optional retry interval
}

// TaskStatusUpdateEvent represents the data payload for a task status update SSE event.
type TaskStatusUpdateEvent struct {
	TaskID string     `json:"taskId"`
	Status TaskStatus `json:"status"`
}

// TaskArtifactUpdateEvent represents the data payload for a task artifact update SSE event.
type TaskArtifactUpdateEvent struct {
	TaskID   string   `json:"taskId"`
	Artifact Artifact `json:"artifact"`
}

// TODO: Add specific request/response structs for each A2A method (e.g., SendTaskRequest, SendTaskResponse).
// TODO: Refine Part handling (e.g., custom marshalling/unmarshalling if needed).
// TODO: Add error constants mapping to JSON-RPC codes (in errors.go).
