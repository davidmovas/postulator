package tools

import (
	"context"

	"github.com/davidmovas/postulator/internal/application/pages"
)

const pagesSetCanonicalName = "pages_set_canonical"

func pagesSetCanonical(deps Deps) Tool {
	return NewTool(Def{
		Name:        pagesSetCanonicalName,
		Description: "Name the page that is the canonical one for an entity.",
		Risk:        RiskWrite,
	}, func(ctx context.Context, _ Binding, in pages.SetCanonicalRequest) (pages.SetCanonicalResponse, error) {
		return deps.Pages.SetCanonical(ctx, in)
	})
}
