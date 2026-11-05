package appdata

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

func TestDirectorySessionSurvivesFileChanges(t *testing.T) {
	isolateDirs(t)
	dir := t.TempDir()
	edited := filepath.Join(dir, "edited.xlsx")
	deleted := filepath.Join(dir, "deleted.xlsx")
	sub := filepath.Join(dir, "sub")
	for _, f := range []string{edited, deleted} {
		os.WriteFile(f, []byte("x"), 0644)
	}
	os.Mkdir(sub, 0755)

	err := SaveDirectorySessionCache(dir,
		[]string{edited, deleted},
		[]string{sub, filepath.Join(dir, "gone")},
		edited,
		map[string][]string{edited: {"Sheet2"}, deleted: {"Sheet1"}})
	if err != nil {
		t.Fatal(err)
	}

	// Edit one file and delete the other
	os.WriteFile(edited, []byte("modified content"), 0644)
	later := time.Now().Add(time.Hour)
	os.Chtimes(edited, later, later)
	os.Remove(deleted)

	session, err := LoadDirectorySessionCache(dir)
	if err != nil {
		t.Fatal(err)
	}
	if session == nil {
		t.Fatal("session not found")
	}
	if want := []string{edited}; !reflect.DeepEqual(session.SelectedFiles, want) {
		t.Errorf("SelectedFiles = %v, want %v", session.SelectedFiles, want)
	}
	if want := []string{sub}; !reflect.DeepEqual(session.ExpandedFolders, want) {
		t.Errorf("ExpandedFolders = %v, want %v", session.ExpandedFolders, want)
	}
	if session.CurrentFile != edited {
		t.Errorf("CurrentFile = %q, want %q", session.CurrentFile, edited)
	}
	if want := map[string][]string{edited: {"Sheet2"}}; !reflect.DeepEqual(session.SheetSelections, want) {
		t.Errorf("SheetSelections = %v, want %v", session.SheetSelections, want)
	}
}

func TestDirectorySessionNotFound(t *testing.T) {
	isolateDirs(t)
	session, err := LoadDirectorySessionCache(t.TempDir())
	if err != nil || session != nil {
		t.Errorf("got (%v, %v), want (nil, nil)", session, err)
	}
}
