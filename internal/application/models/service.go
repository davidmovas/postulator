package models

import (
	"context"
	"slices"
	"strings"
	"time"

	"github.com/davidmovas/postulator/internal/application"
	"github.com/davidmovas/postulator/internal/application/events"
	port "github.com/davidmovas/postulator/internal/application/llm"
	"github.com/davidmovas/postulator/internal/domain/llm"
	"github.com/davidmovas/postulator/internal/kernel/clock"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

const probeTokens = 1

type catalogReader interface {
	List(ctx context.Context) ([]llm.ModelInfo, error)
	Lookup(ctx context.Context, ref llm.ModelRef) (llm.ModelInfo, error)
}

type catalogWriter interface {
	Upsert(ctx context.Context, override llm.ModelOverride) error
}

type profileStore interface {
	Global(ctx context.Context) (map[llm.Role]llm.ModelRef, error)
	Set(ctx context.Context, role llm.Role, ref llm.ModelRef) error
	Resolve(ctx context.Context, siteID string, role llm.Role, templateProfiles map[llm.Role]llm.ModelRef) (llm.ModelRef, error)
}

type spendReader interface {
	SumByRun(ctx context.Context, runID string) (llm.Spend, error)
	SumByConversation(ctx context.Context, conversationID string) (llm.Spend, error)
}

type prober interface {
	Complete(ctx context.Context, req port.Request) (port.Response, error)
}

type secretStore interface {
	Put(ctx context.Context, ref, value string) error
	Has(ctx context.Context, ref string) (bool, error)
	Delete(ctx context.Context, ref string) error
}

type Service struct {
	catalog   catalogReader
	overrides catalogWriter
	profiles  profileStore
	spend     spendReader
	secrets   secretStore
	llm       prober
	publisher application.Publisher
	clock     clock.Clock
}

func New(catalog catalogReader, overrides catalogWriter, profiles profileStore, spend spendReader, secrets secretStore, client prober, publisher application.Publisher, clk clock.Clock) *Service {
	return &Service{
		catalog: catalog, overrides: overrides, profiles: profiles, spend: spend,
		secrets: secrets, llm: client, publisher: publisher, clock: clk,
	}
}

func (s *Service) settingsChanged() error {
	return s.publisher.Publish(events.SettingsChanged, events.SettingsChangedPayload{})
}

func (s *Service) SetProviderKey(ctx context.Context, req SetProviderKeyRequest) (SetProviderKeyResponse, error) {
	provider := strings.TrimSpace(req.Provider)
	key := strings.TrimSpace(req.APIKey)
	switch {
	case provider == "":
		return SetProviderKeyResponse{}, errors.New(errors.Invalid, "a provider key needs a provider").
			WithDetail("field", "provider")
	case key == "":
		return SetProviderKeyResponse{}, errors.New(errors.Invalid, "a provider key must not be empty").
			WithDetail("field", "apiKey")
	}

	if err := s.requireProvider(ctx, provider); err != nil {
		return SetProviderKeyResponse{}, err
	}

	if putErr := s.secrets.Put(ctx, llm.SecretRef(provider), key); putErr != nil {
		return SetProviderKeyResponse{}, putErr
	}
	if publishErr := s.settingsChanged(); publishErr != nil {
		return SetProviderKeyResponse{}, publishErr
	}
	return SetProviderKeyResponse{Provider: provider}, nil
}

func (s *Service) ProviderKeys(ctx context.Context, _ ProviderKeysRequest) (ProviderKeysResponse, error) {
	known, err := s.catalog.List(ctx)
	if err != nil {
		return ProviderKeysResponse{}, err
	}

	named := make([]string, 0, len(known))
	for _, info := range known {
		if !slices.Contains(named, info.Ref.Provider) {
			named = append(named, info.Ref.Provider)
		}
	}
	slices.Sort(named)

	out := make([]ProviderKey, 0, len(named))
	for _, provider := range named {
		configured, hasErr := s.secrets.Has(ctx, llm.SecretRef(provider))
		if hasErr != nil {
			return ProviderKeysResponse{}, hasErr
		}
		out = append(out, ProviderKey{Provider: provider, Configured: configured})
	}
	return ProviderKeysResponse{Providers: out}, nil
}

func (s *Service) DeleteProviderKey(ctx context.Context, req DeleteProviderKeyRequest) (DeleteProviderKeyResponse, error) {
	provider := strings.TrimSpace(req.Provider)
	if provider == "" {
		return DeleteProviderKeyResponse{}, errors.New(errors.Invalid, "a provider key needs a provider").
			WithDetail("field", "provider")
	}
	if err := s.requireProvider(ctx, provider); err != nil {
		return DeleteProviderKeyResponse{}, err
	}

	if err := s.secrets.Delete(ctx, llm.SecretRef(provider)); err != nil && !errors.IsCode(err, errors.NotFound) {
		return DeleteProviderKeyResponse{}, err
	}
	if publishErr := s.settingsChanged(); publishErr != nil {
		return DeleteProviderKeyResponse{}, publishErr
	}
	return DeleteProviderKeyResponse{Provider: provider}, nil
}

func (s *Service) requireProvider(ctx context.Context, provider string) error {
	known, err := s.catalog.List(ctx)
	if err != nil {
		return err
	}
	if !slices.ContainsFunc(known, func(info llm.ModelInfo) bool { return info.Ref.Provider == provider }) {
		return errors.New(errors.NotFound, "the catalog holds no model of this provider").
			WithDetail("provider", provider)
	}
	return nil
}

func (s *Service) now() time.Time {
	return s.clock.Now().UTC().Truncate(time.Second)
}

func (s *Service) ListModels(ctx context.Context, _ ListModelsRequest) (ListModelsResponse, error) {
	known, err := s.catalog.List(ctx)
	if err != nil {
		return ListModelsResponse{}, err
	}

	out := make([]Model, 0, len(known))
	for _, info := range known {
		out = append(out, modelView(info))
	}
	return ListModelsResponse{Models: out}, nil
}

func (s *Service) UpsertModel(ctx context.Context, req UpsertModelRequest) (UpsertModelResponse, error) {
	info := llm.ModelInfo{
		Ref:                llm.ModelRef{Provider: strings.TrimSpace(req.Provider), Model: strings.TrimSpace(req.Model)},
		ContextTokens:      req.ContextTokens,
		MaxOutputTokens:    req.MaxOutputTokens,
		InputUSDPerM:       req.InputUSDPerM,
		OutputUSDPerM:      req.OutputUSDPerM,
		RPM:                req.RPM,
		TPM:                req.TPM,
		SupportsStructured: req.SupportsStructured,
		SupportsImages:     req.SupportsImages,
		Reasoning:          req.Reasoning,
	}
	if err := info.Validate(); err != nil {
		return UpsertModelResponse{}, err
	}

	now := s.now()
	if err := s.overrides.Upsert(ctx, llm.ModelOverride{Info: info, Enabled: true, CreatedAt: now, UpdatedAt: now}); err != nil {
		return UpsertModelResponse{}, err
	}
	return UpsertModelResponse{Model: modelView(info)}, nil
}

func (s *Service) DisableModel(ctx context.Context, req DisableModelRequest) (DisableModelResponse, error) {
	ref := llm.ModelRef{Provider: strings.TrimSpace(req.Provider), Model: strings.TrimSpace(req.Model)}
	if !ref.Valid() {
		return DisableModelResponse{}, errors.New(errors.Invalid, "a model must be named by its provider and model")
	}

	info, err := s.catalog.Lookup(ctx, ref)
	if err != nil {
		return DisableModelResponse{}, err
	}

	now := s.now()
	if upsertErr := s.overrides.Upsert(ctx, llm.ModelOverride{Info: info, Enabled: false, CreatedAt: now, UpdatedAt: now}); upsertErr != nil {
		return DisableModelResponse{}, upsertErr
	}
	return DisableModelResponse{}, nil
}

func (s *Service) SetProfile(ctx context.Context, req SetProfileRequest) (SetProfileResponse, error) {
	role := llm.Role(strings.TrimSpace(req.Role))
	ref := llm.ModelRef{Provider: strings.TrimSpace(req.Provider), Model: strings.TrimSpace(req.Model)}

	if _, err := s.catalog.Lookup(ctx, ref); err != nil {
		return SetProfileResponse{}, err
	}
	if err := s.profiles.Set(ctx, role, ref); err != nil {
		return SetProfileResponse{}, err
	}

	view := refView(ref)
	return SetProfileResponse{Profile: Profile{Role: string(role), Global: view, Effective: view}}, nil
}

func (s *Service) GetProfiles(ctx context.Context, req GetProfilesRequest) (GetProfilesResponse, error) {
	global, err := s.profiles.Global(ctx)
	if err != nil {
		return GetProfilesResponse{}, err
	}

	roles := llm.Roles()
	out := make([]Profile, 0, len(roles))
	for _, role := range roles {
		profile := Profile{Role: string(role)}
		if ref, ok := global[role]; ok {
			profile.Global = refView(ref)
		}

		effective, resolveErr := s.profiles.Resolve(ctx, req.SiteID, role, nil)
		switch {
		case resolveErr == nil:
			profile.Effective = refView(effective)
		case errors.IsCode(resolveErr, errors.NotFound):
		default:
			return GetProfilesResponse{}, resolveErr
		}
		out = append(out, profile)
	}
	return GetProfilesResponse{Profiles: out}, nil
}

func (s *Service) TestProvider(ctx context.Context, req TestProviderRequest) (TestProviderResponse, error) {
	ref := llm.ModelRef{Provider: strings.TrimSpace(req.Provider), Model: strings.TrimSpace(req.Model)}
	if _, err := s.catalog.Lookup(ctx, ref); err != nil {
		return TestProviderResponse{}, err
	}

	started := time.Now()
	resp, err := s.llm.Complete(ctx, port.Request{
		Ref:       ref,
		Messages:  []port.Message{{Role: port.RoleUser, Text: "ping"}},
		MaxTokens: probeTokens,
		Meta:      port.CallMeta{Step: "test_provider"},
	})
	if err != nil {
		return TestProviderResponse{}, err
	}

	return TestProviderResponse{
		Model:     ModelRef{Provider: ref.Provider, Model: ref.Model},
		Usage:     usageView(resp.Usage),
		LatencyMs: time.Since(started).Milliseconds(),
	}, nil
}

func (s *Service) UsageSummary(ctx context.Context, req UsageSummaryRequest) (UsageSummaryResponse, error) {
	runID := strings.TrimSpace(req.RunID)
	conversationID := strings.TrimSpace(req.ConversationID)
	if (runID == "") == (conversationID == "") {
		return UsageSummaryResponse{}, errors.New(errors.Invalid, "name exactly one of the run and the conversation")
	}

	var (
		spend llm.Spend
		err   error
	)
	if runID != "" {
		spend, err = s.spend.SumByRun(ctx, runID)
	} else {
		spend, err = s.spend.SumByConversation(ctx, conversationID)
	}
	if err != nil {
		return UsageSummaryResponse{}, err
	}
	return UsageSummaryResponse{Usage: usageView(spend.Usage), USD: spend.USD, Calls: spend.Calls}, nil
}
