package contextpack_test

import (
	"strconv"
	"strings"
	"testing"

	"github.com/davidmovas/postulator/internal/domain/contextpack"
	"github.com/davidmovas/postulator/internal/domain/finding"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/pagetype"
)

func open(code finding.Code, pageID string) finding.Finding {
	return finding.Finding{ID: "f-" + string(code) + "-" + pageID, SiteID: siteID, PageID: new(pageID), Code: code, Severity: finding.Warning, Status: finding.Open}
}

func withFindings(t *testing.T) contextpack.Snapshot {
	t.Helper()

	f := replicaLike()
	planned := typed(mapped(under(pagemap.Page{
		ID: "planned", SiteID: siteID, Path: "/peptides/tb-500-guide/", Slug: "tb-500-guide", WPType: pagemap.WPPage,
		Status: pagemap.StatusPlanned, Title: "TB-500 guide", CreatedAt: stamp, UpdatedAt: stamp,
	}, "hub", pagemap.ParentWorkbook), "e-bpc"), pagetype.Entity)
	archived := onSitePage("gone", "/old/", "Old page", 90, pagemap.WPPage)
	archived.Status = pagemap.StatusArchived
	f.pages = append(f.pages, planned, archived)
	s := f.snapshot(t)
	s.Findings = []finding.Finding{
		open(finding.ParentDiffers, "bpc"), open(finding.ParentDiffers, "healing"), open(finding.RenderUnreachable, "guides"),
	}
	return s
}

func TestTheOverviewCountsWhatTheSiteHolds(t *testing.T) {
	t.Parallel()

	got := withFindings(t).Overview()
	if got.Pages != 11 || got.OnSite != 10 || got.Planned != 1 {
		t.Errorf("pages = %d, %d on site, %d planned; want 11, 10, 1 (the archived page left out)", got.Pages, got.OnSite, got.Planned)
	}
	if got.Unmapped != 6 {
		t.Errorf("unmapped = %d, want 6: the front page, the guides and four products, never the store category", got.Unmapped)
	}
	if got.Empty != 8 {
		t.Errorf("empty = %d, want 8: every page on the site but BPC-157 and the store category", got.Empty)
	}
	types := make([]string, 0, len(got.Types))
	for _, kind := range got.Types {
		types = append(types, kind.Name+":"+strconv.Itoa(kind.Pages)+"/"+strconv.Itoa(kind.Planned))
	}
	if want := "Entity:4/1 Product:4/0 Hub:2/0 Store category:1/0"; strings.Join(types, " ") != want {
		t.Errorf("types = %v, want %s", types, want)
	}
	if len(got.Hubs) != 1 || got.Hubs[0].Path != "/peptides/" || got.Hubs[0].Pages != 3 ||
		len(got.Hubs[0].Children) != 1 || got.Hubs[0].Children[0].Path != "/healing-peptides/" {
		t.Errorf("hubs = %+v, want /peptides/ over three pages with /healing-peptides/ under it", got.Hubs)
	}
	if len(got.Findings) != 2 || got.Findings[0] != (contextpack.CodeCount{Code: finding.ParentDiffers, Count: 2}) {
		t.Errorf("findings = %+v, want parent_differs twice first", got.Findings)
	}
}

func TestTheBriefSaysWhatTheSiteHolds(t *testing.T) {
	t.Parallel()

	brief := contextpack.Brief{
		Name: "Peptide Lab", BaseURL: "https://shop.example.com", Overview: withFindings(t).Overview(),
		Links: contextpack.LinkHealth{Audited: 4, Targets: 6, Satisfied: 2, Missing: 2, MissingRequired: 1, Pending: 2, Orphans: 3, OffGraph: 1},
	}
	const want = `THE SITE
Peptide Lab (https://shop.example.com): 11 pages, 10 on the site and 1 planned; 6 carry no entity; 8 on the site hold no text yet.

PAGE TYPES
- Entity: 4 pages, 1 of them planned
- Product: 4 pages
- Hub: 2 pages
- Store category: 1 page

HUBS
The top two levels of the hub tree, with the pages under each hub:
- /peptides/ Peptides: 3 pages
  - /healing-peptides/ Healing Peptides: none

LINKS
Of 4 audited pages the graph owes 6 links: 2 placed, 2 missing (1 of them required), 2 waiting on a page. 3 pages are orphans; 1 link points off the graph.

OPEN FINDINGS
- parent_differs: 2
- render_unreachable: 1
`
	if got := brief.Text(); got != want {
		t.Errorf("Text =\n%s\nwant\n%s", got, want)
	}
}

func TestABareSiteSaysSo(t *testing.T) {
	t.Parallel()

	got := contextpack.Brief{Name: "New", BaseURL: "https://new.example.com"}.Text()
	const want = "THE SITE\nNew (https://new.example.com): no page observed yet; sync the site before anything else.\n"
	if got != want {
		t.Errorf("Text = %q, want %q", got, want)
	}
}

func TestABriefOfPlannedPagesSaysNothingOfTheirText(t *testing.T) {
	t.Parallel()

	got := contextpack.Brief{Name: "New", BaseURL: "https://new.example.com", Overview: contextpack.Overview{
		Pages: 1, Planned: 1, Unmapped: 1, Types: []contextpack.TypeCount{{Pages: 1, Planned: 1}},
	}}.Text()
	for _, want := range []string{"New (https://new.example.com): 1 page, all planned; 1 carries no entity.\n", "- No type: 1 page, 1 of them planned\n"} {
		if !strings.Contains(got, want) {
			t.Errorf("Text =\n%s\nwant it to hold %q", got, want)
		}
	}
}

func TestTheBriefStaysShortOnABigSite(t *testing.T) {
	t.Parallel()

	overview := contextpack.Overview{Pages: 90000, OnSite: 90000}
	for i := range 40 {
		overview.Types = append(overview.Types, contextpack.TypeCount{Name: strings.Repeat("Type ", 12) + strconv.Itoa(i), Pages: 1000})
		overview.Findings = append(overview.Findings, contextpack.CodeCount{Code: finding.Code(strings.Repeat("code_", 8) + strconv.Itoa(i)), Count: 5})
		hub := contextpack.Hub{Path: "/" + strings.Repeat("segment-", 12) + strconv.Itoa(i) + "/", Title: strings.Repeat("A long hub title ", 8), Pages: 500}
		for j := range 30 {
			hub.Children = append(hub.Children, contextpack.Hub{Path: hub.Path + strconv.Itoa(j) + "/", Title: hub.Title, Pages: 10})
		}
		overview.Hubs = append(overview.Hubs, hub)
	}
	text := contextpack.Brief{Name: strings.Repeat("Name", 40), BaseURL: "https://big.example.com", Overview: overview}.Text()
	if len(text) > contextpack.BriefMaxBytes {
		t.Errorf("the brief is %d bytes, over its %d", len(text), contextpack.BriefMaxBytes)
	}
	for _, section := range []string{"PAGE TYPES", "HUBS", "LINKS", "OPEN FINDINGS", "more hubs", "more types", "more codes"} {
		if !strings.Contains(text, section) {
			t.Errorf("the brief lost %q:\n%s", section, text)
		}
	}
}
