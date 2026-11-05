package main

import (
	"fmt"
	"os"
	"path/filepath"

	"pdf-preview-go/internal/convert"
)

// savePdfAs saves the current PDF to a specified location
func (a *App) savePdfAs(savePath string) error {
	a.mu.Lock()
	currentPdfPath := a.currentPdfPath
	a.mu.Unlock()

	if currentPdfPath == "" {
		return fmt.Errorf("no PDF to save")
	}

	// Create destination directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(savePath), 0755); err != nil {
		return fmt.Errorf("failed to create destination directory: %v", err)
	}
	if err := convert.CopyFile(currentPdfPath, savePath); err != nil {
		return fmt.Errorf("failed to copy file: %v", err)
	}

	a.mu.Lock()
	a.hasUnsavedChanges = false
	a.mu.Unlock()

	return nil
}

// defaultSavePath returns the default save path: the working directory's
// name with .pdf, next to it
func (a *App) defaultSavePath() string {
	if a.initialDir == "" {
		return ""
	}
	dir := filepath.Clean(a.initialDir)
	return filepath.Join(filepath.Dir(dir), filepath.Base(dir)+".pdf")
}

// HasUnsavedChanges returns whether there are unsaved changes
func (a *App) HasUnsavedChanges() bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.hasUnsavedChanges
}
