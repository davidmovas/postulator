package llm_test

import (
	"encoding/json"
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
		name  string
		ref   llm.ModelRef
		valid bool
		text  string
	}{
		{name: "complete", ref: llm.ModelRef{Provider: "openai", Model: "gpt"}, valid: true, text: "openai:gpt"},
		{name: "no provider", ref: llm.ModelRef{Model: "gpt"}, valid: false, text: ":gpt"},
		{name: "no model", ref: llm.ModelRef{Provider: "openai"}, valid: false, text: "openai:"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := tc.ref.Valid(); got != tc.valid {
				t.Errorf("Valid() = %v, want %v", got, tc.valid)
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

	info := llm.ModelInfo{Ref: llm.ModelRef{Provider: "openai", Model: "gpt"}, InputUSDPerM: 3, OutputUSDPerM: 15}

	cases := []struct {
		name  string
		usage llm.Usage
		want  float64
	}{
		{name: "nothing used", usage: llm.Usage{}, want: 0},
		{name: "one million input tokens", usage: llm.Usage{Input: 1_000_000}, want: 3},
		{name: "half a million output tokens", usage: llm.Usage{Output: 500_000}, want: 7.5},
		{name: "both", usage: llm.Usage{Input: 1_000_000, Output: 500_000, Total: 1_500_000}, want: 10.5},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := llm.Cost(tc.usage, info); got != tc.want {
				t.Errorf("Cost() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestUsageAdd(t *testing.T) {
	t.Parallel()

	sum := llm.Usage{Input: 10, Output: 5, Total: 15}.Add(llm.Usage{Input: 1, Output: 2, Total: 3})
	if sum != (llm.Usage{Input: 11, Output: 7, Total: 18}) {
		t.Fatalf("Add = %+v", sum)
	}
}
