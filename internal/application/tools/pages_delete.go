package tools

import (
	"context"

	"github.com/davidmovas/postulator/internal/application/pages"
)

const pagesDeleteName = "pages_delete"

func pagesDelete(deps Deps) Tool {
	return confirming(NewTool(Def{
		Name:        pagesDeleteName,
		Description: "Delete a page from the page map, and optionally move it to the WordPress trash.",
		Risk:        RiskDangerous,
	}, func(ctx context.Context, _ Binding, in pages.DeleteRequest) (pages.DeleteResponse, error) {
		return deps.Pages.Delete(ctx, in)
	}), func(ctx context.Context, _ Binding, in pages.DeleteRequest) (string, error) {
		if !in.OnSite {
			return "", nil
		}
		held, err := deps.Pages.Get(ctx, pages.GetRequest{ID: in.ID})
		if err != nil {
			return "", err
		}
		return "Move " + held.Page.Path + " to the WordPress trash: it leaves the site for its visitors until it is " +
			"restored there", nil
	})
}
