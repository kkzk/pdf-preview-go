package main

import (
	"embed"
	"flag"
	"os"
	"path/filepath"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	// Parse command line arguments
	var testDir string
	flag.StringVar(&testDir, "test", "", "Use specified test directory")
	flag.Parse()

	// If test directory is specified, use it; otherwise use current directory
	var initialDir string
	if testDir != "" {
		// Convert to absolute path
		absTestDir, err := filepath.Abs(testDir)
		if err != nil {
			println("Error resolving test directory path:", err.Error())
			os.Exit(1)
		}

		// Check if directory exists
		if _, err := os.Stat(absTestDir); os.IsNotExist(err) {
			println("Test directory does not exist:", absTestDir)
			os.Exit(1)
		}

		initialDir = absTestDir
	} else {
		// Use current working directory
		cwd, err := os.Getwd()
		if err != nil {
			println("Error getting current directory:", err.Error())
			os.Exit(1)
		}
		initialDir = cwd
	}

	// Create an instance of the app structure
	app := NewApp(initialDir)

	// Create application with options
	err := wails.Run(&options.App{
		Title:  "pdf-preview-go",
		Width:  1024,
		Height: 768,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 27, G: 38, B: 54, A: 1},
		OnStartup:        app.startup,
		Bind: []interface{}{
			app,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
