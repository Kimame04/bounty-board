package web

import (
	"embed"
	"io"
	"io/fs"
	"net/http"
	"strings"
)

//go:embed all:dist
var embeddedFiles embed.FS

// SPAHandler returns an http.Handler that serves the embedded React SPA.
// If a static asset exists, it is served directly.
// Otherwise, it transparently falls back to index.html for client-side routing.
func SPAHandler() http.Handler {
	distFS, err := fs.Sub(embeddedFiles, "dist")
	if err != nil {
		panic("failed to locate embedded dist folder: " + err.Error())
	}

	fileServer := http.FileServer(http.FS(distFS))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/")
		if path == "" {
			path = "index.html"
		}

		// Check if file exists in embedded filesystem
		f, err := distFS.Open(path)
		if err == nil {
			stat, statErr := f.Stat()
			_ = f.Close()
			if statErr == nil && !stat.IsDir() {
				// Cache static assets aggressively, but not index.html
				if strings.HasPrefix(path, "assets/") {
					w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
				} else {
					w.Header().Set("Cache-Control", "no-cache")
				}
				fileServer.ServeHTTP(w, r)
				return
			}
		}

		// Fallback to index.html for any SPA routes
		indexFile, err := distFS.Open("index.html")
		if err != nil {
			http.Error(w, "index.html not found in embedded assets", http.StatusNotFound)
			return
		}
		defer indexFile.Close()

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-cache")
		_, _ = io.Copy(w, indexFile)
	})
}
