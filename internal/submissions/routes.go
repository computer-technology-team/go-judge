package submissions

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	internalcontext "github.com/computer-technology-team/go-judge/internal/context"
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

// ListSubmissions returns a list of submissions
func (s *ServicerImpl) ListSubmissions(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement list submissions logic
	w.Write([]byte("List submissions endpoint"))
}

// CreateSubmission creates a new submission
func (s *ServicerImpl) CreateSubmission(w http.ResponseWriter, r *http.Request) {
	logger := slog.With("function", "CreateSubmission", "package", "submissions")
	ctx := r.Context()

	err := r.ParseForm()
	if err != nil {
		http.Error(w, "could not parse form", http.StatusBadRequest)
		return
	}

	user, _ := internalcontext.GetUserFromContext(ctx)

	// Parse form data
	problemIDStr := r.FormValue("problem_id")
	code := r.FormValue("code")

	// Validate form data
	if code == "" {
		http.Error(w, "solution code is required", http.StatusBadRequest)
		return
	}

	problemID, err := strconv.Atoi(problemIDStr)
	if err != nil {
		logger.WarnContext(ctx, "problem id is invalid", "error", err,
			"problem_id", problemIDStr)
		http.Error(w, "problem id is invalid", http.StatusBadRequest)
		return
	}

	logger = logger.With("problem_id", problemID, "user_id", user.ID)

	// Create submission in database
	submissionParams := storage.CreateSubmissionParams{
		ProblemID:    int32(problemID),
		UserID:       user.ID,
		SolutionCode: code,
	}

	submission, err := s.querier.CreateSubmission(ctx, s.pool, submissionParams)
	if err != nil {
		logger.ErrorContext(ctx, "could not create submission", "error", err)
		http.Error(w, "could not create submission", http.StatusInternalServerError)
		return
	}

	go s.broker.AddSubmissionEvaluation(submission)
}

// GetSubmission returns a specific submission
func (s *ServicerImpl) GetSubmission(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement get submission logic
	id := chi.URLParam(r, "id")
	w.Write([]byte("Get submission endpoint: " + id))
}

// SubmissionForm implements Handler.
func (s *ServicerImpl) SubmissionForm(w http.ResponseWriter, r *http.Request) {
	logger := slog.With("function", "SubmissionForm", "package", "submissions")

	ctx := r.Context()

	problemID, err := strconv.Atoi(chi.URLParam(r, "problem_id"))
	if err != nil {
		logger.WarnContext(ctx, "problem id is invalid", "error", err,
			"problem_id", chi.URLParam(r, "problem_id"))
		http.Error(w, "problem id is invalid", http.StatusBadRequest)
		return
	}

	logger = logger.With("problem_id", problemID)

	problem, err := s.querier.GetProblemByID(ctx, s.pool, int32(problemID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			http.Error(w, "problem not found", http.StatusNotFound)
			return
		}

		logger.ErrorContext(ctx, "could not retrieve problem", "error", err)
		http.Error(w, "could not retrieve problem", http.StatusInternalServerError)
		return
	}

	err = s.templates.Render(ctx, "submit", w, problem)
	if err != nil {
		logger.ErrorContext(ctx, "could not render template", "error", err)
		http.Error(w, "could not render", http.StatusInternalServerError)
		return
	}
}
