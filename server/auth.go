package server

import (
	"context"
	"net/http"

	"github.com/sammcj/go-a2a/a2a"
	"github.com/sammcj/go-a2a/server/middleware"
)

// CreateAuthValidator creates an AuthValidator function that uses the middleware.AuthMiddleware
// with the provided validator function.
func CreateAuthValidator(validator middleware.AuthValidator) AuthValidator {
	return func(w http.ResponseWriter, r *http.Request, next http.Handler, card *a2a.AgentCard) {
		// Create the middleware
		authMiddleware := middleware.AuthMiddleware(card, validator)

		// Apply the middleware to the next handler
		handler := authMiddleware(next)

		// Call the handler
		handler.ServeHTTP(w, r)
	}
}

// SimpleTokenValidator creates an AuthValidator that validates requests using a simple token comparison.
// This is useful for basic authentication scenarios where a single token is used.
func SimpleTokenValidator(expectedToken string) AuthValidator {
	validator := func(ctx context.Context, info middleware.AuthInfo) (bool, error) {
		// For bearer token authentication
		if info.Type == "bearer" {
			return info.Value == expectedToken, nil
		}

		// For header-based authentication
		if info.Type == "header" {
			return info.Value == expectedToken, nil
		}

		// Unsupported authentication type
		return false, nil
	}

	return CreateAuthValidator(validator)
}

// NoAuthValidator is an AuthValidator that allows all requests.
// This is useful for development or when authentication is not required.
func NoAuthValidator() AuthValidator {
	return func(w http.ResponseWriter, r *http.Request, next http.Handler, card *a2a.AgentCard) {
		next.ServeHTTP(w, r)
	}
}
