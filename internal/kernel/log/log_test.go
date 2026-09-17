package log_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/davidmovas/postulator/internal/kernel/log"
)

func readLines(t *testing.T, path string) []map[string]any {
	t.Helper()

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}

	var entries []map[string]any
	for _, line := range strings.Split(strings.TrimSpace(string(raw)), "\n") {
		if line == "" {
			continue
		}
		var entry map[string]any
		if err = json.Unmarshal([]byte(line), &entry); err != nil {
			t.Fatalf("parse %s: %v", path, err)
		}
		entries = append(entries, entry)
	}
	return entries
}

func TestFileSinks(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	logger, err := log.New(log.Config{
		Level:   "debug",
		Format:  log.FormatJSON,
		Service: "postulator",
		Version: "2.0.0",
		File:    &log.FileConfig{Dir: dir, Enabled: true},
	})
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	if logger.Level.Level() != zapcore.DebugLevel {
		t.Fatalf("level = %v, want debug", logger.Level.Level())
	}

	logger.Info("hello")
	logger.Error("boom")
	if err = logger.Close(); err != nil {
		t.Fatalf("Close() error: %v", err)
	}

	app := readLines(t, filepath.Join(dir, log.AppFile))
	if len(app) != 2 {
		t.Fatalf("app.log has %d entries, want 2", len(app))
	}
	if app[0]["service"] != "postulator" || app[0]["version"] != "2.0.0" {
		t.Fatalf("default fields missing: %v", app[0])
	}

	failures := readLines(t, filepath.Join(dir, log.ErrorFile))
	if len(failures) != 1 {
		t.Fatalf("errors.log has %d entries, want 1", len(failures))
	}
	if failures[0]["m"] != "boom" {
		t.Fatalf("errors.log entry = %v", failures[0])
	}
}

func TestRedaction(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		field zap.Field
	}{
		{name: "password", field: zap.String("password", "hunter2")},
		{name: "apiKey", field: zap.String("apiKey", "hunter2")},
		{name: "token", field: zap.String("token", "hunter2")},
		{name: "authorization", field: zap.String("authorization", "hunter2")},
		{name: "case insensitive", field: zap.String("Authorization", "hunter2")},
		{name: "non string value", field: zap.Any("token", map[string]string{"a": "hunter2"})},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			dir := t.TempDir()
			logger, err := log.New(log.Config{
				Level:  "info",
				Format: log.FormatJSON,
				File:   &log.FileConfig{Dir: dir, Enabled: true},
			})
			if err != nil {
				t.Fatalf("New() error: %v", err)
			}

			logger.Info("call", tc.field)
			logger.With(tc.field).Info("with")
			if err = logger.Close(); err != nil {
				t.Fatalf("Close() error: %v", err)
			}

			raw, err := os.ReadFile(filepath.Join(dir, log.AppFile))
			if err != nil {
				t.Fatalf("read app.log: %v", err)
			}
			if strings.Contains(string(raw), "hunter2") {
				t.Fatalf("secret leaked into the log: %s", raw)
			}
			if !strings.Contains(string(raw), log.Mask) {
				t.Fatalf("redaction mask missing: %s", raw)
			}
		})
	}
}

func TestUnredactedFieldsSurvive(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	logger, err := log.New(log.Config{Level: "info", Format: log.FormatJSON, File: &log.FileConfig{Dir: dir, Enabled: true}})
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}

	logger.Info("call", zap.String("siteId", "site-1"), zap.Int("pages", 12))
	if err = logger.Close(); err != nil {
		t.Fatalf("Close() error: %v", err)
	}

	entries := readLines(t, filepath.Join(dir, log.AppFile))
	if entries[0]["siteId"] != "site-1" {
		t.Fatalf("siteId = %v, want site-1", entries[0]["siteId"])
	}
	if entries[0]["pages"] != float64(12) {
		t.Fatalf("pages = %v, want 12", entries[0]["pages"])
	}
}

func TestRedactedKeys(t *testing.T) {
	t.Parallel()

	want := map[string]struct{}{"password": {}, "apikey": {}, "token": {}, "authorization": {}}
	got := log.RedactedKeys()
	if len(got) != len(want) {
		t.Fatalf("RedactedKeys() = %v", got)
	}
	for _, key := range got {
		if _, ok := want[strings.ToLower(key)]; !ok {
			t.Fatalf("RedactedKeys() contains unexpected %q", key)
		}
	}
}

func TestLevels(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		in   string
		want zapcore.Level
	}{
		{name: "debug", in: "debug", want: zapcore.DebugLevel},
		{name: "info", in: "info", want: zapcore.InfoLevel},
		{name: "warn", in: "warn", want: zapcore.WarnLevel},
		{name: "error", in: "error", want: zapcore.ErrorLevel},
		{name: "fatal", in: "fatal", want: zapcore.FatalLevel},
		{name: "unknown falls back to info", in: "chatty", want: zapcore.InfoLevel},
		{name: "empty falls back to info", in: "", want: zapcore.InfoLevel},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			logger, err := log.New(log.Config{Level: tc.in, Console: true})
			if err != nil {
				t.Fatalf("New() error: %v", err)
			}
			if logger.Level.Level() != tc.want {
				t.Fatalf("level = %v, want %v", logger.Level.Level(), tc.want)
			}
			if err = logger.Close(); err != nil {
				t.Fatalf("Close() error: %v", err)
			}
		})
	}
}

func TestConsoleEncoderSwitch(t *testing.T) {
	t.Parallel()

	for _, format := range []string{log.FormatJSON, log.FormatConsole} {
		t.Run(format, func(t *testing.T) {
			t.Parallel()
			logger, err := log.New(log.Config{Format: format, Console: true})
			if err != nil {
				t.Fatalf("New() error: %v", err)
			}
			logger.Info("rendered")
			if err = logger.Close(); err != nil {
				t.Fatalf("Close() error: %v", err)
			}
		})
	}
}

func TestNewRejectsUnwritableDirectory(t *testing.T) {
	t.Parallel()

	file := filepath.Join(t.TempDir(), "occupied")
	if err := os.WriteFile(file, []byte("x"), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}

	if _, err := log.New(log.Config{File: &log.FileConfig{Dir: file, Enabled: true}}); err == nil {
		t.Fatal("New() must fail when the log directory cannot be created")
	}
}

func TestDisabledFileSink(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	logger, err := log.New(log.Config{File: &log.FileConfig{Dir: dir}, Console: true})
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	logger.Info("nowhere")
	if err = logger.Close(); err != nil {
		t.Fatalf("Close() error: %v", err)
	}

	if _, err = os.Stat(filepath.Join(dir, log.AppFile)); !os.IsNotExist(err) {
		t.Fatalf("a disabled file sink must not create %s", log.AppFile)
	}
}
