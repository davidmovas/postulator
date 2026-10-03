package app

import (
	"context"

	"github.com/gollem-dev/gollem"

	"github.com/davidmovas/postulator/internal/adapters/images"
	"github.com/davidmovas/postulator/internal/adapters/images/metered"
	imageopenai "github.com/davidmovas/postulator/internal/adapters/images/openai"
	"github.com/davidmovas/postulator/internal/adapters/llm/catalog"
	"github.com/davidmovas/postulator/internal/adapters/llm/gollemclient"
	"github.com/davidmovas/postulator/internal/adapters/llm/ledger"
	"github.com/davidmovas/postulator/internal/adapters/llm/limiter"
	"github.com/davidmovas/postulator/internal/adapters/llm/profiles"
	"github.com/davidmovas/postulator/internal/adapters/llm/recordreplay"
	"github.com/davidmovas/postulator/internal/adapters/llm/retry"
	"github.com/davidmovas/postulator/internal/adapters/llm/tuning"
	llmport "github.com/davidmovas/postulator/internal/application/llm"
	"github.com/davidmovas/postulator/internal/application/tools"
	domainllm "github.com/davidmovas/postulator/internal/domain/llm"
)

type AgentProvider interface {
	New(ctx context.Context, ref domainllm.ModelRef) (gollem.LLMClient, error)
	NewForTools(ctx context.Context, ref domainllm.ModelRef) (gollem.LLMClient, error)
}

type llmParts struct {
	catalog    *catalog.Catalog
	profiles   *profiles.Profiles
	providers  AgentProvider
	ledger     *ledger.Ledger
	tuning     *tuning.Policy
	client     *tuning.Client
	images     *metered.Images
	imageModel *domainllm.ModelRef
}

func (c *Core) buildLLM(stores repos) (llmParts, error) {
	modelCatalog, err := catalog.New(stores.overrides)
	if err != nil {
		return llmParts{}, err
	}

	policy := tuning.NewPolicy(stores.values)
	providers := c.agentProvider(stores, modelCatalog)
	book := ledger.New(
		recordreplay.New(c.provider(stores, providers, modelCatalog), recordreplay.Mode(stores.values),
			recordreplay.DefaultDir, tools.Redact),
		stores.llmCalls, modelCatalog, c.Events, stores.now,
	)
	imageModel := images.OpenAIModel(stores.values)

	return llmParts{
		catalog:   modelCatalog,
		profiles:  profiles.New(stores.profiles, stores.sites, modelCatalog, stores.now),
		providers: providers,
		ledger:    book,
		tuning:    policy,
		client: tuning.New(
			retry.New(limiter.New(book, modelCatalog), retry.Retries(stores.values), retry.DefaultBackoff),
			policy,
		),
		images: metered.New(
			imageopenai.New(stores.secrets, imageModel, imageopenai.WithQuality(images.OpenAIQuality(stores.values))),
			domainllm.ModelRef{Provider: imageopenai.Provider, Model: imageModel},
			stores.llmCalls, modelCatalog, c.Events, stores.now,
		),
		imageModel: &domainllm.ModelRef{Provider: imageopenai.Provider, Model: imageModel},
	}, nil
}

func (c *Core) agentProvider(stores repos, modelCatalog *catalog.Catalog) AgentProvider {
	if c.cfg.AgentProvider != nil {
		return c.cfg.AgentProvider
	}
	return gollemclient.NewFactory(stores.secrets, modelCatalog, stores.values)
}

func (c *Core) provider(stores repos, providers AgentProvider, modelCatalog *catalog.Catalog) llmport.Client {
	if c.cfg.Provider != nil {
		return c.cfg.Provider
	}
	return gollemclient.New(providers, modelCatalog, gollemclient.Timeout(stores.values))
}
