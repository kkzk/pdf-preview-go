package appdata

import "testing"

// isolateDirs points the temp and app data directories to fresh temporary
// directories, so tests do not touch the user's cache, history or sessions
func isolateDirs(t *testing.T) {
	t.Helper()
	tmp := t.TempDir()
	t.Setenv("TMP", tmp)
	t.Setenv("TEMP", tmp)
	t.Setenv("APPDATA", t.TempDir())
}
