package agent

import (
	"strings"
	"testing"

	agentapp "github.com/davidmovas/postulator/internal/application/agent"
	"github.com/davidmovas/postulator/internal/application/sitemodel"
	"github.com/davidmovas/postulator/internal/domain/contextpack"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
)

func TestTheInstructionsTeachTheModelsVocabulary(t *testing.T) {
	t.Parallel()

	instructions, err := prompts.One(instructionsPrompt, agentapp.SiteContext{Mode: "confirm"})
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	words := make([]string, 0, 32)
	words = append(words, "page type", "field", "scope", "rewrites", "trail", "twin", "orphan", "card", "finding", "pack", "pages_get",
		"FINDINGS", "op list", "op resolve", "option", "explain", "SWEEPS", "fields", "runs_start", "EDITING A PAGE", "pages_fields",
		"pages_blocks", "place_link", "bodyHash", "CONFIRMATION", "confirms", "runs_get", "runs_revert")
	for _, view := range []sitemodel.View{sitemodel.ViewTree, sitemodel.ViewLinks, sitemodel.ViewGaps, sitemodel.ViewPack} {
		words = append(words, "view "+string(view))
	}
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
