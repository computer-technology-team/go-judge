package templates

import (
	"context"
	"log/slog"
	"net/http"
)

const imagesDir = "/static/images/"

type GeneralErrorData struct {
	Message  string
	Title    string
	Image    string
	ImageAlt string
}

func RenderError(ctx context.Context, wr http.ResponseWriter, message string, code int, tmpls *Templates) {
	wr.WriteHeader(code)

	data := GeneralErrorData{
		Message: message,
		Title:   http.StatusText(code),
	}

	data.Image, data.ImageAlt = codeToImage(code)

	err := tmpls.Render(ctx, "generalerror", wr, data)
	if err != nil {
		slog.ErrorContext(ctx, "could not render general error", "error", err)
		http.Error(wr, message, code)
		return
	}
}

func codeToImage(code int) (string, string) {
	if code >= 500 {
		return imagesDir + "error-this-is-fine.png", "this is fine gopher meme"
	}

	switch code {
	case http.StatusBadRequest, http.StatusUnauthorized:
		return imagesDir + "error-gopher-no.png", "gopher holding a stop sign"
	default:
		return imagesDir + "error-this-is-fine.png", "this is fine gopher meme"
	}

}
