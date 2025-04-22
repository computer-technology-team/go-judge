package problems

import (
	"log/slog"
	"net/http"
)

func (h *DefaultHandler) ShowCreateProblem(w http.ResponseWriter, r *http.Request) {
	err := h.templates.Render(r.Context(), "createproblempage", w, nil)
	if err != nil {
		slog.Error("could not render createproblempage", "error", err)
		http.Error(w, "could not render", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}
