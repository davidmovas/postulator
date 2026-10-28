package runtime_test

import (
	"context"
	"strings"
	"testing"

	"github.com/davidmovas/postulator/internal/domain/run"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

func TestARunWhoseEveryItemFailedCarriesTheirReason(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		pages int
		want  func(item string) string
	}{
		{name: "one item", pages: 1, want: func(item string) string { return item }},
		{name: "two items", pages: 2, want: func(item string) string { return "2 items failed; the last: " + item }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			harness := newHarness(t, tc.pages)
			refused := producing("generate_body", run.ArtifactBodyHTML, nil, func(context.Context, *run.StepContext) (run.Result, error) {
				return run.Result{}, errors.New(errors.Invalid, "reading the body of /about/: WordPress rejected the request: WordPress answered 400 on GET /wp-json/wp/v2/pages/7")
			})

			engine := harness.engine(t, mustRegister(t, refused))
			queued, err := engine.Enqueue(t.Context(), harness.newRun(recipeOf("generate_body")))
			if err != nil {
				t.Fatalf("Enqueue: %v", err)
			}

			finished := harness.waitForRun(t, queued.ID, run.StatusFailed)
			items, err := harness.items.ByRun(t.Context(), queued.ID)
			if err != nil || len(items) != tc.pages {
				t.Fatalf("items = %d, %v", len(items), err)
			}
			if !strings.Contains(items[0].Error, "WordPress answered 400") || finished.Error != tc.want(items[0].Error) {
				t.Errorf("the run's error = %q, want %q", finished.Error, tc.want(items[0].Error))
			}
		})
	}
}
