package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"pdf-preview-go/internal/convert"
	"pdf-preview-go/internal/pdfserver"
	"pdf-preview-go/internal/watch"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// mergedFilePrefix is the file name prefix of merged PDFs in the cache directory
const mergedFilePrefix = "merged_"

// ConversionStatus represents the status of a conversion operation
type ConversionStatus struct {
	Status       string `json:"status"`       // "running", "completed", "error"
	CurrentFile  string `json:"currentFile"`  // Currently processing file
	Progress     int    `json:"progress"`     // Progress percentage
	OutputPath   string `json:"outputPath"`   // Final output PDF path
	ErrorMessage string `json:"errorMessage"` // Error message if status is "error"
}

// conversionRetryDelays are the waits before retrying a failed conversion
// (see convert.Retry)
var conversionRetryDelays = []time.Duration{2 * time.Second, 4 * time.Second}

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
	modTimes := watch.StatModTimes(filePaths)

	var convertedPDFs []string
	var failures []string

	// Convert each file to PDF
	for i, filePath := range filePaths {
		fileName := filepath.Base(filePath)

		// Emit progress event
		runtime.EventsEmit(a.ctx, "conversion:progress", ConversionStatus{
			Status:      "running",
			CurrentFile: fileName,
			Progress:    int((float64(i) / float64(len(filePaths))) * 100),
		})

		outputPath, err := convert.Retry(func() (string, error) {
			return a.converter.ConvertToPDF(filePath, sheetSelections[filePath])
		}, conversionRetryDelays, a.stopCh, func(attempt int, err error) {
			a.emitWarning(fmt.Sprintf("%s の変換に失敗したため再試行します (%d/%d): %v",
				fileName, attempt, len(conversionRetryDelays), err))
		})
		if err != nil {
			failures = append(failures, fmt.Sprintf("%s: %v", fileName, err))
			continue
		}

		convertedPDFs = append(convertedPDFs, outputPath)
	}

	if len(convertedPDFs) == 0 {
		return "", fmt.Errorf("no files were successfully converted: %s", strings.Join(failures, "; "))
	}
	if len(failures) > 0 {
		// The PDF is created without these files; let the user know
		a.emitWarning("一部のファイルを変換できなかったため、PDFに含まれていません: " + strings.Join(failures, "; "))
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
		mergedFileName := fmt.Sprintf("%s%s.pdf", mergedFilePrefix, time.Now().Format("20060102_150405.000"))
		pdfPath = filepath.Join(filepath.Dir(convertedPDFs[0]), mergedFileName)

		if err := convert.MergePDFs(convertedPDFs, pdfPath); err != nil {
			return "", fmt.Errorf("failed to merge PDFs: %v", err)
		}
	}

	// Convert file path to URL with cache buster
	pdfURL := pdfserver.URL(filepath.Base(pdfPath), time.Now().UnixNano())

	runtime.EventsEmit(a.ctx, "conversion:progress", ConversionStatus{
		Status:     "completed",
		Progress:   100,
		OutputPath: pdfURL,
	})

	a.mu.Lock()
	previousPdfPath := a.currentPdfPath
	// Save converted files and sheet selections for auto-update
	a.lastConvertedFiles = filePaths
	a.lastConvertedSheets = sheetSelections
	// Record current PDF path and mark as modified
	a.currentPdfPath = pdfPath
	a.hasUnsavedChanges = true
	a.mu.Unlock()
	// Record file modification times for change detection
	a.watcher.Track(filePaths, modTimes)

	// Remove the previous merged PDF, which is no longer used. This may fail
	// while it is being read (e.g. saved); it is then removed by CleanupCache.
	if previousPdfPath != pdfPath && strings.HasPrefix(filepath.Base(previousPdfPath), mergedFilePrefix) {
		os.Remove(previousPdfPath)
	}

	// Start watching the directory of the first file
	if err := a.watcher.WatchDir(filepath.Dir(filePaths[0])); err != nil {
		log.Printf("Warning: failed to watch directory: %v", err)
	}

	return pdfURL, nil
}

// emitWarning logs a warning and shows it in the frontend log
func (a *App) emitWarning(message string) {
	log.Printf("Warning: %s", message)
	runtime.EventsEmit(a.ctx, "conversion:warning", map[string]interface{}{
		"message": message,
	})
}

// onFileChanged is called by the watcher when a converted file has been
// modified, and regenerates the PDF with the same files and sheet selections
func (a *App) onFileChanged(changedFile string) {
	runtime.EventsEmit(a.ctx, "file-changed", map[string]interface{}{
		"file": changedFile,
	})

	a.mu.Lock()
	convertedFiles := a.lastConvertedFiles
	convertedSheets := a.lastConvertedSheets
	a.mu.Unlock()

	// Skip files that no longer exist
	validFiles := []string{}
	for _, filePath := range convertedFiles {
		if _, err := os.Stat(filePath); err == nil {
			validFiles = append(validFiles, filePath)
		}
	}
	if len(validFiles) == 0 {
		return
	}

	if _, err := a.ConvertToPDF(validFiles, convertedSheets); err != nil {
		runtime.EventsEmit(a.ctx, "conversion:error", map[string]interface{}{
			"message": "Auto-update failed: " + err.Error(),
		})
	}
}
