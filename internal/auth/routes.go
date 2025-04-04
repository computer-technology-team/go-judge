package auth

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
)

// NewRoutes returns a function that registers routes with the given handler
// This allows for dependency injection when setting up routes
func NewRoutes(s Servicer) func(r chi.Router) {
	return func(r chi.Router) {
		r.Get("/login", s.Login)
		r.Get("/signup", s.Signup)
		//r.Post("/login", s.Login)
		//r.Post("/signup", s.Signup)
		//r.Post("/logout", s.Logout)
		//r.Post("/refresh", s.RefreshToken)
	}
}

// Login handles user login
func (s *DefaultServicer) Login(w http.ResponseWriter, r *http.Request) {

	err := s.templates.Render(r.Context(), "login", w, nil)
	if err != nil {
		slog.Error("could not render login", "error", err)
		http.Error(w, "could not render", http.StatusInternalServerError)
		return
	}

	//TODO: Implement sql login logic
	w.WriteHeader(http.StatusOK)
}

// Signup handles user registration
func (s *DefaultServicer) Signup(w http.ResponseWriter, r *http.Request) {
	err := s.templates.Render(r.Context(), "signup", w, nil)
	if err != nil {
		slog.Error("could not render signup", "error", err)
		http.Error(w, "could not render", http.StatusInternalServerError)
		return
	}

	//TODO: Implement sql Signup logic
	w.WriteHeader(http.StatusOK)
}

// Logout handles user logout
func (s *DefaultServicer) Logout(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement logout logic
	w.Write([]byte("Logout endpoint"))
}

// RefreshToken handles token refresh
func (s *DefaultServicer) RefreshToken(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement token refresh logic
	w.Write([]byte("Refresh token endpoint"))
}
