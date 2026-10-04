package main

import (
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
)

// pdfURLPrefix is the URL path under which converted PDFs are served
const pdfURLPrefix = "/pdf/"

// pdfHandler returns a handler that serves PDF files from the cache directory.
// It is registered as the Wails AssetServer handler, so PDFs are served from
// the same origin as the frontend and are not reachable from outside the app.
func (a *App) pdfHandler() http.Handler {
	fileServer := http.StripPrefix(pdfURLPrefix, http.FileServer(http.Dir(a.converter.cacheDir)))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Serve PDF files only (no directory listing, no session JSON)
		if !strings.HasPrefix(r.URL.Path, pdfURLPrefix) ||
			!strings.EqualFold(filepath.Ext(r.URL.Path), ".pdf") {
			http.NotFound(w, r)
			return
		}
		fileServer.ServeHTTP(w, r)
	})
}

// buildPdfURL returns the URL of a cached PDF file with a cache buster
func buildPdfURL(fileName string, version int64) string {
	return pdfURLPrefix + fileName + "?v=" + strconv.FormatInt(version, 10)
}
