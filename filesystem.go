package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

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
