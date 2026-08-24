package main

import (
	"embed"
	"log"

	mainapp "voixpe3per/desktop/main"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	app := mainapp.NewApp()

	err := wails.Run(&options.App{
		Title:     "PeeperPhone",
		Width:     1220,
		Height:    760,
		MinWidth:  980,
		MinHeight: 620,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 7, G: 8, B: 7, A: 1},
		OnStartup:        app.OnStartup,
		OnShutdown:       app.OnShutdown,
		Bind: []interface{}{
			app,
		},
	})
	if err != nil {
		log.Fatal(err)
	}
}
