package sitemodel_test

import (
	"cmp"
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/davidmovas/postulator/internal/application/sitemodel"
	"github.com/davidmovas/postulator/internal/application/sitemodel/sitemodeltest"
	"github.com/davidmovas/postulator/internal/domain/block"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/render"
)

var updatePacks = flag.Bool("update-packs", false, "rewrite the stand pack goldens from the captured models")

const tightBudget = 3000

func stand(t *testing.T, name string) *sitemodeltest.Memory {
	t.Helper()

	m, err := sitemodeltest.Stand(name)
	if err != nil {
		t.Fatalf("load the %s capture: %v", name, err)
	}
	return m
}

func packsOf(t *testing.T, m *sitemodeltest.Memory) string {
	t.Helper()

	pages := slices.Clone(m.Pages)
	pages = slices.DeleteFunc(pages, func(page pagemap.Page) bool { return page.Status == pagemap.StatusArchived })
	slices.SortFunc(pages, func(a, b pagemap.Page) int { return cmp.Compare(a.Path, b.Path) })
	var out strings.Builder
	for i := range pages {
		whole, err := m.Service().Picture(t.Context(), sitemodel.PackRequest{PageID: pages[i].ID})
		if err != nil {
			t.Fatalf("the pack of %s: %v", pages[i].Path, err)
		}
		out.WriteString("=== " + pages[i].Path + " · " + whole.Facts.Page.Type + " · " + string(whole.Fitted.Field) + " · " +
			strconv.Itoa(whole.Pack.Tokens) + " tokens\n")
		out.WriteString(whole.Fitted.Text())
		out.WriteString(dropsAt(t, m, pages[i].ID, whole.Pack.Tokens))
		out.WriteString("\n")
	}
	return out.String()
}

func dropsAt(t *testing.T, m *sitemodeltest.Memory, pageID string, built int) string {
	t.Helper()

	if built <= tightBudget {
		return "--- fits in " + strconv.Itoa(tightBudget) + " tokens\n"
	}
	tight, err := m.Service().Picture(t.Context(), sitemodel.PackRequest{PageID: pageID, Budget: tightBudget})
	if err != nil {
		return "--- " + strconv.Itoa(tightBudget) + " tokens are too few: " + err.Error() + "\n"
	}
	said := make([]string, 0, len(tight.Fitted.Dropped))
	for _, drop := range tight.Fitted.Dropped {
		if drop.Whole {
			said = append(said, string(drop.Section))
			continue
		}
		said = append(said, string(drop.Section)+" ("+strconv.Itoa(drop.Items)+" items)")
	}
	return "--- in " + strconv.Itoa(tightBudget) + " tokens Fit drops " + strings.Join(said, ", ") + "\n"
}

func TestThePackOfEveryStandPage(t *testing.T) {
	t.Parallel()

	for _, name := range sitemodeltest.Stands() {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := packsOf(t, stand(t, name))
			golden := filepath.Join("testdata", name, "packs.golden")
			if *updatePacks {
				if err := os.WriteFile(golden, []byte(got), 0o644); err != nil {
					t.Fatalf("write %s: %v", golden, err)
				}
				return
			}
			want, err := os.ReadFile(golden)
			if err != nil {
				t.Fatalf("read %s: %v; run this test with -update-packs once the capture is reviewed", golden, err)
			}
			if got != string(want) {
				t.Errorf("the %s packs moved from %s; review the difference and rerun with -update-packs", name, golden)
			}
		})
	}
}

const toolResultCap = 16384

func TestEveryPageModelOfBothStandsFitsATurnsToolResult(t *testing.T) {
	t.Parallel()

	for _, name := range sitemodeltest.Stands() {
		m := stand(t, name)
		widest, at := 0, ""
		for i := range m.Pages {
			page := &m.Pages[i]
			if page.Status == pagemap.StatusArchived {
				continue
			}
			answered, err := m.Service().Page(t.Context(), sitemodel.PageRequest{SiteID: m.Site.ID, PageID: page.ID})
			if err != nil {
				t.Fatalf("the %s's page model of %s: %v", name, page.Path, err)
			}
			encoded, err := json.Marshal(answered)
			if err != nil {
				t.Fatalf("encode: %v", err)
			}
			if len(encoded) > widest {
				widest, at = len(encoded), page.Path
			}
			if len(encoded) > toolResultCap {
				t.Errorf("the %s's page model of %s is %d bytes, over a turn's %d-byte tool result", name, page.Path, len(encoded), toolResultCap)
			}
		}
		t.Logf("the %s's widest page model is %s at %d bytes", name, at, widest)
	}
}

func TestAStandCaptureLoadsBackIntoTheModel(t *testing.T) {
	t.Parallel()

	m := site1()
	m.Renders = []render.Record{{PageID: "hub", ExtractedAt: stamp, Extraction: render.Extraction{
		Root: render.RootEmptyRaw, WordsGenerated: 48,
		Regions: []render.Region{{Kind: render.RegionListing, Inside: true, Links: 1, Signal: render.SignalJSONLD}},
	}}}
	m.Written = []block.Written{{PageID: "bpc", BlockHash: "h1", RunID: "run-1", WrittenAt: stamp}}
	raw, err := json.Marshal(sitemodeltest.Capture{
		Site: m.Site, Pages: m.Pages, Entities: m.Entities, Edges: m.Edges, Types: presetTypes(), Scopes: presetScopes(),
		Terms: m.Terms, Fields: m.Fields, Bodies: m.Bodies, Written: m.Written, Renders: m.Renders, Links: m.Links,
		Findings: m.Findings,
	})
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	loaded, err := sitemodeltest.Decode(raw)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if want, got := packsOf(t, m), packsOf(t, loaded); got != want {
		t.Errorf("the packs of a loaded capture =\n%s\nwant\n%s", got, want)
	}
}
