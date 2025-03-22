package auth

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

// Handler defines the interface for auth handlers
type Handler interface {
	Login(w http.ResponseWriter, r *http.Request)
	Register(w http.ResponseWriter, r *http.Request)
	Logout(w http.ResponseWriter, r *http.Request)
	RefreshToken(w http.ResponseWriter, r *http.Request)
}

// NewRoutes returns a function that registers routes with the given handler
// This allows for dependency injection when setting up routes
func NewRoutes(h Handler) func(r chi.Router) {
	return func(r chi.Router) {
		r.Post("/login", h.Login)
		r.Post("/register", h.Register)
		r.Post("/logout", h.Logout)
		r.Post("/refresh", h.RefreshToken)
	}
}

// DefaultHandler is the default implementation of the Handler interface
type DefaultHandler struct {
	// Dependencies can be injected here (e.g., service, repository)
}

// NewHandler creates a new instance of the default auth handler
func NewHandler() Handler {
	return &DefaultHandler{}
}

// Login handles user login
func (h *DefaultHandler) Login(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement login logic
	w.Write([]byte("Login endpoint"))
}

// Register handles user registration
func (h *DefaultHandler) Register(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement registration logic
	w.Write([]byte("Register endpoint"))
}

// Logout handles user logout
func (h *DefaultHandler) Logout(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement logout logic
	w.Write([]byte("Logout endpoint"))
}

// RefreshToken handles token refresh
func (h *DefaultHandler) RefreshToken(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement token refresh logic
	w.Write([]byte("Refresh token endpoint"))
}
