package tools

import (
	"context"

	"github.com/davidmovas/postulator/internal/application/templates"
	"github.com/davidmovas/postulator/internal/kernel/paging"
)

const policiesListName = "policies_list"

func policiesList(deps Deps) Tool {
	return NewTool(Def{
		Name:        policiesListName,
		Description: "List the link policies, global and site scoped.",
		Risk:        RiskRead,
	}, func(ctx context.Context, _ Binding, in templates.ListPoliciesRequest) (paging.List[templates.LinkPolicy], error) {
		return deps.Templates.ListPolicies(ctx, in)
	})
}
