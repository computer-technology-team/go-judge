package problems

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/jackc/pgx/v5"
)

func (h *DefaultHandler) ListProblems(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tx, err := h.pool.Begin(ctx)
	if err != nil {
		slog.Error("could not begin transaction", "error", err)
		http.Error(w, "could not start saving", http.StatusInternalServerError)
		return
	}

	defer func(ctx context.Context, tx pgx.Tx) {
		err := tx.Rollback(ctx)
		if !errors.Is(err, pgx.ErrTxClosed) {
			slog.Error("could not revert transaction", "error", err)
		}
	}(ctx, tx)

	problems, err := h.querier.GetAllProblemsSorted(ctx, tx)
	if err != nil {
		slog.Error("could not fetch problems", "error", err)
		http.Error(w, "could not fetch problems", http.StatusInternalServerError)
		return
	}

	err = h.templates.Render(ctx, "listproblemspage", w, problems)
	if err != nil {
		slog.Error("could not render listproblemspage", "error", err)
		http.Error(w, "could not render", http.StatusInternalServerError)
		return
	}
}
