package main

import (
	"context"
	"embed"
	"errors"
	"flag"
	"log"
	"os"

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

	// Parse command line arguments: a folder, or a file to select in its folder
	// (e.g. from the Explorer context menu)
	flag.Parse()
	initialDir, initialFile, err := resolveStartupPath(flag.Arg(0))
	if err != nil {
		log.Printf("Startup error: %v", err)
		showStartupError(err)
		os.Exit(1)
	}

	// Create an instance of the app structure
	app := NewApp(initialDir, initialFile)

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

	// Explorer context menu ("PDF Previewで開く"), registered per user only
	// when turned on here
	settingsMenu := appMenu.AddSubmenu("設定")
	settingsMenu.AddCheckbox("エクスプローラーの右クリックメニューに追加", app.IsContextMenuEnabled(), nil, func(data *menu.CallbackData) {
		// The checkbox has already been toggled when this is called
		item := data.MenuItem
		if err := app.SetContextMenuEnabled(item.Checked); err != nil {
			item.SetChecked(!item.Checked)
			runtime.MenuUpdateApplicationMenu(app.ctx)
			runtime.MessageDialog(app.ctx, runtime.MessageDialogOptions{
				Type:    runtime.ErrorDialog,
				Title:   "右クリックメニュー",
				Message: "右クリックメニューを変更できませんでした: " + err.Error(),
			})
		}
	})

	// Create application with options
	err = wails.Run(&options.App{
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
