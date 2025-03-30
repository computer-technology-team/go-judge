package profiles

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

// Servicer defines the interface for profile handlers
type Servicer interface {
	GetProfile(w http.ResponseWriter, r *http.Request)
	ToggleSuperUser(w http.ResponseWriter, r *http.Request)
}

// NewRoutes returns a function that registers routes with the given handler
// This allows for dependency injection when setting up routes
func NewRoutes(h Servicer) func(r chi.Router) {
	return func(r chi.Router) {
		r.Get("/{username}", h.GetProfile)
		r.Post("/{username}/toggle-superuser", h.ToggleSuperUser)
	}
}
