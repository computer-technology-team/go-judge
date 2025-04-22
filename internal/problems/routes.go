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
	ListMyProblems(w http.ResponseWriter, r *http.Request)
	ListProblems(w http.ResponseWriter, r *http.Request)
	CreateProblem(w http.ResponseWriter, r *http.Request)
	ViewProblem(w http.ResponseWriter, r *http.Request)
	UpdateProblem(w http.ResponseWriter, r *http.Request)
	ProblemForm(w http.ResponseWriter, r *http.Request)

	ToggleStatus(w http.ResponseWriter, r *http.Request)
}

// NewRoutes returns a function that registers routes with the given handler
// This allows for dependency injection when setting up routes
func NewRoutes(h Handler, sharedTmpls *templates.Templates) func(r chi.Router) {
	return func(r chi.Router) {
		r.Get("/", h.ListProblems)
		r.Group(func(r chi.Router) {
			r.Use(middleware.NewRequireAuthMiddleware(sharedTmpls))
			r.Post("/", h.CreateProblem)
			r.Get("/form/{id}", h.ProblemForm)
			r.Post("/{id}", h.UpdateProblem)
			r.Post("/{id}/toggle-status", h.ToggleStatus)
			r.Get("/my", h.ListMyProblems)
		})
		r.Get("/{id}", h.ViewProblem)
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
