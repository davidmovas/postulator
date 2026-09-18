package models_test

import (
	"context"
	"slices"
	"testing"
	"time"

	port "github.com/davidmovas/postulator/internal/application/llm"
	"github.com/davidmovas/postulator/internal/application/models"
	"github.com/davidmovas/postulator/internal/domain/llm"
	"github.com/davidmovas/postulator/internal/kernel/clock"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

type catalog struct {
	models    map[string]llm.ModelInfo
	overrides []llm.ModelOverride
}

func newCatalog() *catalog {
	ref := llm.ModelRef{Provider: "openai", Model: "gpt-5.6-luna"}
	return &catalog{models: map[string]llm.ModelInfo{ref.String(): {
		Ref:             ref,
		ContextTokens:   1050000,
		MaxOutputTokens: 128000,
		InputUSDPerM:    0.2,
		OutputUSDPerM:   1.2,
		RPM:             60,
		TPM:             120000,
	}}}
}

func (c *catalog) List(context.Context) ([]llm.ModelInfo, error) {
	out := make([]llm.ModelInfo, 0, len(c.models))
	for _, info := range c.models {
		out = append(out, info)
	}
	slices.SortFunc(out, func(a, b llm.ModelInfo) int { return int(a.InputUSDPerM - b.InputUSDPerM) })
	return out, nil
}

func (c *catalog) Lookup(_ context.Context, ref llm.ModelRef) (llm.ModelInfo, error) {
	info, ok := c.models[ref.String()]
	if !ok {
		return llm.ModelInfo{}, errors.New(errors.NotFound, "the model catalog does not carry this model")
	}
	return info, nil
}

func (c *catalog) Upsert(_ context.Context, override llm.ModelOverride) error {
	c.overrides = append(c.overrides, override)
	if override.Enabled {
		c.models[override.Info.Ref.String()] = override.Info
	} else {
		delete(c.models, override.Info.Ref.String())
	}
	return nil
}

type profiles struct {
	global   map[llm.Role]llm.ModelRef
	resolved map[llm.Role]llm.ModelRef
	siteID   string
}

func (p *profiles) Global(context.Context) (map[llm.Role]llm.ModelRef, error) {
	return p.global, nil
}

func (p *profiles) Set(_ context.Context, role llm.Role, ref llm.ModelRef) error {
	if !role.Valid() {
		return errors.New(errors.Invalid, "unknown role")
	}
	p.global[role] = ref
	p.resolved[role] = ref
	return nil
}

func (p *profiles) Resolve(_ context.Context, siteID string, role llm.Role, _ map[llm.Role]llm.ModelRef) (llm.ModelRef, error) {
	p.siteID = siteID
	ref, ok := p.resolved[role]
	if !ok {
		return llm.ModelRef{}, errors.New(errors.NotFound, "no model for this role")
	}
	return ref, nil
}

type spend struct {
	run          llm.Spend
	conversation llm.Spend
	err          error
}

func (s spend) SumByRun(context.Context, string) (llm.Spend, error) {
	return s.run, s.err
}

func (s spend) SumByConversation(context.Context, string) (llm.Spend, error) {
	return s.conversation, s.err
}

type prober struct {
	seen port.Request
	err  error
}

func (p *prober) Complete(_ context.Context, req port.Request) (port.Response, error) {
	p.seen = req
	if p.err != nil {
		return port.Response{}, p.err
	}
	return port.Response{Text: "pong", Usage: llm.Usage{Input: 1, Output: 1, Total: 2}}, nil
}

type harness struct {
	service  *models.Service
	catalog  *catalog
	profiles *profiles
	prober   *prober
}

func newHarness(t *testing.T, book spend) harness {
	t.Helper()

	known := newCatalog()
	people := &profiles{global: map[llm.Role]llm.ModelRef{}, resolved: map[llm.Role]llm.ModelRef{}}
	probe := &prober{}
	return harness{
		service:  models.New(known, known, people, book, probe, clock.NewFake(time.Date(2026, 9, 18, 10, 0, 0, 0, time.UTC))),
		catalog:  known,
		profiles: people,
		prober:   probe,
	}
}

func TestListModels(t *testing.T) {
	t.Parallel()

	h := newHarness(t, spend{})
	resp, err := h.service.ListModels(t.Context(), models.ListModelsRequest{})
	if err != nil {
		t.Fatalf("ListModels: %v", err)
	}
	if len(resp.Models) != 1 || resp.Models[0].Model != "gpt-5.6-luna" || resp.Models[0].ContextTokens != 1050000 {
		t.Fatalf("models = %+v, want the catalog entry", resp.Models)
	}
}

func TestUpsertModel(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		req     models.UpsertModelRequest
		wantErr bool
	}{
		{
			name: "a new model",
			req: models.UpsertModelRequest{
				Provider: " anthropic ", Model: " claude-sonnet-5 ",
				ContextTokens: 1000000, MaxOutputTokens: 128000,
				InputUSDPerM: 2, OutputUSDPerM: 10, RPM: 60, TPM: 120000,
				SupportsStructured: true,
			},
		},
		{name: "no reference", req: models.UpsertModelRequest{ContextTokens: 10, MaxOutputTokens: 5, RPM: 1, TPM: 1}, wantErr: true},
		{
			name:    "no context window",
			req:     models.UpsertModelRequest{Provider: "openai", Model: "x", MaxOutputTokens: 5, RPM: 1, TPM: 1},
			wantErr: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			h := newHarness(t, spend{})
			resp, err := h.service.UpsertModel(t.Context(), tc.req)
			if tc.wantErr {
				if !errors.IsCode(err, errors.Invalid) {
					t.Fatalf("UpsertModel error = %v, want %s", err, errors.Invalid)
				}
				return
			}
			if err != nil {
				t.Fatalf("UpsertModel: %v", err)
			}
			if resp.Model.Provider != "anthropic" || resp.Model.Model != "claude-sonnet-5" {
				t.Errorf("model = %+v, want the trimmed reference", resp.Model)
			}
			if len(h.catalog.overrides) != 1 || !h.catalog.overrides[0].Enabled {
				t.Errorf("overrides = %+v, want one enabled row", h.catalog.overrides)
			}
		})
	}
}

func TestDisableModel(t *testing.T) {
	t.Parallel()

	h := newHarness(t, spend{})
	if _, err := h.service.DisableModel(t.Context(), models.DisableModelRequest{Provider: "openai", Model: "gpt-5.6-luna"}); err != nil {
		t.Fatalf("DisableModel: %v", err)
	}
	if len(h.catalog.overrides) != 1 || h.catalog.overrides[0].Enabled {
		t.Fatalf("overrides = %+v, want one disabled row", h.catalog.overrides)
	}

	if _, err := h.service.DisableModel(t.Context(), models.DisableModelRequest{Provider: "openai"}); !errors.IsCode(err, errors.Invalid) {
		t.Errorf("DisableModel error = %v, want %s", err, errors.Invalid)
	}
	if _, err := h.service.DisableModel(t.Context(), models.DisableModelRequest{Provider: "openai", Model: "ghost"}); !errors.IsCode(err, errors.NotFound) {
		t.Errorf("DisableModel error = %v, want %s", err, errors.NotFound)
	}
}

func TestProfiles(t *testing.T) {
	t.Parallel()

	h := newHarness(t, spend{})
	if _, err := h.service.SetProfile(t.Context(), models.SetProfileRequest{
		Role: string(llm.RoleWriter), Provider: "openai", Model: "gpt-5.6-luna",
	}); err != nil {
		t.Fatalf("SetProfile: %v", err)
	}

	if _, err := h.service.SetProfile(t.Context(), models.SetProfileRequest{
		Role: string(llm.RoleWriter), Provider: "openai", Model: "ghost",
	}); !errors.IsCode(err, errors.NotFound) {
		t.Errorf("SetProfile error = %v, want %s", err, errors.NotFound)
	}
	if _, err := h.service.SetProfile(t.Context(), models.SetProfileRequest{
		Role: "painter", Provider: "openai", Model: "gpt-5.6-luna",
	}); !errors.IsCode(err, errors.Invalid) {
		t.Errorf("SetProfile error = %v, want %s", err, errors.Invalid)
	}

	resp, err := h.service.GetProfiles(t.Context(), models.GetProfilesRequest{SiteID: "site-1"})
	if err != nil {
		t.Fatalf("GetProfiles: %v", err)
	}
	if len(resp.Profiles) != len(llm.Roles()) {
		t.Fatalf("profiles = %d, want one per role", len(resp.Profiles))
	}
	if h.profiles.siteID != "site-1" {
		t.Errorf("site = %q, want the requested site", h.profiles.siteID)
	}

	for _, profile := range resp.Profiles {
		if profile.Role == string(llm.RoleWriter) {
			if profile.Global == nil || profile.Effective == nil || profile.Effective.Model != "gpt-5.6-luna" {
				t.Errorf("writer profile = %+v, want the global model", profile)
			}
			continue
		}
		if profile.Global != nil || profile.Effective != nil {
			t.Errorf("%s profile = %+v, want nothing set", profile.Role, profile)
		}
	}
}

func TestTestProvider(t *testing.T) {
	t.Parallel()

	h := newHarness(t, spend{})
	resp, err := h.service.TestProvider(t.Context(), models.TestProviderRequest{Provider: "openai", Model: "gpt-5.6-luna"})
	if err != nil {
		t.Fatalf("TestProvider: %v", err)
	}
	if resp.Model.Model != "gpt-5.6-luna" || resp.Usage.Total != 2 {
		t.Errorf("response = %+v, want the probe usage", resp)
	}
	if h.prober.seen.MaxTokens != 1 || h.prober.seen.Meta.Step != "test_provider" {
		t.Errorf("probe request = %+v, want a one token call", h.prober.seen)
	}

	if _, err = h.service.TestProvider(t.Context(), models.TestProviderRequest{Provider: "openai", Model: "ghost"}); !errors.IsCode(err, errors.NotFound) {
		t.Errorf("TestProvider error = %v, want %s", err, errors.NotFound)
	}

	h.prober.err = errors.New(errors.Unauthorized, "no key")
	if _, err = h.service.TestProvider(t.Context(), models.TestProviderRequest{Provider: "openai", Model: "gpt-5.6-luna"}); !errors.IsCode(err, errors.Unauthorized) {
		t.Errorf("TestProvider error = %v, want %s", err, errors.Unauthorized)
	}
}

func TestUsageSummary(t *testing.T) {
	t.Parallel()

	book := spend{
		run:          llm.Spend{Usage: llm.Usage{Input: 100, Output: 50, Total: 150}, USD: 1.5, Calls: 3},
		conversation: llm.Spend{Usage: llm.Usage{Input: 10, Output: 5, Total: 15}, USD: 0.1, Calls: 1},
	}

	cases := []struct {
		name      string
		req       models.UsageSummaryRequest
		wantCalls int
		wantErr   bool
	}{
		{name: "by run", req: models.UsageSummaryRequest{RunID: "run-1"}, wantCalls: 3},
		{name: "by conversation", req: models.UsageSummaryRequest{ConversationID: "chat-1"}, wantCalls: 1},
		{name: "neither", req: models.UsageSummaryRequest{}, wantErr: true},
		{name: "both", req: models.UsageSummaryRequest{RunID: "run-1", ConversationID: "chat-1"}, wantErr: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			h := newHarness(t, book)
			resp, err := h.service.UsageSummary(t.Context(), tc.req)
			if tc.wantErr {
				if !errors.IsCode(err, errors.Invalid) {
					t.Fatalf("UsageSummary error = %v, want %s", err, errors.Invalid)
				}
				return
			}
			if err != nil {
				t.Fatalf("UsageSummary: %v", err)
			}
			if resp.Calls != tc.wantCalls {
				t.Errorf("calls = %d, want %d", resp.Calls, tc.wantCalls)
			}
		})
	}
}

func TestUsageSummaryPropagatesTheStoreError(t *testing.T) {
	t.Parallel()

	h := newHarness(t, spend{err: errors.New(errors.Internal, "disk")})
	if _, err := h.service.UsageSummary(t.Context(), models.UsageSummaryRequest{RunID: "run-1"}); !errors.IsCode(err, errors.Internal) {
		t.Fatalf("UsageSummary error = %v, want %s", err, errors.Internal)
	}
}
