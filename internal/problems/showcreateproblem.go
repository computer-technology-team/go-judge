package problems

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/computer-technology-team/go-judge/internal/context"
	"github.com/computer-technology-team/go-judge/internal/storage"
	"github.com/computer-technology-team/go-judge/web/templates"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
)

type problemFormData struct {
	Problem   storage.Problem
	TestCases []storage.TestCase
}

func (h *DefaultHandler) ProblemForm(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	idStr := chi.URLParam(r, "id")
	user, _ := context.GetUserFromContext(ctx)

	var data *problemFormData
	if idStr != "new" {
		id, err := strconv.Atoi(idStr)
		if err != nil {
			templates.RenderError(ctx, w, "invalid problem id", http.StatusBadRequest, h.templates)
			return
		}
		problem, err := h.querier.GetProblemForUser(ctx, h.pool, storage.GetProblemForUserParams{
			ID:        int32(id),
			CreatedBy: user.ID,
			IsAdmin:   user.Superuser,
		})
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				templates.RenderError(ctx, w, "problem not found", http.StatusBadRequest, h.templates)
				return
			}
			templates.RenderError(ctx, w, "could not get problem from storage", http.StatusBadRequest, h.templates)
			return
		}

		testCases, err := h.querier.GetTestCasesByProblemID(ctx, h.pool, problem.ID)
		if err != nil {
			templates.RenderError(ctx, w, "could not get problem from storage", http.StatusBadRequest, h.templates)
			return
		}

		data = &problemFormData{
			Problem:   problem,
			TestCases: testCases,
		}
	}

	err := h.templates.Render(r.Context(), "createproblempage", w, data)
	if err != nil {
		slog.Error("could not render createproblempage", "error", err)
		templates.RenderError(r.Context(), w, "could not render", http.StatusInternalServerError, h.templates)
		return
	}
	w.WriteHeader(http.StatusOK)
}
