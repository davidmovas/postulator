package main

import (
	"Postulator/internal/app"
	"Postulator/internal/config"
	"context"
	"embed"
	"log"
	"os"

	"github.com/getlantern/systray"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/logger"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/mac"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

//go:embed all:frontend/out
var assets embed.FS

//go:embed build/appicon.png
var icon []byte

//go:embed build/icon.ico
var icoIcon []byte

var wailsCtx context.Context

func onReady() {
	systray.SetIcon(icoIcon)
	systray.SetTitle("Postulator")
	systray.SetTooltip("Postulator - Post Creator App")

	// Menu items
	mShow := systray.AddMenuItem("Show", "Show the application")
	mHide := systray.AddMenuItem("Hide", "Hide the application")
	systray.AddSeparator()
	mQuit := systray.AddMenuItem("Quit", "Quit the whole app")

	// Handle menu clicks
	go func() {
		for {
			select {
			case <-mShow.ClickedCh:
				if wailsCtx != nil {
					runtime.WindowShow(wailsCtx)
				}
			case <-mHide.ClickedCh:
				if wailsCtx != nil {
					runtime.WindowHide(wailsCtx)
				}
			case <-mQuit.ClickedCh:
				if wailsCtx != nil {
					runtime.Quit(wailsCtx)
				}
				systray.Quit()
				os.Exit(0)
				return
			}
		}
	}()
}

func onExit() {
	// Clean up here
}

func main() {
	// Load or create default config
	cfg := &config.Config{LogLevel: "info", ConsoleOut: true, PrettyPrint: false}

	// Init internal app
	appInst, err := app.New(cfg)
	if err != nil {
		log.Fatal(err)
	}

	// Initialize systray in a goroutine
	go systray.Run(onReady, onExit)

	// Create application with options
	err = wails.Run(&options.App{
		Title:             "Postulator",
		Width:             1024,
		Height:            768,
		MinWidth:          1024,
		MinHeight:         768,
		DisableResize:     false,
		Fullscreen:        false,
		Frameless:         false,
		StartHidden:       false,
		HideWindowOnClose: true,
		SingleInstanceLock: &options.SingleInstanceLock{
			UniqueId: "com.postulator.app",
			OnSecondInstanceLaunch: func(secondInstanceData options.SecondInstanceData) {
				if wailsCtx != nil {
					runtime.WindowShow(wailsCtx)
					runtime.WindowUnminimise(wailsCtx)
					runtime.WindowSetAlwaysOnTop(wailsCtx, true)
					runtime.WindowSetAlwaysOnTop(wailsCtx, false)
				}
			},
		},
		BackgroundColour: &options.RGBA{R: 255, G: 255, B: 255, A: 255},
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		Menu:             nil,
		Logger:           nil,
		LogLevel:         logger.DEBUG,
		OnStartup:        func(ctx context.Context) { wailsCtx = ctx },
		OnShutdown:       func(ctx context.Context) { appInst.Stop() },
		WindowStartState: options.Normal,
		Bind: []interface{}{
			appInst,
		},
		DragAndDrop: &options.DragAndDrop{
			EnableFileDrop: true,
		},
		// Windows platform specific options
		Windows: &windows.Options{
			WebviewIsTransparent: false,
			WindowIsTranslucent:  false,
			DisableWindowIcon:    false,
			ZoomFactor:           1.0,
		},
		// Mac platform specific options
		Mac: &mac.Options{
			TitleBar: &mac.TitleBar{
				TitlebarAppearsTransparent: true,
				HideTitle:                  false,
				HideTitleBar:               false,
				FullSizeContent:            false,
				UseToolbar:                 false,
				HideToolbarSeparator:       true,
			},
			Appearance:           mac.DefaultAppearance,
			WebviewIsTransparent: true,
			WindowIsTranslucent:  true,
			About: &mac.AboutInfo{
				Title:   "Postulator",
				Message: "",
				Icon:    icon,
			},
		},
	})

	if err != nil {
		log.Fatal(err)
	}
}
