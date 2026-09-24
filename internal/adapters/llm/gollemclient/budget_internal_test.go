package gollemclient

import (
	"testing"

	"github.com/davidmovas/postulator/internal/domain/llm"
)

func TestBudget(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		asked int
		info  llm.ModelInfo
		want  int
	}{
		{
			name:  "no ceiling stays uncapped",
			asked: 0,
			info:  llm.ModelInfo{Reasoning: true, ReasoningEffort: llm.EffortHigh},
			want:  0,
		},
		{
			name:  "a model that does not reason is asked for what the caller wanted",
			asked: 256,
			info:  llm.ModelInfo{MaxOutputTokens: 4096},
			want:  256,
		},
		{
			name:  "a model that does not reason is held to its own ceiling",
			asked: 8192,
			info:  llm.ModelInfo{MaxOutputTokens: 4096},
			want:  4096,
		},
		{
			name:  "an effort of none reasons with nothing and needs no allowance",
			asked: 256,
			info:  llm.ModelInfo{Reasoning: true, ReasoningEffort: llm.EffortNone, MaxOutputTokens: 128000},
			want:  256,
		},
		{
			name:  "a low effort carries the low allowance",
			asked: 16,
			info:  llm.ModelInfo{Reasoning: true, ReasoningEffort: llm.EffortLow, MaxOutputTokens: 128000},
			want:  16 + allowanceLow,
		},
		{
			name:  "a declared medium effort carries the medium allowance",
			asked: 512,
			info:  llm.ModelInfo{Reasoning: true, ReasoningEffort: llm.EffortMedium, MaxOutputTokens: 128000},
			want:  512 + allowanceMedium,
		},
		{
			name:  "an undeclared effort is treated as the provider default",
			asked: 512,
			info:  llm.ModelInfo{Reasoning: true, MaxOutputTokens: 128000},
			want:  512 + allowanceMedium,
		},
		{
			name:  "a high effort carries the high allowance",
			asked: 1024,
			info:  llm.ModelInfo{Reasoning: true, ReasoningEffort: llm.EffortHigh, MaxOutputTokens: 128000},
			want:  1024 + allowanceHigh,
		},
		{
			name:  "an xhigh effort carries the largest allowance",
			asked: 1024,
			info:  llm.ModelInfo{Reasoning: true, ReasoningEffort: llm.EffortXHigh, MaxOutputTokens: 128000},
			want:  1024 + allowanceXHigh,
		},
		{
			name:  "the allowance never exceeds what the model can emit",
			asked: 512,
			info:  llm.ModelInfo{Reasoning: true, ReasoningEffort: llm.EffortHigh, MaxOutputTokens: 4096},
			want:  4096,
		},
		{
			name:  "a model with no declared output ceiling is not clamped",
			asked: 512,
			info:  llm.ModelInfo{Reasoning: true, ReasoningEffort: llm.EffortLow},
			want:  512 + allowanceLow,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := budget(tc.asked, tc.info); got != tc.want {
				t.Errorf("budget(%d) = %d, want %d", tc.asked, got, tc.want)
			}
		})
	}
}

func TestFinishReasonReadsTheCeilingTheProviderWasGiven(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		ceiling int
		output  int
		want    string
	}{
		{name: "no ceiling can never be reached", ceiling: 0, output: 9000, want: "stop"},
		{name: "an answer inside the ceiling stopped on its own", ceiling: 2064, output: 900, want: "stop"},
		{name: "an answer at the ceiling ran out of room", ceiling: 2064, output: 2064, want: "length"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := finishReason(tc.ceiling, llm.Usage{Output: tc.output})
			if string(got) != tc.want {
				t.Errorf("finishReason = %s, want %s", got, tc.want)
			}
		})
	}
}
