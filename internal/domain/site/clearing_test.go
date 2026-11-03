package site_test

import (
	"testing"

	"github.com/davidmovas/postulator/internal/domain/site"
)

func TestClearingTheModelTakesTheHistoryWithIt(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		asked site.Clearing
		want  site.Clearing
		empty bool
	}{
		{name: "nothing asked", asked: site.Clearing{}, want: site.Clearing{}, empty: true},
		{name: "the model takes the history", asked: site.Clearing{Model: true}, want: site.Clearing{Model: true, History: true}},
		{name: "the graph alone", asked: site.Clearing{Graph: true}, want: site.Clearing{Graph: true}},
		{name: "the history alone", asked: site.Clearing{History: true}, want: site.Clearing{History: true}},
		{name: "everything", asked: site.Clearing{Model: true, Graph: true}, want: site.Clearing{Model: true, Graph: true, History: true}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := tc.asked.Normalized()
			if got != tc.want {
				t.Errorf("Normalized() = %+v, want %+v", got, tc.want)
			}
			if got.Empty() != tc.empty {
				t.Errorf("Empty() = %v, want %v", got.Empty(), tc.empty)
			}
		})
	}
}
