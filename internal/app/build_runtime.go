package app

import (
	"github.com/davidmovas/postulator/internal/adapters/images"
	"github.com/davidmovas/postulator/internal/adapters/images/localfile"
	"github.com/davidmovas/postulator/internal/adapters/images/wpmedia"
	"github.com/davidmovas/postulator/internal/application/observe"
	"github.com/davidmovas/postulator/internal/domain/run"
	"github.com/davidmovas/postulator/internal/domain/template"
	"github.com/davidmovas/postulator/internal/runtime"
	"github.com/davidmovas/postulator/internal/runtime/scheduler"
	"github.com/davidmovas/postulator/internal/runtime/steps"
)

type runtimeParts struct {
	steps   *run.Registry
	engine  *runtime.Engine
	observe *observe.Service
}

func (c *Core) buildObserver(stores repos, writing authoring) *observe.Service {
	return observe.New(observe.Deps{
		Sites: stores.sites, Pages: stores.pages, Identity: stores.identities, Fields: stores.fields, Bodies: stores.bodies,
		Renders: stores.renders, Links: stores.links, Findings: stores.findings, Types: writing.pageTypes, Maps: writing.siteModel,
		UnitOfWork: stores.store, Publisher: c.Events, Clock: stores.now,
	})
}

func (c *Core) buildRuntime(stores repos, llm llmParts, writing authoring) (runtimeParts, error) {
	observer := c.buildObserver(stores, writing)
	stepRegistry := run.NewRegistry()
	if err := steps.Register(stepRegistry, c.stepDeps(stores, llm, writing, observer)); err != nil {
		return runtimeParts{}, err
	}

	return runtimeParts{
		steps:   stepRegistry,
		observe: observer,
		engine: runtime.New(runtime.Deps{
			Runs:       stores.runs,
			Items:      stores.items,
			Artifacts:  stores.artifacts,
			Execs:      stores.execs,
			Events:     stores.runEvents,
			Pages:      stores.pages,
			Specs:      writing.templates,
			Types:      writing.pageTypes,
			Keys:       stores.secrets,
			Spend:      stores.llmCalls,
			Catalog:    llm.catalog,
			Profiles:   llm.profiles,
			Tuning:     llm.tuning,
			Snapshots:  writing.siteModel,
			UnitOfWork: stores.store,
			Publisher:  c.Events,
		}, stepRegistry, runtime.Settings(stores.values), stores.now, c.logger),
	}, nil
}

func (c *Core) stepDeps(stores repos, llm llmParts, writing authoring, observer *observe.Service) steps.Deps {
	return steps.Deps{
		Entities:      stores.entities,
		Edges:         stores.edges,
		Pages:         stores.pages,
		Links:         stores.links,
		Observe:       observer,
		Bodies:        stores.bodies,
		Fields:        stores.fields,
		Items:         stores.items,
		Artifacts:     stores.artifacts,
		Sites:         stores.sites,
		SiteWriter:    stores.sites,
		WordPress:     writing.wordpress,
		Types:         writing.pageTypes,
		Profiles:      llm.profiles,
		Tuning:        llm.tuning,
		Models:        llm.catalog,
		Pictures:      writing.siteModel,
		Content:       writing.content,
		LLM:           llm.client,
		ImageProvider: llm.images,
		ImageModel:    llm.imageModel,
		ImageSources: map[template.ImageSource]steps.ImageSource{
			template.ImagesWPMedia: wpmedia.New(writing.wordpress),
			template.ImagesLocal:   localfile.New(images.LocalDir(stores.values)),
		},
		UnitOfWork: stores.store,
		Publisher:  c.Events,
		Clock:      stores.now,
		BatchSize:  steps.BatchSize(stores.values),
	}
}

func (c *Core) buildScheduler(stores repos, services useCases) *scheduler.Scheduler {
	return scheduler.New(services.schedules, scheduler.TickInterval(stores.values), c.logger)
}
