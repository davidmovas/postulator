package tools_test

import (
	"context"
	"encoding/json"
	"strings"
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
	"github.com/davidmovas/postulator/internal/application/findings"
	"github.com/davidmovas/postulator/internal/application/graph"
	"github.com/davidmovas/postulator/internal/application/imports"
	"github.com/davidmovas/postulator/internal/application/models"
	"github.com/davidmovas/postulator/internal/application/pages"
	"github.com/davidmovas/postulator/internal/application/pagetypes"
	"github.com/davidmovas/postulator/internal/application/reports"
	"github.com/davidmovas/postulator/internal/application/runs"
	"github.com/davidmovas/postulator/internal/application/schedules"
	"github.com/davidmovas/postulator/internal/application/sitemodel"
	"github.com/davidmovas/postulator/internal/application/sites"
	"github.com/davidmovas/postulator/internal/application/sync"
	"github.com/davidmovas/postulator/internal/application/templates"
	"github.com/davidmovas/postulator/internal/application/tools"
	domainagent "github.com/davidmovas/postulator/internal/domain/agent"
	"github.com/davidmovas/postulator/internal/domain/field"
	domainllm "github.com/davidmovas/postulator/internal/domain/llm"
	domainrun "github.com/davidmovas/postulator/internal/domain/run"
	"github.com/davidmovas/postulator/internal/domain/site"
	"github.com/davidmovas/postulator/internal/kernel/clock"
	kctx "github.com/davidmovas/postulator/internal/kernel/ctx"
	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/kernel/id"
)

type stubProbe struct{}

func (stubProbe) Probe(context.Context, site.Site) (site.Extensions, error) {
	return site.Extensions{Plugin: site.PluginState{Capabilities: []string{}}}, nil
}

func (stubProbe) TestConnection(context.Context, site.Candidate) (site.Reachability, error) {
	return site.Reachability{Reach: site.ReachOK}, nil
}

type stubPreview struct{}

func (stubPreview) IssuePreview(context.Context, string, int64, string) (pages.IssuedPreview, error) {
	return pages.IssuedPreview{}, errors.New(errors.Invalid, "no site in this test issues a preview").
		WithDetail("code", "plugin_missing")
}

func (stubPreview) TrashItem(context.Context, string, int64, string) error {
	return errors.New(errors.Invalid, "no site in this test removes a page")
}

type stubEngine struct{}

func (stubEngine) Enqueue(_ context.Context, record domainrun.Run, _ ...domainrun.Directive) (domainrun.Run, error) {
	record.ID = id.New()
	record.Status = domainrun.StatusPending
	return record, nil
}

func (stubEngine) EstimateRun(context.Context, domainrun.Run, map[string]string, ...domainrun.Directive) (domainrun.Estimate, error) {
	return domainrun.Estimate{}, nil
}

func (stubEngine) Pause(context.Context, string, domainrun.PauseReason) error { return nil }

func (stubEngine) Resume(context.Context, string) error { return nil }

func (stubEngine) Cancel(context.Context, string) error { return nil }

func (stubEngine) RetryStep(context.Context, string) error { return nil }

func (stubEngine) Accept(context.Context, string, []field.Key) error { return nil }

func (stubEngine) Regenerate(context.Context, string, []string) error { return nil }

type stubPacker struct{}

func (stubPacker) Package() ([]byte, error) {
	return []byte("PK"), nil
}

type fixture struct {
	store    *sqlite.Store
	types    *pagetypes.Service
	pages    *pages.Service
	site     string
	entity   string
	other    string
	page     string
	template string
}

func wired(t *testing.T) (registry *tools.Registry, binding tools.Binding, seeded fixture) {
	t.Helper()

	store := sqlitetest.Open(t)
	owner := sqlitetest.Site(t, store, "shop")
	entity := sqlitetest.Entity(t, store, owner.ID, "Coffee")
	other := sqlitetest.Entity(t, store, owner.ID, "Espresso")
	page := sqlitetest.Page(t, store, owner.ID, "/coffee/")
	guide := sqlitetest.Template(t, store, "Guide")
	seeded = fixture{
		site: owner.ID, entity: entity.ID, other: other.ID, page: page.ID, template: guide.ID,
	}

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
	templateService := templates.New(templateRepo, pageRepo, entityRepo, siteRepo, sqlite.NewPageTypeRepo(store), store, bus, now)
	types := pagetypes.New(pagetypes.Deps{
		Types: sqlite.NewPageTypeRepo(store), Scopes: sqlite.NewFieldScopeRepo(store), Rules: sqlite.NewPageTypeRuleRepo(store),
		Pages: pageRepo, Templates: templateRepo,
		Sites: siteRepo, UnitOfWork: store, Publisher: bus, Clock: now,
	})
	if seedErr := types.Seed(t.Context(), owner.ID); seedErr != nil {
		t.Fatalf("seed the page types: %v", seedErr)
	}
	reportsService := reports.New(reports.Deps{
		Entities: entityRepo, Edges: edgeRepo, Pages: pageRepo, Fields: sqlite.NewPageFieldRepo(store), Links: linkRepo,
		Runs: runRepo, Items: itemRepo, Artifacts: artifactRepo,
		Sites: siteRepo, Specs: templateService, Types: types,
	})

	pagesService := pages.New(pages.Deps{
		Pages: pageRepo, Links: linkRepo, Entities: entityRepo, Edges: edgeRepo, Sites: siteRepo, Types: sqlite.NewPageTypeRepo(store),
		Bodies: sqlite.NewPageBodyRepo(store), Fields: sqlite.NewPageFieldRepo(store), Renders: sqlite.NewPageRenderRepo(store),
		Rows: sqlite.NewPageMapRepo(store), Templates: templateRepo, UnitOfWork: store, Publisher: bus, Clock: now, Preview: stubPreview{},
	})
	seeded.store, seeded.types, seeded.pages = store, types, pagesService
	siteModel := sitemodel.New(sitemodel.Deps{
		Budgets: wholeWindow{}, Audits: reportsService, Reader: store, UnitOfWork: store, Maps: sqlite.NewPageMapRepo(store),
		Sites: siteRepo, Pages: pageRepo,
		Entities: entityRepo, Edges: edgeRepo, Types: types, Terms: sqlite.NewPageIdentityRepo(store),
		Fields: sqlite.NewPageFieldRepo(store), Bodies: sqlite.NewPageBodyRepo(store),
		Renders: sqlite.NewPageRenderRepo(store), Links: linkRepo, Findings: sqlite.NewFindingRepo(store),
		Runs: runRepo, Items: itemRepo,
	})
	graphService := graph.New(graph.Deps{
		Entities: entityRepo, Edges: edgeRepo, Sites: siteRepo, Pages: pageRepo, Findings: sqlite.NewFindingRepo(store),
		Identities: sqlite.NewPageIdentityRepo(store), Fields: sqlite.NewPageFieldRepo(store), Maps: siteModel,
		Profiles: modelProfiles, LLM: book, UnitOfWork: store, Publisher: bus, Clock: now,
	})
	importsService := imports.New(imports.Deps{
		Tables:   importer.New(),
		Entities: entityRepo, Edges: edgeRepo, Pages: pageRepo, Types: sqlite.NewPageTypeRepo(store),
		Fields:   sqlite.NewPageFieldRepo(store),
		Mappings: sqlite.NewImportMappingRepo(store), Sites: siteRepo, UnitOfWork: store,
		Findings: sqlite.NewFindingRepo(store), Terms: sqlite.NewPageIdentityRepo(store),
		Publisher: bus, Clock: now,
	})
	return tools.New(tools.Deps{
		Sites:     sites.New(siteRepo, secrets.NewStore(sqlite.NewSecretsRepo(store, now), sqlitetest.Key()), store, types, stubProbe{}, bus, now),
		Graph:     graphService,
		Pages:     pagesService,
		PageTypes: types,
		Classify: pagetypes.NewClassification(pagetypes.ClassificationDeps{
			Facts: siteModel, Types: types, Pages: pageRepo, Findings: sqlite.NewFindingRepo(store), UnitOfWork: store,
			Publisher: bus, Clock: now,
		}),
		Templates: templateService,
		Runs:      runs.New(stubEngine{}, runRepo, itemRepo, artifactRepo, sqlite.NewRunEventRepo(store), templateService, pageRepo, types, domainrun.NewRegistry(), pagesService, siteModel, graphService),
		Sync:      sync.New(stubEngine{}, runRepo, siteRepo, stubProbe{}, stubPacker{}, now),
		Reports:   reportsService,
		Imports:   importsService,
		Findings: findings.New(findings.Deps{
			Findings: sqlite.NewFindingRepo(store), Fields: sqlite.NewPageFieldRepo(store), Bodies: sqlite.NewPageBodyRepo(store),
			Pages: pageRepo, Cards: graphService, Rows: importsService, Parents: graphService, Places: siteModel,
			UnitOfWork: store, Publisher: bus, Clock: now,
		}),
		Models: models.New(built, modelRepo, modelProfiles, book,
			secrets.NewStore(sqlite.NewSecretsRepo(store, now), sqlitetest.Key()), book, bus, now),
		Content: content.New(content.Deps{
			Pages: pageRepo, Entities: entityRepo, Edges: edgeRepo, Fields: sqlite.NewPageFieldRepo(store), Specs: templateService,
			Types: types, Profiles: modelProfiles, LLM: book,
		}),
		Schedules: schedules.New(schedules.Deps{
			Schedules: sqlite.NewScheduleRepo(store), Pages: pageRepo, Sites: siteRepo,
			Runs:      runs.New(stubEngine{}, runRepo, itemRepo, artifactRepo, sqlite.NewRunEventRepo(store), templateService, pageRepo, types, domainrun.NewRegistry(), pagesService, siteModel, graphService),
			RunReader: runRepo, Publisher: bus, Clock: now,
		}),
		SiteModel:   siteModel,
		ResultBytes: func() int { return 16384 },
		Actions:     sqlite.NewPendingActionRepo(store),
		Plans:       sqlite.NewPlanRepo(store),
		Publisher:   bus,
		Clock:       now,
	}), tools.Binding{SiteID: owner.ID, ConversationID: conversation.ID}, seeded
}

type wholeWindow struct{}

func (wholeWindow) InputBudget(domainllm.Role) int {
	return 0
}

func TestLinksAnswersEveryViewOverTheStore(t *testing.T) {
	t.Parallel()

	registry, binding, seeded := wired(t)
	binding.Mode = domainagent.ModeAutonomous
	calls := map[string]string{
		"page":    `{"view":"page","pageId":"` + seeded.page + `"}`,
		"section": `{"view":"section","pageId":"` + seeded.page + `"}`,
		"map":     `{"view":"map"}`,
		"owed":    `{"view":"owed","missing":true}`,
		"orphans": `{"view":"orphans","limit":10}`,
		"suggest": `{"view":"suggest"}`,
	}
	for view, args := range calls {
		t.Run(view, func(t *testing.T) {
			t.Parallel()

			out, err := registry.Call(t.Context(), binding, "links", json.RawMessage(args))
			if err != nil {
				t.Fatalf("links %s: %v", view, err)
			}
			answered, ok := out.(sitemodel.LinksAnswer)
			if !ok || string(answered.View) != view {
				t.Fatalf("links %s answered %+v", view, out)
			}
		})
	}
	section := called[sitemodel.LinksAnswer](t, registry, binding, "links", `{"view":"section"}`)
	if section.Totals == nil || section.Totals.Pages != 1 || section.Pages == nil || section.Pages.Items[0].Path != "/coffee/" {
		t.Errorf("the whole site's section = %+v, want the one seeded page", section)
	}
	if _, err := registry.Call(t.Context(), binding, "links", json.RawMessage(`{"view":"page"}`)); !errors.IsCode(err, errors.Invalid) {
		t.Errorf("a page view naming no page = %v, want Invalid", err)
	}
}

func TestEveryReadToolRunsAgainstTheRealUseCases(t *testing.T) {
	t.Parallel()

	registry, binding, _ := wired(t)
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

	registry, binding, _ := wired(t)
	binding.Mode = domainagent.ModeConfirm

	proposals := 0
	for _, tool := range registry.Build(binding) {
		if tool.Def.Risk == tools.RiskRead {
			continue
		}

		out, err := tool.Run(t.Context(), binding, argumentsFor(tool.Def.Name))
		if err != nil {
			if errors.IsCode(err, errors.Internal) {
				t.Fatalf("%s failed with an internal error: %v", tool.Def.Name, err)
			}
			t.Logf("%s refused before it proposed: %v", tool.Def.Name, err)
			continue
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

	registry, binding, _ := wired(t)
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

const minimalSpec = `{"sections":[{"heading":"Overview","intent":"what it is","targetWords":200,"required":true,` +
	`"keywordRules":{"include":[],"primaryInHeading":false}}],"tone":"plain","length":{"min":300,"max":900},` +
	`"keywordRules":{"primaryInTitle":true,"primaryInH1":true,"primaryInFirstParagraph":true,"maxDensity":0.02},` +
	`"metaRules":{"titlePattern":"{primaryKeyword}","descriptionMax":155},` +
	`"images":{"featured":false,"inline":0,"source":"ai"}}`

func TestAToolCallActsAsTheAgent(t *testing.T) {
	t.Parallel()

	registry, binding, seeded := wired(t)
	binding.Mode = domainagent.ModeAutonomous
	asPerson := kctx.WithActor(t.Context(), kctx.ActorUser)

	out, err := registry.Call(asPerson, binding, "graph_edit",
		json.RawMessage(`{"op":"anchors","entityId":"`+seeded.entity+`","anchors":[{"text":"espresso","weight":1}]}`))
	if err != nil {
		t.Fatalf("graph_edit anchors: %v", err)
	}
	set, ok := out.(graph.SetAnchorsResponse)
	if !ok || len(set.Entity.Anchors) != 1 || set.Entity.Anchors[0].Source != "ai" {
		t.Fatalf("the anchor the agent set = %+v, want it the AI's though the turn began in a person's call", out)
	}
}

func TestAProposedCardWaitsForAPersonUnlessTheCallWasApproved(t *testing.T) {
	t.Parallel()

	registry, binding, seeded := wired(t)
	args := json.RawMessage(`{"op":"cards","cards":[{"entityId":"` + seeded.entity + `","definition":"Brewed from roasted beans.","reason":"the hub's text"}]}`)

	binding.Mode = domainagent.ModeAutonomous
	filed, err := registry.Call(t.Context(), binding, "graph_propose", args)
	if err != nil {
		t.Fatalf("propose: %v", err)
	}
	proposed, ok := filed.(graph.ProposeCardsResponse)
	if !ok || len(proposed.Proposals) != 1 || proposed.Proposals[0].Proposed.Definition != "Brewed from roasted beans." {
		t.Fatalf("an autonomous call = %+v, want one proposal filed", filed)
	}
	read, err := registry.Call(t.Context(), binding, "graph_read", json.RawMessage(`{"view":"entity","entityId":"`+seeded.entity+`"}`))
	if held, isEntity := read.(tools.GraphReadResponse); err != nil || !isEntity || held.Entity == nil || held.Entity.Definition != "" {
		t.Fatalf("the entity after a proposal = %+v, %v; want nothing written", read, err)
	}

	binding.Mode, binding.Approved = domainagent.ModeConfirm, true
	written, err := registry.Call(t.Context(), binding, "graph_propose", args)
	if err != nil {
		t.Fatalf("an approved call: %v", err)
	}
	if applied, isApplied := written.(graph.ApplyCardsResponse); !isApplied || len(applied.Entities) != 1 || applied.Entities[0].Definition != "Brewed from roasted beans." {
		t.Errorf("an approved call = %+v, want the card written, the approval being the acceptance", written)
	}
}

func TestATemplateToolSaysHowManyKeywordsThePageMustUse(t *testing.T) {
	t.Parallel()

	registry, binding, _ := wired(t)
	binding.Mode = domainagent.ModeAutonomous

	cases := []struct {
		name string
		rule string
		want string
	}{
		{name: "the first three", rule: `,"requiredKeywords":3`, want: `"requiredKeywords":3`},
		{name: "none of them", rule: `,"requiredKeywords":0`, want: `"requiredKeywords":0`},
		{name: "every one when the tool does not say", rule: ``, want: ``},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			spec := strings.Replace(minimalSpec, `"maxDensity":0.02}`, `"maxDensity":0.02`+tc.rule+`}`, 1)
			out, err := registry.Call(t.Context(), binding, "templates",
				json.RawMessage(`{"op":"create","name":"Keyed `+tc.name+`","spec":`+spec+`}`))
			if err != nil {
				t.Fatalf("templates create: %v", err)
			}
			encoded, err := json.Marshal(out)
			if err != nil {
				t.Fatalf("encode the answer: %v", err)
			}
			if tc.want == "" {
				if strings.Contains(string(encoded), "requiredKeywords") {
					t.Fatalf("the template carries a keyword count nobody set: %s", encoded)
				}
				return
			}
			if !strings.Contains(string(encoded), tc.want) {
				t.Fatalf("the template lost %s: %s", tc.want, encoded)
			}
		})
	}
}

func TestATemplateToolDeclaresWhatAProductGets(t *testing.T) {
	t.Parallel()

	registry, binding, _ := wired(t)
	binding.Mode = domainagent.ModeAutonomous

	product := `,"product":{"shortDescription":{"enabled":true,"intent":"Say what it is","targetWords":40,` +
		`"primaryKeyword":true}}}`
	spec := strings.TrimSuffix(minimalSpec, "}") + product
	out, err := registry.Call(t.Context(), binding, "templates",
		json.RawMessage(`{"op":"create","name":"Product outputs","spec":`+spec+`}`))
	if err != nil {
		t.Fatalf("templates create: %v", err)
	}
	encoded, err := json.Marshal(out)
	if err != nil {
		t.Fatalf("encode the answer: %v", err)
	}
	for _, want := range []string{`"shortDescription":{"enabled":true`, `"targetWords":40`} {
		if !strings.Contains(string(encoded), want) {
			t.Errorf("the template lost %s: %s", want, encoded)
		}
	}
	retired := strings.TrimSuffix(minimalSpec, "}") + `,"product":{"specifications":[{"name":"Form"}]}}`
	if _, err = registry.Call(t.Context(), binding, "templates",
		json.RawMessage(`{"op":"create","name":"Retired outputs","spec":`+retired+`}`)); err == nil {
		t.Error("a template naming the retired specifications was taken; the store owns a product's attributes")
	}

	bare, err := registry.Call(t.Context(), binding, "templates",
		json.RawMessage(`{"op":"create","name":"No product outputs","spec":`+minimalSpec+`}`))
	if err != nil {
		t.Fatalf("templates create: %v", err)
	}
	if encoded, err = json.Marshal(bare); err != nil || strings.Contains(string(encoded), `"product":{`) {
		t.Errorf("a template that says nothing of products carries a product block: %s", encoded)
	}
}

func argumentsFor(name string) json.RawMessage {
	switch name {
	case "imports_inspect", "imports_preview":
		return json.RawMessage(`{"path":"C:/nowhere/absent.csv","mapping":{"columns":[]}}`)
	case "templates":
		return json.RawMessage(`{"op":"create","name":"Guide","spec":` + minimalSpec + `}`)
	case "links":
		return json.RawMessage(`{"view":"section"}`)
	case "pages_tree":
		return json.RawMessage(`{"view":"roots"}`)
	case "findings":
		return json.RawMessage(`{"op":"resolve","findingId":"f1","option":"ignore"}`)
	default:
		return json.RawMessage(`{}`)
	}
}
