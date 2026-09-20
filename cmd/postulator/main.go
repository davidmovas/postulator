package main

import (
	"context"
	_ "embed"
	"log"

	"github.com/wailsapp/wails/v3/pkg/application"

	"github.com/davidmovas/postulator/frontend"
	"github.com/davidmovas/postulator/internal/app"
	"github.com/davidmovas/postulator/internal/kernel/clock"
)

//go:embed appicon.png
var appIcon []byte

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

	logger, err := app.Logger()
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := logger.Close(); closeErr != nil {
			log.Print(closeErr)
		}
	}()

	core, err := app.Open(context.Background(), cfg, logger.Logger)
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
		Icon:        appIcon,
		Services:    core.Services(logger),
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(frontend.Assets),
		},
	})
	if connectErr := core.Events.Connect(wails.Event, clock.System{}); connectErr != nil {
		return connectErr
	}

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
