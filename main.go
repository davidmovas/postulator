package main

import (
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

var appInstance *App
var appContext context.Context

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
				if appContext != nil {
					runtime.WindowShow(appContext)
				}
			case <-mHide.ClickedCh:
				if appContext != nil {
					runtime.WindowHide(appContext)
				}
			case <-mQuit.ClickedCh:
				if appContext != nil {
					runtime.Quit(appContext)
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
	// Create an instance of the app structure
	app := NewApp()
	appInstance = app

	// Initialize systray in a goroutine
	go systray.Run(onReady, onExit)

	// Create application with options
	err := wails.Run(&options.App{
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
			UniqueId: "com.mycompany.postulator",
			OnSecondInstanceLaunch: func(secondInstanceData options.SecondInstanceData) {
				if appContext != nil {
					runtime.WindowShow(appContext)                  // Показываем окно, если оно было скрыто
					runtime.WindowUnminimise(appContext)            // Восстанавливаем из свернутого состояния
					runtime.WindowSetAlwaysOnTop(appContext, true)  // Временно делаем поверх всех окон
					runtime.WindowSetAlwaysOnTop(appContext, false) // Убираем флаг поверх всех окон
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
		OnStartup:        app.startup,
		OnDomReady:       app.domReady,
		OnBeforeClose:    app.beforeClose,
		OnShutdown:       app.shutdown,
		WindowStartState: options.Normal,
		Bind: []interface{}{
			app,
			app.binder,
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
