package main

import (
	"context"
	"net/http"
	"time"

	"github.com/fsnotify/fsnotify"
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

// SheetSelectionCache represents cached sheet selections for a directory
type SheetSelectionCache struct {
	DirectoryHash string              `json:"directoryHash"` // MD5 hash of directory path
	LastUpdated   time.Time           `json:"lastUpdated"`   // When cache was last updated
	Selections    map[string][]string `json:"selections"`    // File path -> selected sheets
	FileHashes    map[string]string   `json:"fileHashes"`    // File path -> file content hash
	ExpiryTime    time.Time           `json:"expiryTime"`    // When cache expires
}

// DirectoryHistory represents history of directory usage
type DirectoryHistory struct {
	Path        string    `json:"path"`        // Directory path
	DisplayName string    `json:"displayName"` // Display name (usually folder name)
	LastUsed    time.Time `json:"lastUsed"`    // When directory was last accessed
	UsageCount  int       `json:"usageCount"`  // How many times this directory was used
}

// DirectorySessionCache represents cached state for a specific directory
type DirectorySessionCache struct {
	DirectoryPath   string              `json:"directoryPath"`   // Directory path
	DirectoryHash   string              `json:"directoryHash"`   // MD5 hash of directory path
	LastUpdated     time.Time           `json:"lastUpdated"`     // When cache was last updated
	SelectedFiles   []string            `json:"selectedFiles"`   // List of selected file paths
	ExpandedFolders []string            `json:"expandedFolders"` // List of expanded folder paths
	CurrentFile     string              `json:"currentFile"`     // Currently selected file
	SheetSelections map[string][]string `json:"sheetSelections"` // File path -> selected sheets
	FileHashes      map[string]string   `json:"fileHashes"`      // File path -> file content hash for validation
	ExpiryTime      time.Time           `json:"expiryTime"`      // When cache expires
}
