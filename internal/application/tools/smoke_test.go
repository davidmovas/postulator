package tools_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/davidmovas/postulator/internal/adapters/importer"
	"github.com/davidmovas/postulator/internal/adapters/llm/catalog"
	"github.com/davidmovas/postulator/internal/adapters/llm/fake"
	"github.com/davidmovas/postulator/internal/adapters/llm/ledger"
	"github.com/davidmovas/postulator/internal/adapters/llm/profiles"
	"github.com/davidmovas/postulator/internal/adapters/secrets"
	"github.com/davidmovas/postulator/internal/adapters/sqlite"
	"github.com/davidmovas/postulator/internal/adapters/sqlite/sqlitetest"
	"github.com/davidmovas/postulator/internal/application/applicationtest"
	"github.com/davidmovas/postulator/internal/application/content"
	"github.com/davidmovas/postulator/internal/application/graph"
	"github.com/davidmovas/postulator/internal/application/imports"
	"github.com/davidmovas/postulator/internal/application/models"
	"github.com/davidmovas/postulator/internal/application/pages"
	"github.com/davidmovas/postulator/internal/application/reports"
	"github.com/davidmovas/postulator/internal/application/runs"
	"github.com/davidmovas/postulator/internal/application/schedules"
	"github.com/davidmovas/postulator/internal/application/sites"
	"github.com/davidmovas/postulator/internal/application/sync"
	"github.com/davidmovas/postulator/internal/application/templates"
	"github.com/davidmovas/postulator/internal/application/tools"
	domainagent "github.com/davidmovas/postulator/internal/domain/agent"
	domainrun "github.com/davidmovas/postulator/internal/domain/run"
	"github.com/davidmovas/postulator/internal/domain/site"
	"github.com/davidmovas/postulator/internal/domain/template"
	"github.com/davidmovas/postulator/internal/kernel/clock"
	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/kernel/id"
)

type stubProbe struct{}

func (stubProbe) Probe(context.Context, site.Site) (site.PluginState, error) {
	return site.PluginState{Capabilities: []string{}}, nil
}

func (stubProbe) TestConnection(context.Context, site.Candidate) (site.Reachability, error) {
	return site.Reachability{Reach: site.ReachOK}, nil
}

type stubEngine struct{}

func (stubEngine) Enqueue(_ context.Context, record domainrun.Run) (domainrun.Run, error) {
	record.ID = id.New()
	record.Status = domainrun.StatusPending
	return record, nil
}

func (stubEngine) EstimateRun(context.Context, domainrun.Run, template.TemplateSpec) (domainrun.Estimate, error) {
	return domainrun.Estimate{}, nil
}

func (stubEngine) Pause(context.Context, string, domainrun.PauseReason) error { return nil }

func (stubEngine) Resume(context.Context, string) error { return nil }

func (stubEngine) Cancel(context.Context, string) error { return nil }

func (stubEngine) RetryStep(context.Context, string) error { return nil }

type stubPacker struct{}

func (stubPacker) Package() ([]byte, error) {
	return []byte("PK"), nil
}

func wired(t *testing.T) (registry *tools.Registry, binding tools.Binding) {
	t.Helper()

	store := sqlitetest.Open(t)
	owner := sqlitetest.Site(t, store, "shop")
	sqlitetest.Entity(t, store, owner.ID, "Coffee")
	sqlitetest.Page(t, store, owner.ID, "/coffee/")
	sqlitetest.Template(t, store, "Guide")

	conversation, err := domainagent.NewConversation(domainagent.Conversation{
		ID: id.New(), SiteID: &owner.ID, Title: "smoke", CreatedAt: stamp, UpdatedAt: stamp,
	})
	if err != nil {
		t.Fatalf("NewConversation: %v", err)
	}
	if insertErr := sqlite.NewConversationRepo(store).Insert(t.Context(), conversation); insertErr != nil {
		t.Fatalf("insert the conversation: %v", insertErr)
	}

	now := clock.NewFake(stamp)
	bus := &applicationtest.Recorder{}
	siteRepo := sqlite.NewSiteRepo(store)
	pageRepo := sqlite.NewPageRepo(store)
	entityRepo := sqlite.NewEntityRepo(store)
	edgeRepo := sqlite.NewEdgeRepo(store)
	linkRepo := sqlite.NewPageLinkRepo(store)
	templateRepo := sqlite.NewTemplateRepo(store)
	modelRepo := sqlite.NewModelCatalogRepo(store)
	callRepo := sqlite.NewLLMCallRepo(store)
	runRepo := sqlite.NewRunRepo(store)
	itemRepo := sqlite.NewRunItemRepo(store)
	artifactRepo := sqlite.NewArtifactRepo(store)

	built, catalogErr := catalog.New(modelRepo)
	if catalogErr != nil {
		t.Fatalf("catalog: %v", catalogErr)
	}

	client := fake.New()
	book := ledger.New(client, callRepo, built, bus, now)
	modelProfiles := profiles.New(sqlite.NewModelProfileRepo(store), siteRepo, built, now)
	templateService := templates.New(templateRepo, sqlite.NewLinkPolicyRepo(store), pageRepo, siteRepo, store, bus, now)
	reportsService := reports.New(entityRepo, edgeRepo, pageRepo, linkRepo, runRepo, itemRepo, artifactRepo)

	return tools.New(tools.Deps{
		Sites: sites.New(siteRepo, secrets.NewStore(sqlite.NewSecretsRepo(store, now), sqlitetest.Key()), store, stubProbe{}, bus, now),
		Graph: graph.New(graph.Deps{
			Entities: entityRepo, Edges: edgeRepo, Sites: siteRepo, Pages: pageRepo,
			Profiles: modelProfiles, LLM: book, UnitOfWork: store, Publisher: bus, Clock: now,
		}),
		Pages:     pages.New(pageRepo, linkRepo, entityRepo, siteRepo, store, bus, now),
		Templates: templateService,
		Runs:      runs.New(stubEngine{}, runRepo, itemRepo, artifactRepo, sqlite.NewRunEventRepo(store), templateService, domainrun.NewRegistry()),
		Sync:      sync.New(stubEngine{}, siteRepo, stubProbe{}, stubPacker{}, now),
		Reports:   reportsService,
		Imports: imports.New(imports.Deps{
			Tables:   importer.New(),
			Entities: entityRepo, Edges: edgeRepo, Pages: pageRepo, Templates: templateRepo,
			Mappings: sqlite.NewImportMappingRepo(store), Sites: siteRepo, UnitOfWork: store,
			Publisher: bus, Clock: now,
		}),
		Models: models.New(built, modelRepo, modelProfiles, book,
			secrets.NewStore(sqlite.NewSecretsRepo(store, now), sqlitetest.Key()), book, bus, now),
		Content: content.New(content.Deps{
			Pages: pageRepo, Entities: entityRepo, Edges: edgeRepo, Specs: templateService,
			Policies: templateService, Profiles: modelProfiles, LLM: book,
		}),
		Schedules: schedules.New(schedules.Deps{
			Schedules: sqlite.NewScheduleRepo(store), Pages: pageRepo, Sites: siteRepo,
			Runs:      runs.New(stubEngine{}, runRepo, itemRepo, artifactRepo, sqlite.NewRunEventRepo(store), templateService, domainrun.NewRegistry()),
			RunReader: runRepo, Publisher: bus, Clock: now,
		}),
		Actions:   sqlite.NewPendingActionRepo(store),
		Publisher: bus,
		Clock:     now,
	}), tools.Binding{SiteID: owner.ID, ConversationID: conversation.ID}
}

func TestEveryReadToolRunsAgainstTheRealUseCases(t *testing.T) {
	t.Parallel()

	registry, binding := wired(t)
	binding.Mode = domainagent.ModeAutonomous

	for _, tool := range registry.Build(binding) {
		if tool.Def.Risk != tools.RiskRead {
			continue
		}

		t.Run(tool.Def.Name, func(t *testing.T) {
			_, err := registry.Call(t.Context(), binding, tool.Def.Name, argumentsFor(tool.Def.Name))
			if errors.IsCode(err, errors.Internal) {
				t.Fatalf("%s failed with an internal error: %v", tool.Def.Name, err)
			}
		})
	}
}

func TestEveryWriteToolProposesInsteadOfWriting(t *testing.T) {
	t.Parallel()

	registry, binding := wired(t)
	binding.Mode = domainagent.ModeConfirm

	proposals := 0
	for _, tool := range registry.Build(binding) {
		if tool.Def.Risk == tools.RiskRead {
			continue
		}

		out, err := tool.Run(t.Context(), binding, json.RawMessage(`{}`))
		if err != nil {
			t.Fatalf("%s proposed with %v", tool.Def.Name, err)
		}
		confirmation, ok := out.(tools.Confirmation)
		if !ok || confirmation.ActionID == "" {
			t.Fatalf("%s answered %+v", tool.Def.Name, out)
		}
		proposals++
	}

	if proposals == 0 {
		t.Fatal("no write tool was proposed")
	}
}

func TestEveryWriteToolRunsAgainstTheRealUseCases(t *testing.T) {
	t.Parallel()

	registry, binding := wired(t)
	binding.Mode = domainagent.ModeAutonomous

	for _, tool := range registry.Build(binding) {
		if tool.Def.Risk == tools.RiskRead {
			continue
		}

		t.Run(tool.Def.Name, func(t *testing.T) {
			_, err := registry.Call(t.Context(), binding, tool.Def.Name, argumentsFor(tool.Def.Name))
			if errors.IsCode(err, errors.Internal) {
				t.Fatalf("%s failed with an internal error: %v", tool.Def.Name, err)
			}
		})
	}
}

func argumentsFor(name string) json.RawMessage {
	switch name {
	case "imports_inspect", "imports_preview":
		return json.RawMessage(`{"path":"C:/nowhere/absent.csv","mapping":{"columns":[]}}`)
	default:
		return json.RawMessage(`{}`)
	}
}
