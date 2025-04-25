package problems

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/computer-technology-team/go-judge/web/templates"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
)

// ViewProblem returns a specific problem
func (h *DefaultHandler) ViewProblem(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	intID, err := strconv.Atoi(id)
	if err != nil {
		slog.Error("invalid problem ID", "error", err)
		templates.RenderError(r.Context(), w, "invalid problem ID", http.StatusBadRequest, h.templates)
		return
	}

	p, err := h.querier.GetProblemByID(r.Context(), h.pool, int32(intID))

	if err != nil {
		slog.Error("could not get problem by ID", "error", err)
		if errors.Is(err, pgx.ErrNoRows) {
			templates.RenderError(r.Context(), w, "problem not found", http.StatusNotFound, h.templates)
		} else {
			templates.RenderError(r.Context(), w, "could not get problem", http.StatusInternalServerError, h.templates)
		}
		return
	}

	err = h.templates.Render(r.Context(), "viewproblempage", w, p)
	if err != nil {
		slog.Error("could not render viewproblempage", "error", err)
		templates.RenderError(r.Context(), w, "could not render", http.StatusInternalServerError, h.templates)
		return
	}
}
