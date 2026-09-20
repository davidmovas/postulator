package main

import (
	"github.com/wailsapp/wails/v3/pkg/application"
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
