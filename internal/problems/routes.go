package problems

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

// Handler defines the interface for problem handlers
type Handler interface {
	ListProblems(w http.ResponseWriter, r *http.Request)
	CreateProblem(w http.ResponseWriter, r *http.Request)
	GetProblem(w http.ResponseWriter, r *http.Request)
	UpdateProblem(w http.ResponseWriter, r *http.Request)
	DeleteProblem(w http.ResponseWriter, r *http.Request)
}

// NewRoutes returns a function that registers routes with the given handler
// This allows for dependency injection when setting up routes
func NewRoutes(h Handler) func(r chi.Router) {
	return func(r chi.Router) {
		r.Get("/", h.ListProblems)
		r.Post("/", h.CreateProblem)
		r.Get("/{id}", h.GetProblem)
		r.Put("/{id}", h.UpdateProblem)
		r.Delete("/{id}", h.DeleteProblem)
	}
}

// DefaultHandler is the default implementation of the Handler interface
type DefaultHandler struct {
	// Dependencies can be injected here (e.g., service, repository)
}

// NewHandler creates a new instance of the default problem handler
func NewHandler() Handler {
	return &DefaultHandler{}
}

// ListProblems returns a list of problems
func (h *DefaultHandler) ListProblems(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement list problems logic
	w.Write([]byte("List problems endpoint"))
}

// CreateProblem creates a new problem
func (h *DefaultHandler) CreateProblem(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement create problem logic
	w.Write([]byte("Create problem endpoint"))
}

// GetProblem returns a specific problem
func (h *DefaultHandler) GetProblem(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement get problem logic
	id := chi.URLParam(r, "id")
	w.Write([]byte("Get problem endpoint: " + id))
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
