package submissions

import (
	"log/slog"
	"net/http"

	"github.com/computer-technology-team/go-judge/internal/context"
	"github.com/computer-technology-team/go-judge/web/templates"
)

// ListSubmissions returns a list of submissions
func (s *ServicerImpl) ListSubmissions(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	user, _ := context.GetUserFromContext(ctx)

	submissions, err := s.querier.GetUserSubmissions(ctx, s.pool, user.ID)
	if err != nil {
		slog.Error("could not retrieve submissions", "error", err)
		templates.RenderError(ctx, w, "could not retrieve submissions", http.StatusInternalServerError, s.templates)
		return
	}

	err = s.templates.Render(ctx, "submissionslist", w, submissions)
	if err != nil {
		slog.Error("could not render submissionslist template", "error", err)
		templates.RenderError(ctx, w, "could not render template", http.StatusInternalServerError, s.templates)
		return
	}
}
