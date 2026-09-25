package middleware

import (
	"context"
	"encoding/json"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/mohamedelhefni/siraaj/internal/domain"
)

type contextKey string

const (
	principalContextKey contextKey = "auth-principal"
	trackingContextKey  contextKey = "tracking-identity"
)

type AccessTokenAuthenticator interface {
	AuthenticateAccessToken(token string) (domain.Principal, error)
}

type TrackingTokenAuthenticator interface {
	AuthenticateTrackingToken(token string) (domain.TrackingIdentity, error)
}

type rateLimitEntry struct {
	count   int
	resetAt time.Time
}

type RateLimiter struct {
	mu          sync.Mutex
	attempts    map[string]rateLimitEntry
	limit       int
	window      time.Duration
	lastCleanup time.Time
}

func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	return &RateLimiter{attempts: make(map[string]rateLimitEntry), limit: limit, window: window, lastCleanup: time.Now()}
}

func (l *RateLimiter) Limit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		allowed, retryAfter := l.allow(clientAddress(r), time.Now())
		if !allowed {
			w.Header().Set("Retry-After", strconv.Itoa(int(retryAfter.Seconds())+1))
			http.Error(w, "Too many attempts", http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (l *RateLimiter) allow(client string, now time.Time) (bool, time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if now.Sub(l.lastCleanup) >= l.window {
		for address, attempt := range l.attempts {
			if !now.Before(attempt.resetAt) {
				delete(l.attempts, address)
			}
		}
		l.lastCleanup = now
	}
	attempt := l.attempts[client]
	if attempt.resetAt.IsZero() || !now.Before(attempt.resetAt) {
		attempt = rateLimitEntry{resetAt: now.Add(l.window)}
	}
	if attempt.count >= l.limit {
		return false, attempt.resetAt.Sub(now)
	}
	attempt.count++
	l.attempts[client] = attempt
	return true, 0
}

func clientAddress(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		return host
	}
	return r.RemoteAddr
}

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
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Siraaj-Token")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		if strings.HasPrefix(r.URL.Path, "/api/auth/") {
			w.Header().Set("Cache-Control", "no-store")
		}
		next.ServeHTTP(w, r)
	})
}

func AccessAuth(authenticator AccessTokenAuthenticator, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := bearerToken(r)
		if token == "" {
			unauthorizedJSON(w)
			return
		}
		principal, err := authenticator.AuthenticateAccessToken(token)
		if err != nil {
			unauthorizedJSON(w)
			return
		}
		ctx := context.WithValue(r.Context(), principalContextKey, principal)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func TrackingAuth(authenticator TrackingTokenAuthenticator, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := strings.TrimSpace(r.Header.Get("X-Siraaj-Token"))
		if token == "" {
			token = bearerToken(r)
		}
		identity, err := authenticator.AuthenticateTrackingToken(token)
		if err != nil {
			unauthorizedJSON(w)
			return
		}
		ctx := context.WithValue(r.Context(), trackingContextKey, identity)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func PrincipalFromContext(ctx context.Context) domain.Principal {
	principal, _ := ctx.Value(principalContextKey).(domain.Principal)
	return principal
}

func TrackingIdentityFromContext(ctx context.Context) domain.TrackingIdentity {
	identity, _ := ctx.Value(trackingContextKey).(domain.TrackingIdentity)
	return identity
}

func bearerToken(r *http.Request) string {
	auth := strings.TrimSpace(r.Header.Get("Authorization"))
	const prefix = "Bearer "
	if len(auth) <= len(prefix) || !strings.EqualFold(auth[:len(prefix)], prefix) {
		return ""
	}
	return strings.TrimSpace(auth[len(prefix):])
}

func unauthorizedJSON(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("WWW-Authenticate", "Bearer")
	w.WriteHeader(http.StatusUnauthorized)
	if err := json.NewEncoder(w).Encode(map[string]string{"error": "authentication required"}); err != nil {
		log.Printf("Error encoding authentication response: %v", err)
	}
}
