package tools

import (
	"context"

	"github.com/davidmovas/postulator/internal/application/sitemodel"
	"github.com/davidmovas/postulator/internal/kernel/paging"
)

const linksName = "links"

type linksArgs struct {
	View    string `json:"view" enum:"page,section,map,owed,orphans,suggest" description:"page: one page's links in and out; section its pages' totals; map the links between them; owed what the graph asks of them; suggest the parent edges and links to add"`
	PageID  string `json:"pageId,omitempty" description:"page: the page; the others: the section's root, the whole site without one"`
	Missing bool   `json:"missing,omitempty" description:"owed: only the links still missing"`
	Cursor  string `json:"cursor,omitempty" description:"nextCursor of the last call"`
	Limit   int    `json:"limit,omitempty" description:"Default 50, at most 200"`
}

func linksTool(deps Deps) Tool {
	return newSiteTool(Def{
		Name: linksName,
		Description: "Read the site's contextual links, the ones written in a page's own text: a page's in and out, a " +
			"section's, what the entity graph owes and what is missing, the orphans, and what to add.",
		Risk: RiskRead,
	}, func(ctx context.Context, b Binding, in linksArgs) (sitemodel.LinksAnswer, error) {
		return fitted(deps, in.Limit, func(limit int) (sitemodel.LinksAnswer, error) {
			return deps.SiteModel.Links(ctx, sitemodel.LinksRequest{
				SiteID: b.SiteID, View: sitemodel.LinksShape(in.View), PageID: in.PageID, Missing: in.Missing,
				Cursor: paging.Cursor(in.Cursor), Limit: limit,
			})
		})
	})
}
