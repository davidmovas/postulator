package main

import (
	"context"
	"embed"
	"log"
	"os"

	"github.com/davidmovas/postulator/internal/app"
	"github.com/davidmovas/postulator/internal/config"
	"github.com/davidmovas/postulator/internal/infra/notifyicon"
	"github.com/davidmovas/postulator/internal/infra/window"

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

var appCtx context.Context

func onReady(ctx context.Context, app *app.App) {
	if err := app.Start(ctx); err != nil {
		log.Printf("Failed to start app: %v", err)
		log.Fatal(err)
	}

	systray.SetIcon(icoIcon)
	systray.SetTitle("Postulator")
	systray.SetTooltip("Postulator - Post Creator App")

	mShow := systray.AddMenuItem("Show", "Show the application")
	mHide := systray.AddMenuItem("Hide", "Hide the application")
	systray.AddSeparator()
	mQuit := systray.AddMenuItem("Quit", "Quit the whole app")

	go func() {
		for {
			select {
			case <-mShow.ClickedCh:
				if appCtx != nil {
					runtime.WindowShow(appCtx)
					window.SetWindowOpen(true)
				}
			case <-mHide.ClickedCh:
				if appCtx != nil {
					runtime.WindowHide(appCtx)
					window.SetWindowOpen(false)
				}
			case <-mQuit.ClickedCh:
				if appCtx != nil {
					runtime.Quit(appCtx)
					window.SetWindowOpen(false)
				}
				systray.Quit()
				os.Exit(0)
				return
			}
		}
	}()
}

func onExit(_ context.Context, app *app.App) {
	_ = app.Stop()
}

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatal(err)
	}

	notifyicon.SetTempIcons(icon)
	ctx, cancel := context.WithCancel(context.Background())

	appInst, err := app.New(cfg)
	if err != nil {
		log.Fatal(err)
	}

	go systray.Run(
		func() {
			onReady(ctx, appInst)
		},
		func() {
			cancel()
			onExit(ctx, appInst)
		},
	)

	err = wails.Run(&options.App{
		Title:         "Postulator",
		Width:         1280,
		Height:        720,
		MinWidth:      1024,
		MinHeight:     480,
		DisableResize: false,
		Fullscreen:    false,
		Frameless:     false,
		StartHidden:   false,
		// HideWindowOnClose: true,
		SingleInstanceLock: &options.SingleInstanceLock{
			UniqueId: "com.postulator.app",
			OnSecondInstanceLaunch: func(secondInstanceData options.SecondInstanceData) {
				if appCtx != nil {
					runtime.WindowShow(appCtx)
					runtime.WindowUnminimise(appCtx)
					runtime.WindowSetAlwaysOnTop(appCtx, true)
					runtime.WindowSetAlwaysOnTop(appCtx, false)
				}
			},
		},
		BackgroundColour: &options.RGBA{R: 255, G: 255, B: 255, A: 255},
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		Menu:     nil,
		Logger:   nil,
		LogLevel: logger.DEBUG,
		OnStartup: func(ctx context.Context) {
			appCtx = ctx
			appInst.SetWailsContext(ctx)
			window.SetWindowOpen(true)
		},
		OnBeforeClose: func(ctx context.Context) (prevent bool) {
			if ctx != nil {
				runtime.WindowHide(ctx)
			}
			window.SetWindowOpen(false)
			return true
		},
		OnShutdown: func(ctx context.Context) {
			window.SetWindowOpen(false)
			_ = appInst.Stop()
		},
		WindowStartState: options.Normal,
		Bind:             appInst.GetBinds(),
		DragAndDrop: &options.DragAndDrop{
			EnableFileDrop: true,
		},
		Windows: &windows.Options{
			WebviewIsTransparent: false,
			WindowIsTranslucent:  false,
			DisableWindowIcon:    false,
			ZoomFactor:           1.0,
		},
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
				Title: "Postulator",
				Icon:  icon,
			},
		},
	})
	if err != nil {
		log.Fatal(err)
	}
}
