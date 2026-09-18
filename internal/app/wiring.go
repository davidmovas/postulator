package app

import (
	"github.com/wailsapp/wails/v3/pkg/application"

	"github.com/davidmovas/postulator/internal/kernel/log"
	"github.com/davidmovas/postulator/internal/transport/wails"
)

func Logger() (*log.Logger, error) {
	return log.New(log.Config{
		Level:   "info",
		Format:  log.FormatConsole,
		Service: "postulator",
		Version: Version,
		Console: true,
	})
}

func (c *Core) Services(logger *log.Logger) []application.Service {
	return wails.Services(logger.Logger, wails.BuildInfo{
		Version:   Version,
		Commit:    Commit,
		BuildDate: BuildDate,
	}, wails.Deps{
		Sites:     c.Sites,
		Graph:     c.Graph,
		Pages:     c.Pages,
		Templates: c.Templates,
		Runs:      c.Runs,
		Sync:      c.Sync,
		Reports:   c.Reports,
		Imports:   c.Imports,
		Models:    c.Models,
		Agent:     c.Agent,
		Schedules: c.Schedules,
		Tools:     c.Tools,
		Settings: wails.SettingsDeps{
			Declarations: c.Declarations,
			Values:       c.Settings,
			Store:        c.SettingsStore,
			Models:       c.Models,
		},
	})
}
