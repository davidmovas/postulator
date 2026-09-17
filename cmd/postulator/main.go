package main

import (
	"context"
	"log"

	"github.com/wailsapp/wails/v3/pkg/application"

	"github.com/davidmovas/postulator/frontend"
	"github.com/davidmovas/postulator/internal/app"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	cfg, err := app.DefaultConfig()
	if err != nil {
		return err
	}

	core, err := app.Open(context.Background(), cfg)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := core.Close(); closeErr != nil {
			log.Print(closeErr)
		}
	}()

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

	return wails.Run()
}
