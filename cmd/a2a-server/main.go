package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/sammcj/go-a2a/cmd/common"
	"github.com/sammcj/go-a2a/pkg/config"
	"github.com/sammcj/go-a2a/server"
)

var (
	configFile    = flag.String("config", "", "Path to configuration file (JSON or YAML)")
	listenAddress = flag.String("listen", ":8080", "Address to listen on")
	agentCardFile = flag.String("agent-card", "", "Path to agent card file (JSON or YAML)")
	agentCardPath = flag.String("agent-card-path", "/.well-known/agent.json", "Path to serve agent card at")
	a2aPathPrefix = flag.String("a2a-path-prefix", "/a2a", "Path prefix for A2A endpoints")
	logLevel      = flag.String("log-level", "info", "Log level (debug, info, warn, error, fatal)")
	pluginPath    = flag.String("plugin-path", "", "Path to plugin directory")
)

func main() {
	// Parse command line flags
	flag.Parse()

	// Create logger
	logger := common.NewLogger(os.Stdout, *logLevel)
	logger.Info("Starting A2A server")

	// Load configuration
	var config config.ServerConfig
	if *configFile != "" {
		logger.Info("Loading configuration from %s", *configFile)
		loadedConfig, err := common.LoadConfig[common.ServerConfig](*configFile)
		if err != nil {
			logger.Fatal("Failed to load configuration: %v", err)
		}
		config = *loadedConfig
	} else {
		// If no config file is specified, use default values
		config = common.DefaultServerConfig()
		if *listenAddress != "" {
			config.ListenAddress = *listenAddress
		}
		if *agentCardPath != "" {
			config.AgentCardPath = *agentCardPath
		}
		if *a2aPathPrefix != "" {
			config.A2APathPrefix = *a2aPathPrefix
		}
		if *logLevel != "" {
			config.LogLevel = *logLevel
		}
		if *pluginPath != "" {
			config.PluginPath = *pluginPath
		}
	}

	// Load LLM config
	var llmConfig config.LLMConfig
	if config.LLMConfig.Provider != "" {
		llmConfig = config.LLMConfig
	} else {
		llmConfig = config.DefaultLLMConfig()
	}

	gollmOpts, err := server.NewGollmOptionsFromConfig(llmConfig)
	if err != nil {
		logger.Fatal("Failed to create gollm options: %v", err)
	}

	var agentCard *config.AgentCardConfig
	if *agentCardFile != "" {
		logger.Info("Loading agent card from %s", *agentCardFile)
		loadedCard, err := common.LoadConfig[common.AgentCardConfig](*agentCardFile)
		if err != nil {
			logger.Fatal("Failed to load agent card: %v", err)
		}
		agentCard = loadedCard
	} else {
		// If no agent card file is specified, use the agent card from the server configuration
		agentCard = &config.AgentCard
	}

	// Convert agent card config to A2A format
	a2aAgentCard := common.ConvertToAgentCard(*agentCard)

	// Load plugins
	var taskHandler server.TaskHandler
	if config.PluginPath != "" {
		logger.Info("Loading plugins from %s", config.PluginPath)
		plugins, err := common.LoadPlugins(config.PluginPath)
		if err != nil {
			logger.Fatal("Failed to load plugins: %v", err)
		}

		if len(plugins) == 0 {
			logger.Warn("No plugins found, using built-in echo plugin")
			taskHandler = common.NewEchoPlugin().GetTaskHandler()
		} else {
			logger.Info("Loaded %d plugins", len(plugins))
			taskHandler = common.MergeTaskHandlers(plugins)
		}
	} else {
		// Use built-in echo plugin
		logger.Info("No plugin path specified, using built-in echo plugin")
		taskHandler = common.NewEchoPlugin().GetTaskHandler()
	}

	// Create server options
	opts := []server.Option{
		server.WithListenAddress(config.ListenAddress),
		server.WithAgentCard(a2aAgentCard),
		server.WithAgentCardPath(config.AgentCardPath),
		server.WithA2APathPrefix(config.A2APathPrefix),
		server.WithTaskHandler(taskHandler),
		server.WithGollmOptions(gollmOpts),
	}

	// Create server
	srv, err := server.NewServer(opts...)
	if err != nil {
		logger.Fatal("Failed to create server: %v", err)
	}

	// Start server in a goroutine
	go func() {
		logger.Info("Server listening on %s", config.ListenAddress)
		if err := srv.Start(); err != nil {
			logger.Fatal("Server error: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shut down the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.Info("Shutting down server...")

	// Create a deadline to wait for
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Doesn't block if no connections, but will otherwise wait until the timeout deadline
	if err := srv.Stop(ctx); err != nil {
		logger.Fatal("Server forced to shutdown: %v", err)
	}

	logger.Info("Server exited properly")
}

// saveDefaultConfig saves a default configuration file.
func saveDefaultConfig(path string) error {
	config := config.DefaultServerConfig()
	return common.SaveConfig(config, path)
}

// saveDefaultAgentCard saves a default agent card file.
func saveDefaultAgentCard(path string) error {
	config := config.DefaultServerConfig().AgentCard
	return common.SaveConfig(config, path)
}

// ensureDirectory ensures that the directory exists.
func ensureDirectory(path string) error {
	dir := filepath.Dir(path)
	return os.MkdirAll(dir, 0755)
}

// printUsage prints usage information.
func printUsage() {
	fmt.Println("Usage: a2a-server [options]")
	fmt.Println("Options:")
	flag.PrintDefaults()
}
