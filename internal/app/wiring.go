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

func live[T any](c *Core, pick func(*Core) T) wails.Source[T] {
	return func() (T, error) {
		c.mu.RLock()
		defer c.mu.RUnlock()

		if c.Store == nil {
			var zero T
			return zero, locked()
		}
		return pick(c), nil
	}
}

func (c *Core) Services(logger *log.Logger) []application.Service {
	return wails.Services(logger.Logger, wails.BuildInfo{
		Version:   Version,
		Commit:    Commit,
		BuildDate: BuildDate,
	}, wails.Deps{
		Sites:     live(c, func(c *Core) wails.SitesUseCase { return c.Sites }),
		Graph:     live(c, func(c *Core) wails.GraphUseCase { return c.Graph }),
		Pages:     live(c, func(c *Core) wails.PagesUseCase { return c.Pages }),
		Templates: live(c, func(c *Core) wails.TemplatesUseCase { return c.Templates }),
		Runs:      live(c, func(c *Core) wails.RunsUseCase { return c.Runs }),
		Sync:      live(c, func(c *Core) wails.SyncUseCase { return c.Sync }),
		Reports:   live(c, func(c *Core) wails.ReportsUseCase { return c.Reports }),
		Imports:   live(c, func(c *Core) wails.ImportsUseCase { return c.Imports }),
		Models:    live(c, func(c *Core) wails.ModelsUseCase { return c.Models }),
		Agent:     live(c, func(c *Core) wails.AgentUseCase { return c.Agent }),
		Schedules: live(c, func(c *Core) wails.SchedulesUseCase { return c.Schedules }),
		Tools:     live(c, func(c *Core) wails.ToolCatalog { return c.Tools }),
		Settings: wails.SettingsDeps{
			Access: live(c, func(c *Core) wails.SettingsAccess {
				return wails.SettingsAccess{
					Declarations: c.Declarations,
					Values:       c.Settings,
					Store:        c.SettingsStore,
					Models:       c.Models,
				}
			}),
			Backup: live(c, func(c *Core) wails.BackupControl { return c }),
			Lock:   c,
		},
	})
}
