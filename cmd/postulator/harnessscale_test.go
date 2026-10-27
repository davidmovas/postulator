//go:build uiharness

package main

import (
	"slices"
	"strconv"
	"testing"

	"github.com/davidmovas/postulator/internal/application/findings"
	"github.com/davidmovas/postulator/internal/application/graph"
	"github.com/davidmovas/postulator/internal/application/pages"
	"github.com/davidmovas/postulator/internal/application/pagetypes"
	"github.com/davidmovas/postulator/internal/application/sites"
	"github.com/davidmovas/postulator/internal/domain/finding"
	"github.com/davidmovas/postulator/internal/domain/pagetype"
)

func TestTheHarnessSeedsALargeSiteTheRulesType(t *testing.T) {
	const products = 94
	fixed := len(scaleShape(0))
	t.Setenv(scaleVariable, strconv.Itoa(fixed+products))
	core := seeded(t)

	listed, err := core.Sites.List(t.Context(), seededSites)
	if err != nil {
		t.Fatalf("List sites: %v", err)
	}
	large := slices.IndexFunc(listed.Items, func(held sites.Site) bool { return held.Name == scaleSiteName })
	if len(listed.Items) != 3 || listed.Items[0].Name != siteName || large < 0 {
		t.Fatalf("seeded sites = %+v, want the large site beside the other two and %s still first", listed.Items, siteName)
	}
	siteID := listed.Items[large].ID

	preview, err := core.Classification.Preview(t.Context(), pagetypes.PreviewRequest{SiteID: siteID})
	if err != nil || preview.Moving != 0 || preview.Waiting != 0 || preview.ByDefault != len(scaleLeftovers) {
		t.Fatalf("the preview after the seed = %+v %v, want nothing left to move and the leftovers by default", preview, err)
	}
	types, err := core.PageTypes.List(t.Context(), pagetypes.ListRequest{SiteID: siteID})
	if err != nil {
		t.Fatalf("List page types: %v", err)
	}
	want := map[pagetype.Preset]int{
		pagetype.Landing: 1 + len(scaleDepartments), pagetype.Technical: 3, pagetype.Catalog: 1,
		pagetype.Category: len(scaleDepartments) * len(scaleShapes), pagetype.StoreCategory: len(scaleDepartments) * len(scaleShapes),
		pagetype.Guide: 1 + scaleGuides, pagetype.Info: len(scaleInfo) + len(scaleLeftovers), pagetype.Product: products,
	}
	total := 0
	for i := range types.Types {
		kind := &types.Types[i]
		counted, countErr := core.Pages.Count(t.Context(), pages.CountRequest{SiteID: siteID, PageFilter: pages.PageFilter{PageTypeID: kind.ID}})
		if countErr != nil {
			t.Fatalf("Count %s: %v", kind.Name, countErr)
		}
		if counted.Pages != want[pagetype.Preset(kind.Preset)] {
			t.Errorf("%s holds %d pages, want %d", kind.Name, counted.Pages, want[pagetype.Preset(kind.Preset)])
		}
		total += counted.Pages
	}
	if total != fixed+products {
		t.Errorf("the types hold %d pages, want every one of the %d", total, fixed+products)
	}

	proposed, err := core.Graph.PreviewFromPages(t.Context(), graph.PreviewFromPagesRequest{SiteID: siteID, Source: graph.SourceSite})
	if err != nil {
		t.Fatalf("PreviewFromPages: %v", err)
	}
	owned := (products + productUnowned - 1) / productUnowned
	categories := (len(scaleDepartments)*len(scaleShapes) + categoryUnowned - 1) / categoryUnowned
	wanted := scaleGuides - scaleGuides/guideKept + owned + categories + len(scaleInfo) + len(scaleLeftovers)
	if len(proposed.Entities) != wanted {
		t.Errorf("%d proposals, want one for each of the %d writable pages without an entity", len(proposed.Entities), wanted)
	}
	counted, err := core.Findings.Counts(t.Context(), findings.CountsRequest{SiteID: siteID})
	if err != nil {
		t.Fatalf("Counts: %v", err)
	}
	flagged := slices.IndexFunc(counted.Codes, func(code findings.CodeCount) bool { return code.Code == string(finding.PageWithoutEntity) })
	if flagged < 0 || counted.Codes[flagged].Count != wanted {
		t.Errorf("the findings by code = %+v, want a page_without_entity for each of the %d pages the dialog proposes for", counted.Codes, wanted)
	}
	sources := map[string]bool{}
	for i := range proposed.Entities {
		sources[proposed.PageTypes[proposed.Entities[i].PageID]] = true
	}
	if len(sources) != 4 || sources[""] {
		t.Errorf("the proposals come from %d page types (%v), want Guide, Product, Category and Info", len(sources), sources)
	}
}
