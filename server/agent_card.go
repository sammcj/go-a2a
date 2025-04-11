package server

import (
	"encoding/json"
	"net/http"

	"github.com/sammcj/go-a2a/a2a"
)

// DefaultAgentCardPath is the default path for serving the agent card.
const DefaultAgentCardPath = "/.well-known/agent.json"

// AgentCardHandler returns an HTTP handler that serves the agent card.
func AgentCardHandler(card *a2a.AgentCard) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*") // Allow CORS for discovery

		// Marshal the agent card to JSON
		jsonData, err := json.MarshalIndent(card, "", "  ")
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		w.Write(jsonData)
	}
}

// RegisterAgentCardHandler registers the agent card handler with the provided ServeMux.
func RegisterAgentCardHandler(mux *http.ServeMux, card *a2a.AgentCard, cardPath string) {
	if cardPath == "" {
		cardPath = DefaultAgentCardPath
	}

	// Ensure the path starts with a slash
	if cardPath[0] != '/' {
		cardPath = "/" + cardPath
	}

	mux.HandleFunc(cardPath, AgentCardHandler(card))
}

// WithAgentCardPath returns an Option that sets the path for serving the agent card.
func WithAgentCardPath(cardPath string) Option {
	return func(c *Config) {
		c.AgentCardPath = cardPath
	}
}
