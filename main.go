package main

import (
	"context"
	"embed"
	"flag"
	"log"
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
	// Setup logging for debugging
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Println("Starting PDF Preview Go application...")

	// Parse command line arguments
	flag.Parse()
	log.Printf("Command line args: %v", os.Args)

	// Get the directory from positional arguments
	var initialDir string
	args := flag.Args()
	log.Printf("Parsed args: %v", args)

	if len(args) > 0 {
		// Use the first positional argument as the directory
		targetDir := args[0]
		log.Printf("Target directory from args: %s", targetDir)

		// Convert to absolute path
		absDir, err := filepath.Abs(targetDir)
		if err != nil {
			log.Printf("Error resolving directory path: %v", err)
			println("Error resolving directory path:", err.Error())
			os.Exit(1)
		}
		log.Printf("Absolute directory path: %s", absDir)

		// Check if directory exists
		if _, err := os.Stat(absDir); os.IsNotExist(err) {
			log.Printf("Directory does not exist: %s", absDir)
			println("Directory does not exist:", absDir)
			os.Exit(1)
		}

		initialDir = absDir
		log.Printf("Using initial directory: %s", initialDir)
	} else {
		// Use current working directory if no argument provided
		cwd, err := os.Getwd()
		if err != nil {
			log.Printf("Error getting current directory: %v", err)
			println("Error getting current directory:", err.Error())
			os.Exit(1)
		}
		initialDir = cwd
		log.Printf("Using current working directory: %s", initialDir)
	}

	// Create an instance of the app structure
	log.Println("Creating app instance...")
	app := NewApp(initialDir)
	log.Println("App instance created successfully")

	// Create application menu
	log.Println("Setting up application menu...")
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
		log.Println("Application quit requested from menu")
	})

	// Create application with options
	log.Println("Starting Wails application...")
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
				case "Yes":
					if err := app.ShowSaveDialog(); err != nil {
						if err.Error() == "user_cancelled" {
							// User cancelled save dialog, prevent app close
							return true
						}
						runtime.LogError(ctx, "Save error: "+err.Error())
						return true // Prevent close on save error
					}
					return false // Allow close after successful save
				case "No":
					return false // Allow close without saving
				default: // "Cancel" or closed dialog
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
		log.Printf("Wails application error: %v", err)
		println("Error:", err.Error())
	} else {
		log.Println("Wails application closed successfully")
	}
}
