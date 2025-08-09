package main

import (
	"context"
	"embed"
	"flag"
	"os"
	"path/filepath"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/menu"
	"github.com/wailsapp/wails/v2/pkg/menu/keys"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	// Parse command line arguments
	flag.Parse()

	// Get the directory from positional arguments
	var initialDir string
	args := flag.Args()

	if len(args) > 0 {
		// Use the first positional argument as the directory
		targetDir := args[0]

		// Convert to absolute path
		absDir, err := filepath.Abs(targetDir)
		if err != nil {
			println("Error resolving directory path:", err.Error())
			os.Exit(1)
		}

		// Check if directory exists
		if _, err := os.Stat(absDir); os.IsNotExist(err) {
			println("Directory does not exist:", absDir)
			os.Exit(1)
		}

		initialDir = absDir
	} else {
		// Use current working directory if no argument provided
		cwd, err := os.Getwd()
		if err != nil {
			println("Error getting current directory:", err.Error())
			os.Exit(1)
		}
		initialDir = cwd
	}

	// Create an instance of the app structure
	app := NewApp(initialDir)

	// Create application menu
	appMenu := menu.NewMenu()
	fileMenu := appMenu.AddSubmenu("ファイル")
	fileMenu.AddText("フォルダを選択", keys.CmdOrCtrl("o"), func(_ *menu.CallbackData) {
		app.ChangeWorkingDirectory()
	})
	fileMenu.AddSeparator()
	fileMenu.AddText("PDFを保存", keys.CmdOrCtrl("s"), func(_ *menu.CallbackData) {
		if err := app.ShowSaveDialog(); err != nil {
			runtime.LogError(app.ctx, err.Error())
		}
	})
	fileMenu.AddSeparator()
	fileMenu.AddText("終了", keys.CmdOrCtrl("q"), func(_ *menu.CallbackData) {
		// App will quit automatically
	})

	// Create application with options
	err := wails.Run(&options.App{
		Title:  "pdf-preview-go",
		Width:  1024,
		Height: 768,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		Menu:             appMenu,
		BackgroundColour: &options.RGBA{R: 27, G: 38, B: 54, A: 1},
		OnStartup:        app.Startup,
		OnShutdown:       app.Shutdown,
		OnBeforeClose: func(ctx context.Context) (prevent bool) {
			if app.HasUnsavedChanges() {
				// Show confirmation dialog
				selection, err := runtime.MessageDialog(ctx, runtime.MessageDialogOptions{
					Type:          runtime.QuestionDialog,
					Title:         "未保存の変更があります",
					Message:       "PDFファイルに未保存の変更があります。保存しますか？",
					Buttons:       []string{"保存", "保存しない", "キャンセル"},
					DefaultButton: "保存",
				})

				if err != nil {
					runtime.LogError(ctx, "Dialog error: "+err.Error())
					return false // Allow close on error
				}

				switch selection {
				case "保存":
					if err := app.ShowSaveDialog(); err != nil {
						if err.Error() == "user_cancelled" {
							// User cancelled save dialog, prevent app close
							return true
						}
						runtime.LogError(ctx, "Save error: "+err.Error())
						return true // Prevent close on save error
					}
					return false // Allow close after successful save
				case "保存しない":
					return false // Allow close without saving
				default: // "キャンセル" or closed dialog
					return true // Prevent close
				}
			}
			return false // Allow close if no unsaved changes
		},
		Bind: []interface{}{
			app,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
