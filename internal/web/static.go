package web

import (
	"io/fs"
	"net/http"
	"strings"
)

// spaHandler serves files from distFS, falling back to index.html for any
// path that doesn't match a real file — so client-side routes (e.g.
// /login, reached by a direct URL or a page refresh) resolve to the SPA
// instead of a 404.
func spaHandler(distFS fs.FS) http.Handler {
	fileServer := http.FileServer(http.FS(distFS))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/")
		if path == "" {
			path = "index.html"
		}

		if _, err := fs.Stat(distFS, path); err != nil {
			r = r.Clone(r.Context())
			r.URL.Path = "/"
		}

		fileServer.ServeHTTP(w, r)
	})
}
