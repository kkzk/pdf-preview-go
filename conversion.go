package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// mergedFilePrefix is the file name prefix of merged PDFs in the cache directory
const mergedFilePrefix = "merged_"

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

		outputPath, err := retryConversion(func() (string, error) {
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
	previousPdfPath := a.currentPdfPath
	// Save converted files and sheet selections for auto-update
	a.lastConvertedFiles = filePaths
	a.lastConvertedSheets = sheetSelections
	// Record current PDF path and mark as modified
	a.currentPdfPath = pdfPath
	a.hasUnsavedChanges = true
	// Record file modification times for change detection
	a.fileModTimes = modTimes
	a.mu.Unlock()

	// Remove the previous merged PDF, which is no longer used. This may fail
	// while it is being read (e.g. saved); it is then removed by CleanupCache.
	if previousPdfPath != pdfPath && strings.HasPrefix(filepath.Base(previousPdfPath), mergedFilePrefix) {
		os.Remove(previousPdfPath)
	}

	// Start watching the directory of the first file
	if err := a.StartWatchingDirectory(filepath.Dir(filePaths[0])); err != nil {
		log.Printf("Warning: failed to watch directory: %v", err)
	}

	return pdfURL, nil
}

// conversionRetryDelays are the waits before retrying a failed conversion.
// A file that has just been saved may be briefly locked or missing, e.g.
// while Office saves it or a sync client (Box Drive, OneDrive) uploads it.
var conversionRetryDelays = []time.Duration{2 * time.Second, 4 * time.Second}

// retryConversion calls convert, retrying after each of delays while it
// fails with a retryable error. onRetry is called before each retry with the
// attempt number (1-based) and the error. Waiting stops when stop is closed.
func retryConversion(convert func() (string, error), delays []time.Duration, stop <-chan struct{}, onRetry func(attempt int, err error)) (string, error) {
	outputPath, err := convert()
	for i, delay := range delays {
		if err == nil || !isRetryableConversionError(err) {
			break
		}
		onRetry(i+1, err)
		select {
		case <-stop:
			return outputPath, err
		case <-time.After(delay):
		}
		outputPath, err = convert()
	}
	return outputPath, err
}

// isRetryableConversionError reports whether a conversion error may go away
// by retrying (e.g. the file is locked), as opposed to a permanent one
func isRetryableConversionError(err error) bool {
	return !errors.Is(err, ErrUnsupportedFileType) && !errors.Is(err, ErrSheetsNotFound)
}

// emitWarning logs a warning and shows it in the frontend log
func (a *App) emitWarning(message string) {
	log.Printf("Warning: %s", message)
	runtime.EventsEmit(a.ctx, "conversion:warning", map[string]interface{}{
		"message": message,
	})
}
