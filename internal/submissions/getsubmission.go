package submissions

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/computer-technology-team/go-judge/internal/context"
	"github.com/computer-technology-team/go-judge/web/templates"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// GetSubmission returns a specific submission
func (s *ServicerImpl) GetSubmission(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := chi.URLParam(r, "id")

	idUUID, err := uuid.Parse(id)
	if err != nil {
		templates.RenderError(ctx, w, "invalid id, id must be uuid", http.StatusBadRequest, s.templates)
		return
	}

	user, _ := context.GetUserFromContext(ctx)

	submission, err := s.querier.GetSubmissionForUser(ctx, s.pool, user.ID, pgtype.UUID{
		Bytes: idUUID,
		Valid: true,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			templates.RenderError(ctx, w, "submission not found", http.StatusNotFound, s.templates)
			return
		}
		slog.Error("could not get submission from database", "error", err)
		templates.RenderError(ctx, w, "could not retrieve submission", http.StatusInternalServerError, s.templates)
		return
	}

	err = s.templates.Render(ctx, "submission", w, submission)
	if err != nil {
		slog.Error("could not render submssion template", "error", err)
		templates.RenderError(ctx, w, "could not render template", http.StatusInternalServerError, s.templates)
		return
	}
}
