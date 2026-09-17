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
	})
}
