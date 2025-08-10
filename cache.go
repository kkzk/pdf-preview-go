package main

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// SaveSheetSelections saves sheet selections to cache
func (a *App) SaveSheetSelections(filePath string, sheetSelections map[string][]string) error {
	if filePath == "" {
		return fmt.Errorf("directory path is empty")
	}

	// Create cache directory hash
	dirHash := a.createDirectoryHash(filePath)
	cacheFilePath := filepath.Join(os.TempDir(), "pdf-preview-go-cache", fmt.Sprintf("sheet_selections_%s.json", dirHash))

	// Ensure cache directory exists
	if err := os.MkdirAll(filepath.Dir(cacheFilePath), 0755); err != nil {
		return fmt.Errorf("failed to create cache directory: %v", err)
	}

	// Create file hashes for validation
	fileHashes := make(map[string]string)
	for path := range sheetSelections {
		hash, err := a.calculateFileHash(path)
		if err != nil {
			// If we can't hash the file, skip it (file might not exist)
			continue
		}
		fileHashes[path] = hash
	}

	// Create cache structure
	cache := SheetSelectionCache{
		DirectoryHash: dirHash,
		LastUpdated:   time.Now(),
		Selections:    sheetSelections,
		FileHashes:    fileHashes,
		ExpiryTime:    time.Now().AddDate(0, 1, 0), // Expire after 1 month
	}

	// Write to JSON file
	data, err := json.MarshalIndent(cache, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal cache data: %v", err)
	}

	if err := os.WriteFile(cacheFilePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write cache file: %v", err)
	}

	return nil
}

// LoadSheetSelections loads sheet selections from cache
func (a *App) LoadSheetSelections(dirPath string) (map[string][]string, error) {
	if dirPath == "" {
		return make(map[string][]string), nil
	}

	// Create cache directory hash
	dirHash := a.createDirectoryHash(dirPath)
	cacheFilePath := filepath.Join(os.TempDir(), "pdf-preview-go-cache", fmt.Sprintf("sheet_selections_%s.json", dirHash))

	// Check if cache file exists
	if _, err := os.Stat(cacheFilePath); os.IsNotExist(err) {
		return make(map[string][]string), nil // No cache, return empty map
	}

	// Read cache file
	data, err := os.ReadFile(cacheFilePath)
	if err != nil {
		return make(map[string][]string), nil // Can't read cache, return empty map
	}

	// Parse JSON
	var cache SheetSelectionCache
	if err := json.Unmarshal(data, &cache); err != nil {
		return make(map[string][]string), nil // Invalid cache, return empty map
	}

	// Check if cache is expired
	if time.Now().After(cache.ExpiryTime) {
		// Remove expired cache file
		os.Remove(cacheFilePath)
		return make(map[string][]string), nil
	}

	// Validate file hashes to ensure files haven't changed
	validSelections := make(map[string][]string)
	for path, sheets := range cache.Selections {
		// Check if file still exists
		if _, err := os.Stat(path); os.IsNotExist(err) {
			continue // File no longer exists, skip
		}

		// Check if file hash matches
		if expectedHash, exists := cache.FileHashes[path]; exists {
			currentHash, err := a.calculateFileHash(path)
			if err == nil && currentHash == expectedHash {
				// File hasn't changed, keep the selections
				validSelections[path] = sheets
			}
		}
	}

	return validSelections, nil
}

// CleanupSheetSelectionsCache removes old sheet selection cache files
func (a *App) CleanupSheetSelectionsCache(maxAge time.Duration) error {
	cacheDir := filepath.Join(os.TempDir(), "pdf-preview-go-cache")
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

		// Only process sheet selection cache files
		if !strings.HasPrefix(entry.Name(), "sheet_selections_") || !strings.HasSuffix(entry.Name(), ".json") {
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
				var cache SheetSelectionCache
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
		fmt.Printf("Cleaned up %d old sheet selection cache files\n", cleaned)
	}

	return nil
}

// createDirectoryHash creates a hash for directory path
func (a *App) createDirectoryHash(dirPath string) string {
	// Normalize path
	absPath, err := filepath.Abs(dirPath)
	if err != nil {
		absPath = dirPath
	}

	// Create MD5 hash
	hash := md5.Sum([]byte(strings.ToLower(absPath)))
	return hex.EncodeToString(hash[:])
}

// calculateFileHash calculates hash of file for change detection
func (a *App) calculateFileHash(filePath string) (string, error) {
	// For Excel files, we use a combination of file size and modification time
	// This is faster than reading entire file content
	info, err := os.Stat(filePath)
	if err != nil {
		return "", err
	}

	// Combine size and mod time for a unique identifier
	hashInput := fmt.Sprintf("%d_%d", info.Size(), info.ModTime().Unix())
	hash := md5.Sum([]byte(hashInput))
	return hex.EncodeToString(hash[:]), nil
}

// SaveSheetSelectionsForDirectory saves sheet selections for current directory
func (a *App) SaveSheetSelectionsForDirectory(sheetSelections map[string][]string) error {
	if a.initialDir == "" {
		return fmt.Errorf("no working directory set")
	}

	return a.SaveSheetSelections(a.initialDir, sheetSelections)
}

// LoadSheetSelectionsForDirectory loads sheet selections for current directory
func (a *App) LoadSheetSelectionsForDirectory() (map[string][]string, error) {
	if a.initialDir == "" {
		return make(map[string][]string), nil
	}

	return a.LoadSheetSelections(a.initialDir)
}
