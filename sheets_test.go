package main

import (
	"path/filepath"
	"reflect"
	"testing"

	"github.com/go-ole/go-ole"
	"github.com/go-ole/go-ole/oleutil"
)

func TestGetExcelSheetsInfoXlsx(t *testing.T) {
	dir := t.TempDir()
	states := filepath.Join(dir, "states.xlsx")
	writeWorkbookWithSheetStates(t, states, map[string]string{"Sheet1": "veryHidden", "Sheet2": "hidden"})
	explicit := filepath.Join(dir, "explicit.xlsx")
	writeWorkbookWithSheetStates(t, explicit, map[string]string{"Sheet1": "visible"})

	tests := []struct {
		name string
		path string
		want []ExcelSheetInfo
	}{
		{"visible sheets", testdataPath(t, "testdata1.xlsx"), []ExcelSheetInfo{
			{Name: "Sheet1", Visible: true, Index: 0},
			{Name: "Sheet2", Visible: true, Index: 1},
		}},
		{"hidden sheets", states, []ExcelSheetInfo{
			{Name: "Sheet1", Visible: false, Index: 0},
			{Name: "Sheet2", Visible: false, Index: 1},
		}},
		{"explicit visible state", explicit, []ExcelSheetInfo{
			{Name: "Sheet1", Visible: true, Index: 0},
			{Name: "Sheet2", Visible: true, Index: 1},
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetExcelSheetsInfo(tt.path)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("got %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestGetExcelSheetsInfoInvalidFile(t *testing.T) {
	if _, err := GetExcelSheetsInfo(filepath.Join(t.TempDir(), "missing.xlsx")); err == nil {
		t.Error("expected an error for a missing file")
	}
}

func TestGetExcelSheetsInfoXls(t *testing.T) {
	requireOffice(t, "Excel.Application")

	// Create an .xls copy with Sheet2 hidden using Excel
	dir := t.TempDir()
	hidden := filepath.Join(dir, "hidden2.xlsx")
	writeWorkbookWithSheetStates(t, hidden, map[string]string{"Sheet2": "hidden"})
	xls := filepath.Join(dir, "legacy.xls")
	err := withCOM(func() error {
		return withExcelWorkbook(hidden, func(wb *ole.IDispatch) error {
			_, err := oleutil.CallMethod(wb, "SaveAs", xls, 56) // 56 = xlExcel8
			return err
		})
	})
	if err != nil {
		t.Fatal(err)
	}

	got, err := GetExcelSheetsInfo(xls)
	if err != nil {
		t.Fatal(err)
	}
	want := []ExcelSheetInfo{
		{Name: "Sheet1", Visible: true, Index: 0},
		{Name: "Sheet2", Visible: false, Index: 1},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v, want %+v", got, want)
	}
}
