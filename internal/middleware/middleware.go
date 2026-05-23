package middleware

import (
	"crypto/subtle"
	"encoding/base64"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

func Logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip logging for high-frequency tracking endpoints to reduce overhead
		if strings.HasPrefix(r.URL.Path, "/api/track") {
			next.ServeHTTP(w, r)
			return
		}
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %s", r.Method, r.RequestURI, time.Since(start))
	})
}

func CORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cors := os.Getenv("CORS")
		if cors == "" {
			cors = "*"
		}
		w.Header().Set("Access-Control-Allow-Origin", cors)
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// BasicAuth middleware for protecting routes with basic authentication
// Credentials are read from environment variables: DASHBOARD_USERNAME and DASHBOARD_PASSWORD
func BasicAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get credentials from environment
		username := os.Getenv("DASHBOARD_USERNAME")
		password := os.Getenv("DASHBOARD_PASSWORD")

		// If credentials are not set, allow access (authentication disabled)
		if username == "" || password == "" {
			next.ServeHTTP(w, r)
			return
		}

		// Get Authorization header
		auth := r.Header.Get("Authorization")
		if auth == "" {
			requireAuth(w)
			return
		}

		// Parse Basic Auth header
		const prefix = "Basic "
		if !strings.HasPrefix(auth, prefix) {
			requireAuth(w)
			return
		}

		// Decode base64 credentials
		decoded, err := base64.StdEncoding.DecodeString(auth[len(prefix):])
		if err != nil {
			requireAuth(w)
			return
		}

		// Split username:password
		credentials := strings.SplitN(string(decoded), ":", 2)
		if len(credentials) != 2 {
			requireAuth(w)
			return
		}

		// Use constant-time comparison to prevent timing attacks
		usernameMatch := subtle.ConstantTimeCompare([]byte(credentials[0]), []byte(username)) == 1
		passwordMatch := subtle.ConstantTimeCompare([]byte(credentials[1]), []byte(password)) == 1

		if !usernameMatch || !passwordMatch {
			requireAuth(w)
			return
		}

		// Authentication successful
		next.ServeHTTP(w, r)
	})
}

// requireAuth sends a 401 Unauthorized response with WWW-Authenticate header
func requireAuth(w http.ResponseWriter) {
	w.Header().Set("WWW-Authenticate", `Basic realm="Siraaj Dashboard"`)
	w.WriteHeader(http.StatusUnauthorized)
	if _, err := w.Write([]byte("401 Unauthorized - Authentication required\n")); err != nil {
		log.Printf("Error writing auth response: %v", err)
	}
}
