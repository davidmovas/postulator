package app

import (
	"github.com/davidmovas/postulator/internal/adapters/browser/tor"
	"github.com/davidmovas/postulator/internal/adapters/importer"
	"github.com/davidmovas/postulator/internal/adapters/wp"
	"github.com/davidmovas/postulator/internal/adapters/wp/plugin"
	"github.com/davidmovas/postulator/internal/adapters/wp/registry"
	"github.com/davidmovas/postulator/internal/application/browser"
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
)

type authoring struct {
	templates *templates.Service
	pageTypes *pagetypes.Service
	siteModel *sitemodel.Service
	reports   *reports.Service
	wordpress *registry.Registry
	content   *content.Service
}

type useCases struct {
	sites     *sites.Service
	graph     *graph.Service
	pages     *pages.Service
	imports   *imports.Service
	findings  *findings.Service
	models    *models.Service
	runs      *runs.Service
	sync      *sync.Service
	reports   *reports.Service
	schedules *schedules.Service
	browser   *browser.Service
}

func (c *Core) buildAuthoring(stores repos, llm llmParts) authoring {
	templateService := templates.New(stores.templates, stores.pages, stores.entities, stores.sites, stores.pageTypes,
		stores.store, c.Events, stores.now)
	wordpress := registry.New(stores.sites, stores.secrets, append(wp.FromSettings(stores.values), c.cfg.WordPress...)...)
	pageTypes := pagetypes.New(pagetypes.Deps{
		Types: stores.pageTypes, Scopes: stores.scopes, Pages: stores.pages, Templates: stores.templates,
		Sites: stores.sites, UnitOfWork: stores.store, Publisher: c.Events, Clock: stores.now,
	})

	reportsService := reports.New(reports.Deps{
		Entities: stores.entities, Edges: stores.edges, Pages: stores.pages, Fields: stores.fields, Links: stores.links,
		Runs: stores.runs, Items: stores.items, Artifacts: stores.artifacts,
		Sites: stores.sites, Specs: templateService, Types: pageTypes,
	})
	siteModel := sitemodel.New(sitemodel.Deps{
		Budgets: llm.tuning, Audits: reportsService, Reader: stores.store, UnitOfWork: stores.store, Maps: stores.pageMaps,
		Sites: stores.sites, Pages: stores.pages,
		Entities: stores.entities, Edges: stores.edges, Types: pageTypes, Terms: stores.identities, Fields: stores.fields,
		Bodies: stores.bodies, Renders: stores.renders, Links: stores.links, Findings: stores.findings,
	})

	return authoring{
		templates: templateService,
		pageTypes: pageTypes,
		siteModel: siteModel,
		reports:   reportsService,
		wordpress: wordpress,
		content: content.New(content.Deps{
			Pages: stores.pages, Entities: stores.entities, Edges: stores.edges, Fields: stores.fields, Specs: templateService,
			Types: pageTypes, Profiles: llm.profiles, Raw: rawContent{clients: wordpress}, Pictures: siteModel, LLM: llm.client,
		}),
	}
}

func (c *Core) buildServices(stores repos, llm llmParts, writing authoring, running runtimeParts) useCases {
	values := stores.values
	pagesService := pages.New(pages.Deps{
		Pages: stores.pages, Types: stores.pageTypes, Links: stores.links, Entities: stores.entities, Edges: stores.edges,
		Sites: stores.sites, Bodies: stores.bodies, Findings: stores.findings, UnitOfWork: stores.store, Publisher: c.Events,
		Clock: stores.now, Preview: previewIssuer{clients: writing.wordpress},
	})
	runsService := runs.New(running.engine, stores.runs, stores.items, stores.artifacts, stores.runEvents,
		writing.templates, stores.pages, running.steps, pagesService)
	graphService := graph.New(graph.Deps{
		Entities: stores.entities, Edges: stores.edges, Sites: stores.sites, Pages: stores.pages, Findings: stores.findings,
		Maps: writing.siteModel, Identities: stores.identities, Fields: stores.fields, Work: stores.items, Profiles: llm.profiles,
		LLM: llm.client, UnitOfWork: stores.store,
		Publisher: c.Events, Clock: stores.now,
	})
	importsService := imports.New(imports.Deps{
		Tables:     importer.New(),
		Entities:   stores.entities,
		Edges:      stores.edges,
		Pages:      stores.pages,
		Types:      stores.pageTypes,
		Fields:     stores.fields,
		Mappings:   stores.mappings,
		Sites:      stores.sites,
		Findings:   stores.findings,
		Terms:      stores.identities,
		Items:      stores.items,
		UnitOfWork: stores.store,
		Publisher:  c.Events,
		Clock:      stores.now,
		MaxRows:    imports.MaxRows(values),
	})

	return useCases{
		sites:   sites.New(stores.sites, stores.secrets, stores.store, writing.pageTypes, writing.wordpress, c.Events, stores.now),
		graph:   graphService,
		pages:   pagesService,
		imports: importsService,
		findings: findings.New(findings.Deps{
			Findings: stores.findings, Fields: stores.fields, Bodies: stores.bodies, Pages: stores.pages,
			Cards: graphService, Rows: importsService, Places: writing.siteModel, UnitOfWork: stores.store, Publisher: c.Events,
			Clock: stores.now,
		}),
		models: models.New(llm.catalog, stores.overrides, llm.profiles, llm.ledger, stores.secrets, llm.client,
			c.Events, stores.now),
		runs:    runsService,
		sync:    sync.New(running.engine, stores.sites, writing.wordpress, packer{}, stores.now),
		reports: writing.reports,
		schedules: schedules.New(schedules.Deps{
			Schedules: stores.schedules, Pages: stores.pages, Sites: stores.sites, Runs: runsService,
			RunReader: stores.runs, Publisher: c.Events, Clock: stores.now,
		}),
		browser: browser.New(browser.Deps{
			Browser: tor.New(c.cfg.environment()),
			TorPath: func() string { return tor.Path(values) },
		}),
	}
}

type packer struct{}

func (packer) Package() ([]byte, error) {
	return plugin.Package()
}
