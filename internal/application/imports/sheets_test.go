package imports_test

import (
	"path/filepath"
	"slices"
	"testing"

	"github.com/davidmovas/postulator/internal/adapters/sqlite"
	"github.com/davidmovas/postulator/internal/adapters/sqlite/sqlitetest"
	"github.com/davidmovas/postulator/internal/application/imports"
	"github.com/davidmovas/postulator/internal/domain/graph"
	"github.com/davidmovas/postulator/internal/domain/importmap"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/kernel/id"
)

const clientSheet = "Root Entity,Category,Subcategory,Recommended URL Layer,Title,H1,Keywords\n" +
	"Peptides,BPC-157,,/peptides/bpc-157/,BPC-157 peptide,BPC-157,\"bpc 157 (12000), buy bpc 157 (5400)\"\n" +
	"Peptides,BPC-157,Liquid,/peptides/bpc-157/liquid/,BPC-157 liquid,BPC-157 Liquid,bpc 157 liquid (900)\n" +
	"Peptides,BPC-157,Powder,/peptides/bpc-157/powder/,BPC-157 powder,BPC-157 Powder,\n" +
	"Peptides,TB-500,Liquid,/peptides/tb-500/liquid/,TB-500 liquid,TB-500 Liquid,tb 500 liquid\n"

func (h harness) detected(t *testing.T, path string) imports.Mapping {
	t.Helper()

	seen, err := h.service.Inspect(t.Context(), imports.InspectRequest{SiteID: h.siteID, Path: path})
	if err != nil {
		t.Fatalf("Inspect: %v", err)
	}
	mapping := seen.Detected
	mapping.Name = "sheet"
	return mapping
}

func entitiesNamed(report imports.PreviewReport, name string) []imports.PreviewEntity {
	out := make([]imports.PreviewEntity, 0)
	for _, held := range report.Entities {
		if held.Name == name {
			out = append(out, held)
		}
	}
	return out
}

func hasEdge(report imports.PreviewReport, from, to, kind string) bool {
	for _, edge := range report.Edges {
		if edge.From == from && edge.To == to && edge.Kind == kind {
			return true
		}
	}
	return false
}

func TestEveryPageRowGetsAnEntity(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		row   string
		path  string
		named string
	}{
		{name: "from the entity column", row: "/a/,Alpha,Heading,Title", path: "/a/", named: "Alpha"},
		{name: "from the heading when no entity is named", row: "/a/,,Heading,Title", path: "/a/", named: "Heading"},
		{name: "from the title when the heading is empty", row: "/a/,,,Title", path: "/a/", named: "Title"},
		{name: "from the slug when nothing names it", row: "/bpc-157-liquid/,,,", path: "/bpc-157-liquid/", named: "Bpc 157 Liquid"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			h := newHarness(t)
			report := h.preview(t, h.file(t, "rows.csv", "url,entity,h1,title\n"+tc.row+"\n"), h.mapping(map[string]string{
				string(importmap.FieldPath): "url", string(importmap.FieldEntity): "entity",
				string(importmap.FieldH1): "h1", string(importmap.FieldTitle): "title",
			}))
			if found, _ := page(report, tc.path); found.Entity != tc.named {
				t.Fatalf("the page belongs to %q, want %q", found.Entity, tc.named)
			}
			if named, found := entity(report, tc.named); !found || named.Action != string(imports.ActionCreate) {
				t.Fatalf("entities = %+v, want %s created", report.Entities, tc.named)
			}
		})
	}
}

func TestATechnicalPageCarriesNoEntity(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	report := h.preview(t, h.file(t, "technical.csv", "url,title,is entity\n/about/,About,no\n/about/team/,Team,\n/shop/,Shop,yes\n/faq/,FAQ,maybe\n"),
		h.mapping(map[string]string{
			string(importmap.FieldPath): "url", string(importmap.FieldTitle): "title", string(importmap.FieldOwnEntity): "is entity",
		}))

	if about, _ := page(report, "/about/"); about.Entity != "" {
		t.Fatalf("the technical page belongs to %q", about.Entity)
	}
	if _, found := entity(report, "About"); found {
		t.Fatalf("a technical page made an entity: %+v", report.Entities)
	}
	for path, named := range map[string]string{"/about/team/": "Team", "/shop/": "Shop", "/faq/": "FAQ"} {
		if found, _ := page(report, path); found.Entity != named {
			t.Fatalf("%s belongs to %q, want %q", path, found.Entity, named)
		}
	}
	if technical := findings(report.Warnings, imports.CodeTechnicalParent); len(technical) != 1 || technical[0].Row != 2 {
		t.Fatalf("technical parent findings = %+v, want one on the row of /about/", technical)
	}
	if unread := findings(report.Warnings, imports.CodeUnknownOwnEntity); len(unread) != 1 || unread[0].Row != 5 {
		t.Fatalf("own entity findings = %+v, want one on the row that says maybe", unread)
	}
	for _, edge := range report.Edges {
		if edge.From == "Team" {
			t.Fatalf("Team was put under %s although only a technical page sits above it", edge.To)
		}
	}
}

func TestTheUrlTreeGivesTheParentAFileDoesNotName(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	report := h.preview(t, h.file(t, "tree.csv",
		"url,title\n/peptides/,Peptides\n/peptides/bpc-157/,BPC-157\n/peptides/tb-500/liquid/,TB-500 Liquid\n"),
		h.mapping(map[string]string{string(importmap.FieldPath): "url", string(importmap.FieldTitle): "title"}))

	if len(report.Errors) != 0 {
		t.Fatalf("errors = %+v", report.Errors)
	}
	for _, want := range [][2]string{{"BPC-157", "Peptides"}, {"Tb 500", "Peptides"}, {"TB-500 Liquid", "Tb 500"}} {
		if !hasEdge(report, want[0], want[1], string(graph.EdgeParent)) {
			t.Fatalf("edges = %+v, want %s under %s", report.Edges, want[0], want[1])
		}
	}
	if middle, _ := page(report, "/peptides/tb-500/"); !middle.Generated || middle.Entity != "Tb 500" {
		t.Fatalf("the intermediate page = %+v, want it generated with an entity of its own", middle)
	}
}

func TestLevelColumnsMakeAChainOfGroups(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	path := h.file(t, "client.csv", clientSheet)
	mapping := h.detected(t, path)
	if !slices.Equal(mapping.Options.LevelColumns, []string{"Root Entity", "Category", "Subcategory"}) {
		t.Fatalf("levels = %v", mapping.Options.LevelColumns)
	}

	report := h.preview(t, path, mapping)
	if len(report.Errors) != 0 {
		t.Fatalf("errors = %+v", report.Errors)
	}
	owners := map[string]string{
		"/peptides/":                "Peptides",
		"/peptides/bpc-157/":        "BPC-157",
		"/peptides/bpc-157/liquid/": "Liquid",
		"/peptides/bpc-157/powder/": "Powder",
		"/peptides/tb-500/":         "TB-500",
		"/peptides/tb-500/liquid/":  "Liquid",
	}
	for at, named := range owners {
		if found, ok := page(report, at); !ok || found.Entity != named {
			t.Fatalf("%s = %+v, want it owned by %s", at, found, named)
		}
	}
	liquids := entitiesNamed(report, "Liquid")
	parents := make([]string, 0, len(liquids))
	for _, liquid := range liquids {
		parents = append(parents, liquid.Parent)
	}
	slices.Sort(parents)
	if !slices.Equal(parents, []string{"BPC-157", "TB-500"}) {
		t.Fatalf("the Liquids sit under %v, want BPC-157 and TB-500", parents)
	}
	if len(report.Entities) != 6 {
		t.Fatalf("entities = %+v, want six", report.Entities)
	}
	for _, want := range [][2]string{
		{"BPC-157", "Peptides"}, {"TB-500", "Peptides"}, {"Liquid", "BPC-157"}, {"Powder", "BPC-157"}, {"Liquid", "TB-500"},
	} {
		if !hasEdge(report, want[0], want[1], string(graph.EdgeParent)) {
			t.Fatalf("edges = %+v, want %s under %s", report.Edges, want[0], want[1])
		}
	}
	if len(findings(report.Warnings, imports.CodeGroupWithoutPage)) != 0 {
		t.Fatalf("a group found no page: %+v", report.Warnings)
	}

	first := h.apply(t, path, mapping)
	if first.Counts.EntitiesCreated != 6 || first.Counts.PagesCreated != 6 {
		t.Fatalf("counts = %+v", first.Counts)
	}
	stored := h.entities(t)
	byID := make(map[string]graph.Entity, len(stored))
	for _, held := range stored {
		byID[held.ID] = held
	}
	scoped := []string{}
	for _, held := range stored {
		if held.Name == "Liquid" && held.ScopeID != nil {
			scoped = append(scoped, byID[*held.ScopeID].Name)
		}
		if held.CanonicalPageID == nil {
			t.Fatalf("%s has no canonical page", held.Name)
		}
	}
	slices.Sort(scoped)
	if !slices.Equal(scoped, []string{"BPC-157", "TB-500"}) {
		t.Fatalf("the stored Liquids sit under %v", scoped)
	}

	again := h.apply(t, path, mapping)
	if again.Counts.EntitiesCreated != 0 || again.Counts.EdgesCreated != 0 || again.Counts.PagesCreated != 0 {
		t.Fatalf("a second import of the same sheet wrote %+v", again.Counts)
	}
	if len(h.entities(t)) != 6 {
		t.Fatalf("entities after a second import = %d", len(h.entities(t)))
	}
}

func TestAGroupTakesAPageOnlyOnEvidence(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		sheet  string
		page   string
		orphan bool
	}{
		{name: "a row named as the group", sheet: "category,url,h1\nPeptides,/catalog/,Peptides\nPeptides,/catalog/bpc/,BPC\n", page: "/catalog/"},
		{name: "a row whose slug is the group's", sheet: "category,url,h1\nPeptides,/peptides/,All our peptides\nPeptides,/peptides/bpc/,BPC\n", page: "/peptides/"},
		{name: "a row above every other row of the group", sheet: "category,url,h1\nPeptides,/catalog/,Catalog\nPeptides,/catalog/bpc/,BPC\nPeptides,/catalog/tb/,TB\n", page: "/catalog/"},
		{name: "an intermediate page whose slug is the group's", sheet: "category,url,h1\nPeptides,/peptides/bpc/,BPC\n", page: "/peptides/"},
		{name: "nothing that says which page", sheet: "category,url,h1\nPeptides,/a/,Alpha\nPeptides,/b/,Beta\n", orphan: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			h := newHarness(t)
			path := h.file(t, "groups.csv", tc.sheet)
			report := h.preview(t, path, h.detected(t, path))
			if len(report.Errors) != 0 {
				t.Fatalf("errors = %+v", report.Errors)
			}
			orphaned := findings(report.Warnings, imports.CodeGroupWithoutPage)
			if tc.orphan {
				if len(orphaned) != 1 {
					t.Fatalf("group findings = %+v, want one", orphaned)
				}
				for _, at := range []string{"/a/", "/b/"} {
					found, _ := page(report, at)
					if found.Entity == "Peptides" || !hasEdge(report, found.Entity, "Peptides", string(graph.EdgeParent)) {
						t.Fatalf("%s = %+v, want a page of its own under the group", at, found)
					}
				}
				return
			}
			if len(orphaned) != 0 {
				t.Fatalf("group findings = %+v, want none", orphaned)
			}
			if found, _ := page(report, tc.page); found.Entity != "Peptides" {
				t.Fatalf("%s belongs to %q, want the group", tc.page, found.Entity)
			}
			if groups := report.Groups; len(groups) != 1 || groups[0].Page != tc.page {
				t.Fatalf("groups = %+v, want Peptides on %s", groups, tc.page)
			}
		})
	}
}

func TestTheVariationSheetPutsEachFormUnderItsProduct(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	client := h.file(t, "client.csv", clientSheet)
	h.apply(t, client, h.detected(t, client))

	variations := h.file(t, "variations.csv", "Parent Product Entity,URL,Detected Form / Variation,Entity?,Reason\n"+
		"BPC-157,/peptides/bpc-157/capsules/,Capsules,NO,A form the shop sells\n"+
		"TB-500,/peptides/tb-500/liquid/,Liquid,NO,Already a page\n")
	mapping := h.detected(t, variations)
	if _, mapped := mapping.Columns[string(importmap.FieldOwnEntity)]; mapped {
		t.Fatalf("Entity? was detected: %v", mapping.Columns)
	}

	applied := h.apply(t, variations, mapping)
	if len(applied.Report.Errors) != 0 {
		t.Fatalf("errors = %+v", applied.Report.Errors)
	}
	if applied.Counts.EntitiesCreated != 1 {
		t.Fatalf("counts = %+v, want the capsules alone created", applied.Counts)
	}
	byID := make(map[string]graph.Entity)
	for _, held := range h.entities(t) {
		byID[held.ID] = held
		if held.Name == "Yes" || held.Name == "No" || held.Name == "NO" {
			t.Fatalf("the Entity? column made an entity %q", held.Name)
		}
	}
	for _, held := range byID {
		if held.Name == "Capsules" && (held.ScopeID == nil || byID[*held.ScopeID].Name != "BPC-157") {
			t.Fatalf("Capsules sits under %v, want BPC-157", held.ScopeID)
		}
	}
}

func TestAParentOrANameTwoEntitiesShareIsAnErrorNotAGuess(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		sheet string
		code  imports.FindingCode
	}{
		{name: "a parent named Liquid", sheet: "url,entity,parent\n/drops/,Drops,Liquid\n", code: imports.CodeAmbiguousParent},
		{name: "an entity named Liquid with nothing above it", sheet: "url,entity\n/liquid/,Liquid\n", code: imports.CodeAmbiguousEntity},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			h := newHarness(t)
			client := h.file(t, "client.csv", clientSheet)
			h.apply(t, client, h.detected(t, client))

			path := h.file(t, "ambiguous.csv", tc.sheet)
			report := h.preview(t, path, h.detected(t, path))
			if len(findings(report.Errors, tc.code)) != 1 {
				t.Fatalf("errors = %+v, want one %s", report.Errors, tc.code)
			}
		})
	}
}

func TestARowsGroupSaysWhichParentItMeans(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	client := h.file(t, "client.csv", clientSheet)
	h.apply(t, client, h.detected(t, client))

	path := h.file(t, "drops.csv", "Category,URL,Entity,Parent\nTB-500,/peptides/tb-500/liquid/drops/,Drops,Liquid\n")
	applied := h.apply(t, path, h.detected(t, path))
	if len(applied.Report.Errors) != 0 {
		t.Fatalf("errors = %+v", applied.Report.Errors)
	}

	byID := make(map[string]graph.Entity)
	for _, held := range h.entities(t) {
		byID[held.ID] = held
	}
	for _, held := range byID {
		if held.Name != "Drops" {
			continue
		}
		parent := byID[*held.ScopeID]
		if parent.Name != "Liquid" || byID[*parent.ScopeID].Name != "TB-500" {
			t.Fatalf("Drops sits under %s, want the Liquid of TB-500", parent.Name)
		}
		return
	}
	t.Fatal("Drops was not created")
}

func notesAt(t *testing.T, h harness, path string) []pagemap.Note {
	t.Helper()

	stored := h.pages(t)
	for i := range stored {
		if stored[i].Path == path {
			return stored[i].Notes
		}
	}
	t.Fatalf("no page at %s", path)
	return nil
}

func TestNoteColumnsTravelWithThePage(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	first := h.file(t, "notes.csv", "URL,Title,Intent Owner,Notes\n/bpc-157/,BPC-157,Commercial,Sold as a 10 ml vial\n/tb-500/,TB-500,,\n")
	mapping := h.detected(t, first)
	if !slices.Equal(mapping.Options.NoteColumns, []string{"Intent Owner", "Notes"}) {
		t.Fatalf("note columns = %v", mapping.Options.NoteColumns)
	}
	h.apply(t, first, mapping)

	want := []pagemap.Note{{Label: "Intent Owner", Text: "Commercial"}, {Label: "Notes", Text: "Sold as a 10 ml vial"}}
	if got := notesAt(t, h, "/bpc-157/"); !slices.Equal(got, want) {
		t.Fatalf("notes = %+v, want %+v", got, want)
	}
	if got := notesAt(t, h, "/tb-500/"); len(got) != 0 {
		t.Fatalf("an empty note cell gave the page %+v", got)
	}

	second := h.file(t, "again.csv", "URL,Title,Notes\n/bpc-157/,BPC-157,Now a 5 ml vial\n/tb-500/,TB-500,\n")
	h.apply(t, second, h.detected(t, second))
	want = []pagemap.Note{{Label: "Intent Owner", Text: "Commercial"}, {Label: "Notes", Text: "Now a 5 ml vial"}}
	if got := notesAt(t, h, "/bpc-157/"); !slices.Equal(got, want) {
		t.Fatalf("notes after a second import = %+v, want %+v", got, want)
	}

	exported := filepath.Join(h.dir, "export.xlsx")
	if _, err := h.service.Export(t.Context(), imports.ExportRequest{SiteID: h.siteID, Path: exported}); err != nil {
		t.Fatalf("Export: %v", err)
	}
	target := newHarness(t)
	target.apply(t, exported, target.detected(t, exported))
	if got := notesAt(t, target, "/bpc-157/"); !slices.Equal(got, want) {
		t.Fatalf("notes after an export and an import = %+v, want %+v", got, want)
	}
}

func TestAnUnpublishedAncestorWithoutAnEntityGetsOne(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	guides, err := pagemap.NewPage(pagemap.Page{
		ID: id.New(), SiteID: h.siteID, Path: "/guides/", Slug: "guides", WPType: pagemap.WPPage, Title: "Guides",
		Status: pagemap.StatusPlanned, CreatedAt: sqlitetest.Stamp, UpdatedAt: sqlitetest.Stamp,
	})
	if err != nil {
		t.Fatalf("build the page: %v", err)
	}
	if err = sqlite.NewPageRepo(h.store).Insert(t.Context(), guides); err != nil {
		t.Fatalf("insert the page: %v", err)
	}

	path := h.file(t, "guides.csv", "url,title\n/guides/espresso/,Espresso\n")
	report := h.preview(t, path, h.detected(t, path))
	if found, _ := page(report, "/guides/"); found.Entity != "Guides" || found.Action != string(imports.ActionUpdate) {
		t.Fatalf("the unpublished ancestor = %+v, want it given an entity", found)
	}
	if !hasEdge(report, "Espresso", "Guides", string(graph.EdgeParent)) {
		t.Fatalf("edges = %+v, want Espresso under Guides", report.Edges)
	}
}
