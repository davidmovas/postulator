package tuning_test

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/davidmovas/postulator/internal/adapters/llm/tuning"
	port "github.com/davidmovas/postulator/internal/application/llm"
	"github.com/davidmovas/postulator/internal/domain/llm"
	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/kernel/settings"
)

type recorder struct {
	mu       sync.Mutex
	requests []port.Request
}

func (r *recorder) Complete(_ context.Context, req port.Request) (port.Response, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.requests = append(r.requests, req)
	return port.Response{Text: "ok", FinishReason: port.FinishStop}, nil
}

func (r *recorder) Stream(_ context.Context, req port.Request) (<-chan port.Delta, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.requests = append(r.requests, req)
	out := make(chan port.Delta, 1)
	out <- port.Delta{Done: true}
	close(out)
	return out, nil
}

func (r *recorder) last(t *testing.T) port.Request {
	t.Helper()
	r.mu.Lock()
	defer r.mu.Unlock()
	if len(r.requests) == 0 {
		t.Fatal("the decorated client was never called")
	}
	return r.requests[len(r.requests)-1]
}

func stored(t *testing.T, pairs map[string]string) *settings.Values {
	t.Helper()

	values := settings.Default().NewValues()
	raw := make(map[string]json.RawMessage, len(pairs))
	for key, value := range pairs {
		encoded, err := json.Marshal(value)
		if err != nil {
			t.Fatalf("encode %s: %v", key, err)
		}
		raw[key] = encoded
	}
	if _, err := settings.Default().Apply(values, raw); err != nil {
		t.Fatalf("Apply: %v", err)
	}
	return values
}

func request(role llm.Role) port.Request {
	return port.Request{
		Ref:      llm.ModelRef{Provider: "openai", Model: "gpt-5.6-terra"},
		Messages: []port.Message{{Role: port.RoleUser, Text: "write"}},
		Meta:     port.CallMeta{Step: "generate_body", Role: role},
	}
}

func TestTheDefaultsTuneEveryRoleThatCallsAModel(t *testing.T) {
	t.Parallel()

	cases := []struct {
		role   llm.Role
		effort llm.ReasoningEffort
		tier   llm.ServiceTier
	}{
		{role: llm.RoleWriter, effort: llm.EffortMedium, tier: llm.TierFlex},
		{role: llm.RoleEditor, effort: llm.EffortLow, tier: llm.TierDefault},
		{role: llm.RoleLinker, effort: llm.EffortLow, tier: llm.TierDefault},
		{role: llm.RoleJudge, effort: llm.EffortLow, tier: llm.TierDefault},
		{role: llm.RoleTitler, effort: llm.EffortNone, tier: llm.TierDefault},
		{role: llm.RoleChat},
		{role: llm.RoleImage},
		{role: ""},
	}

	policy := tuning.NewPolicy(settings.Default().NewValues())
	for _, tc := range cases {
		t.Run(string(tc.role), func(t *testing.T) {
			t.Parallel()

			if got := policy.Effort(tc.role); got != tc.effort {
				t.Errorf("Effort(%q) = %q, want %q", tc.role, got, tc.effort)
			}
			if got := policy.Tier(tc.role); got != tc.tier {
				t.Errorf("Tier(%q) = %q, want %q", tc.role, got, tc.tier)
			}
		})
	}
	if got := policy.FlexPatience(); got != 8*time.Minute {
		t.Errorf("FlexPatience = %s, want eight minutes", got)
	}
}

func TestThePolicyReadsWhatTheSettingsHoldNow(t *testing.T) {
	t.Parallel()

	values := settings.Default().NewValues()
	policy := tuning.NewPolicy(values)

	raw := map[string]json.RawMessage{
		"llm.effort.writer": json.RawMessage(`"xhigh"`),
		"llm.tier.writer":   json.RawMessage(`"standard"`),
		"llm.tier.judge":    json.RawMessage(`"flex"`),
		"llm.effort.titler": json.RawMessage(`"low"`),
		"llm.flexPatience":  json.RawMessage(`"90s"`),
	}
	if _, err := settings.Default().Apply(values, raw); err != nil {
		t.Fatalf("Apply: %v", err)
	}

	if got := policy.Effort(llm.RoleWriter); got != llm.EffortXHigh {
		t.Errorf("the writer's effort = %q, want xhigh", got)
	}
	if got := policy.Tier(llm.RoleWriter); got != llm.TierDefault {
		t.Errorf("the writer's tier = %q, want default", got)
	}
	if got := policy.Tier(llm.RoleJudge); got != llm.TierFlex {
		t.Errorf("the judge's tier = %q, want flex", got)
	}
	if got := policy.Effort(llm.RoleTitler); got != llm.EffortLow {
		t.Errorf("the titler's effort = %q, want low", got)
	}
	if got := policy.FlexPatience(); got != 90*time.Second {
		t.Errorf("FlexPatience = %s, want ninety seconds", got)
	}
}

func TestAPolicyWithoutValuesAnswersTheDefaults(t *testing.T) {
	t.Parallel()

	policy := tuning.NewPolicy(nil)
	if got := policy.Effort(llm.RoleWriter); got != llm.EffortMedium {
		t.Errorf("Effort = %q, want medium", got)
	}
	if got := policy.Tier(llm.RoleWriter); got != llm.TierFlex {
		t.Errorf("Tier = %q, want flex", got)
	}
}

func TestTheSettingsRefuseWhatTheProviderRefuses(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		key   string
		value string
		ok    bool
	}{
		{name: "every effort the provider takes", key: "llm.effort.writer", value: `"xhigh"`, ok: true},
		{name: "no reasoning", key: "llm.effort.judge", value: `"none"`, ok: true},
		{name: "minimal is refused by the provider", key: "llm.effort.editor", value: `"minimal"`},
		{name: "an unknown effort", key: "llm.effort.linker", value: `"extreme"`},
		{name: "a chat effort is not a setting", key: "llm.effort.chat", value: `"low"`},
		{name: "flex", key: "llm.tier.editor", value: `"flex"`, ok: true},
		{name: "standard", key: "llm.tier.writer", value: `"standard"`, ok: true},
		{name: "the wire name of the standard tier", key: "llm.tier.writer", value: `"default"`},
		{name: "a chat tier is not a setting", key: "llm.tier.chat", value: `"flex"`},
		{name: "the shortest patience", key: "llm.flexPatience", value: `"30s"`, ok: true},
		{name: "the longest patience stays under the writer's step", key: "llm.flexPatience", value: `"14m"`, ok: true},
		{name: "a patience too short to answer", key: "llm.flexPatience", value: `"10s"`},
		{name: "a patience past the writer's step", key: "llm.flexPatience", value: `"15m"`},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := settings.Default().Validate(tc.key, json.RawMessage(tc.value))
			if tc.ok && err != nil {
				t.Fatalf("Validate(%s, %s) = %v, want it accepted", tc.key, tc.value, err)
			}
			if !tc.ok && err == nil {
				t.Fatalf("Validate(%s, %s) accepted it", tc.key, tc.value)
			}
		})
	}
}

func TestTheSettingsDescribeTheirChoices(t *testing.T) {
	t.Parallel()

	schema, err := settings.Default().Schema()
	if err != nil {
		t.Fatalf("Schema: %v", err)
	}
	described := make(map[string]settings.Descriptor, len(schema))
	for _, descriptor := range schema {
		described[descriptor.Key] = descriptor
	}

	cases := []struct {
		key      string
		kind     string
		fallback string
		enum     []string
	}{
		{key: "llm.effort.writer", kind: "enum", fallback: `"medium"`, enum: []string{"none", "low", "medium", "high", "xhigh"}},
		{key: "llm.effort.editor", kind: "enum", fallback: `"low"`, enum: []string{"none", "low", "medium", "high", "xhigh"}},
		{key: "llm.effort.linker", kind: "enum", fallback: `"low"`, enum: []string{"none", "low", "medium", "high", "xhigh"}},
		{key: "llm.effort.judge", kind: "enum", fallback: `"low"`, enum: []string{"none", "low", "medium", "high", "xhigh"}},
		{key: "llm.effort.titler", kind: "enum", fallback: `"none"`, enum: []string{"none", "low", "medium", "high", "xhigh"}},
		{key: "llm.tier.writer", kind: "enum", fallback: `"flex"`, enum: []string{"standard", "flex"}},
		{key: "llm.tier.editor", kind: "enum", fallback: `"standard"`, enum: []string{"standard", "flex"}},
		{key: "llm.tier.linker", kind: "enum", fallback: `"standard"`, enum: []string{"standard", "flex"}},
		{key: "llm.tier.judge", kind: "enum", fallback: `"standard"`, enum: []string{"standard", "flex"}},
		{key: "llm.tier.titler", kind: "enum", fallback: `"standard"`, enum: []string{"standard", "flex"}},
		{key: "llm.flexPatience", kind: "duration", fallback: `"8m0s"`},
	}

	for _, tc := range cases {
		t.Run(tc.key, func(t *testing.T) {
			t.Parallel()

			descriptor, declared := described[tc.key]
			if !declared {
				t.Fatalf("%s is not declared", tc.key)
			}
			if descriptor.Type != tc.kind || string(descriptor.Default) != tc.fallback || descriptor.Group != "llm" {
				t.Fatalf("%s is described as %+v", tc.key, descriptor)
			}
			if len(descriptor.Enum) != len(tc.enum) {
				t.Fatalf("%s offers %v, want %v", tc.key, descriptor.Enum, tc.enum)
			}
			for i := range tc.enum {
				if descriptor.Enum[i] != tc.enum[i] {
					t.Fatalf("%s offers %v, want %v", tc.key, descriptor.Enum, tc.enum)
				}
			}
		})
	}
}

func TestTheClientFillsWhatTheRequestLeavesOpen(t *testing.T) {
	t.Parallel()

	tuned := map[string]string{"llm.effort.editor": "high", "llm.tier.linker": "flex"}

	cases := []struct {
		name   string
		req    port.Request
		effort llm.ReasoningEffort
		tier   llm.ServiceTier
	}{
		{name: "the writer", req: request(llm.RoleWriter), effort: llm.EffortMedium, tier: llm.TierFlex},
		{name: "the editor as the settings hold it", req: request(llm.RoleEditor), effort: llm.EffortHigh, tier: llm.TierDefault},
		{name: "the linker on flex", req: request(llm.RoleLinker), effort: llm.EffortLow, tier: llm.TierFlex},
		{name: "the judge", req: request(llm.RoleJudge), effort: llm.EffortLow, tier: llm.TierDefault},
		{name: "the titler thinks not at all", req: request(llm.RoleTitler), effort: llm.EffortNone, tier: llm.TierDefault},
		{
			name: "a request that names its own effort and tier keeps them",
			req: func() port.Request {
				req := request(llm.RoleWriter)
				req.Effort, req.Tier = llm.EffortHigh, llm.TierDefault
				return req
			}(),
			effort: llm.EffortHigh, tier: llm.TierDefault,
		},
		{
			name: "a request that names only its effort is given the role's tier",
			req: func() port.Request {
				req := request(llm.RoleWriter)
				req.Effort = llm.EffortLow
				return req
			}(),
			effort: llm.EffortLow, tier: llm.TierFlex,
		},
		{
			name: "chat sets its own effort and no setting overrides it",
			req: func() port.Request {
				req := request(llm.RoleChat)
				req.Effort = llm.EffortNone
				return req
			}(),
			effort: llm.EffortNone,
		},
		{name: "a call that names no role is left as it came", req: request("")},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			for _, streamed := range []bool{false, true} {
				next := &recorder{}
				client := tuning.New(next, tuning.NewPolicy(stored(t, tuned)))

				if streamed {
					deltas, err := client.Stream(t.Context(), tc.req)
					if err != nil {
						t.Fatalf("Stream: %v", err)
					}
					if last := drained(deltas); !last.Done {
						t.Fatalf("the stream ended on %+v, want the provider's last delta", last)
					}
				} else if _, err := client.Complete(t.Context(), tc.req); err != nil {
					t.Fatalf("Complete: %v", err)
				}

				sent := next.last(t)
				if sent.Effort != tc.effort || sent.Tier != tc.tier {
					t.Fatalf("streamed %t: sent effort %q on tier %q, want %q on %q",
						streamed, sent.Effort, sent.Tier, tc.effort, tc.tier)
				}
				if sent.Meta != tc.req.Meta || sent.Ref != tc.req.Ref || len(sent.Messages) != len(tc.req.Messages) {
					t.Fatalf("streamed %t: the request changed beyond its effort and tier: %+v", streamed, sent)
				}
			}
		})
	}
}

func TestTheClientPassesTheAnswerAndTheFailureThrough(t *testing.T) {
	t.Parallel()

	refused := errors.New(errors.RateLimited, "slow down")
	client := tuning.New(failing{err: refused}, tuning.NewPolicy(nil))

	if _, err := client.Complete(t.Context(), request(llm.RoleWriter)); !errors.IsCode(err, errors.RateLimited) {
		t.Fatalf("Complete = %v, want the provider's refusal", err)
	}
	if _, err := client.Stream(t.Context(), request(llm.RoleWriter)); !errors.IsCode(err, errors.RateLimited) {
		t.Fatalf("Stream = %v, want the provider's refusal", err)
	}

	answered, err := tuning.New(&recorder{}, tuning.NewPolicy(nil)).Complete(t.Context(), request(llm.RoleEditor))
	if err != nil || answered.Text != "ok" {
		t.Fatalf("Complete = %+v, %v; want the provider's answer", answered, err)
	}
}

func drained(deltas <-chan port.Delta) port.Delta {
	var last port.Delta
	for delta := range deltas {
		last = delta
	}
	return last
}

type failing struct {
	err error
}

func (f failing) Complete(context.Context, port.Request) (port.Response, error) {
	return port.Response{}, f.err
}

func (f failing) Stream(context.Context, port.Request) (<-chan port.Delta, error) {
	return nil, f.err
}
