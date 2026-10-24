package tools

import (
	"context"

	"github.com/davidmovas/postulator/internal/application/pagetypes"
)

const pagetypesListName = "pagetypes_list"

type pagetypesListArgs struct {
	SiteID string `json:"siteId" description:"Site id"`
}

func pagetypesList(deps Deps) Tool {
	return NewTool(Def{
		Name:        pagetypesListName,
		Description: "List a site's page types with their pages, field scopes and link policies, and the field catalog.",
		Risk:        RiskRead,
	}, func(ctx context.Context, _ Binding, in pagetypesListArgs) (pagetypes.ListResponse, error) {
		return deps.PageTypes.List(ctx, pagetypes.ListRequest{SiteID: in.SiteID})
	})
}
