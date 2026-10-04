package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDataDirIsSeparateFromCache(t *testing.T) {
	isolateDirs(t)
	if filepath.Dir(dataDir()) != os.Getenv("APPDATA") {
		t.Errorf("dataDir() = %s, want under APPDATA %s", dataDir(), os.Getenv("APPDATA"))
	}
	if dataDir() == pdfCacheDir() {
		t.Error("data and PDF cache share a directory")
	}
}

func TestDirectoryHashIgnoresCase(t *testing.T) {
	if directoryHash(`C:\Work\Docs`) != directoryHash(`c:\work\docs`) {
		t.Error("directory hash depends on case")
	}
	if directoryHash(`C:\Work\Docs`) == directoryHash(`C:\Work\Other`) {
		t.Error("different directories have the same hash")
	}
}

func TestMigrateLegacyData(t *testing.T) {
	isolateDirs(t)
	oldDir := pdfCacheDir()
	newDir := dataDir()
	write := func(dir, name, content string) {
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}
	for _, name := range []string{"directory_history.json", "session_a.json", "session_b.json", "sheet_selections_a.json", "x.pdf"} {
		write(oldDir, name, "old")
	}
	write(newDir, "session_b.json", "new") // already migrated

	migrateLegacyData()

	read := func(dir, name string) string {
		data, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			return "<missing>"
		}
		return string(data)
	}
	checks := []struct {
		dir, name, want string
	}{
		{newDir, "directory_history.json", "old"},
		{newDir, "session_a.json", "old"},
		{newDir, "session_b.json", "new"}, // not overwritten
		{oldDir, "directory_history.json", "<missing>"},
		{oldDir, "session_a.json", "<missing>"},
		{oldDir, "session_b.json", "<missing>"},
		{oldDir, "sheet_selections_a.json", "<missing>"}, // legacy, removed
		{newDir, "sheet_selections_a.json", "<missing>"},
		{oldDir, "x.pdf", "old"}, // PDFs stay in the cache
	}
	for _, c := range checks {
		if got := read(c.dir, c.name); got != c.want {
			t.Errorf("%s in %s = %q, want %q", c.name, filepath.Base(c.dir), got, c.want)
		}
	}
}
