package watch

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

	w := New(nil, nil)
	w.Track([]string{a, b}, StatModTimes([]string{a, b}))

	if got := w.findChangedFile(); got != "" {
		t.Errorf("unchanged files: got %q, want none", got)
	}

	// Office lock files and other files in the directory are ignored
	os.WriteFile(filepath.Join(dir, "~$a.xlsx"), []byte("lock"), 0644)
	if got := w.findChangedFile(); got != "" {
		t.Errorf("after lock file: got %q, want none", got)
	}

	later := time.Now().Add(time.Hour)
	os.Chtimes(b, later, later)
	if got := w.findChangedFile(); got != b {
		t.Errorf("modified file: got %q, want %q", got, b)
	}

	w.SetEnabled(false)
	if got := w.findChangedFile(); got != "" {
		t.Errorf("auto-update disabled: got %q, want none", got)
	}

	// A file that is temporarily missing (e.g. while Office saves) is not a change
	w.SetEnabled(true)
	w.Track([]string{a, b}, StatModTimes([]string{a, b}))
	os.Remove(a)
	if got := w.findChangedFile(); got != "" {
		t.Errorf("missing file: got %q, want none", got)
	}
}

func TestRequestCheckCoalesces(t *testing.T) {
	w := New(nil, nil)
	for i := 0; i < 5; i++ {
		w.requestCheck() // must not block
	}
	if n := len(w.requestCh); n != 1 {
		t.Errorf("pending requests = %d, want 1", n)
	}
}

func TestWatcherCallsOnChange(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "a.xlsx")
	os.WriteFile(file, []byte("x"), 0644)

	changed := make(chan string, 1)
	w := New(func(f string) {
		select {
		case changed <- f:
		default:
		}
	}, func(err error) { t.Errorf("watcher error: %v", err) })
	w.Start()
	defer w.Stop()
	if err := w.WatchDir(dir); err != nil {
		t.Fatal(err)
	}
	w.Track([]string{file}, StatModTimes([]string{file}))

	later := time.Now().Add(time.Hour)
	os.Chtimes(file, later, later)

	select {
	case got := <-changed:
		if got != file {
			t.Errorf("onChange(%q), want %q", got, file)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("onChange was not called")
	}
}
