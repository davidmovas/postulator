//go:build uiharness

package main

import (
	"context"
	"log"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"

	"github.com/davidmovas/postulator/internal/adapters/llm/fake"
	"github.com/davidmovas/postulator/internal/adapters/wp/wptest"
	"github.com/davidmovas/postulator/internal/app"
	port "github.com/davidmovas/postulator/internal/application/llm"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

const (
	devtoolsVariable = "POSTULATOR_DEVTOOLS_PORT"
	siteVariable     = "POSTULATOR_WP_ADDRESS"
	torHideVariable  = "POSTULATOR_TOR_HIDE"

	defaultSiteAddress = "127.0.0.1:9223"

	harnessUser     = "postulator"
	harnessPassword = "abcd EFGH 1234 ijkl"

	heldCallDelay = 45 * time.Second
)

type reporter struct {
	mu       sync.Mutex
	cleanups []func()
}

func (r *reporter) Helper() {}

func (r *reporter) Cleanup(fn func()) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.cleanups = append(r.cleanups, fn)
}

func (r *reporter) Errorf(format string, args ...any) {
	log.Printf("the fake site: "+format, args...)
}

func (r *reporter) Logf(format string, args ...any) {
	log.Printf("the fake site: "+format, args...)
}

type pacedProvider struct {
	inner   *fake.Scripted
	failing atomic.Pointer[string]
	held    atomic.Bool
}

func newPacedProvider() *pacedProvider {
	return &pacedProvider{inner: fake.NewScripted(harnessReplies()...)}
}

func (p *pacedProvider) failOn(match string) {
	p.failing.Store(&match)
}

func (p *pacedProvider) hold() {
	p.held.Store(true)
}

func (p *pacedProvider) gate(ctx context.Context, req port.Request) error {
	if req.Meta.RunID == "" {
		return nil
	}

	prompt := ""
	if len(req.Messages) > 0 {
		prompt = req.Messages[len(req.Messages)-1].Text
	}

	if match := p.failing.Load(); match != nil && *match != "" && strings.Contains(prompt, *match) {
		return errors.New(errors.Unauthorized, "the model provider rejected the key for this account").
			WithDetail("provider", req.Ref.Provider)
	}
	if !p.held.Load() {
		return nil
	}

	select {
	case <-ctx.Done():
		return errors.New(errors.Cancelled, "the model call was stopped").WithInternal(ctx.Err())
	case <-time.After(heldCallDelay):
		return nil
	}
}

func (p *pacedProvider) Complete(ctx context.Context, req port.Request) (port.Response, error) {
	if err := p.gate(ctx, req); err != nil {
		return port.Response{}, err
	}
	return p.inner.Complete(ctx, req)
}

func (p *pacedProvider) Stream(ctx context.Context, req port.Request) (<-chan port.Delta, error) {
	if err := p.gate(ctx, req); err != nil {
		return nil, err
	}
	return p.inner.Stream(ctx, req)
}

func configure(cfg app.Config) (harness, error) {
	fresh, err := absent(cfg.DatabasePath)
	if err != nil {
		return harness{}, err
	}

	site := wptest.New(&reporter{}, wptest.WithAddress(address()),
		wptest.WithClock(time.Now), wptest.WithCredentials(harnessUser, harnessPassword))

	provider := newPacedProvider()
	script := &assistantScript{}
	cfg.Provider = provider
	cfg.AgentProvider = fake.NewGollem(fake.WithScript(script.answer))
	cfg.Environment = environment()

	if !fresh {
		return harness{Config: cfg, Seed: func(ctx context.Context, core *app.Core) error {
			return repopulate(ctx, core, site)
		}}, nil
	}
	return harness{Config: cfg, Seed: func(ctx context.Context, core *app.Core) error {
		return seed(ctx, core, site.URL(), provider, script)
	}}, nil
}

func options(opts application.Options) application.Options {
	endpoint := strings.TrimSpace(os.Getenv(devtoolsVariable))
	if endpoint != "" {
		opts.Windows.AdditionalBrowserArgs = append(opts.Windows.AdditionalBrowserArgs,
			"--remote-debugging-port="+endpoint)
	}
	opts.SingleInstance = onlyInstance(harnessInstanceID(endpoint))
	return opts
}

func harnessInstanceID(endpoint string) string {
	if endpoint == "" {
		endpoint = "default"
	}
	return ProductionInstanceID + ".uiharness." + endpoint
}

func environment() func(string) string {
	if strings.TrimSpace(os.Getenv(torHideVariable)) != "1" {
		return os.Getenv
	}
	return func(string) string { return "" }
}

func address() string {
	if configured := strings.TrimSpace(os.Getenv(siteVariable)); configured != "" {
		return configured
	}
	return defaultSiteAddress
}

func absent(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return false, nil
	}
	if os.IsNotExist(err) {
		return true, nil
	}
	return false, errors.Wrap(err, errors.Internal, "look for the harness database")
}
