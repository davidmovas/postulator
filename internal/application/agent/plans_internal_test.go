package agent

import (
	"encoding/json"
	"testing"
)

func TestAStepsRunIsReadWhereverItsToolAnswersIt(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		result string
		want   string
	}{
		{name: "a run start", result: `{"runId":"r-1","estimate":{}}`, want: "r-1"},
		{name: "an edit of a field or a block", result: `{"edit":{"runId":"r-2"}}`, want: "r-2"},
		{name: "a write that starts no run", result: `{"page":{"id":"p-1"}}`, want: ""},
		{name: "an answer that is no object", result: `[]`, want: ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := runIDOf(json.RawMessage(tc.result)); got != tc.want {
				t.Errorf("runIDOf(%s) = %q, want %q", tc.result, got, tc.want)
			}
		})
	}
}
