package app

import (
	"github.com/davidmovas/postulator/internal/adapters/images"
	"github.com/davidmovas/postulator/internal/adapters/images/localfile"
	"github.com/davidmovas/postulator/internal/adapters/images/wpmedia"
	"github.com/davidmovas/postulator/internal/domain/run"
	"github.com/davidmovas/postulator/internal/domain/template"
	"github.com/davidmovas/postulator/internal/runtime"
	"github.com/davidmovas/postulator/internal/runtime/scheduler"
	"github.com/davidmovas/postulator/internal/runtime/steps"
)

type runtimeParts struct {
	steps  *run.Registry
	engine *runtime.Engine
}

func (c *Core) buildRuntime(stores repos, llm llmParts, writing authoring) (runtimeParts, error) {
	stepRegistry := run.NewRegistry()
	if err := steps.Register(stepRegistry, c.stepDeps(stores, llm, writing)); err != nil {
		return runtimeParts{}, err
	}

	return runtimeParts{
		steps: stepRegistry,
		engine: runtime.New(runtime.Deps{
			Runs:       stores.runs,
			Items:      stores.items,
			Artifacts:  stores.artifacts,
			Execs:      stores.execs,
			Events:     stores.runEvents,
			Pages:      stores.pages,
			Specs:      writing.templates,
			Keys:       stores.secrets,
			Spend:      stores.llmCalls,
			Catalog:    llm.catalog,
			Profiles:   llm.profiles,
			Tuning:     llm.tuning,
			UnitOfWork: stores.store,
			Publisher:  c.Events,
		}, stepRegistry, runtime.Settings(stores.values), stores.now, c.logger),
	}, nil
}

func (c *Core) stepDeps(stores repos, llm llmParts, writing authoring) steps.Deps {
	return steps.Deps{
		Entities:      stores.entities,
		Edges:         stores.edges,
		Pages:         stores.pages,
		Links:         stores.links,
		Items:         stores.items,
		Artifacts:     stores.artifacts,
		Sites:         stores.sites,
		SiteWriter:    stores.sites,
		WordPress:     writing.wordpress,
		Policies:      writing.templates,
		Profiles:      llm.profiles,
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
