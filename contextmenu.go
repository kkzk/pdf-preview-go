package main

import (
	"errors"
	"fmt"
	"os"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

// Explorer context menu ("PDF Previewで開く") for folders, folder backgrounds
// and supported files. It is registered per user under HKCU only when the
// user turns it on from the app's 設定 menu, so no admin rights are needed.
// The installer only removes it (and the all-users entries of earlier
// versions) on uninstall.

const (
	// userClassesKey is where per-user file classes are registered (HKCU)
	userClassesKey = `Software\Classes`

	contextMenuVerb  = "OpenWithPdfPreview"
	contextMenuLabel = "PDF Previewで開く(&P)"
)

// contextMenuTarget is a file class the menu entry is added to
type contextMenuTarget struct {
	class string // e.g. "Directory", `SystemFileAssociations\.xlsx`
	arg   string // path placeholder passed to the app
	// singleSelection hides the entry when multiple items are selected.
	// It must not be set for a folder background, where no item is
	// selected: MultiSelectModel=Single would hide the entry there.
	singleSelection bool
}

var contextMenuTargets = []contextMenuTarget{
	{"Directory", "%1", true},
	{`Directory\Background`, "%V", false}, // %V: the folder whose background was clicked
	{`SystemFileAssociations\.xlsx`, "%1", true},
	{`SystemFileAssociations\.xlsm`, "%1", true},
	{`SystemFileAssociations\.xls`, "%1", true},
	{`SystemFileAssociations\.docx`, "%1", true},
	{`SystemFileAssociations\.doc`, "%1", true},
	{`SystemFileAssociations\.pdf`, "%1", true},
}

func contextMenuVerbKey(classesKey string, target contextMenuTarget) string {
	return classesKey + `\` + target.class + `\shell\` + contextMenuVerb
}

// registerContextMenu adds the menu entries under classesKey (in HKCU) to
// start exePath. Entries for folders and files are shown only when a single
// item is selected.
func registerContextMenu(classesKey, exePath string) error {
	for _, target := range contextMenuTargets {
		verbKey := contextMenuVerbKey(classesKey, target)
		verbValues := map[string]string{
			"":     contextMenuLabel,
			"Icon": exePath,
		}
		if target.singleSelection {
			verbValues["MultiSelectModel"] = "Single"
		}
		if err := setRegistryStrings(verbKey, verbValues); err != nil {
			return err
		}
		if !target.singleSelection {
			// Remove the value written by earlier versions, which hid the entry
			if err := deleteRegistryValue(verbKey, "MultiSelectModel"); err != nil {
				return err
			}
		}
		command := fmt.Sprintf(`"%s" "%s"`, exePath, target.arg)
		if err := setRegistryStrings(verbKey+`\command`, map[string]string{"": command}); err != nil {
			return err
		}
	}
	notifyAssociationsChanged()
	return nil
}

// unregisterContextMenu removes the menu entries under classesKey (in HKCU).
// Entries that do not exist are ignored.
func unregisterContextMenu(classesKey string) error {
	var errs []error
	for _, target := range contextMenuTargets {
		verbKey := contextMenuVerbKey(classesKey, target)
		// A key must be deleted after its subkeys
		for _, keyPath := range []string{verbKey + `\command`, verbKey} {
			err := registry.DeleteKey(registry.CURRENT_USER, keyPath)
			if err != nil && !errors.Is(err, registry.ErrNotExist) {
				errs = append(errs, fmt.Errorf("failed to delete %s: %v", keyPath, err))
			}
		}
	}
	notifyAssociationsChanged()
	return errors.Join(errs...)
}

// isContextMenuRegistered reports whether the menu entry for folders exists
// under classesKey (in HKCU)
func isContextMenuRegistered(classesKey string) bool {
	key, err := registry.OpenKey(registry.CURRENT_USER, contextMenuVerbKey(classesKey, contextMenuTargets[0])+`\command`, registry.QUERY_VALUE)
	if err != nil {
		return false
	}
	key.Close()
	return true
}

func setRegistryStrings(keyPath string, values map[string]string) error {
	key, _, err := registry.CreateKey(registry.CURRENT_USER, keyPath, registry.SET_VALUE)
	if err != nil {
		return fmt.Errorf("failed to create %s: %v", keyPath, err)
	}
	defer key.Close()
	for name, value := range values {
		if err := key.SetStringValue(name, value); err != nil {
			return fmt.Errorf("failed to set %s\\%s: %v", keyPath, name, err)
		}
	}
	return nil
}

// deleteRegistryValue deletes a value, ignoring one that does not exist
func deleteRegistryValue(keyPath, name string) error {
	key, err := registry.OpenKey(registry.CURRENT_USER, keyPath, registry.SET_VALUE)
	if err != nil {
		return fmt.Errorf("failed to open %s: %v", keyPath, err)
	}
	defer key.Close()
	if err := key.DeleteValue(name); err != nil && !errors.Is(err, registry.ErrNotExist) {
		return fmt.Errorf("failed to delete %s\\%s: %v", keyPath, name, err)
	}
	return nil
}

// notifyAssociationsChanged tells Explorer that file associations changed,
// so that the context menu is updated
func notifyAssociationsChanged() {
	const shcneAssocChanged = 0x08000000
	proc := windows.NewLazySystemDLL("shell32.dll").NewProc("SHChangeNotify")
	if proc.Find() == nil {
		proc.Call(shcneAssocChanged, 0, 0, 0)
	}
}

// IsContextMenuEnabled reports whether the Explorer context menu is registered
// for the current user
func (a *App) IsContextMenuEnabled() bool {
	return isContextMenuRegistered(userClassesKey)
}

// SetContextMenuEnabled turns the Explorer context menu on or off for the
// current user. When turned on, it starts the running executable.
func (a *App) SetContextMenuEnabled(enabled bool) error {
	if !enabled {
		return unregisterContextMenu(userClassesKey)
	}
	exePath, err := os.Executable()
	if err != nil {
		return err
	}
	return registerContextMenu(userClassesKey, exePath)
}
