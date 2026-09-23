package tools

import (
	"context"

	"github.com/davidmovas/postulator/internal/application/pages"
)

const pagesReplaceLinksName = "pages_replace_links"

func pagesReplaceLinks(deps Deps) Tool {
	return NewTool(Def{
		Name:        pagesReplaceLinksName,
		Description: "Replace the recorded outgoing links of a page.",
		Risk:        RiskWrite,
	}, func(ctx context.Context, _ Binding, in pages.ReplaceLinksRequest) (pages.ReplaceLinksResponse, error) {
		return deps.Pages.ReplaceLinks(ctx, in)
	})
}
