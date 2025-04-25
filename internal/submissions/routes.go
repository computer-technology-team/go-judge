package submissions

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/computer-technology-team/go-judge/internal/storage"
	"github.com/computer-technology-team/go-judge/web/templates"
)

// Servicer defines the interface for submission handlers
type Servicer interface {
	ListSubmissions(w http.ResponseWriter, r *http.Request)
	SubmissionForm(w http.ResponseWriter, r *http.Request)
	CreateSubmission(w http.ResponseWriter, r *http.Request)
	GetSubmission(w http.ResponseWriter, r *http.Request)
}

// NewRoutes returns a function that registers routes with the given handler
// This allows for dependency injection when setting up routes
func NewRoutes(s Servicer) func(r chi.Router) {
	return func(r chi.Router) {
		r.Get("/", s.ListSubmissions)
		r.Get("/problem/{problem_id}/new", s.SubmissionForm)
		r.Post("/", s.CreateSubmission)
		r.Get("/{id}", s.GetSubmission)
	}
}

// ServicerImpl is the default implementation of the Handler interface
type ServicerImpl struct {
	broker    Broker
	querier   storage.Querier
	pool      *pgxpool.Pool
	templates *templates.Templates
}

// NewServicer creates a new instance of the default submission handler
func NewServicer(broker Broker, templates *templates.Templates, querier storage.Querier, pool *pgxpool.Pool) Servicer {
	return &ServicerImpl{
		broker:    broker,
		querier:   querier,
		pool:      pool,
		templates: templates,
	}
}
