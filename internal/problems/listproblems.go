package problems

import (
	"log/slog"
	"net/http"
	"strconv"

	"github.com/computer-technology-team/go-judge/web/templates"
	"github.com/samber/lo"
)

const defaultPageSize = 5

type listProblemsData struct {
	Problems    []any
	CurrentPage int
	PageSize    int
}

func (h *DefaultHandler) ListProblems(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

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

	limit := pageSize
	offset := pageSize * (page - 1)

	problems, err := h.querier.GetAllPublishedProblemsSorted(ctx, h.pool, int32(limit), int32(offset))
	if err != nil {
		slog.Error("could not fetch problems", "error", err)
		templates.RenderError(ctx, w, "could not fetch problems", http.StatusInternalServerError, h.templates)
		return
	}

	err = h.templates.Render(ctx, "listproblemspage", w, listProblemsData{
		Problems: lo.ToAnySlice(problems), CurrentPage: page, PageSize: pageSize,
	})
	if err != nil {
		slog.Error("could not render listproblemspage", "error", err)
		templates.RenderError(ctx, w, "could not render", http.StatusInternalServerError, h.templates)
		return
	}
}
