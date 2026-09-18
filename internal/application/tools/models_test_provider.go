package tools

import (
	"context"

	"github.com/davidmovas/postulator/internal/application/models"
)

const modelsTestProviderName = "models_test_provider"

func modelsTestProvider(deps Deps) Tool {
	return NewTool(Def{
		Name:        modelsTestProviderName,
		Description: "Send one short probe to a provider and report its latency and usage.",
		Risk:        RiskWrite,
	}, func(ctx context.Context, _ Binding, in models.TestProviderRequest) (models.TestProviderResponse, error) {
		return deps.Models.TestProvider(ctx, in)
	})
}
