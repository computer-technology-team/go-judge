package auth

import (
	"github.com/go-chi/chi/v5"

	"github.com/computer-technology-team/go-judge/internal/middleware"
)

// NewRoutes returns a function that registers routes with the given handler
// This allows for dependency injection when setting up routes
func NewRoutes(s Servicer) func(r chi.Router) {
	return func(r chi.Router) {
		r.Get("/login", s.ShowLoginPage)
		r.Get("/signup", s.ShowSignupPage)
		r.With(middleware.RequireAuth).Get("/logout", s.Logout)
		r.Post("/login", s.Login)
		r.Post("/signup", s.Signup)
	}
}
