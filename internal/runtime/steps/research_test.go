package steps_test

import (
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/davidmovas/postulator/internal/adapters/llm/fake"
	port "github.com/davidmovas/postulator/internal/application/llm"
	"github.com/davidmovas/postulator/internal/domain/content"
	domainllm "github.com/davidmovas/postulator/internal/domain/llm"
	"github.com/davidmovas/postulator/internal/domain/run"
	"github.com/davidmovas/postulator/internal/domain/template"
	"github.com/davidmovas/postulator/internal/kernel/id"
	"github.com/davidmovas/postulator/internal/runtime/steps"
)

const researchedFact = "A double espresso takes 18 grams of ground coffee [sca.coffee]"

var searchSettings = port.WebSearch{Depth: domainllm.DepthHigh, MaxCalls: 3, Blocked: []string{"rival.example"}}

type searchStub struct{}

func (searchStub) WebSearch() port.WebSearch {
	return searchSettings
}

func TestAResearchedRunHandsTheFactsToTheWritersAndTheJudge(t *testing.T) {
	t.Parallel()

	p := newPipeline(t)
	p.llm = fake.NewScripted(
		fake.Reply{Step: steps.NameResearch, Text: "- " + researchedFact},
		fake.Reply{Step: steps.NameGenerateBody, Text: guideDraft},
		fake.Reply{Step: steps.NameGenerateMeta, Text: guideMeta},
		fake.Reply{Step: steps.NameJudge, Text: guideJudge},
		fake.Reply{Step: steps.NameRepairLinks, Text: repairSentence},
	)
	p.search = searchStub{}
	p.start(t)

	recipe := slices.Insert(slices.Clone(p.recipe), 1, template.StepSpec{
		Name: steps.NameResearch, Enabled: true, Params: map[string]any{steps.ParamFocus: "the brewing ratio and the water temperature"},
	})
	queued, err := p.engine.Enqueue(t.Context(), run.Run{
		ID: id.New(), SiteID: p.siteID, Kind: run.KindGenerate, Targets: []string{p.pageID},
		Recipe: recipe, TemplateID: "guide", TemplateVersion: 1, PublishMode: run.PublishDraft,
	})
	if err != nil {
		t.Fatalf("Enqueue: %v", err)
	}
	item := p.waitFor(t, queued.ID)

	var research content.Research
	p.artifactOf(t, item.ID, run.ArtifactResearch, &research)
	if !slices.Contains(research.Facts(), researchedFact) {
		t.Fatalf("the research kept %+v, want the fact the search found", research)
	}

	asked := p.requestTo(t, steps.NameResearch)
	if asked.WebSearch == nil || !reflect.DeepEqual(*asked.WebSearch, searchSettings) || asked.Schema != nil {
		t.Errorf("the research asked with search %+v and schema %v, want the settings' search and a plain answer", asked.WebSearch, asked.Schema)
	}
	if asked.Meta.Role != domainllm.RoleResearcher {
		t.Errorf("the research is booked as %q, want the researcher", asked.Meta.Role)
	}
	prompt := p.promptTo(t, steps.NameResearch)
	for _, want := range []string{"the brewing ratio and the water temperature", "Step-by-Step Instructions", "Espresso"} {
		if !strings.Contains(prompt, want) {
			t.Errorf("the research prompt lacks %q", want)
		}
	}

	for _, step := range []string{steps.NameGenerateBody, steps.NameGenerateMeta, steps.NameJudge} {
		if prompt := p.promptTo(t, step); !strings.Contains(prompt, researchedFact) {
			t.Errorf("the %s prompt lacks the researched fact", step)
		}
		if written := p.requestTo(t, step); written.WebSearch != nil {
			t.Errorf("the %s call searched the web itself", step)
		}
	}
	if p.llm.CallsTo(steps.NameRepairLinks) > 0 && strings.Contains(p.promptTo(t, steps.NameRepairLinks), researchedFact) {
		t.Error("the linker was handed the research it never reads")
	}
}

func (p *pipeline) requestTo(t *testing.T, step string) port.Request {
	t.Helper()

	requests := p.llm.Requests()
	for i := range requests {
		if requests[i].Meta.Step == step {
			return requests[i]
		}
	}
	t.Fatalf("the model was never asked for %s", step)
	return port.Request{}
}
