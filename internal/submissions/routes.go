package submissions

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

// Handler defines the interface for submission handlers
type Handler interface {
	ListSubmissions(w http.ResponseWriter, r *http.Request)
	CreateSubmission(w http.ResponseWriter, r *http.Request)
	GetSubmission(w http.ResponseWriter, r *http.Request)
	UpdateSubmission(w http.ResponseWriter, r *http.Request)
	DeleteSubmission(w http.ResponseWriter, r *http.Request)
}

// NewRoutes returns a function that registers routes with the given handler
// This allows for dependency injection when setting up routes
func NewRoutes(h Handler) func(r chi.Router) {
	return func(r chi.Router) {
		r.Get("/", h.ListSubmissions)
		r.Post("/", h.CreateSubmission)
		r.Get("/{id}", h.GetSubmission)
		r.Put("/{id}", h.UpdateSubmission)
		r.Delete("/{id}", h.DeleteSubmission)
	}
}

// DefaultHandler is the default implementation of the Handler interface
type DefaultHandler struct {
	// Dependencies can be injected here (e.g., service, repository)
}

// NewHandler creates a new instance of the default submission handler
func NewHandler() Handler {
	return &DefaultHandler{}
}

// ListSubmissions returns a list of submissions
func (h *DefaultHandler) ListSubmissions(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement list submissions logic
	w.Write([]byte("List submissions endpoint"))
}

// CreateSubmission creates a new submission
func (h *DefaultHandler) CreateSubmission(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement create submission logic
	w.Write([]byte("Create submission endpoint"))
}

// GetSubmission returns a specific submission
func (h *DefaultHandler) GetSubmission(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement get submission logic
	id := chi.URLParam(r, "id")
	w.Write([]byte("Get submission endpoint: " + id))
}

// UpdateSubmission updates a specific submission
func (h *DefaultHandler) UpdateSubmission(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement update submission logic
	id := chi.URLParam(r, "id")
	w.Write([]byte("Update submission endpoint: " + id))
}

// DeleteSubmission deletes a specific submission
func (h *DefaultHandler) DeleteSubmission(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement delete submission logic
	id := chi.URLParam(r, "id")
	w.Write([]byte("Delete submission endpoint: " + id))
}
