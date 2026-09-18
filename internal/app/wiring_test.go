package app_test

import (
	"context"
	"path/filepath"
	"testing"

	"go.uber.org/zap/zaptest"

	"github.com/davidmovas/postulator/internal/app"
	"github.com/davidmovas/postulator/internal/transport/wails"
)

func TestLoggerBuildsAConsoleLoggerThatCloses(t *testing.T) {
	t.Parallel()

	logger, err := app.Logger()
	if err != nil {
		t.Fatalf("Logger: %v", err)
	}
	if logger.Logger == nil {
		t.Fatal("Logger returned a logger with no zap core")
	}
	if err = logger.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
}

func TestServicesCarryTheInjectedStamps(t *testing.T) {
	t.Parallel()

	home := t.TempDir()
	core, err := app.Open(t.Context(), app.Config{
		DatabasePath: filepath.Join(home, "postulator.db"),
		KeyDir:       home,
	}, zaptest.NewLogger(t))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() {
		if closeErr := core.Close(); closeErr != nil {
			t.Errorf("Close: %v", closeErr)
		}
	})

	logger, err := app.Logger()
	if err != nil {
		t.Fatalf("Logger: %v", err)
	}
	t.Cleanup(func() {
		if closeErr := logger.Close(); closeErr != nil {
			t.Errorf("Close: %v", closeErr)
		}
	})

	services := core.Services(logger)
	if len(services) != 1 {
		t.Fatalf("Services returned %d services, want 1", len(services))
	}

	health, ok := services[0].Instance().(*wails.HealthService)
	if !ok {
		t.Fatalf("Services()[0] is %T, want *wails.HealthService", services[0].Instance())
	}

	build, err := health.Ping(context.Background(), wails.PingRequest{})
	if err != nil {
		t.Fatalf("Ping: %v", err)
	}
	if build.Version != app.Version || build.Commit != app.Commit || build.BuildDate != app.BuildDate {
		t.Fatalf("Ping = %+v, want the ldflags stamps %q/%q/%q", build, app.Version, app.Commit, app.BuildDate)
	}
}
