package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/sys/windows"
)

// GetDirectoryContents returns the folders and supported files directly in a
// directory. The tree is loaded one level at a time as folders are expanded,
// so that large or cloud-backed (e.g. Box Drive) folders open quickly.
func (a *App) GetDirectoryContents(dirPath string) ([]FileInfo, error) {
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

		// Office files and PDFs only, excluding Office lock files (~$Book.xlsx)
		ext := strings.ToLower(filepath.Ext(entry.Name()))
		if !entry.IsDir() && (!isOfficeFile(ext) || strings.HasPrefix(entry.Name(), "~$")) {
			continue
		}

		files = append(files, newFileInfo(filepath.Join(dirPath, entry.Name()), info))
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

// GetFilesInfo returns information about the given files, skipping files
// that no longer exist. Used to restore selections without loading the
// folders that contain them.
func (a *App) GetFilesInfo(filePaths []string) []FileInfo {
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

// OpenFile opens a file with its associated application (Excel, Word, a PDF
// viewer, ...). Only existing files of the types shown in the tree are opened.
func (a *App) OpenFile(filePath string) error {
	info, err := os.Stat(filePath)
	if err != nil {
		return fmt.Errorf("file not found: %v", err)
	}
	if info.IsDir() || !isOfficeFile(strings.ToLower(filepath.Ext(filePath))) {
		return fmt.Errorf("%w: %s", ErrUnsupportedFileType, filepath.Ext(filePath))
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
