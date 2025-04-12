// Package server provides the server-side implementation of the A2A protocol.
package server

import (
	"context"

	"github.com/sammcj/go-a2a/a2a"
	"github.com/sammcj/go-a2a/llm"
	"github.com/sammcj/go-a2a/pkg/task"
)

// AgentEngine defines the interface for agent intelligence.
// Implementations of this interface provide the core logic for processing tasks
// and generating responses.
type AgentEngine interface {
	// ProcessTask processes a task and returns a channel for updates.
	// It takes a context for cancellation and a TaskContext containing the task details.
	// It returns a channel for yielding updates (status changes, artifacts) and any error that occurred.
	ProcessTask(ctx context.Context, taskCtx task.Context) (<-chan task.YieldUpdate, error)

	// GetCapabilities returns the agent's capabilities.
	GetCapabilities() AgentCapabilities
}

// AgentCapabilities represents the capabilities of an agent.
type AgentCapabilities struct {
	// SupportsStreaming indicates whether the agent supports streaming responses.
	SupportsStreaming bool

	// SupportedInputModalities is a list of input modalities the agent supports.
	SupportedInputModalities []string

	// SupportedOutputModalities is a list of output modalities the agent supports.
	SupportedOutputModalities []string
}

// BasicLLMAgent implements AgentEngine using an LLM.
type BasicLLMAgent struct {
	llm          llm.LLMInterface
	systemPrompt string
	skills       []a2a.AgentSkill
	capabilities AgentCapabilities
}

// NewBasicLLMAgent creates a new BasicLLMAgent.
func NewBasicLLMAgent(llmInterface llm.LLMInterface, systemPrompt string) *BasicLLMAgent {
	// Get model info to determine capabilities
	modelInfo := llmInterface.GetModelInfo()

	return &BasicLLMAgent{
		llm:          llmInterface,
		systemPrompt: systemPrompt,
		capabilities: AgentCapabilities{
			SupportsStreaming:         true,
			SupportedInputModalities:  modelInfo.InputModalities,
			SupportedOutputModalities: modelInfo.OutputModalities,
		},
	}
}

// ProcessTask implements AgentEngine.ProcessTask.
func (a *BasicLLMAgent) ProcessTask(ctx context.Context, taskCtx task.Context) (<-chan task.YieldUpdate, error) {
	updateChan := make(chan task.YieldUpdate)

	go func() {
		defer close(updateChan)

		// Extract the user's message
		userMessage := taskCtx.UserMessage
		var userText string
		for _, part := range userMessage.Parts {
			if textPart, ok := part.(a2a.TextPart); ok {
				userText = textPart.Text
				break
			}
		}

		// Send a working status update
		updateChan <- task.StatusUpdate{
			State: a2a.TaskStateWorking,
		}

		// Process the message with the LLM
		response, err := a.llm.Generate(ctx, userText, llm.WithSystemPrompt(a.systemPrompt))
		if err != nil {
			// Send a failed status update
			updateChan <- task.StatusUpdate{
				State: a2a.TaskStateFailed,
				Message: &a2a.Message{
					Role: a2a.RoleSystem,
					Parts: []a2a.Part{
						a2a.TextPart{
							Type: "text",
							Text: "Failed to process message: " + err.Error(),
						},
					},
				},
			}
			return
		}

		// Create a response message
		responseMessage := a2a.Message{
			Role: a2a.RoleAgent,
			Parts: []a2a.Part{
				a2a.TextPart{
					Type: "text",
					Text: response,
				},
			},
		}

		// Send a working status update with the response
		updateChan <- task.StatusUpdate{
			State:   a2a.TaskStateWorking,
			Message: &responseMessage,
		}

		// Send a completed status update
		updateChan <- task.StatusUpdate{
			State: a2a.TaskStateCompleted,
		}
	}()

	return updateChan, nil
}

// GetCapabilities implements AgentEngine.GetCapabilities.
func (a *BasicLLMAgent) GetCapabilities() AgentCapabilities {
	return a.capabilities
}

// ToolAugmentedAgent implements AgentEngine using an LLM with tools.
type ToolAugmentedAgent struct {
	llm          llm.LLMInterface
	tools        []Tool
	systemPrompt string
	capabilities AgentCapabilities
}

// Tool defines a tool that can be used by an agent.
type Tool interface {
	// Name returns the name of the tool.
	Name() string

	// Description returns a description of the tool.
	Description() string

	// Execute executes the tool with the given parameters.
	Execute(ctx context.Context, params map[string]interface{}) (interface{}, error)
}

// NewToolAugmentedAgent creates a new ToolAugmentedAgent.
func NewToolAugmentedAgent(llmInterface llm.LLMInterface, tools []Tool) *ToolAugmentedAgent {
	// Get model info to determine capabilities
	modelInfo := llmInterface.GetModelInfo()

	// Create a system prompt that includes tool descriptions
	systemPrompt := "You are a helpful assistant with access to the following tools:\n\n"
	for _, tool := range tools {
		systemPrompt += "- " + tool.Name() + ": " + tool.Description() + "\n"
	}
	systemPrompt += "\nWhen you need to use a tool, specify the tool name and parameters in your response."

	return &ToolAugmentedAgent{
		llm:          llmInterface,
		tools:        tools,
		systemPrompt: systemPrompt,
		capabilities: AgentCapabilities{
			SupportsStreaming:         true,
			SupportedInputModalities:  modelInfo.InputModalities,
			SupportedOutputModalities: modelInfo.OutputModalities,
		},
	}
}

// ProcessTask implements AgentEngine.ProcessTask.
func (a *ToolAugmentedAgent) ProcessTask(ctx context.Context, taskCtx task.Context) (<-chan task.YieldUpdate, error) {
	updateChan := make(chan task.YieldUpdate)

	go func() {
		defer close(updateChan)

		// Extract the user's message
		userMessage := taskCtx.UserMessage
		var userText string
		for _, part := range userMessage.Parts {
			if textPart, ok := part.(a2a.TextPart); ok {
				userText = textPart.Text
				break
			}
		}

		// Send a working status update
		updateChan <- task.StatusUpdate{
			State: a2a.TaskStateWorking,
		}

		// Process the message with the LLM
		response, err := a.llm.Generate(ctx, userText, llm.WithSystemPrompt(a.systemPrompt))
		if err != nil {
			// Send a failed status update
			updateChan <- task.StatusUpdate{
				State: a2a.TaskStateFailed,
				Message: &a2a.Message{
					Role: a2a.RoleSystem,
					Parts: []a2a.Part{
						a2a.TextPart{
							Type: "text",
							Text: "Failed to process message: " + err.Error(),
						},
					},
				},
			}
			return
		}

		// TODO: Parse the response to identify tool usage and execute tools
		// This is a simplified implementation that doesn't actually execute tools

		// Create a response message
		responseMessage := a2a.Message{
			Role: a2a.RoleAgent,
			Parts: []a2a.Part{
				a2a.TextPart{
					Type: "text",
					Text: response,
				},
			},
		}

		// Send a working status update with the response
		updateChan <- task.StatusUpdate{
			State:   a2a.TaskStateWorking,
			Message: &responseMessage,
		}

		// Send a completed status update
		updateChan <- task.StatusUpdate{
			State: a2a.TaskStateCompleted,
		}
	}()

	return updateChan, nil
}

// GetCapabilities implements AgentEngine.GetCapabilities.
func (a *ToolAugmentedAgent) GetCapabilities() AgentCapabilities {
	return a.capabilities
}
