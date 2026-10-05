package main

import (
	"fmt"
	"os"
	"testing"

	"golang.org/x/sys/windows/registry"
)

// testClassesKey returns a temporary key in HKCU to register the context
// menu under, so that tests do not change the user's real context menu
func testClassesKey(t *testing.T) string {
	t.Helper()
	root := fmt.Sprintf(`Software\pdf-preview-go-test-%d`, os.Getpid())
	t.Cleanup(func() {
		if err := deleteRegistryTree(registry.CURRENT_USER, root); err != nil {
			t.Errorf("failed to clean up %s: %v", root, err)
		}
	})
	return root + `\Classes`
}

func deleteRegistryTree(base registry.Key, path string) error {
	key, err := registry.OpenKey(base, path, registry.ENUMERATE_SUB_KEYS)
	if err == registry.ErrNotExist {
		return nil
	}
	if err != nil {
		return err
	}
	names, err := key.ReadSubKeyNames(-1)
	key.Close()
	if err != nil {
		return err
	}
	for _, name := range names {
		if err := deleteRegistryTree(base, path+`\`+name); err != nil {
			return err
		}
	}
	return registry.DeleteKey(base, path)
}

func readRegistryString(t *testing.T, path, name string) string {
	t.Helper()
	key, err := registry.OpenKey(registry.CURRENT_USER, path, registry.QUERY_VALUE)
	if err != nil {
		t.Fatalf("open %s: %v", path, err)
	}
	defer key.Close()
	value, _, err := key.GetStringValue(name)
	if err != nil {
		t.Fatalf("read %s\\%s: %v", path, name, err)
	}
	return value
}

func TestContextMenuRegistration(t *testing.T) {
	classes := testClassesKey(t)
	exe := `C:\Program Files\kkzk\pdf-preview-go\pdf-preview-go.exe`

	if isContextMenuRegistered(classes) {
		t.Fatal("registered before registerContextMenu")
	}
	if err := registerContextMenu(classes, exe); err != nil {
		t.Fatal(err)
	}
	if !isContextMenuRegistered(classes) {
		t.Fatal("not registered after registerContextMenu")
	}

	// Each target has the label, icon, single-selection model and command
	for _, target := range contextMenuTargets {
		verbKey := contextMenuVerbKey(classes, target)
		if got := readRegistryString(t, verbKey, ""); got != contextMenuLabel {
			t.Errorf("%s label = %q", target.class, got)
		}
		if got := readRegistryString(t, verbKey, "Icon"); got != exe {
			t.Errorf("%s icon = %q", target.class, got)
		}
		if target.singleSelection {
			if got := readRegistryString(t, verbKey, "MultiSelectModel"); got != "Single" {
				t.Errorf("%s MultiSelectModel = %q", target.class, got)
			}
		} else if hasRegistryValue(t, verbKey, "MultiSelectModel") {
			// It would hide the entry, as nothing is selected on a background
			t.Errorf("%s has MultiSelectModel", target.class)
		}
		want := fmt.Sprintf(`"%s" "%s"`, exe, target.arg)
		if got := readRegistryString(t, verbKey+`\command`, ""); got != want {
			t.Errorf("%s command = %q, want %q", target.class, got, want)
		}
	}
	background := contextMenuVerbKey(classes, contextMenuTarget{class: `Directory\Background`})
	if got := readRegistryString(t, background+`\command`, ""); got != `"`+exe+`" "%V"` {
		t.Errorf("folder background command = %q, want %%V", got)
	}

	if err := unregisterContextMenu(classes); err != nil {
		t.Fatal(err)
	}
	if isContextMenuRegistered(classes) {
		t.Error("still registered after unregisterContextMenu")
	}
	for _, target := range contextMenuTargets {
		if key, err := registry.OpenKey(registry.CURRENT_USER, contextMenuVerbKey(classes, target), registry.QUERY_VALUE); err == nil {
			key.Close()
			t.Errorf("%s entry remains", target.class)
		}
	}

	// Unregistering again is not an error
	if err := unregisterContextMenu(classes); err != nil {
		t.Errorf("second unregister: %v", err)
	}
}

func hasRegistryValue(t *testing.T, path, name string) bool {
	t.Helper()
	key, err := registry.OpenKey(registry.CURRENT_USER, path, registry.QUERY_VALUE)
	if err != nil {
		t.Fatalf("open %s: %v", path, err)
	}
	defer key.Close()
	_, _, err = key.GetStringValue(name)
	return err == nil
}

func TestContextMenuRegistrationFixesBackground(t *testing.T) {
	// Earlier versions set MultiSelectModel=Single on the folder background
	// entry, which hid it; registering again must remove it
	classes := testClassesKey(t)
	background := contextMenuVerbKey(classes, contextMenuTarget{class: `Directory\Background`})
	if err := setRegistryStrings(background, map[string]string{"MultiSelectModel": "Single"}); err != nil {
		t.Fatal(err)
	}

	if err := registerContextMenu(classes, `C:\app\pdf-preview-go.exe`); err != nil {
		t.Fatal(err)
	}
	if hasRegistryValue(t, background, "MultiSelectModel") {
		t.Error("MultiSelectModel was not removed from the folder background entry")
	}
}
