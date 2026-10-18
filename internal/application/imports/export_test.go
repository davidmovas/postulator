package imports_test

import (
	"maps"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/davidmovas/postulator/internal/adapters/importer"
	"github.com/davidmovas/postulator/internal/application/imports"
	"github.com/davidmovas/postulator/internal/domain/graph"
	"github.com/davidmovas/postulator/internal/domain/importmap"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/pagetype"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

const richSheet = "path,title,h1,primary keyword,keywords,anchors,entity,page type,parent,related,meta title,meta description,post type\n" +
	"/,Home,Welcome,shop,shop|store,our shop,Shop,hub,,,Shop,The shop,page\n" +
	"/hosting/,Hosting,Hosting plans,hosting,vps|servers,hosting plans|cheap hosting,Hosting,Category,Shop,Domains,Hosting,Hosting plans,page\n" +
	"/domains/,Domains,Domains,domains,tld|registrar,buy a domain,Domains,guide,Shop,Hosting,Domains,Buy a domain,page\n" +
	",,,,,,Support,topic,Shop,,,,\n"

func richMapping(h harness) imports.Mapping {
	mapping := h.mapping(map[string]string{
		string(importmap.FieldPath):            "path",
		string(importmap.FieldTitle):           "title",
		string(importmap.FieldH1):              "h1",
		string(importmap.FieldPrimaryKeyword):  "primary keyword",
		string(importmap.FieldKeywords):        "keywords",
		string(importmap.FieldAnchors):         "anchors",
		string(importmap.FieldEntity):          "entity",
		string(importmap.FieldPageType):        "page type",
		string(importmap.FieldParentEntity):    "parent",
		string(importmap.FieldRelated):         "related",
		string(importmap.FieldMetaTitle):       "meta title",
		string(importmap.FieldMetaDescription): "meta description",
		string(importmap.FieldWPType):          "post type",
	})
	return mapping
}

type shape struct {
	entities []string
	edges    []string
	pages    []string
}

func snapshot(t *testing.T, h harness) shape {
	t.Helper()

	entities := h.entities(t)
	typed := h.typeNames(t)
	byID := make(map[string]graph.Entity, len(entities))
	out := shape{}
	for i := range entities {
		byID[entities[i].ID] = entities[i]
		anchors := make([]string, 0, len(entities[i].Anchors))
		for _, anchor := range entities[i].Anchors {
			anchors = append(anchors, anchor.Text)
		}
		out.entities = append(out.entities, strings.Join([]string{
			entities[i].Name, entities[i].Keywords.Main(),
			strings.Join(entities[i].Keywords.Rest(), ","), strings.Join(anchors, ","),
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
		out.pages = append(out.pages, strings.Join([]string{
			found.Path, found.Title, found.H1, found.MetaTitle, found.MetaDescription,
			string(found.WPType), string(found.Status), name, canonical, typed[found.Path],
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
	source.apply(t, source.file(t, "rich.csv", richSheet), richMapping(source))
	if typed := source.typeNames(t); !maps.Equal(typed, map[string]string{"/hosting/": "Hub", "/domains/": "Entity"}) {
		t.Fatalf("the source typed its pages %v", typed)
	}

	exported := filepath.Join(source.dir, "export.xlsx")
	got, err := source.service.Export(t.Context(), imports.ExportRequest{SiteID: source.siteID, Path: exported})
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	if got.Pages != 2 || got.Entities != 4 {
		t.Fatalf("export = %+v", got)
	}

	target := newHarness(t)
	detected, err := target.service.Inspect(t.Context(), imports.InspectRequest{SiteID: target.siteID, Path: exported})
	if err != nil {
		t.Fatalf("Inspect the export: %v", err)
	}
	if len(detected.Detected.Columns) != len(detected.Headers) {
		t.Fatalf("the export is not fully auto-detected: %+v from %v", detected.Detected.Columns, detected.Headers)
	}
	if _, primary := detected.Detected.Columns[string(importmap.FieldPrimaryKeyword)]; primary {
		t.Fatalf("the export writes the keywords in two columns: %v", detected.Headers)
	}

	applied := target.apply(t, exported, detected.Detected)
	if len(applied.Report.Errors) != 0 {
		t.Fatalf("errors = %+v", applied.Report.Errors)
	}
	if len(applied.Report.Warnings) != 0 {
		t.Fatalf("warnings = %+v", applied.Report.Warnings)
	}

	before, after := snapshot(t, source), snapshot(t, target)
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

func TestExportWritesTheKeywordsWithTheirVolumesInOneColumn(t *testing.T) {
	t.Parallel()

	source := newHarness(t)
	mapping := source.mapping(map[string]string{
		string(importmap.FieldPath): "url", string(importmap.FieldTitle): "title", string(importmap.FieldKeywords): "keywords",
	})
	source.apply(t, source.file(t, "volumes.csv",
		"url,title,keywords\n/bpc-157/,BPC-157,\"bpc 157 (12000), buy bpc 157 (5,400), bpc-157\"\n"), mapping)

	exported := filepath.Join(source.dir, "export.xlsx")
	if _, err := source.service.Export(t.Context(), imports.ExportRequest{SiteID: source.siteID, Path: exported}); err != nil {
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
	if got := binding.Text(table.Rows[0], importmap.FieldKeywords); got != "bpc 157 (12000), buy bpc 157 (5400), bpc-157" {
		t.Fatalf("the keywords cell = %q", got)
	}

	target := newHarness(t)
	detected, err := target.service.Inspect(t.Context(), imports.InspectRequest{SiteID: target.siteID, Path: exported})
	if err != nil {
		t.Fatalf("Inspect: %v", err)
	}
	target.apply(t, exported, detected.Detected)
	if before, after := source.pages(t)[0].Keywords, target.pages(t)[0].Keywords; !after.Equal(before) {
		t.Fatalf("keywords after the round trip = %+v, want %+v", after, before)
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
			entities: 18,
			edges:    12,
		},
		{
			name: "sitemap-import-example.xlsx",
			detected: []importmap.Field{
				importmap.FieldPath, importmap.FieldTitle, importmap.FieldKeywords, importmap.FieldPrimaryKeyword,
				importmap.FieldEntity, importmap.FieldParentEntity, importmap.FieldAnchors, importmap.FieldPageType,
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

func TestTheExportWritesThePageTypeByName(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	h.rename(t, pagetype.Hub, "Collection")
	mapping := h.mapping(map[string]string{
		string(importmap.FieldPath): "path", string(importmap.FieldEntity): "entity", string(importmap.FieldPageType): "page type",
	})
	got := h.apply(t, h.file(t, "types.csv", "path,entity,page type\n/hub/,Hub,Pillar\n/plain/,Plain,\n,Loose,Hub\n"), mapping)
	if len(got.Report.Warnings) != 0 || len(got.Report.Errors) != 0 {
		t.Fatalf("findings = %+v %+v", got.Report.Warnings, got.Report.Errors)
	}
	for _, held := range h.pages(t) {
		if held.TemplateID != nil {
			t.Fatalf("the import gave %s the template %s, want the template left to the type", held.Path, *held.TemplateID)
		}
	}

	exported := filepath.Join(h.dir, "export.xlsx")
	if _, err := h.service.Export(t.Context(), imports.ExportRequest{SiteID: h.siteID, Path: exported}); err != nil {
		t.Fatalf("Export: %v", err)
	}
	table, err := importer.Read(t.Context(), exported, importer.ReadOptions{})
	if err != nil {
		t.Fatalf("Read the export: %v", err)
	}
	for _, gone := range []string{"entity_kind", "page_kind"} {
		if slices.Contains(table.Headers, gone) {
			t.Fatalf("the export still writes %s: %v", gone, table.Headers)
		}
	}
	if !slices.Contains(table.Headers, string(importmap.FieldPageType)) {
		t.Fatalf("headers = %v, want a page_type column", table.Headers)
	}
	binding, err := importmap.AutoDetect(table.Headers).Bind(table.Headers)
	if err != nil {
		t.Fatalf("Bind: %v", err)
	}
	cells := make(map[string]string, len(table.Rows))
	for _, row := range table.Rows {
		cells[binding.Text(row, importmap.FieldPath)+"#"+binding.Text(row, importmap.FieldEntity)] = binding.Text(row, importmap.FieldPageType)
	}
	want := map[string]string{"/hub/#Hub": "Collection", "/plain/#Plain": "", "#Loose": ""}
	if !maps.Equal(cells, want) {
		t.Fatalf("the exported page types = %v, want %v", cells, want)
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
			if len(detected.Detected.Columns) != len(detected.Headers) {
				t.Fatalf("the export is not fully auto-detected: %+v from %v", detected.Detected.Columns, detected.Headers)
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
