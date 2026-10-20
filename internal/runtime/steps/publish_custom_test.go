package steps_test

import (
	"context"
	"encoding/json"
	"slices"
	"testing"

	"github.com/davidmovas/postulator/internal/adapters/wp/wptest"
	"github.com/davidmovas/postulator/internal/domain/block"
	"github.com/davidmovas/postulator/internal/domain/content"
	"github.com/davidmovas/postulator/internal/domain/field"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/pagetype"
	"github.com/davidmovas/postulator/internal/domain/run"
	"github.com/davidmovas/postulator/internal/runtime/steps"
)

var dosageNote = field.Instance(field.Meta, "dosage_note")

func TestACustomFieldIsWrittenThroughThePluginAndPutBack(t *testing.T) {
	t.Parallel()

	stand := newRevertStand(t)
	deps := stand.deps
	deps.Types = customTypes{provider: field.ProviderPlugin}
	seeded := stand.server.Seed(wptest.Item{Type: wptest.TypePost, Title: "Espresso", Slug: "espresso-post", Status: "publish", Content: oursBody,
		Meta: map[string]string{"origin": "The owner's"}})
	sc := rewritten(publishContext(t))
	sc.Page.WPType, sc.Page.WPID = pagemap.WPPost, &seeded[0].ID
	withFields(t, sc, steps.FieldValue{Key: string(dosageNote), Value: "Once a day, with water."})

	published := runPublish(t, deps, sc)
	stored, _ := stand.server.Lookup(seeded[0].ID)
	if stored.Meta["dosage_note"] != "Once a day, with water." || stored.Meta["origin"] != "The owner's" {
		t.Fatalf("the post's meta = %v, want the custom field written and the owner's other keys left", stored.Meta)
	}
	want := steps.FieldWrite{Key: dosageNote, Before: "", Written: "Once a day, with water.", Provider: field.ProviderPlugin}
	if len(published.PreviousFields) != 1 || published.PreviousFields[0] != want {
		t.Fatalf("the publish kept %+v, want %+v", published.PreviousFields, want)
	}

	page := pagemap.Page{ID: "page-post", SiteID: "site", Path: "/espresso-post/", Slug: "espresso-post", WPType: pagemap.WPPost,
		Status: pagemap.StatusPublished, WPID: &seeded[0].ID}
	stand.pages.items[page.ID] = page
	stand.wpIDs[page.ID] = seeded[0].ID
	stand.source.record(t, page.ID, map[run.ArtifactKind]any{run.ArtifactPublishResult: published})
	reverted, _ := stand.revert(t, page.ID)
	if reverted.Outcome != steps.OutcomeRestored {
		t.Fatalf("reverted = %+v", reverted)
	}
	if back, _ := stand.server.Lookup(seeded[0].ID); back.Meta["dosage_note"] != "" || back.Meta["origin"] != "The owner's" {
		t.Errorf("after the revert the meta is %v, want the run's key gone and the owner's kept", back.Meta)
	}

	edited := stand.server.Seed(wptest.Item{Type: wptest.TypePost, Title: "Ristretto", Slug: "ristretto-post", Status: "publish", Content: oursBody})
	sc.Page.WPID, sc.Page.Slug = &edited[0].ID, "ristretto-post"
	again := runPublish(t, deps, sc)
	held, _ := stand.server.Lookup(edited[0].ID)
	held.Meta = map[string]string{"dosage_note": "The owner changed it after the run."}
	stand.server.Restore(held)
	second := pagemap.Page{ID: "page-ristretto-post", SiteID: "site", Path: "/ristretto-post/", Slug: "ristretto-post", WPType: pagemap.WPPost,
		Status: pagemap.StatusPublished, WPID: &edited[0].ID}
	stand.pages.items[second.ID] = second
	stand.wpIDs[second.ID] = edited[0].ID
	stand.source.record(t, second.ID, map[run.ArtifactKind]any{run.ArtifactPublishResult: again})
	refused, _ := stand.revert(t, second.ID)
	if _, found := findingOf(refused, steps.CodeRevertFieldKept); !found {
		t.Errorf("a revert over a custom field the owner edited = %+v, want it kept and said", refused)
	}
	if back, _ := stand.server.Lookup(edited[0].ID); back.Meta["dosage_note"] != "The owner changed it after the run." {
		t.Errorf("the owner's later value became %q", back.Meta["dosage_note"])
	}
}

func TestAProductsCustomFieldIsWrittenThroughTheStoreAndPutBack(t *testing.T) {
	t.Parallel()

	h := newProductHarness(t, storeProduct())
	h.deps.Types = customTypes{provider: field.ProviderStore}
	sc := storeContext(t, h.held.ID)
	withFields(t, sc, steps.FieldValue{Key: string(dosageNote), Value: "Once a day, with water."})
	published := runPublish(t, h.deps, sc)

	stored, _ := h.server.Lookup(h.held.ID)
	if value := metaDataOf(stored, "dosage_note"); value != "Once a day, with water." {
		t.Fatalf("the product's meta data holds %q, want the custom field written through the store", value)
	}
	if stored.RegularPrice != "120" || stored.SKU != "EM-1" {
		t.Errorf("the store's own fields moved: %+v", stored)
	}

	page := *h.recorded
	source := newSourceRecord()
	source.record(t, page.ID, map[run.ArtifactKind]any{run.ArtifactPublishResult: published})
	h.deps.Pages, h.deps.Links, h.deps.Items, h.deps.Artifacts = newPageMap(page), &linkRecorder{}, source, source
	reverted, result := runRevert(t, h.deps, "page-child")
	if result.Next == run.TransitionPause || reverted.Outcome != steps.OutcomeRestored {
		t.Fatalf("the revert answered %+v / %s", reverted, result.Message)
	}
	if back, _ := h.server.Lookup(h.held.ID); metaDataOf(back, "dosage_note") != "" {
		t.Errorf("after the revert the custom field holds %q, want what it held before", metaDataOf(back, "dosage_note"))
	}
}

func TestAProductsUpsellsAreWrittenAsIDsAndPutBack(t *testing.T) {
	t.Parallel()

	item := storeProduct()
	item.CrossSells = []int64{77}
	h := newProductHarness(t, item)
	sc := storeContext(t, h.held.ID)
	withFields(t, sc,
		steps.FieldValue{Key: string(field.Upsells), Value: "31,30"},
		steps.FieldValue{Key: string(field.CrossSells), Value: "30"},
	)
	h.deps.Types = listTypes{}
	published := runPublish(t, h.deps, sc)

	stored, _ := h.server.Lookup(h.held.ID)
	if !slices.Equal(stored.Upsells, []int64{31, 30}) {
		t.Errorf("upsells = %v, want the ids the run chose, in their order", stored.Upsells)
	}
	if !slices.Equal(stored.CrossSells, []int64{30}) {
		t.Errorf("cross-sells = %v, want ours: the run rewrites", stored.CrossSells)
	}
	if snapshot := published.PreviousProduct; snapshot == nil || len(snapshot.Lists) != 2 {
		t.Fatalf("the snapshot keeps %+v, want both lists before and after", published.PreviousProduct)
	}

	page := *h.recorded
	source := newSourceRecord()
	source.record(t, page.ID, map[run.ArtifactKind]any{run.ArtifactPublishResult: published})
	h.deps.Pages, h.deps.Links, h.deps.Items, h.deps.Artifacts = newPageMap(page), &linkRecorder{}, source, source
	reverted, result := runRevert(t, h.deps, "page-child")
	if result.Next == run.TransitionPause || reverted.Outcome != steps.OutcomeRestored {
		t.Fatalf("the revert answered %+v / %s", reverted, result.Message)
	}
	back, _ := h.server.Lookup(h.held.ID)
	if len(back.Upsells) != 0 || !slices.Equal(back.CrossSells, []int64{77}) {
		t.Errorf("after the revert = %v / %v, want what the product listed before", back.Upsells, back.CrossSells)
	}
}

func TestTheClientsOwnListIsKeptWithoutRewrites(t *testing.T) {
	t.Parallel()

	item := storeProduct()
	item.CrossSells = []int64{77}
	h := newProductHarness(t, item)
	h.deps.Types = listTypes{}
	sc := storeContext(t, h.held.ID)
	delete(sc.Params, run.ParamRewrite)
	var draft content.ContentDraft
	if err := json.Unmarshal(sc.Artifacts[run.ArtifactDraft].Blob, &draft); err != nil {
		t.Fatalf("decode the draft: %v", err)
	}
	draft.Kept = &content.KeptBody{Raw: "<p>old description</p>", Hash: block.BodyHash("<p>old description</p>"), Written: []string{}}
	blob, err := json.Marshal(draft)
	if err != nil {
		t.Fatalf("encode the draft: %v", err)
	}
	sc.Artifacts[run.ArtifactDraft] = run.Artifact{Kind: run.ArtifactDraft, Blob: blob}
	withFields(t, sc,
		steps.FieldValue{Key: string(field.Upsells), Value: "31"},
		steps.FieldValue{Key: string(field.CrossSells), Value: "30"},
	)
	runPublish(t, h.deps, sc)

	stored, _ := h.server.Lookup(h.held.ID)
	if !slices.Equal(stored.CrossSells, []int64{77}) {
		t.Errorf("cross-sells = %v, want the client's own list kept: the scope does not rewrite it", stored.CrossSells)
	}
	if !slices.Equal(stored.Upsells, []int64{31}) {
		t.Errorf("upsells = %v, want an empty list filled", stored.Upsells)
	}
}

type listTypes struct{}

func (listTypes) Set(_ context.Context, siteID string) (pagetype.Set, error) {
	kind := pagetype.Type{ID: "type-product", SiteID: siteID, Name: "Product", Preset: pagetype.Product}
	scopes := pagetype.DefaultScopes(pagetype.Product, kind.ID)
	for _, key := range []field.Key{field.Upsells, field.CrossSells} {
		scope := pagetype.DefaultScope(pagetype.Product, key, kind.ID)
		scope.Enabled = true
		scopes = append(scopes, scope)
	}
	return pagetype.NewSet([]pagetype.Type{kind}, scopes), nil
}

func metaDataOf(item wptest.Item, key string) string {
	for _, entry := range item.MetaData {
		if entry.Key != key {
			continue
		}
		if text, ok := entry.Value.(string); ok {
			return text
		}
		encoded, err := json.Marshal(entry.Value)
		if err != nil {
			return ""
		}
		return string(encoded)
	}
	return ""
}
