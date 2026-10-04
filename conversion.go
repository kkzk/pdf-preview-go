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

// ConvertToPDF converts selected files to PDF and merges them.
// Conversions are serialized: a call waits until any running conversion
// (including auto-regeneration) has finished.
func (a *App) ConvertToPDF(filePaths []string, sheetSelections map[string][]string) (string, error) {
	if len(filePaths) == 0 {
		return "", fmt.Errorf("no files selected for conversion")
	}

	a.convertMu.Lock()
	defer a.convertMu.Unlock()

	// Record modification times before converting, so that a save made
	// during the conversion is detected as a change afterwards
	modTimes := statModTimes(filePaths)

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

	// If only one file, use it directly; otherwise merge them using pdfcpu
	pdfPath := convertedPDFs[0]
	if len(convertedPDFs) > 1 {
		runtime.EventsEmit(a.ctx, "conversion:progress", ConversionStatus{
			Status:      "running",
			CurrentFile: "PDFファイルを結合中...",
			Progress:    90,
		})

		// Generate merged PDF filename with timestamp
		// (all PDFs are in the same cache directory)
		mergedFileName := fmt.Sprintf("merged_%s.pdf", time.Now().Format("20060102_150405"))
		pdfPath = filepath.Join(filepath.Dir(convertedPDFs[0]), mergedFileName)

		if err := MergePDFs(convertedPDFs, pdfPath); err != nil {
			return "", fmt.Errorf("failed to merge PDFs: %v", err)
		}
	}

	// Convert file path to URL with cache buster
	pdfURL := buildPdfURL(filepath.Base(pdfPath), time.Now().UnixNano())

	runtime.EventsEmit(a.ctx, "conversion:progress", ConversionStatus{
		Status:     "completed",
		Progress:   100,
		OutputPath: pdfURL,
	})

	a.mu.Lock()
	// Save converted files and sheet selections for auto-update
	a.lastConvertedFiles = filePaths
	a.lastConvertedSheets = sheetSelections
	// Record current PDF path and mark as modified
	a.currentPdfPath = pdfPath
	a.hasUnsavedChanges = true
	// Record file modification times for change detection
	a.fileModTimes = modTimes
	a.mu.Unlock()

	// Start watching the directory of the first file
	if err := a.StartWatchingDirectory(filepath.Dir(filePaths[0])); err != nil {
		fmt.Printf("Warning: failed to watch directory: %v\n", err)
	}

	return pdfURL, nil
}
