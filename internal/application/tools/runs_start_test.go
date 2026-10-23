package tools_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/davidmovas/postulator/internal/adapters/sqlite"
	"github.com/davidmovas/postulator/internal/application/tools"
	domainagent "github.com/davidmovas/postulator/internal/domain/agent"
)

func onTheSite(t *testing.T, seeded fixture) {
	t.Helper()

	repo := sqlite.NewPageRepo(seeded.store)
	page, err := repo.Get(t.Context(), seeded.page)
	if err != nil {
		t.Fatalf("read the page: %v", err)
	}
	wpID := int64(41)
	page.WPID, page.EntityID = &wpID, &seeded.entity
	if err = repo.Update(t.Context(), page); err != nil {
		t.Fatalf("put the page on the site: %v", err)
	}
}

func TestRunsStartTakesAFieldScopeAndShowsItsEstimateOnTheCard(t *testing.T) {
	t.Parallel()

	registry, binding, seeded := wired(t)
	onTheSite(t, seeded)
	binding.Mode = domainagent.ModeConfirm
	args := `{"pageIds":["` + seeded.page + `"],"templateId":"` + seeded.template +
		`","fields":["seo_description"],"rewrite":["` + seeded.page + `"],"publishMode":"publish"}`

	out, err := registry.Call(t.Context(), binding, "runs_start", json.RawMessage(args))
	if err != nil {
		t.Fatalf("runs_start: %v", err)
	}
	asked, ok := out.(tools.Confirmation)
	if !ok {
		t.Fatalf("runs_start in confirm mode answered %+v, want a card", out)
	}
	for _, said := range []string{"/coffee/", "seo_description", "estimated", "own text"} {
		if !strings.Contains(asked.Summary, said) {
			t.Errorf("the card says %q, want it to name %q", asked.Summary, said)
		}
	}

	binding.Mode = domainagent.ModeAutonomous
	started, startErr := registry.Call(t.Context(), binding, "runs_start", json.RawMessage(args))
	if _, waited := started.(tools.Confirmation); waited {
		t.Fatalf("an autonomous runs_start answered %+v, want the start itself, as the mode says", started)
	}
	t.Logf("the autonomous start answered %v, the fixture's engine knowing no step", startErr)
}
