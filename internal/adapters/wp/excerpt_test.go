package wp

import (
	"encoding/json"
	"testing"
)

func TestAnExcerptIsTheOwnersOrNothingNeverWordPresssAutoExcerpt(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		payload string
		want    string
	}{
		{name: "an excerpt the owner wrote", payload: `{"excerpt":{"raw":"Pull a shot.","rendered":"<p>Pull a shot.</p>"}}`, want: "Pull a shot."},
		{name: "no excerpt, WordPress's auto-excerpt rendered", payload: `{"excerpt":{"raw":"","rendered":"<p>Espresso is a way to make coffee&hellip;</p>"}}`, want: ""},
		{name: "a view without the raw", payload: `{"excerpt":{"rendered":"<p>Pull a shot.</p>"}}`, want: "<p>Pull a shot.</p>"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var payload itemPayload
			if err := json.Unmarshal([]byte(tc.payload), &payload); err != nil {
				t.Fatalf("decode: %v", err)
			}
			if got := payload.item(TypePost).Excerpt; got != tc.want {
				t.Errorf("Excerpt = %q, want %q", got, tc.want)
			}
		})
	}
}
