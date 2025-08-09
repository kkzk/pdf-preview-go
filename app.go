package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// App struct
type App struct {
	ctx        context.Context
	converter  *OfficeConverter
	initialDir string // Initial directory to open
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

// ConversionStatus represents the status of a conversion operation
type ConversionStatus struct {
	Status       string `json:"status"`       // "running", "completed", "error"
	CurrentFile  string `json:"currentFile"`  // Currently processing file
	Progress     int    `json:"progress"`     // Progress percentage
	OutputPath   string `json:"outputPath"`   // Final output PDF path
	ErrorMessage string `json:"errorMessage"` // Error message if status is "error"
}

// NewApp creates a new App application struct
func NewApp(initialDir string) *App {
	// Create cache directory
	cacheDir := filepath.Join(os.TempDir(), "pdf-preview-go-cache")

	return &App{
		converter:  NewOfficeConverter(cacheDir),
		initialDir: initialDir,
	}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	// Set initial window title
	if a.initialDir != "" {
		a.SetWindowTitle(a.initialDir)
	}

	// Clean up old cache files (older than 2 days)
	go func() {
		if err := a.converter.CleanupCache(48 * time.Hour); err != nil {
			fmt.Printf("Warning: cache cleanup failed: %v\n", err)
		}
	}()
}

// Greet returns a greeting for the given name
func (a *App) Greet(name string) string {
	return fmt.Sprintf("Hello %s, It's show time!", name)
}

// GetInitialDirectory returns the initial directory set via command line
func (a *App) GetInitialDirectory() string {
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

// GetDirectoryTree returns a recursive file tree structure for a given directory
func (a *App) GetDirectoryTree(dirPath string) ([]FileInfo, error) {
	if dirPath == "" {
		return nil, fmt.Errorf("directory path is empty")
	}

	return a.buildDirectoryTree(dirPath, 0, 3) // Max depth of 3 levels
}

// buildDirectoryTree recursively builds directory tree
func (a *App) buildDirectoryTree(dirPath string, currentDepth, maxDepth int) ([]FileInfo, error) {
	if currentDepth >= maxDepth {
		return nil, nil
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

		// For directories, always include them
		// For files, only include Office files and PDFs
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

		// If it's a directory, recursively get its contents
		if entry.IsDir() {
			children, err := a.buildDirectoryTree(fileInfo.Path, currentDepth+1, maxDepth)
			if err == nil {
				fileInfo.Children = children
			}
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
	return GetExcelSheetsInfo(filePath)
}

// ConvertToPDF converts selected files to PDF and merges them
func (a *App) ConvertToPDF(filePaths []string, sheetSelections map[string][]string) (string, error) {
	if len(filePaths) == 0 {
		return "", fmt.Errorf("no files selected for conversion")
	}

	var convertedPDFs []string
	var errors []string

	// Convert each file to PDF
	for i, filePath := range filePaths {
		// Emit progress event
		runtime.EventsEmit(a.ctx, "conversion:progress", ConversionStatus{
			Status:      "running",
			CurrentFile: filepath.Base(filePath),
			Progress:    int((float64(i) / float64(len(filePaths))) * 100),
		})

		outputPath, err := a.converter.ConvertToPDF(filePath, sheetSelections, false)
		if err != nil {
			errors = append(errors, fmt.Sprintf("%s: %v", filepath.Base(filePath), err))
			continue
		}

		convertedPDFs = append(convertedPDFs, outputPath)
	}

	if len(convertedPDFs) == 0 {
		return "", fmt.Errorf("no files were successfully converted: %v", errors)
	}

	// If only one file, return it directly
	if len(convertedPDFs) == 1 {
		runtime.EventsEmit(a.ctx, "conversion:progress", ConversionStatus{
			Status:     "completed",
			Progress:   100,
			OutputPath: convertedPDFs[0],
		})
		return convertedPDFs[0], nil
	}

	// For multiple files, we would merge them here
	// For now, return the first file as placeholder
	// TODO: Implement PDF merging
	runtime.EventsEmit(a.ctx, "conversion:progress", ConversionStatus{
		Status:       "completed",
		Progress:     100,
		OutputPath:   convertedPDFs[0],
		ErrorMessage: "Multiple PDF merging not yet implemented - returning first file",
	})

	return convertedPDFs[0], nil
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
