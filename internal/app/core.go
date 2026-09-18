package app

import (
	"context"
	stderrors "errors"
	"os"
	"path/filepath"

	"go.uber.org/zap"

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
	"github.com/davidmovas/postulator/internal/adapters/secrets/masterkey"
	"github.com/davidmovas/postulator/internal/adapters/sqlite"
	"github.com/davidmovas/postulator/internal/adapters/wp"
	"github.com/davidmovas/postulator/internal/adapters/wp/plugin"
	"github.com/davidmovas/postulator/internal/adapters/wp/registry"
	"github.com/davidmovas/postulator/internal/application/agent"
	"github.com/davidmovas/postulator/internal/application/content"
	"github.com/davidmovas/postulator/internal/application/graph"
	"github.com/davidmovas/postulator/internal/application/imports"
	llmport "github.com/davidmovas/postulator/internal/application/llm"
	"github.com/davidmovas/postulator/internal/application/models"
	"github.com/davidmovas/postulator/internal/application/pages"
	"github.com/davidmovas/postulator/internal/application/reports"
	"github.com/davidmovas/postulator/internal/application/runs"
	"github.com/davidmovas/postulator/internal/application/sites"
	"github.com/davidmovas/postulator/internal/application/sync"
	"github.com/davidmovas/postulator/internal/application/templates"
	"github.com/davidmovas/postulator/internal/application/tools"
	"github.com/davidmovas/postulator/internal/domain/run"
	"github.com/davidmovas/postulator/internal/domain/template"
	"github.com/davidmovas/postulator/internal/kernel/clock"
	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/kernel/settings"
	"github.com/davidmovas/postulator/internal/runtime"
	"github.com/davidmovas/postulator/internal/runtime/steps"
	agentrunner "github.com/davidmovas/postulator/internal/transport/agent"
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
}

func Open(ctx context.Context, cfg Config, logger *zap.Logger) (*Core, error) {
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
	modelRepo := sqlite.NewModelCatalogRepo(store)
	profileRepo := sqlite.NewModelProfileRepo(store)
	callRepo := sqlite.NewLLMCallRepo(store)

	modelCatalog, err := catalog.New(modelRepo)
	if err != nil {
		return nil, stderrors.Join(err, store.Close())
	}

	templateService := templates.New(templateRepo, policyRepo, pageRepo, siteRepo, store, relay, now)
	modelProfiles := profiles.New(profileRepo, siteRepo, modelCatalog, now)
	providers := gollemclient.NewFactory(secretStore, values)
	book := ledger.New(
		recordreplay.New(
			gollemclient.New(providers, gollemclient.Timeout(values)),
			recordreplay.Mode(values),
			recordreplay.DefaultDir,
		),
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
		return nil, stderrors.Join(err, store.Close())
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

	sitesService := sites.New(siteRepo, secretStore, store, now)
	graphService := graph.New(graph.Deps{
		Entities: entityRepo, Edges: edgeRepo, Sites: siteRepo, Pages: pageRepo,
		Profiles: modelProfiles, LLM: client, UnitOfWork: store, Publisher: relay, Clock: now,
	})
	pagesService := pages.New(pageRepo, linkRepo, entityRepo, siteRepo, store, relay, now)
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
	modelsService := models.New(modelCatalog, modelRepo, modelProfiles, book, secretStore, client, now)
	runsService := runs.New(engine, runRepo, itemRepo, artifactRepo, eventRepo, templateService)
	syncService := sync.New(engine, siteRepo, wordpress, packer{}, now)
	reportsService := reports.New(entityRepo, edgeRepo, pageRepo, linkRepo, runRepo, itemRepo, artifactRepo)

	actionRepo := sqlite.NewPendingActionRepo(store)
	toolRegistry := tools.New(tools.Deps{
		Sites: sitesService, Graph: graphService, Pages: pagesService, Templates: templateService,
		Runs: runsService, Sync: syncService, Reports: reportsService, Imports: importsService,
		Models: modelsService, Content: contentService, Actions: actionRepo, Publisher: relay, Clock: now,
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
		Registry:      toolRegistry,
		Runner: agentrunner.New(agentrunner.Deps{
			Factory:  providers,
			Registry: toolRegistry,
			History:  sqlite.NewConversationHistoryRepo(store),
			Calls:    callRepo,
			Catalog:  modelCatalog,
			Clock:    now,
			Logger:   logger,
		}, agentrunner.Config{MaxToolResultBytes: agent.MaxToolResultBytes(values)}),
		Publisher:     relay,
		Clock:         now,
		LoopLimit:     agent.LoopLimit(values),
		HistoryBudget: agent.HistoryBudgetChars(values),
	})

	core := &Core{
		Store:           store,
		Secrets:         secretStore,
		Settings:        values,
		UnknownSettings: unknown,
		Events:          relay,
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
	}
	if err = core.Templates.EnsureSeeded(ctx); err != nil {
		return nil, stderrors.Join(err, store.Close())
	}
	if err = engine.Start(ctx); err != nil {
		return nil, stderrors.Join(err, store.Close())
	}
	return core, nil
}

type packer struct{}

func (packer) Package() ([]byte, error) {
	return plugin.Package()
}

func (c *Core) Close() error {
	c.Agent.Close()
	c.Engine.Stop()
	return c.Store.Close()
}
