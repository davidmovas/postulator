package handlers

import (
	"context"
	"os"

	"github.com/davidmovas/postulator/internal/dto"
	"github.com/getlantern/systray"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// AppHandler handles application-level operations like quit
type AppHandler struct {
	ctx context.Context
}

func NewAppHandler() *AppHandler {
	return &AppHandler{}
}

// SetContext sets the Wails runtime context
func (h *AppHandler) SetContext(ctx context.Context) {
	h.ctx = ctx
}

// QuitApp gracefully exits the application
func (h *AppHandler) QuitApp() *dto.Response[string] {
	go func() {
		if h.ctx != nil {
			runtime.Quit(h.ctx)
		}
		systray.Quit()
		os.Exit(0)
	}()

	return ok("Application is shutting down")
}
