package app

import (
	"context"
	stderrors "errors"
	"os"
	"path/filepath"

	"github.com/davidmovas/postulator/internal/adapters/secrets"
	"github.com/davidmovas/postulator/internal/adapters/secrets/masterkey"
	"github.com/davidmovas/postulator/internal/adapters/sqlite"
	"github.com/davidmovas/postulator/internal/application/graph"
	"github.com/davidmovas/postulator/internal/application/pages"
	"github.com/davidmovas/postulator/internal/application/sites"
	"github.com/davidmovas/postulator/internal/application/templates"
	"github.com/davidmovas/postulator/internal/kernel/clock"
	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/kernel/settings"
)

const (
	homeDirectory = "Postulator"
	databaseFile  = "postulator.db"
)

type Config struct {
	DatabasePath string
	KeyDir       string
}

func (c Config) recovery() string {
	return "remove " + filepath.Join(c.KeyDir, masterkey.FileName) + " and " + c.DatabasePath +
		" to reset the application state"
}

func DefaultConfig() (Config, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return Config{}, errors.Wrap(err, errors.Internal, "locate the user configuration directory")
	}

	home := filepath.Join(base, homeDirectory)
	return Config{DatabasePath: filepath.Join(home, databaseFile), KeyDir: home}, nil
}

type Core struct {
	Store           *sqlite.Store
	Secrets         *secrets.Store
	Settings        *settings.Values
	UnknownSettings []string
	Events          *EventRelay
	Sites           *sites.Service
	Graph           *graph.Service
	Pages           *pages.Service
	Templates       *templates.Service
}

func Open(ctx context.Context, cfg Config) (*Core, error) {
	if cfg.DatabasePath == "" {
		return nil, errors.New(errors.Invalid, "the database path must not be empty")
	}
	if cfg.KeyDir == "" {
		return nil, errors.New(errors.Invalid, "the key directory must not be empty")
	}

	recovery := cfg.recovery()

	key, err := masterkey.Load(masterkey.Config{Dir: cfg.KeyDir, Recovery: recovery})
	if err != nil {
		return nil, err
	}

	store, err := sqlite.Open(sqlite.Config{Path: cfg.DatabasePath, Key: key, Recovery: recovery})
	if err != nil {
		return nil, err
	}

	now := clock.System{}
	values, unknown, err := LoadSettings(ctx, sqlite.NewSettingsRepo(store, now), settings.Default())
	if err != nil {
		return nil, stderrors.Join(err, store.Close())
	}

	secretStore := secrets.NewStore(sqlite.NewSecretsRepo(store, now), key)
	relay := &EventRelay{}
	siteRepo := sqlite.NewSiteRepo(store)
	entityRepo := sqlite.NewEntityRepo(store)
	edgeRepo := sqlite.NewEdgeRepo(store)
	pageRepo := sqlite.NewPageRepo(store)
	linkRepo := sqlite.NewPageLinkRepo(store)
	templateRepo := sqlite.NewTemplateRepo(store)
	policyRepo := sqlite.NewLinkPolicyRepo(store)

	core := &Core{
		Store:           store,
		Secrets:         secretStore,
		Settings:        values,
		UnknownSettings: unknown,
		Events:          relay,
		Sites:           sites.New(siteRepo, secretStore, store, now),
		Graph:           graph.New(entityRepo, edgeRepo, siteRepo, store, relay, now),
		Pages:           pages.New(pageRepo, linkRepo, entityRepo, siteRepo, store, relay, now),
		Templates:       templates.New(templateRepo, policyRepo, pageRepo, siteRepo, store, relay, now),
	}
	if err = core.Templates.EnsureSeeded(ctx); err != nil {
		return nil, stderrors.Join(err, store.Close())
	}
	return core, nil
}

func (c *Core) Close() error {
	return c.Store.Close()
}
