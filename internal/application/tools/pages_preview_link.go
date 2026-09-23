package tools

import (
	"context"

	"github.com/davidmovas/postulator/internal/application/pages"
)

const pagesPreviewLinkName = "pages_preview_link"

func pagesPreviewLink(deps Deps) Tool {
	return NewTool(Def{
		Name: pagesPreviewLinkName,
		Description: "Answer where a page can be seen as the site's theme renders it: the public address of a " +
			"published page, or a signed link that works for an hour for a draft. The link shows the draft to anyone who has it.",
		Risk: RiskWrite,
	}, func(ctx context.Context, _ Binding, in pages.PreviewLinkRequest) (pages.PreviewLinkResponse, error) {
		return deps.Pages.PreviewLink(ctx, in)
	})
}
