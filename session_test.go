package main

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

	a := &App{}
	err := a.SaveDirectorySessionCache(dir,
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

	session, err := a.LoadDirectorySessionCache(dir)
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
	session, err := (&App{}).LoadDirectorySessionCache(t.TempDir())
	if err != nil || session != nil {
		t.Errorf("got (%v, %v), want (nil, nil)", session, err)
	}
}

func TestDirectoryHistory(t *testing.T) {
	isolateDirs(t)
	a := &App{}
	first, second := t.TempDir(), t.TempDir()
	gone := filepath.Join(t.TempDir(), "gone")
	os.Mkdir(gone, 0755)

	for _, dir := range []string{first, gone, second, first} {
		if err := a.AddDirectoryToHistory(dir); err != nil {
			t.Fatal(err)
		}
		time.Sleep(10 * time.Millisecond) // distinct LastUsed
	}
	os.Remove(gone)

	history, err := a.GetDirectoryHistory()
	if err != nil {
		t.Fatal(err)
	}
	var paths []string
	for _, h := range history {
		paths = append(paths, h.Path)
	}
	// Most recent first, without the removed directory
	if want := []string{first, second}; !reflect.DeepEqual(paths, want) {
		t.Fatalf("history = %v, want %v", paths, want)
	}
	if history[0].UsageCount != 2 {
		t.Errorf("UsageCount = %d, want 2", history[0].UsageCount)
	}
}
