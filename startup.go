package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/sys/windows"
)

// resolveStartupPath interprets the path given on the command line, e.g. by
// the Explorer context menu. It returns the directory to open and, when a
// supported file was given, the file to select in it.
//   - empty: the current directory
//   - a directory: that directory
//   - a supported file: its directory, with the file selected
//   - another file: its directory
func resolveStartupPath(arg string) (dir, file string, err error) {
	if arg == "" {
		dir, err = os.Getwd()
		if err != nil {
			return "", "", fmt.Errorf("カレントディレクトリを取得できません: %v", err)
		}
		return dir, "", nil
	}

	path, err := filepath.Abs(arg)
	if err != nil {
		return "", "", fmt.Errorf("パスを解釈できません: %s: %v", arg, err)
	}
	info, err := os.Stat(path)
	if err != nil {
		return "", "", fmt.Errorf("フォルダまたはファイルが見つかりません: %s", path)
	}
	if info.IsDir() {
		return path, "", nil
	}

	dir = filepath.Dir(path)
	name := filepath.Base(path)
	if isOfficeFile(strings.ToLower(filepath.Ext(name))) && !strings.HasPrefix(name, "~$") {
		file = path
	}
	return dir, file, nil
}

// showStartupError shows an error before the app window exists. The app has
// no console, so an error that is only logged would not be seen.
func showStartupError(err error) {
	title, _ := windows.UTF16PtrFromString("PDF Preview")
	message, _ := windows.UTF16PtrFromString(err.Error())
	windows.MessageBox(0, message, title, windows.MB_OK|windows.MB_ICONERROR)
}
