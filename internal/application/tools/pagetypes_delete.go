package tools

import (
	"context"
	"strconv"

	"github.com/davidmovas/postulator/internal/application/pagetypes"
)

const pagetypesDeleteName = "pagetypes_delete"

type pagetypesDeleteArgs struct {
	ID         string `json:"id" description:"Page type id"`
	ReassignTo string `json:"reassignTo" description:"Page type id its pages move to"`
}

func pagetypesDelete(deps Deps) Tool {
	return confirming(NewTool(Def{
		Name:        pagetypesDeleteName,
		Description: "Delete a page type, moving its pages to another type of the site.",
		Risk:        RiskDangerous,
	}, func(ctx context.Context, _ Binding, in pagetypesDeleteArgs) (pagetypes.DeleteResponse, error) {
		return deps.PageTypes.Delete(ctx, pagetypes.DeleteRequest{ID: in.ID, ReassignTo: in.ReassignTo})
	}), func(ctx context.Context, _ Binding, in pagetypesDeleteArgs) (string, error) {
		target, err := deps.PageTypes.Get(ctx, pagetypes.GetRequest{ID: in.ReassignTo})
		if err != nil {
			return "", err
		}
		fields := rewrittenFields(target)
		if len(fields) == 0 {
			return "", nil
		}
		leaving, err := deps.PageTypes.Get(ctx, pagetypes.GetRequest{ID: in.ID})
		if err != nil || leaving.Pages == 0 {
			return "", err
		}
		return "Delete the page type " + leaving.Name + " and move its " + strconv.Itoa(leaving.Pages) + " pages to " +
			whoseScopeRewrites(target, fields) + ": their own text will be rewritten by the next run", nil
	})
}
