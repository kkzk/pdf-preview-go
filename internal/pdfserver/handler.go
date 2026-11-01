// Package pdfserver serves converted PDFs to the frontend.
package pdfserver

import (
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
)

// urlPrefix is the URL path under which converted PDFs are served
const urlPrefix = "/pdf/"

// NewHandler returns a handler that serves PDF files from dir. It is
// registered as the Wails AssetServer handler, so PDFs are served from the
// same origin as the frontend and are not reachable from outside the app.
func NewHandler(dir string) http.Handler {
	fileServer := http.StripPrefix(urlPrefix, http.FileServer(http.Dir(dir)))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Serve PDF files only (no directory listing, no session JSON)
		if !strings.HasPrefix(r.URL.Path, urlPrefix) ||
			!strings.EqualFold(filepath.Ext(r.URL.Path), ".pdf") {
			http.NotFound(w, r)
			return
		}
		fileServer.ServeHTTP(w, r)
	})
}

// URL returns the URL of a PDF file in the served directory, with a cache
// buster
func URL(fileName string, version int64) string {
	return urlPrefix + fileName + "?v=" + strconv.FormatInt(version, 10)
}
