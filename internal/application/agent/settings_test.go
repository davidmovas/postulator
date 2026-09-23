package agent_test

import (
	"encoding/json"
	"testing"

	"github.com/davidmovas/postulator/internal/application/agent"
	"github.com/davidmovas/postulator/internal/kernel/settings"
)

func TestTheHistoryToolResultCeilingIsDeclaredAndBounded(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		stored string
		want   int
		refuse bool
	}{
		{name: "nothing stored", want: agent.DefaultHistoryToolResultBytes},
		{name: "a value inside the range", stored: "8192", want: 8192},
		{name: "the smallest allowed", stored: "512", want: agent.MinHistoryToolResultBytes},
		{name: "the largest allowed", stored: "65536", want: agent.MaxHistoryToolResultBytes},
		{name: "under the floor", stored: "511", refuse: true},
		{name: "over the ceiling", stored: "65537", refuse: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			declarations := settings.Default()
			if !declarations.Has("agent.historyToolResultBytes") {
				t.Fatal("agent.historyToolResultBytes is not a declared setting")
			}

			values := declarations.NewValues()
			if tc.stored == "" {
				if got := agent.HistoryToolResultBytes(values); got != tc.want {
					t.Fatalf("HistoryToolResultBytes = %d, want %d", got, tc.want)
				}
				return
			}

			raw := json.RawMessage(tc.stored)
			if err := declarations.Validate("agent.historyToolResultBytes", raw); tc.refuse != (err != nil) {
				t.Fatalf("Validate(%s) = %v, want refused = %t", tc.stored, err, tc.refuse)
			}
			if tc.refuse {
				return
			}

			if _, err := declarations.Apply(values, map[string]json.RawMessage{
				"agent.historyToolResultBytes": raw,
			}); err != nil {
				t.Fatalf("Apply: %v", err)
			}
			if got := agent.HistoryToolResultBytes(values); got != tc.want {
				t.Fatalf("HistoryToolResultBytes = %d, want %d", got, tc.want)
			}
		})
	}
}
