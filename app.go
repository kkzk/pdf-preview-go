package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// App struct
type App struct {
	ctx                 context.Context
	converter           *OfficeConverter
	initialDir          string // Initial directory to open
	httpServer          *http.Server
	httpPort            int
	watcher             *fsnotify.Watcher
	watchedDir          string
	lastConvertedFiles  []string
	lastConvertedSheets map[string][]string
	autoUpdateEnabled   bool
	fileModTimes        map[string]time.Time // Track file modification times
	pollingTicker       *time.Ticker
	currentPdfPath      string // Current PDF file path in temp
	savedPdfPath        string // Last saved PDF path
	hasUnsavedChanges   bool   // Whether there are unsaved changes
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
	os.MkdirAll(cacheDir, 0755)

	app := &App{
		converter:           NewOfficeConverter(cacheDir),
		initialDir:          initialDir,
		httpPort:            0, // Will be set when server starts
		watchedDir:          "",
		lastConvertedFiles:  []string{},
		lastConvertedSheets: make(map[string][]string),
		autoUpdateEnabled:   true,
		fileModTimes:        make(map[string]time.Time),
		currentPdfPath:      "",
		savedPdfPath:        "",
		hasUnsavedChanges:   false,
	}

	return app
}

// Startup is called when the app starts. The context passed
// is the app's context. Additional initialization can be done here.
func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx

	// Start HTTP server for serving PDF files
	a.startHTTPServer()

	// Initialize file watcher
	a.initFileWatcher()
}

// Shutdown is called when the app is closing
func (a *App) Shutdown(ctx context.Context) {
	if a.pollingTicker != nil {
		a.pollingTicker.Stop()
	}
	if a.watcher != nil {
		a.watcher.Close()
	}
	if a.httpServer != nil {
		a.httpServer.Close()
	}
	// Note: OfficeConverter doesn't have a Close method
	// COM objects are automatically cleaned up
}

// startHTTPServer starts a local HTTP server to serve PDF files
func (a *App) startHTTPServer() {
	// Find available port
	for port := 8080; port < 8090; port++ {
		mux := http.NewServeMux()

		// Serve PDF files from cache directory
		cacheDir := filepath.Join(os.TempDir(), "pdf-preview-go-cache")
		mux.Handle("/pdf/", http.StripPrefix("/pdf/", http.FileServer(http.Dir(cacheDir))))

		// Add CORS headers for WebView compatibility
		corsHandler := func(h http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Access-Control-Allow-Origin", "*")
				w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
				w.Header().Set("Access-Control-Allow-Headers", "*")
				if r.Method == "OPTIONS" {
					w.WriteHeader(http.StatusOK)
					return
				}
				h.ServeHTTP(w, r)
			})
		}

		a.httpServer = &http.Server{
			Addr:    ":" + strconv.Itoa(port),
			Handler: corsHandler(mux),
		}

		go func() {
			if err := a.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				// Port might be in use, try next one
			}
		}()

		// Test if server started successfully
		time.Sleep(100 * time.Millisecond)
		resp, err := http.Get("http://localhost:" + strconv.Itoa(port) + "/pdf/")
		if err == nil {
			resp.Body.Close()
			a.httpPort = port
			return
		}
	}
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

		// Force regeneration if sheet selections exist for this file
		forceRegeneration := false
		if sheets, exists := sheetSelections[filePath]; exists && len(sheets) > 0 {
			forceRegeneration = true
		}

		outputPath, err := a.converter.ConvertToPDF(filePath, sheetSelections, forceRegeneration)
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
		// Convert file path to HTTP URL with cache buster
		fileName := filepath.Base(convertedPDFs[0])
		timestamp := time.Now().UnixNano()
		pdfURL := fmt.Sprintf("http://localhost:%d/pdf/%s?v=%d", a.httpPort, fileName, timestamp)

		runtime.EventsEmit(a.ctx, "conversion:progress", ConversionStatus{
			Status:     "completed",
			Progress:   100,
			OutputPath: pdfURL,
		})

		// Save converted files and sheet selections for auto-update
		a.lastConvertedFiles = filePaths
		a.lastConvertedSheets = sheetSelections

		// Record current PDF path and mark as modified
		a.currentPdfPath = convertedPDFs[0]
		a.hasUnsavedChanges = true

		// Record file modification times
		a.recordFileModTimes(filePaths)

		// Start watching the directory of the file
		dirToWatch := filepath.Dir(filePaths[0])
		a.StartWatchingDirectory(dirToWatch)

		// Start polling for file changes (as backup for fsnotify)
		a.startPolling()

		return pdfURL, nil
	}

	// For multiple files, merge them using pdfcpu
	runtime.EventsEmit(a.ctx, "conversion:progress", ConversionStatus{
		Status:      "running",
		CurrentFile: "PDFファイルを結合中...",
		Progress:    90,
	})

	// Generate merged PDF filename with timestamp
	timestamp := time.Now().Format("20060102_150405")
	mergedFileName := fmt.Sprintf("merged_%s.pdf", timestamp)

	// Get cache directory from converter
	cacheDir := filepath.Dir(convertedPDFs[0]) // All PDFs are in the same cache directory
	mergedPath := filepath.Join(cacheDir, mergedFileName)

	// Merge PDFs using pdfcpu
	err := MergePDFs(convertedPDFs, mergedPath)
	if err != nil {
		return "", fmt.Errorf("failed to merge PDFs: %v", err)
	}

	// Convert merged file path to HTTP URL with cache buster
	timestampCacheBuster := time.Now().UnixNano()
	pdfURL := fmt.Sprintf("http://localhost:%d/pdf/%s?v=%d", a.httpPort, mergedFileName, timestampCacheBuster)

	runtime.EventsEmit(a.ctx, "conversion:progress", ConversionStatus{
		Status:     "completed",
		Progress:   100,
		OutputPath: pdfURL,
	})

	// Save converted files and sheet selections for auto-update
	a.lastConvertedFiles = filePaths
	a.lastConvertedSheets = sheetSelections

	// Record current PDF path and mark as modified
	a.currentPdfPath = mergedPath
	a.hasUnsavedChanges = true

	// Record file modification times
	a.recordFileModTimes(filePaths)

	// Start watching the directory of the first file
	if len(filePaths) > 0 {
		dirToWatch := filepath.Dir(filePaths[0])
		a.StartWatchingDirectory(dirToWatch)
	}

	// Start polling for file changes (as backup for fsnotify)
	a.startPolling()

	return pdfURL, nil
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
