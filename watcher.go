package main

import (
	"fmt"
	"os"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

const (
	// regenerateDebounce is how long to wait after the last change before
	// regenerating, so that a burst of file events results in one conversion
	regenerateDebounce = 1 * time.Second
	// pollingInterval is the interval for polling file changes (backup for fsnotify)
	pollingInterval = 2 * time.Second
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

// handleFileEvent processes file system events.
// Any event in the watched directory (including Office lock files such as
// ~$Book.xlsx) only requests a check; regenerateLoop regenerates the PDF
// only if a converted file's modification time has actually changed.
func (a *App) handleFileEvent(event fsnotify.Event) {
	if event.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Rename|fsnotify.Remove) == 0 {
		return
	}
	a.requestRegenerate()
}

// StartWatchingDirectory starts watching a directory for file changes
func (a *App) StartWatchingDirectory(dirPath string) error {
	if a.watcher == nil {
		return fmt.Errorf("file watcher not initialized")
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	if a.watchedDir == dirPath {
		return nil
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
	a.mu.Lock()
	defer a.mu.Unlock()
	a.autoUpdateEnabled = enabled
}

// GetAutoUpdateEnabled returns current auto-update status
func (a *App) GetAutoUpdateEnabled() bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.autoUpdateEnabled
}

// requestRegenerate asks regenerateLoop to regenerate the PDF.
// Requests made while one is already pending are coalesced.
func (a *App) requestRegenerate() {
	select {
	case a.regenerateCh <- struct{}{}:
	default: // a request is already pending
	}
}

// regenerateLoop regenerates the PDF on request, waiting until changes have
// settled for regenerateDebounce. Runs until the app shuts down.
func (a *App) regenerateLoop() {
	for {
		select {
		case <-a.stopCh:
			return
		case <-a.regenerateCh:
		}

		// Debounce: wait until no new request arrives for regenerateDebounce
		timer := time.NewTimer(regenerateDebounce)
	debounce:
		for {
			select {
			case <-a.stopCh:
				timer.Stop()
				return
			case <-a.regenerateCh:
				timer.Reset(regenerateDebounce)
			case <-timer.C:
				break debounce
			}
		}

		// Regenerate only if a converted file has actually been modified
		changedFile := a.findChangedFile()
		if changedFile == "" {
			continue
		}

		runtime.EventsEmit(a.ctx, "file-changed", map[string]interface{}{
			"file": changedFile,
		})
		a.autoRegeneratePDF()
	}
}

// findChangedFile returns the first converted file whose modification time
// differs from the one recorded at the last conversion, or "" if none has
// changed or auto-update is disabled.
func (a *App) findChangedFile() string {
	a.mu.Lock()
	defer a.mu.Unlock()

	if !a.autoUpdateEnabled {
		return ""
	}
	for _, filePath := range a.lastConvertedFiles {
		info, err := os.Stat(filePath)
		if err != nil {
			continue // e.g. temporarily missing while Office saves
		}
		if lastModTime, exists := a.fileModTimes[filePath]; exists && !info.ModTime().Equal(lastModTime) {
			return filePath
		}
	}
	return ""
}

// autoRegeneratePDF automatically regenerates PDF when files change
func (a *App) autoRegeneratePDF() {
	a.mu.Lock()
	convertedFiles := a.lastConvertedFiles
	convertedSheets := a.lastConvertedSheets
	a.mu.Unlock()

	if len(convertedFiles) == 0 {
		return
	}

	// Check if all files still exist
	validFiles := []string{}
	for _, filePath := range convertedFiles {
		if _, err := os.Stat(filePath); err == nil {
			validFiles = append(validFiles, filePath)
		}
	}

	if len(validFiles) == 0 {
		return
	}

	// Re-convert with same sheet selections
	_, err := a.ConvertToPDF(validFiles, convertedSheets)
	if err != nil {
		runtime.EventsEmit(a.ctx, "conversion:error", map[string]interface{}{
			"message": "Auto-update failed: " + err.Error(),
		})
	}
}

// statModTimes returns the modification times of files
func statModTimes(filePaths []string) map[string]time.Time {
	modTimes := make(map[string]time.Time)
	for _, filePath := range filePaths {
		if info, err := os.Stat(filePath); err == nil {
			modTimes[filePath] = info.ModTime()
		}
	}
	return modTimes
}

// pollFileModifications polls for file changes as a backup for fsnotify.
// Runs until the app shuts down.
func (a *App) pollFileModifications() {
	ticker := time.NewTicker(pollingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-a.stopCh:
			return
		case <-ticker.C:
			a.checkFileModifications()
		}
	}
}

// checkFileModifications requests regeneration if any converted file has
// been modified. Missed fsnotify events are caught here.
func (a *App) checkFileModifications() {
	if a.findChangedFile() != "" {
		a.requestRegenerate()
	}
}
