package middleware

import (
	"context"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
)

type contextKey string

const (
	// SessionCookieName is the name of the session cookie.
	SessionCookieName = "session_id"

	// UserIDKey is the context key for the user ID.
	UserIDKey = contextKey("user_id")

	// SessionIDKey is the context key for the session ID.
	SessionIDKey = contextKey("session_id")
)

// SessionValidator validates session tokens and returns user ID on success.
type SessionValidator interface {
	ValidateSession(ctx context.Context, sessionID string) (userID string, err error)
}

// OptionalSession extracts user context from session cookie if present.
// Does NOT require authentication - continues even without valid token.
// Use this globally to make user info available to all routes.
func OptionalSession(validator SessionValidator) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie(SessionCookieName)
			if err == nil && cookie.Value != "" {
				userID, err := validator.ValidateSession(r.Context(), cookie.Value)
				if err == nil {
					ctx := r.Context()
					ctx = context.WithValue(ctx, UserIDKey, userID)
					ctx = context.WithValue(ctx, SessionIDKey, cookie.Value)
					r = r.WithContext(ctx)
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}

// Session validates session cookies and injects user context.
// If the session is invalid, it clears the cookie and redirects to /login.
// Use this for routes that REQUIRE authentication.
func Session(validator SessionValidator) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie(SessionCookieName)
			if err != nil {
				http.Redirect(w, r, "/login", http.StatusSeeOther)
				return
			}

			userID, err := validator.ValidateSession(r.Context(), cookie.Value)
			if err != nil {
				ClearSessionCookie(w)
				http.Redirect(w, r, "/login", http.StatusSeeOther)
				return
			}

			ctx := r.Context()
			ctx = context.WithValue(ctx, UserIDKey, userID)
			ctx = context.WithValue(ctx, SessionIDKey, cookie.Value)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// ClearSessionCookie clears the session cookie by setting MaxAge to -1.
func ClearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}

// SetSessionCookie sets the session cookie with the given value and TTL.
func SetSessionCookie(w http.ResponseWriter, value string, maxAge int) {
	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookieName,
		Value:    value,
		Path:     "/",
		MaxAge:   maxAge,
		HttpOnly: true,
		Secure:   false, // Desktop app, no HTTPS required
		SameSite: http.SameSiteLaxMode,
	})
}

// GetUserID extracts the user ID from the context.
// Returns an empty string if no user ID is found.
func GetUserID(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if id, ok := ctx.Value(UserIDKey).(string); ok {
		return id
	}
	return ""
}

// GetSessionID extracts the session ID from the context.
// Returns an empty string if no session ID is found.
func GetSessionID(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if id, ok := ctx.Value(SessionIDKey).(string); ok {
		return id
	}
	return ""
}

// LocalhostOnly rejects requests that don't originate from localhost.
// This protects the application from network or internet access.
func LocalhostOnly(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !isLocalhost(r) {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func isLocalhost(r *http.Request) bool {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}

	if host == "127.0.0.1" || host == "::1" || host == "localhost" {
		return true
	}

	// Check X-Forwarded-For for proxied requests (only trust if behind localhost proxy)
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		ips := strings.Split(xff, ",")
		if len(ips) > 0 {
			firstIP := strings.TrimSpace(ips[0])
			if firstIP == "127.0.0.1" || firstIP == "::1" {
				return true
			}
		}
	}

	ip := net.ParseIP(host)
	if ip == nil {
		return false
	}

	return ip.IsLoopback()
}

// DefaultStack applies the default middleware stack to a router.
func DefaultStack(r chi.Router) {
	r.Use(LocalhostOnly)
	r.Use(chimw.RequestID)
	r.Use(chimw.RealIP)
	r.Use(chimw.Recoverer)
	r.Use(chimw.Timeout(60 * time.Second))
}
