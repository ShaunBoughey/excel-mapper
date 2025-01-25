package auth

import (
	"net/http"
	"os"
	"strings"
)

var (
	// apiKeys stores the valid API keys
	apiKeys map[string]bool
)

// InitAPIKeys initializes the API keys from environment variables
func InitAPIKeys() {
	apiKeys = make(map[string]bool)

	// Get API keys from environment variable
	keys := os.Getenv("API_KEYS")
	if keys != "" {
		// Split multiple API keys by comma
		for _, key := range strings.Split(keys, ",") {
			apiKeys[strings.TrimSpace(key)] = true
		}
	}
}

// RequireAPIKey is a middleware that checks for a valid API key
func RequireAPIKey(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		key := r.Header.Get("X-API-Key")
		if key == "" {
			http.Error(w, "API key is missing", http.StatusUnauthorized)
			return
		}

		if !apiKeys[key] {
			http.Error(w, "Invalid API key", http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	}
}
