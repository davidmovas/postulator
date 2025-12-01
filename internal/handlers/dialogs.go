package handlers

import (
	"context"

	"github.com/davidmovas/postulator/internal/dto"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// DialogsHandler handles native OS dialogs
type DialogsHandler struct {
	ctx context.Context
}

func NewDialogsHandler() *DialogsHandler {
	return &DialogsHandler{}
}

// SetContext sets the Wails runtime context (called from main.go OnStartup)
func (h *DialogsHandler) SetContext(ctx context.Context) {
	h.ctx = ctx
}

// OpenFileDialog opens a native file picker dialog
func (h *DialogsHandler) OpenFileDialog(title string, filters []dto.FileFilter) *dto.Response[string] {
	if h.ctx == nil {
		return fail[string](nil)
	}

	// Convert DTO filters to Wails filters
	var wailsFilters []runtime.FileFilter
	for _, f := range filters {
		wailsFilters = append(wailsFilters, runtime.FileFilter{
			DisplayName: f.DisplayName,
			Pattern:     f.Pattern,
		})
	}

	path, err := runtime.OpenFileDialog(h.ctx, runtime.OpenDialogOptions{
		Title:   title,
		Filters: wailsFilters,
	})
	if err != nil {
		return fail[string](err)
	}

	return ok(path)
}

// OpenMultipleFilesDialog opens a native file picker dialog that allows multiple file selection
func (h *DialogsHandler) OpenMultipleFilesDialog(title string, filters []dto.FileFilter) *dto.Response[[]string] {
	if h.ctx == nil {
		return fail[[]string](nil)
	}

	var wailsFilters []runtime.FileFilter
	for _, f := range filters {
		wailsFilters = append(wailsFilters, runtime.FileFilter{
			DisplayName: f.DisplayName,
			Pattern:     f.Pattern,
		})
	}

	paths, err := runtime.OpenMultipleFilesDialog(h.ctx, runtime.OpenDialogOptions{
		Title:   title,
		Filters: wailsFilters,
	})
	if err != nil {
		return fail[[]string](err)
	}

	return ok(paths)
}

// OpenDirectoryDialog opens a native directory picker dialog
func (h *DialogsHandler) OpenDirectoryDialog(title string) *dto.Response[string] {
	if h.ctx == nil {
		return fail[string](nil)
	}

	path, err := runtime.OpenDirectoryDialog(h.ctx, runtime.OpenDialogOptions{
		Title: title,
	})
	if err != nil {
		return fail[string](err)
	}

	return ok(path)
}

// SaveFileDialog opens a native save file dialog
func (h *DialogsHandler) SaveFileDialog(title string, defaultFilename string, filters []dto.FileFilter) *dto.Response[string] {
	if h.ctx == nil {
		return fail[string](nil)
	}

	var wailsFilters []runtime.FileFilter
	for _, f := range filters {
		wailsFilters = append(wailsFilters, runtime.FileFilter{
			DisplayName: f.DisplayName,
			Pattern:     f.Pattern,
		})
	}

	path, err := runtime.SaveFileDialog(h.ctx, runtime.SaveDialogOptions{
		Title:           title,
		DefaultFilename: defaultFilename,
		Filters:         wailsFilters,
	})
	if err != nil {
		return fail[string](err)
	}

	return ok(path)
}
