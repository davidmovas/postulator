package steps_test

import (
	"strings"
	"testing"
	"time"

	"github.com/davidmovas/postulator/internal/domain/content"
	"github.com/davidmovas/postulator/internal/domain/run"
	"github.com/davidmovas/postulator/internal/kernel/id"
	"github.com/davidmovas/postulator/internal/runtime/steps"
)

func TestARunCarriesItsBriefToTheWriter(t *testing.T) {
	t.Parallel()

	p := newPipeline(t)
	p.start(t)
	child, err := p.pages.Get(t.Context(), p.pageID)
	if err != nil || child.ParentPageID == nil {
		t.Fatalf("read the page: %+v, %v", child, err)
	}
	parent, err := p.pages.Get(t.Context(), *child.ParentPageID)
	if err != nil {
		t.Fatalf("read the parent page: %v", err)
	}
	brief := run.Brief{
		Goal: "Win the home barista who already owns a machine", Tone: "warm and exact", Avoid: "brand names",
		Reference: "A double shot takes 18 grams of coffee.\n\nThe water sits at 93 °C.", StyleSample: "Short lines. One step each.",
	}
	directive := run.Directive{
		PageID: p.pageID, Notes: "Open with the dose.", AddSections: []string{"Dialing in"}, Keywords: []string{"espresso ratio"},
		MustLink: []run.MustLink{{PageID: parent.ID, Anchor: "our coffee guide"}}, MinWords: 400, MaxWords: 800,
	}
	queued, err := p.engine.Enqueue(t.Context(), run.Run{
		ID: id.New(), SiteID: p.siteID, Kind: run.KindGenerate, Targets: []string{p.pageID}, Brief: brief,
		Recipe: p.recipe, TemplateID: "guide", TemplateVersion: 1, PublishMode: run.PublishDraft,
	}, directive)
	if err != nil {
		t.Fatalf("Enqueue: %v", err)
	}
	item := p.waitFor(t, queued.ID)

	wanted := []string{
		"THE RUN'S BRIEF", "Goal: Win the home barista who already owns a machine", "Tone: warm and exact", "Avoid: brand names",
		"For this page: Open with the dose.", "Add these sections: Dialing in", "Use these keywords: espresso ratio",
		"Link to /drinks/coffee/ with the anchor “our coffee guide”", "Length for this run: 400 to 800 words",
		"A double shot takes 18 grams of coffee.", "Short lines. One step each.",
	}
	for _, step := range []string{steps.NameGenerateBody, steps.NameGenerateMeta, steps.NameJudge} {
		prompt := p.promptTo(t, step)
		for _, want := range wanted {
			if step != steps.NameGenerateBody && strings.Contains(want, "Short lines") {
				continue
			}
			if !strings.Contains(prompt, want) {
				t.Errorf("the %s prompt lacks %q", step, want)
			}
		}
	}

	var links content.LinkContext
	p.artifactOf(t, item.ID, run.ArtifactLinkContext, &links)
	target, found := links.ByPageID(parent.ID)
	if !found || !target.Required || len(target.Anchors) == 0 || target.Anchors[0] != "our coffee guide" {
		t.Errorf("the parent in the link plan = %+v, want it required with the directive's anchor first", target)
	}
}

func (p *pipeline) waitFor(t *testing.T, runID string) run.Item {
	t.Helper()

	deadline := waitDeadline(t)
	for time.Now().Before(deadline) {
		items, err := p.items.ByRun(t.Context(), runID)
		if err == nil && len(items) == 1 && items[0].Status.Terminal() {
			if items[0].Status != run.StatusCompleted {
				t.Fatalf("the item reached %q: %s", items[0].Status, items[0].Error)
			}
			return items[0]
		}
		if err == nil && len(items) == 1 {
			p.acceptAReview(t, items[0])
		}
		time.Sleep(pollInterval)
	}
	t.Fatal("timed out waiting for the run to finish")
	return run.Item{}
}

func (p *pipeline) promptTo(t *testing.T, step string) string {
	t.Helper()

	requests := p.llm.Requests()
	for i := range requests {
		req := &requests[i]
		if req.Meta.Step != step {
			continue
		}
		var held strings.Builder
		held.WriteString(req.System)
		for _, message := range req.Messages {
			held.WriteString("\n" + message.Text)
		}
		return held.String()
	}
	t.Fatalf("the model was never asked for %s", step)
	return ""
}
