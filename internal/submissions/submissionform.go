package submissions

import (
	"errors"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"log/slog"
	"net/http"
	"strconv"
)

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
