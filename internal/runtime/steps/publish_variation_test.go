package steps_test

import (
	"encoding/json"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/davidmovas/postulator/internal/adapters/wp/wptest"
	"github.com/davidmovas/postulator/internal/domain/field"
	"github.com/davidmovas/postulator/internal/domain/run"
	"github.com/davidmovas/postulator/internal/runtime/steps"
)

type variedProduct struct {
	publishedProduct
	small, large wptest.Item
}

func publishVariations(t *testing.T) variedProduct {
	t.Helper()

	return publishVariationsOver(t, "<p>Old small.</p>")
}

func publishVariationsOver(t *testing.T, smallText string) variedProduct {
	t.Helper()

	item := storeProduct()
	item.ProductType = "variable"
	h := newProductHarness(t, item)
	h.deps.Types = variationTypes{}
	seeded := h.server.Seed(
		wptest.Item{Type: wptest.TypeProductVariation, Parent: h.held.ID, SKU: "EM-1-S", RegularPrice: "120", Status: "publish",
			Content: smallText, Choices: []wptest.Choice{{ID: 3, Name: "Size", Option: "Small"}}},
		wptest.Item{Type: wptest.TypeProductVariation, Parent: h.held.ID, SKU: "EM-1-L", RegularPrice: "150", Status: "publish",
			Choices: []wptest.Choice{{ID: 3, Name: "Size", Option: "Large"}}},
	)
	small, large := seeded[0], seeded[1]
	sc := storeContext(t, h.held.ID)
	withFields(t, sc,
		steps.FieldValue{Key: string(field.ProductShortDescription), Value: productShort},
		steps.FieldValue{Key: string(variationKey(small)), Value: "<p>A small machine.</p>"},
		steps.FieldValue{Key: string(variationKey(large)), Value: "<p>A large machine.</p>"},
	)
	published := runPublish(t, h.deps, sc)

	page := *h.recorded
	pages := newPageMap(page)
	source := newSourceRecord()
	source.record(t, page.ID, map[run.ArtifactKind]any{run.ArtifactPublishResult: published})
	h.deps.Pages = pages
	h.deps.Links = &linkRecorder{}
	h.deps.Items = source
	h.deps.Artifacts = source
	return variedProduct{
		publishedProduct: publishedProduct{productHarness: h, published: published, pages: pages, source: source},
		small:            small, large: large,
	}
}

func variationKey(variation wptest.Item) field.Key {
	return field.Instance(field.VariationDescription, strconv.FormatInt(variation.ID, 10))
}

func TestAVariationsDescriptionIsWrittenAloneAndPutBack(t *testing.T) {
	t.Parallel()

	p := publishVariations(t)
	small, _ := p.server.Lookup(p.small.ID)
	large, _ := p.server.Lookup(p.large.ID)
	if small.Content != "<p>A small machine.</p>" || large.Content != "<p>A large machine.</p>" {
		t.Fatalf("the variations hold %q and %q, want the run's", small.Content, large.Content)
	}
	if small.SKU != "EM-1-S" || small.RegularPrice != "120" || len(small.Choices) != 1 || small.Choices[0].Option != "Small" {
		t.Errorf("the store's facts of the variation moved: %+v", small)
	}
	writes := 0
	for _, request := range p.server.Requests() {
		if request.Method != http.MethodPost || !strings.Contains(request.Path, "/variations/") {
			continue
		}
		writes++
		var body map[string]any
		if err := json.Unmarshal(request.Body, &body); err != nil {
			t.Fatalf("decode the sent body: %v", err)
		}
		if _, sent := body["description"]; !sent || len(body) != 1 {
			t.Errorf("a variation write sent %v, want the description alone: its attributes, price and stock are the store's", body)
		}
	}
	if writes != 2 {
		t.Errorf("the run sent %d variation writes, want one per variation", writes)
	}
	want := []steps.FieldWrite{
		{Key: variationKey(p.small), Before: "<p>Old small.</p>\n", Written: "<p>A small machine.</p>\n"},
		{Key: variationKey(p.large), Before: "", Written: "<p>A large machine.</p>\n"},
	}
	if got := p.published.PreviousVariations; len(got) != 2 || got[0] != want[0] || got[1] != want[1] {
		t.Errorf("the publish kept %+v, want %+v", got, want)
	}

	reverted, result := runRevert(t, p.deps, "page-child")
	if result.Next == run.TransitionPause || reverted.Outcome != steps.OutcomeRestored {
		t.Fatalf("the revert answered %+v / %s", reverted, result.Message)
	}
	small, _ = p.server.Lookup(p.small.ID)
	large, _ = p.server.Lookup(p.large.ID)
	if small.Content != "<p>Old small.</p>\n" || large.Content != "" {
		t.Errorf("after the revert the variations hold %q and %q, want what they held before", small.Content, large.Content)
	}
}

func TestARevertedVariationDescriptionInTheStoresFormIsAccepted(t *testing.T) {
	t.Parallel()

	p := publishVariationsOver(t, "<p>Don't shake the small one.</p>")
	formatted := "<p>Don&#8217;t shake the small one.</p>\n"
	if got := p.published.PreviousVariations[0].Before; got != formatted {
		t.Fatalf("the publish kept %q as the client's, want the store's formatted answer %q", got, formatted)
	}
	fields := &fieldLog{}
	p.deps.Fields = fields

	reverted, result := runRevert(t, p.deps, "page-child")
	if result.Next == run.TransitionPause || reverted.Outcome != steps.OutcomeRestored {
		t.Fatalf("the revert answered %+v / %s, want the client's text put back in the store's form", reverted, result.Message)
	}
	wordpress, ok := p.deps.WordPress.(oneClient)
	if !ok {
		t.Fatalf("the harness holds %T, want one client", p.deps.WordPress)
	}
	answered, err := wordpress.client.GetVariation(t.Context(), p.held.ID, p.small.ID)
	if err != nil {
		t.Fatalf("read the variation back: %v", err)
	}
	if !strings.Contains(answered.Description, "Don&#8217;t shake the small one.") {
		t.Errorf("the store answers %q, want the client's text in its formatted form", answered.Description)
	}
	if recorded := fields.recordedValue(variationKey(p.small)); recorded != answered.Description {
		t.Errorf("the revert recorded %q as the field's value, want what the store answers, %q, or the next walk reads the store's own formatting as the client's change",
			recorded, answered.Description)
	}
}

func TestAVariationTheClientEditedSinceIsHandedBack(t *testing.T) {
	t.Parallel()

	p := publishVariations(t)
	edited, _ := p.server.Lookup(p.large.ID)
	edited.Content = "<p>The owner's own large.</p>"
	p.server.Restore(edited)

	reverted, _ := runRevert(t, p.deps, "page-child")
	if large, _ := p.server.Lookup(p.large.ID); large.Content != "<p>The owner's own large.</p>" {
		t.Errorf("the owner's later variation description became %q", large.Content)
	}
	if small, _ := p.server.Lookup(p.small.ID); small.Content != "<p>Old small.</p>\n" {
		t.Errorf("the variation nobody touched since holds %q, want it put back: each field reverts on its own", small.Content)
	}
	kept, found := findingOf(reverted, steps.CodeRevertFieldKept)
	if !found || !strings.Contains(kept.Message, string(variationKey(p.large))) {
		t.Errorf("findings = %+v, want the edited variation named and handed back", reverted.Findings)
	}
}

func TestADescriptionTheClientEditsDuringTheStoresSaveIsNotWrittenOver(t *testing.T) {
	t.Parallel()

	for _, unit := range []string{"the store's fields", "a variation"} {
		t.Run(unit, func(t *testing.T) {
			t.Parallel()

			item := storeProduct()
			item.ProductType = "variable"
			h := newProductHarness(t, item)
			h.deps.Types = variationTypes{}
			variation := h.server.Seed(wptest.Item{Type: wptest.TypeProductVariation, Parent: h.held.ID, SKU: "EM-1-S", Status: "publish",
				Content: "<p>Old small.</p>", Choices: []wptest.Choice{{ID: 3, Name: "Size", Option: "Small"}}})[0]
			sc := storeContext(t, h.held.ID)
			if unit == "a variation" {
				withFields(t, sc, steps.FieldValue{Key: string(variationKey(variation)), Value: "<p>A small machine.</p>"})
			}

			def := steps.Publish(h.deps)
			var result run.Result
			for execution := range 16 {
				if execution == 2 {
					edited, _ := h.server.Lookup(h.held.ID)
					edited.Content = "<p>The owner's description, edited a moment ago.</p>"
					h.server.Restore(edited)
				}
				var err error
				result, err = def.Run(t.Context(), sc)
				if err != nil {
					t.Fatalf("Publish: %v", err)
				}
				settle(def, sc, result)
				if result.Next != run.TransitionWait {
					break
				}
			}
			if result.Next != run.TransitionPause || !strings.Contains(result.Message, string(field.ProductDescription)) {
				t.Errorf("publish = %s %q, want a pause naming the description that moved", result.Next, result.Message)
			}
			if stored, _ := h.server.Lookup(h.held.ID); stored.Content != "<p>The owner's description, edited a moment ago.</p>" {
				t.Errorf("the owner's edit became %q: a save that cannot change the description does not make his edit our base", stored.Content)
			}
		})
	}
}

func TestAProductImagesAltAndTitleAreWrittenWhereAttachedAndPutBack(t *testing.T) {
	t.Parallel()

	h := newProductHarness(t, storeProduct())
	h.deps.Types = imageTypes{}
	attached := h.server.SeedUpload(wptest.Upload{Filename: "machine.jpg", MimeType: "image/jpeg", Title: "machine", Parent: h.held.ID})
	shared := h.server.SeedUpload(wptest.Upload{Filename: "logo.jpg", MimeType: "image/jpeg", Title: "logo", Alt: "Shop logo"})
	sc := storeContext(t, h.held.ID)
	altOf := func(upload wptest.Upload) string {
		return string(field.Instance(field.ImageAlt, strconv.FormatInt(upload.ID, 10)))
	}
	withFields(t, sc,
		steps.FieldValue{Key: altOf(attached), Value: "An espresso machine on a counter"},
		steps.FieldValue{Key: altOf(shared), Value: "An espresso machine"},
	)
	published := runPublish(t, h.deps, sc)

	uploads := map[int64]wptest.Upload{}
	for _, upload := range h.server.Uploads() {
		uploads[upload.ID] = upload
	}
	if uploads[attached.ID].Alt != "An espresso machine on a counter" || uploads[attached.ID].Title != "machine" {
		t.Errorf("the attached image holds %+v, want its alt text written and its title left", uploads[attached.ID])
	}
	if uploads[shared.ID].Alt != "Shop logo" {
		t.Errorf("an image attached to no page became %q: another page may show it", uploads[shared.ID].Alt)
	}
	if !slices.ContainsFunc(published.Skipped, func(skip string) bool { return strings.Contains(skip, altOf(shared)) }) {
		t.Errorf("skipped = %v, want the shared image named", published.Skipped)
	}
	if got := published.PreviousMedia; len(got) != 1 || got[0] != (steps.FieldWrite{Key: field.Key(altOf(attached)), Before: "", Written: "An espresso machine on a counter"}) {
		t.Errorf("the publish kept %+v", got)
	}

	page := *h.recorded
	source := newSourceRecord()
	source.record(t, page.ID, map[run.ArtifactKind]any{run.ArtifactPublishResult: published})
	h.deps.Pages, h.deps.Links, h.deps.Items, h.deps.Artifacts = newPageMap(page), &linkRecorder{}, source, source
	reverted, result := runRevert(t, h.deps, "page-child")
	if result.Next == run.TransitionPause || reverted.Outcome != steps.OutcomeRestored {
		t.Fatalf("the revert answered %+v / %s", reverted, result.Message)
	}
	for _, upload := range h.server.Uploads() {
		if upload.ID == attached.ID && upload.Alt != "" {
			t.Errorf("after the revert the image's alt is %q, want it empty again", upload.Alt)
		}
	}
}

func TestAVariationIsNotWrittenOverAProductWhoseSaveWouldChangeIt(t *testing.T) {
	t.Parallel()

	item := storeProduct()
	item.ProductType, item.Content = "variable", "<p>old<br>description</p>"
	h := newProductHarness(t, item, wptest.WithFilteredHTML())
	h.deps.Types = variationTypes{}
	variation := h.server.Seed(wptest.Item{Type: wptest.TypeProductVariation, Parent: h.held.ID, SKU: "EM-1-S", Status: "publish",
		Content: "<p>Old small.</p>", Choices: []wptest.Choice{{ID: 3, Name: "Size", Option: "Small"}}})[0]
	sc := storeContext(t, h.held.ID)
	withFields(t, sc, steps.FieldValue{Key: string(variationKey(variation)), Value: "<p>A small machine.</p>"})

	result, err := drive(t, steps.Publish(h.deps), sc)
	if err != nil {
		t.Fatalf("Publish: %v", err)
	}
	if result.Next != run.TransitionPause || !strings.Contains(result.Message, steps.ReasonSaveAltersClient) {
		t.Errorf("publish = %s %q, want a pause saying the save would change the client's description", result.Next, result.Message)
	}
	if stored, _ := h.server.Lookup(variation.ID); stored.Content != "<p>Old small.</p>" {
		t.Errorf("the variation became %q: its save re-saves the product, whose description the filters would change", stored.Content)
	}
	if parent, _ := h.server.Lookup(h.held.ID); parent.Content != "<p>old<br>description</p>" {
		t.Errorf("the client's description became %q", parent.Content)
	}
}
