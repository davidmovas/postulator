package tools_test

import (
	"encoding/json"
	"strconv"
	"strings"
	"testing"

	"github.com/davidmovas/postulator/internal/application/sitemodel"
	"github.com/davidmovas/postulator/internal/application/tools"
	domainagent "github.com/davidmovas/postulator/internal/domain/agent"
	"github.com/davidmovas/postulator/internal/domain/contextpack"
	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/kernel/paging"
)

func TestThePageReadsAnswerOverTheStore(t *testing.T) {
	t.Parallel()

	registry, binding, seeded := wired(t)
	editable(t, seeded)
	binding.Mode = domainagent.ModeConfirm

	found := called[paging.List[contextpack.MapNode]](t, registry, binding, "pages_find", `{"query":"coffee"}`)
	if len(found.Items) != 1 || found.Items[0].Page.PageID != seeded.page {
		t.Errorf("pages_find coffee = %+v, want the seeded page", found.Items)
	}
	none := called[paging.List[contextpack.MapNode]](t, registry, binding, "pages_find", `{"status":"archived"}`)
	if len(none.Items) != 0 {
		t.Errorf("pages_find archived = %+v, want none", none.Items)
	}

	roots := called[sitemodel.TreeAnswer](t, registry, binding, "pages_tree", `{"view":"roots"}`)
	if len(roots.Nodes.Items) != 1 || roots.Nodes.Items[0].Page.Path != "/coffee/" || roots.Nodes.Items[0].Why == nil {
		t.Errorf("pages_tree roots = %+v, want the one page with why it has no parent", roots.Nodes.Items)
	}
	walked := called[sitemodel.TreeAnswer](t, registry, binding, "pages_tree",
		`{"view":"walk","generation":`+strconv.FormatInt(roots.Generation, 10)+`}`)
	if walked.Stale || len(walked.Nodes.Items) != 1 {
		t.Errorf("pages_tree walk = %+v, want the one page and a fresh generation", walked)
	}

	pack := called[tools.PagesGetResponse](t, registry, binding, "pages_get", `{"view":"pack","path":"/coffee/"}`)
	if pack.Pack == nil || pack.Pack.Budget != 16384/4 || !strings.Contains(pack.Pack.Text, "/coffee/") || pack.Page != nil {
		t.Errorf("pages_get pack = %+v, want the writer's pack fitted to a quarter of the tool-result cap in tokens", pack)
	}
	if _, err := registry.Call(t.Context(), binding, "pages_get", json.RawMessage(`{"view":"reports","pageId":"`+seeded.page+`"}`)); !errors.IsCode(err, errors.NotFound) ||
		!strings.Contains(err.Error(), "no run has touched this page") {
		t.Errorf("pages_get reports of a page no run touched = %v, want it said", err)
	}

	blocks := called[tools.PagesBlocksResponse](t, registry, binding, "pages_blocks", `{"op":"list","path":"/coffee/","text":true}`)
	if blocks.Blocks == nil || len(blocks.Blocks.Blocks.Items) != 2 || blocks.Blocks.Blocks.Items[1].Text != "Our paragraph on keeping beans fresh." {
		t.Errorf("pages_blocks text = %+v, want both blocks with their whole text", blocks.Blocks)
	}
}
