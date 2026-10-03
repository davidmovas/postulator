package llm_test

import (
	"encoding/json"
	"math"
	"testing"

	"github.com/davidmovas/postulator/internal/domain/llm"
)

func TestRoles(t *testing.T) {
	t.Parallel()

	want := []llm.Role{llm.RoleWriter, llm.RoleEditor, llm.RoleLinker, llm.RoleJudge, llm.RoleChat, llm.RoleImage, llm.RoleTitler}
	got := llm.Roles()
	if len(got) != len(want) {
		t.Fatalf("Roles() = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("Roles()[%d] = %q, want %q", i, got[i], want[i])
		}
		if !got[i].Valid() {
			t.Errorf("%q must be valid", got[i])
		}
	}
	if llm.Role("painter").Valid() {
		t.Error("an unknown role must not be valid")
	}
}

func TestModelRef(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name      string
		ref       llm.ModelRef
		valid     bool
		supported bool
		text      string
	}{
		{name: "complete", ref: llm.ModelRef{Provider: "openai", Model: "gpt"}, valid: true, supported: true, text: "openai:gpt"},
		{name: "no provider", ref: llm.ModelRef{Model: "gpt"}, valid: false, text: ":gpt"},
		{name: "no model", ref: llm.ModelRef{Provider: "openai"}, valid: false, text: "openai:"},
		{name: "a provider that is no longer supported", ref: llm.ModelRef{Provider: "retired", Model: "old"}, valid: true, text: "retired:old"},
		{name: "the provider is matched exactly", ref: llm.ModelRef{Provider: "OpenAI", Model: "gpt"}, valid: true, text: "OpenAI:gpt"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := tc.ref.Valid(); got != tc.valid {
				t.Errorf("Valid() = %v, want %v", got, tc.valid)
			}
			if got := tc.ref.Supported(); got != tc.supported {
				t.Errorf("Supported() = %v, want %v", got, tc.supported)
			}
			if got := tc.ref.String(); got != tc.text {
				t.Errorf("String() = %q, want %q", got, tc.text)
			}
		})
	}
}

func TestModelRefJSONIsCamelCase(t *testing.T) {
	t.Parallel()

	encoded, err := json.Marshal(llm.ModelRef{Provider: "openai", Model: "gpt"})
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if string(encoded) != `{"provider":"openai","model":"gpt"}` {
		t.Fatalf("Marshal = %s", encoded)
	}
}

func TestCost(t *testing.T) {
	t.Parallel()

	priced := llm.ModelInfo{
		Ref: llm.ModelRef{Provider: "openai", Model: "gpt"}, InputUSDPerM: 3, OutputUSDPerM: 15,
		CachedInputUSDPerM: 0.3,
	}
	unpriced := llm.ModelInfo{Ref: llm.ModelRef{Provider: "openai", Model: "gpt"}, InputUSDPerM: 3, OutputUSDPerM: 15}
	flexible := llm.ModelInfo{
		Ref: llm.ModelRef{Provider: "openai", Model: "gpt"}, InputUSDPerM: 3, CachedInputUSDPerM: 0.3, OutputUSDPerM: 15,
		FlexInputUSDPerM: 1.5, FlexCachedInputUSDPerM: 0.15, FlexOutputUSDPerM: 7.5,
	}
	flexibleUncached := llm.ModelInfo{
		Ref: llm.ModelRef{Provider: "openai", Model: "gpt"}, InputUSDPerM: 3, CachedInputUSDPerM: 0.3, OutputUSDPerM: 15,
		FlexInputUSDPerM: 1.5, FlexOutputUSDPerM: 7.5,
	}
	writing := llm.ModelInfo{
		Ref: llm.ModelRef{Provider: "openai", Model: "gpt"}, InputUSDPerM: 3, CachedInputUSDPerM: 0.3,
		CacheWriteUSDPerM: 3.75, OutputUSDPerM: 15,
		FlexInputUSDPerM: 1.5, FlexCachedInputUSDPerM: 0.15, FlexCacheWriteUSDPerM: 1.875, FlexOutputUSDPerM: 7.5,
	}

	cases := []struct {
		name  string
		info  llm.ModelInfo
		tier  llm.ServiceTier
		usage llm.Usage
		want  float64
	}{
		{name: "nothing used", info: priced, tier: llm.TierDefault, usage: llm.Usage{}, want: 0},
		{name: "one million input tokens", info: priced, tier: llm.TierDefault, usage: llm.Usage{Input: 1_000_000}, want: 3},
		{name: "half a million output tokens", info: priced, tier: llm.TierDefault, usage: llm.Usage{Output: 500_000}, want: 7.5},
		{
			name: "both", info: priced, tier: llm.TierDefault,
			usage: llm.Usage{Input: 1_000_000, Output: 500_000, Total: 1_500_000}, want: 10.5,
		},
		{
			name: "a cached input token costs the cached rate", info: priced, tier: llm.TierDefault,
			usage: llm.Usage{Input: 1_000_000, CachedInput: 1_000_000, Total: 1_000_000}, want: 0.3,
		},
		{
			name: "a partly cached prompt is charged at both rates", info: priced, tier: llm.TierDefault,
			usage: llm.Usage{Input: 1_000_000, CachedInput: 900_000, Total: 1_000_000}, want: 0.57,
		},
		{
			name: "a model with no cached price charges the full rate", info: unpriced, tier: llm.TierDefault,
			usage: llm.Usage{Input: 1_000_000, CachedInput: 1_000_000, Total: 1_000_000}, want: 3,
		},
		{
			name: "more cached tokens than input tokens cannot go negative", info: priced, tier: llm.TierDefault,
			usage: llm.Usage{Input: 1_000_000, CachedInput: 4_000_000, Total: 1_000_000}, want: 0.3,
		},
		{
			name: "reasoning tokens are part of the output and are not charged twice", info: priced, tier: llm.TierDefault,
			usage: llm.Usage{Output: 1_000_000, Reasoning: 600_000, Total: 1_000_000}, want: 15,
		},
		{
			name: "an unnamed tier is charged the standard prices", info: flexible,
			usage: llm.Usage{Input: 1_000_000, Output: 1_000_000, Total: 2_000_000}, want: 18,
		},
		{
			name: "the default tier is charged the standard prices on a row that offers flex", info: flexible,
			tier: llm.TierDefault, usage: llm.Usage{Input: 1_000_000, Output: 1_000_000, Total: 2_000_000}, want: 18,
		},
		{
			name: "the flex tier is charged the flex prices", info: flexible, tier: llm.TierFlex,
			usage: llm.Usage{Input: 1_000_000, Output: 1_000_000, Total: 2_000_000}, want: 9,
		},
		{
			name: "a cached input token on flex costs the flex cached rate", info: flexible, tier: llm.TierFlex,
			usage: llm.Usage{Input: 1_000_000, CachedInput: 900_000, Total: 1_000_000}, want: 0.285,
		},
		{
			name: "a flex row with no cached price charges the flex input rate", info: flexibleUncached, tier: llm.TierFlex,
			usage: llm.Usage{Input: 1_000_000, CachedInput: 1_000_000, Total: 1_000_000}, want: 1.5,
		},
		{
			name: "flex on a row without flex prices is charged the standard prices", info: priced, tier: llm.TierFlex,
			usage: llm.Usage{Input: 1_000_000, Output: 1_000_000, Total: 2_000_000}, want: 18,
		},
		{
			name: "reasoning on flex is part of the flex output", info: flexible, tier: llm.TierFlex,
			usage: llm.Usage{Output: 1_000_000, Reasoning: 900_000, Total: 1_000_000}, want: 7.5,
		},
		{
			name: "a written input token costs the cache write rate", info: writing, tier: llm.TierDefault,
			usage: llm.Usage{Input: 1_000_000, CacheWrite: 1_000_000, Total: 1_000_000}, want: 3.75,
		},
		{
			name: "a prompt that is partly read, partly written and partly fresh is charged at three rates",
			info: writing, tier: llm.TierDefault,
			usage: llm.Usage{Input: 1_000_000, CachedInput: 500_000, CacheWrite: 250_000, Total: 1_000_000},
			want:  0.75 + 0.15 + 0.9375,
		},
		{
			name: "a model with no cache write price charges the input rate for a write", info: priced, tier: llm.TierDefault,
			usage: llm.Usage{Input: 1_000_000, CacheWrite: 1_000_000, Total: 1_000_000}, want: 3,
		},
		{
			name: "a written input token on flex costs the flex cache write rate", info: writing, tier: llm.TierFlex,
			usage: llm.Usage{Input: 1_000_000, CacheWrite: 1_000_000, Total: 1_000_000}, want: 1.875,
		},
		{
			name: "a flex row with no cache write price charges the flex input rate for a write", info: flexible,
			tier: llm.TierFlex, usage: llm.Usage{Input: 1_000_000, CacheWrite: 1_000_000, Total: 1_000_000}, want: 1.5,
		},
		{
			name: "more read and written tokens than input tokens cannot go negative", info: writing, tier: llm.TierDefault,
			usage: llm.Usage{Input: 1_000_000, CachedInput: 800_000, CacheWrite: 600_000, Total: 1_000_000},
			want:  0.24 + 0.75,
		},
		{
			name: "written tokens are charged beside the output", info: writing, tier: llm.TierDefault,
			usage: llm.Usage{Input: 1_000_000, CacheWrite: 400_000, Output: 100_000, Total: 1_100_000},
			want:  1.8 + 1.5 + 1.5,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := llm.Cost(tc.usage, tc.info, tc.tier); math.Abs(got-tc.want) > 1e-9 {
				t.Errorf("Cost() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestModelInfoOffersFlex(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		info llm.ModelInfo
		want bool
	}{
		{name: "a row with no flex prices", info: llm.ModelInfo{InputUSDPerM: 3, OutputUSDPerM: 15}, want: false},
		{
			name: "a row with flex prices",
			info: llm.ModelInfo{InputUSDPerM: 3, OutputUSDPerM: 15, FlexInputUSDPerM: 1.5, FlexOutputUSDPerM: 7.5},
			want: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := tc.info.OffersFlex(); got != tc.want {
				t.Errorf("OffersFlex() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestServiceTierValid(t *testing.T) {
	t.Parallel()

	cases := []struct {
		tier llm.ServiceTier
		want bool
	}{
		{tier: llm.TierDefault, want: true},
		{tier: llm.TierFlex, want: true},
		{tier: "", want: false},
		{tier: "standard", want: false},
		{tier: "priority", want: false},
	}

	for _, tc := range cases {
		t.Run(string(tc.tier), func(t *testing.T) {
			t.Parallel()
			if got := tc.tier.Valid(); got != tc.want {
				t.Errorf("ServiceTier(%q).Valid() = %v, want %v", tc.tier, got, tc.want)
			}
		})
	}
}

func TestServiceTierSpeaksTheProviderWireValues(t *testing.T) {
	t.Parallel()

	if llm.TierDefault != "default" || llm.TierFlex != "flex" {
		t.Fatalf("tiers = %q, %q, want default and flex", llm.TierDefault, llm.TierFlex)
	}
}

func TestAllowance(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		effort llm.ReasoningEffort
		want   int
	}{
		{name: "none reasons with nothing", effort: llm.EffortNone, want: 0},
		{name: "low", effort: llm.EffortLow, want: 2048},
		{name: "medium", effort: llm.EffortMedium, want: 8192},
		{name: "high", effort: llm.EffortHigh, want: 24576},
		{name: "xhigh", effort: llm.EffortXHigh, want: 49152},
		{name: "an undeclared effort is the provider default", effort: "", want: 8192},
		{name: "an unknown effort is the provider default", effort: "extreme", want: 8192},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := llm.Allowance(tc.effort); got != tc.want {
				t.Errorf("Allowance(%q) = %d, want %d", tc.effort, got, tc.want)
			}
		})
	}
}

func TestUsageAdd(t *testing.T) {
	t.Parallel()

	sum := llm.Usage{Input: 10, Output: 5, CachedInput: 4, CacheWrite: 2, Reasoning: 3, Total: 15}.
		Add(llm.Usage{Input: 1, Output: 2, CachedInput: 1, CacheWrite: 1, Reasoning: 2, Total: 3})
	if sum != (llm.Usage{Input: 11, Output: 7, CachedInput: 5, CacheWrite: 3, Reasoning: 5, Total: 18}) {
		t.Fatalf("Add = %+v", sum)
	}
}

func TestUsageAndPricesAreCamelCase(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		value any
		want  string
	}{
		{
			name:  "usage",
			value: llm.Usage{Input: 10, CachedInput: 4, CacheWrite: 3, Output: 6, Reasoning: 2, Total: 16},
			want:  `{"input":10,"cachedInput":4,"cacheWrite":3,"output":6,"reasoning":2,"total":16}`,
		},
		{
			name: "cache write and flex prices",
			value: llm.ModelInfo{
				InputUSDPerM: 3, CacheWriteUSDPerM: 3.75, OutputUSDPerM: 15,
				FlexInputUSDPerM: 1.5, FlexCachedInputUSDPerM: 0.15, FlexCacheWriteUSDPerM: 1.875, FlexOutputUSDPerM: 7.5,
			},
			want: `{"ref":{"provider":"","model":""},"contextTokens":0,"maxOutputTokens":0,"inputUsdPerM":3,` +
				`"cacheWriteUsdPerM":3.75,"outputUsdPerM":15,"flexInputUsdPerM":1.5,"flexCachedInputUsdPerM":0.15,` +
				`"flexCacheWriteUsdPerM":1.875,"flexOutputUsdPerM":7.5,` +
				`"rpm":0,"tpm":0,"supportsStructured":false,"supportsImages":false,"reasoning":false}`,
		},
		{
			name:  "a row without flex prices does not mention them",
			value: llm.ModelInfo{InputUSDPerM: 3, OutputUSDPerM: 15},
			want: `{"ref":{"provider":"","model":""},"contextTokens":0,"maxOutputTokens":0,"inputUsdPerM":3,` +
				`"outputUsdPerM":15,"rpm":0,"tpm":0,"supportsStructured":false,"supportsImages":false,"reasoning":false}`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			encoded, err := json.Marshal(tc.value)
			if err != nil {
				t.Fatalf("Marshal: %v", err)
			}
			if string(encoded) != tc.want {
				t.Fatalf("Marshal = %s, want %s", encoded, tc.want)
			}
		})
	}
}
