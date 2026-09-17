package main

import (
	"log"

	"github.com/wailsapp/wails/v3/pkg/application"

	"github.com/davidmovas/postulator/frontend"
	"github.com/davidmovas/postulator/internal/app"
)

func main() {
	wails := application.New(application.Options{
		Name:        "Postulator",
		Description: "Entity-graph driven WordPress content factory",
		Services: []application.Service{
			application.NewService(app.NewHealthService()),
		},
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(frontend.Assets),
		},
	})

	wails.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:            "Postulator",
		Width:            1280,
		Height:           820,
		MinWidth:         960,
		MinHeight:        600,
		BackgroundColour: application.NewRGB(14, 16, 22),
		URL:              "/",
	})

	if err := wails.Run(); err != nil {
		log.Fatal(err)
	}
}
