package imports_test

import (
	"encoding/json"
	"maps"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/davidmovas/postulator/internal/adapters/sqlite"
	"github.com/davidmovas/postulator/internal/adapters/sqlite/sqlitetest"
	"github.com/davidmovas/postulator/internal/application/imports"
	"github.com/davidmovas/postulator/internal/domain/graph"
	"github.com/davidmovas/postulator/internal/domain/importmap"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/kernel/id"
)

const clientSheet = "Root Entity,Category,Subcategory,Recommended URL Layer,Title,H1,Keywords\n" +
	"Peptides,,,/peptides/,Research peptides,Peptides,research peptides\n" +
	"Peptides,BPC-157,,/peptides/bpc-157/,BPC-157 peptide,BPC-157,\"bpc 157 (12000), buy bpc 157 (5400)\"\n" +
	"Peptides,BPC-157,Liquid,/peptides/bpc-157/liquid/,BPC-157 liquid,BPC-157 Liquid,bpc 157 liquid (900)\n" +
	"Peptides,BPC-157,Powder,/peptides/bpc-157/powder/,BPC-157 powder,BPC-157 Powder,\n" +
	"Peptides,TB-500,,/peptides/tb-500/,TB-500 peptide,TB-500,tb 500\n" +
	"Peptides,TB-500,Liquid,/peptides/tb-500/liquid/,TB-500 liquid,TB-500 Liquid,tb 500 liquid\n"

const twoLiquids = "url,entity,parent\n/bpc-157/,BPC-157,\n/bpc-157/liquid/,Liquid,BPC-157\n/tb-500/,TB-500,\n/tb-500/liquid/,Liquid,TB-500\n"

const twoLiquidsOneUnderARoot = "url,entity,parent\n/peptides/,Peptides,\n/peptides/bpc-157/,BPC-157,Peptides\n" +
	"/peptides/bpc-157/liquid/,Liquid,BPC-157\n/tb-500/,TB-500,\n/tb-500/liquid/,Liquid,TB-500\n"

func (h harness) scopeOf(t *testing.T, name string) []string {
	t.Helper()

	stored := h.entities(t)
	byID := make(map[string]graph.Entity, len(stored))
	for i := range stored {
		byID[stored[i].ID] = stored[i]
	}
	out := make([]string, 0, 1)
	for i := range stored {
		if stored[i].Name != name {
			continue
		}
		chain := ""
		for at, held := byID[deref(stored[i].ScopeID)]; held; at, held = byID[deref(at.ScopeID)] {
			chain = at.Name + " › " + chain
		}
		out = append(out, chain+name)
	}
	slices.Sort(out)
	return out
}

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
	for i := range report.Entities {
		if report.Entities[i].Name == name {
			out = append(out, report.Entities[i])
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

func TestOnlyTheRootColumnMakesAGroupAndTheUrlPlacesTheRowsInIt(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	path := h.file(t, "client.csv", clientSheet)
	mapping := h.detected(t, path)
	if !slices.Equal(mapping.Options.LevelColumns, []string{"Root Entity"}) {
		t.Fatalf("levels = %v", mapping.Options.LevelColumns)
	}

	report := h.preview(t, path, mapping)
	if len(report.Errors) != 0 {
		t.Fatalf("errors = %+v", report.Errors)
	}
	if left := ignored(report); !slices.Equal(left, []string{"Category", "Subcategory"}) {
		t.Fatalf("ignored = %v, want the category columns", left)
	}
	owners := map[string]string{
		"/peptides/":                "Peptides",
		"/peptides/bpc-157/":        "BPC-157",
		"/peptides/bpc-157/liquid/": "BPC-157 Liquid",
		"/peptides/bpc-157/powder/": "BPC-157 Powder",
		"/peptides/tb-500/":         "TB-500",
		"/peptides/tb-500/liquid/":  "TB-500 Liquid",
	}
	for at, named := range owners {
		if found, ok := page(report, at); !ok || found.Entity != named {
			t.Fatalf("%s = %+v, want it owned by %s", at, found, named)
		}
	}
	if len(report.Entities) != 6 || len(entitiesNamed(report, "Liquid")) != 0 {
		t.Fatalf("entities = %+v, want the six rows and no entity of a category cell", report.Entities)
	}
	for _, want := range [][2]string{
		{"BPC-157", "Peptides"}, {"TB-500", "Peptides"}, {"BPC-157 Liquid", "BPC-157"}, {"BPC-157 Powder", "BPC-157"},
		{"TB-500 Liquid", "TB-500"},
	} {
		if !hasEdge(report, want[0], want[1], string(graph.EdgeParent)) {
			t.Fatalf("edges = %+v, want %s under %s", report.Edges, want[0], want[1])
		}
	}
	if len(report.Edges) != 5 {
		t.Fatalf("edges = %+v, want five", report.Edges)
	}
	if groups := report.Groups; len(groups) != 1 || groups[0].Page != "/peptides/" || groups[0].Rows != 6 {
		t.Fatalf("groups = %+v, want the root group alone on /peptides/", groups)
	}

	first := h.apply(t, path, mapping)
	if first.Counts.EntitiesCreated != 6 || first.Counts.PagesCreated != 6 || first.Counts.EdgesCreated != 5 {
		t.Fatalf("counts = %+v", first.Counts)
	}
	for _, held := range h.entities(t) {
		if held.CanonicalPageID == nil {
			t.Fatalf("%s has no canonical page", held.Name)
		}
	}
	if got := h.scopeOf(t, "TB-500 Liquid"); len(got) != 1 || got[0] != "Peptides › TB-500 › TB-500 Liquid" {
		t.Fatalf("the stored TB-500 Liquid sits at %v", got)
	}

	again := h.apply(t, path, mapping)
	if again.Counts.EntitiesCreated != 0 || again.Counts.EdgesCreated != 0 || again.Counts.PagesCreated != 0 {
		t.Fatalf("a second import of the same sheet wrote %+v", again.Counts)
	}
	if len(h.entities(t)) != 6 {
		t.Fatalf("entities after a second import = %d", len(h.entities(t)))
	}
}

func (h harness) kinds(t *testing.T) map[string]graph.Kind {
	t.Helper()

	stored := h.entities(t)
	out := make(map[string]graph.Kind, len(stored))
	for i := range stored {
		out[stored[i].Name] = stored[i].Kind
	}
	return out
}

func TestOnlyTheRootLevelMakesAnEntityOfItsGroup(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		sheet string
		want  map[string]graph.Kind
	}{
		{
			name: "a root beside category and subcategory columns",
			sheet: "Root Entity,Category,Subcategory,URL,H1\nPeptides,,,/peptides/,Peptides\nPeptides,BPC-157,,/peptides/bpc-157/,BPC-157\n" +
				"Peptides,BPC-157,Liquid,/peptides/bpc-157/liquid/,BPC-157 Liquid\n",
			want: map[string]graph.Kind{"Peptides": graph.KindHub, "BPC-157": graph.KindTopic, "BPC-157 Liquid": graph.KindTopic},
		},
		{
			name:  "a root category column is ignored",
			sheet: "Root Category,Category,URL,H1\nPeptides,,/peptides/,Peptides\nPeptides,BPC-157,/peptides/bpc-157/,BPC-157\n",
			want:  map[string]graph.Kind{"Peptides": graph.KindTopic, "BPC-157": graph.KindTopic},
		},
		{
			name:  "a kind cell names the kind",
			sheet: "Root,Category,URL,Entity Level\nPeptides,,/peptides/,Topic\nPeptides,BPC-157,/peptides/bpc-157/,Compound/Product\n",
			want:  map[string]graph.Kind{"Peptides": graph.KindTopic, "Bpc 157": graph.KindProduct},
		},
		{
			name:  "category columns alone make no entity",
			sheet: "Category,Subcategory,URL,H1\nPeptides,Liquid,/a/,Alpha\nPeptides,Powder,/b/,Beta\n",
			want:  map[string]graph.Kind{"Alpha": graph.KindTopic, "Beta": graph.KindTopic},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			h := newHarness(t)
			path := h.file(t, "levels.csv", tc.sheet)
			applied := h.apply(t, path, h.detected(t, path))
			if len(applied.Report.Errors) != 0 {
				t.Fatalf("errors = %+v", applied.Report.Errors)
			}
			if orphaned := findings(applied.Report.Warnings, imports.CodeGroupWithoutPage); len(orphaned) != 0 {
				t.Fatalf("groups without a page = %+v", orphaned)
			}
			if got := h.kinds(t); !maps.Equal(got, tc.want) {
				t.Fatalf("entities = %+v, want %+v", got, tc.want)
			}
		})
	}
}

func TestAUrlParentWinsInsideTheRowsRootGroup(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		before string
		sheet  string
		scopes map[string][]string
		edges  [][2]string
		absent [][2]string
	}{
		{
			name: "a row under its url parent, not under the root",
			sheet: "Root Entity,URL,H1\nPeptides,/peptides/,Peptides\nPeptides,/peptides/bpc-157/,BPC-157\n" +
				"Peptides,/peptides/bpc-157/liquid/,BPC-157 Liquid\n",
			scopes: map[string][]string{
				"BPC-157": {"Peptides › BPC-157"}, "BPC-157 Liquid": {"Peptides › BPC-157 › BPC-157 Liquid"},
			},
			edges:  [][2]string{{"BPC-157", "Peptides"}, {"BPC-157 Liquid", "BPC-157"}},
			absent: [][2]string{{"BPC-157 Liquid", "Peptides"}},
		},
		{
			name:   "a url parent in another root group",
			sheet:  "Root Entity,URL,H1\nPeptides,/peptides/,Peptides\nBlends,/blends/,Blends\nBlends,/peptides/mix/,Mix\n",
			scopes: map[string][]string{"Mix": {"Blends › Mix"}},
			edges:  [][2]string{{"Mix", "Blends"}},
			absent: [][2]string{{"Mix", "Peptides"}},
		},
		{
			name:   "an intermediate page the import makes under the group's page",
			sheet:  "Root Entity,URL,H1\nPeptides,/peptides/,Peptides\nPeptides,/peptides/bpc-157/liquid/,Liquid\n",
			scopes: map[string][]string{"Bpc 157": {"Peptides › Bpc 157"}, "Liquid": {"Peptides › Bpc 157 › Liquid"}},
			edges:  [][2]string{{"Bpc 157", "Peptides"}, {"Liquid", "Bpc 157"}},
		},
		{
			name:   "an intermediate page the import makes under the group's page on the site",
			before: "Root Entity,URL,H1\nPeptides,/peptides/,Peptides\n",
			sheet:  "Root Entity,URL,H1\nPeptides,/peptides/bpc-157/liquid/,Liquid\n",
			scopes: map[string][]string{"Bpc 157": {"Peptides › Bpc 157"}, "Liquid": {"Peptides › Bpc 157 › Liquid"}},
			edges:  [][2]string{{"Bpc 157", "Peptides"}, {"Liquid", "Bpc 157"}},
			absent: [][2]string{{"Liquid", "Peptides"}},
		},
		{
			name:   "a url parent outside every group that names its own parent",
			sheet:  "Root Entity,URL,H1,Parent\nPeptides,/peptides/,Peptides,\n,/shop/,Shop,Peptides\nPeptides,/shop/vial/,Vial,\n",
			scopes: map[string][]string{"Shop": {"Peptides › Shop"}, "Vial": {"Peptides › Vial"}},
			edges:  [][2]string{{"Shop", "Peptides"}, {"Vial", "Peptides"}},
			absent: [][2]string{{"Vial", "Shop"}},
		},
		{
			name: "a parent cell over the url parent",
			sheet: "Root Entity,URL,Entity,Parent\nPeptides,/peptides/,Peptides,\nPeptides,/peptides/bpc-157/,BPC-157,\n" +
				"Peptides,/peptides/tb-500/,TB-500,\nPeptides,/peptides/bpc-157/vial/,Vial,TB-500\n",
			scopes: map[string][]string{"Vial": {"Peptides › TB-500 › Vial"}},
			edges:  [][2]string{{"Vial", "TB-500"}},
			absent: [][2]string{{"Vial", "BPC-157"}},
		},
		{
			name:   "an entity the site holds under another parent keeps it",
			before: "url,entity,parent\n/tb-500/,TB-500,\n/tb-500/drops/,Drops,TB-500\n",
			sheet: "Root Entity,URL,Entity\nPeptides,/peptides/,Peptides\nPeptides,/peptides/bpc-157/,BPC-157\n" +
				"Peptides,/peptides/bpc-157/drops/,Drops\n",
			scopes: map[string][]string{"Drops": {"TB-500 › Drops"}},
			absent: [][2]string{{"Drops", "BPC-157"}, {"Drops", "Peptides"}},
		},
		{
			name: "two rows of one name under two url parents of one group",
			sheet: "Root Entity,URL,Entity\nPeptides,/peptides/,Peptides\nPeptides,/peptides/bpc-157/,BPC-157\n" +
				"Peptides,/peptides/bpc-157/liquid/,Liquid\nPeptides,/peptides/tb-500/,TB-500\nPeptides,/peptides/tb-500/liquid/,Liquid\n",
			scopes: map[string][]string{"Liquid": {"Peptides › BPC-157 › Liquid", "Peptides › TB-500 › Liquid"}},
			edges:  [][2]string{{"Liquid", "BPC-157"}, {"Liquid", "TB-500"}},
		},
		{
			name:   "a parent cell naming a namesake the site holds deeper in the group",
			before: twoLiquidsOneUnderARoot,
			sheet:  "Root Entity,URL,Entity,Parent\nPeptides,/drops/,Drops,Liquid\n",
			scopes: map[string][]string{"Drops": {"Peptides › BPC-157 › Liquid › Drops"}},
		},
		{
			name: "a parent cell naming a namesake the sheet plans deeper in the group",
			sheet: "Root Entity,URL,Entity,Parent\nPeptides,/peptides/,Peptides,\nPeptides,/peptides/bpc-157/,BPC-157,\n" +
				"Peptides,/peptides/bpc-157/liquid/,Liquid,\n,/tb-500/,TB-500,\n,/tb-500/liquid/,Liquid,TB-500\nPeptides,/drops/,Drops,Liquid\n",
			scopes: map[string][]string{
				"Drops": {"Peptides › BPC-157 › Liquid › Drops"}, "Liquid": {"Peptides › BPC-157 › Liquid", "TB-500 › Liquid"},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			h := newHarness(t)
			if tc.before != "" {
				before := h.file(t, "before.csv", tc.before)
				if applied := h.apply(t, before, h.detected(t, before)); len(applied.Report.Errors) != 0 {
					t.Fatalf("the site's own sheet reported %+v", applied.Report.Errors)
				}
			}
			path := h.file(t, "sheet.csv", tc.sheet)
			applied := h.apply(t, path, h.detected(t, path))
			if len(applied.Report.Errors) != 0 {
				t.Fatalf("errors = %+v", applied.Report.Errors)
			}
			for name, want := range tc.scopes {
				if got := h.scopeOf(t, name); !slices.Equal(got, want) {
					t.Errorf("%s sits at %v, want %v", name, got, want)
				}
			}
			for _, want := range tc.edges {
				if !hasEdge(applied.Report, want[0], want[1], string(graph.EdgeParent)) {
					t.Errorf("edges = %+v, want %s under %s", applied.Report.Edges, want[0], want[1])
				}
			}
			for _, unwanted := range tc.absent {
				if hasEdge(applied.Report, unwanted[0], unwanted[1], string(graph.EdgeParent)) {
					t.Errorf("edges = %+v, want no %s under %s", applied.Report.Edges, unwanted[0], unwanted[1])
				}
			}
		})
	}
}

func TestASavedCategoryLevelColumnIsIgnored(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	path := h.file(t, "client.csv", clientSheet)
	mapping := h.detected(t, path)
	mapping.Options.LevelColumns = []string{"Root Entity", "Category", "Subcategory"}
	saved, err := h.service.SaveMapping(t.Context(), imports.SaveMappingRequest{Mapping: mapping})
	if err != nil {
		t.Fatalf("SaveMapping: %v", err)
	}
	if !slices.Equal(saved.Mapping.Options.LevelColumns, mapping.Options.LevelColumns) {
		t.Fatalf("saved levels = %v, want them kept as given", saved.Mapping.Options.LevelColumns)
	}

	report := h.preview(t, path, imports.Mapping{ID: saved.Mapping.ID})
	if left := ignored(report); !slices.Equal(left, []string{"Category", "Subcategory"}) {
		t.Fatalf("ignored = %v, want the category columns", left)
	}
	if groups := report.Groups; len(groups) != 1 || !slices.Equal(groups[0].Path, []string{"Peptides"}) {
		t.Fatalf("groups = %+v, want the root alone", groups)
	}
	if len(report.Entities) != 6 || len(entitiesNamed(report, "BPC-157")) != 1 {
		t.Fatalf("entities = %+v, want one per row", report.Entities)
	}

	alone := h.mapping(nil)
	alone.Name = "categories"
	alone.Options.LevelColumns = []string{"Category", "Subcategory"}
	if _, err := h.service.SaveMapping(t.Context(), imports.SaveMappingRequest{Mapping: alone}); !errors.IsCode(err, errors.Invalid) {
		t.Fatalf("SaveMapping of category columns alone = %v, want it refused as mapping nothing", err)
	}
}

func TestThePreviewAndTheApplyCarryNoCategories(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	path := h.file(t, "client.csv", clientSheet)
	applied := h.apply(t, path, h.detected(t, path))
	encoded, err := json.Marshal(applied)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	for _, key := range []string{`"categories":`, `"categoriesCreated":`, `"categoriesDeleted":`} {
		if strings.Contains(string(encoded), key) {
			t.Fatalf("the apply carries %s: %s", key, encoded)
		}
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
		{name: "a row named as the group", sheet: "root,url,h1\nPeptides,/catalog/,Peptides\nPeptides,/catalog/bpc/,BPC\n", page: "/catalog/"},
		{name: "a row whose slug is the group's", sheet: "root,url,h1\nPeptides,/peptides/,All our peptides\nPeptides,/peptides/bpc/,BPC\n", page: "/peptides/"},
		{name: "a row above every other row of the group", sheet: "root,url,h1\nPeptides,/catalog/,Catalog\nPeptides,/catalog/bpc/,BPC\nPeptides,/catalog/tb/,TB\n", page: "/catalog/"},
		{name: "an intermediate page whose slug is the group's", sheet: "root,url,h1\nPeptides,/peptides/bpc/,BPC\n", page: "/peptides/"},
		{name: "nothing that says which page", sheet: "root,url,h1\nPeptides,/a/,Alpha\nPeptides,/b/,Beta\n", orphan: true},
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
	if applied.Counts.EntitiesCreated != 1 || applied.Counts.EdgesCreated != 1 {
		t.Fatalf("counts = %+v, want the capsules alone created and put under BPC-157", applied.Counts)
	}
	for _, held := range h.entities(t) {
		if held.Name == "Yes" || held.Name == "No" || held.Name == "NO" {
			t.Fatalf("the Entity? column made an entity %q", held.Name)
		}
	}
	if got := h.scopeOf(t, "Capsules"); len(got) != 1 || got[0] != "Peptides › BPC-157 › Capsules" {
		t.Fatalf("Capsules sits at %v, want under BPC-157", got)
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
			liquids := h.file(t, "liquids.csv", twoLiquids)
			h.apply(t, liquids, h.detected(t, liquids))

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
	liquids := h.file(t, "liquids.csv", twoLiquids)
	h.apply(t, liquids, h.detected(t, liquids))

	path := h.file(t, "drops.csv", "Root Entity,URL,Entity,Parent\nTB-500,/tb-500/liquid/drops/,Drops,Liquid\n")
	applied := h.apply(t, path, h.detected(t, path))
	if len(applied.Report.Errors) != 0 {
		t.Fatalf("errors = %+v", applied.Report.Errors)
	}
	if got := h.scopeOf(t, "Drops"); len(got) != 1 || got[0] != "TB-500 › Liquid › Drops" {
		t.Fatalf("Drops sits at %v, want under the Liquid of TB-500", got)
	}
}

func TestThePreviewSaysWhatEachColumnBecame(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	path := h.file(t, "variations.csv", "Parent Product Entity,URL,Detected Form / Variation,Entity?,Reason,Entity ID\n"+
		"BPC-157,/bpc-157/liquid/,Liquid,NO,A form the shop sells,E-17\n")
	report := h.preview(t, path, h.detected(t, path))

	want := []imports.PreviewColumn{
		{Header: "Parent Product Entity", Use: "field", Field: "parent_entity"},
		{Header: "URL", Use: "field", Field: "path"},
		{Header: "Detected Form / Variation", Use: "note"},
		{Header: "Entity?", Use: "ignored"},
		{Header: "Reason", Use: "note"},
		{Header: "Entity ID", Use: "ignored"},
	}
	if !slices.Equal(report.Columns, want) {
		t.Fatalf("columns =\n%+v\nwant\n%+v", report.Columns, want)
	}
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
