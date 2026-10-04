package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestFindChangedFile(t *testing.T) {
	dir := t.TempDir()
	a := filepath.Join(dir, "a.xlsx")
	b := filepath.Join(dir, "b.xlsx")
	os.WriteFile(a, []byte("x"), 0644)
	os.WriteFile(b, []byte("x"), 0644)

	app := &App{
		autoUpdateEnabled:  true,
		lastConvertedFiles: []string{a, b},
		fileModTimes:       statModTimes([]string{a, b}),
	}

	if got := app.findChangedFile(); got != "" {
		t.Errorf("unchanged files: got %q, want none", got)
	}

	// Office lock files and other files in the directory are ignored
	os.WriteFile(filepath.Join(dir, "~$a.xlsx"), []byte("lock"), 0644)
	if got := app.findChangedFile(); got != "" {
		t.Errorf("after lock file: got %q, want none", got)
	}

	later := time.Now().Add(time.Hour)
	os.Chtimes(b, later, later)
	if got := app.findChangedFile(); got != b {
		t.Errorf("modified file: got %q, want %q", got, b)
	}

	app.autoUpdateEnabled = false
	if got := app.findChangedFile(); got != "" {
		t.Errorf("auto-update disabled: got %q, want none", got)
	}

	// A file that is temporarily missing (e.g. while Office saves) is not a change
	app.autoUpdateEnabled = true
	app.fileModTimes = statModTimes([]string{a, b})
	os.Remove(a)
	if got := app.findChangedFile(); got != "" {
		t.Errorf("missing file: got %q, want none", got)
	}
}

func TestRequestRegenerateCoalesces(t *testing.T) {
	app := &App{regenerateCh: make(chan struct{}, 1)}
	for i := 0; i < 5; i++ {
		app.requestRegenerate() // must not block
	}
	if n := len(app.regenerateCh); n != 1 {
		t.Errorf("pending requests = %d, want 1", n)
	}
}
