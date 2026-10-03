package steps_test

import (
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/davidmovas/postulator/internal/adapters/wp"
	"github.com/davidmovas/postulator/internal/adapters/wp/wptest"
	"github.com/davidmovas/postulator/internal/domain/run"
	"github.com/davidmovas/postulator/internal/runtime/steps"
)

type publishedProduct struct {
	productHarness
	published steps.PublishResult
	pages     *pageMap
}

func publishAndRecord(t *testing.T, item wptest.Item, opts ...wptest.Option) publishedProduct {
	t.Helper()

	h := newProductHarness(t, item, opts...)
	sc := storeContext(t, h.held.ID)
	sc.Artifacts[run.ArtifactImages] = run.Artifact{Kind: run.ArtifactImages, Blob: []byte(`{"featuredId":77}`)}
	return recordedProduct(t, h, runPublish(t, h.deps, sc))
}

func recordedProduct(t *testing.T, h productHarness, published steps.PublishResult) publishedProduct {
	t.Helper()

	page := *h.recorded
	pages := newPageMap(page)
	source := newSourceRecord()
	source.record(t, page.ID, map[run.ArtifactKind]any{run.ArtifactPublishResult: published})
	h.deps.Pages = pages
	h.deps.Links = &linkRecorder{}
	h.deps.Items = source
	h.deps.Artifacts = source
	return publishedProduct{productHarness: h, published: published, pages: pages}
}

func TestRevertPutsAProductBackAsTheClientHadIt(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name        string
		description string
		images      []int64
		wantImages  []int64
	}{
		{name: "a product the run filled in", description: "<p>old description</p>", images: []int64{}, wantImages: []int64{}},
		{name: "a product that had no description", description: "", images: []int64{}, wantImages: []int64{}},
		{name: "a product that kept its own image", description: "<p>old description</p>", images: []int64{5}, wantImages: []int64{5}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			item := storeProduct()
			item.Content = tc.description
			item.Images = tc.images
			p := publishAndRecord(t, item)

			reverted, result := runRevert(t, p.deps, "page-child")
			if result.Next == run.TransitionPause || reverted.Outcome != steps.OutcomeRestored {
				t.Fatalf("the revert answered %+v / %s", reverted, result.Message)
			}

			stored, _ := p.server.Lookup(p.held.ID)
			if stored.Content != tc.description {
				t.Errorf("description = %q, want %q", stored.Content, tc.description)
			}
			if stored.Excerpt != "<p>old short</p>" {
				t.Errorf("short description = %q, want the one the run replaced", stored.Excerpt)
			}
			if !reflect.DeepEqual(stored.Attributes, storeProduct().Attributes) {
				t.Errorf("attributes = %+v, want %+v", stored.Attributes, storeProduct().Attributes)
			}
			if !slices.Equal(stored.Images, tc.wantImages) {
				t.Errorf("images = %v, want %v", stored.Images, tc.wantImages)
			}
			if stored.Title != "Espresso Machine" || stored.RegularPrice != "120" || stored.Status != "publish" {
				t.Errorf("the revert moved the store's own fields: %+v", stored)
			}
			if stored.Meta["_yoast_wpseo_title"] != "" {
				t.Errorf("meta = %v, want the run's title taken back", stored.Meta)
			}
			if page := p.pages.items["page-child"]; page.ContentHash != wp.ContentHash(tc.description) || page.Drift {
				t.Errorf("the page records %+v", page)
			}
		})
	}
}

func TestRevertHandsBackAProductAHumanChangedSince(t *testing.T) {
	t.Parallel()

	short := "<p>a human's short description</p>"
	wall := []wp.ProductAttribute{
		{Name: "Origin", Options: []string{"Italy"}, Visible: true},
		{Name: "form", Options: []string{"Wall"}, Position: 1},
		{ID: 3, Name: "Size", Options: []string{}, Position: 2, Visible: true},
	}
	cases := []struct {
		name   string
		change func(*testing.T, publishedProduct)
		reason string
	}{
		{
			name: "the short description",
			change: func(t *testing.T, p publishedProduct) {
				if _, err := syncClient(t, p.server).UpdateProduct(t.Context(), p.held.ID, wp.UpdateProduct{ShortDescription: &short}); err != nil {
					t.Fatalf("edit the short description: %v", err)
				}
			},
			reason: "short description",
		},
		{
			name: "an attribute the run filled",
			change: func(t *testing.T, p publishedProduct) {
				if _, err := syncClient(t, p.server).UpdateProduct(t.Context(), p.held.ID, wp.UpdateProduct{Attributes: &wall}); err != nil {
					t.Fatalf("edit the attribute: %v", err)
				}
			},
			reason: "form",
		},
		{
			name: "the description",
			change: func(t *testing.T, p publishedProduct) {
				if !p.server.Rewrite(p.held.ID, "<p>a human's description</p>") {
					t.Fatal("the product could not be rewritten")
				}
			},
			reason: "edited on the site",
		},
		{
			name: "the store, which no longer holds it",
			change: func(t *testing.T, p publishedProduct) {
				if !p.server.Delete(p.held.ID) {
					t.Fatal("the product could not be removed")
				}
			},
			reason: steps.ReasonRevertGone,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			p := publishAndRecord(t, storeProduct())
			tc.change(t, p)
			before, _ := p.server.Lookup(p.held.ID)

			reverted, result := runRevert(t, p.deps, "page-child")
			if result.Next != run.TransitionPause || reverted.Outcome != steps.OutcomeNeedsHand {
				t.Fatalf("the revert answered %+v, want it handed to a human", reverted)
			}
			if !strings.Contains(reverted.Detail, tc.reason) {
				t.Errorf("detail = %q, want it to name %q", reverted.Detail, tc.reason)
			}
			if after, held := p.server.Lookup(p.held.ID); held && !reflect.DeepEqual(after, before) {
				t.Errorf("the refused revert wrote to the product: %+v, was %+v", after, before)
			}
		})
	}
}

func TestRevertLeavesAnAttributeAHumanRemovedSince(t *testing.T) {
	t.Parallel()

	p := publishAndRecord(t, storeProduct())
	kept := []wp.ProductAttribute{
		{Name: "Origin", Options: []string{"Italy"}, Visible: true},
		{ID: 3, Name: "Size", Options: []string{}, Position: 2, Visible: true},
	}
	if _, err := syncClient(t, p.server).UpdateProduct(t.Context(), p.held.ID, wp.UpdateProduct{Attributes: &kept}); err != nil {
		t.Fatalf("remove the attribute: %v", err)
	}

	reverted, result := runRevert(t, p.deps, "page-child")
	if result.Next == run.TransitionPause || reverted.Outcome != steps.OutcomeRestored {
		t.Fatalf("the revert answered %+v / %s", reverted, result.Message)
	}
	stored, _ := p.server.Lookup(p.held.ID)
	want := []wptest.Attribute{
		{Name: "Origin", Options: []string{"Italy"}, Visible: true},
		{ID: 3, Name: "Size", Options: []string{}, Position: 2, Visible: true},
	}
	if !reflect.DeepEqual(stored.Attributes, want) {
		t.Errorf("attributes = %+v, want the human's removal kept", stored.Attributes)
	}
	if stored.Content != "<p>old description</p>" || stored.Excerpt != "<p>old short</p>" {
		t.Errorf("the product holds %q / %q", stored.Content, stored.Excerpt)
	}
}
