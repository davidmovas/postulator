package tools

import (
	"context"

	"github.com/davidmovas/postulator/internal/application/sitemodel"
	"github.com/davidmovas/postulator/internal/domain/contextpack"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/kernel/paging"
)

const pagesFindName = "pages_find"

type pagesFindArgs struct {
	PageIDs    []string `json:"pageIds,omitempty" description:"These pages"`
	Query      string   `json:"query,omitempty" description:"Words of a title, H1, path or keyword"`
	PageTypeID string   `json:"pageTypeId,omitempty" description:"Only this type, from pagetypes"`
	Under      string   `json:"under,omitempty" description:"Only pages below this page id"`
	EntityID   string   `json:"entityId,omitempty" description:"Only this entity's pages"`
	Status     string   `json:"status,omitempty" enum:"planned,exists,published,archived" description:"Only this state"`
	Gap        string   `json:"gap,omitempty" enum:"planned,empty,unmapped,orphan,missing_links,findings" description:"Only pages with this gap"`
	Cursor     string   `json:"cursor,omitempty" description:"nextCursor of the last call"`
	Limit      int      `json:"limit,omitempty" description:"Default 50, at most 200"`
}

func pagesFind(deps Deps) Tool {
	return newSiteTool(Def{
		Name: pagesFindName,
		Description: "Find pages by ids, words, type, subtree, entity, state or gap, in path order; each with its type, " +
			"parent or why it has none, body state and link and finding counts.",
		Risk: RiskRead,
	}, func(ctx context.Context, b Binding, in pagesFindArgs) (paging.List[contextpack.MapNode], error) {
		return fitted(deps, in.Limit, func(limit int) (paging.List[contextpack.MapNode], error) {
			return deps.SiteModel.Find(ctx, sitemodel.FindRequest{
				SiteID: b.SiteID, PageIDs: in.PageIDs, Query: in.Query, PageTypeID: in.PageTypeID, Under: in.Under,
				EntityID: in.EntityID, Status: pagemap.Status(in.Status), Gap: contextpack.Gap(in.Gap),
				Cursor: paging.Cursor(in.Cursor), Limit: limit,
			})
		})
	})
}
