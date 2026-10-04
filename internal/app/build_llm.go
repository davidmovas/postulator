package app

import (
	"github.com/davidmovas/postulator/internal/adapters/images"
	"github.com/davidmovas/postulator/internal/adapters/images/metered"
	imageopenai "github.com/davidmovas/postulator/internal/adapters/images/openai"
	"github.com/davidmovas/postulator/internal/adapters/llm/catalog"
	"github.com/davidmovas/postulator/internal/adapters/llm/ledger"
	"github.com/davidmovas/postulator/internal/adapters/llm/limiter"
	"github.com/davidmovas/postulator/internal/adapters/llm/openai"
	"github.com/davidmovas/postulator/internal/adapters/llm/profiles"
	"github.com/davidmovas/postulator/internal/adapters/llm/recordreplay"
	"github.com/davidmovas/postulator/internal/adapters/llm/retry"
	"github.com/davidmovas/postulator/internal/adapters/llm/tuning"
	llmport "github.com/davidmovas/postulator/internal/application/llm"
	"github.com/davidmovas/postulator/internal/application/tools"
	domainllm "github.com/davidmovas/postulator/internal/domain/llm"
)

type llmParts struct {
	catalog    *catalog.Catalog
	profiles   *profiles.Profiles
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
	book := ledger.New(
		recordreplay.New(c.provider(stores, modelCatalog, policy), recordreplay.Mode(stores.values),
			recordreplay.DefaultDir, tools.Redact),
		stores.llmCalls, modelCatalog, c.Events, stores.now,
	)
	imageModel := images.OpenAIModel(stores.values)

	return llmParts{
		catalog:  modelCatalog,
		profiles: profiles.New(stores.profiles, stores.sites, modelCatalog, stores.now),
		ledger:   book,
		tuning:   policy,
		client: tuning.New(
			retry.New(limiter.New(book, modelCatalog), retry.Retries(stores.values), retry.DefaultBackoff),
			policy,
		),
		images: metered.New(
			imageopenai.New(stores.secrets, imageModel, imageopenai.WithQuality(images.OpenAIQuality(stores.values))),
			domainllm.ModelRef{Provider: domainllm.ProviderOpenAI, Model: imageModel},
			stores.llmCalls, modelCatalog, c.Events, stores.now,
		),
		imageModel: &domainllm.ModelRef{Provider: domainllm.ProviderOpenAI, Model: imageModel},
	}, nil
}

func (c *Core) provider(stores repos, modelCatalog *catalog.Catalog, policy *tuning.Policy) llmport.Client {
	if c.cfg.Provider != nil {
		return c.cfg.Provider
	}
	return openai.New(stores.secrets, modelCatalog,
		openai.WithBaseURL(openai.BaseURL(stores.values)),
		openai.WithTimeout(openai.Timeout(stores.values)),
		openai.WithFlexPatience(policy.FlexPatience()),
	)
}
