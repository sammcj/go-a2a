package common

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"plugin"

	"github.com/sammcj/go-a2a/a2a"
)

// TaskHandlerPlugin represents a plugin that provides a task handler.
type TaskHandlerPlugin interface {
	GetTaskHandler() func(ctx context.Context, taskCtx TaskContext) (<-chan TaskYieldUpdate, error)

	GetSkills() []a2a.AgentSkill 
}

// LoadPlugin loads a plugin from the specified path.
func LoadPlugin(path string) (TaskHandlerPlugin, error) {
	// Load the plugin
	p, err := plugin.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to load plugin: %w", err)
	}

	// Look up the TaskHandlerPlugin symbol
	sym, err := p.Lookup("Plugin")
	if err != nil {
		return nil, fmt.Errorf("plugin does not export 'Plugin' symbol: %w", err)
	}

	// Assert that the symbol is a TaskHandlerPlugin
	plugin, ok := sym.(TaskHandlerPlugin)
	if !ok {
		return nil, fmt.Errorf("plugin does not implement TaskHandlerPlugin interface")
	}

	return plugin, nil
}

// LoadPlugins loads all plugins from the specified directory.
func LoadPlugins(dir string) ([]TaskHandlerPlugin, error) {
	// Find all .so files in the directory
	matches, err := filepath.Glob(filepath.Join(dir, "*.so"))
	if err != nil {
		return nil, fmt.Errorf("failed to find plugins: %w", err)
	}

	// Load each plugin
	plugins := make([]TaskHandlerPlugin, 0, len(matches))
	for _, match := range matches {
		plugin, err := LoadPlugin(match)
		if err != nil {
			return nil, fmt.Errorf("failed to load plugin %s: %w", match, err)
		}
		plugins = append(plugins, plugin)
	}

	return plugins, nil
}

// MergeTaskHandlers merges multiple task handlers into a single task handler.
// The resulting task handler will delegate to the appropriate plugin based on the skill ID.
func MergeTaskHandlers(plugins []TaskHandlerPlugin) func(ctx context.Context, taskCtx TaskContext) (<-chan TaskYieldUpdate, error) {
	// Create a map of skill ID to task handler
	handlers := make(map[string]func(ctx context.Context, taskCtx TaskContext) (<-chan TaskYieldUpdate, error))
	for _, p := range plugins{
		handler := p.GetTaskHandler()
		for _, skill := range p.GetSkills() {
			handlers[skill.ID] = handler
		}
	}

	// Return a task handler that delegates to the appropriate plugin
	return func(ctx context.Context, taskCtx TaskContext) (<-chan TaskYieldUpdate, error) {
		// Get the skill ID from the message or use a default
		skillID := ""
		// Try to extract skill ID from the message metadata if available
		if taskCtx.UserMessage.Metadata != nil {
			if metadata, ok := taskCtx.UserMessage.Metadata.(map[string]interface{}); ok {
				if skillIDValue, ok := metadata["skillId"]; ok {
					if skillIDStr, ok := skillIDValue.(string); ok {
						skillID = skillIDStr
					}
				}
			}
		}

		// Find the handler for the skill
		handler, ok := handlers[skillID]
		if !ok {
			// If no handler is found, use the first handler as a fallback
			for _, h := range handlers {
				handler = h
				break
			}
		}

		// If no handler is found, return an error
		if handler == nil {
			return nil, errors.New("no handler found")
		}

		// Delegate to the handler
		return handler(ctx, taskCtx)
	}
}
