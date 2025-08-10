package main

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

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
