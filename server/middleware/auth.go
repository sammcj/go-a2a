package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/sammcj/go-a2a/a2a"
)

// AuthKey is the context key for storing authentication information.
type AuthKey struct{}

// AuthInfo represents authentication information extracted from a request.
type AuthInfo struct {
	Type   string      // The type of authentication (e.g., "bearer", "oauth2")
	Scheme string      // The authentication scheme
	Value  string      // The authentication value (e.g., token)
	Data   interface{} // Additional authentication data
}

// AuthValidator is a function that validates authentication information.
type AuthValidator func(ctx context.Context, info AuthInfo) (bool, error)

// AuthMiddleware creates middleware that authenticates requests based on the agent card's authentication schemes.
func AuthMiddleware(card *a2a.AgentCard, validator AuthValidator) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip authentication for agent card requests
			if strings.HasSuffix(r.URL.Path, "/.well-known/agent.json") {
				next.ServeHTTP(w, r)
				return
			}

			// Skip authentication if no authentication schemes are defined
			if card.Authentication == nil || len(card.Authentication) == 0 {
				next.ServeHTTP(w, r)
				return
			}

			// Extract authentication information from the request
			authInfo, err := extractAuthInfo(r, card.Authentication)
			if err != nil {
				// Convert error to A2A error if needed
				var a2aErr *a2a.Error
				if e, ok := err.(*a2a.Error); ok {
					a2aErr = e
				} else {
					a2aErr = a2a.WrapError(err, a2a.CodeAuthenticationFailed, "Authentication extraction failed")
				}
				writeAuthError(w, r, a2aErr)
				return
			}

			// If no authentication information was found, require authentication
			if authInfo == nil {
				writeAuthError(w, r, a2a.ErrAuthenticationRequired())
				return
			}

			// Validate the authentication information
			if validator != nil {
				valid, err := validator(r.Context(), *authInfo)
				if err != nil {
					writeAuthError(w, r, a2a.WrapError(err, a2a.CodeAuthenticationFailed, "Authentication validation failed"))
					return
				}
				if !valid {
					writeAuthError(w, r, a2a.ErrAuthenticationFailed("Invalid credentials"))
					return
				}
			}

			// Add authentication information to the request context
			ctx := context.WithValue(r.Context(), AuthKey{}, authInfo)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// extractAuthInfo extracts authentication information from the request based on the agent card's authentication schemes.
func extractAuthInfo(r *http.Request, schemes []a2a.AgentAuthentication) (*AuthInfo, error) {
	// Try each authentication scheme in order
	for _, scheme := range schemes {
		switch scheme.Type {
		case "bearer":
			// Extract bearer token from Authorization header
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				continue
			}
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
				continue
			}
			return &AuthInfo{
				Type:   "bearer",
				Scheme: "Bearer",
				Value:  parts[1],
			}, nil
		case "header":
			// Extract value from custom header
			if scheme.Configuration == nil {
				continue
			}
			config, ok := scheme.Configuration.(map[string]interface{})
			if !ok {
				continue
			}
			headerName, ok := config["headerName"].(string)
			if !ok {
				continue
			}
			headerValue := r.Header.Get(headerName)
			if headerValue == "" {
				continue
			}
			return &AuthInfo{
				Type:   "header",
				Scheme: headerName,
				Value:  headerValue,
				Data:   config,
			}, nil
		case "oauth2":
			// Extract OAuth2 token from Authorization header
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				continue
			}
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
				continue
			}
			return &AuthInfo{
				Type:   "oauth2",
				Scheme: "OAuth2",
				Value:  parts[1],
				Data:   scheme.Configuration,
			}, nil
		// Add more authentication types as needed
		default:
			// Skip unknown authentication types
			continue
		}
	}

	// No authentication information found
	return nil, nil
}

// writeAuthError writes an authentication error response.
func writeAuthError(w http.ResponseWriter, r *http.Request, err *a2a.Error) {
	// Set WWW-Authenticate header for 401 responses
	if err.Code == a2a.CodeAuthenticationRequired {
		w.Header().Set("WWW-Authenticate", "Bearer")
	}

	// Convert error to JSON-RPC error
	response := a2a.JSONRPCResponse{
		JSONRPC: "2.0",
		Error:   err.ToJSONRPCError(),
		ID:      nil, // We don't have a request ID here
	}

	// Marshal response
	jsonResp, marshalErr := json.Marshal(response)
	if marshalErr != nil {
		// If marshalling fails, return a simple error
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Write response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	w.Write(jsonResp)
}
