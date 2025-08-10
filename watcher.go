package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// initFileWatcher initializes the file system watcher
func (a *App) initFileWatcher() {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return
	}
	a.watcher = watcher

	// Start watching in a goroutine
	go a.watchFiles()
}

// watchFiles runs the file watching loop
func (a *App) watchFiles() {
	for {
		select {
		case event, ok := <-a.watcher.Events:
			if !ok {
				return
			}
			a.handleFileEvent(event)
		case err, ok := <-a.watcher.Errors:
			if !ok {
				return
			}
			runtime.LogError(a.ctx, fmt.Sprintf("File watcher error: %v", err))
		}
	}
}

// handleFileEvent processes file system events
func (a *App) handleFileEvent(event fsnotify.Event) {
	if !a.autoUpdateEnabled || len(a.lastConvertedFiles) == 0 {
		return
	}

	// Check if the changed file is one of our converted files or related to them
	eventPath := filepath.Clean(event.Name)
	isWatchedFile := false
	watchedFilePath := ""

	for _, convertedFile := range a.lastConvertedFiles {
		cleanConvertedFile := filepath.Clean(convertedFile)

		// Direct match
		if cleanConvertedFile == eventPath {
			isWatchedFile = true
			watchedFilePath = convertedFile
			break
		}

		// Check for Excel temporary files (starts with ~$ or similar patterns)
		eventFileName := filepath.Base(eventPath)
		convertedFileName := filepath.Base(cleanConvertedFile)
		eventDir := filepath.Dir(eventPath)
		convertedDir := filepath.Dir(cleanConvertedFile)

		// Excel temporary file patterns
		if eventDir == convertedDir &&
			(strings.HasPrefix(eventFileName, "~$") ||
				strings.HasPrefix(eventFileName, ".~") ||
				strings.Contains(eventFileName, convertedFileName)) {
			isWatchedFile = true
			watchedFilePath = convertedFile
			break
		}
	}

	if !isWatchedFile {
		return
	}

	// Process Write, Remove, Create, and Rename events
	if event.Op&fsnotify.Write == fsnotify.Write ||
		event.Op&fsnotify.Remove == fsnotify.Remove ||
		event.Op&fsnotify.Create == fsnotify.Create ||
		event.Op&fsnotify.Rename == fsnotify.Rename {

		// Debounce - wait a short time for multiple events
		time.Sleep(500 * time.Millisecond)

		// Check if the actual target file still exists and has been modified
		if watchedFilePath != "" {
			if _, err := os.Stat(watchedFilePath); os.IsNotExist(err) {
				return // Target file was deleted
			}
		}

		// Emit event to frontend to trigger auto-update
		runtime.EventsEmit(a.ctx, "file-changed", map[string]interface{}{
			"file":      watchedFilePath,
			"operation": event.Op.String(),
		})

		// Auto-regenerate PDF
		go a.autoRegeneratePDF()
	}
}

// StartWatchingDirectory starts watching a directory for file changes
func (a *App) StartWatchingDirectory(dirPath string) error {
	if a.watcher == nil {
		return fmt.Errorf("file watcher not initialized")
	}

	// Stop watching previous directory
	if a.watchedDir != "" {
		a.watcher.Remove(a.watchedDir)
	}

	// Start watching new directory
	err := a.watcher.Add(dirPath)
	if err != nil {
		return err
	}

	a.watchedDir = dirPath
	return nil
}

// SetAutoUpdateEnabled enables or disables automatic PDF updates
func (a *App) SetAutoUpdateEnabled(enabled bool) {
	a.autoUpdateEnabled = enabled
}

// GetAutoUpdateEnabled returns current auto-update status
func (a *App) GetAutoUpdateEnabled() bool {
	return a.autoUpdateEnabled
}

// autoRegeneratePDF automatically regenerates PDF when files change
func (a *App) autoRegeneratePDF() {
	if len(a.lastConvertedFiles) == 0 {
		return
	}

	// Check if all files still exist
	validFiles := []string{}
	for _, filePath := range a.lastConvertedFiles {
		if _, err := os.Stat(filePath); err == nil {
			validFiles = append(validFiles, filePath)
		}
	}

	if len(validFiles) == 0 {
		return
	}

	// Re-convert with same sheet selections
	_, err := a.ConvertToPDF(validFiles, a.lastConvertedSheets)
	if err != nil {
		runtime.EventsEmit(a.ctx, "conversion:error", map[string]interface{}{
			"message": "Auto-update failed: " + err.Error(),
		})
	}
}

// recordFileModTimes records the modification times of files
func (a *App) recordFileModTimes(filePaths []string) {
	a.fileModTimes = make(map[string]time.Time)
	for _, filePath := range filePaths {
		if info, err := os.Stat(filePath); err == nil {
			a.fileModTimes[filePath] = info.ModTime()
		}
	}
}

// startPolling starts polling for file changes as backup
func (a *App) startPolling() {
	if a.pollingTicker != nil {
		a.pollingTicker.Stop()
	}

	// Poll every 2 seconds
	a.pollingTicker = time.NewTicker(2 * time.Second)

	go func() {
		for range a.pollingTicker.C {
			a.checkFileModifications()
		}
	}()
}

// checkFileModifications checks if any watched files have been modified
func (a *App) checkFileModifications() {
	if !a.autoUpdateEnabled || len(a.lastConvertedFiles) == 0 {
		return
	}

	hasChanges := false
	for _, filePath := range a.lastConvertedFiles {
		if info, err := os.Stat(filePath); err == nil {
			if lastModTime, exists := a.fileModTimes[filePath]; exists {
				if info.ModTime().After(lastModTime) {
					hasChanges = true
					a.fileModTimes[filePath] = info.ModTime()

					runtime.EventsEmit(a.ctx, "file-changed", map[string]interface{}{
						"file":      filePath,
						"operation": "MODIFIED (polling)",
					})
					break
				}
			}
		}
	}

	if hasChanges {
		go a.autoRegeneratePDF()
	}
}
