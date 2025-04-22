package problems

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/computer-technology-team/go-judge/internal/context"
	"github.com/computer-technology-team/go-judge/web/templates"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
)

func (h *DefaultHandler) ToggleStatus(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user, _ := context.GetUserFromContext(ctx)
	if !user.Superuser {
		templates.RenderError(ctx, w, "only admins can publish", http.StatusUnauthorized, h.templates)
		return
	}

	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		templates.RenderError(ctx, w, "invalid problem id", http.StatusBadRequest, h.templates)
		return
	}

	problem, err := h.querier.GetProblemByID(ctx, h.pool, int32(id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			templates.RenderError(ctx, w, "problem not found", http.StatusBadRequest, h.templates)
			return
		}
		templates.RenderError(ctx, w, "could not retrive problem", http.StatusInternalServerError, h.templates)
		return
	}

	if problem.Draft {
		err = h.querier.PublishProblem(ctx, h.pool, int32(id))
		if err != nil {
			templates.RenderError(ctx, w, "could not publish problem", http.StatusBadRequest, h.templates)
			return
		}
	} else {
		err = h.querier.DraftProblem(ctx, h.pool, int32(id))
		if err != nil {
			templates.RenderError(ctx, w, "could not publish problem", http.StatusBadRequest, h.templates)
			return
		}
	}

	http.Redirect(w, r, "/problems/my", http.StatusMovedPermanently)
}
