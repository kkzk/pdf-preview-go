package files

import (
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"
)

func TestListDirectory(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"a.xlsx", "b.docx", "c.pdf", "notes.txt", "~$a.xlsx"} {
		os.WriteFile(filepath.Join(dir, name), []byte("x"), 0644)
	}
	os.MkdirAll(filepath.Join(dir, "sub", "deep"), 0755)
	os.WriteFile(filepath.Join(dir, "sub", "d.xlsx"), []byte("x"), 0644)

	files, err := ListDirectory(dir)
	if err != nil {
		t.Fatal(err)
	}

	var names []string
	for _, f := range files {
		names = append(names, f.Name)
		if f.Children != nil {
			t.Errorf("%s: children are loaded, want only one level", f.Name)
		}
	}
	sort.Strings(names)
	// Unsupported files and Office lock files are excluded
	if want := []string{"a.xlsx", "b.docx", "c.pdf", "sub"}; !reflect.DeepEqual(names, want) {
		t.Errorf("names = %v, want %v", names, want)
	}

	// An empty folder returns an empty list (not nil), so the frontend can
	// tell it apart from a folder that has not been loaded
	empty, err := ListDirectory(filepath.Join(dir, "sub", "deep"))
	if err != nil {
		t.Fatal(err)
	}
	if empty == nil || len(empty) != 0 {
		t.Errorf("empty folder = %#v, want []", empty)
	}
}

func TestStat(t *testing.T) {
	dir := t.TempDir()
	existing := filepath.Join(dir, "a.xlsx")
	os.WriteFile(existing, []byte("12345"), 0644)

	files := Stat([]string{existing, filepath.Join(dir, "missing.xlsx"), dir})
	if len(files) != 1 {
		t.Fatalf("files = %+v, want only %s", files, existing)
	}
	if f := files[0]; f.Path != existing || f.Name != "a.xlsx" || f.Size != 5 || f.IsDir {
		t.Errorf("file = %+v", f)
	}

	if files := Stat(nil); files == nil || len(files) != 0 {
		t.Errorf("no paths = %#v, want []", files)
	}
}

func TestOpenRejectsUnsupported(t *testing.T) {
	// The success path is not tested, as it would launch Excel/Word
	dir := t.TempDir()
	txt := filepath.Join(dir, "notes.txt")
	os.WriteFile(txt, []byte("x"), 0644)

	for _, path := range []string{filepath.Join(dir, "missing.xlsx"), dir, txt} {
		if err := Open(path); err == nil {
			t.Errorf("Open(%s) succeeded, want an error", path)
		}
	}
}
