package main

import (
	"pdf-preview-go/internal/appdata"
	"pdf-preview-go/internal/contextmenu"
	"pdf-preview-go/internal/convert"
	"pdf-preview-go/internal/files"
)

// Methods bound to the frontend that only delegate to internal packages.

// GetDirectoryContents returns the folders and supported files directly in a
// directory (one level only)
func (a *App) GetDirectoryContents(dirPath string) ([]files.FileInfo, error) {
	return files.ListDirectory(dirPath)
}

// GetFilesInfo returns information about the given files, skipping files
// that no longer exist
func (a *App) GetFilesInfo(filePaths []string) []files.FileInfo {
	return files.Stat(filePaths)
}

// OpenFile opens a supported file with its associated application
func (a *App) OpenFile(filePath string) error {
	return files.Open(filePath)
}

// GetExcelSheets returns sheet information for an Excel file
func (a *App) GetExcelSheets(filePath string) ([]convert.ExcelSheetInfo, error) {
	return convert.GetExcelSheetsInfo(filePath)
}

// SaveDirectorySessionCache saves the complete session state for a directory
func (a *App) SaveDirectorySessionCache(dirPath string, selectedFiles []string, expandedFolders []string, currentFile string, sheetSelections map[string][]string) error {
	return appdata.SaveDirectorySessionCache(dirPath, selectedFiles, expandedFolders, currentFile, sheetSelections)
}

// LoadDirectorySessionCache loads the complete session state for a directory,
// or nil if there is none
func (a *App) LoadDirectorySessionCache(dirPath string) (*appdata.DirectorySessionCache, error) {
	return appdata.LoadDirectorySessionCache(dirPath)
}

// SetAutoUpdateEnabled enables or disables automatic PDF updates
func (a *App) SetAutoUpdateEnabled(enabled bool) {
	a.watcher.SetEnabled(enabled)
}

// GetAutoUpdateEnabled returns current auto-update status
func (a *App) GetAutoUpdateEnabled() bool {
	return a.watcher.Enabled()
}

// IsContextMenuEnabled reports whether the Explorer context menu is registered
// for the current user
func (a *App) IsContextMenuEnabled() bool {
	return contextmenu.Enabled()
}

// SetContextMenuEnabled turns the Explorer context menu on or off for the
// current user
func (a *App) SetContextMenuEnabled(enabled bool) error {
	return contextmenu.SetEnabled(enabled)
}
