package runs

import (
	"context"

	"github.com/davidmovas/postulator/internal/application/templates"
	"github.com/davidmovas/postulator/internal/domain/run"
	"github.com/davidmovas/postulator/internal/domain/template"
	"github.com/davidmovas/postulator/internal/kernel/dto"
	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/kernel/paging"
)

type engine interface {
	Enqueue(ctx context.Context, record run.Run) (run.Run, error)
	EstimateRun(ctx context.Context, record run.Run, spec template.TemplateSpec) (run.Estimate, error)
	Pause(ctx context.Context, runID string, reason run.PauseReason) error
	Resume(ctx context.Context, runID string) error
	Cancel(ctx context.Context, runID string) error
	RetryStep(ctx context.Context, itemID string) error
}

type runStore interface {
	Get(ctx context.Context, id string) (run.Run, error)
	List(ctx context.Context, q run.Query, page paging.Request) (paging.List[run.Run], error)
}

type itemStore interface {
	Get(ctx context.Context, id string) (run.Item, error)
	List(ctx context.Context, q run.ItemQuery, page paging.Request) (paging.List[run.Item], error)
}

type artifactStore interface {
	ByItem(ctx context.Context, itemID string) ([]run.Artifact, error)
	PurgedByItems(ctx context.Context, itemIDs []string) (map[string][]run.ArtifactKind, error)
}

type stepDefs interface {
	Lookup(name string) (run.StepDef, bool)
}

type eventStore interface {
	List(ctx context.Context, runID string, sinceSeq int64, limit int) ([]run.Event, error)
}

type specResolver interface {
	ResolveForPage(ctx context.Context, req templates.ResolveForPageRequest) (templates.ResolveForPageResponse, error)
}

type Service struct {
	engine    engine
	runs      runStore
	items     itemStore
	artifacts artifactStore
	events    eventStore
	specs     specResolver
	steps     stepDefs
}

func New(engine engine, runs runStore, items itemStore, artifacts artifactStore, events eventStore,
	specs specResolver, steps stepDefs) *Service {
	return &Service{
		engine: engine, runs: runs, items: items, artifacts: artifacts, events: events, specs: specs, steps: steps,
	}
}

func invalid(message, field string) *errors.Error {
	return errors.New(errors.Invalid, message).WithDetail("field", field)
}

func statusFilter(raw string) (*run.Status, error) {
	if raw == "" {
		return nil, nil
	}
	status := run.Status(raw)
	if !status.Valid() {
		return nil, invalid("run status is not recognized", "status")
	}
	return &status, nil
}

func kindFilter(raw string) (*run.Kind, error) {
	if raw == "" {
		return nil, nil
	}
	kind := run.Kind(raw)
	if !kind.Valid() {
		return nil, invalid("run kind is not recognized", "kind")
	}
	return &kind, nil
}

func sortOf(sort *dto.Sort) (key run.Sort, desc bool, err error) {
	if sort == nil {
		return run.SortCreatedAt, false, nil
	}
	key = run.Sort(sort.Field)
	if !key.Valid() {
		return "", false, invalid("runs are sorted by createdAt or status", "sort.field")
	}
	return key, sort.Desc, nil
}
