package main

import (
	"context"
	_ "embed"
	"log"

	"github.com/wailsapp/wails/v3/pkg/application"
	wailsevents "github.com/wailsapp/wails/v3/pkg/events"

	"github.com/davidmovas/postulator/frontend"
	"github.com/davidmovas/postulator/internal/app"
	"github.com/davidmovas/postulator/internal/kernel/clock"
)

//go:embed appicon.png
var appIcon []byte

type harness struct {
	Config app.Config
	Seed   func(context.Context, *app.Core) error
}

func main() {
	if err := launch(); err != nil {
		log.Fatal(err)
	}
}

func launch() error {
	ctx := context.Background()

	cfg, err := app.DefaultConfig()
	if err != nil {
		return err
	}

	tuned, err := configure(cfg)
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

	core, err := app.Open(ctx, tuned.Config, logger.Logger)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := core.Close(); closeErr != nil {
			log.Print(closeErr)
		}
	}()

	if tuned.Seed != nil {
		if seedErr := tuned.Seed(ctx, core); seedErr != nil {
			return seedErr
		}
	}

	wails := application.New(options(application.Options{
		Name:        "Postulator",
		Description: "Entity-graph driven WordPress content factory",
		Icon:        appIcon,
		Services:    core.Services(logger),
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(frontend.Assets),
		},
	}))
	if connectErr := core.Events.Connect(wails.Event, clock.System{}); connectErr != nil {
		return connectErr
	}

	window := wails.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:            "Postulator",
		Width:            1280,
		Height:           820,
		MinWidth:         960,
		MinHeight:        600,
		Frameless:        true,
		EnableFileDrop:   true,
		BackgroundColour: application.NewRGB(14, 16, 22),
		URL:              "/",
	})
	window.OnWindowEvent(wailsevents.Common.WindowFilesDropped, func(dropped *application.WindowEvent) {
		relayDrop(core, dropped)
	})

	return wails.Run()
}
