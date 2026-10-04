package main

import (
	"context"
	"embed"
	"errors"
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
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Println("Starting PDF Preview Go application...")

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
			log.Fatalf("Error resolving directory path: %v", err)
		}

		// Check if directory exists
		if _, err := os.Stat(absDir); os.IsNotExist(err) {
			log.Fatalf("Directory does not exist: %s", absDir)
		}

		initialDir = absDir
	} else {
		// Use current working directory if no argument provided
		cwd, err := os.Getwd()
		if err != nil {
			log.Fatalf("Error getting current directory: %v", err)
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
		if err := app.ShowSaveDialog(); err != nil && !errors.Is(err, ErrUserCancelled) {
			runtime.LogError(app.ctx, err.Error())
		}
	})
	fileMenu.AddSeparator()
	fileMenu.AddText("終了", keys.CmdOrCtrl("q"), func(_ *menu.CallbackData) {
		// Goes through OnBeforeClose, so unsaved changes are confirmed
		runtime.Quit(app.ctx)
	})

	// Create application with options
	err := wails.Run(&options.App{
		Title:  "pdf-preview-go",
		Width:  1024,
		Height: 768,
		AssetServer: &assetserver.Options{
			Assets:  assets,
			Handler: app.pdfHandler(),
		},
		Menu:             appMenu,
		BackgroundColour: &options.RGBA{R: 27, G: 38, B: 54, A: 1},
		OnStartup:        app.Startup,
		OnShutdown:       app.Shutdown,
		OnBeforeClose: func(ctx context.Context) (prevent bool) {
			if app.HasUnsavedChanges() {
				switch confirmUnsavedChanges() {
				case CloseSave:
					if err := app.ShowSaveDialog(); err != nil {
						if !errors.Is(err, ErrUserCancelled) {
							runtime.LogError(ctx, "Save error: "+err.Error())
						}
						return true // Prevent close if save was cancelled or failed
					}
					return false // Allow close after successful save
				case CloseDiscard:
					return false // Allow close without saving
				default:
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
	}
}
