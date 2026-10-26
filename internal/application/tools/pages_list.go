package tools

import (
	"context"

	"github.com/davidmovas/postulator/internal/application/pages"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/kernel/dto"
	"github.com/davidmovas/postulator/internal/kernel/paging"
)

const pagesListName = "pages_list"

type pagesListArgs struct {
	dto.ListRequest
	Status             string `json:"status,omitempty" enum:"planned,exists,published,archived" description:"Only this state"`
	EntityID           string `json:"entityId,omitempty" description:"Only pages of this entity"`
	IncludeDescendants bool   `json:"includeDescendants,omitempty" description:"With entityId, also the entities below it"`
	Unmapped           bool   `json:"unmapped,omitempty" description:"Only pages with no entity"`
	PathPrefix         string `json:"pathPrefix,omitempty" description:"Only paths starting with this"`
	PageTypeID         string `json:"pageTypeId,omitempty" description:"Only pages of this type"`
	Under              string `json:"under,omitempty" description:"Only this page and those under it"`
}

func pagesList(deps Deps) Tool {
	return sortedBy(newSiteTool(Def{
		Name:        pagesListName,
		Description: "List the pages of the site, optionally by status, entity, type, subtree or path prefix.",
		Risk:        RiskRead,
	}, func(ctx context.Context, b Binding, in pagesListArgs) (paging.List[pages.Page], error) {
		return deps.Pages.List(ctx, pages.ListRequest{
			ListRequest: in.ListRequest, SiteID: b.SiteID,
			PageFilter: pages.PageFilter{
				Status: in.Status, EntityID: in.EntityID, IncludeDescendants: in.IncludeDescendants, Unmapped: in.Unmapped,
				PathPrefix: in.PathPrefix, PageTypeID: in.PageTypeID, Under: in.Under,
			},
		})
	}), string(pagemap.SortCreatedAt), string(pagemap.SortPath), string(pagemap.SortTitle))
}
