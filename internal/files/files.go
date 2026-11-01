// Package files lists and opens the files the app works with.
package files

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"pdf-preview-go/internal/convert"

	"golang.org/x/sys/windows"
)

// FileInfo represents file information
type FileInfo struct {
	Name     string     `json:"name"`
	Path     string     `json:"path"`
	Size     int64      `json:"size"`
	IsDir    bool       `json:"isDir"`
	ModTime  string     `json:"modTime"`
	Children []FileInfo `json:"children,omitempty"`
}

// ListDirectory returns the folders and supported files directly in a
// directory. The tree is loaded one level at a time as folders are expanded,
// so that large or cloud-backed (e.g. Box Drive) folders open quickly.
func ListDirectory(dirPath string) ([]FileInfo, error) {
	if dirPath == "" {
		return nil, fmt.Errorf("directory path is empty")
	}

	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, err
	}

	files := []FileInfo{} // Empty (not null) for an empty folder
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			continue
		}

		if !entry.IsDir() && !IsSupported(entry.Name()) {
			continue
		}

		files = append(files, newFileInfo(filepath.Join(dirPath, entry.Name()), info))
	}

	return files, nil
}

// IsSupported reports whether a file is shown and can be converted: Office
// files and PDFs, excluding Office lock files (~$Book.xlsx)
func IsSupported(path string) bool {
	name := filepath.Base(path)
	return convert.IsSupportedExt(strings.ToLower(filepath.Ext(name))) && !strings.HasPrefix(name, "~$")
}

// Stat returns information about the given files, skipping files
// that no longer exist. Used to restore selections without loading the
// folders that contain them.
func Stat(filePaths []string) []FileInfo {
	files := []FileInfo{}
	for _, filePath := range filePaths {
		info, err := os.Stat(filePath)
		if err != nil || info.IsDir() {
			continue
		}
		files = append(files, newFileInfo(filePath, info))
	}
	return files
}

// newFileInfo creates a FileInfo for a path
func newFileInfo(path string, info os.FileInfo) FileInfo {
	return FileInfo{
		Name:    info.Name(),
		Path:    path,
		Size:    info.Size(),
		IsDir:   info.IsDir(),
		ModTime: info.ModTime().Format("2006-01-02 15:04:05"),
	}
}

// Details returns basic file information
func Details(filePath string) (map[string]interface{}, error) {
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

// Open opens a file with its associated application (Excel, Word, a PDF
// viewer, ...). Only existing files of the types shown in the tree are opened.
func Open(filePath string) error {
	info, err := os.Stat(filePath)
	if err != nil {
		return fmt.Errorf("file not found: %v", err)
	}
	if info.IsDir() || !convert.IsSupportedExt(strings.ToLower(filepath.Ext(filePath))) {
		return fmt.Errorf("%w: %s", convert.ErrUnsupportedFileType, filepath.Ext(filePath))
	}

	verb, err := windows.UTF16PtrFromString("open")
	if err != nil {
		return err
	}
	file, err := windows.UTF16PtrFromString(filePath)
	if err != nil {
		return err
	}
	dir, err := windows.UTF16PtrFromString(filepath.Dir(filePath))
	if err != nil {
		return err
	}
	if err := windows.ShellExecute(0, verb, file, nil, dir, windows.SW_SHOWNORMAL); err != nil {
		return fmt.Errorf("failed to open %s: %v", filepath.Base(filePath), err)
	}
	return nil
}
