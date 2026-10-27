//go:build e2e

package e2e_test

import (
	"encoding/json"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/davidmovas/postulator/internal/adapters/llm/fake"
	"github.com/davidmovas/postulator/internal/application/findings"
	"github.com/davidmovas/postulator/internal/application/graph"
	"github.com/davidmovas/postulator/internal/application/pagetypes"
	"github.com/davidmovas/postulator/internal/application/sitemodel"
	"github.com/davidmovas/postulator/internal/application/tools"
	domainagent "github.com/davidmovas/postulator/internal/domain/agent"
	"github.com/davidmovas/postulator/internal/domain/finding"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/pagetype"
	"github.com/davidmovas/postulator/internal/domain/run"
	"github.com/davidmovas/postulator/internal/runtime/steps"
)

const (
	structureInstruction = "Lead with how each peptide is stored once it arrives."
	sampleRule           = "Capability sample pages"
	guidesRule           = "Capability lab guides"
	storagePath          = "/peptides/peptide-storage/"
	editedGoal           = "Show the lab manager how each peptide keeps until it is used"
	editedNotes          = "Open with the freezer temperature BPC-157 needs."
	selectGoal           = "Answer what a buyer asks before ordering a healing peptide"
	bpcDefinition        = "BPC-157 is a synthetic peptide of fifteen amino acids, studied for tissue repair."
	tbDefinition         = "TB-500 is a synthetic fragment of thymosin beta-4, studied for wound healing."
)

func (c *capability) entityOfPage(t *testing.T, path string) string {
	t.Helper()

	page := c.page(t, path)
	if page.EntityID == nil {
		t.Fatalf("%s carries no entity", path)
	}
	return *page.EntityID
}

func (c *capability) idsOfPaths(t *testing.T, paths ...string) []string {
	t.Helper()

	out := make([]string, 0, len(paths))
	for _, path := range paths {
		out = append(out, c.page(t, path).ID)
	}
	return out
}

func treeParentPaths() map[string]string {
	return map[string]string{
		capabilityHub: "", capabilityCategory: capabilityHub, capabilityBPC: capabilityCategory, capabilityTB: capabilityCategory,
		capabilityGuides: "", capabilityRecon: capabilityGuides,
	}
}

func TestTheAssistantPlansTheModel(t *testing.T) {
	c := openCapability(t, fake.NewScripted(), withoutAGraph)

	t.Run("12 entities, cards, related edges and parents from the tree", func(t *testing.T) { plansTheGraphFromTheTree(t, c) })
	t.Run("13 structure", func(t *testing.T) { plansTheStructure(t, c) })
	t.Run("15 resolutions for findings", func(t *testing.T) { plansResolutions(t, c) })
	t.Run("19 types, templates and rules in bulk", func(t *testing.T) { changesTypesAndTemplatesInBulk(t, c) })
}

func plansTheGraphFromTheTree(t *testing.T, c *capability) {
	conversation := c.conversation(t)
	parents := treeParentPaths()
	paths := make([]string, 0, len(parents))
	for path := range parents {
		paths = append(paths, path)
	}
	slices.Sort(paths)
	preview := ask[graph.PreviewFromPagesResponse](t, c, conversation, "graph_propose", map[string]any{
		"op": "preview", "source": "site", "pageIds": c.idsOfPaths(t, paths...),
	})
	if len(preview.Entities) != len(paths) {
		t.Fatalf("the site preview proposes %d entities for the %d pages: %+v", len(preview.Entities), len(paths), preview.Entities)
	}
	for i := range preview.Entities {
		proposed := &preview.Entities[i]
		if proposed.Clash != "" || proposed.Path == "" {
			t.Errorf("the preview proposes %+v, want it for its page without a clash", *proposed)
		}
		t.Logf("proposed %s for %s under %q", proposed.Name, proposed.Path, proposed.Parent)
	}

	plan := proposeAPlan(t, c, conversation, "Build the graph from the page tree", planStep{
		tool: "graph_edit", args: map[string]any{"op": "apply_proposals", "proposalsOf": "site", "proposals": preview.Entities},
		reason: "Each page of the two sections becomes an entity under the parent its page sits under.",
	})
	assertEveryStep(t, approveThePlan(t, c, conversation, plan, nil), domainagent.StepDone)
	c.refresh(t)
	for path, parent := range parents {
		detail := entityRead(t, c, conversation, c.entityOfPage(t, path))
		switch {
		case parent == "" && len(detail.Parents) != 0:
			t.Errorf("%s sits at the top of the tree and its entity has the parents %+v", path, detail.Parents)
		case parent != "" && (len(detail.Parents) != 1 || detail.Parents[0].EntityID != c.entityOfPage(t, parent) ||
			detail.Parents[0].Status != "approved"):
			t.Errorf("the entity of %s has the parents %+v, want the entity of %s, approved", path, detail.Parents, parent)
		}
		if held := openFindings(t, c, finding.PageWithoutEntity); slices.ContainsFunc(held, func(f findings.Finding) bool { return f.PagePath == path }) {
			t.Errorf("%s still carries page_without_entity after its proposal was written", path)
		}
	}

	bpc, tb := c.entityOfPage(t, capabilityBPC), c.entityOfPage(t, capabilityTB)
	cards := proposeAPlan(t, c, conversation, "Give the healing peptides their cards and relate them",
		planStep{tool: "graph_propose", args: map[string]any{"op": "cards", "cards": []map[string]any{
			{"entityId": bpc, "definition": bpcDefinition, "reason": "From the page's own text."},
			{"entityId": tb, "definition": tbDefinition, "reason": "From the page's own text."},
		}}, reason: "Both healing peptides have no card for the writer to read."},
		planStep{tool: "graph_edit", args: map[string]any{
			"op": "add_edge", "kind": "related", "entityId": bpc, "toEntityId": tb, "weight": 0.8,
			"reason": "Both are studied for tissue repair and bought together.",
		}, reason: "The two siblings should link to each other."},
	)
	assertEveryStep(t, approveThePlan(t, c, conversation, cards, nil), domainagent.StepDone)
	if held := entityRead(t, c, conversation, bpc); held.Definition != bpcDefinition ||
		!slices.ContainsFunc(held.Related, func(link sitemodel.EntityLink) bool { return link.EntityID == tb && link.Status == "approved" }) {
		t.Errorf("after the plan BPC-157 reads %q related to %+v, want its card and TB-500, approved", held.Definition, held.Related)
	}
	if held := entityRead(t, c, conversation, tb); held.Definition != tbDefinition {
		t.Errorf("after the plan TB-500's card is %q", held.Definition)
	}
}

func ruleNamed(t *testing.T, c *capability, name string) (pagetypes.Rule, bool) {
	t.Helper()

	listed, err := c.core.PageTypes.ListRules(t.Context(), pagetypes.ListRulesRequest{SiteID: c.siteID})
	if err != nil {
		t.Fatalf("list the classification rules: %v", err)
	}
	at := slices.IndexFunc(listed.Rules, func(rule pagetypes.Rule) bool { return rule.Name == name })
	if at < 0 {
		return pagetypes.Rule{}, false
	}
	return listed.Rules[at], true
}

func findingAt(t *testing.T, c *capability, findingID string) findings.Finding {
	t.Helper()

	held, err := c.core.Findings.Get(t.Context(), findings.GetRequest{SiteID: c.siteID, FindingID: findingID})
	if err != nil {
		t.Fatalf("read the finding %s: %v", findingID, err)
	}
	return held
}

func plansTheStructure(t *testing.T, c *capability) {
	conversation := c.conversation(t)
	types := typesOf(t, c.core, c.siteID)
	category, guide, info := types[pagetype.Category], types[pagetype.Guide], types[pagetype.Info]
	guideTemplate := templateNamed(t, c.core, "Guide")
	hello := c.page(t, "/hello-world/")

	plan := proposeAPlan(t, c, conversation, "Shape how the categories and the posts are written",
		planStep{tool: "pagetypes", args: map[string]any{
			"op": "scope", "pageTypeId": category.ID, "field": "body", "instruction": structureInstruction,
		}, reason: "Every category page should open with storage."},
		planStep{tool: "pagetypes", args: map[string]any{"op": "save", "pageTypeId": info.ID, "templateId": guideTemplate},
			reason: "An info page reads best with the guide's sections."},
		planStep{tool: "pagetypes_classify", args: map[string]any{
			"op": "save_rule", "name": sampleRule, "typeId": info.ID,
			"conditions": map[string]any{"urls": []map[string]string{{"mode": "prefix", "value": "/sample-page/"}}},
		}, reason: "The sample page is not a guide."},
		planStep{tool: "pages_manage", args: map[string]any{"op": "update", "pageId": hello.ID, "pageTypeId": guide.ID},
			reason: "The post is a guide to the shop."},
	)
	edits := plan.Steps[3].Edits
	if len(edits) != 1 || edits[0].Name != "type" || edits[0].Value != guide.ID ||
		!slices.ContainsFunc(edits[0].Choices, func(choice domainagent.EditChoice) bool { return choice.Value == info.ID }) {
		t.Fatalf("the retyping step offers the edits %+v, want the type picked from the site's types", edits)
	}
	assertEveryStep(t, approveThePlan(t, c, conversation, plan, map[int]map[string]string{4: {"type": info.ID}}), domainagent.StepDone)

	after := typesOf(t, c.core, c.siteID)
	if scope, held := scopeIn(after[pagetype.Category], "body"); !held || scope.Instruction != structureInstruction {
		t.Errorf("the category type's body scope is %+v, want the plan's instruction", scope)
	}
	if held := after[pagetype.Info].TemplateID; held == nil || *held != guideTemplate {
		t.Errorf("the info type carries the template %v, want the guide's", held)
	}
	if rule, held := ruleNamed(t, c, sampleRule); !held || rule.TypeID != info.ID || !rule.Enabled || len(rule.Conditions.URLs) != 1 {
		t.Errorf("the rule %q after the plan is %+v, want it typing the sample page Info", sampleRule, rule)
	}
	c.refresh(t)
	if retyped := c.page(t, "/hello-world/"); retyped.PageTypeID == nil || *retyped.PageTypeID != info.ID ||
		retyped.TypeSource != string(pagemap.TypeSourceManual) {
		t.Errorf("/hello-world/ is typed %v (%s), want the type the person picked on the card, by hand", retyped.PageTypeID, retyped.TypeSource)
	}
}

func plansResolutions(t *testing.T, c *capability) {
	conversation := c.conversation(t)
	open := openFindings(t, c, finding.PageWithoutEntity)
	if len(open) < 2 {
		t.Fatalf("the replica holds %d pages without an entity after the graph was built, want two to settle", len(open))
	}
	settled := open[:2]
	stepsOf := make([]planStep, 0, len(settled))
	for i := range settled {
		stepsOf = append(stepsOf, planStep{tool: "findings", args: map[string]any{
			"op": "resolve", "findingId": settled[i].ID, "option": string(finding.Ignore),
		}, reason: settled[i].PagePath + " needs no entity of its own."})
	}
	plan := proposeAPlan(t, c, conversation, "Settle the pages that need no entity", stepsOf...)
	for i := range plan.Steps {
		if said := plan.Steps[i].Summary; !strings.Contains(said, "nothing is written to the site") {
			t.Errorf("the resolution step says %q, want what it changes and that the site is untouched", said)
		}
	}
	assertEveryStep(t, approveThePlan(t, c, conversation, plan, nil), domainagent.StepDone)
	for i := range settled {
		if held := findingAt(t, c, settled[i].ID); held.Status == string(finding.Open) || held.ResolvedBy != finding.ResolvedByAgent {
			t.Errorf("the finding on %s after the plan is %+v, want it settled by the agent", settled[i].PagePath, held)
		}
	}
	if still := openFindings(t, c, finding.PageWithoutEntity); len(still) != len(open)-len(settled) {
		t.Errorf("after the plan %d page_without_entity findings are open, want %d", len(still), len(open)-len(settled))
	}
}

func changesTypesAndTemplatesInBulk(t *testing.T, c *capability) {
	conversation := c.conversation(t)
	types := typesOf(t, c.core, c.siteID)
	info, guide := types[pagetype.Info], types[pagetype.Guide]
	hubTemplate := templateNamed(t, c.core, "Hub")
	guides, bpc, tb := c.page(t, capabilityGuides), c.page(t, capabilityBPC), c.page(t, capabilityTB)

	plan := proposeAPlan(t, c, conversation, "Retype the lab guides and re-template the healing peptides",
		planStep{tool: "pages_manage", args: map[string]any{
			"op": "set_types", "select": map[string]any{"under": []string{guides.ID}}, "pageTypeId": info.ID,
		}, reason: "The lab guides are reference pages, not articles."},
		planStep{tool: "pages_manage", args: map[string]any{
			"op": "set_templates", "select": map[string]any{"pageIds": []string{bpc.ID, tb.ID}}, "templateId": hubTemplate,
		}, reason: "Both peptides list their products like a hub."},
		planStep{tool: "pagetypes_classify", args: map[string]any{
			"op": "save_rule", "name": guidesRule, "typeId": guide.ID,
			"conditions": map[string]any{"urls": []map[string]string{{"mode": "prefix", "value": capabilityGuides}}},
		}, reason: "A page under the lab guides is a guide."},
	)
	if !strings.Contains(plan.Steps[0].Summary, "2 pages") || !strings.Contains(plan.Steps[0].Summary, capabilityRecon) {
		t.Errorf("the retyping step says %q, want the two pages of the lab guides", plan.Steps[0].Summary)
	}
	assertEveryStep(t, approveThePlan(t, c, conversation, plan, nil), domainagent.StepDone)
	c.refresh(t)
	for _, path := range []string{capabilityGuides, capabilityRecon} {
		if page := c.page(t, path); page.PageTypeID == nil || *page.PageTypeID != info.ID || page.TypeSource != string(pagemap.TypeSourceManual) {
			t.Errorf("%s is typed %v (%s) after the bulk retype, want info by hand", path, page.PageTypeID, page.TypeSource)
		}
	}
	for _, path := range []string{capabilityBPC, capabilityTB} {
		if page := c.page(t, path); page.TemplateID == nil || *page.TemplateID != hubTemplate {
			t.Errorf("%s carries the template %v after the bulk re-template, want the hub's", path, page.TemplateID)
		}
	}
	if rule, held := ruleNamed(t, c, guidesRule); !held || rule.TypeID != guide.ID || !rule.Enabled {
		t.Errorf("the rule %q after the plan is %+v, want it typing the lab guides Guide", guidesRule, rule)
	}

	reset := proposeAPlan(t, c, conversation, "Hand the lab guides back to the rules",
		planStep{tool: "pages_manage", args: map[string]any{
			"op": "set_types", "select": map[string]any{"under": []string{guides.ID}}, "reset": true,
		}, reason: "The rules type them now."},
		planStep{tool: "pages_manage", args: map[string]any{
			"op": "set_templates", "select": map[string]any{"pageIds": []string{bpc.ID, tb.ID}}, "reset": true,
		}, reason: "Their type's template fits them again."},
	)
	assertEveryStep(t, approveThePlan(t, c, conversation, reset, nil), domainagent.StepDone)
	c.refresh(t)
	for _, path := range []string{capabilityGuides, capabilityRecon} {
		if page := c.page(t, path); page.TypeSource != string(pagemap.TypeSourceAuto) || page.PageTypeID == nil || *page.PageTypeID != guide.ID {
			t.Errorf("%s is typed %v (%s) after it was handed back, want the rules' guide", path, page.PageTypeID, page.TypeSource)
		}
	}
	for _, path := range []string{capabilityBPC, capabilityTB} {
		if page := c.page(t, path); page.TemplateID != nil && *page.TemplateID == hubTemplate {
			t.Errorf("%s keeps the hub template after it was handed back", path)
		}
	}
}

func TestTheAssistantPlansContent(t *testing.T) {
	c := openCapability(t, fake.NewScripted(fake.Reply{Step: steps.NameGenerateBody, Make: bodyFromPrompt}), withTheTree)

	t.Run("14 a content plan", func(t *testing.T) { plansContent(t, c) })
	t.Run("18 a run over a selection with a brief", func(t *testing.T) { startsARunOverASelection(t, c) })
}

func plansContent(t *testing.T, c *capability) {
	conversation := c.conversation(t)
	bpc, tb := c.page(t, capabilityBPC), c.page(t, capabilityTB)
	brief := run.Brief{Goal: "Show the lab manager how each peptide is kept", Focus: "storage", Tone: "calm", Avoid: "dosage claims"}
	plan := proposeAPlan(t, c, conversation, "Write the healing peptides afresh and plan their storage page",
		planStep{tool: "pages_manage", args: map[string]any{"op": "create", "path": storagePath, "title": "Peptide Storage"},
			reason: "The section has no page on storage, which every peptide page should send the reader to."},
		planStep{tool: "runs_control", args: map[string]any{
			"op": "start", "pageIds": []string{bpc.ID, tb.ID}, "steps": []string{steps.NameResolveContext, steps.NameGenerateBody},
			"publishMode": "draft", "brief": brief, "directives": []run.Directive{
				{PageID: bpc.ID, Notes: "Open with how BPC-157 is stored."}, {PageID: tb.ID, Notes: "Open with how TB-500 is stored."},
			},
		}, reason: "Both bodies say nothing of storage."},
	)
	written := plan.Steps[1]
	if written.EstimateUSD <= 0 || written.EstimateTokens <= 0 || !strings.Contains(written.Summary, "estimated $") {
		t.Errorf("the run step carries the estimate %v USD, %d tokens and says %q", written.EstimateUSD, written.EstimateTokens, written.Summary)
	}
	names := make([]string, 0, len(written.Edits))
	for _, edit := range written.Edits {
		names = append(names, edit.Name)
	}
	for _, want := range []string{"goal", "focus", "tone", "avoid", "notes for " + capabilityBPC, "notes for " + capabilityTB} {
		if !slices.Contains(names, want) {
			t.Errorf("the run step offers the edits %v, want %q among them", names, want)
		}
	}
	approved := approveThePlan(t, c, conversation, plan, map[int]map[string]string{
		2: {"goal": editedGoal, "notes for " + capabilityBPC: editedNotes},
	})
	if approved.Steps[0].Status != domainagent.StepDone || approved.Steps[1].RunID == "" ||
		!strings.Contains(approved.Steps[1].Summary, editedGoal) {
		t.Fatalf("the approved content plan stands at %+v, want the page planned and the run started with the edited goal", approved.Steps)
	}
	runID := approved.Steps[1].RunID
	awaitRun(t, c.core.Runs, runID)
	awaitPlan(t, c, conversation, plan.ID)
	items := itemsByTarget(t, c.core.Runs, runID)
	for path, notes := range map[string]string{capabilityBPC: editedNotes, capabilityTB: "Open with how TB-500 is stored."} {
		asked := requestsOf(c.script, runID, items[c.page(t, path).ID].ID, steps.NameGenerateBody)
		if len(asked) == 0 {
			t.Fatalf("the writer was never asked about %s", path)
		}
		prompt := promptText(asked[0])
		for _, line := range []string{"Goal: " + editedGoal, "Focus: storage", "For this page: " + notes} {
			if !strings.Contains(prompt, line) {
				t.Errorf("the writer's request about %s lacks %q", path, line)
			}
		}
	}
	c.refresh(t)
	if planned := c.page(t, storagePath); planned.Status != string(pagemap.StatusPlanned) || planned.WPID != nil {
		t.Errorf("%s after the plan is %+v, want it planned in the map and not on the site", storagePath, planned)
	}
}

func startsARunOverASelection(t *testing.T, c *capability) {
	conversation := c.conversation(t)
	category := c.page(t, capabilityCategory)
	categoryType := typeIDOf(t, c, category)
	subtree := treeOf(t, c, conversation, map[string]any{"view": "subtree", "pageId": category.ID, "depth": 32, "limit": 200})
	want := []string{category.ID}
	for i := range subtree.Nodes.Items {
		if page := c.pages[subtree.Nodes.Items[i].Page.Path]; page.PageTypeID != nil && *page.PageTypeID == categoryType {
			want = append(want, page.ID)
		}
	}
	slices.Sort(want)

	asked := askACard(t, c, conversation, "runs_control", map[string]any{
		"op": "start", "select": map[string]any{"under": []string{category.ID}, "ofType": categoryType},
		"steps": []string{steps.NameResolveContext, steps.NameGenerateBody}, "publishMode": "draft",
		"brief": map[string]string{"goal": selectGoal},
	})
	if !strings.Contains(asked.Summary, "over "+strconv.Itoa(len(want))+" pages") || !strings.Contains(asked.Summary, "“"+selectGoal+"”") {
		t.Errorf("the run's card says %q, want the %d pages of the selection and the goal", asked.Summary, len(want))
	}
	started := decoded[map[string]json.RawMessage](t, approveTheCard(t, c.core, conversation, "runs_control").Result)
	runID := decoded[string](t, started["runId"])
	awaitRun(t, c.core.Runs, runID)
	items := itemsByTarget(t, c.core.Runs, runID)
	got := make([]string, 0, len(items))
	for target := range items {
		got = append(got, target)
	}
	slices.Sort(got)
	if !slices.Equal(got, want) {
		t.Errorf("the run over the selection writes %v, want the category-typed pages of %s's subtree %v", got, capabilityCategory, want)
	}
	for target := range items {
		asked := requestsOf(c.script, runID, items[target].ID, steps.NameGenerateBody)
		if len(asked) == 0 || !strings.Contains(promptText(asked[0]), "Goal: "+selectGoal) {
			t.Errorf("the writer's request about %s does not carry the run's goal", target)
		}
	}
}

func TestTheAssistantEditsTheGraphInBulk(t *testing.T) {
	c := openCapability(t, fake.NewScripted(), withAFlatGraph)

	t.Run("21 findings resolved in bulk", func(t *testing.T) { resolvesFindingsInBulk(t, c) })
	t.Run("20 the graph edited in bulk", func(t *testing.T) { editsTheGraphInBulk(t, c) })
}

func resolvesFindingsInBulk(t *testing.T, c *capability) {
	conversation := c.conversation(t)
	open := openFindings(t, c, finding.PageWithoutEntity)
	if len(open) < 3 {
		t.Fatalf("the replica holds %d pages without an entity, want three to settle at once", len(open))
	}
	settled := open[:3]
	asked := askACard(t, c, conversation, "findings", map[string]any{
		"op": "resolve_many", "code": string(finding.PageWithoutEntity), "option": string(finding.Ignore), "findingIds": idsOf(settled),
	})
	t.Logf("the bulk resolution's card: %s", asked.Summary)
	for i := range settled {
		if held := findingAt(t, c, settled[i].ID); held.Status != string(finding.Open) {
			t.Fatalf("the finding on %s was settled before the person approved: %+v", settled[i].PagePath, held)
		}
	}
	approved := decoded[tools.FindingsResponse](t, approveTheCard(t, c.core, conversation, "findings").Result)
	if approved.ResolveMany == nil || len(approved.ResolveMany.Findings) != len(settled) {
		t.Fatalf("resolve_many answered %+v, want the %d findings settled", approved.ResolveMany, len(settled))
	}
	for i := range settled {
		if held := findingAt(t, c, settled[i].ID); held.Status == string(finding.Open) || held.ResolvedBy != finding.ResolvedByAgent {
			t.Errorf("the finding on %s after the bulk resolution is %+v, want it settled by the agent", settled[i].PagePath, held)
		}
	}
	if still := openFindings(t, c, finding.PageWithoutEntity); len(still) != len(open)-len(settled) {
		t.Errorf("after the bulk resolution %d page_without_entity findings are open, want %d", len(still), len(open)-len(settled))
	}
}

func editsTheGraphInBulk(t *testing.T, c *capability) {
	conversation := c.conversation(t)
	want := treeParents(writerStandOf(t, c.live.env).graph, c.entities)
	adoption := ask[tools.GraphReadResponse](t, c, conversation, "graph_read", map[string]any{"view": "adoption"}).Adoption
	if adoption == nil || adoption.Adopting == 0 {
		t.Fatalf("the flat graph has nothing to adopt: %+v", adoption)
	}
	bpc, tb, recon := c.entities["BPC-157"], c.entities["TB-500"], c.entities["Reconstitution"]
	plan := proposeAPlan(t, c, conversation, "Give the graph the tree's parents and relate the healing peptides",
		planStep{tool: "graph_edit", args: map[string]any{"op": "adopt_parents", "hash": adoption.Hash},
			reason: "The graph is flat and the page tree shows where every entity sits."},
		planStep{tool: "graph_edit", args: map[string]any{
			"op": "add_edge", "kind": "related", "entityId": bpc, "toEntityId": tb, "weight": 0.8, "reason": "Studied together for repair.",
		}, reason: "The two siblings should link to each other.", dependsOn: []int{1}},
		planStep{tool: "graph_edit", args: map[string]any{
			"op": "add_edge", "kind": "related", "entityId": tb, "toEntityId": recon, "weight": 0.7, "status": "proposed",
			"reason": "TB-500 is reconstituted before use.",
		}, reason: "The reconstitution guide may help a TB-500 reader; the person decides.", dependsOn: []int{1}},
	)
	assertEveryStep(t, approveThePlan(t, c, conversation, plan, nil), domainagent.StepDone)
	for entityID, parentID := range want {
		detail := entityRead(t, c, conversation, entityID)
		got := ""
		if len(detail.Parents) == 1 && detail.Parents[0].Status == "approved" {
			got = detail.Parents[0].EntityID
		}
		if got != parentID {
			t.Errorf("after the plan %s has the parents %+v, the tree's is %q", detail.Name, detail.Parents, parentID)
		}
	}
	related := entityRead(t, c, conversation, tb).Related
	approvedEdge := slices.IndexFunc(related, func(link sitemodel.EntityLink) bool { return link.EntityID == bpc && link.Status == "approved" })
	proposedEdge := slices.IndexFunc(related, func(link sitemodel.EntityLink) bool { return link.EntityID == recon && link.Status == "proposed" })
	if approvedEdge < 0 || proposedEdge < 0 {
		t.Fatalf("after the plan TB-500 relates to %+v, want BPC-157 approved and Reconstitution proposed", related)
	}

	settle := proposeAPlan(t, c, conversation, "Approve the reconstitution link",
		planStep{tool: "graph_edit", args: map[string]any{"op": "approve_edge", "edgeId": related[proposedEdge].EdgeID},
			reason: "The person agreed the guide helps."},
	)
	assertEveryStep(t, approveThePlan(t, c, conversation, settle, nil), domainagent.StepDone)
	if after := entityRead(t, c, conversation, tb).Related; !slices.ContainsFunc(after, func(link sitemodel.EntityLink) bool {
		return link.EntityID == recon && link.Status == "approved"
	}) {
		t.Errorf("after the second plan TB-500 relates to %+v, want Reconstitution approved", after)
	}
	if open := openFindings(t, c, finding.EntityParentDiffers); len(open) != 0 {
		t.Errorf("after the adoption the findings %+v are still open", open)
	}
}
