package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// GetDirectoryHistory returns the list of recently used directories
func (a *App) GetDirectoryHistory() ([]DirectoryHistory, error) {
	cacheFilePath := historyFilePath()

	// Check if cache file exists
	if _, err := os.Stat(cacheFilePath); os.IsNotExist(err) {
		return []DirectoryHistory{}, nil
	}

	// Read cache file
	data, err := os.ReadFile(cacheFilePath)
	if err != nil {
		return []DirectoryHistory{}, nil
	}

	// Parse JSON
	var history []DirectoryHistory
	if err := json.Unmarshal(data, &history); err != nil {
		return []DirectoryHistory{}, nil
	}

	// Filter out directories that no longer exist and sort by last used
	validHistory := []DirectoryHistory{}
	for _, dir := range history {
		if _, err := os.Stat(dir.Path); err == nil {
			validHistory = append(validHistory, dir)
		}
	}

	// Sort by last used time (most recent first)
	sort.Slice(validHistory, func(i, j int) bool {
		return validHistory[i].LastUsed.After(validHistory[j].LastUsed)
	})

	// Keep only the most recent 20 directories
	if len(validHistory) > 20 {
		validHistory = validHistory[:20]
	}

	return validHistory, nil
}

// AddDirectoryToHistory adds a directory to the usage history
func (a *App) AddDirectoryToHistory(dirPath string) error {
	if dirPath == "" {
		return fmt.Errorf("directory path cannot be empty")
	}

	// Normalize path
	absPath, err := filepath.Abs(dirPath)
	if err != nil {
		absPath = dirPath
	}

	// Get current history
	history, err := a.GetDirectoryHistory()
	if err != nil {
		history = []DirectoryHistory{}
	}

	// Check if directory already exists in history
	var existingDir *DirectoryHistory
	for i := range history {
		if history[i].Path == absPath {
			existingDir = &history[i]
			break
		}
	}

	// Get display name (folder name)
	displayName := filepath.Base(absPath)
	if displayName == "." || displayName == "" {
		displayName = absPath
	}

	if existingDir != nil {
		// Update existing entry
		existingDir.LastUsed = time.Now()
		existingDir.UsageCount++
		existingDir.DisplayName = displayName
	} else {
		// Add new entry
		newEntry := DirectoryHistory{
			Path:        absPath,
			DisplayName: displayName,
			LastUsed:    time.Now(),
			UsageCount:  1,
		}
		history = append([]DirectoryHistory{newEntry}, history...)
	}

	// Keep only the most recent 20 directories
	if len(history) > 20 {
		history = history[:20]
	}

	// Save updated history
	cacheFilePath := historyFilePath()
	os.MkdirAll(filepath.Dir(cacheFilePath), 0755)

	data, err := json.MarshalIndent(history, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal directory history: %v", err)
	}

	if err := os.WriteFile(cacheFilePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write directory history: %v", err)
	}

	return nil
}

// SaveDirectorySessionCache saves the complete session state for a directory
func (a *App) SaveDirectorySessionCache(dirPath string, selectedFiles []string, expandedFolders []string, currentFile string, sheetSelections map[string][]string) error {
	if dirPath == "" {
		return fmt.Errorf("directory path cannot be empty")
	}

	// Normalize path
	absPath, err := filepath.Abs(dirPath)
	if err != nil {
		absPath = dirPath
	}

	dirHash := directoryHash(absPath)
	cacheFilePath := sessionFilePath(absPath)

	// Create cache directory if it doesn't exist
	os.MkdirAll(filepath.Dir(cacheFilePath), 0755)

	// Create cache structure
	cache := DirectorySessionCache{
		DirectoryPath:   absPath,
		DirectoryHash:   dirHash,
		LastUpdated:     time.Now(),
		SelectedFiles:   selectedFiles,
		ExpandedFolders: expandedFolders,
		CurrentFile:     currentFile,
		SheetSelections: sheetSelections,
		ExpiryTime:      time.Now().AddDate(0, 3, 0), // Expire after 3 months
	}

	// Write to JSON file
	data, err := json.MarshalIndent(cache, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal session cache: %v", err)
	}

	if err := os.WriteFile(cacheFilePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write session cache: %v", err)
	}

	return nil
}

// LoadDirectorySessionCache loads the complete session state for a directory
func (a *App) LoadDirectorySessionCache(dirPath string) (*DirectorySessionCache, error) {
	if dirPath == "" {
		return nil, fmt.Errorf("directory path cannot be empty")
	}

	// Normalize path
	absPath, err := filepath.Abs(dirPath)
	if err != nil {
		absPath = dirPath
	}

	cacheFilePath := sessionFilePath(absPath)

	// Check if cache file exists
	if _, err := os.Stat(cacheFilePath); os.IsNotExist(err) {
		return nil, nil // No cache found
	}

	// Read cache file
	data, err := os.ReadFile(cacheFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read session cache: %v", err)
	}

	// Parse JSON
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

// CleanupDirectorySessionCache removes old session cache files
func (a *App) CleanupDirectorySessionCache(maxAge time.Duration) error {
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
		if entry.IsDir() {
			continue
		}

		// Only process session cache files
		if !strings.HasPrefix(entry.Name(), "session_") || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}

		filePath := filepath.Join(cacheDir, entry.Name())

		// Check file modification time
		info, err := entry.Info()
		if err != nil {
			continue
		}

		// Also check cache content for expiry
		shouldRemove := info.ModTime().Before(cutoff)

		if !shouldRemove {
			// Check cache file content for expiry
			if data, err := os.ReadFile(filePath); err == nil {
				var cache DirectorySessionCache
				if json.Unmarshal(data, &cache) == nil {
					if time.Now().After(cache.ExpiryTime) {
						shouldRemove = true
					}
				}
			}
		}

		if shouldRemove {
			if err := os.Remove(filePath); err == nil {
				cleaned++
			}
		}
	}

	if cleaned > 0 {
		log.Printf("Cleaned up %d old session cache files", cleaned)
	}

	return nil
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
