package tools

import (
	"context"

	"github.com/davidmovas/postulator/internal/application/sitemodel"
	"github.com/davidmovas/postulator/internal/application/templates"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

const pagesGetName = "pages_get"

type pagesGetArgs struct {
	PageID string `json:"pageId,omitempty" description:"Page id, or give path"`
	Path   string `json:"path,omitempty" description:"Page path"`
}

type TemplateApplied struct {
	ID       string   `json:"id"`
	Name     string   `json:"name"`
	Source   string   `json:"source"`
	Version  int      `json:"version"`
	Headings []string `json:"headings"`
}

type PagesGetResponse struct {
	Page     sitemodel.PageModel `json:"page"`
	Template *TemplateApplied    `json:"template"`
}

func pagesGet(deps Deps) Tool {
	return newSiteTool(Def{
		Name: pagesGetName,
		Description: "Read one page's model: identity, type and its source, parent with reason, fields with current, base " +
			"and owner, the body in sum, neighbors, open findings, and the template a run would use.",
		Risk: RiskRead,
	}, func(ctx context.Context, b Binding, in pagesGetArgs) (PagesGetResponse, error) {
		model, err := deps.SiteModel.Page(ctx, sitemodel.PageRequest{SiteID: b.SiteID, PageID: in.PageID, Path: in.Path})
		if err != nil {
			return PagesGetResponse{}, err
		}
		answered := PagesGetResponse{Page: model}
		resolved, err := deps.Templates.ResolveForPage(ctx, templates.ResolveForPageRequest{PageID: model.Page.PageID})
		switch {
		case errors.IsCode(err, errors.NotFound):
		case err != nil:
			return PagesGetResponse{}, err
		default:
			applied := &TemplateApplied{
				ID: resolved.TemplateID, Name: resolved.Name, Source: resolved.Source, Version: resolved.Version,
				Headings: make([]string, 0, len(resolved.Spec.Sections)),
			}
			for _, section := range resolved.Spec.Sections {
				applied.Headings = append(applied.Headings, section.Heading)
			}
			answered.Template = applied
		}
		return answered, nil
	})
}
