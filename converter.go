package main

import (
	"crypto/md5"
	"fmt"
	"io"
	"os"
	"path/filepath"
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

	fmt.Printf("Cache key input: %s\n", hashInput)
	fmt.Printf("Output path: %s\n", outputPath)

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
				fmt.Printf("Using cached PDF: %s\n", outputPath)
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
		fmt.Printf("Warning: could not set modification time: %v\n", err)
	}

	return outputPath, nil
}

// convertExcelToPDF converts Excel file to PDF using Excel application
func (c *OfficeConverter) convertExcelToPDF(srcPath, outputPath string, selectedSheets []string) error {
	fmt.Printf("Converting Excel file: %s\n", srcPath)
	fmt.Printf("Selected sheets: %v\n", selectedSheets)

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

	// Set properties
	oleutil.PutProperty(excel, "DisplayAlerts", false)
	oleutil.PutProperty(excel, "Visible", false)

	// Get workbooks collection
	workbooks := oleutil.MustGetProperty(excel, "Workbooks").ToIDispatch()
	defer workbooks.Release()

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

	// Handle sheet selection
	if len(selectedSheets) > 0 {
		fmt.Printf("Processing sheet selection for %d sheets\n", len(selectedSheets))

		// Get worksheets collection
		worksheets := oleutil.MustGetProperty(wb, "Worksheets").ToIDispatch()
		defer worksheets.Release()

		// First, hide all sheets except the selected ones
		totalSheets := int(oleutil.MustGetProperty(worksheets, "Count").Val)
		fmt.Printf("Total sheets in workbook: %d\n", totalSheets)

		// Get all sheet names first
		var allSheetNames []string
		for i := 1; i <= totalSheets; i++ {
			sheet := oleutil.MustGetProperty(worksheets, "Item", i).ToIDispatch()
			sheetName := oleutil.MustGetProperty(sheet, "Name").ToString()
			allSheetNames = append(allSheetNames, sheetName)
			sheet.Release()
		}
		fmt.Printf("All sheet names: %v\n", allSheetNames)

		// Hide non-selected sheets
		for _, sheetName := range allSheetNames {
			isSelected := false
			for _, selectedName := range selectedSheets {
				if sheetName == selectedName {
					isSelected = true
					break
				}
			}

			sheet := oleutil.MustGetProperty(worksheets, "Item", sheetName).ToIDispatch()
			if !isSelected {
				fmt.Printf("Hiding sheet: %s\n", sheetName)
				// Hide the sheet (xlSheetHidden = 0)
				oleutil.PutProperty(sheet, "Visible", 0)
			} else {
				fmt.Printf("Keeping sheet visible: %s\n", sheetName)
				// Ensure selected sheets are visible (xlSheetVisible = -1)
				oleutil.PutProperty(sheet, "Visible", -1)
			}
			sheet.Release()
		}

		// Select the first selected sheet to make it active
		if len(selectedSheets) > 0 {
			fmt.Printf("Activating first selected sheet: %s\n", selectedSheets[0])
			firstSheet := oleutil.MustGetProperty(worksheets, "Item", selectedSheets[0]).ToIDispatch()
			oleutil.CallMethod(firstSheet, "Select")
			firstSheet.Release()
		}

		fmt.Printf("Exporting workbook with selected sheets only\n")
		// Export entire workbook (now only visible sheets will be exported)
		_, err = oleutil.CallMethod(wb, "ExportAsFixedFormat", 0, outputPath, 0)
		if err != nil {
			return fmt.Errorf("failed to export Excel to PDF: %v", err)
		}
	} else {
		fmt.Printf("No specific sheets selected, exporting entire workbook\n")
		// Export entire workbook
		_, err = oleutil.CallMethod(wb, "ExportAsFixedFormat", 0, outputPath, 0)
		if err != nil {
			return fmt.Errorf("failed to export Excel to PDF: %v", err)
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

	// Set properties
	oleutil.PutProperty(word, "DisplayAlerts", false)
	oleutil.PutProperty(word, "Visible", false)

	// Get documents collection
	documents := oleutil.MustGetProperty(word, "Documents").ToIDispatch()
	defer documents.Release()

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
