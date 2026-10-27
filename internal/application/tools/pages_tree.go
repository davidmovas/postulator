package tools

import (
	"context"

	"github.com/davidmovas/postulator/internal/application/sitemodel"
	"github.com/davidmovas/postulator/internal/kernel/paging"
)

const pagesTreeName = "pages_tree"

type pagesTreeArgs struct {
	View       string `json:"view" enum:"roots,children,subtree,ancestors,siblings,walk" description:"walk is the whole tree, a parent before its children"`
	PageID     string `json:"pageId,omitempty" description:"The page it starts from; roots and walk take none"`
	PageTypeID string `json:"pageTypeId,omitempty" description:"Only pages of this type"`
	Depth      int    `json:"depth,omitempty" description:"subtree: levels below the page, default 2, at most 32"`
	Generation int64  `json:"generation,omitempty" description:"generation of the last call; stale then says start over"`
	Cursor     string `json:"cursor,omitempty" description:"nextCursor of the last call"`
	Limit      int    `json:"limit,omitempty" description:"Default 50, at most 200"`
}

func pagesTree(deps Deps) Tool {
	return newSiteTool(Def{
		Name: pagesTreeName,
		Description: "Read the page tree the site shows (breadcrumbs, then WordPress parents, then paths): each page " +
			"with its depth, parent or why it has none, type, entity and counts.",
		Risk: RiskRead,
	}, func(ctx context.Context, b Binding, in pagesTreeArgs) (sitemodel.TreeAnswer, error) {
		return fitted(deps, in.Limit, func(limit int) (sitemodel.TreeAnswer, error) {
			return deps.SiteModel.Tree(ctx, sitemodel.TreeRequest{
				SiteID: b.SiteID, Shape: sitemodel.TreeShape(in.View), PageID: in.PageID, PageTypeID: in.PageTypeID,
				Cursor: paging.Cursor(in.Cursor), Generation: in.Generation, Depth: in.Depth, Limit: limit,
			})
		})
	})
}
