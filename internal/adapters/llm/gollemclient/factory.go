package gollemclient

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
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

	ProviderGeminiOpenAI = "gemini-openai"

	toolsSuffix = "|tools"
)

func SecretRef(provider string) string {
	return llm.SecretRef(provider)
}

type secretReader interface {
	Get(ctx context.Context, ref string) (string, error)
}

type cached struct {
	client      gollem.LLMClient
	fingerprint string
}

type Factory struct {
	secrets secretReader
	models  ModelReader
	values  *settings.Values
	clients map[string]cached
	mu      sync.Mutex
}

func NewFactory(secrets secretReader, models ModelReader, values *settings.Values) *Factory {
	return &Factory{secrets: secrets, models: models, values: values, clients: make(map[string]cached)}
}

func (f *Factory) New(ctx context.Context, ref llm.ModelRef) (gollem.LLMClient, error) {
	return f.client(ctx, ref, false)
}

func (f *Factory) NewForTools(ctx context.Context, ref llm.ModelRef) (gollem.LLMClient, error) {
	return f.client(ctx, ref, true)
}

func (f *Factory) client(ctx context.Context, ref llm.ModelRef, carriesTools bool) (gollem.LLMClient, error) {
	if !ref.Valid() {
		return nil, errors.New(errors.Invalid, "a model reference must name a provider and a model")
	}
	mark := f.fingerprint(ctx, ref.Provider)
	key := ref.String()
	if carriesTools {
		key += toolsSuffix
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	if held, ok := f.clients[key]; ok && held.fingerprint == mark {
		return held.client, nil
	}

	client, err := f.build(ctx, ref, carriesTools)
	if err != nil {
		return nil, err
	}
	f.clients[key] = cached{client: client, fingerprint: mark}
	return client, nil
}

func (f *Factory) fingerprint(ctx context.Context, provider string) string {
	if provider == ProviderGemini {
		return ""
	}

	key, err := f.secrets.Get(ctx, SecretRef(provider))
	if err != nil || key == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(key))
	return hex.EncodeToString(sum[:])
}

func (f *Factory) build(ctx context.Context, ref llm.ModelRef, carriesTools bool) (gollem.LLMClient, error) {
	switch ref.Provider {
	case ProviderOpenAI:
		return f.compatible(ctx, ref, openaiBaseURL.Get(f.values), carriesTools)
	case ProviderGeminiOpenAI:
		base := geminiOpenAIBase.Get(f.values)
		if base == "" {
			base = DefaultGeminiOpenAIBaseURL
		}
		return f.compatible(ctx, ref, base, carriesTools)
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

func (f *Factory) compatible(ctx context.Context, ref llm.ModelRef, base string, carriesTools bool) (gollem.LLMClient, error) {
	key, err := f.key(ctx, ref.Provider)
	if err != nil {
		return nil, err
	}

	options := []openai.Option{
		openai.WithModel(ref.Model),
		openai.WithReasoningEffort(string(f.effort(ctx, ref, carriesTools))),
		openai.WithVerbosity(""),
	}
	if base != "" {
		options = append(options, openai.WithBaseURL(base))
	}
	return wrap(openai.New(ctx, key, options...))
}

func wrap[T gollem.LLMClient](client T, err error) (gollem.LLMClient, error) {
	if err != nil {
		return nil, errors.Wrap(err, errors.External, "build the provider client")
	}
	return client, nil
}

func (f *Factory) effort(ctx context.Context, ref llm.ModelRef, carriesTools bool) llm.ReasoningEffort {
	if f.models == nil {
		return ""
	}

	info, err := f.models.Lookup(ctx, ref)
	if err != nil {
		return ""
	}
	if carriesTools && info.Reasoning {
		return llm.EffortNone
	}
	if !info.ReasoningEffort.Valid() {
		return ""
	}
	return info.ReasoningEffort
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
