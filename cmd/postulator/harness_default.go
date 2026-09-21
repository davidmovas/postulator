//go:build !uiharness

package main

import (
	"github.com/wailsapp/wails/v3/pkg/application"

	"github.com/davidmovas/postulator/internal/app"
)

func configure(cfg app.Config) (harness, error) {
	return harness{Config: cfg}, nil
}

func options(opts application.Options) application.Options {
	tuned, _ := withDevtools(opts)
	tuned.SingleInstance = onlyInstance(ProductionInstanceID)
	return tuned
}
