package app

import (
	"context"
	stderrors "errors"
	"os"
	"path/filepath"
	"strings"
	stdsync "sync"
	"sync/atomic"
	"time"

	"github.com/gollem-dev/gollem"
	"go.uber.org/zap"

	"github.com/davidmovas/postulator/internal/adapters/browser/tor"
	"github.com/davidmovas/postulator/internal/adapters/images"
	"github.com/davidmovas/postulator/internal/adapters/images/localfile"
	imageopenai "github.com/davidmovas/postulator/internal/adapters/images/openai"
	"github.com/davidmovas/postulator/internal/adapters/images/wpmedia"
	"github.com/davidmovas/postulator/internal/adapters/importer"
	"github.com/davidmovas/postulator/internal/adapters/llm/catalog"
	"github.com/davidmovas/postulator/internal/adapters/llm/gollemclient"
	"github.com/davidmovas/postulator/internal/adapters/llm/ledger"
	"github.com/davidmovas/postulator/internal/adapters/llm/limiter"
	"github.com/davidmovas/postulator/internal/adapters/llm/profiles"
	"github.com/davidmovas/postulator/internal/adapters/llm/recordreplay"
	"github.com/davidmovas/postulator/internal/adapters/llm/retry"
	"github.com/davidmovas/postulator/internal/adapters/secrets"
	"github.com/davidmovas/postulator/internal/adapters/secrets/export"
	"github.com/davidmovas/postulator/internal/adapters/secrets/masterkey"
	"github.com/davidmovas/postulator/internal/adapters/sqlite"
	"github.com/davidmovas/postulator/internal/adapters/wp"
	"github.com/davidmovas/postulator/internal/adapters/wp/plugin"
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
	domainllm "github.com/davidmovas/postulator/internal/domain/llm"
	"github.com/davidmovas/postulator/internal/domain/run"
	"github.com/davidmovas/postulator/internal/domain/template"
	"github.com/davidmovas/postulator/internal/kernel/clock"
	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/kernel/settings"
	"github.com/davidmovas/postulator/internal/runtime"
	"github.com/davidmovas/postulator/internal/runtime/scheduler"
	"github.com/davidmovas/postulator/internal/runtime/steps"
	agentrunner "github.com/davidmovas/postulator/internal/transport/agent"
)

const (
	HomeVariable = "POSTULATOR_HOME"

	homeDirectory = "Postulator"
	databaseFile  = "postulator.db"
)

type AgentProvider interface {
	New(ctx context.Context, ref domainllm.ModelRef) (gollem.LLMClient, error)
	NewForTools(ctx context.Context, ref domainllm.ModelRef) (gollem.LLMClient, error)
}

type Config struct {
	DatabasePath  string
	KeyDir        string
	Provider      llmport.Client
	AgentProvider AgentProvider
	Environment   func(name string) string
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
	cfg, logger, relay := c.cfg, c.logger, c.Events
	recovery := cfg.recovery()

	store, err := sqlite.Open(sqlite.Config{Path: cfg.DatabasePath, Key: key, Recovery: recovery})
	if err != nil {
		return kit{}, err
	}

	now := clock.System{}
	declarations := settings.Default()
	settingsStore := sqlite.NewSettingsRepo(store, now)
	values, unknown, err := LoadSettings(ctx, settingsStore, declarations)
	if err != nil {
		return kit{}, stderrors.Join(err, store.Close())
	}

	secretStore := secrets.NewStore(sqlite.NewSecretsRepo(store, now), key)
	siteRepo := sqlite.NewSiteRepo(store)
	entityRepo := sqlite.NewEntityRepo(store)
	edgeRepo := sqlite.NewEdgeRepo(store)
	pageRepo := sqlite.NewPageRepo(store)
	linkRepo := sqlite.NewPageLinkRepo(store)
	templateRepo := sqlite.NewTemplateRepo(store)
	policyRepo := sqlite.NewLinkPolicyRepo(store)
	modelRepo := sqlite.NewModelCatalogRepo(store)
	profileRepo := sqlite.NewModelProfileRepo(store)
	callRepo := sqlite.NewLLMCallRepo(store)

	modelCatalog, err := catalog.New(modelRepo)
	if err != nil {
		return kit{}, stderrors.Join(err, store.Close())
	}

	templateService := templates.New(templateRepo, policyRepo, pageRepo, siteRepo, store, relay, now)
	modelProfiles := profiles.New(profileRepo, siteRepo, modelCatalog, now)
	var providers AgentProvider = gollemclient.NewFactory(secretStore, modelCatalog, values)
	if cfg.AgentProvider != nil {
		providers = cfg.AgentProvider
	}
	provider := cfg.Provider
	if provider == nil {
		provider = gollemclient.New(providers, modelCatalog, gollemclient.Timeout(values))
	}
	book := ledger.New(
		recordreplay.New(provider, recordreplay.Mode(values), recordreplay.DefaultDir),
		callRepo, modelCatalog, relay, now,
	)
	client := retry.New(limiter.New(book, modelCatalog), retry.Retries(values), retry.DefaultBackoff)

	wordpress := registry.New(siteRepo, secretStore, wp.FromSettings(values)...)
	contentService := content.New(content.Deps{
		Pages: pageRepo, Entities: entityRepo, Edges: edgeRepo, Specs: templateService,
		Policies: templateService, Profiles: modelProfiles, Raw: rawContent{clients: wordpress}, LLM: client,
	})

	stepRegistry := run.NewRegistry()
	if err = steps.Register(stepRegistry, steps.Deps{
		Entities:      entityRepo,
		Edges:         edgeRepo,
		Pages:         pageRepo,
		Links:         linkRepo,
		Sites:         siteRepo,
		SiteWriter:    siteRepo,
		WordPress:     wordpress,
		Policies:      templateService,
		Profiles:      modelProfiles,
		Content:       contentService,
		LLM:           client,
		ImageProvider: imageopenai.New(secretStore, images.OpenAIModel(values)),
		ImageSources: map[template.ImageSource]steps.ImageSource{
			template.ImagesWPMedia: wpmedia.New(wordpress),
			template.ImagesLocal:   localfile.New(images.LocalDir(values)),
		},
		UnitOfWork: store,
		Publisher:  relay,
		Clock:      now,
		BatchSize:  steps.BatchSize(values),
	}); err != nil {
		return kit{}, stderrors.Join(err, store.Close())
	}

	runRepo := sqlite.NewRunRepo(store)
	itemRepo := sqlite.NewRunItemRepo(store)
	artifactRepo := sqlite.NewArtifactRepo(store)
	execRepo := sqlite.NewStepExecRepo(store)
	eventRepo := sqlite.NewRunEventRepo(store)

	engine := runtime.New(runtime.Deps{
		Runs:       runRepo,
		Items:      itemRepo,
		Artifacts:  artifactRepo,
		Execs:      execRepo,
		Events:     eventRepo,
		Pages:      pageRepo,
		Specs:      templateService,
		Spend:      callRepo,
		Catalog:    modelCatalog,
		Profiles:   modelProfiles,
		UnitOfWork: store,
		Publisher:  relay,
	}, stepRegistry, runtime.Settings(values), now, logger)

	sitesService := sites.New(siteRepo, secretStore, store, wordpress, relay, now)
	graphService := graph.New(graph.Deps{
		Entities: entityRepo, Edges: edgeRepo, Sites: siteRepo, Pages: pageRepo, Work: itemRepo,
		Profiles: modelProfiles, LLM: client, UnitOfWork: store, Publisher: relay, Clock: now,
	})
	pagesService := pages.New(pageRepo, linkRepo, entityRepo, siteRepo, store, relay, now, previewIssuer{clients: wordpress})
	importsService := imports.New(imports.Deps{
		Tables:     importer.New(),
		Entities:   entityRepo,
		Edges:      edgeRepo,
		Pages:      pageRepo,
		Templates:  templateRepo,
		Mappings:   sqlite.NewImportMappingRepo(store),
		Sites:      siteRepo,
		UnitOfWork: store,
		Publisher:  relay,
		Clock:      now,
		MaxRows:    imports.MaxRows(values),
	})
	modelsService := models.New(modelCatalog, modelRepo, modelProfiles, book, secretStore, client, relay, now)
	runsService := runs.New(engine, runRepo, itemRepo, artifactRepo, eventRepo, templateService, stepRegistry)
	syncService := sync.New(engine, siteRepo, wordpress, packer{}, now)
	reportsService := reports.New(reports.Deps{
		Entities: entityRepo, Edges: edgeRepo, Pages: pageRepo, Links: linkRepo,
		Runs: runRepo, Items: itemRepo, Artifacts: artifactRepo,
		Sites: siteRepo, Specs: templateService, Policies: templateService,
	})

	scheduleRepo := sqlite.NewScheduleRepo(store)
	schedulesService := schedules.New(schedules.Deps{
		Schedules: scheduleRepo, Pages: pageRepo, Sites: siteRepo, Runs: runsService,
		RunReader: runRepo, Publisher: relay, Clock: now,
	})

	browserService := browser.New(browser.Deps{
		Browser: tor.New(cfg.environment()),
		TorPath: func() string { return tor.Path(values) },
	})

	actionRepo := sqlite.NewPendingActionRepo(store)
	toolRegistry := tools.New(tools.Deps{
		Sites: sitesService, Graph: graphService, Pages: pagesService, Templates: templateService,
		Runs: runsService, Sync: syncService, Reports: reportsService, Imports: importsService,
		Models: modelsService, Content: contentService, Schedules: schedulesService,
		Actions: actionRepo, Publisher: relay, Clock: now,
	})

	agentService := agent.New(agent.Deps{
		Conversations: sqlite.NewConversationRepo(store),
		Messages:      sqlite.NewMessageRepo(store),
		Actions:       actionRepo,
		Calls:         sqlite.NewToolCallRepo(store),
		Sites:         siteRepo,
		Reports:       reportsService,
		Templates:     templateService,
		Profiles:      modelProfiles,
		LLM:           client,
		Registry:      toolRegistry,
		Runner: agentrunner.New(agentrunner.Deps{
			Factory:  providers,
			Registry: toolRegistry,
			History:  sqlite.NewConversationHistoryRepo(store),
			Calls:    callRepo,
			Catalog:  modelCatalog,
			Clock:    now,
			Logger:   logger,
		}, agentrunner.Config{
			MaxToolResultBytes: agent.MaxToolResultBytes(values),
			Retries:            retry.Retries(values),
			Backoff:            retry.DefaultBackoff,
		}),
		Publisher:     relay,
		Clock:         now,
		TurnTimeout:   func() time.Duration { return agent.TurnTimeout(values) },
		LoopLimit:     func() int { return agent.LoopLimit(values) },
		HistoryBudget: func() int { return agent.HistoryBudgetChars(values) },
		MaxToolResult: func() int { return agent.MaxToolResultBytes(values) },
	})

	built := kit{
		Archive:         export.New(store, now),
		Store:           store,
		Secrets:         secretStore,
		Settings:        values,
		Declarations:    declarations,
		SettingsStore:   settingsStore,
		UnknownSettings: unknown,
		Sites:           sitesService,
		Graph:           graphService,
		Pages:           pagesService,
		Imports:         importsService,
		Templates:       templateService,
		Content:         contentService,
		LLM:             client,
		Catalog:         modelCatalog,
		Profiles:        modelProfiles,
		Ledger:          book,
		Models:          modelsService,
		Steps:           stepRegistry,
		Engine:          engine,
		Runs:            runsService,
		Sync:            syncService,
		Reports:         reportsService,
		WordPress:       wordpress,
		Tools:           toolRegistry,
		Agent:           agentService,
		Browser:         browserService,
		Schedules:       schedulesService,
		Scheduler:       scheduler.New(schedulesService, scheduler.TickInterval(values), logger),
	}

	if err = templateService.EnsureSeeded(ctx); err != nil {
		return kit{}, stderrors.Join(err, store.Close())
	}
	if err = engine.Start(ctx); err != nil {
		return kit{}, stderrors.Join(err, store.Close())
	}
	if err = built.Scheduler.Start(ctx); err != nil {
		engine.Stop()
		return kit{}, stderrors.Join(err, store.Close())
	}
	return built, nil
}

type packer struct{}

func (packer) Package() ([]byte, error) {
	return plugin.Package()
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
