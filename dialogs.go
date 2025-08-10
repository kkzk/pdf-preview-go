package main

import (
	"fmt"
	"path/filepath"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// GetInitialDirectory returns the initial directory set via command line
func (a *App) GetInitialDirectory() string {
	if a.initialDir != "" {
		// Add to directory history when accessed
		if err := a.AddDirectoryToHistory(a.initialDir); err != nil {
			fmt.Printf("Warning: failed to add directory to history: %v\n", err)
		}
	}
	return a.initialDir
}

// OpenFileDialog opens a file dialog to select PDF files
func (a *App) OpenFileDialog() (string, error) {
	file, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "PDFファイルを選択",
		Filters: []runtime.FileFilter{
			{
				DisplayName: "PDFファイル (*.pdf)",
				Pattern:     "*.pdf",
			},
		},
	})
	return file, err
}

// OpenDirectoryDialog opens a directory selection dialog
func (a *App) OpenDirectoryDialog() (string, error) {
	dir, err := runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "フォルダを選択",
	})
	return dir, err
}

// ChangeWorkingDirectory changes the current working directory and emits event
func (a *App) ChangeWorkingDirectory() (string, error) {
	dir, err := runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "作業フォルダを選択",
	})
	if err != nil {
		return "", err
	}
	if dir != "" {
		// Update initial directory
		a.initialDir = dir

		// Add to directory history
		if err := a.AddDirectoryToHistory(dir); err != nil {
			fmt.Printf("Warning: failed to add directory to history: %v\n", err)
		}

		// Update window title with new directory
		runtime.WindowSetTitle(a.ctx, fmt.Sprintf("PDF Preview - %s", filepath.Base(dir)))

		// Emit event to notify frontend
		runtime.EventsEmit(a.ctx, "directory-changed", dir)
		return dir, nil
	}
	return "", nil
}

// SetWindowTitle updates the window title with current directory
func (a *App) SetWindowTitle(dirPath string) {
	if dirPath != "" {
		title := fmt.Sprintf("PDF Preview - %s", filepath.Base(dirPath))
		runtime.WindowSetTitle(a.ctx, title)
	} else {
		runtime.WindowSetTitle(a.ctx, "PDF Preview")
	}
}

// ShowSaveDialog shows the save dialog and saves the PDF
func (a *App) ShowSaveDialog() error {
	if a.currentPdfPath == "" {
		return fmt.Errorf("no PDF to save")
	}

	// Get default save path
	defaultPath := a.GetDefaultSavePath()
	if defaultPath == "" {
		return fmt.Errorf("unable to determine default save path")
	}

	// Show save file dialog
	filePath, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		DefaultDirectory:     filepath.Dir(defaultPath),
		DefaultFilename:      filepath.Base(defaultPath),
		Title:                "PDFファイルを保存",
		ShowHiddenFiles:      false,
		CanCreateDirectories: true,
		Filters: []runtime.FileFilter{
			{
				DisplayName: "PDFファイル (*.pdf)",
				Pattern:     "*.pdf",
			},
		},
	})

	if err != nil {
		return fmt.Errorf("failed to show save dialog: %v", err)
	}

	if filePath == "" {
		// User cancelled - return a special error to indicate cancellation
		return fmt.Errorf("user_cancelled")
	}

	// Save the PDF
	err = a.SavePdfAs(filePath)
	if err != nil {
		return fmt.Errorf("failed to save PDF: %v", err)
	}

	// Successfully saved
	return nil
}
