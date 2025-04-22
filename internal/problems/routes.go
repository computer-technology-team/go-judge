package problems

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/computer-technology-team/go-judge/internal/middleware"
	"github.com/computer-technology-team/go-judge/internal/storage"
	"github.com/computer-technology-team/go-judge/web/templates"
)

// Handler defines the interface for problem handlers
type Handler interface {
	ListProblems(w http.ResponseWriter, r *http.Request)
	CreateProblem(w http.ResponseWriter, r *http.Request)
	ViewProblem(w http.ResponseWriter, r *http.Request)
	UpdateProblem(w http.ResponseWriter, r *http.Request)
	DeleteProblem(w http.ResponseWriter, r *http.Request)
	ShowCreateProblem(w http.ResponseWriter, r *http.Request)
}

// NewRoutes returns a function that registers routes with the given handler
// This allows for dependency injection when setting up routes
func NewRoutes(h Handler, sharedTmpls *templates.Templates) func(r chi.Router) {
	return func(r chi.Router) {
		r.Get("/", h.ListProblems)
		r.Group(func(r chi.Router) {
			r.Use(middleware.NewRequireAuthMiddleware(sharedTmpls))
			r.Post("/", h.CreateProblem)
			r.Get("/create", h.ShowCreateProblem)
		})
		r.Get("/{id}", h.ViewProblem)
		r.Put("/{id}", h.UpdateProblem)
		r.Delete("/{id}", h.DeleteProblem)
	}
}

// DefaultHandler is the default implementation of the Handler interface
type DefaultHandler struct {
	templates *templates.Templates
	pool      *pgxpool.Pool
	querier   storage.Querier
}

// NewHandler creates a new instance of the default problem handler
func NewHandler(templates *templates.Templates, pool *pgxpool.Pool, querier storage.Querier) Handler {
	return &DefaultHandler{templates: templates, pool: pool, querier: querier}
}

// UpdateProblem updates a specific problem
func (h *DefaultHandler) UpdateProblem(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement update problem logic
	id := chi.URLParam(r, "id")
	w.Write([]byte("Update problem endpoint: " + id))
}

// DeleteProblem deletes a specific problem
func (h *DefaultHandler) DeleteProblem(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement delete problem logic
	id := chi.URLParam(r, "id")
	w.Write([]byte("Delete problem endpoint: " + id))
}
