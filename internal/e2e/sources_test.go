//go:build e2e

package e2e_test

import (
	"context"
	"encoding/json"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/davidmovas/postulator/internal/adapters/llm/fake"
	"github.com/davidmovas/postulator/internal/adapters/wp"
	port "github.com/davidmovas/postulator/internal/application/llm"
	"github.com/davidmovas/postulator/internal/application/sites"
	"github.com/davidmovas/postulator/internal/application/templates"
	"github.com/davidmovas/postulator/internal/domain/content"
	"github.com/davidmovas/postulator/internal/domain/run"
	"github.com/davidmovas/postulator/internal/domain/template"
	"github.com/davidmovas/postulator/internal/runtime/steps"
)

const (
	sourceAddress = "https://www.who.int/peptide-storage"
	sourceFact    = "Lyophilised peptides keep longest frozen and dry."
	sourceAnchor  = "keep longest frozen and dry"
	rivalAddress  = "https://rival.example/peptides"
)

func citingBody(req port.Request) string {
	written := bodyFromPrompt(req)
	var draft content.DraftAnswer
	if err := json.Unmarshal([]byte(written), &draft); err != nil || len(draft.Sections) == 0 {
		return written
	}
	draft.Sections[0].HTML += `<p>Peptides <a href="` + sourceAddress + `?utm_source=openai">` + sourceAnchor +
		`</a>, and <a href="` + rivalAddress + `">a rival</a> says so too.</p>`
	encoded, err := json.Marshal(draft)
	if err != nil {
		return written
	}
	return string(encoded)
}

func TestAResearchedPageReachesTheSiteWithTheLinksToItsSources(t *testing.T) {
	live := newSite(t)
	requirePlugin(t, live.env, true)
	stand := writerStandOf(t, live.env)
	path := stand.clientPage

	core := openCoreWith(t, fake.NewScripted(
		fake.Reply{
			Step: steps.NameResearch, Text: "- " + sourceFact + " ([who.int](" + sourceAddress + "?utm_source=openai)) [who.int]",
			Sources: []port.Source{{URL: sourceAddress + "?utm_source=openai", Title: "Peptide storage"}},
		},
		fake.Reply{Step: steps.NameGenerateBody, Make: citingBody},
	))
	owner, err := core.Sites.Create(t.Context(), sites.CreateRequest{
		Name: "Researched Shop", BaseURL: live.env.baseURL, Username: live.env.user, Password: live.env.pass, AllowInsecure: true,
	})
	if err != nil {
		t.Fatalf("create the site: %v", err)
	}
	siteID := owner.Site.ID
	runSync(t, core, siteID)
	seedStandGraph(t, core, siteID, stand.graph)
	page := pagesByPath(t, core.Pages, siteID)[path]
	if page.WPID == nil {
		t.Fatalf("%s is not on the site; reset the stand", path)
	}
	client := wordpress(t, core, siteID)
	itemType := wp.ItemType(page.WPType)
	base := rawOf(t, client, itemType, *page.WPID)
	t.Cleanup(func() {
		if held, readErr := client.GetRaw(context.WithoutCancel(t.Context()), itemType, *page.WPID); readErr == nil && held.Content != base {
			wpCLI(t, live.env, "post", "update", strconv.FormatInt(*page.WPID, 10), "--post_content="+base, "--user="+live.env.user)
		}
	})

	researched := slices.Insert(writing(steps.NamePublish), 1, template.StepSpec{Name: steps.NameResearch, Enabled: true})
	awaitRun(t, core.Runs, startRun(t, core, siteID, run.PublishLive, researched, page.ID))
	held := rawOf(t, client, itemType, *page.WPID)
	if !strings.Contains(held, `<a href="`+sourceAddress+`">`+sourceAnchor+`</a>`) {
		t.Fatalf("%s holds no link to the cited source, without its tracking:\n%s", path, held)
	}
	if strings.Contains(held, rivalAddress) || !strings.Contains(held, "a rival") {
		t.Errorf("%s kept the link to an address the research never cited:\n%s", path, held)
	}

	resolved, err := core.Templates.ResolveForPage(t.Context(), templates.ResolveForPageRequest{PageID: page.ID})
	if err != nil {
		t.Fatalf("resolve the template of %s: %v", path, err)
	}
	if !resolved.Spec.SourceLinks.Cites() {
		t.Fatalf("the shipped template %s links no source: %+v", resolved.Name, resolved.Spec.SourceLinks)
	}
	if _, err = core.Templates.SetOverride(t.Context(), templates.SetOverrideRequest{
		TemplateID: resolved.TemplateID, Scope: string(template.OverrideSite), TargetID: siteID,
		Patch: json.RawMessage(`{"sourceLinks":{"max":0}}`),
	}); err != nil {
		t.Fatalf("turn the links to sources off for the site: %v", err)
	}
	awaitRun(t, core.Runs, startRun(t, core, siteID, run.PublishLive, researched, page.ID))
	if again := rawOf(t, client, itemType, *page.WPID); strings.Contains(again, sourceAddress) || !strings.Contains(again, sourceAnchor) {
		t.Errorf("under a template that links no source %s holds:\n%s", path, again)
	}
}
