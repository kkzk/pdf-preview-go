package main

import (
	"context"
	"log"
	"sync"
	"time"

	"pdf-preview-go/internal/appdata"
	"pdf-preview-go/internal/convert"
	"pdf-preview-go/internal/watch"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// App is bound to the frontend: its exported methods are callable from it
type App struct {
	ctx         context.Context
	converter   *convert.OfficeConverter
	watcher     *watch.Watcher
	initialDir  string // Initial directory to open
	initialFile string // File to select at startup (from the command line), if any

	// convertMu serializes conversions so that only one runs at a time
	convertMu sync.Mutex
	// stopCh is closed on shutdown to stop background goroutines
	stopCh chan struct{}

	// mu guards the fields below, which are accessed from the watcher's
	// goroutine and frontend calls
	mu                  sync.Mutex
	lastConvertedFiles  []string
	lastConvertedSheets map[string][]string
	currentPdfPath      string // Current PDF file path in the cache
	hasUnsavedChanges   bool   // Whether there are unsaved changes
}

// NewApp creates the app. initialFile, if not empty, is selected in initialDir.
func NewApp(initialDir, initialFile string) *App {
	app := &App{
		// The converter creates the cache directory when it converts
		converter:           convert.NewOfficeConverter(appdata.PDFCacheDir()),
		initialDir:          initialDir,
		initialFile:         initialFile,
		lastConvertedFiles:  []string{},
		lastConvertedSheets: make(map[string][]string),
		stopCh:              make(chan struct{}),
	}
	app.watcher = watch.New(app.onFileChanged, func(err error) {
		runtime.LogErrorf(app.ctx, "File watcher error: %v", err)
	})

	return app
}

// Startup is called when the app starts. The context passed
// is the app's context. Additional initialization can be done here.
func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx

	// Move data stored in the cache directory by earlier versions. This runs
	// before the frontend loads the directory history and session.
	appdata.MigrateLegacyData()

	// Start watching converted files for auto-regeneration
	a.watcher.Start()

	// Clean up old cache files
	go func() {
		// Cleanup session cache (older than 3 months)
		if err := appdata.CleanupDirectorySessionCache(90 * 24 * time.Hour); err != nil {
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
	a.watcher.Stop()
	// Note: OfficeConverter doesn't have a Close method
	// COM objects are automatically cleaned up
}
