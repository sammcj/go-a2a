package main

import (
	"flag"
	"fmt"
	"os"
	"time"
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"os/signal"
	"strings"
	"syscall"
	"github.com/sammcj/go-a2a/pkg/config"
	"github.com/sammcj/go-a2a/a2a"
	"github.com/sammcj/go-a2a/client"
	"github.com/sammcj/go-a2a/cmd/common"
)

var (
	configFile    = flag.String("config", "", "Path to configuration file (JSON or YAML)")
	agentURL      = flag.String("url", "", "URL of the A2A agent")
	outputFormat  = flag.String("output", "pretty", "Output format (json, pretty)")
	authHeader    = flag.String("auth", "", "Authentication header (format: 'Name: Value')")
	timeout       = flag.Duration("timeout", 30*time.Second, "Request timeout")
	interactive   = flag.Bool("interactive", false, "Interactive mode")
)

func main() {
	// Define subcommands
	sendCmd := flag.NewFlagSet("send", flag.ExitOnError)
	sendMessage := sendCmd.String("message", "", "Message to send")
	sendFile := sendCmd.String("file", "", "File containing message to send")
	sendSkill := sendCmd.String("skill", "", "Skill ID to use")
	sendTaskID := sendCmd.String("task", "", "Task ID to resume")
	sendStream := sendCmd.Bool("stream", false, "Stream task updates")

	getCmd := flag.NewFlagSet("get", flag.ExitOnError)
	getTaskID := getCmd.String("task", "", "Task ID to get")

	cancelCmd := flag.NewFlagSet("cancel", flag.ExitOnError)
	cancelTaskID := cancelCmd.String("task", "", "Task ID to cancel")

	subscribeCmd := flag.NewFlagSet("subscribe", flag.ExitOnError)
	subscribeTaskID := subscribeCmd.String("task", "", "Task ID to subscribe to")
	subscribeLastEventID := subscribeCmd.String("last-event", "", "Last event ID received")

	pushCmd := flag.NewFlagSet("push", flag.ExitOnError)
	pushTaskID := pushCmd.String("task", "", "Task ID to configure push notifications for")
	pushURL := pushCmd.String("url", "", "URL to send push notifications to")
	pushAuth := pushCmd.String("auth", "", "Authentication for push notifications (format: 'type:value' or 'type:name:value')")
	pushIncludeTask := pushCmd.Bool("include-task", true, "Include task data in push notifications")
	pushIncludeArtifacts := pushCmd.Bool("include-artifacts", false, "Include artifacts in push notifications")
	pushGet := pushCmd.Bool("get", false, "Get push notification configuration instead of setting it")

	cardCmd := flag.NewFlagSet("card", flag.ExitOnError)

	// Parse command line flags
	flag.Parse()

	// Create logger
	logger := common.NewLogger(os.Stdout, "info")

	// Load configuration
	var config config.ClientConfig
	if *configFile != "" {
		logger.Info("Loading configuration from %s", *configFile)
		loadedConfig, err := common.LoadConfig[config.ClientConfig](*configFile)
		if err != nil {
			logger.Fatal("Failed to load configuration: %v", err)
		}
		config = *loadedConfig
	} else {
		// Use default configuration with command line overrides
		config = common.DefaultClientConfig()
		if *agentURL != "" {
			config.DefaultAgentURL = *agentURL
		}
		if *outputFormat != "" {
			config.OutputFormat = *outputFormat
		}
		if *authHeader != "" {
			if config.Authentication == nil {
				config.Authentication = make(map[string]interface{})
			}
			parts := strings.SplitN(*authHeader, ":", 2)
			if len(parts) == 2 {
				config.Authentication[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
			}
		}
	}

	// Check if a subcommand was provided
	if flag.NArg() == 0 {
		if *interactive {
			runInteractiveMode(config, logger)
			return
		}
		printUsage()
		os.Exit(1)
	}

	// Get the subcommand
	subcommand := flag.Arg(0)

	// Create HTTP client
	httpClient := &http.Client{
		Timeout: *timeout,
	}

	// Create A2A client options
	options := []client.Option{
		client.WithBaseURL(config.DefaultAgentURL),
		client.WithHTTPClient(httpClient),
	}

	// Add authentication headers
	for k, v := range config.Authentication {
		if strVal, ok := v.(string); ok {
			options = append(options, client.WithAuthHeader(k, strVal))
		}
	}

	// Create A2A client
	a2aClient, err := client.NewClient(options...)
	if err != nil {
		logger.Fatal("Failed to create A2A client: %v", err)
	}

	// Execute the appropriate subcommand
	switch subcommand {
	case "send":
		sendCmd.Parse(flag.Args()[1:])
		handleSendCommand(a2aClient, *sendMessage, *sendFile, *sendSkill, *sendTaskID, *sendStream, config, logger)
	case "get":
		getCmd.Parse(flag.Args()[1:])
		handleGetCommand(a2aClient, *getTaskID, config, logger)
	case "cancel":
		cancelCmd.Parse(flag.Args()[1:])
		handleCancelCommand(a2aClient, *cancelTaskID, config, logger)
	case "subscribe":
		subscribeCmd.Parse(flag.Args()[1:])
		handleSubscribeCommand(a2aClient, *subscribeTaskID, *subscribeLastEventID, config, logger)
	case "push":
		pushCmd.Parse(flag.Args()[1:])
		handlePushCommand(a2aClient, *pushTaskID, *pushURL, *pushAuth, *pushIncludeTask, *pushIncludeArtifacts, *pushGet, config, logger)
	case "card":
		cardCmd.Parse(flag.Args()[1:])
		handleCardCommand(a2aClient, config, logger)
	default:
		logger.Fatal("Unknown subcommand: %s", subcommand)
	}
}

// handleSendCommand handles the 'send' subcommand.
func handleSendCommand(a2aClient *client.Client, message, file, skillID, taskID string, stream bool, config config.ClientConfig, logger *common.Logger) {
	// Get message content
	var messageContent string
	if message != "" {
		messageContent = message
	} else if file != "" {
		content, err := ioutil.ReadFile(file)
		if err != nil {
			logger.Fatal("Failed to read file: %v", err)
		}
		messageContent = string(content)
	} else {
		logger.Fatal("Either -message or -file must be specified")
	}

	// Create message
	msg := a2a.Message{
		Role:      a2a.RoleUser,
		Timestamp: time.Now(),
		Parts: []a2a.Part{
			a2a.TextPart{
				Type: "text",
				Text: messageContent,
			},
		},
	}

	// Create params
	params := &a2a.TaskSendParams{
		Message: msg,
	}

	// Set optional fields
	if skillID != "" {
		params.SkillID = &skillID
	}
	if taskID != "" {
		params.TaskID = &taskID
	}
	if sessionID != "" {
		params.SessionID = &sessionID
	}

	// Send task
	if stream {
		// Create context with cancellation
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// Set up signal handling for graceful shutdown
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		go func() {
			<-sigChan
			logger.Info("Cancelling subscription...")
			cancel()
		}()

		// Send task with streaming
		updateChan, errChan := a2aClient.SendSubscribe(ctx, params)

		// Process updates
		for {
			select {
			case update, ok := <-updateChan:
				if !ok {
					// Channel closed, we're done
					return
				}
				printTaskUpdate(update, config.OutputFormat, logger)
			case err, ok := <-errChan:
				if !ok {
					// Channel closed
					continue
				}
				logger.Error("Error: %v", err)
				return
			case <-ctx.Done():
				logger.Info("Subscription cancelled")
				return
			}
		}
	} else {
		// Send task without streaming
		task, err := a2aClient.SendTask(context.Background(), params)
		if err != nil {
			logger.Fatal("Failed to send task: %v", err)
		}

		// Print task
		printTask(task, config.OutputFormat, logger)
	}
}

// handleGetCommand handles the 'get' subcommand.
func handleGetCommand(a2aClient *client.Client, taskID string, config config.ClientConfig, logger *common.Logger) {
	if taskID == "" {
		logger.Fatal("Task ID must be specified")
	}

	// Get task
	task, err := a2aClient.GetTask(context.Background(), taskID)
	if err != nil {
		logger.Fatal("Failed to get task: %v", err)
	}

	// Print task
	printTask(task, config.OutputFormat, logger)
}

// handleCancelCommand handles the 'cancel' subcommand.
func handleCancelCommand(a2aClient *client.Client, taskID string, config config.ClientConfig, logger *common.Logger) {
	if taskID == "" {
		logger.Fatal("Task ID must be specified")
	}

	// Cancel task
	task, err := a2aClient.CancelTask(context.Background(), taskID)
	if err != nil {
		logger.Fatal("Failed to cancel task: %v", err)
	}

	// Print task
	printTask(task, config.OutputFormat, logger)
}

// handleSubscribeCommand handles the 'subscribe' subcommand.
func handleSubscribeCommand(a2aClient *client.Client, taskID, lastEventID string, config config.ClientConfig, logger *common.Logger) {
	if taskID == "" {
		logger.Fatal("Task ID must be specified")
	}

	// Create context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Set up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		logger.Info("Cancelling subscription...")
		cancel()
	}()

	// Subscribe to task
	updateChan, errChan := a2aClient.Resubscribe(ctx, taskID, lastEventID)

	// Process updates
	for {
		select {
		case update, ok := <-updateChan:
			if !ok {
				// Channel closed, we're done
				return
			}
			printTaskUpdate(update, config.OutputFormat, logger)
		case err, ok := <-errChan:
			if !ok {
				// Channel closed
				continue
			}
			logger.Error("Error: %v", err)
			return
		case <-ctx.Done():
			logger.Info("Subscription cancelled")
			return
		}
	}
}

// handlePushCommand handles the 'push' subcommand.
func handlePushCommand(a2aClient *client.Client, taskID, url, auth string, includeTask, includeArtifacts, get bool, config config.ClientConfig, logger *common.Logger) {
	if taskID == "" {
		logger.Fatal("Task ID must be specified")
	}

	if get {
		// Get push notification configuration
		pushConfig, err := a2aClient.GetTaskPushNotification(context.Background(), taskID)
		if err != nil {
			logger.Fatal("Failed to get push notification configuration: %v", err)
		}

		// Print push notification configuration
		printPushConfig(pushConfig, config.OutputFormat, logger)
		return
	}

	if url == "" {
		logger.Fatal("URL must be specified")
	}

	// Create push notification configuration
	params := &a2a.TaskPushNotificationConfigParams{
		TaskID: taskID,
		URL:    url,
	}

	// Set optional fields
	if auth != "" {
		// Parse authentication
		authParts := strings.SplitN(auth, ":", 3)
		if len(authParts) < 2 {
			logger.Fatal("Invalid authentication format. Use 'type:value' or 'type:name:value'")
		}

		authType := authParts[0]
		var authConfig map[string]interface{}

		if authType == "bearer" {
			// Bearer token
			if len(authParts) != 2 {
				logger.Fatal("Bearer authentication should be in the format 'bearer:token'")
			}
			authConfig = map[string]interface{}{
				"token": authParts[1],
			}
		} else if authType == "header" {
			// Custom header
			if len(authParts) != 3 {
				logger.Fatal("Header authentication should be in the format 'header:name:value'")
			}
			authConfig = map[string]interface{}{
				"headerName": authParts[1],
				"value":      authParts[2],
			}
		} else {
			logger.Fatal("Unsupported authentication type: %s", authType)
		}

		params.Authentication = &a2a.AuthenticationInfo{
			Type:          authType,
			Configuration: authConfig,
		}
	}

	params.IncludeTaskData = &includeTask
	params.IncludeArtifacts = &includeArtifacts

	// Set push notification configuration
	pushConfig, err := a2aClient.SetTaskPushNotification(context.Background(), params)
	if err != nil {
		logger.Fatal("Failed to set push notification configuration: %v", err)
	}

	// Print push notification configuration
	printPushConfig(pushConfig, config.OutputFormat, logger)
}

// handleCardCommand handles the 'card' subcommand.
func handleCardCommand(a2aClient *client.Client, config config.ClientConfig, logger *common.Logger) {
	// Fetch agent card
	card, err := a2aClient.FetchAgentCard(context.Background())
	if err != nil {
		logger.Fatal("Failed to fetch agent card: %v", err)
	}

	// Print agent card
	printAgentCard(card, config.OutputFormat, logger)
}

// runInteractiveMode runs the client in interactive mode.
func runInteractiveMode(config config.ClientConfig, logger *common.Logger) {
	logger.Info("Interactive mode not implemented yet")
	// TODO: Implement interactive mode
}

// printTask prints a task.
func printTask(task *a2a.Task, format string, logger *common.Logger) {
	switch format {
	case "json":
		// Print as JSON
		jsonData, err := json.MarshalIndent(task, "", "  ")
		if err != nil {
			logger.Error("Failed to marshal task: %v", err)
			return
		}
		fmt.Println(string(jsonData))
	case "pretty":
		// Print in a human-readable format
		fmt.Printf("Task ID: %s\n", task.ID)
		if task.SessionID != nil {
			fmt.Printf("Session ID: %s\n", *task.SessionID)
		}
		fmt.Printf("Status: %s (%s)\n", task.Status.State, task.Status.Timestamp.Format(time.RFC3339))
		if task.Status.Message != nil {
			fmt.Printf("Status Message: %s\n", getMessageText(task.Status.Message))
		}
		fmt.Printf("History: %d messages\n", len(task.History))
		for i, msg := range task.History {
			fmt.Printf("  [%d] %s (%s): %s\n", i, msg.Role, msg.Timestamp.Format(time.RFC3339), getMessageText(&msg))
		}
		fmt.Printf("Artifacts: %d\n", len(task.Artifacts))
		for i, artifact := range task.Artifacts {
			fmt.Printf("  [%d] %s (%s): %s\n", i, artifact.ID, artifact.Timestamp.Format(time.RFC3339), getPartDescription(artifact.Part))
		}
	default:
		logger.Error("Unknown output format: %s", format)
	}
}

// printTaskUpdate prints a task update.
func printTaskUpdate(update client.TaskUpdate, format string, logger *common.Logger) {
	switch format {
	case "json":
		// Print as JSON
		jsonData, err := json.MarshalIndent(update, "", "  ")
		if err != nil {
			logger.Error("Failed to marshal task update: %v", err)
			return
		}
		fmt.Println(string(jsonData))
	case "pretty":
		// Print in a human-readable format
		switch update.Type {
		case "status":
			fmt.Printf("Status Update: %s\n", update.Status.State)
			if update.Status.Message != nil {
				fmt.Printf("  Message: %s\n", getMessageText(update.Status.Message))
			}
		case "artifact":
			fmt.Printf("Artifact Update: %s\n", update.Artifact.ID)
			fmt.Printf("  Type: %s\n", getPartDescription(update.Artifact.Part))
		default:
			fmt.Printf("Unknown update type: %s\n", update.Type)
		}
	default:
		logger.Error("Unknown output format: %s", format)
	}
}

// printPushConfig prints a push notification configuration.
func printPushConfig(config *a2a.PushNotificationConfig, format string, logger *common.Logger) {
	switch format {
	case "json":
		// Print as JSON
		jsonData, err := json.MarshalIndent(config, "", "  ")
		if err != nil {
			logger.Error("Failed to marshal push notification configuration: %v", err)
			return
		}
		fmt.Println(string(jsonData))
	case "pretty":
		// Print in a human-readable format
		fmt.Printf("Task ID: %s\n", config.TaskID)
		fmt.Printf("URL: %s\n", config.URL)
		if config.Authentication != nil {
			fmt.Printf("Authentication Type: %s\n", config.Authentication.Type)
		}
		if config.IncludeTaskData != nil {
			fmt.Printf("Include Task Data: %t\n", *config.IncludeTaskData)
		} else {
			fmt.Printf("Include Task Data: true (default)\n")
		}
		if config.IncludeArtifacts != nil {
			fmt.Printf("Include Artifacts: %t\n", *config.IncludeArtifacts)
		} else {
			fmt.Printf("Include Artifacts: false (default)\n")
		}
	default:
		logger.Error("Unknown output format: %s", format)
	}
}

// printAgentCard prints an agent card.
func printAgentCard(card *a2a.AgentCard, format string, logger *common.Logger) {
	switch format {
	case "json":
		// Print as JSON
		jsonData, err := json.MarshalIndent(card, "", "  ")
		if err != nil {
			logger.Error("Failed to marshal agent card: %v", err)
			return
		}
		fmt.Println(string(jsonData))
	case "pretty":
		// Print in a human-readable format
		fmt.Printf("Agent ID: %s\n", card.ID)
		fmt.Printf("Name: %s\n", card.Name)
		if card.Description != nil {
			fmt.Printf("Description: %s\n", *card.Description)
		}
		if card.Provider != nil {
			fmt.Printf("Provider: %s\n", card.Provider.Name)
			if card.Provider.URI != nil {
				fmt.Printf("Provider URI: %s\n", *card.Provider.URI)
			}
		}
		fmt.Printf("Skills: %d\n", len(card.Skills))
		for i, skill := range card.Skills {
			fmt.Printf("  [%d] %s (%s)\n", i, skill.Name, skill.ID)
			if skill.Description != nil {
				fmt.Printf("      %s\n", *skill.Description)
			}
		}
		if card.Capabilities != nil {
			fmt.Printf("Capabilities:\n")
			fmt.Printf("  Supports Streaming: %t\n", card.Capabilities.SupportsStreaming)
			fmt.Printf("  Supports Sessions: %t\n", card.Capabilities.SupportsSessions)
			fmt.Printf("  Supports Push Notification: %t\n", card.Capabilities.SupportsPushNotification)
		}
		fmt.Printf("Authentication Methods: %d\n", len(card.Authentication))
		for i, auth := range card.Authentication {
			fmt.Printf("  [%d] %s\n", i, auth.Type)
			if auth.Scheme != nil {
				fmt.Printf("      Scheme: %s\n", *auth.Scheme)
			}
		}
	default:
		logger.Error("Unknown output format: %s", format)
	}
}

// getMessageText returns the text content of a message.
func getMessageText(msg *a2a.Message) string {
	for _, part := range msg.Parts {
		if textPart, ok := part.(a2a.TextPart); ok {
			return textPart.Text
		}
	}
	return "[No text content]"
}

// getPartDescription returns a description of a part.
func getPartDescription(part a2a.Part) string {
	switch p := part.(type) {
	case a2a.TextPart:
		return fmt.Sprintf("Text: %s", p.Text)
	case a2a.FilePart:
		return fmt.Sprintf("File: %s (%s)", p.Filename, p.MimeType)
	case a2a.DataPart:
		return fmt.Sprintf("Data: %s", p.MimeType)
	default:
		return fmt.Sprintf("Unknown part type: %T", part)
	}
}

// printUsage prints usage information.
func printUsage() {
	fmt.Println("Usage: a2a-client [options] <command> [command options]")
	fmt.Println("Options:")
	flag.PrintDefaults()
	fmt.Println("\nCommands:")
	fmt.Println("  send        Send a task to an agent")
	fmt.Println("  get         Get a task from an agent")
	fmt.Println("  cancel      Cancel a task")
	fmt.Println("  subscribe   Subscribe to task updates")
	fmt.Println("  push        Configure push notifications")
	fmt.Println("  card        Get agent card information")
}

// saveDefaultConfig saves a default configuration file.
func saveDefaultConfig(path string) error {
	config := common.DefaultClientConfig()
	return common.SaveConfig(config, path)
}
