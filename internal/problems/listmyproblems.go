package problems

import (
	"log/slog"
	"net/http"
	"strconv"

	"github.com/computer-technology-team/go-judge/internal/context"
	"github.com/computer-technology-team/go-judge/internal/storage"
	"github.com/computer-technology-team/go-judge/web/templates"
	"github.com/samber/lo"
)

func (h *DefaultHandler) ListMyProblems(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user, ok := context.GetUserFromContext(ctx)
	if !ok {
		templates.RenderError(ctx, w, "user not found in context", http.StatusUnauthorized, h.templates)
		return
	}

	pageStr := r.URL.Query().Get("page")
	pageSizeStr := r.URL.Query().Get("page-size")

	var page int
	if pageStr != "" {
		var err error
		page, err = strconv.Atoi(pageStr)
		if err != nil {
			slog.WarnContext(ctx, "could not convert page param", "error", err)
			templates.RenderError(ctx, w, "invalid page param", http.StatusBadRequest, h.templates)
			return
		}
	} else {
		page = 1 // Default to first page
	}

	var pageSize int
	if pageSizeStr != "" {
		var err error
		pageSize, err = strconv.Atoi(pageSizeStr)
		if err != nil {
			slog.WarnContext(ctx, "could not convert page size param", "error", err)
			templates.RenderError(ctx, w, "invalid page size param", http.StatusBadRequest, h.templates)
			return
		}
	} else {
		pageSize = defaultPageSize
	}

	limit := pageSize * page
	offset := pageSize * (page - 1)
	var problems []any

	if !user.Superuser {
		userProbs, err := h.querier.GetUserProblemsSorted(ctx, h.pool, storage.GetUserProblemsSortedParams{
			Limit:     int32(limit),
			Offset:    int32(offset),
			CreatedBy: user.ID,
		})
		if err != nil {
			slog.Error("could not fetch user problems", "error", err)
			http.Error(w, "could not fetch problems", http.StatusInternalServerError)
			return
		}
		problems = lo.ToAnySlice(userProbs)
	} else {
		allProbs, err := h.querier.GetAllProblemsSorted(ctx, h.pool,
			int32(limit),
			int32(offset),
		)
		if err != nil {
			slog.Error("could not fetch all problems", "error", err)
			http.Error(w, "could not fetch problems", http.StatusInternalServerError)
			return
		}
		problems = lo.ToAnySlice(allProbs)
	}

	err := h.templates.Render(ctx, "listmyproblemspage", w, listProblemsData{
		Problems: problems, CurrentPage: page, PageSize: pageSize,
	})
	if err != nil {
		slog.Error("could not render listmyproblemspage", "error", err)
		http.Error(w, "could not render", http.StatusInternalServerError)
		return
	}
}
