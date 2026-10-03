package app_test

import (
	"context"
	"path/filepath"
	"sync"
	"testing"

	"github.com/davidmovas/postulator/internal/adapters/llm/fake"
	"github.com/davidmovas/postulator/internal/app"
	llmport "github.com/davidmovas/postulator/internal/application/llm"
	domainllm "github.com/davidmovas/postulator/internal/domain/llm"
)

type recordingProvider struct {
	inner *fake.Client
	mu    sync.Mutex
	asked []llmport.Request
}

func (p *recordingProvider) note(req llmport.Request) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.asked = append(p.asked, req)
}

func (p *recordingProvider) last() llmport.Request {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.asked[len(p.asked)-1]
}

func (p *recordingProvider) Complete(ctx context.Context, req llmport.Request) (llmport.Response, error) {
	p.note(req)
	return p.inner.Complete(ctx, req)
}

func (p *recordingProvider) Stream(ctx context.Context, req llmport.Request) (<-chan llmport.Delta, error) {
	p.note(req)
	return p.inner.Stream(ctx, req)
}

func TestEveryCallIsTunedForTheRoleItIsMadeFor(t *testing.T) {
	t.Parallel()

	provider := &recordingProvider{inner: fake.New()}
	home := t.TempDir()
	core := openCore(t, app.Config{DatabasePath: filepath.Join(home, "postulator.db"), KeyDir: home, Provider: provider})
	t.Cleanup(func() {
		if closeErr := core.Close(); closeErr != nil {
			t.Errorf("Close: %v", closeErr)
		}
	})

	cases := []struct {
		name       string
		role       domainllm.Role
		effort     domainllm.ReasoningEffort
		tier       domainllm.ServiceTier
		wantEffort domainllm.ReasoningEffort
		wantTier   domainllm.ServiceTier
	}{
		{name: "the writer thinks and waits for flex", role: domainllm.RoleWriter, wantEffort: domainllm.EffortMedium, wantTier: domainllm.TierFlex},
		{name: "the judge thinks a little at the standard price", role: domainllm.RoleJudge, wantEffort: domainllm.EffortLow, wantTier: domainllm.TierDefault},
		{
			name: "what the caller chose is kept", role: domainllm.RoleChat,
			effort: domainllm.EffortNone, tier: domainllm.TierDefault,
			wantEffort: domainllm.EffortNone, wantTier: domainllm.TierDefault,
		},
	}

	for _, tc := range cases {
		if _, err := core.LLM.Complete(t.Context(), llmport.Request{
			Ref:      domainllm.ModelRef{Provider: "openai", Model: "gpt-5.6-terra"},
			Messages: []llmport.Message{{Role: llmport.RoleUser, Text: "write"}},
			Meta:     llmport.CallMeta{Step: "tuning_probe", Role: tc.role},
			Effort:   tc.effort,
			Tier:     tc.tier,
		}); err != nil {
			t.Fatalf("%s: Complete: %v", tc.name, err)
		}

		asked := provider.last()
		if asked.Effort != tc.wantEffort || asked.Tier != tc.wantTier {
			t.Fatalf("%s: the provider was asked with %q on %q, want %q on %q",
				tc.name, asked.Effort, asked.Tier, tc.wantEffort, tc.wantTier)
		}
	}
}
