package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// NewApp creates a new App application struct
func NewApp(initialDir string) *App {
	// Create cache directory
	cacheDir := filepath.Join(os.TempDir(), "pdf-preview-go-cache")
	os.MkdirAll(cacheDir, 0755)

	app := &App{
		converter:           NewOfficeConverter(cacheDir),
		initialDir:          initialDir,
		httpPort:            0, // Will be set when server starts
		watchedDir:          "",
		lastConvertedFiles:  []string{},
		lastConvertedSheets: make(map[string][]string),
		autoUpdateEnabled:   true,
		fileModTimes:        make(map[string]time.Time),
		currentPdfPath:      "",
		savedPdfPath:        "",
		hasUnsavedChanges:   false,
	}

	return app
}

// Startup is called when the app starts. The context passed
// is the app's context. Additional initialization can be done here.
func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx

	// Start HTTP server for serving PDF files
	a.startHTTPServer()

	// Initialize file watcher
	a.initFileWatcher()

	// Clean up old cache files (older than 30 days)
	go func() {
		if err := a.CleanupSheetSelectionsCache(30 * 24 * time.Hour); err != nil {
			fmt.Printf("Warning: failed to cleanup sheet selection cache: %v\n", err)
		}

		// Cleanup session cache (older than 3 months)
		if err := a.CleanupDirectorySessionCache(90 * 24 * time.Hour); err != nil {
			fmt.Printf("Warning: failed to cleanup session cache: %v\n", err)
		}

		// Also cleanup PDF cache
		if err := a.converter.CleanupCache(30 * 24 * time.Hour); err != nil {
			fmt.Printf("Warning: failed to cleanup PDF cache: %v\n", err)
		}
	}()
}

// Shutdown is called when the app is closing
func (a *App) Shutdown(ctx context.Context) {
	if a.pollingTicker != nil {
		a.pollingTicker.Stop()
	}
	if a.watcher != nil {
		a.watcher.Close()
	}
	if a.httpServer != nil {
		a.httpServer.Close()
	}
	// Note: OfficeConverter doesn't have a Close method
	// COM objects are automatically cleaned up
}
