package static

import (
	"embed"
	"net/http"
)

//go:embed css images
var staticFS embed.FS

func StaticFilerHandler() http.Handler {
	return http.FileServer(http.FS(staticFS))
}
