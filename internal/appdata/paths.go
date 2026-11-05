// Package appdata stores the data the app keeps between runs (per-directory
// sessions) and knows the location of the PDF cache.
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

// legacyHistoryFile is the directory history written by earlier versions,
// which is no longer used
const legacyHistoryFile = "directory_history.json"

// PDFCacheDir returns the directory for converted PDFs. They can be
// regenerated at any time, so the temp directory is used.
func PDFCacheDir() string {
	return filepath.Join(os.TempDir(), appName+"-cache")
}

// dataDir returns the directory for data that should be kept, such as the
// per-directory sessions (%AppData%\pdf-preview-go).
func dataDir() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		return PDFCacheDir()
	}
	return filepath.Join(dir, appName)
}

// sessionFilePath returns the path of the session file for an absolute
// directory path
func sessionFilePath(absDirPath string) string {
	return filepath.Join(dataDir(), fmt.Sprintf("session_%s.json", directoryHash(absDirPath)))
}

// directoryHash returns a hash identifying an absolute directory path
func directoryHash(absDirPath string) string {
	hash := md5.Sum([]byte(strings.ToLower(absDirPath)))
	return hex.EncodeToString(hash[:])
}

// MigrateLegacyData moves the sessions, which were stored in the PDF cache
// directory by earlier versions, to dataDir. Files already present in dataDir
// are kept. Files that are no longer used (sheet selections, whose contents
// are also stored in sessions, and the directory history) are removed.
func MigrateLegacyData() {
	oldDir := PDFCacheDir()
	newDir := dataDir()
	os.Remove(filepath.Join(newDir, legacyHistoryFile))
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
		case name == legacyHistoryFile || strings.HasPrefix(name, "sheet_selections_"):
			os.Remove(oldPath)
		case strings.HasPrefix(name, "session_"):
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
