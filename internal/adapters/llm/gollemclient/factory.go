package gollemclient

import (
	"context"
	"sync"

	"github.com/gollem-dev/gollem"
	"github.com/gollem-dev/gollem/llm/claude"
	"github.com/gollem-dev/gollem/llm/gemini"
	"github.com/gollem-dev/gollem/llm/openai"

	"github.com/davidmovas/postulator/internal/domain/llm"
	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/kernel/settings"
)

const (
	ProviderOpenAI    = "openai"
	ProviderAnthropic = "anthropic"
	ProviderGemini    = "gemini"
)

func SecretRef(provider string) string {
	return "llm:" + provider + ":api_key"
}

type secretReader interface {
	Get(ctx context.Context, ref string) (string, error)
}

type Factory struct {
	secrets secretReader
	values  *settings.Values
	clients map[string]gollem.LLMClient
	mu      sync.Mutex
}

func NewFactory(secrets secretReader, values *settings.Values) *Factory {
	return &Factory{secrets: secrets, values: values, clients: make(map[string]gollem.LLMClient)}
}

func (f *Factory) New(ctx context.Context, ref llm.ModelRef) (gollem.LLMClient, error) {
	if !ref.Valid() {
		return nil, errors.New(errors.Invalid, "a model reference must name a provider and a model")
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	if client, ok := f.clients[ref.String()]; ok {
		return client, nil
	}

	client, err := f.build(ctx, ref)
	if err != nil {
		return nil, err
	}
	f.clients[ref.String()] = client
	return client, nil
}

func (f *Factory) build(ctx context.Context, ref llm.ModelRef) (gollem.LLMClient, error) {
	switch ref.Provider {
	case ProviderOpenAI:
		key, err := f.key(ctx, ref.Provider)
		if err != nil {
			return nil, err
		}
		options := []openai.Option{openai.WithModel(ref.Model)}
		if base := openaiBaseURL.Get(f.values); base != "" {
			options = append(options, openai.WithBaseURL(base))
		}
		return wrap(openai.New(ctx, key, options...))
	case ProviderAnthropic:
		key, err := f.key(ctx, ref.Provider)
		if err != nil {
			return nil, err
		}
		options := []claude.Option{claude.WithModel(ref.Model), claude.WithTimeout(timeoutSetting.Get(f.values))}
		if base := anthropicBaseURL.Get(f.values); base != "" {
			options = append(options, claude.WithBaseURL(base))
		}
		return wrap(claude.New(ctx, key, options...))
	case ProviderGemini:
		project, location := geminiProject.Get(f.values), geminiLocation.Get(f.values)
		if project == "" || location == "" {
			return nil, errors.New(errors.Invalid, "gemini needs llm.gemini.projectId and llm.gemini.location to be set").
				WithDetail("provider", ref.Provider)
		}
		return wrap(gemini.New(ctx, project, location, gemini.WithModel(ref.Model)))
	default:
		return nil, errors.New(errors.Invalid, "this provider is not supported").WithDetail("provider", ref.Provider)
	}
}

func wrap[T gollem.LLMClient](client T, err error) (gollem.LLMClient, error) {
	if err != nil {
		return nil, errors.Wrap(err, errors.External, "build the provider client")
	}
	return client, nil
}

func (f *Factory) key(ctx context.Context, provider string) (string, error) {
	key, err := f.secrets.Get(ctx, SecretRef(provider))
	if err != nil {
		if errors.IsCode(err, errors.NotFound) {
			return "", errors.New(errors.Unauthorized, "no api key is stored for this provider").WithDetail("provider", provider)
		}
		return "", err
	}
	if key == "" {
		return "", errors.New(errors.Unauthorized, "the stored api key for this provider is empty").WithDetail("provider", provider)
	}
	return key, nil
}
