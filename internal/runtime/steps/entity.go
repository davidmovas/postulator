package steps

import (
	"context"

	"github.com/davidmovas/postulator/internal/domain/graph"
	"github.com/davidmovas/postulator/internal/domain/run"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

func entityOf(ctx context.Context, deps Deps, sc *run.StepContext) (graph.Entity, error) {
	if sc.Page.EntityID == nil {
		return graph.Entity{}, errors.New(errors.Invalid, "the page is not mapped to an entity, so it has no graph context").
			WithDetail("pageId", sc.Page.ID).
			WithDetail("path", sc.Page.Path)
	}

	entities, err := deps.Entities.ListBySite(ctx, sc.Run.SiteID)
	if err != nil {
		return graph.Entity{}, err
	}
	for i := range entities {
		if entities[i].ID == *sc.Page.EntityID {
			entity := entities[i]
			entity.Name = graph.Labels(entities)[entity.ID]
			return entity, nil
		}
	}
	return graph.Entity{}, errors.New(errors.NotFound, "the entity the page is mapped to is gone").
		WithDetail("entityId", *sc.Page.EntityID)
}
