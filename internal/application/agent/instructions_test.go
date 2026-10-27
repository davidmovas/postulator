package agent_test

import (
	"strings"
	"testing"

	"github.com/tiktoken-go/tokenizer"

	"github.com/davidmovas/postulator/internal/application/agent"
	"github.com/davidmovas/postulator/internal/application/sitemodel/sitemodeltest"
	domainagent "github.com/davidmovas/postulator/internal/domain/agent"
	"github.com/davidmovas/postulator/internal/domain/contextpack"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
)

const instructionsCeilingTokens = 1961

func TestTheInstructionsAndTheReplicasBriefFitTheirCeiling(t *testing.T) {
	t.Parallel()

	replica, err := sitemodeltest.Stand("replica")
	if err != nil {
		t.Fatalf("load the replica's capture: %v", err)
	}
	brief, err := replica.Service().Brief(t.Context(), replica.Site.ID)
	if err != nil {
		t.Fatalf("the replica's brief: %v", err)
	}
	codec, err := tokenizer.Get(tokenizer.O200kBase)
	if err != nil {
		t.Fatalf("the o200k_base encoding: %v", err)
	}

	widest, at := 0, ""
	for _, mode := range []domainagent.Mode{domainagent.ModeConfirm, domainagent.ModeAutonomous} {
		instructions, err := agent.Instructions(agent.SiteContext{
			SiteID: replica.Site.ID, SiteName: replica.Site.Name, Mode: string(mode), Brief: brief.Text(),
		})
		if err != nil {
			t.Fatalf("render the %s instructions: %v", mode, err)
		}
		tokens, err := codec.Count(instructions)
		if err != nil {
			t.Fatalf("count the %s instructions: %v", mode, err)
		}
		t.Logf("the %s instructions with the replica's brief: %d bytes, %d o200k_base tokens; the brief alone %d bytes, "+
			"without link totals, which a capture holds no audit for", mode, len(instructions), tokens, len(brief.Text()))
		if tokens > widest {
			widest, at = tokens, string(mode)
		}
	}
	if widest > instructionsCeilingTokens {
		t.Errorf("the %s instructions with the replica's brief grew to %d o200k_base tokens, over the %d this build allows; "+
			"every round of every turn resends them", at, widest, instructionsCeilingTokens)
	}
}

func TestTheInstructionsTeachTheModelsVocabulary(t *testing.T) {
	t.Parallel()

	instructions, err := agent.Instructions(agent.SiteContext{Mode: "confirm"})
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	words := make([]string, 0, 32)
	words = append(words, "page type", "field", "scope", "rewrites", "trail", "twin", "orphan", "card", "finding", "pack", "pages_get",
		"FINDINGS", "op list", "op resolve", "option", "explain", "SWEEPS", "fields", "runs_start", "EDITING A PAGE", "pages_fields",
		"pages_blocks", "place_link", "bodyHash", "CONFIRMATION", "confirms", "runs_get", "runs_revert",
		"pages_tree", "pages_find", "links", "view pack")
	for _, gap := range contextpack.Gaps() {
		words = append(words, string(gap))
	}
	for _, kind := range []pagemap.LinkType{pagemap.LinkBreadcrumb, pagemap.LinkListing, pagemap.LinkNavigation, pagemap.LinkContextual, pagemap.LinkUnknown} {
		words = append(words, string(kind))
	}
	for _, word := range words {
		if !strings.Contains(instructions, word) {
			t.Errorf("the instructions never say %q", word)
		}
	}
}
