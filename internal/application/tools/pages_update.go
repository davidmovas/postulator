package tools

import (
	"context"

	"github.com/davidmovas/postulator/internal/application/pages"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
)

const pagesUpdateName = "pages_update"

type pagesUpdateArgs struct {
	pages.UpdateRequest
	PageTypeID *string `json:"pageTypeId,omitempty" description:"New page type id from pagetypes_list"`
}

func (in *pagesUpdateArgs) touchesPage() bool {
	r := &in.UpdateRequest
	return r.Path != nil || r.WPType != nil || r.Title != nil || r.H1 != nil || r.MetaTitle != nil ||
		r.MetaDescription != nil || r.Canonical != nil || r.Keywords != nil || r.Status != nil || r.TemplateID != nil
}

func pagesUpdate(deps Deps) Tool {
	return NewTool(Def{
		Name:        pagesUpdateName,
		Description: "Change the path, titles, meta, template or type of a page.",
		Risk:        RiskWrite,
	}, func(ctx context.Context, _ Binding, in pagesUpdateArgs) (pages.UpdateResponse, error) {
		if in.PageTypeID == nil || in.touchesPage() {
			updated, err := deps.Pages.Update(ctx, in.UpdateRequest)
			if err != nil || in.PageTypeID == nil {
				return updated, err
			}
		}
		typed, err := deps.Pages.SetType(ctx, pages.SetTypeRequest{PageID: in.ID, PageTypeID: *in.PageTypeID, By: pagemap.TypeByAI})
		if err != nil {
			return pages.UpdateResponse{}, err
		}
		return pages.UpdateResponse(typed), nil
	})
}
