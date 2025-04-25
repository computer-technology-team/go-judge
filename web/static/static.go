package static

import (
	"crypto/md5"
	"embed"
	"fmt"
	"net/http"
	"time"
)

//go:embed css images favicon
var staticFS embed.FS

var serverStartTime = time.Now().Unix()

type CacheControlHandler struct {
	handler http.Handler
	maxAge  time.Duration
	etag    string
}

func (c *CacheControlHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", fmt.Sprintf("public, max-age=%d", int(c.maxAge.Seconds())))
	w.Header().Set("Expires", time.Now().Add(c.maxAge).UTC().Format(http.TimeFormat))

	w.Header().Set("ETag", c.etag)

	if match := r.Header.Get("If-None-Match"); match != "" {
		if match == c.etag {
			w.WriteHeader(http.StatusNotModified)
			return
		}
	}

	c.handler.ServeHTTP(w, r)
}

func StaticFilerHandler() http.Handler {
	fileServer := http.FileServer(http.FS(staticFS))

	etag := fmt.Sprintf("W/\"%x\"", md5.Sum(fmt.Appendf(nil, "%d", serverStartTime)))

	return &CacheControlHandler{
		handler: fileServer,
		maxAge:  7 * 24 * time.Hour, // 7 days
		etag:    etag,
	}
}
