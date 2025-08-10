package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// SavePdfAs saves the current PDF to a specified location
func (a *App) SavePdfAs(savePath string) error {
	if a.currentPdfPath == "" {
		return fmt.Errorf("no PDF to save")
	}

	// Copy the current PDF to the specified location
	sourceFile, err := os.Open(a.currentPdfPath)
	if err != nil {
		return fmt.Errorf("failed to open source PDF: %v", err)
	}
	defer sourceFile.Close()

	// Create destination directory if it doesn't exist
	destDir := filepath.Dir(savePath)
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("failed to create destination directory: %v", err)
	}

	destFile, err := os.Create(savePath)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %v", err)
	}
	defer destFile.Close()

	// Copy file content
	_, err = destFile.ReadFrom(sourceFile)
	if err != nil {
		return fmt.Errorf("failed to copy file: %v", err)
	}

	a.savedPdfPath = savePath
	a.hasUnsavedChanges = false

	return nil
}

// GetDefaultSavePath returns the default save path based on initial directory or file
func (a *App) GetDefaultSavePath() string {
	if a.initialDir == "" {
		return ""
	}

	// Clean the path and get absolute path
	absPath, err := filepath.Abs(a.initialDir)
	if err != nil {
		absPath = a.initialDir
	}

	// Check if it's a file or directory
	if info, err := os.Stat(absPath); err == nil {
		if info.IsDir() {
			// Directory: use directory name + .pdf
			dirName := filepath.Base(absPath)
			return filepath.Join(filepath.Dir(absPath), dirName+".pdf")
		} else {
			// File: use file name (without extension) + .pdf
			baseName := strings.TrimSuffix(filepath.Base(absPath), filepath.Ext(absPath))
			return filepath.Join(filepath.Dir(absPath), baseName+".pdf")
		}
	}

	// Fallback: treat as directory
	dirName := filepath.Base(absPath)
	return filepath.Join(filepath.Dir(absPath), dirName+".pdf")
}

// HasUnsavedChanges returns whether there are unsaved changes
func (a *App) HasUnsavedChanges() bool {
	return a.hasUnsavedChanges
}

// MarkAsModified marks the current PDF as having unsaved changes
func (a *App) MarkAsModified() {
	a.hasUnsavedChanges = true
}
