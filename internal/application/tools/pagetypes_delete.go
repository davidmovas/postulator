package tools

import (
	"context"

	"github.com/davidmovas/postulator/internal/application/pagetypes"
)

const pagetypesDeleteName = "pagetypes_delete"

type pagetypesDeleteArgs struct {
	ID         string `json:"id" description:"Page type id"`
	ReassignTo string `json:"reassignTo" description:"Page type id its pages move to"`
}

func pagetypesDelete(deps Deps) Tool {
	return NewTool(Def{
		Name:        pagetypesDeleteName,
		Description: "Delete a page type, moving its pages to another type of the site.",
		Risk:        RiskDangerous,
	}, func(ctx context.Context, _ Binding, in pagetypesDeleteArgs) (pagetypes.DeleteResponse, error) {
		return deps.PageTypes.Delete(ctx, pagetypes.DeleteRequest{ID: in.ID, ReassignTo: in.ReassignTo})
	})
}
