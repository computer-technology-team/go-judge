package home

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/computer-technology-team/go-judge/web/templates"
)

// Handler defines the interface for home handlers
type Handler interface {
	Home(w http.ResponseWriter, r *http.Request)
	Root(w http.ResponseWriter, r *http.Request)
}

// NewRoutes returns a function that registers routes with the given handler
// This allows for dependency injection when setting up routes
func NewRoutes(h Handler) func(r chi.Router) {
	return func(r chi.Router) {
		r.Get("/", h.Root)
		r.Get("/home", h.Home)
	}
}

// DefaultHandler is the default implementation of the Handler interface
type DefaultHandler struct {
	templates *templates.Templates
}

// NewHandler creates a new instance of the default home handler
func NewHandler(templates *templates.Templates) Handler {
	return &DefaultHandler{templates: templates}
}

// Home handles the home page
func (h *DefaultHandler) Home(w http.ResponseWriter, r *http.Request) {
	err := h.templates.Render(r.Context(), "homepage", w, nil)
	if err != nil {
		slog.Error("could not render home", "error", err)
		templates.RenderError(r.Context(), w, "could not render", http.StatusInternalServerError, h.templates)
		return
	}
	w.WriteHeader(http.StatusOK)
}

// Root redirects to the home page
func (h *DefaultHandler) Root(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/home", http.StatusSeeOther)
}
