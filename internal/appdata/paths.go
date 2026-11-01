// Package appdata stores the data the app keeps between runs: the directory
// history and per-directory sessions, and the location of the PDF cache.
package appdata

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
)

const appName = "pdf-preview-go"

// PDFCacheDir returns the directory for converted PDFs. They can be
// regenerated at any time, so the temp directory is used.
func PDFCacheDir() string {
	return filepath.Join(os.TempDir(), appName+"-cache")
}

// dataDir returns the directory for data that should be kept, such as the
// directory history and per-directory sessions (%AppData%\pdf-preview-go).
func dataDir() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		return PDFCacheDir()
	}
	return filepath.Join(dir, appName)
}

// historyFilePath returns the path of the directory history file
func historyFilePath() string {
	return filepath.Join(dataDir(), "directory_history.json")
}

// sessionFilePath returns the path of the session file for a directory
func sessionFilePath(dirPath string) string {
	return filepath.Join(dataDir(), fmt.Sprintf("session_%s.json", directoryHash(dirPath)))
}

// directoryHash returns a hash identifying a directory path
func directoryHash(dirPath string) string {
	absPath, err := filepath.Abs(dirPath)
	if err != nil {
		absPath = dirPath
	}

	hash := md5.Sum([]byte(strings.ToLower(absPath)))
	return hex.EncodeToString(hash[:])
}

// MigrateLegacyData moves the directory history and sessions, which were
// stored in the PDF cache directory by earlier versions, to dataDir. Files
// already present in dataDir are kept. Legacy sheet selection files, whose
// contents are also stored in sessions, are removed.
func MigrateLegacyData() {
	oldDir := PDFCacheDir()
	newDir := dataDir()
	if oldDir == newDir {
		return
	}

	entries, err := os.ReadDir(oldDir)
	if err != nil {
		return // Nothing to migrate
	}

	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".json") {
			continue
		}
		oldPath := filepath.Join(oldDir, name)

		switch {
		case strings.HasPrefix(name, "sheet_selections_"):
			os.Remove(oldPath)
		case name == "directory_history.json" || strings.HasPrefix(name, "session_"):
			if err := os.MkdirAll(newDir, 0755); err != nil {
				log.Printf("Warning: failed to create data directory: %v", err)
				return
			}
			newPath := filepath.Join(newDir, name)
			if _, err := os.Stat(newPath); err == nil {
				os.Remove(oldPath) // Already migrated
				continue
			}
			if err := os.Rename(oldPath, newPath); err != nil {
				log.Printf("Warning: failed to migrate %s: %v", name, err)
			}
		}
	}
}
