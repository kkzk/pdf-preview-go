package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/pdfcpu/pdfcpu/pkg/api"
)

func TestCacheFileName(t *testing.T) {
	if cacheFileName("a.xlsx", []string{"A", "B"}) != cacheFileName("a.xlsx", []string{"B", "A"}) {
		t.Error("cache file name depends on sheet selection order")
	}
	if cacheFileName("a.xlsx", []string{"a,b"}) == cacheFileName("a.xlsx", []string{"a", "b"}) {
		t.Error("sheet names containing a comma collide")
	}
	if cacheFileName("a.xlsx", nil) == cacheFileName("a.xlsx", []string{"A"}) {
		t.Error("sheet selection is not part of the cache file name")
	}
	if cacheFileName("a.xlsx", nil) == cacheFileName("b.xlsx", nil) {
		t.Error("source path is not part of the cache file name")
	}

	// The caller's slice must not be reordered
	sheets := []string{"B", "A"}
	cacheFileName("a.xlsx", sheets)
	if sheets[0] != "B" {
		t.Error("cacheFileName modified the selected sheets")
	}
}

func TestConvertToPDFUsesCache(t *testing.T) {
	dir := t.TempDir()
	c := NewOfficeConverter(filepath.Join(dir, "cache"))

	// PDF sources are copied, so the cache can be tested without Office
	src := filepath.Join(dir, "src.pdf")
	writeTestPDF(t, src, 1)
	srcTime := time.Now().Add(-time.Hour).Truncate(time.Second)
	if err := os.Chtimes(src, srcTime, srcTime); err != nil {
		t.Fatal(err)
	}

	out, err := c.ConvertToPDF(src, nil)
	if err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(out)
	if err != nil {
		t.Fatal(err)
	}
	if !info.ModTime().Equal(srcTime) {
		t.Errorf("cached PDF mtime = %v, want source mtime %v", info.ModTime(), srcTime)
	}

	// Up to date: the cached file is reused as is
	if err := os.WriteFile(out, []byte("cached"), 0644); err != nil {
		t.Fatal(err)
	}
	os.Chtimes(out, time.Now(), srcTime)
	if _, err := c.ConvertToPDF(src, nil); err != nil {
		t.Fatal(err)
	}
	if data, _ := os.ReadFile(out); string(data) != "cached" {
		t.Error("up-to-date cached PDF was regenerated")
	}

	// Source modified: the PDF is regenerated
	newTime := srcTime.Add(time.Minute)
	os.Chtimes(src, newTime, newTime)
	if _, err := c.ConvertToPDF(src, nil); err != nil {
		t.Fatal(err)
	}
	if data, _ := os.ReadFile(out); string(data) == "cached" {
		t.Error("cached PDF was not regenerated after the source changed")
	}
}

func TestConvertToPDFUnsupported(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "a.txt")
	os.WriteFile(src, []byte("x"), 0644)

	_, err := NewOfficeConverter(dir).ConvertToPDF(src, nil)
	if err == nil || !strings.Contains(err.Error(), "unsupported") {
		t.Errorf("err = %v, want unsupported file type", err)
	}
}

func TestCleanupCache(t *testing.T) {
	dir := t.TempDir()
	now := time.Now()
	old := now.AddDate(0, 0, -40)
	write := func(name string, atime, mtime time.Time) {
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, []byte("x"), 0644); err != nil {
			t.Fatal(err)
		}
		if err := os.Chtimes(path, atime, mtime); err != nil {
			t.Fatal(err)
		}
	}

	write("unused.pdf", old, now)                       // not used for 40 days
	write("old_source.pdf", now, now.AddDate(-2, 0, 0)) // source is old, but used recently
	write("session_x.json", old, old)                   // not a PDF

	if err := NewOfficeConverter(dir).CleanupCache(30 * 24 * time.Hour); err != nil {
		t.Fatal(err)
	}

	for name, want := range map[string]bool{
		"unused.pdf":     false,
		"old_source.pdf": true,
		"session_x.json": true,
	} {
		_, err := os.Stat(filepath.Join(dir, name))
		if exists := err == nil; exists != want {
			t.Errorf("%s exists = %v, want %v", name, exists, want)
		}
	}
}

func TestMergePDFs(t *testing.T) {
	dir := t.TempDir()
	a := filepath.Join(dir, "a.pdf")
	b := filepath.Join(dir, "b.pdf")
	writeTestPDF(t, a, 2)
	writeTestPDF(t, b, 3)

	merged := filepath.Join(dir, "merged.pdf")
	if err := MergePDFs([]string{a, b}, merged); err != nil {
		t.Fatal(err)
	}
	if n, err := api.PageCountFile(merged); err != nil || n != 5 {
		t.Errorf("merged page count = %d (err %v), want 5", n, err)
	}

	single := filepath.Join(dir, "single.pdf")
	if err := MergePDFs([]string{a}, single); err != nil {
		t.Fatal(err)
	}
	if n, err := api.PageCountFile(single); err != nil || n != 2 {
		t.Errorf("single page count = %d (err %v), want 2", n, err)
	}

	if err := MergePDFs(nil, filepath.Join(dir, "none.pdf")); err == nil {
		t.Error("expected an error for no input PDFs")
	}
}

func TestConvertExcelSheetSelection(t *testing.T) {
	requireOffice(t, "Excel.Application")

	dir := t.TempDir()
	c := NewOfficeConverter(filepath.Join(dir, "cache"))
	workbook := testdataPath(t, "testdata1.xlsx") // Sheet1, Sheet2 (1 page each)
	hidden := filepath.Join(dir, "hidden2.xlsx")  // Sheet2 hidden
	writeWorkbookWithSheetStates(t, hidden, map[string]string{"Sheet2": "hidden"})

	tests := []struct {
		name      string
		src       string
		sheets    []string
		wantPages int
		wantErr   bool
	}{
		{"all sheets", workbook, nil, 2, false},
		{"Sheet1 only", workbook, []string{"Sheet1"}, 1, false},
		{"Sheet2 only", workbook, []string{"Sheet2"}, 1, false},
		// Sheet1 must be hidden although it is the only visible sheet
		{"hidden sheet only", hidden, []string{"Sheet2"}, 1, false},
		{"missing sheet is skipped", workbook, []string{"Sheet1", "NoSuch"}, 1, false},
		{"all sheets missing", workbook, []string{"NoSuch"}, 0, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, err := c.ConvertToPDF(tt.src, tt.sheets)
			if tt.wantErr {
				if err == nil {
					t.Error("expected an error")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if n, err := api.PageCountFile(out); err != nil || n != tt.wantPages {
				t.Errorf("page count = %d (err %v), want %d", n, err, tt.wantPages)
			}
		})
	}
}

func TestConvertWord(t *testing.T) {
	requireOffice(t, "Word.Application")

	out, err := NewOfficeConverter(t.TempDir()).ConvertToPDF(testdataPath(t, "testdata3.docx"), nil)
	if err != nil {
		t.Fatal(err)
	}
	if n, err := api.PageCountFile(out); err != nil || n < 1 {
		t.Errorf("page count = %d (err %v), want at least 1", n, err)
	}
}
