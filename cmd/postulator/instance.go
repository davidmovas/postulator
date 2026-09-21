package main

import (
	"log"

	"github.com/wailsapp/wails/v3/pkg/application"

	"github.com/davidmovas/postulator/internal/app"
	"github.com/davidmovas/postulator/internal/application/events"
)

const ProductionInstanceID = "com.davidmovas.postulator"

type raisable interface {
	Restore()
	Show() application.Window
	Focus()
}

func onlyInstance(uniqueID string) *application.SingleInstanceOptions {
	return &application.SingleInstanceOptions{
		UniqueID: uniqueID,
		OnSecondInstanceLaunch: func(application.SecondInstanceData) {
			raise(firstWindow())
		},
	}
}

func raise(window raisable) {
	if window == nil {
		return
	}
	window.Restore()
	window.Show()
	window.Focus()
}

func firstWindow() raisable {
	running := application.Get()
	if running == nil {
		return nil
	}
	if current := running.Window.Current(); current != nil {
		return current
	}
	if found, ok := running.Window.GetByName(""); ok {
		return found
	}
	return nil
}

type publisher interface {
	Publish(eventType events.Type, payload any) error
}

func relayDrop(core *app.Core, dropped *application.WindowEvent) {
	publishDrop(core.Events, dropped.Context().DroppedFiles())
}

func publishDrop(relay publisher, paths []string) {
	if len(paths) == 0 {
		return
	}
	if err := relay.Publish(events.FilesDropped, events.FilesDroppedPayload{Paths: paths}); err != nil {
		log.Print(err)
	}
}
