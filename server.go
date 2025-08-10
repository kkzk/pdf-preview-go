package main

import (
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

// startHTTPServer starts a local HTTP server to serve PDF files
func (a *App) startHTTPServer() {
	// Find available port
	for port := 8080; port < 8090; port++ {
		mux := http.NewServeMux()

		// Serve PDF files from cache directory
		cacheDir := filepath.Join(os.TempDir(), "pdf-preview-go-cache")
		mux.Handle("/pdf/", http.StripPrefix("/pdf/", http.FileServer(http.Dir(cacheDir))))

		// Add CORS headers for WebView compatibility
		corsHandler := func(h http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Access-Control-Allow-Origin", "*")
				w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
				w.Header().Set("Access-Control-Allow-Headers", "*")
				if r.Method == "OPTIONS" {
					w.WriteHeader(http.StatusOK)
					return
				}
				h.ServeHTTP(w, r)
			})
		}

		a.httpServer = &http.Server{
			Addr:    ":" + strconv.Itoa(port),
			Handler: corsHandler(mux),
		}

		go func() {
			if err := a.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				// Port might be in use, try next one
			}
		}()

		// Test if server started successfully
		time.Sleep(100 * time.Millisecond)
		resp, err := http.Get("http://localhost:" + strconv.Itoa(port) + "/pdf/")
		if err == nil {
			resp.Body.Close()
			a.httpPort = port
			return
		}
	}
}
