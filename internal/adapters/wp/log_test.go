package wp_test

import (
	"encoding/base64"
	"fmt"
	"maps"
	"net/http"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/davidmovas/postulator/internal/adapters/wp"
	"github.com/davidmovas/postulator/internal/adapters/wp/wptest"
)

type recordingCore struct {
	zapcore.LevelEnabler
	mu      *sync.Mutex
	entries *[]string
}

func newRecordingCore() (core zapcore.Core, recorded func() []string) {
	var guard sync.Mutex
	entries := make([]string, 0)

	return &recordingCore{LevelEnabler: zapcore.DebugLevel, mu: &guard, entries: &entries}, func() []string {
		guard.Lock()
		defer guard.Unlock()
		return slices.Clone(entries)
	}
}

func (c *recordingCore) With([]zapcore.Field) zapcore.Core {
	return c
}

func (c *recordingCore) Check(entry zapcore.Entry, checked *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	if c.Enabled(entry.Level) {
		return checked.AddCore(entry, c)
	}
	return checked
}

func (c *recordingCore) Write(entry zapcore.Entry, fields []zapcore.Field) error {
	encoder := zapcore.NewMapObjectEncoder()
	for _, field := range fields {
		field.AddTo(encoder)
	}

	var builder strings.Builder
	builder.WriteString(entry.Message)
	for _, key := range slices.Sorted(maps.Keys(encoder.Fields)) {
		builder.WriteString(" ")
		builder.WriteString(key)
		builder.WriteString("=")
		fmt.Fprint(&builder, encoder.Fields[key])
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	*c.entries = append(*c.entries, builder.String())
	return nil
}

func (c *recordingCore) Sync() error {
	return nil
}

func TestTheApplicationPasswordNeverReachesALogLine(t *testing.T) {
	t.Parallel()

	const (
		user     = "editor"
		password = "s3cret app pass"
		body     = "<p>Koffein ist ein Alkaloid.</p>"
	)

	server := wptest.New(t, wptest.WithCredentials(user, password))
	seeded := server.Seed(wptest.Item{Type: wptest.TypePage, Title: "Koffein", Content: body})

	core, recorded := newRecordingCore()
	client, err := wp.New(wp.Config{
		BaseURL:       server.URL(),
		Username:      user,
		AppPassword:   password,
		AllowInsecure: true,
	},
		wp.WithRateLimit(0),
		wp.WithBackoff(func(int) time.Duration { return 0 }),
		wp.WithLogger(zap.New(core)),
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	if _, err = client.Probe(t.Context()); err != nil {
		t.Fatalf("Probe: %v", err)
	}
	if _, err = client.GetItem(t.Context(), wp.TypePage, seeded[0].ID); err != nil {
		t.Fatalf("GetItem: %v", err)
	}

	server.FailNext(http.StatusForbidden, 1)
	if _, err = client.GetItem(t.Context(), wp.TypePage, seeded[0].ID); err == nil {
		t.Fatal("the injected fault must reach the caller")
	}

	lines := recorded()
	if len(lines) != 3 {
		t.Fatalf("the adapter logged %d lines, want one per request", len(lines))
	}

	secrets := []string{
		password,
		base64.StdEncoding.EncodeToString([]byte(user + ":" + password)),
		"Basic ",
		body,
	}
	for _, line := range lines {
		for _, secret := range secrets {
			if strings.Contains(line, secret) {
				t.Errorf("a log line leaked %q: %s", secret, line)
			}
		}
	}
}

func TestALogLineCarriesTheRequestShapeAndTheWordPressCode(t *testing.T) {
	t.Parallel()

	server := wptest.New(t)
	core, recorded := newRecordingCore()

	client, err := wp.New(wp.Config{
		BaseURL:       server.URL(),
		Username:      wptest.DefaultUser,
		AppPassword:   wptest.DefaultPassword,
		AllowInsecure: true,
	},
		wp.WithRateLimit(0),
		wp.WithRetries(0),
		wp.WithLogger(zap.New(core)),
	)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	if _, err = client.Probe(t.Context()); err != nil {
		t.Fatalf("Probe: %v", err)
	}

	server.FailNext(http.StatusForbidden, 1)
	if _, err = client.GetItem(t.Context(), wp.TypePage, 1); err == nil {
		t.Fatal("the injected fault must reach the caller")
	}

	lines := recorded()
	if len(lines) != 2 {
		t.Fatalf("the adapter logged %d lines, want 2", len(lines))
	}

	for _, wanted := range []string{"method=GET", "path=/wp-json", "status=200", "durationMs="} {
		if !strings.Contains(lines[0], wanted) {
			t.Errorf("the success line %q is missing %q", lines[0], wanted)
		}
	}
	for _, wanted := range []string{"status=403", "code=UNAUTHORIZED", "wpCode=internal_server_error"} {
		if !strings.Contains(lines[1], wanted) {
			t.Errorf("the failure line %q is missing %q", lines[1], wanted)
		}
	}
}
