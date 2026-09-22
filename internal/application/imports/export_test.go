package imports_test

import (
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/davidmovas/postulator/internal/adapters/importer"
	"github.com/davidmovas/postulator/internal/adapters/sqlite"
	"github.com/davidmovas/postulator/internal/adapters/sqlite/sqlitetest"
	"github.com/davidmovas/postulator/internal/application/imports"
	"github.com/davidmovas/postulator/internal/domain/graph"
	"github.com/davidmovas/postulator/internal/domain/importmap"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/template"
	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/kernel/id"
)

const richSheet = "path,title,h1,primary keyword,keywords,anchors,entity,entity kind,parent,related,page type,meta title,meta description,post type\n" +
	"/,Home,Welcome,shop,shop|store,our shop,Shop,hub,,,hub,Shop,The shop,page\n" +
	"/hosting/,Hosting,Hosting plans,hosting,vps|servers,hosting plans|cheap hosting,Hosting,category,Shop,Domains,category,Hosting,Hosting plans,page\n" +
	"/domains/,Domains,Domains,domains,tld|registrar,buy a domain,Domains,category,Shop,Hosting,category,Domains,Buy a domain,page\n" +
	",,,,,,Support,topic,Shop,,,,,\n"

func richMapping(h harness) imports.Mapping {
	mapping := h.mapping(map[string]string{
		string(importmap.FieldPath):            "path",
		string(importmap.FieldTitle):           "title",
		string(importmap.FieldH1):              "h1",
		string(importmap.FieldPrimaryKeyword):  "primary keyword",
		string(importmap.FieldKeywords):        "keywords",
		string(importmap.FieldAnchors):         "anchors",
		string(importmap.FieldEntity):          "entity",
		string(importmap.FieldEntityKind):      "entity kind",
		string(importmap.FieldParentEntity):    "parent",
		string(importmap.FieldRelated):         "related",
		string(importmap.FieldPageKind):        "page type",
		string(importmap.FieldMetaTitle):       "meta title",
		string(importmap.FieldMetaDescription): "meta description",
		string(importmap.FieldWPType):          "post type",
	})
	mapping.Options.KeywordSeparator = "|"
	return mapping
}

type shape struct {
	entities []string
	edges    []string
	pages    []string
}

func seedTemplates(t *testing.T, h harness) map[string]string {
	t.Helper()

	repo := sqlite.NewTemplateRepo(h.store)
	kinds := make(map[string]string)
	seeds := template.Seed()
	for i := range seeds {
		record := template.Template{
			ID: id.New(), Scope: template.ScopeGlobal, Name: seeds[i].Name, PageKind: seeds[i].PageKind,
			Version: 1, Spec: seeds[i].Spec, CreatedAt: sqlitetest.Stamp, UpdatedAt: sqlitetest.Stamp,
		}
		if err := repo.Insert(t.Context(), record); err != nil {
			t.Fatalf("insert %s: %v", record.Name, err)
		}
		kinds[record.ID] = record.PageKind
	}
	return kinds
}

func snapshot(t *testing.T, h harness, kinds map[string]string) shape {
	t.Helper()

	entities := h.entities(t)
	byID := make(map[string]graph.Entity, len(entities))
	out := shape{}
	for i := range entities {
		byID[entities[i].ID] = entities[i]
		anchors := make([]string, 0, len(entities[i].Anchors))
		for _, anchor := range entities[i].Anchors {
			anchors = append(anchors, anchor.Text)
		}
		out.entities = append(out.entities, strings.Join([]string{
			entities[i].Name, string(entities[i].Kind), entities[i].PrimaryKeyword,
			strings.Join(entities[i].SecondaryKeywords, ","), strings.Join(anchors, ","),
		}, "|"))
	}

	edges := h.edges(t)
	for i := range edges {
		ends := []string{byID[edges[i].FromEntityID].Name, byID[edges[i].ToEntityID].Name}
		if edges[i].Kind == graph.EdgeRelated {
			slices.Sort(ends)
		}
		out.edges = append(out.edges, ends[0]+"|"+ends[1]+"|"+string(edges[i].Kind))
	}

	listed := h.pages(t)
	for i := range listed {
		found := &listed[i]
		name := ""
		if found.EntityID != nil {
			name = byID[*found.EntityID].Name
		}
		canonical := ""
		if entity, ok := byID[refOfEntity(found, entities)]; ok {
			canonical = entity.Name
		}
		kind := ""
		if found.TemplateID != nil {
			kind = kinds[*found.TemplateID]
		}
		out.pages = append(out.pages, strings.Join([]string{
			found.Path, found.Title, found.H1, found.MetaTitle, found.MetaDescription,
			string(found.WPType), string(found.Status), name, canonical, kind,
		}, "|"))
	}

	slices.Sort(out.entities)
	slices.Sort(out.edges)
	slices.Sort(out.pages)
	return out
}

func refOfEntity(page *pagemap.Page, entities []graph.Entity) string {
	for i := range entities {
		if entities[i].CanonicalPageID != nil && *entities[i].CanonicalPageID == page.ID {
			return entities[i].ID
		}
	}
	return ""
}

func TestExportAndImportRoundTripTheWholeSite(t *testing.T) {
	t.Parallel()

	source := newHarness(t)
	sourceKinds := seedTemplates(t, source)
	source.apply(t, source.file(t, "rich.csv", richSheet), richMapping(source))

	exported := filepath.Join(source.dir, "export.xlsx")
	got, err := source.service.Export(t.Context(), imports.ExportRequest{SiteID: source.siteID, Path: exported})
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	if got.Pages != 3 || got.Entities != 4 {
		t.Fatalf("export = %+v", got)
	}

	target := newHarness(t)
	targetKinds := seedTemplates(t, target)
	detected, err := target.service.Inspect(t.Context(), imports.InspectRequest{SiteID: target.siteID, Path: exported})
	if err != nil {
		t.Fatalf("Inspect the export: %v", err)
	}
	if len(detected.Detected.Columns) != len(importmap.Fields()) {
		t.Fatalf("the export is not fully auto-detected: %+v", detected.Detected.Columns)
	}

	applied := target.apply(t, exported, detected.Detected)
	if len(applied.Report.Errors) != 0 {
		t.Fatalf("errors = %+v", applied.Report.Errors)
	}
	if len(applied.Report.Warnings) != 0 {
		t.Fatalf("warnings = %+v", applied.Report.Warnings)
	}

	before, after := snapshot(t, source, sourceKinds), snapshot(t, target, targetKinds)
	if !slices.Equal(before.entities, after.entities) {
		t.Fatalf("entities\n before %v\n after  %v", before.entities, after.entities)
	}
	if !slices.Equal(before.edges, after.edges) {
		t.Fatalf("edges\n before %v\n after  %v", before.edges, after.edges)
	}
	if !slices.Equal(before.pages, after.pages) {
		t.Fatalf("pages\n before %v\n after  %v", before.pages, after.pages)
	}
}

func TestExportRefusesWhatItCannotWrite(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	if _, err := h.service.Export(t.Context(), imports.ExportRequest{SiteID: h.siteID, Path: filepath.Join(h.dir, "export.csv")}); !errors.IsCode(err, errors.Invalid) {
		t.Fatalf("Export to a csv = %v, want an invalid error", err)
	}
	if _, err := h.service.Export(t.Context(), imports.ExportRequest{Path: filepath.Join(h.dir, "export.xlsx")}); !errors.IsCode(err, errors.Invalid) {
		t.Fatalf("Export without a site = %v, want an invalid error", err)
	}
}

func TestTheClientSamplesStillImport(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		detected []importmap.Field
		pages    int
		entities int
		edges    int
	}{
		{
			name:     "sitemap-import-example.csv",
			detected: []importmap.Field{importmap.FieldPath, importmap.FieldTitle, importmap.FieldKeywords},
			pages:    10,
		},
		{
			name: "sitemap-import-example.xlsx",
			detected: []importmap.Field{
				importmap.FieldPath, importmap.FieldTitle, importmap.FieldKeywords, importmap.FieldPrimaryKeyword,
				importmap.FieldEntity, importmap.FieldParentEntity, importmap.FieldAnchors, importmap.FieldPageKind,
			},
			pages:    8,
			entities: 8,
			edges:    7,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			h := newHarness(t)
			path := filepath.Join("..", "..", "..", "examples", tc.name)

			seen, err := h.service.Inspect(t.Context(), imports.InspectRequest{SiteID: h.siteID, Path: path})
			if err != nil {
				t.Fatalf("Inspect: %v", err)
			}
			for _, field := range tc.detected {
				if seen.Detected.Columns[string(field)] == "" {
					t.Fatalf("%s is not detected in %v", field, seen.Headers)
				}
			}

			applied := h.apply(t, path, seen.Detected)
			if len(applied.Report.Errors) != 0 {
				t.Fatalf("errors = %+v", applied.Report.Errors)
			}
			if applied.Counts.PagesCreated < tc.pages {
				t.Fatalf("pages created = %d, want at least %d", applied.Counts.PagesCreated, tc.pages)
			}
			if applied.Counts.EntitiesCreated != tc.entities {
				t.Fatalf("entities created = %d, want %d", applied.Counts.EntitiesCreated, tc.entities)
			}
			if applied.Counts.EdgesCreated != tc.edges {
				t.Fatalf("edges created = %d, want %d", applied.Counts.EdgesCreated, tc.edges)
			}
		})
	}
}

func TestThePageKindPicksTheSiteTemplateOverTheGlobalOne(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	seed := template.Seed()[3]
	repo := sqlite.NewTemplateRepo(h.store)
	global := template.Template{
		ID: id.New(), Scope: template.ScopeGlobal, Name: "global hub", PageKind: seed.PageKind,
		Version: 1, Spec: seed.Spec, CreatedAt: sqlitetest.Stamp, UpdatedAt: sqlitetest.Stamp,
	}
	owned := template.Template{
		ID: id.New(), Scope: template.ScopeSite, SiteID: &h.siteID, Name: "site hub", PageKind: seed.PageKind,
		Version: 1, Spec: seed.Spec, CreatedAt: sqlitetest.Stamp, UpdatedAt: sqlitetest.Stamp,
	}
	for _, record := range []template.Template{global, owned} {
		if err := repo.Insert(t.Context(), record); err != nil {
			t.Fatalf("insert %s: %v", record.Name, err)
		}
	}

	mapping := h.mapping(map[string]string{string(importmap.FieldPath): "path", string(importmap.FieldPageKind): "page type"})
	got := h.apply(t, h.file(t, "kinds.csv", "path,page type\n/,"+seed.PageKind+"\n"), mapping)
	if len(got.Report.Warnings) != 0 {
		t.Fatalf("warnings = %+v", got.Report.Warnings)
	}

	pages := h.pages(t)
	if len(pages) != 1 || pages[0].TemplateID == nil || *pages[0].TemplateID != owned.ID {
		t.Fatalf("pages = %+v, want the site template", pages)
	}

	exported := filepath.Join(h.dir, "export.xlsx")
	if _, err := h.service.Export(t.Context(), imports.ExportRequest{SiteID: h.siteID, Path: exported}); err != nil {
		t.Fatalf("Export: %v", err)
	}
	table, err := importer.Read(t.Context(), exported, importer.ReadOptions{})
	if err != nil {
		t.Fatalf("Read the export: %v", err)
	}
	binding, err := importmap.AutoDetect(table.Headers).Bind(table.Headers)
	if err != nil {
		t.Fatalf("Bind: %v", err)
	}
	if got := binding.Text(table.Rows[0], importmap.FieldPageKind); got != seed.PageKind {
		t.Fatalf("the exported page kind = %q, want %q", got, seed.PageKind)
	}
}

func TestExportWritesTheFormatItIsAsked(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		file   string
		format string
		want   string
	}{
		{name: "a workbook by default", file: "export.xlsx", want: "xlsx"},
		{name: "a workbook asked for", file: "export.xlsx", format: "xlsx", want: "xlsx"},
		{name: "separated values", file: "export.csv", format: "csv", want: "csv"},
		{name: "the case does not matter", file: "export.csv", format: "CSV", want: "csv"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			h := newHarness(t)
			h.apply(t, h.file(t, "rich.csv", richSheet), richMapping(h))

			path := filepath.Join(h.dir, tc.file)
			got, err := h.service.Export(t.Context(), imports.ExportRequest{
				SiteID: h.siteID, Path: path, Format: tc.format,
			})
			if err != nil {
				t.Fatalf("Export: %v", err)
			}
			if got.Format != tc.want {
				t.Fatalf("Format = %q, want %q", got.Format, tc.want)
			}

			detected, err := h.service.Inspect(t.Context(), imports.InspectRequest{SiteID: h.siteID, Path: path})
			if err != nil {
				t.Fatalf("Inspect what was written: %v", err)
			}
			if len(detected.Detected.Columns) != len(importmap.Fields()) {
				t.Fatalf("the export is not fully auto-detected: %+v", detected.Detected.Columns)
			}
		})
	}
}

func TestExportRefusesAFormatItDoesNotWrite(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		file   string
		format string
	}{
		{name: "an unknown format", file: "export.xlsx", format: "json"},
		{name: "a csv asked for a workbook name", file: "export.xlsx", format: "csv"},
		{name: "a workbook asked for a csv name", file: "export.csv", format: "xlsx"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			h := newHarness(t)
			_, err := h.service.Export(t.Context(), imports.ExportRequest{
				SiteID: h.siteID, Path: filepath.Join(h.dir, tc.file), Format: tc.format,
			})
			if !errors.IsCode(err, errors.Invalid) {
				t.Fatalf("Export = %v, want INVALID", err)
			}
		})
	}
}
