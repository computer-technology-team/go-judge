package middleware

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/computer-technology-team/go-judge/web/templates"
)

func NewRecoveryHandler(tmpls *templates.Templates) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func(ctx context.Context) {
				if rvr := recover(); rvr != nil {
					slog.Error("panic happened", "panic", rvr)
					templates.RenderError(ctx, w, "PANNNIICCCCCC.....", http.StatusInternalServerError, tmpls)
				}
			}(r.Context())
			next.ServeHTTP(w, r)
		})
	}
}
