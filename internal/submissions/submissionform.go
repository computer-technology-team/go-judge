package submissions

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/computer-technology-team/go-judge/web/templates"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
)

// SubmissionForm implements Handler.
func (s *ServicerImpl) SubmissionForm(w http.ResponseWriter, r *http.Request) {
	logger := slog.With("function", "SubmissionForm", "package", "submissions")

	ctx := r.Context()

	problemID, err := strconv.Atoi(chi.URLParam(r, "problem_id"))
	if err != nil {
		logger.WarnContext(ctx, "problem id is invalid", "error", err,
			"problem_id", chi.URLParam(r, "problem_id"))
		templates.RenderError(ctx, w, "problem id is invalid", http.StatusBadRequest, s.templates)
		return
	}

	logger = logger.With("problem_id", problemID)

	problem, err := s.querier.GetProblemByID(ctx, s.pool, int32(problemID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			templates.RenderError(ctx, w, "problem not found", http.StatusNotFound, s.templates)
			http.Error(w, "problem not found", http.StatusNotFound)
			return
		}

		logger.ErrorContext(ctx, "could not retrieve problem", "error", err)
		templates.RenderError(ctx, w, "could not retrieve problem", http.StatusInternalServerError, s.templates)
		return
	}

	err = s.templates.Render(ctx, "submit", w, problem)
	if err != nil {
		logger.ErrorContext(ctx, "could not render template", "error", err)
		templates.RenderError(ctx, w, "could not render", http.StatusInternalServerError, s.templates)
		return
	}
}
