package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveStartupPath(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"book.xlsx", "doc.pdf", "notes.txt", "~$book.xlsx"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("x"), 0644); err != nil {
			t.Fatal(err)
		}
	}
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name     string
		arg      string
		wantDir  string
		wantFile string
	}{
		{"no argument opens the current directory", "", cwd, ""},
		{"directory", dir, dir, ""},
		{"Excel file is selected in its folder", filepath.Join(dir, "book.xlsx"), dir, filepath.Join(dir, "book.xlsx")},
		{"PDF file is selected in its folder", filepath.Join(dir, "doc.pdf"), dir, filepath.Join(dir, "doc.pdf")},
		{"unsupported file opens its folder", filepath.Join(dir, "notes.txt"), dir, ""},
		{"Office lock file opens its folder", filepath.Join(dir, "~$book.xlsx"), dir, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotDir, gotFile, err := resolveStartupPath(tt.arg)
			if err != nil {
				t.Fatal(err)
			}
			if gotDir != tt.wantDir || gotFile != tt.wantFile {
				t.Errorf("got (%q, %q), want (%q, %q)", gotDir, gotFile, tt.wantDir, tt.wantFile)
			}
		})
	}
}

func TestResolveStartupPathRelative(t *testing.T) {
	// Relative paths are resolved against the current directory
	dir := t.TempDir()
	t.Chdir(dir)
	os.Mkdir("sub", 0755)

	gotDir, gotFile, err := resolveStartupPath("sub")
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(dir, "sub"); gotDir != want || gotFile != "" {
		t.Errorf("got (%q, %q), want (%q, \"\")", gotDir, gotFile, want)
	}
}

func TestResolveStartupPathMissing(t *testing.T) {
	if _, _, err := resolveStartupPath(filepath.Join(t.TempDir(), "missing")); err == nil {
		t.Error("expected an error for a missing path")
	}
}
