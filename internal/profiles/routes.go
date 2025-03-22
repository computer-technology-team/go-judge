package profiles

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

// Handler defines the interface for profile handlers
type Handler interface {
	GetProfile(w http.ResponseWriter, r *http.Request)
	UpdateProfile(w http.ResponseWriter, r *http.Request)
	GetUserSubmissions(w http.ResponseWriter, r *http.Request)
	GetUserStats(w http.ResponseWriter, r *http.Request)
}

// NewRoutes returns a function that registers routes with the given handler
// This allows for dependency injection when setting up routes
func NewRoutes(h Handler) func(r chi.Router) {
	return func(r chi.Router) {
		r.Get("/{username}", h.GetProfile)
		r.Put("/", h.UpdateProfile)
		r.Get("/{username}/submissions", h.GetUserSubmissions)
		r.Get("/{username}/stats", h.GetUserStats)
	}
}

// DefaultHandler is the default implementation of the Handler interface
type DefaultHandler struct {
	// Dependencies can be injected here (e.g., service, repository)
}

// NewHandler creates a new instance of the default profile handler
func NewHandler() Handler {
	return &DefaultHandler{}
}

// GetProfile returns a user's profile
func (h *DefaultHandler) GetProfile(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement get profile logic
	username := chi.URLParam(r, "username")
	w.Write([]byte("Get profile endpoint: " + username))
}

// UpdateProfile updates the current user's profile
func (h *DefaultHandler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement update profile logic
	w.Write([]byte("Update profile endpoint"))
}

// GetUserSubmissions returns a user's submissions
func (h *DefaultHandler) GetUserSubmissions(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement get user submissions logic
	username := chi.URLParam(r, "username")
	w.Write([]byte("Get user submissions endpoint: " + username))
}

// GetUserStats returns a user's statistics
func (h *DefaultHandler) GetUserStats(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement get user stats logic
	username := chi.URLParam(r, "username")
	w.Write([]byte("Get user stats endpoint: " + username))
}
