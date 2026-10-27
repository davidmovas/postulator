//go:build e2e

package e2e_test

import (
	"encoding/json"
	"slices"
	"strings"
	"testing"

	"github.com/davidmovas/postulator/internal/adapters/llm/fake"
	port "github.com/davidmovas/postulator/internal/application/llm"
	"github.com/davidmovas/postulator/internal/application/pages"
	"github.com/davidmovas/postulator/internal/application/runs"
	"github.com/davidmovas/postulator/internal/application/sitemodel"
	"github.com/davidmovas/postulator/internal/application/tools"
	"github.com/davidmovas/postulator/internal/domain/content"
	"github.com/davidmovas/postulator/internal/domain/contextpack"
	"github.com/davidmovas/postulator/internal/domain/finding"
	"github.com/davidmovas/postulator/internal/domain/run"
	"github.com/davidmovas/postulator/internal/domain/template"
	"github.com/davidmovas/postulator/internal/runtime/steps"
)

const (
	briefHeading = "THE RUN'S BRIEF"
	walkLimit    = 3
	walkPlanned  = "/peptides/capability-walk/"
)

func bareDraft(req port.Request) string {
	prompt := req.Messages[len(req.Messages)-1].Text
	primary := lineAfter(prompt, "Primary keyword: ")
	draft := draftHead(lineAfter(prompt, "Title: "), lineAfter(prompt, "H1: "), primary)
	draft.WriteString(`,"sections":[`)
	for i, entry := range listUnder(prompt, markerSections) {
		if i > 0 {
			draft.WriteString(",")
		}
		heading := headingOf(entry)
		draft.WriteString(`{"heading":` + quoted(heading) + `,"html":` + quoted("<p>"+filler(heading, 8)+"</p>") + `}`)
	}
	draft.WriteString(`],"summary":` + quoted("What the shop knows about "+primary+".") + `}`)
	return draft.String()
}

func promptText(req port.Request) string {
	var held strings.Builder
	held.WriteString(req.System)
	for _, message := range req.Messages {
		if !strings.HasPrefix(message.Text, fake.JSONDirective) {
			held.WriteString("\n" + message.Text)
		}
	}
	return held.String()
}

func requestsOf(script *fake.Scripted, runID, itemID, step string) []port.Request {
	return slices.DeleteFunc(script.Requests(), func(req port.Request) bool {
		return req.Meta.RunID != runID || req.Meta.ItemID != itemID || req.Meta.Step != step
	})
}

func linkContextOf(t *testing.T, c *capability, itemID string) content.LinkContext {
	t.Helper()

	stored, err := c.core.Runs.GetArtifact(t.Context(), runs.GetArtifactRequest{ItemID: itemID, Kind: string(run.ArtifactLinkContext)})
	if err != nil {
		t.Fatalf("read the link plan of item %s: %v", itemID, err)
	}
	var plan content.LinkContext
	if err = json.Unmarshal([]byte(stored.Artifact.Content), &plan); err != nil {
		t.Fatalf("decode the link plan of item %s: %v", itemID, err)
	}
	return plan
}

func briefRecipe() []template.StepSpec {
	recipe := make([]template.StepSpec, 0, 5)
	for _, name := range briefSteps() {
		recipe = append(recipe, template.StepSpec{Name: name, Enabled: true})
	}
	return recipe
}

func briefSteps() []string {
	return []string{steps.NameResolveContext, steps.NameGenerateBody, steps.NameInsertLinks, steps.NameRepairLinks, steps.NameJudge}
}

func TestTheAssistantWalksTheWholeTreeInPages(t *testing.T) {
	c := openCapability(t, fake.NewScripted(), withTheTree)
	conversation := c.conversation(t)

	whole := treeOf(t, c, conversation, map[string]any{"view": "walk", "limit": 200})
	if whole.Nodes.Next != "" {
		t.Fatalf("the replica's tree does not fit one page of 200, so the reference walk is incomplete")
	}
	roots := treeOf(t, c, conversation, map[string]any{"view": "roots", "limit": 200})
	counted := 0
	for i := range roots.Nodes.Items {
		counted += 1 + roots.Nodes.Items[i].Descendants
	}
	if counted != len(whole.Nodes.Items) {
		t.Errorf("the roots count %d pages with their descendants, and the walk holds %d", counted, len(whole.Nodes.Items))
	}

	walked, calls := walkInPages(t, c, conversation)
	if got, want := pathsInOrder(walked), pathsInOrder(whole.Nodes.Items); !slices.Equal(got, want) {
		t.Errorf("the paged walk gave %v, the whole walk %v", got, want)
	}
	if calls < len(whole.Nodes.Items)/walkLimit {
		t.Errorf("the walk of %d pages took %d calls of %d", len(whole.Nodes.Items), calls, walkLimit)
	}
	at := make(map[string]int, len(walked))
	for i := range walked {
		at[walked[i].Page.PageID] = i
	}
	for i := range walked {
		node := &walked[i]
		if parent, held := at[node.ParentID]; node.ParentID != "" && (!held || parent >= i) {
			t.Errorf("the walk gives %s at %d before its parent %s at %d", node.Page.Path, i, node.ParentID, parent)
		}
	}
	twins := map[string]bool{}
	for i := range walked {
		if walked[i].Twin != nil {
			twins[walked[i].Twin.PageID] = true
		}
	}
	for path, page := range livePages(c) {
		if _, held := at[page.ID]; !held && !twins[page.ID] {
			t.Errorf("the walk missed %s, which is neither on the tree nor folded into a hub", path)
		}
	}
	t.Logf("the walk: %d pages in %d calls of %d, generation %d", len(walked), calls, walkLimit, whole.Generation)

	first := treeOf(t, c, conversation, map[string]any{"view": "walk", "limit": walkLimit})
	if _, err := c.core.Pages.Create(t.Context(), pages.CreateRequest{
		SiteID: c.siteID, Path: walkPlanned, Title: "Capability Walk", Status: "planned",
	}); err != nil {
		t.Fatalf("plan %s mid-walk: %v", walkPlanned, err)
	}
	moved := treeOf(t, c, conversation, map[string]any{
		"view": "walk", "limit": walkLimit, "cursor": string(first.Nodes.Next), "generation": first.Generation,
	})
	if !moved.Stale || moved.Generation == first.Generation {
		t.Fatalf("a walk whose tree moved under it answered stale %v at generation %d, it began at %d", moved.Stale,
			moved.Generation, first.Generation)
	}
	again, _ := walkInPages(t, c, conversation)
	if len(again) != len(walked)+1 || !slices.Contains(pathsInOrder(again), walkPlanned) {
		t.Errorf("the walk started over holds %d pages, want the %d before and %s", len(again), len(walked), walkPlanned)
	}
}

func walkInPages(t *testing.T, c *capability, conversation string) (walked []contextpack.MapNode, calls int) {
	t.Helper()

	walked = make([]contextpack.MapNode, 0, 32)
	seen := map[string]bool{}
	cursor, generation := "", int64(0)
	for {
		args := map[string]any{"view": "walk", "limit": walkLimit}
		if cursor != "" {
			args["cursor"], args["generation"] = cursor, generation
		}
		page := treeOf(t, c, conversation, args)
		calls++
		if page.Stale {
			t.Fatalf("the walk went stale at call %d with nothing changing the tree", calls)
		}
		if generation != 0 && page.Generation != generation {
			t.Fatalf("the generation moved from %d to %d mid-walk", generation, page.Generation)
		}
		generation = page.Generation
		for i := range page.Nodes.Items {
			node := page.Nodes.Items[i]
			if seen[node.Page.PageID] {
				t.Errorf("the walk gave %s twice", node.Page.Path)
			}
			seen[node.Page.PageID] = true
			walked = append(walked, node)
		}
		if page.Nodes.Next == "" {
			return walked, calls
		}
		cursor = string(page.Nodes.Next)
	}
}

func pathsInOrder(nodes []contextpack.MapNode) []string {
	out := make([]string, 0, len(nodes))
	for i := range nodes {
		out = append(out, nodes[i].Page.Path)
	}
	return out
}

func treeParents(seeds []seededEntity, entities map[string]string) map[string]string {
	out := make(map[string]string, len(seeds))
	for _, seed := range seeds {
		out[entities[seed.name]] = entities[seed.parent]
	}
	return out
}

func TestTheAssistantSeesTheTwoHierarchiesDisagree(t *testing.T) {
	c := openCapability(t, fake.NewScripted(), withAFlatGraph)
	runSync(t, c.core, c.siteID)
	tree := writerStandOf(t, c.live.env).graph
	want := treeParents(tree, c.entities)
	moving := map[string]bool{}
	for entityID, parentID := range want {
		if parentID != "" {
			moving[entityID] = true
		}
	}
	conversation := c.conversation(t)

	listed := ask[tools.FindingsResponse](t, c, conversation, "findings", map[string]any{
		"op": "list", "code": string(finding.EntityParentDiffers), "limit": 200,
	})
	differs := map[string]bool{}
	for _, held := range listed.List.Items {
		differs[held.EntityID] = true
		options := make([]string, 0, len(held.Options))
		for _, option := range held.Options {
			options = append(options, option.Name)
		}
		if !slices.Equal(options, []string{string(finding.AdoptTree), string(finding.KeepGraph), string(finding.Ignore)}) {
			t.Errorf("the entity_parent_differs finding on %s offers %v", held.PagePath, options)
		}
	}
	if len(differs) != len(moving) {
		t.Fatalf("the flat graph raised entity_parent_differs on %d entities, the tree places %d under a parent: %+v",
			len(differs), len(moving), listed.List.Items)
	}
	for entityID := range moving {
		if !differs[entityID] {
			t.Errorf("the entity %s sits under a parent in the tree and no finding says the graph disagrees", entityID)
		}
		detail := entityRead(t, c, conversation, entityID)
		if len(detail.Parents) != 0 || detail.ParentWhy == "" || detail.Tree == nil || detail.Tree.EntityID != want[entityID] {
			t.Errorf("%s reads its graph parents %+v (%q) beside the tree's %+v, want none in the graph and %s in the tree",
				detail.Name, detail.Parents, detail.ParentWhy, detail.Tree, want[entityID])
		}
	}

	adoption := ask[tools.GraphReadResponse](t, c, conversation, "graph_read", map[string]any{"view": "adoption"}).Adoption
	if adoption == nil || adoption.Adopting != len(moving) || adoption.Skipped != 0 || adoption.Hash == "" {
		t.Fatalf("the adoption preview is %+v, want the %d entities the tree places", adoption, len(moving))
	}
	for _, step := range adoption.Steps.Items {
		if step.To == nil || step.To.ID != want[step.EntityID] || len(step.From) != 0 || step.Skip != "" {
			t.Errorf("the adoption moves %s from %+v to %+v, want it under %s", step.Name, step.From, step.To, want[step.EntityID])
		}
	}

	asked := askACard(t, c, conversation, "graph_edit", map[string]any{"op": "adopt_parents", "hash": adoption.Hash})
	if !strings.HasPrefix(asked.Summary, "Give 4 entities the parent the page tree shows") {
		t.Errorf("the adoption card says %q", asked.Summary)
	}
	if held := entityRead(t, c, conversation, c.entities["TB-500"]); len(held.Parents) != 0 {
		t.Fatalf("the adoption wrote before the person approved it: %+v", held.Parents)
	}
	approveTheCard(t, c.core, conversation, "graph_edit")

	if open := openFindings(t, c, finding.EntityParentDiffers); len(open) != 0 {
		t.Errorf("after the adoption the findings %+v are still open", open)
	}
	for _, held := range findingsIn(t, c, finding.EntityParentDiffers, finding.Resolved) {
		if held.Resolution != finding.ResolutionParentAdopted || held.ResolvedBy != finding.ResolvedByAgent {
			t.Errorf("the finding on %s settled as %s by %s, want parent_adopted by the agent", held.PagePath, held.Resolution, held.ResolvedBy)
		}
	}
	for entityID, parentID := range want {
		detail := entityRead(t, c, conversation, entityID)
		got := ""
		if len(detail.Parents) == 1 && detail.Parents[0].Status == "approved" {
			got = detail.Parents[0].EntityID
		}
		if got != parentID || len(detail.Parents) > 1 {
			t.Errorf("after the adoption %s has the graph parents %+v, the tree's is %q", detail.Name, detail.Parents, parentID)
		}
	}
	after := ask[tools.GraphReadResponse](t, c, conversation, "graph_read", map[string]any{"view": "adoption"}).Adoption
	if after == nil || after.Adopting != 0 {
		t.Errorf("after the adoption the preview still moves %+v", after)
	}
}

type briefPage struct {
	path   string
	notes  string
	target string
	anchor string
}

func briefPages() []briefPage {
	return []briefPage{
		{path: capabilityCategory, notes: "Open with what the two healing peptides share.", target: capabilityHub, anchor: "every research peptide we stock"},
		{path: capabilityBPC, notes: "Lead with how BPC-157 is stored once it arrives.", target: capabilityCategory, anchor: "the healing peptides overview"},
		{path: capabilityTB, notes: "Say how TB-500 differs from BPC-157 in handling.", target: capabilityCategory, anchor: "all healing peptides"},
	}
}

func (c *capability) directivesOf(t *testing.T, wanted []briefPage) ([]string, []run.Directive) {
	t.Helper()

	ids := make([]string, 0, len(wanted))
	directives := make([]run.Directive, 0, len(wanted))
	for _, held := range wanted {
		page := c.page(t, held.path)
		ids = append(ids, page.ID)
		directives = append(directives, run.Directive{
			PageID: page.ID, Notes: held.notes, MustLink: []run.MustLink{{PageID: c.page(t, held.target).ID, Anchor: held.anchor}},
		})
	}
	return ids, directives
}

func TestARunCarriesItsBriefToTheWriter(t *testing.T) {
	script := fake.NewScripted(
		fake.Reply{Step: steps.NameGenerateBody, Make: bareDraft},
		fake.Reply{Step: steps.NameRepairLinks, Make: sentenceFromPrompt},
		fake.Reply{Step: steps.NameJudge, Text: guideJudge},
	)
	c := openCapability(t, script, withTheTree)
	wanted := briefPages()

	t.Run("started by the assistant", func(t *testing.T) {
		brief := run.Brief{Goal: "Win the lab manager who reorders every month", Focus: "storage and handling",
			Tone: "precise and calm", Avoid: "dosage claims"}
		ids, directives := c.directivesOf(t, wanted)
		conversation := c.conversation(t)
		asked := askACard(t, c, conversation, "runs_control", map[string]any{
			"op": "start", "pageIds": ids, "steps": briefSteps(), "publishMode": "draft", "brief": brief, "directives": directives,
		})
		if !strings.Contains(asked.Summary, "“"+brief.Goal+"”") || !strings.Contains(asked.Summary, "with a directive for 3 pages") {
			t.Errorf("the run's card says %q, want its goal and its directives on it", asked.Summary)
		}
		started := decoded[map[string]json.RawMessage](t, approveTheCard(t, c.core, conversation, "runs_control").Result)
		runID := decoded[string](t, started["runId"])
		awaitRun(t, c.core.Runs, runID)
		assertTheBriefReachedEveryStep(t, c, runID, brief, wanted)
	})

	t.Run("started from the run drawer", func(t *testing.T) {
		brief := run.Brief{Goal: "Answer the buyer who compares two peptides", Focus: "what sets each one apart",
			Tone: "plain and exact", Avoid: "medical promises"}
		ids, directives := c.directivesOf(t, wanted)
		started, err := c.core.Runs.Start(t.Context(), runs.StartRequest{
			SiteID: c.siteID, PageIDs: ids, PublishMode: string(run.PublishDraft), Recipe: briefRecipe(), Brief: &brief, Directives: directives,
		})
		if err != nil {
			t.Fatalf("start the run with its brief from the drawer: %v", err)
		}
		awaitRun(t, c.core.Runs, started.RunID)
		assertTheBriefReachedEveryStep(t, c, started.RunID, brief, wanted)
	})
}

func assertTheBriefReachedEveryStep(t *testing.T, c *capability, runID string, brief run.Brief, wanted []briefPage) {
	t.Helper()

	items := itemsByTarget(t, c.core.Runs, runID)
	for _, held := range wanted {
		page, target := c.page(t, held.path), c.page(t, held.target)
		item, found := items[page.ID]
		if !found || item.Status != string(run.StatusCompleted) {
			t.Fatalf("the run holds %+v for %s, want its completed item", item, held.path)
		}
		lines := []string{
			briefHeading, "Goal: " + brief.Goal, "Focus: " + brief.Focus, "Tone: " + brief.Tone, "Avoid: " + brief.Avoid,
			"For this page: " + held.notes, "Link to " + held.target + " with the anchor “" + held.anchor + "”",
		}
		for _, step := range []string{steps.NameGenerateBody, steps.NameJudge, steps.NameRepairLinks} {
			asked := requestsOf(c.script, runID, item.ID, step)
			if len(asked) == 0 {
				t.Errorf("the %s was never asked about %s in run %s", step, held.path, runID)
				continue
			}
			prompt := promptText(asked[0])
			for _, line := range lines {
				if !strings.Contains(prompt, line) {
					t.Errorf("the %s's request about %s lacks %q:\n%s", step, held.path, line, prompt)
				}
			}
		}
		plan := linkContextOf(t, c, item.ID)
		owed, found := plan.ByPageID(target.ID)
		if !found || !owed.Required || len(owed.Anchors) == 0 || owed.Anchors[0] != held.anchor {
			t.Errorf("the link plan of %s holds %+v for %s, want it required with the directive's anchor first", held.path, owed, held.target)
		}
		anchors := anchorsTo(t, c, item.ID, held.target)
		if len(anchors) == 0 {
			t.Errorf("the body written for %s does not link to %s, which its directive must link", held.path, held.target)
		}
		t.Logf("%s links to %s as %q; the directive asked for %q", held.path, held.target, anchors, held.anchor)
	}
}

func anchorsTo(t *testing.T, c *capability, itemID, target string) []string {
	t.Helper()

	doc, err := content.Parse(lastWritten(t, c, itemID, run.ArtifactBodyHTML))
	if err != nil {
		t.Fatalf("parse the body item %s wrote: %v", itemID, err)
	}
	out := make([]string, 0)
	for _, link := range doc.Links() {
		if strings.HasSuffix(link.Href, target) {
			out = append(out, strings.Join(strings.Fields(link.Anchor), " "))
		}
	}
	return out
}

func TestTheAssistantDesignsTheLinksOfASection(t *testing.T) {
	c := openCapability(t, fake.NewScripted(fake.Reply{Step: steps.NameGenerateBody, Make: bodyFromPrompt}), withTheTree)
	conversation := c.conversation(t)
	section, bpc, tb, recon := c.page(t, capabilityCategory), c.page(t, capabilityBPC), c.page(t, capabilityTB), c.page(t, capabilityRecon)

	totals := linksOfSection(t, c, conversation, map[string]any{"view": "section", "pageId": section.ID})
	mapped := linksOfSection(t, c, conversation, map[string]any{"view": "map", "pageId": section.ID})
	owed := linksOfSection(t, c, conversation, map[string]any{"view": "owed", "pageId": section.ID, "missing": true})
	suggested := linksOfSection(t, c, conversation, map[string]any{"view": "suggest", "pageId": section.ID})
	if totals.Totals == nil || totals.Totals.Pages == 0 || suggested.Suggestions == nil {
		t.Fatalf("the section %s reads totals %+v and suggestions %+v", section.Path, totals.Totals, suggested.Suggestions)
	}
	for _, held := range suggested.Suggestions.Items {
		if held.Kind == sitemodel.SuggestLink && (held.ToPath == "" || len(held.Anchors) == 0) {
			t.Errorf("the suggestion %+v names no target or anchor", held)
		}
	}
	t.Logf("the section %s: %+v, %d links on its map, %d owed links missing, %d suggestions", section.Path, *totals.Totals,
		len(mapped.Links.Items), len(owed.Owed.Items), len(suggested.Suggestions.Items))
	for _, gap := range []pages.Page{bpc, tb} {
		read := ask[tools.PagesGetResponse](t, c, conversation, "pages_get", map[string]any{"pageId": gap.ID})
		if slices.ContainsFunc(read.Page.Targets, func(target contextpack.Target) bool { return target.PageID == recon.ID }) {
			t.Fatalf("the graph already connects %s to %s; the test needs a link the graph lacks", gap.Path, capabilityRecon)
		}
	}

	designed := []briefPage{
		{path: capabilityBPC, notes: "Send the reader to the reconstitution guide before the dosing table.", target: capabilityRecon,
			anchor: "reconstituting BPC-157 step by step"},
		{path: capabilityTB, notes: "Send the reader to the reconstitution guide for TB-500's own solvent.", target: capabilityRecon,
			anchor: "how TB-500 is reconstituted"},
	}
	ids, directives := c.directivesOf(t, designed)
	plan := proposeAPlan(t, c, conversation, "Link the healing peptides to the reconstitution guide",
		planStep{tool: "graph_edit", args: map[string]any{
			"op": "add_edge", "kind": "related", "entityId": c.entities["BPC-157"], "toEntityId": c.entities["Reconstitution"],
			"weight": 0.9, "reason": "BPC-157 arrives as a powder the reader reconstitutes first.",
		}, reason: "The graph does not connect BPC-157 to the reconstitution guide, so no run may link them yet."},
		planStep{tool: "graph_edit", args: map[string]any{
			"op": "add_edge", "kind": "related", "entityId": c.entities["TB-500"], "toEntityId": c.entities["Reconstitution"],
			"weight": 0.9, "reason": "TB-500 arrives as a powder the reader reconstitutes first.",
		}, reason: "The graph does not connect TB-500 to the reconstitution guide, so no run may link them yet."},
		planStep{tool: "runs_control", args: map[string]any{
			"op": "start", "pageIds": ids, "steps": []string{steps.NameResolveContext, steps.NameGenerateBody, steps.NameInsertLinks},
			"publishMode": "draft", "brief": run.Brief{Goal: "Send every healing peptide's reader to the reconstitution guide"},
			"directives": directives,
		}, reason: "Write both pages with the links the section lacks, each with its anchor.", dependsOn: []int{1, 2}},
	)
	if plan.Steps[2].EstimateTokens == 0 {
		t.Errorf("the run step of the plan carries no estimate: %+v", plan.Steps[2])
	}
	approved := approveThePlan(t, c, conversation, plan, nil)
	if approved.Steps[0].Status != "done" || approved.Steps[1].Status != "done" || approved.Steps[2].RunID == "" {
		t.Fatalf("the approved plan stands at %+v, want both edges written and the run started", approved.Steps)
	}
	awaitRun(t, c.core.Runs, approved.Steps[2].RunID)
	finished := awaitPlan(t, c, conversation, plan.ID)
	assertEveryStep(t, finished, "done")
	toldThePlan(t, c, conversation, plan.Title)

	for _, name := range []string{"BPC-157", "TB-500"} {
		detail := entityRead(t, c, conversation, c.entities[name])
		if !slices.ContainsFunc(detail.Related, func(held sitemodel.EntityLink) bool {
			return held.EntityID == c.entities["Reconstitution"] && held.Status == "approved"
		}) {
			t.Errorf("after the plan %s relates to %+v, want Reconstitution, approved", name, detail.Related)
		}
	}
	items := itemsByTarget(t, c.core.Runs, approved.Steps[2].RunID)
	for _, held := range designed {
		item := items[c.page(t, held.path).ID]
		target, found := linkContextOf(t, c, item.ID).ByPageID(recon.ID)
		if !found || !target.Required || len(target.Anchors) == 0 || target.Anchors[0] != held.anchor {
			t.Errorf("the run's link plan for %s holds %+v, want %s required with %q first", held.path, target, capabilityRecon, held.anchor)
		}
		if anchors := anchorsTo(t, c, item.ID, capabilityRecon); !slices.Contains(anchors, held.anchor) {
			t.Errorf("the body written for %s links to %s as %q, want the must-link's anchor %q", held.path, capabilityRecon, anchors, held.anchor)
		}
	}
}
