package sitemodel_test

import (
	"cmp"
	"encoding/json"
	stderrors "errors"
	"flag"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/davidmovas/postulator/internal/application/sitemodel"
	"github.com/davidmovas/postulator/internal/domain/block"
	"github.com/davidmovas/postulator/internal/domain/field"
	"github.com/davidmovas/postulator/internal/domain/finding"
	"github.com/davidmovas/postulator/internal/domain/graph"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/pagetype"
	"github.com/davidmovas/postulator/internal/domain/render"
	"github.com/davidmovas/postulator/internal/domain/site"
)

var updatePacks = flag.Bool("update-packs", false, "rewrite the stand pack goldens from the captured models")

const tightBudget = 3000

type capture struct {
	Site     site.Site                 `json:"site"`
	Pages    []pagemap.Page            `json:"pages"`
	Entities []graph.Entity            `json:"entities"`
	Edges    []graph.Edge              `json:"edges"`
	Types    []pagetype.Type           `json:"types"`
	Scopes   []field.Scope             `json:"scopes"`
	Terms    map[string][]pagemap.Term `json:"terms"`
	Fields   map[string][]field.Value  `json:"fields"`
	Bodies   []block.Snapshot          `json:"bodies"`
	Written  []block.Written           `json:"written"`
	Renders  []render.Record           `json:"renders"`
	Links    []pagemap.PageLink        `json:"links"`
	Findings []finding.Finding         `json:"findings"`
}

func captured(t *testing.T, raw []byte) *memory {
	t.Helper()

	var held capture
	if err := json.Unmarshal(raw, &held); err != nil {
		t.Fatalf("decode the capture: %v", err)
	}
	return &memory{
		site: held.Site, pages: held.Pages, entities: held.Entities, edges: held.Edges,
		types: pagetype.NewSet(held.Types, held.Scopes), terms: held.Terms, fields: held.Fields, bodies: held.Bodies,
		written: held.Written, renders: held.Renders, links: held.Links, findings: held.Findings,
	}
}

func packsOf(t *testing.T, m *memory) string {
	t.Helper()

	pages := slices.Clone(m.pages)
	pages = slices.DeleteFunc(pages, func(page pagemap.Page) bool { return page.Status == pagemap.StatusArchived })
	slices.SortFunc(pages, func(a, b pagemap.Page) int { return cmp.Compare(a.Path, b.Path) })
	var out strings.Builder
	for i := range pages {
		whole, err := m.service().Picture(t.Context(), sitemodel.PackRequest{PageID: pages[i].ID})
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

func dropsAt(t *testing.T, m *memory, pageID string, built int) string {
	t.Helper()

	if built <= tightBudget {
		return "--- fits in " + strconv.Itoa(tightBudget) + " tokens\n"
	}
	tight, err := m.service().Picture(t.Context(), sitemodel.PackRequest{PageID: pageID, Budget: tightBudget})
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

	for _, stand := range []string{"vanilla", "replica"} {
		t.Run(stand, func(t *testing.T) {
			t.Parallel()

			dir := filepath.Join("testdata", stand)
			raw, err := os.ReadFile(filepath.Join(dir, "site.json"))
			if stderrors.Is(err, fs.ErrNotExist) {
				t.Skipf("the %s stand is not captured: on a fresh stand run E2E_CAPTURE=1 task e2e:full STAND=%s -- -run "+
					"TestTheStandModelIsCaptured, then this test with -update-packs", stand, stand)
			}
			if err != nil {
				t.Fatalf("read the capture: %v", err)
			}
			got := packsOf(t, captured(t, raw))
			golden := filepath.Join(dir, "packs.golden")
			if *updatePacks {
				if err = os.WriteFile(golden, []byte(got), 0o644); err != nil {
					t.Fatalf("write %s: %v", golden, err)
				}
				return
			}
			want, err := os.ReadFile(golden)
			if err != nil {
				t.Fatalf("read %s: %v; run this test with -update-packs once the capture is reviewed", golden, err)
			}
			if got != string(want) {
				t.Errorf("the %s packs moved from %s; review the difference and rerun with -update-packs", stand, golden)
			}
		})
	}
}

const toolResultCap = 16384

func TestEveryPageModelOfBothStandsFitsATurnsToolResult(t *testing.T) {
	t.Parallel()

	for _, stand := range []string{"replica", "vanilla"} {
		raw, err := os.ReadFile(filepath.Join("testdata", stand, "site.json"))
		if err != nil {
			t.Fatalf("read the %s's capture: %v", stand, err)
		}
		m := captured(t, raw)
		widest, at := 0, ""
		for i := range m.pages {
			page := &m.pages[i]
			if page.Status == pagemap.StatusArchived {
				continue
			}
			answered, err := m.service().Page(t.Context(), sitemodel.PageRequest{SiteID: m.site.ID, PageID: page.ID})
			if err != nil {
				t.Fatalf("the %s's page model of %s: %v", stand, page.Path, err)
			}
			encoded, err := json.Marshal(answered)
			if err != nil {
				t.Fatalf("encode: %v", err)
			}
			if len(encoded) > widest {
				widest, at = len(encoded), page.Path
			}
			if len(encoded) > toolResultCap {
				t.Errorf("the %s's page model of %s is %d bytes, over a turn's %d-byte tool result", stand, page.Path, len(encoded), toolResultCap)
			}
		}
		t.Logf("the %s's widest page model is %s at %d bytes", stand, at, widest)
	}
}

func TestAStandCaptureLoadsBackIntoTheModel(t *testing.T) {
	t.Parallel()

	m := site1()
	m.renders = []render.Record{{PageID: "hub", ExtractedAt: stamp, Extraction: render.Extraction{
		Root: render.RootEmptyRaw, WordsGenerated: 48,
		Regions: []render.Region{{Kind: render.RegionListing, Inside: true, Links: 1, Signal: render.SignalJSONLD}},
	}}}
	m.written = []block.Written{{PageID: "bpc", BlockHash: "h1", RunID: "run-1", WrittenAt: stamp}}
	raw, err := json.Marshal(capture{
		Site: m.site, Pages: m.pages, Entities: m.entities, Edges: m.edges, Types: presetTypes(), Scopes: presetScopes(),
		Terms: m.terms, Fields: m.fields, Bodies: m.bodies, Written: m.written, Renders: m.renders, Links: m.links,
		Findings: m.findings,
	})
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	if want, got := packsOf(t, m), packsOf(t, captured(t, raw)); got != want {
		t.Errorf("the packs of a loaded capture =\n%s\nwant\n%s", got, want)
	}
}
