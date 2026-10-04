package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestPdfHandler(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "a.pdf"), []byte("%PDF-1.4 test"), 0644)
	os.WriteFile(filepath.Join(dir, "session_x.json"), []byte("{}"), 0644)
	h := (&App{converter: NewOfficeConverter(dir)}).pdfHandler()

	tests := []struct {
		path string
		want int
	}{
		{buildPdfURL("a.pdf", 1), http.StatusOK},
		{"/pdf/missing.pdf", http.StatusNotFound},
		{"/pdf/", http.StatusNotFound},               // no directory listing
		{"/pdf/session_x.json", http.StatusNotFound}, // PDFs only
		{"/other/a.pdf", http.StatusNotFound},
	}
	for _, tt := range tests {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, tt.path, nil))
		if w.Code != tt.want {
			t.Errorf("GET %s = %d, want %d", tt.path, w.Code, tt.want)
		}
	}

	// Range requests are supported (used by the PDF viewer)
	req := httptest.NewRequest(http.MethodGet, "/pdf/a.pdf", nil)
	req.Header.Set("Range", "bytes=0-3")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusPartialContent || w.Body.String() != "%PDF" {
		t.Errorf("range request = %d %q, want 206 %q", w.Code, w.Body.String(), "%PDF")
	}
}
