package main

import (
	"archive/zip"
	"encoding/xml"
	"fmt"
	"path"
	"path/filepath"
	"strings"

	"github.com/go-ole/go-ole"
	"github.com/go-ole/go-ole/oleutil"
)

// GetExcelSheetsInfo returns information about sheets in an Excel file.
// .xlsx/.xlsm files are read directly; .xls files are read via Excel.
func GetExcelSheetsInfo(filePath string) ([]ExcelSheetInfo, error) {
	switch strings.ToLower(filepath.Ext(filePath)) {
	case ".xls":
		return readSheetsWithExcel(filePath)
	default:
		return readXlsxSheets(filePath)
	}
}

// readXlsxSheets reads the sheet list from the workbook part of an OOXML
// workbook, without loading the sheets themselves
func readXlsxSheets(filePath string) ([]ExcelSheetInfo, error) {
	zr, err := zip.OpenReader(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open Excel file: %v", err)
	}
	defer zr.Close()

	workbookPath := findWorkbookPart(&zr.Reader)
	var workbook struct {
		Sheets []struct {
			Name  string `xml:"name,attr"`
			State string `xml:"state,attr"` // "visible" (default), "hidden" or "veryHidden"
		} `xml:"sheets>sheet"`
	}
	if err := decodeZipXML(&zr.Reader, workbookPath, &workbook); err != nil {
		return nil, fmt.Errorf("failed to read workbook: %v", err)
	}

	sheets := make([]ExcelSheetInfo, 0, len(workbook.Sheets))
	for i, sheet := range workbook.Sheets {
		sheets = append(sheets, ExcelSheetInfo{
			Name:    sheet.Name,
			Visible: sheet.State == "" || sheet.State == "visible",
			Index:   i,
		})
	}
	return sheets, nil
}

// findWorkbookPart returns the path of the workbook part in the package,
// following the package relationships (usually "xl/workbook.xml")
func findWorkbookPart(zr *zip.Reader) string {
	const defaultPath = "xl/workbook.xml"

	var rels struct {
		Relationships []struct {
			Type   string `xml:"Type,attr"`
			Target string `xml:"Target,attr"`
		} `xml:"Relationship"`
	}
	if err := decodeZipXML(zr, "_rels/.rels", &rels); err != nil {
		return defaultPath
	}
	for _, rel := range rels.Relationships {
		if strings.HasSuffix(rel.Type, "/officeDocument") {
			return path.Clean(strings.TrimPrefix(rel.Target, "/"))
		}
	}
	return defaultPath
}

// decodeZipXML decodes an XML file in a zip archive
func decodeZipXML(zr *zip.Reader, name string, v any) error {
	f, err := zr.Open(name)
	if err != nil {
		return err
	}
	defer f.Close()
	return xml.NewDecoder(f).Decode(v)
}

// readSheetsWithExcel reads the sheet list using Excel, for formats that
// cannot be read directly (.xls)
func readSheetsWithExcel(filePath string) ([]ExcelSheetInfo, error) {
	var sheets []ExcelSheetInfo
	err := withCOM(func() error {
		return withExcelWorkbook(filePath, func(wb *ole.IDispatch) error {
			sheetsVar, err := oleutil.GetProperty(wb, "Sheets")
			if err != nil {
				return fmt.Errorf("failed to get sheets: %v", err)
			}
			sheetsDisp := sheetsVar.ToIDispatch()
			defer sheetsDisp.Release()

			count := collectionCount(sheetsDisp)
			if count < 0 {
				return fmt.Errorf("failed to get sheet count")
			}
			for i := 1; i <= count; i++ {
				sheetVar, err := oleutil.GetProperty(sheetsDisp, "Item", i)
				if err != nil {
					return fmt.Errorf("failed to get sheet %d: %v", i, err)
				}
				sheet := sheetVar.ToIDispatch()
				nameVar, nameErr := oleutil.GetProperty(sheet, "Name")
				visibleVar, visibleErr := oleutil.GetProperty(sheet, "Visible")
				sheet.Release()
				if nameErr != nil || visibleErr != nil {
					return fmt.Errorf("failed to get properties of sheet %d", i)
				}
				visibility, _ := visibleVar.Value().(int32) // XlSheetVisibility
				sheets = append(sheets, ExcelSheetInfo{
					Name:    nameVar.ToString(),
					Visible: visibility == xlSheetVisible,
					Index:   i - 1,
				})
			}
			return nil
		})
	})
	if err != nil {
		return nil, err
	}
	return sheets, nil
}
