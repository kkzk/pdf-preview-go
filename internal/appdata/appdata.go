package appdata

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// DirectorySessionCache represents cached state for a specific directory
type DirectorySessionCache struct {
	DirectoryPath   string              `json:"directoryPath"`   // Directory path
	SelectedFiles   []string            `json:"selectedFiles"`   // List of selected file paths
	ExpandedFolders []string            `json:"expandedFolders"` // List of expanded folder paths
	CurrentFile     string              `json:"currentFile"`     // Currently selected file
	SheetSelections map[string][]string `json:"sheetSelections"` // File path -> selected sheets
	ExpiryTime      time.Time           `json:"expiryTime"`      // When cache expires
}

// SaveDirectorySessionCache saves the complete session state for a directory
func SaveDirectorySessionCache(dirPath string, selectedFiles []string, expandedFolders []string, currentFile string, sheetSelections map[string][]string) error {
	if dirPath == "" {
		return fmt.Errorf("directory path cannot be empty")
	}

	absPath := absolutePath(dirPath)
	cacheFilePath := sessionFilePath(absPath)

	// Create cache directory if it doesn't exist
	os.MkdirAll(filepath.Dir(cacheFilePath), 0755)

	cache := DirectorySessionCache{
		DirectoryPath:   absPath,
		SelectedFiles:   selectedFiles,
		ExpandedFolders: expandedFolders,
		CurrentFile:     currentFile,
		SheetSelections: sheetSelections,
		ExpiryTime:      time.Now().AddDate(0, 3, 0), // Expire after 3 months
	}

	data, err := json.MarshalIndent(cache, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal session cache: %v", err)
	}

	if err := os.WriteFile(cacheFilePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write session cache: %v", err)
	}

	return nil
}

// LoadDirectorySessionCache loads the complete session state for a directory,
// or nil if there is none
func LoadDirectorySessionCache(dirPath string) (*DirectorySessionCache, error) {
	if dirPath == "" {
		return nil, fmt.Errorf("directory path cannot be empty")
	}

	cacheFilePath := sessionFilePath(absolutePath(dirPath))

	data, err := os.ReadFile(cacheFilePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // No cache found
		}
		return nil, fmt.Errorf("failed to read session cache: %v", err)
	}

	var cache DirectorySessionCache
	if err := json.Unmarshal(data, &cache); err != nil {
		return nil, fmt.Errorf("failed to parse session cache: %v", err)
	}

	// Check if cache is expired
	if time.Now().After(cache.ExpiryTime) {
		// Remove expired cache file
		os.Remove(cacheFilePath)
		return nil, nil
	}

	// Validate selected files still exist (they may have been modified)
	validSelectedFiles := []string{}
	for _, filePath := range cache.SelectedFiles {
		if _, err := os.Stat(filePath); err == nil {
			validSelectedFiles = append(validSelectedFiles, filePath)
		}
	}
	cache.SelectedFiles = validSelectedFiles

	// Validate expanded folders still exist
	validExpandedFolders := []string{}
	for _, folderPath := range cache.ExpandedFolders {
		if info, err := os.Stat(folderPath); err == nil && info.IsDir() {
			validExpandedFolders = append(validExpandedFolders, folderPath)
		}
	}
	cache.ExpandedFolders = validExpandedFolders

	// Validate current file
	if cache.CurrentFile != "" {
		if _, err := os.Stat(cache.CurrentFile); err != nil {
			cache.CurrentFile = ""
		}
	}

	// Validate sheet selections
	cache.SheetSelections = existingFileSelections(cache.SheetSelections)

	return &cache, nil
}

// CleanupDirectorySessionCache removes session files that have not been saved
// for maxAge. Expired sessions that remain are removed when they are loaded.
func CleanupDirectorySessionCache(maxAge time.Duration) error {
	cacheDir := dataDir()
	entries, err := os.ReadDir(cacheDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // Cache directory doesn't exist yet
		}
		return err
	}

	cutoff := time.Now().Add(-maxAge)
	cleaned := 0

	for _, entry := range entries {
		// Only process session cache files
		if entry.IsDir() || !strings.HasPrefix(entry.Name(), "session_") || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}

		info, err := entry.Info()
		if err != nil || !info.ModTime().Before(cutoff) {
			continue
		}
		if err := os.Remove(filepath.Join(cacheDir, entry.Name())); err == nil {
			cleaned++
		}
	}

	if cleaned > 0 {
		log.Printf("Cleaned up %d old session cache files", cleaned)
	}

	return nil
}

// absolutePath returns the absolute form of path, or path itself if it cannot
// be determined
func absolutePath(path string) string {
	if absPath, err := filepath.Abs(path); err == nil {
		return absPath
	}
	return path
}

// existingFileSelections returns the sheet selections of files that still exist.
// Selections are kept even if a file has been modified: sheet names that no
// longer exist are dropped by the frontend when it loads the sheet list.
func existingFileSelections(selections map[string][]string) map[string][]string {
	valid := make(map[string][]string)
	for path, sheets := range selections {
		if _, err := os.Stat(path); err == nil {
			valid[path] = sheets
		}
	}
	return valid
}
