package templates

import (
	"context"
	"log/slog"
	"net/http"
)

type GeneralErrorData struct {
	Message string
	Title   string
}

func RenderError(ctx context.Context, wr http.ResponseWriter, message string, code int, tmpls *Templates) {
	wr.WriteHeader(code)

	err := tmpls.Render(ctx, "generalerror", wr, GeneralErrorData{
		Message: message,
		Title:   http.StatusText(code),
	})
	if err != nil {
		slog.ErrorContext(ctx, "could not render general error", "error", err)
		http.Error(wr, message, code)
		return
	}
}
