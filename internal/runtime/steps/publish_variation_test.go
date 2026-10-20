package steps_test

import (
	"encoding/json"
	"net/http"
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

	item := storeProduct()
	item.ProductType = "variable"
	h := newProductHarness(t, item)
	h.deps.Types = variationTypes{}
	seeded := h.server.Seed(
		wptest.Item{Type: wptest.TypeProductVariation, Parent: h.held.ID, SKU: "EM-1-S", RegularPrice: "120", Status: "publish",
			Content: "<p>Old small.</p>", Choices: []wptest.Choice{{ID: 3, Name: "Size", Option: "Small"}}},
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
	want := []steps.VariationWrite{
		{Key: variationKey(p.small), Before: "<p>Old small.</p>", Written: "<p>A small machine.</p>"},
		{Key: variationKey(p.large), Before: "", Written: "<p>A large machine.</p>"},
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
	if small.Content != "<p>Old small.</p>" || large.Content != "" {
		t.Errorf("after the revert the variations hold %q and %q, want what they held before", small.Content, large.Content)
	}
}

func TestAVariationTheClientEditedSinceIsHandedBack(t *testing.T) {
	t.Parallel()

	p := publishVariations(t)
	edited, _ := p.server.Lookup(p.large.ID)
	edited.Content = "<p>The owner's own large.</p>"
	p.server.Restore(edited)

	reverted, result := runRevert(t, p.deps, "page-child")
	if result.Next != run.TransitionPause && reverted.Outcome == steps.OutcomeRestored {
		t.Fatalf("a revert over a variation the client edited = %+v, want it handed back", reverted)
	}
	if large, _ := p.server.Lookup(p.large.ID); large.Content != "<p>The owner's own large.</p>" {
		t.Errorf("the owner's later variation description became %q", large.Content)
	}
	if small, _ := p.server.Lookup(p.small.ID); small.Content != "<p>A small machine.</p>" {
		t.Errorf("a refused revert still wrote a variation back: %q", small.Content)
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
