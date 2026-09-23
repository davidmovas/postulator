package tools

import (
	"context"

	"github.com/davidmovas/postulator/internal/application/models"
)

const modelsUsageSummaryName = "models_usage_summary"

func modelsUsageSummary(deps Deps) Tool {
	return NewTool(Def{
		Name:        modelsUsageSummaryName,
		Description: "Read the token and dollar spend of a run or of a conversation.",
		Risk:        RiskRead,
	}, func(ctx context.Context, _ Binding, in models.UsageSummaryRequest) (models.UsageSummaryResponse, error) {
		return deps.Models.UsageSummary(ctx, in)
	})
}
