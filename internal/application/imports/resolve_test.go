package imports_test

import (
	"reflect"
	"testing"

	"github.com/davidmovas/postulator/internal/adapters/sqlite/sqlitetest"
	"github.com/davidmovas/postulator/internal/application/imports"
	"github.com/davidmovas/postulator/internal/domain/importmap"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

func TestAMappingIsResolvedBeforeTheSheetIsPlanned(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	saved, err := h.service.SaveMapping(t.Context(), imports.SaveMappingRequest{Mapping: graphMapping(h)})
	if err != nil {
		t.Fatalf("SaveMapping: %v", err)
	}
	graph := h.file(t, "graph.csv", graphSheet)
	client := h.file(t, "client.csv", clientSheet)
	products := h.file(t, "products.csv", productSheet)
	asProducts := productMapping(h, importmap.RowProducts)

	cases := []struct {
		name  string
		path  string
		given imports.Mapping
		want  imports.Mapping
	}{
		{
			name:  "no columns are detected from the headers",
			path:  client,
			given: imports.Mapping{},
			want:  h.detected(t, client),
		},
		{
			name:  "category columns alone map nothing and are detected from the headers",
			path:  client,
			given: imports.Mapping{Options: imports.Options{LevelColumns: []string{"Category", "Subcategory"}}},
			want:  h.detected(t, client),
		},
		{
			name:  "detection keeps the options it was given",
			path:  products,
			given: imports.Mapping{Options: imports.Options{RowType: importmap.RowProducts}},
			want:  asProducts,
		},
		{
			name:  "an id and no columns read the saved mapping",
			path:  graph,
			given: imports.Mapping{ID: saved.Mapping.ID},
			want:  graphMapping(h),
		},
		{
			name:  "an id with columns keeps the columns it was given",
			path:  graph,
			given: func() imports.Mapping { m := graphMapping(h); m.ID = saved.Mapping.ID; return m }(),
			want:  graphMapping(h),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := h.preview(t, tc.path, tc.given)
			want := h.preview(t, tc.path, tc.want)
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("resolved to\n%+v\nwant\n%+v", got, want)
			}
		})
	}
}

func TestApplySavesTheMappingItResolved(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	path := h.file(t, "client.csv", clientSheet)
	if _, err := h.service.Apply(t.Context(), imports.ApplyRequest{
		SiteID: h.siteID, Path: path, Options: imports.ApplyOptions{SaveMappingAs: "detected"},
	}); err != nil {
		t.Fatalf("Apply: %v", err)
	}

	listed, err := h.service.ListMappings(t.Context(), imports.ListMappingsRequest{SiteID: h.siteID})
	if err != nil {
		t.Fatalf("ListMappings: %v", err)
	}
	want := h.detected(t, path)
	if len(listed.Mappings) != 1 || !reflect.DeepEqual(listed.Mappings[0].Columns, want.Columns) ||
		!reflect.DeepEqual(listed.Mappings[0].Options.LevelColumns, want.Options.LevelColumns) {
		t.Fatalf("saved = %+v, want the detected columns and levels", listed.Mappings)
	}
}

func TestASavedMappingTakesTheRowTypeAndSheetsItIsGiven(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	h.storeProduct(t, "mak-liquid", "Mak Liquid", 501)
	saved, err := h.service.SaveMapping(t.Context(), imports.SaveMappingRequest{Mapping: productMapping(h, importmap.RowPages)})
	if err != nil {
		t.Fatalf("SaveMapping: %v", err)
	}
	path := h.file(t, "products.csv", productSheet)

	report := h.preview(t, path, imports.Mapping{ID: saved.Mapping.ID, Options: imports.Options{RowType: importmap.RowProducts}})
	if found, _ := page(report, "/product/mak-liquid/"); found.MatchedBy == "" {
		t.Fatalf("pages = %+v, want the saved mapping read as products", report.Pages)
	}
}

func TestASavedMappingOfAnotherSiteOrNoneIsRefused(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	other := sqlitetest.Site(t, h.store, "other")
	elsewhere := graphMapping(h)
	elsewhere.SiteID = other.ID
	saved, err := h.service.SaveMapping(t.Context(), imports.SaveMappingRequest{Mapping: elsewhere})
	if err != nil {
		t.Fatalf("SaveMapping: %v", err)
	}
	path := h.file(t, "graph.csv", graphSheet)

	for name, given := range map[string]string{
		"another site's mapping": saved.Mapping.ID,
		"a mapping nobody saved": "00000000-0000-4000-8000-000000000000",
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			_, err := h.service.Preview(t.Context(), imports.PreviewRequest{SiteID: h.siteID, Path: path, Mapping: imports.Mapping{ID: given}})
			if !errors.IsCode(err, errors.NotFound) {
				t.Fatalf("Preview = %v, want not found", err)
			}
		})
	}
}
