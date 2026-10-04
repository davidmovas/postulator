package tools

import (
	"context"

	"github.com/davidmovas/postulator/internal/application/pages"
)

const pagesPreviewLinkName = "pages_preview_link"

func pagesPreviewLink(deps Deps) Tool {
	return NewTool(Def{
		Name: pagesPreviewLinkName,
		Description: "Link to a page as the theme renders it: its public address, or for a draft a signed " +
			"link that shows it to anyone holding it for an hour.",
		Risk: RiskWrite,
	}, func(ctx context.Context, _ Binding, in pages.PreviewLinkRequest) (pages.PreviewLinkResponse, error) {
		return deps.Pages.PreviewLink(ctx, in)
	})
}
