package tools

import (
	"context"

	"github.com/davidmovas/postulator/internal/application/sitemodel"
	"github.com/davidmovas/postulator/internal/domain/contextpack"
	"github.com/davidmovas/postulator/internal/domain/field"
	"github.com/davidmovas/postulator/internal/kernel/paging"
)

const (
	sitesExploreName = "sites_explore"
	bytesPerToken    = 4
)

type sitesExploreArgs struct {
	View    string `json:"view" enum:"page,tree,links,gaps,pack" description:"tree lists a page's children, the top level without a page"`
	PageID  string `json:"pageId,omitempty" description:"Page id; page, links and pack need it or path"`
	Path    string `json:"path,omitempty" description:"The page's path, used when pageId is empty"`
	Field   string `json:"field,omitempty" enum:"title,excerpt,body,seo_title,seo_description,product_description,product_short_description,variation_description,image_alt,image_title" description:"pack: default the body"`
	MetaKey string `json:"metaKey,omitempty" description:"pack: a custom field's meta key, in place of field"`
	Gap     string `json:"gap,omitempty" enum:"planned,empty,unmapped,orphan,missing_links,findings" description:"gaps: only pages with this gap"`
	Cursor  string `json:"cursor,omitempty" description:"tree, gaps: nextCursor of the last call"`
	Limit   int    `json:"limit,omitempty" description:"tree, gaps: default 50, at most 200"`
}

func sitesExplore(deps Deps) Tool {
	return newSiteTool(Def{
		Name: sitesExploreName,
		Description: "Read the site's model: a page's facts, the page tree, a page's owed links, the pages with gaps, " +
			"or the pack the writer reads for a field.",
		Risk: RiskRead,
	}, func(ctx context.Context, b Binding, in sitesExploreArgs) (sitemodel.ExploreResponse, error) {
		req := sitemodel.ExploreRequest{
			SiteID: b.SiteID, View: sitemodel.View(in.View), PageID: in.PageID, Path: in.Path, Field: in.Field,
			Gap: contextpack.Gap(in.Gap), Cursor: paging.Cursor(in.Cursor), Limit: in.Limit,
		}
		if in.MetaKey != "" {
			req.Field = string(field.Instance(field.Meta, in.MetaKey))
		}
		if deps.ResultBytes != nil {
			req.Budget = deps.ResultBytes() / bytesPerToken
		}
		return deps.SiteModel.Explore(ctx, req)
	})
}
