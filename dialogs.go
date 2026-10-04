package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/wailsapp/wails/v2/pkg/runtime"
	"golang.org/x/sys/windows"
)

// ErrUserCancelled is returned when the user cancels a dialog.
// The frontend detects cancellation by this message, so keep it unchanged.
var ErrUserCancelled = errors.New("user_cancelled")

// GetInitialDirectory returns the initial directory set via command line
func (a *App) GetInitialDirectory() string {
	if a.initialDir != "" {
		// Add to directory history when accessed
		if err := a.AddDirectoryToHistory(a.initialDir); err != nil {
			log.Printf("Warning: failed to add directory to history: %v", err)
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
			log.Printf("Warning: failed to add directory to history: %v", err)
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
	a.mu.Lock()
	hasPdf := a.currentPdfPath != ""
	a.mu.Unlock()
	if !hasPdf {
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
		return ErrUserCancelled
	}

	// Save the PDF
	err = a.SavePdfAs(filePath)
	if err != nil {
		return fmt.Errorf("failed to save PDF: %v", err)
	}

	// Successfully saved
	return nil
}

// CloseChoice is the user's answer to the unsaved changes dialog
type CloseChoice int

const (
	CloseSave    CloseChoice = iota // Save, then close
	CloseDiscard                    // Close without saving
	CloseCancel                     // Do not close
)

// MessageBox return values
const (
	idYes = 6
	idNo  = 7
)

// confirmUnsavedChanges asks whether to save the PDF before closing.
// Wails' MessageDialog on Windows only offers Yes/No for questions, so the
// Win32 MessageBox is used directly to provide a Cancel button.
func confirmUnsavedChanges() CloseChoice {
	title, _ := windows.UTF16PtrFromString("未保存の変更があります")
	message, _ := windows.UTF16PtrFromString(
		"PDFファイルに未保存の変更があります。保存しますか？\n\n" +
			"はい: 保存して終了\nいいえ: 保存せずに終了\nキャンセル: 終了しない")

	ret, _ := windows.MessageBox(ownWindow(), message, title,
		windows.MB_YESNOCANCEL|windows.MB_ICONWARNING|windows.MB_SETFOREGROUND)

	switch ret {
	case idYes:
		return CloseSave
	case idNo:
		return CloseDiscard
	default: // Cancel, closed dialog or error
		return CloseCancel
	}
}

// ownWindow returns the foreground window if it belongs to this process,
// to be used as the owner of a message box, or 0 otherwise
func ownWindow() windows.HWND {
	hwnd := windows.GetForegroundWindow()
	var pid uint32
	if _, err := windows.GetWindowThreadProcessId(hwnd, &pid); err != nil || pid != uint32(os.Getpid()) {
		return 0
	}
	return hwnd
}
