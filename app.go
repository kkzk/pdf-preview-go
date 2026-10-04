package main

import (
	"context"
	"log"
	"os"
	"time"
)

// NewApp creates a new App application struct
func NewApp(initialDir string) *App {
	// Create cache directory
	cacheDir := pdfCacheDir()
	os.MkdirAll(cacheDir, 0755)

	app := &App{
		converter:           NewOfficeConverter(cacheDir),
		initialDir:          initialDir,
		watchedDir:          "",
		lastConvertedFiles:  []string{},
		lastConvertedSheets: make(map[string][]string),
		autoUpdateEnabled:   true,
		fileModTimes:        make(map[string]time.Time),
		currentPdfPath:      "",
		savedPdfPath:        "",
		hasUnsavedChanges:   false,
		regenerateCh:        make(chan struct{}, 1),
		stopCh:              make(chan struct{}),
	}

	return app
}

// Startup is called when the app starts. The context passed
// is the app's context. Additional initialization can be done here.
func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx

	// Move data stored in the cache directory by earlier versions. This runs
	// before the frontend loads the directory history and session.
	migrateLegacyData()

	// Initialize file watcher, polling and auto-regeneration
	a.initFileWatcher()
	go a.pollFileModifications()
	go a.regenerateLoop()

	// Clean up old cache files
	go func() {
		// Cleanup session cache (older than 3 months)
		if err := a.CleanupDirectorySessionCache(90 * 24 * time.Hour); err != nil {
			log.Printf("Warning: failed to cleanup session cache: %v", err)
		}

		// Also cleanup PDF cache
		if err := a.converter.CleanupCache(30 * 24 * time.Hour); err != nil {
			log.Printf("Warning: failed to cleanup PDF cache: %v", err)
		}
	}()
}

// Shutdown is called when the app is closing
func (a *App) Shutdown(ctx context.Context) {
	close(a.stopCh)
	if a.watcher != nil {
		a.watcher.Close()
	}
	// Note: OfficeConverter doesn't have a Close method
	// COM objects are automatically cleaned up
}
