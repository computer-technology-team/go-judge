package middleware

import (
	"context"
	"net/http"
	"strings"
)

// Key for user context
type contextKey string

const UserContextKey = contextKey("user")

// Authenticate middleware checks for valid authentication
func Authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get token from Authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			// Check for token in cookie
			cookie, err := r.Cookie("auth_token")
			if err != nil {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
			authHeader = cookie.Value
		} else {
			// Remove Bearer prefix if present
			authHeader = strings.TrimPrefix(authHeader, "Bearer ")
		}

		// TODO: Validate token and get user

		// For now, just pass a dummy user ID to the context
		ctx := context.WithValue(r.Context(), UserContextKey, "user-123")
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequireAuth middleware ensures the user is authenticated
func RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Context().Value(UserContextKey) == nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}
