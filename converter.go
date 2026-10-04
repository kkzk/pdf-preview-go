package main

import (
	"crypto/md5"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/go-ole/go-ole"
	"github.com/go-ole/go-ole/oleutil"
	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/tealeg/xlsx/v3"
)

// OfficeConverter handles conversion of Office documents to PDF
type OfficeConverter struct {
	cacheDir string
}

// NewOfficeConverter creates a new converter instance
func NewOfficeConverter(cacheDir string) *OfficeConverter {
	return &OfficeConverter{
		cacheDir: cacheDir,
	}
}

// ConvertResult contains the result of a conversion operation
type ConvertResult struct {
	OutputPath string
	Error      error
}

// ConvertToPDF converts an Office file to PDF using Office applications
func (c *OfficeConverter) ConvertToPDF(srcPath string, selectedSheets map[string][]string, force bool) (string, error) {
	// Generate cache file name based on file hash and sheet selection
	hashInput := srcPath
	if sheets, exists := selectedSheets[srcPath]; exists && len(sheets) > 0 {
		hashInput += "|" + strings.Join(sheets, ",")
	}
	hash := md5.Sum([]byte(hashInput))
	outputFileName := fmt.Sprintf("%x.pdf", hash)
	outputPath := filepath.Join(c.cacheDir, outputFileName)

	// Create cache directory if it doesn't exist
	if err := os.MkdirAll(c.cacheDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create cache directory: %v", err)
	}

	// Check if source file exists
	srcInfo, err := os.Stat(srcPath)
	if err != nil {
		return "", fmt.Errorf("source file not found: %v", err)
	}

	// Check if output already exists and is up to date (unless force is true)
	if !force {
		if outputInfo, err := os.Stat(outputPath); err == nil {
			if srcInfo.ModTime().Equal(outputInfo.ModTime()) {
				return outputPath, nil // File is up to date
			}
		}
	}

	ext := strings.ToLower(filepath.Ext(srcPath))

	// Handle PDF files (just copy)
	if ext == ".pdf" {
		if err := copyFile(srcPath, outputPath); err != nil {
			return "", err
		}
		return outputPath, nil
	}

	// COM requires all calls to be made from the thread that initialized it,
	// so pin this goroutine to its OS thread until COM is uninitialized
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	// Initialize COM
	if err := ole.CoInitializeEx(0, ole.COINIT_MULTITHREADED); err != nil {
		return "", fmt.Errorf("failed to initialize COM: %v", err)
	}
	defer ole.CoUninitialize()

	// Convert based on file type
	switch ext {
	case ".xlsx", ".xls", ".xlsm":
		err = c.convertExcelToPDF(srcPath, outputPath, selectedSheets[srcPath])
	case ".docx", ".doc":
		err = c.convertWordToPDF(srcPath, outputPath)
	default:
		return "", fmt.Errorf("unsupported file type: %s", ext)
	}

	if err != nil {
		return "", err
	}

	// Set the same modification time as source file
	if err := os.Chtimes(outputPath, srcInfo.ModTime(), srcInfo.ModTime()); err != nil {
		// Log warning but don't fail
	}

	return outputPath, nil
}

// convertExcelToPDF converts Excel file to PDF using Excel application
func (c *OfficeConverter) convertExcelToPDF(srcPath, outputPath string, selectedSheets []string) error {
	// Create Excel application
	unknown, err := oleutil.CreateObject("Excel.Application")
	if err != nil {
		return fmt.Errorf("failed to create Excel application: %v", err)
	}
	defer unknown.Release()

	excel, err := unknown.QueryInterface(ole.IID_IDispatch)
	if err != nil {
		return fmt.Errorf("failed to get Excel IDispatch: %v", err)
	}
	defer excel.Release()

	// Get workbooks collection (cleanup quits Excel if we started it)
	workbooks, cleanup, err := prepareOfficeApp(excel, "Workbooks")
	if err != nil {
		return fmt.Errorf("failed to prepare Excel application: %v", err)
	}
	defer cleanup()

	// Open workbook
	workbook, err := oleutil.CallMethod(workbooks, "Open", srcPath, false, true)
	if err != nil {
		return fmt.Errorf("failed to open Excel file: %v", err)
	}
	defer func() {
		oleutil.PutProperty(workbook.ToIDispatch(), "Saved", true)
		oleutil.CallMethod(workbook.ToIDispatch(), "Close")
		workbook.Clear()
	}()

	wb := workbook.ToIDispatch()

	// Handle sheet selection: leave only the selected sheets visible
	if len(selectedSheets) > 0 {
		if err := showOnlySelectedSheets(wb, selectedSheets); err != nil {
			return err
		}
		fmt.Printf("Exporting workbook with selected sheets only\n")
	} else {
		fmt.Printf("No specific sheets selected, exporting entire workbook\n")
	}

	// Export entire workbook (only visible sheets are exported)
	_, err = oleutil.CallMethod(wb, "ExportAsFixedFormat", 0, outputPath, 0)
	if err != nil {
		return fmt.Errorf("failed to export Excel to PDF: %v", err)
	}

	return nil
}

// Excel XlSheetVisibility values
const (
	xlSheetVisible = -1
	xlSheetHidden  = 0
)

// showOnlySelectedSheets makes the selected sheets visible and hides the rest.
// Selected sheets are shown first, because Excel refuses to hide the last
// visible sheet. Selected names missing from the workbook are skipped, but it
// is an error if none of them exist.
func showOnlySelectedSheets(wb *ole.IDispatch, selectedSheets []string) error {
	sheetsVar, err := oleutil.GetProperty(wb, "Sheets")
	if err != nil {
		return fmt.Errorf("failed to get sheets: %v", err)
	}
	sheets := sheetsVar.ToIDispatch()
	defer sheets.Release()

	count := collectionCount(sheets)
	if count < 0 {
		return fmt.Errorf("failed to get sheet count")
	}

	selected := make(map[string]bool, len(selectedSheets))
	for _, name := range selectedSheets {
		selected[name] = true
	}

	// Collect sheets and their names
	type namedSheet struct {
		name  string
		sheet *ole.IDispatch
	}
	var all []namedSheet
	defer func() {
		for _, s := range all {
			s.sheet.Release()
		}
	}()
	for i := 1; i <= count; i++ {
		sheetVar, err := oleutil.GetProperty(sheets, "Item", i)
		if err != nil {
			return fmt.Errorf("failed to get sheet %d: %v", i, err)
		}
		sheet := sheetVar.ToIDispatch()
		nameVar, err := oleutil.GetProperty(sheet, "Name")
		if err != nil {
			sheet.Release()
			return fmt.Errorf("failed to get name of sheet %d: %v", i, err)
		}
		all = append(all, namedSheet{name: nameVar.ToString(), sheet: sheet})
	}

	// Show selected sheets first
	found := 0
	for _, s := range all {
		if !selected[s.name] {
			continue
		}
		fmt.Printf("Keeping sheet visible: %s\n", s.name)
		if _, err := oleutil.PutProperty(s.sheet, "Visible", xlSheetVisible); err != nil {
			return fmt.Errorf("failed to show sheet %q: %v", s.name, err)
		}
		found++
	}
	if found == 0 {
		return fmt.Errorf("selected sheets not found in workbook: %s", strings.Join(selectedSheets, ", "))
	}
	if found < len(selected) {
		fmt.Printf("Warning: some selected sheets were not found: %s\n", strings.Join(selectedSheets, ", "))
	}

	// Then hide the others
	for _, s := range all {
		if selected[s.name] {
			continue
		}
		fmt.Printf("Hiding sheet: %s\n", s.name)
		if _, err := oleutil.PutProperty(s.sheet, "Visible", xlSheetHidden); err != nil {
			return fmt.Errorf("failed to hide sheet %q: %v", s.name, err)
		}
	}

	return nil
}

// convertWordToPDF converts Word document to PDF using Word application
func (c *OfficeConverter) convertWordToPDF(srcPath, outputPath string) error {
	// Create Word application
	unknown, err := oleutil.CreateObject("Word.Application")
	if err != nil {
		return fmt.Errorf("failed to create Word application: %v", err)
	}
	defer unknown.Release()

	word, err := unknown.QueryInterface(ole.IID_IDispatch)
	if err != nil {
		return fmt.Errorf("failed to get Word IDispatch: %v", err)
	}
	defer word.Release()

	// Get documents collection (cleanup quits Word if we started it)
	documents, cleanup, err := prepareOfficeApp(word, "Documents", wdDoNotSaveChanges)
	if err != nil {
		return fmt.Errorf("failed to prepare Word application: %v", err)
	}
	defer cleanup()

	// Open document
	document, err := oleutil.CallMethod(documents, "Open", srcPath, false, true, false, "")
	if err != nil {
		return fmt.Errorf("failed to open Word document: %v", err)
	}
	defer func() {
		doc := document.ToIDispatch()
		oleutil.PutProperty(doc, "Saved", true)
		oleutil.CallMethod(doc, "Close")
		document.Clear()
	}()

	// Export as PDF
	doc := document.ToIDispatch()
	_, err = oleutil.CallMethod(doc, "ExportAsFixedFormat", outputPath, 17) // 17 = wdExportFormatPDF
	if err != nil {
		return fmt.Errorf("failed to export Word to PDF: %v", err)
	}

	return nil
}

// wdDoNotSaveChanges is the SaveChanges argument for Word's Application.Quit
const wdDoNotSaveChanges = 0

// prepareOfficeApp configures an Office application for background conversion
// and returns its document collection (e.g. "Workbooks", "Documents") along
// with a cleanup function. Defer the cleanup before opening the document so
// that it runs after the document has been closed.
//
// CreateObject may return an instance the user is already working with
// (notably Word). Such an instance is left visible and running, and only its
// DisplayAlerts setting is restored. An instance started for this conversion
// is quit, but only when no documents remain open in it.
func prepareOfficeApp(app *ole.IDispatch, collectionName string, quitArgs ...interface{}) (*ole.IDispatch, func(), error) {
	collectionVar, err := oleutil.GetProperty(app, collectionName)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get %s: %v", collectionName, err)
	}
	collection := collectionVar.ToIDispatch()

	// If the count is unknown (-1), treat it as a user instance to be safe
	userInstance := isAppVisible(app) || collectionCount(collection) != 0

	origAlerts, alertsErr := oleutil.GetProperty(app, "DisplayAlerts")
	oleutil.PutProperty(app, "DisplayAlerts", false)
	if !userInstance {
		oleutil.PutProperty(app, "Visible", false)
	}

	cleanup := func() {
		if userInstance {
			if alertsErr == nil {
				oleutil.PutProperty(app, "DisplayAlerts", origAlerts.Value())
			}
		} else if collectionCount(collection) == 0 {
			if _, err := oleutil.CallMethod(app, "Quit", quitArgs...); err != nil {
				fmt.Printf("Warning: failed to quit Office application: %v\n", err)
			}
		}
		collection.Release()
	}

	return collection, cleanup, nil
}

// isAppVisible reports whether the Office application window is visible
func isAppVisible(app *ole.IDispatch) bool {
	v, err := oleutil.GetProperty(app, "Visible")
	if err != nil {
		return false
	}
	visible, _ := v.Value().(bool)
	return visible
}

// collectionCount returns the Count of a COM collection, or -1 on error
func collectionCount(collection *ole.IDispatch) int {
	v, err := oleutil.GetProperty(collection, "Count")
	if err != nil {
		return -1
	}
	return int(v.Val)
}

// GetExcelSheetsInfo returns information about sheets in an Excel file
func GetExcelSheetsInfo(filePath string) ([]ExcelSheetInfo, error) {
	file, err := xlsx.OpenFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open Excel file: %v", err)
	}

	var sheets []ExcelSheetInfo
	for i, sheet := range file.Sheets {
		sheets = append(sheets, ExcelSheetInfo{
			Name:    sheet.Name,
			Visible: !sheet.Hidden, // xlsx library uses Hidden property
			Index:   i,
		})
	}

	return sheets, nil
}

// MergePDFs combines multiple PDF files into one using pdfcpu library
func MergePDFs(inputPaths []string, outputPath string) error {
	if len(inputPaths) == 0 {
		return fmt.Errorf("no input PDFs provided")
	}

	// If only one file, just copy it
	if len(inputPaths) == 1 {
		return copyFile(inputPaths[0], outputPath)
	}

	// Validate all input files exist
	for _, inputPath := range inputPaths {
		if _, err := os.Stat(inputPath); os.IsNotExist(err) {
			return fmt.Errorf("input file does not exist: %s", inputPath)
		}
	}

	// Use pdfcpu to merge PDFs
	err := api.MergeCreateFile(inputPaths, outputPath, false, nil)
	if err != nil {
		return fmt.Errorf("failed to merge PDFs: %v", err)
	}

	return nil
}

// copyFile copies a file from src to dst
func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	if err != nil {
		return err
	}

	// Copy file info
	sourceInfo, err := os.Stat(src)
	if err != nil {
		return err
	}

	return os.Chmod(dst, sourceInfo.Mode())
}

// CleanupCache removes old cache files (older than specified duration)
func (c *OfficeConverter) CleanupCache(maxAge time.Duration) error {
	entries, err := os.ReadDir(c.cacheDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // Cache directory doesn't exist yet
		}
		return err
	}

	cutoff := time.Now().Add(-maxAge)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		if info.ModTime().Before(cutoff) {
			filePath := filepath.Join(c.cacheDir, entry.Name())
			if err := os.Remove(filePath); err != nil {
				fmt.Printf("Warning: could not remove cache file %s: %v\n", filePath, err)
			}
		}
	}

	return nil
}
