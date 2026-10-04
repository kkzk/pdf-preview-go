package main

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-ole/go-ole"
)

// isolateDirs points the temp and app data directories to fresh temporary
// directories, so tests do not touch the user's cache, history or sessions
func isolateDirs(t *testing.T) {
	t.Helper()
	tmp := t.TempDir()
	t.Setenv("TMP", tmp)
	t.Setenv("TEMP", tmp)
	t.Setenv("APPDATA", t.TempDir())
}

// requireOffice skips the test in short mode or if the given Office
// application (e.g. "Excel.Application") is not installed
func requireOffice(t *testing.T, progID string) {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping Office test in short mode")
	}
	err := withCOM(func() error {
		_, err := ole.ClassIDFrom(progID)
		return err
	})
	if err != nil {
		t.Skipf("%s is not available: %v", progID, err)
	}
}

// testdataPath returns the absolute path of a file in test/testdata
func testdataPath(t *testing.T, name string) string {
	t.Helper()
	path, err := filepath.Abs(filepath.Join("test", "testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	return path
}

// writeTestPDF writes a minimal PDF with the given number of blank pages
func writeTestPDF(t *testing.T, path string, pages int) {
	t.Helper()

	var b bytes.Buffer
	var offsets []int
	obj := func(body string) {
		offsets = append(offsets, b.Len())
		fmt.Fprintf(&b, "%d 0 obj\n%s\nendobj\n", len(offsets), body)
	}

	b.WriteString("%PDF-1.4\n")
	kids := make([]string, pages)
	for i := range kids {
		kids[i] = fmt.Sprintf("%d 0 R", i+3)
	}
	obj("<< /Type /Catalog /Pages 2 0 R >>")
	obj(fmt.Sprintf("<< /Type /Pages /Kids [%s] /Count %d >>", strings.Join(kids, " "), pages))
	for i := 0; i < pages; i++ {
		obj("<< /Type /Page /Parent 2 0 R /MediaBox [0 0 200 200] >>")
	}

	xref := b.Len()
	fmt.Fprintf(&b, "xref\n0 %d\n0000000000 65535 f \n", len(offsets)+1)
	for _, offset := range offsets {
		fmt.Fprintf(&b, "%010d 00000 n \n", offset)
	}
	fmt.Fprintf(&b, "trailer\n<< /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n", len(offsets)+1, xref)

	if err := os.WriteFile(path, b.Bytes(), 0644); err != nil {
		t.Fatal(err)
	}
}

// writeWorkbookWithSheetStates copies test/testdata/testdata1.xlsx (sheets
// "Sheet1" and "Sheet2") to dst, setting the state attribute of the given
// sheets (e.g. "hidden", "veryHidden") in xl/workbook.xml
func writeWorkbookWithSheetStates(t *testing.T, dst string, states map[string]string) {
	t.Helper()

	zr, err := zip.OpenReader(testdataPath(t, "testdata1.xlsx"))
	if err != nil {
		t.Fatal(err)
	}
	defer zr.Close()

	out, err := os.Create(dst)
	if err != nil {
		t.Fatal(err)
	}
	defer out.Close()
	zw := zip.NewWriter(out)

	for _, f := range zr.File {
		rc, err := f.Open()
		if err != nil {
			t.Fatal(err)
		}
		data, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			t.Fatal(err)
		}

		if f.Name == "xl/workbook.xml" {
			content := string(data)
			for name, state := range states {
				attr := fmt.Sprintf(`name="%s"`, name)
				if !strings.Contains(content, attr) {
					t.Fatalf("sheet %q not found in workbook.xml", name)
				}
				content = strings.Replace(content, attr, fmt.Sprintf(`%s state="%s"`, attr, state), 1)
			}
			data = []byte(content)
		}

		w, err := zw.Create(f.Name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write(data); err != nil {
			t.Fatal(err)
		}
	}

	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
}
