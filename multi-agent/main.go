package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/sammcj/go-a2a/a2a"
	"github.com/sammcj/go-a2a/client"
	"github.com/sammcj/go-a2a/llm"
	"github.com/sammcj/go-a2a/llm/gollm"
	"github.com/sammcj/go-a2a/server"
)

// AgentConfig represents the configuration for an agent.
type AgentConfig struct {
	ListenAddress string                 `json:"listenAddress"`
	AgentCardPath string                 `json:"agentCardPath"`
	A2APathPrefix string                 `json:"a2aPathPrefix"`
	LogLevel      string                 `json:"logLevel"`
	LLMConfig     LLMConfig              `json:"llmConfig"`
	MCPConfig     MCPConfig              `json:"mcpConfig"`
	AgentCard     a2a.AgentCard          `json:"agentCard"`
	Extra         map[string]interface{} `json:"extra"`
}

// LLMConfig represents the configuration for an LLM.
type LLMConfig struct {
	Provider     string `json:"provider"`
	Model        string `json:"model"`
	APIKey       string `json:"apiKey"`
	SystemPrompt string `json:"systemPrompt"`
	BaseURL      string `json:"baseUrl"`
}

// Agent represents an A2A agent.
type Agent struct {
	Config     AgentConfig
	Server     *server.Server
	Client     *client.Client
	MCPClient  *CustomMCPClient
	TaskRouter *TaskRouter
}

// TaskRouter routes tasks between agents.
type TaskRouter struct {
	WebAgent      *client.Client
	CustomerAgent *client.Client
	ReasonerAgent *client.Client
	mu            sync.Mutex
}

// NewTaskRouter creates a new TaskRouter.
func NewTaskRouter() *TaskRouter {
	return &TaskRouter{
		mu: sync.Mutex{},
	}
}

// SetWebAgent sets the web agent client.
func (r *TaskRouter) SetWebAgent(client *client.Client) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.WebAgent = client
}

// SetCustomerAgent sets the customer agent client.
func (r *TaskRouter) SetCustomerAgent(client *client.Client) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.CustomerAgent = client
}

// SetReasonerAgent sets the reasoner agent client.
func (r *TaskRouter) SetReasonerAgent(client *client.Client) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.ReasonerAgent = client
}

// RouteToWebAgent routes a task to the web agent.
func (r *TaskRouter) RouteToWebAgent(ctx context.Context, message a2a.Message) (*a2a.Task, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.WebAgent == nil {
		return nil, fmt.Errorf("web agent not set")
	}
	return r.WebAgent.SendTask(ctx, &a2a.TaskSendParams{
		Message: message,
	})
}

// RouteToReasonerAgent routes a task to the reasoner agent.
func (r *TaskRouter) RouteToReasonerAgent(ctx context.Context, message a2a.Message) (*a2a.Task, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.ReasonerAgent == nil {
		return nil, fmt.Errorf("reasoner agent not set")
	}
	return r.ReasonerAgent.SendTask(ctx, &a2a.TaskSendParams{
		Message: message,
	})
}

// LoadAgentConfig loads an agent configuration from a file.
func LoadAgentConfig(path string) (*AgentConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config AgentConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Replace environment variables in the configuration
	if config.LLMConfig.APIKey != "" && config.LLMConfig.APIKey[0] == '$' {
		envVar := config.LLMConfig.APIKey[1:]
		if envVar[0] == '{' && envVar[len(envVar)-1] == '}' {
			envVar = envVar[1 : len(envVar)-1]
		}
		config.LLMConfig.APIKey = os.Getenv(envVar)
	}

	if config.LLMConfig.BaseURL != "" && config.LLMConfig.BaseURL[0] == '$' {
		envVar := config.LLMConfig.BaseURL[1:]
		if envVar[0] == '{' && envVar[len(envVar)-1] == '}' {
			envVar = envVar[1 : len(envVar)-1]
		}
		baseURL := os.Getenv(envVar)
		if baseURL != "" {
			config.LLMConfig.BaseURL = baseURL
		} else {
			// Default to OpenAI's API if not specified
			if config.LLMConfig.Provider == "openai" {
				config.LLMConfig.BaseURL = "https://api.openai.com/v1"
			}
		}
	}

	// Process MCP tool configurations
	if len(config.MCPConfig.Tools) > 0 {
		for i, tool := range config.MCPConfig.Tools {
			if env, ok := tool.Config["env"].(map[string]interface{}); ok {
				for key, value := range env {
					if strValue, ok := value.(string); ok && strValue != "" && strValue[0] == '$' {
						envVar := strValue[1:]
						if envVar[0] == '{' && envVar[len(envVar)-1] == '}' {
							envVar = envVar[1 : len(envVar)-1]
						}
						envValue := os.Getenv(envVar)
						if envValue != "" {
							env[key] = envValue
							tool.Config["env"] = env
							config.MCPConfig.Tools[i] = tool
						}
					}
				}
			}
		}
	}

	return &config, nil
}

// CreateAgent creates an A2A agent from a configuration.
func CreateAgent(config *AgentConfig, taskRouter *TaskRouter) (*Agent, error) {
	agent := &Agent{
		Config:     *config,
		TaskRouter: taskRouter,
	}

	// Create MCP client if MCP config is provided
	if len(config.MCPConfig.Tools) > 0 {
		agent.MCPClient = NewCustomMCPClient(config.MCPConfig)
	}

	// Create gollm options
	gollmOptions := []gollm.Option{
		gollm.WithProvider(config.LLMConfig.Provider),
		gollm.WithModel(config.LLMConfig.Model),
	}

	if config.LLMConfig.APIKey != "" {
		gollmOptions = append(gollmOptions, gollm.WithAPIKey(config.LLMConfig.APIKey))
	}

	// Create server options
	serverOptions := []server.Option{
		server.WithAgentCard(&config.AgentCard),
		server.WithListenAddress(config.ListenAddress),
		server.WithA2APathPrefix(config.A2APathPrefix),
	}

	// Add MCP client if available
	if agent.MCPClient != nil {
		// Create gollm adapter
		adapter, err := gollm.NewAdapter(gollmOptions...)
		if err != nil {
			return nil, fmt.Errorf("failed to create gollm adapter: %w", err)
		}

		// Add MCP tool augmented agent
		serverOptions = append(serverOptions, server.WithMCPToolAugmentedAgent(adapter, agent.MCPClient))
	} else {
		// Create task handler based on agent type
		var taskHandler func(ctx context.Context, taskCtx server.TaskContext) (<-chan server.TaskYieldUpdate, error)
		switch config.AgentCard.ID {
		case "web-agent":
			taskHandler = createWebAgentHandler(config.LLMConfig.SystemPrompt, gollmOptions)
		case "customer-agent":
			taskHandler = createCustomerAgentHandler(config.LLMConfig.SystemPrompt, gollmOptions, taskRouter)
		case "reasoner-agent":
			taskHandler = createReasonerAgentHandler(config.LLMConfig.SystemPrompt, gollmOptions)
		default:
			return nil, fmt.Errorf("unknown agent type: %s", config.AgentCard.ID)
		}

		// Add task handler
		serverOptions = append(serverOptions, server.WithTaskHandler(taskHandler))
	}

	// Create server
	a2aServer, err := server.NewServer(serverOptions...)
	if err != nil {
		return nil, fmt.Errorf("failed to create A2A server: %w", err)
	}
	agent.Server = a2aServer

	// Create client
	a2aClient, err := client.NewClient(
		client.WithBaseURL(fmt.Sprintf("http://localhost%s", config.ListenAddress)),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create A2A client: %w", err)
	}
	agent.Client = a2aClient

	return agent, nil
}

// createWebAgentHandler creates a task handler for the web agent.
func createWebAgentHandler(systemPrompt string, gollmOptions []gollm.Option) func(ctx context.Context, taskCtx server.TaskContext) (<-chan server.TaskYieldUpdate, error) {
	return func(ctx context.Context, taskCtx server.TaskContext) (<-chan server.TaskYieldUpdate, error) {
		updateChan := make(chan server.TaskYieldUpdate)

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
			updateChan <- server.StatusUpdate{
				State: a2a.TaskStateWorking,
			}

			// Create gollm adapter
			adapter, err := gollm.NewAdapter(gollmOptions...)
			if err != nil {
				updateChan <- server.StatusUpdate{
					State: a2a.TaskStateFailed,
					Message: &a2a.Message{
						Role: a2a.RoleSystem,
						Parts: []a2a.Part{
							a2a.TextPart{
								Type: "text",
								Text: fmt.Sprintf("Failed to create gollm adapter: %v", err),
							},
						},
					},
				}
				return
			}

			// Process the message with the LLM
			response, err := adapter.Generate(ctx, userText, llm.WithSystemPrompt(systemPrompt))
			if err != nil {
				updateChan <- server.StatusUpdate{
					State: a2a.TaskStateFailed,
					Message: &a2a.Message{
						Role: a2a.RoleSystem,
						Parts: []a2a.Part{
							a2a.TextPart{
								Type: "text",
								Text: fmt.Sprintf("Failed to generate response: %v", err),
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
}

// createCustomerAgentHandler creates a task handler for the customer agent.
func createCustomerAgentHandler(systemPrompt string, gollmOptions []gollm.Option, taskRouter *TaskRouter) func(ctx context.Context, taskCtx server.TaskContext) (<-chan server.TaskYieldUpdate, error) {
	return func(ctx context.Context, taskCtx server.TaskContext) (<-chan server.TaskYieldUpdate, error) {
		updateChan := make(chan server.TaskYieldUpdate)

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
			updateChan <- server.StatusUpdate{
				State: a2a.TaskStateWorking,
			}

			// Create gollm adapter
			adapter, err := gollm.NewAdapter(gollmOptions...)
			if err != nil {
				updateChan <- server.StatusUpdate{
					State: a2a.TaskStateFailed,
					Message: &a2a.Message{
						Role: a2a.RoleSystem,
						Parts: []a2a.Part{
							a2a.TextPart{
								Type: "text",
								Text: fmt.Sprintf("Failed to create gollm adapter: %v", err),
							},
						},
					},
				}
				return
			}

			// Determine if we need to route to another agent
			routePrompt := fmt.Sprintf(`
User message: %s

Based on this message, determine if it requires:
1. Web search or information retrieval (route to web agent)
2. Complex reasoning or analysis (route to reasoner agent)
3. Direct response (handle directly)

Respond with one of: "web", "reasoner", or "direct"
`, userText)

			routeDecision, err := adapter.Generate(ctx, routePrompt, llm.WithSystemPrompt("You are a routing agent that determines which agent should handle a user request."))
			if err != nil {
				updateChan <- server.StatusUpdate{
					State: a2a.TaskStateFailed,
					Message: &a2a.Message{
						Role: a2a.RoleSystem,
						Parts: []a2a.Part{
							a2a.TextPart{
								Type: "text",
								Text: fmt.Sprintf("Failed to determine routing: %v", err),
							},
						},
					},
				}
				return
			}

			// Process based on routing decision
			var finalResponse string
			if routeDecision == "web" || routeDecision == "WEB" {
				// Route to web agent
				webMessage := a2a.Message{
					Role: a2a.RoleUser,
					Parts: []a2a.Part{
						a2a.TextPart{
							Type: "text",
							Text: userText,
						},
					},
				}

				webTask, err := taskRouter.RouteToWebAgent(ctx, webMessage)
				if err != nil {
					updateChan <- server.StatusUpdate{
						State: a2a.TaskStateFailed,
						Message: &a2a.Message{
							Role: a2a.RoleSystem,
							Parts: []a2a.Part{
								a2a.TextPart{
									Type: "text",
									Text: fmt.Sprintf("Failed to route to web agent: %v", err),
								},
							},
						},
					}
					return
				}

				// Get the web agent's response
				finalResponse = fmt.Sprintf("I've consulted our web agent for this query. Here's what I found:\n\n%s", getTaskResponse(webTask))
			} else if routeDecision == "reasoner" || routeDecision == "REASONER" {
				// Route to reasoner agent
				reasonerMessage := a2a.Message{
					Role: a2a.RoleUser,
					Parts: []a2a.Part{
						a2a.TextPart{
							Type: "text",
							Text: userText,
						},
					},
				}

				reasonerTask, err := taskRouter.RouteToReasonerAgent(ctx, reasonerMessage)
				if err != nil {
					updateChan <- server.StatusUpdate{
						State: a2a.TaskStateFailed,
						Message: &a2a.Message{
							Role: a2a.RoleSystem,
							Parts: []a2a.Part{
								a2a.TextPart{
									Type: "text",
									Text: fmt.Sprintf("Failed to route to reasoner agent: %v", err),
								},
							},
						},
					}
					return
				}

				// Get the reasoner agent's response
				finalResponse = fmt.Sprintf("I've consulted our reasoning specialist for this query. Here's the analysis:\n\n%s", getTaskResponse(reasonerTask))
			} else {
				// Handle directly
				directResponse, err := adapter.Generate(ctx, userText, llm.WithSystemPrompt(systemPrompt))
				if err != nil {
					updateChan <- server.StatusUpdate{
						State: a2a.TaskStateFailed,
						Message: &a2a.Message{
							Role: a2a.RoleSystem,
							Parts: []a2a.Part{
								a2a.TextPart{
									Type: "text",
									Text: fmt.Sprintf("Failed to generate direct response: %v", err),
								},
							},
						},
					}
					return
				}
				finalResponse = directResponse
			}

			// Create a response message
			responseMessage := a2a.Message{
				Role: a2a.RoleAgent,
				Parts: []a2a.Part{
					a2a.TextPart{
						Type: "text",
						Text: finalResponse,
					},
				},
			}

			// Send a working status update with the response
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
}

// createReasonerAgentHandler creates a task handler for the reasoner agent.
func createReasonerAgentHandler(systemPrompt string, gollmOptions []gollm.Option) func(ctx context.Context, taskCtx server.TaskContext) (<-chan server.TaskYieldUpdate, error) {
	return func(ctx context.Context, taskCtx server.TaskContext) (<-chan server.TaskYieldUpdate, error) {
		updateChan := make(chan server.TaskYieldUpdate)

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
			updateChan <- server.StatusUpdate{
				State: a2a.TaskStateWorking,
			}

			// Create gollm adapter
			adapter, err := gollm.NewAdapter(gollmOptions...)
			if err != nil {
				updateChan <- server.StatusUpdate{
					State: a2a.TaskStateFailed,
					Message: &a2a.Message{
						Role: a2a.RoleSystem,
						Parts: []a2a.Part{
							a2a.TextPart{
								Type: "text",
								Text: fmt.Sprintf("Failed to create gollm adapter: %v", err),
							},
						},
					},
				}
				return
			}

			// Process the message with the LLM
			response, err := adapter.Generate(ctx, userText, llm.WithSystemPrompt(systemPrompt))
			if err != nil {
				updateChan <- server.StatusUpdate{
					State: a2a.TaskStateFailed,
					Message: &a2a.Message{
						Role: a2a.RoleSystem,
						Parts: []a2a.Part{
							a2a.TextPart{
								Type: "text",
								Text: fmt.Sprintf("Failed to generate response: %v", err),
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
}

// getTaskResponse extracts the response text from a task.
func getTaskResponse(task *a2a.Task) string {
	if task == nil || task.Status.Message == nil {
		return "No response available"
	}

	var responseText string
	for _, part := range task.Status.Message.Parts {
		if textPart, ok := part.(a2a.TextPart); ok {
			responseText += textPart.Text
		}
	}

	return responseText
}

func main() {
	// Create task router
	taskRouter := NewTaskRouter()

	// Load agent configurations
	webAgentConfig, err := LoadAgentConfig("config/web-agent.json")
	if err != nil {
		log.Fatalf("Failed to load web agent config: %v", err)
	}

	customerAgentConfig, err := LoadAgentConfig("config/customer-agent.json")
	if err != nil {
		log.Fatalf("Failed to load customer agent config: %v", err)
	}

	reasonerAgentConfig, err := LoadAgentConfig("config/reasoner-agent.json")
	if err != nil {
		log.Fatalf("Failed to load reasoner agent config: %v", err)
	}

	// Create agents
	webAgent, err := CreateAgent(webAgentConfig, taskRouter)
	if err != nil {
		log.Fatalf("Failed to create web agent: %v", err)
	}

	customerAgent, err := CreateAgent(customerAgentConfig, taskRouter)
	if err != nil {
		log.Fatalf("Failed to create customer agent: %v", err)
	}

	reasonerAgent, err := CreateAgent(reasonerAgentConfig, taskRouter)
	if err != nil {
		log.Fatalf("Failed to create reasoner agent: %v", err)
	}

	// Set agent clients in task router
	taskRouter.SetWebAgent(webAgent.Client)
	taskRouter.SetCustomerAgent(customerAgent.Client)
	taskRouter.SetReasonerAgent(reasonerAgent.Client)

	// Start agents in goroutines
	var wg sync.WaitGroup
	wg.Add(3)

	go func() {
		defer wg.Done()
		log.Printf("Starting web agent on %s", webAgentConfig.ListenAddress)
		if err := webAgent.Server.Start(); err != nil {
			log.Printf("Web agent error: %v", err)
		}
	}()

	go func() {
		defer wg.Done()
		log.Printf("Starting customer agent on %s", customerAgentConfig.ListenAddress)
		if err := customerAgent.Server.Start(); err != nil {
			log.Printf("Customer agent error: %v", err)
		}
	}()

	go func() {
		defer wg.Done()
		log.Printf("Starting reasoner agent on %s", reasonerAgentConfig.ListenAddress)
		if err := reasonerAgent.Server.Start(); err != nil {
			log.Printf("Reasoner agent error: %v", err)
		}
	}()

	// Wait for agents to start
	time.Sleep(1 * time.Second)
	log.Println("All agents started successfully")
	log.Printf("Web agent: http://localhost%s", webAgentConfig.ListenAddress)
	log.Printf("Customer agent: http://localhost%s", customerAgentConfig.ListenAddress)
	log.Printf("Reasoner agent: http://localhost%s", reasonerAgentConfig.ListenAddress)

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Println("Shutting down agents...")
	// TODO: Implement graceful shutdown for agents

	wg.Wait()
	log.Println("All agents stopped")
}
