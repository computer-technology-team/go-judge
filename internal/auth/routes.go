package auth

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

// NewRoutes returns a function that registers routes with the given handler
// This allows for dependency injection when setting up routes
func NewRoutes(s Servicer) func(r chi.Router) {
	return func(r chi.Router) {
		r.Post("/login", s.Login)
		r.Post("/register", s.Register)
		r.Post("/logout", s.Logout)
		r.Post("/refresh", s.RefreshToken)
	}
}

// Login handles user login
func (h *DefaultServicer) Login(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement login logic
	w.Write([]byte("Login endpoint"))
}

// Register handles user registration
func (h *DefaultServicer) Register(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement registration logic
	w.Write([]byte("Register endpoint"))
}

// Logout handles user logout
func (h *DefaultServicer) Logout(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement logout logic
	w.Write([]byte("Logout endpoint"))
}

// RefreshToken handles token refresh
func (h *DefaultServicer) RefreshToken(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement token refresh logic
	w.Write([]byte("Refresh token endpoint"))
}
