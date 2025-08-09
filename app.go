package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// App struct
type App struct {
	ctx context.Context
}

// FileInfo represents file information
type FileInfo struct {
	Name     string     `json:"name"`
	Path     string     `json:"path"`
	Size     int64      `json:"size"`
	IsDir    bool       `json:"isDir"`
	ModTime  string     `json:"modTime"`
	Children []FileInfo `json:"children,omitempty"`
}

// ExcelSheetInfo represents Excel sheet information
type ExcelSheetInfo struct {
	Name    string `json:"name"`
	Visible bool   `json:"visible"`
	Index   int    `json:"index"`
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

// Greet returns a greeting for the given name
func (a *App) Greet(name string) string {
	return fmt.Sprintf("Hello %s, It's show time!", name)
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

// GetDirectoryContents returns file tree structure for a given directory
func (a *App) GetDirectoryContents(dirPath string) ([]FileInfo, error) {
	if dirPath == "" {
		return nil, fmt.Errorf("directory path is empty")
	}

	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, err
	}

	var files []FileInfo
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			continue
		}

		// Office files and PDFs only
		ext := strings.ToLower(filepath.Ext(entry.Name()))
		if !entry.IsDir() && !isOfficeFile(ext) {
			continue
		}

		fileInfo := FileInfo{
			Name:    entry.Name(),
			Path:    filepath.Join(dirPath, entry.Name()),
			Size:    info.Size(),
			IsDir:   entry.IsDir(),
			ModTime: info.ModTime().Format("2006-01-02 15:04:05"),
		}

		files = append(files, fileInfo)
	}

	return files, nil
}

// isOfficeFile checks if the file extension is an Office file
func isOfficeFile(ext string) bool {
	officeExts := []string{".xlsx", ".xls", ".xlsm", ".docx", ".doc", ".pdf"}
	for _, officeExt := range officeExts {
		if ext == officeExt {
			return true
		}
	}
	return false
}

// GetExcelSheets returns sheet information for an Excel file
func (a *App) GetExcelSheets(filePath string) ([]ExcelSheetInfo, error) {
	// TODO: Implement Excel sheet reading using Go library
	// For now, return mock data
	return []ExcelSheetInfo{
		{Name: "Sheet1", Visible: true, Index: 0},
		{Name: "Sheet2", Visible: true, Index: 1},
		{Name: "Hidden", Visible: false, Index: 2},
	}, nil
}

// ConvertToPDF converts selected files to PDF
func (a *App) ConvertToPDF(filePaths []string, sheetSelections map[string][]string) (string, error) {
	// TODO: Implement PDF conversion logic
	// For now, return a placeholder
	return "conversion_result.pdf", fmt.Errorf("not implemented yet")
}

// GetFileInfo returns basic file information
func (a *App) GetFileInfo(filePath string) (map[string]interface{}, error) {
	if filePath == "" {
		return nil, fmt.Errorf("file path is empty")
	}

	info, err := os.Stat(filePath)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"name":    info.Name(),
		"size":    info.Size(),
		"path":    filePath,
		"dir":     filepath.Dir(filePath),
		"modTime": info.ModTime(),
	}, nil
}
