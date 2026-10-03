package app

import (
	"context"
	stderrors "errors"
	"os"
	"path/filepath"
	"strings"
	stdsync "sync"
	"sync/atomic"

	"go.uber.org/zap"

	"github.com/davidmovas/postulator/internal/adapters/llm/catalog"
	"github.com/davidmovas/postulator/internal/adapters/llm/ledger"
	"github.com/davidmovas/postulator/internal/adapters/llm/profiles"
	"github.com/davidmovas/postulator/internal/adapters/secrets"
	"github.com/davidmovas/postulator/internal/adapters/secrets/export"
	"github.com/davidmovas/postulator/internal/adapters/secrets/masterkey"
	"github.com/davidmovas/postulator/internal/adapters/sqlite"
	"github.com/davidmovas/postulator/internal/adapters/wp/registry"
	"github.com/davidmovas/postulator/internal/application/agent"
	"github.com/davidmovas/postulator/internal/application/browser"
	"github.com/davidmovas/postulator/internal/application/content"
	"github.com/davidmovas/postulator/internal/application/graph"
	"github.com/davidmovas/postulator/internal/application/imports"
	llmport "github.com/davidmovas/postulator/internal/application/llm"
	"github.com/davidmovas/postulator/internal/application/models"
	"github.com/davidmovas/postulator/internal/application/pages"
	"github.com/davidmovas/postulator/internal/application/reports"
	"github.com/davidmovas/postulator/internal/application/runs"
	"github.com/davidmovas/postulator/internal/application/schedules"
	"github.com/davidmovas/postulator/internal/application/sites"
	"github.com/davidmovas/postulator/internal/application/sync"
	"github.com/davidmovas/postulator/internal/application/templates"
	"github.com/davidmovas/postulator/internal/application/tools"
	"github.com/davidmovas/postulator/internal/domain/run"
	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/kernel/settings"
	"github.com/davidmovas/postulator/internal/runtime"
	"github.com/davidmovas/postulator/internal/runtime/scheduler"
)

const (
	HomeVariable = "POSTULATOR_HOME"

	homeDirectory = "Postulator"
	databaseFile  = "postulator.db"
)

type Config struct {
	DatabasePath  string
	KeyDir        string
	Provider      llmport.Client
	AgentProvider AgentProvider
	Environment   func(name string) string
	Open          func(cfg sqlite.Config) (*sqlite.Store, error)
}

func (c Config) environment() func(string) string {
	if c.Environment != nil {
		return c.Environment
	}
	return os.Getenv
}

func (c Config) recovery() string {
	return "remove " + filepath.Join(c.KeyDir, masterkey.FileName) + " and " + c.DatabasePath +
		" to reset the application state"
}

func DefaultConfig() (Config, error) {
	home, err := homePath()
	if err != nil {
		return Config{}, err
	}
	return Config{DatabasePath: filepath.Join(home, databaseFile), KeyDir: home}, nil
}

func homePath() (string, error) {
	if override := strings.TrimSpace(os.Getenv(HomeVariable)); override != "" {
		absolute, err := filepath.Abs(override)
		if err != nil {
			return "", errors.Wrap(err, errors.Invalid, "resolve "+HomeVariable)
		}
		return absolute, nil
	}

	base, err := os.UserConfigDir()
	if err != nil {
		return "", errors.Wrap(err, errors.Internal, "locate the user configuration directory")
	}
	return filepath.Join(base, homeDirectory), nil
}

type Core struct {
	cfg      Config
	logger   *zap.Logger
	mu       stdsync.RWMutex
	changing atomic.Bool
	closed   bool
	key      []byte
	Events   *EventRelay
	kit
}

type kit struct {
	Archive         *export.Archive
	Store           *sqlite.Store
	Secrets         *secrets.Store
	Settings        *settings.Values
	Declarations    *settings.Registry
	SettingsStore   *sqlite.SettingsRepo
	UnknownSettings []string
	Sites           *sites.Service
	Graph           *graph.Service
	Pages           *pages.Service
	Imports         *imports.Service
	Templates       *templates.Service
	Content         *content.Service
	LLM             llmport.Client
	Catalog         *catalog.Catalog
	Profiles        *profiles.Profiles
	Ledger          *ledger.Ledger
	Models          *models.Service
	Steps           *run.Registry
	Engine          *runtime.Engine
	Runs            *runs.Service
	Sync            *sync.Service
	Reports         *reports.Service
	WordPress       *registry.Registry
	Tools           *tools.Registry
	Agent           *agent.Service
	Browser         *browser.Service
	Schedules       *schedules.Service
	Scheduler       *scheduler.Scheduler
}

func Open(ctx context.Context, cfg Config, logger *zap.Logger) (*Core, error) {
	if cfg.DatabasePath == "" {
		return nil, errors.New(errors.Invalid, "the database path must not be empty")
	}
	if cfg.KeyDir == "" {
		return nil, errors.New(errors.Invalid, "the key directory must not be empty")
	}

	core := &Core{cfg: cfg, logger: logger, Events: &EventRelay{}}

	protected, err := masterkey.Protected(core.keyConfig())
	if err != nil {
		return nil, err
	}
	if protected {
		return core, nil
	}

	key, err := masterkey.Load(core.keyConfig())
	if err != nil {
		return nil, err
	}
	if err := core.compose(ctx, key); err != nil {
		return nil, err
	}
	return core, nil
}

func (c *Core) keyConfig() masterkey.Config {
	return masterkey.Config{Dir: c.cfg.KeyDir, Recovery: c.cfg.recovery()}
}

func (c *Core) compose(ctx context.Context, key []byte) error {
	built, err := c.build(ctx, key)
	if err != nil {
		return err
	}
	if !c.install(key, built) {
		quiesce(built)
		return stderrors.Join(locked(), built.Store.Close())
	}
	return nil
}

func (c *Core) build(ctx context.Context, key []byte) (kit, error) {
	store, err := c.openStore(key)
	if err != nil {
		return kit{}, err
	}
	built, err := c.assemble(ctx, store, key)
	if err != nil {
		return kit{}, stderrors.Join(err, store.Close())
	}
	return built, nil
}

func (c *Core) assemble(ctx context.Context, store *sqlite.Store, key []byte) (kit, error) {
	stores, err := openRepos(ctx, store, key)
	if err != nil {
		return kit{}, err
	}
	llm, err := c.buildLLM(stores)
	if err != nil {
		return kit{}, err
	}
	writing := c.buildAuthoring(stores, llm)
	running, err := c.buildRuntime(stores, llm, writing)
	if err != nil {
		return kit{}, err
	}
	services := c.buildServices(stores, llm, writing, running)
	assistant := c.buildAgent(stores, llm, writing, services)

	built := kit{
		Archive:         export.New(store, stores.now),
		Store:           store,
		Secrets:         stores.secrets,
		Settings:        stores.values,
		Declarations:    stores.declarations,
		SettingsStore:   stores.settingsStore,
		UnknownSettings: stores.unknown,
		Sites:           services.sites,
		Graph:           services.graph,
		Pages:           services.pages,
		Imports:         services.imports,
		Templates:       writing.templates,
		Content:         writing.content,
		LLM:             llm.client,
		Catalog:         llm.catalog,
		Profiles:        llm.profiles,
		Ledger:          llm.ledger,
		Models:          services.models,
		Steps:           running.steps,
		Engine:          running.engine,
		Runs:            services.runs,
		Sync:            services.sync,
		Reports:         services.reports,
		WordPress:       writing.wordpress,
		Tools:           assistant.tools,
		Agent:           assistant.agent,
		Browser:         services.browser,
		Schedules:       services.schedules,
		Scheduler:       c.buildScheduler(stores, services),
	}
	if err := start(ctx, built); err != nil {
		return kit{}, err
	}
	return built, nil
}

func start(ctx context.Context, built kit) error {
	if err := built.Templates.EnsureSeeded(ctx); err != nil {
		return err
	}
	if err := built.Engine.Start(ctx); err != nil {
		return err
	}
	if err := built.Scheduler.Start(ctx); err != nil {
		built.Engine.Stop()
		return err
	}
	return nil
}

func (c *Core) Close() error {
	c.mu.Lock()
	c.closed = true
	retired, key := c.kit, c.key
	c.kit = kit{}
	c.key = nil
	c.mu.Unlock()

	if retired.Store == nil {
		return nil
	}

	quiesce(retired)
	clear(key)
	return retired.Store.Close()
}
