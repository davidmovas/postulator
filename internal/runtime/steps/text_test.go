package steps_test

import (
	"strings"
	"testing"

	"github.com/davidmovas/postulator/internal/adapters/wp/wptest"
	"github.com/davidmovas/postulator/internal/domain/run"
	"github.com/davidmovas/postulator/internal/runtime/steps"
)

func TestAKsesEscapedTitleIsNoMismatch(t *testing.T) {
	t.Parallel()

	deps, server := imageDepsWith(t, wptest.WithFilteredHTML())
	sc := publishContext(t)
	sc.Artifacts[run.ArtifactDraft] = run.Artifact{Kind: run.ArtifactDraft,
		Blob: []byte(strings.Replace(goodDraft, `"title":"Espresso guide"`, `"title":"Tea & espresso guide"`, 1))}

	result, err := drive(t, steps.Publish(deps), sc)
	if err != nil {
		t.Fatalf("Publish: %v", err)
	}
	if result.Next == run.TransitionPause {
		t.Fatalf("the publish paused with %q, want a title the site stored with its ampersand escaped taken as the one asked for", result.Message)
	}
	published := runPublish(t, deps, sc)
	if stored, _ := server.Lookup(published.WPID); stored.Title != "Tea &amp; espresso guide" {
		t.Fatalf("the site holds the title %q, want kses's escaped form, which this test is about", stored.Title)
	}
	if len(published.Mismatches) != 0 {
		t.Errorf("the publish reports %+v, want no mismatch: the text is the same", published.Mismatches)
	}
}
