package reports

import (
	"context"

	"github.com/davidmovas/postulator/internal/domain/graph"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/run"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

const (
	topEntitiesCap = 10
	recentItems    = 20
)

type entityReader interface {
	ListBySite(ctx context.Context, siteID string) ([]graph.Entity, error)
}

type edgeReader interface {
	ListBySite(ctx context.Context, siteID string) ([]graph.Edge, error)
}

type pageReader interface {
	ListBySite(ctx context.Context, siteID string) ([]pagemap.Page, error)
	Get(ctx context.Context, id string) (pagemap.Page, error)
}

type linkReader interface {
	ListBySite(ctx context.Context, siteID string) ([]pagemap.PageLink, error)
}

type runReader interface {
	Get(ctx context.Context, id string) (run.Run, error)
}

type itemReader interface {
	ByRun(ctx context.Context, runID string) ([]run.Item, error)
	ByTarget(ctx context.Context, targetID string, limit int) ([]run.Item, error)
}

type artifactReader interface {
	ByItem(ctx context.Context, itemID string) ([]run.Artifact, error)
}

type Service struct {
	entities  entityReader
	edges     edgeReader
	pages     pageReader
	links     linkReader
	runs      runReader
	items     itemReader
	artifacts artifactReader
}

func New(entities entityReader, edges edgeReader, pages pageReader, links linkReader,
	runs runReader, items itemReader, artifacts artifactReader) *Service {
	return &Service{
		entities: entities, edges: edges, pages: pages, links: links,
		runs: runs, items: items, artifacts: artifacts,
	}
}

func invalid(message, field string) *errors.Error {
	return errors.New(errors.Invalid, message).WithDetail("field", field)
}
