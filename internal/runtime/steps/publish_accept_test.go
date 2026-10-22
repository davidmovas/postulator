package steps_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/davidmovas/postulator/internal/adapters/wp/wptest"
	"github.com/davidmovas/postulator/internal/domain/field"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/run"
	"github.com/davidmovas/postulator/internal/runtime/steps"
)

const (
	ownersExcerpt = "The owner's excerpt."
	ourExcerpt    = "Pull a shot at nine bars."
)

func acceptOnly(t *testing.T, sc *run.StepContext, keys ...field.Key) {
	t.Helper()

	accept(t, sc)
	if err := run.Set(sc.Check, run.CheckpointAcceptFields, keys); err != nil {
		t.Fatalf("accept the fields: %v", err)
	}
}

type reviewedPage struct {
	deps   steps.Deps
	server *wptest.Server
	sc     *run.StepContext
	bodies *bodyLog
	fields *fieldLog
	wpID   int64
}

func reviewWithAnExcerpt(t *testing.T, opts ...wptest.Option) reviewedPage {
	t.Helper()

	deps, server := imageDepsWith(t, opts...)
	bodies, fields := &bodyLog{}, &fieldLog{}
	deps.Bodies, deps.Fields = bodies, fields
	deps.Types = scopedTypes{keys: []field.Key{field.Excerpt}}
	seeded := server.Seed(wptest.Item{
		Type: "page", Title: "Espresso", Slug: "espresso", Status: "publish", Content: clientText, Excerpt: ownersExcerpt,
	})
	sc := rewritten(publishContext(t))
	sc.Page.WPID = &seeded[0].ID
	sc.Page.Status = pagemap.StatusPublished
	sc.Run.PublishMode = run.PublishDraft
	sc.Item.CurrentStep = steps.NamePublish
	withFields(t, sc, steps.FieldValue{Key: string(field.Excerpt), Value: ourExcerpt})

	result, err := drive(t, steps.Publish(deps), sc)
	if err != nil || !awaitsReview(result) {
		t.Fatalf("Publish = %+v, %v; want a review first", result, err)
	}
	plan, held := editPlanIn(t, result)
	if !held || len(plan.Pages) != 1 || !hasFieldChange(plan.Pages[0].Fields, field.Excerpt) || plan.Pages[0].Body != field.Body {
		t.Fatalf("the review shows %+v, want the excerpt and the body offered", plan)
	}
	server.ResetRequests()
	return reviewedPage{deps: deps, server: server, sc: sc, bodies: bodies, fields: fields, wpID: seeded[0].ID}
}

func (p reviewedPage) acceptOnly(t *testing.T, keys ...field.Key) steps.PublishResult {
	t.Helper()

	acceptOnly(t, p.sc, keys...)
	result, err := drive(t, steps.Publish(p.deps), p.sc)
	if err != nil {
		t.Fatalf("Publish after the acceptance: %v", err)
	}
	if result.Next == run.TransitionPause {
		t.Fatalf("result = %+v, want the accepted fields written", result)
	}
	published, held := publishedIn(t, result)
	if !held {
		t.Fatal("the written page carries no publish result")
	}
	return published
}

func changedKeys(changes []field.Change) []field.Key {
	out := make([]field.Key, 0, len(changes))
	for i := range changes {
		out = append(out, changes[i].Key)
	}
	return out
}

func TestAnAcceptedFieldIsWrittenAndADeclinedOneIsLeft(t *testing.T) {
	t.Parallel()

	t.Run("through the plugin", func(t *testing.T) {
		t.Parallel()

		p := reviewWithAnExcerpt(t)
		published := p.acceptOnly(t, field.SEOTitle, field.Body)
		stored, _ := p.server.Lookup(p.wpID)
		if want := strings.TrimPrefix(publishBody, "<h1>Espresso</h1>"); stored.Content != want {
			t.Errorf("the site holds %q, want the accepted body", stored.Content)
		}
		if stored.Excerpt != ownersExcerpt {
			t.Errorf("the site's excerpt is %q, want the owner's: it was declined", stored.Excerpt)
		}
		if published.WrittenMeta == nil || published.WrittenMeta.Title == "" || published.WrittenMeta.Description != "" {
			t.Errorf("the SEO write = %+v, want the title alone", published.WrittenMeta)
		}
		if want := []field.Key{field.Excerpt, field.SEODescription}; !slices.Equal(published.Declined, want) {
			t.Errorf("declined = %v, want %v", published.Declined, want)
		}
		if got := changedKeys(published.Fields); slices.Contains(got, field.Excerpt) || slices.Contains(got, field.SEODescription) {
			t.Errorf("the result's fields name %v, want only what was written", got)
		}
		for _, request := range writesOf(p.server) {
			if strings.Contains(request.Path, "/wp/v2/pages/") {
				t.Errorf("%s %s sent the page's core fields, want nothing: the excerpt was declined", request.Method, request.Path)
			}
		}
	})

	t.Run("through core, without the plugin", func(t *testing.T) {
		t.Parallel()

		p := reviewWithAnExcerpt(t, wptest.WithoutPlugin())
		published := p.acceptOnly(t, field.Excerpt)
		stored, _ := p.server.Lookup(p.wpID)
		if stored.Content != clientText {
			t.Errorf("the site holds %q, want the owner's body: it was declined", stored.Content)
		}
		if stored.Excerpt != ourExcerpt {
			t.Errorf("the site's excerpt is %q, want the accepted one", stored.Excerpt)
		}
		if !slices.Equal(published.Declined, []field.Key{field.Body}) || len(published.Blocks) != 0 {
			t.Errorf("declined %v with blocks %+v, want the body declined and no block on record", published.Declined, published.Blocks)
		}
	})
}

func TestADeclinedFieldIsNeverRecordedAsOurs(t *testing.T) {
	t.Parallel()

	p := reviewWithAnExcerpt(t)
	p.acceptOnly(t, field.SEOTitle, field.Body)
	for _, key := range []field.Key{field.Excerpt, field.SEODescription} {
		if _, recorded := p.fields.recorded[key]; recorded {
			t.Errorf("the declined %s was recorded under the run, so the next run would take the owner's value for ours", key)
		}
	}
	if got := p.fields.recordedValue(field.SEOTitle); got == "" {
		t.Error("the accepted SEO title was not recorded")
	}
}

func TestADeclinedBodyLeavesTheClientsBlocksAndOurRecordAlone(t *testing.T) {
	t.Parallel()

	p := reviewWithAnExcerpt(t)
	published := p.acceptOnly(t, field.Excerpt, field.SEOTitle)
	stored, _ := p.server.Lookup(p.wpID)
	if stored.Content != clientText || stored.Excerpt != ourExcerpt {
		t.Errorf("the site holds %q with %q, want the owner's body and our excerpt", stored.Content, stored.Excerpt)
	}
	for _, request := range writesOf(p.server) {
		if strings.Contains(request.Path, "/raw") {
			t.Errorf("%s %s wrote the body, which was declined", request.Method, request.Path)
		}
	}
	if ours := p.bodies.ours(p.sc.Page.ID); len(ours) != 0 {
		t.Errorf("our blocks = %v, want none recorded for a body we did not write", ours)
	}
	if len(published.Blocks) != 0 || published.ContentHash != "" {
		t.Errorf("the result names blocks %+v and hash %q, want no body on record", published.Blocks, published.ContentHash)
	}
	if recorded := p.bodies.recorded(p.sc.Page.ID); recorded != "" {
		t.Errorf("the body snapshot = %q, want none recorded for a body we did not write", recorded)
	}
}

func TestAPlanThatMovedAfterAPartialAcceptanceIsAcceptedWhole(t *testing.T) {
	t.Parallel()

	p := reviewWithAnExcerpt(t)
	acceptOnly(t, p.sc, field.SEOTitle)
	moved := clientText + "\n\n<p>Added by the owner after the review.</p>"
	p.server.Rewrite(p.wpID, moved)
	result, err := drive(t, steps.Publish(p.deps), p.sc)
	if err != nil || !awaitsReview(result) {
		t.Fatalf("Publish = %+v, %v; want the moved plan shown again", result, err)
	}

	accept(t, p.sc)
	result, err = drive(t, steps.Publish(p.deps), p.sc)
	if err != nil || result.Next == run.TransitionPause {
		t.Fatalf("Publish after the whole acceptance = %+v, %v", result, err)
	}
	published, _ := publishedIn(t, result)
	if len(published.Declined) != 0 {
		t.Errorf("declined %v, want every field written: the second acceptance took the whole plan", published.Declined)
	}
	stored, _ := p.server.Lookup(p.wpID)
	if stored.Excerpt != ourExcerpt || published.WrittenMeta == nil || published.WrittenMeta.Description == "" {
		t.Errorf("the excerpt is %q and the SEO write %+v, want both written", stored.Excerpt, published.WrittenMeta)
	}
}

func TestAProductReviewAcceptsItsFieldsOneByOne(t *testing.T) {
	t.Parallel()

	item := storeProduct()
	item.ProductType = "variable"
	h := newProductHarness(t, item)
	h.deps.Types = variationTypes{}
	seeded := h.server.Seed(
		wptest.Item{Type: wptest.TypeProductVariation, Parent: h.held.ID, SKU: "EM-1-S", RegularPrice: "120", Status: "publish",
			Content: "<p>Old small.</p>", Choices: []wptest.Choice{{ID: 3, Name: "Size", Option: "Small"}}},
		wptest.Item{Type: wptest.TypeProductVariation, Parent: h.held.ID, SKU: "EM-1-L", RegularPrice: "150", Status: "publish",
			Content: "<p>Old large.</p>", Choices: []wptest.Choice{{ID: 3, Name: "Size", Option: "Large"}}},
	)
	small, large := seeded[0], seeded[1]
	sc := storeContext(t, h.held.ID)
	sc.Run.PublishMode = run.PublishDraft
	sc.Item.CurrentStep = steps.NamePublish
	withFields(t, sc,
		steps.FieldValue{Key: string(field.ProductShortDescription), Value: productShort},
		steps.FieldValue{Key: string(variationKey(small)), Value: "<p>A small machine.</p>"},
		steps.FieldValue{Key: string(variationKey(large)), Value: "<p>A large machine.</p>"},
	)
	before, _ := h.server.Lookup(h.held.ID)

	result, err := drive(t, steps.Publish(h.deps), sc)
	if err != nil || !awaitsReview(result) {
		t.Fatalf("Publish = %+v, %v; want a review first", result, err)
	}
	if plan, held := editPlanIn(t, result); !held || plan.Pages[0].Body != field.ProductDescription {
		t.Fatalf("the review shows %+v, want the product's description offered under its own key", plan)
	}
	acceptOnly(t, sc, variationKey(small), field.ProductDescription)
	result, err = drive(t, steps.Publish(h.deps), sc)
	if err != nil || result.Next == run.TransitionPause {
		t.Fatalf("Publish after the acceptance = %+v, %v", result, err)
	}
	published, _ := publishedIn(t, result)

	if got, _ := h.server.Lookup(small.ID); got.Content != "<p>A small machine.</p>" {
		t.Errorf("the small variation holds %q, want the accepted description", got.Content)
	}
	if got, _ := h.server.Lookup(large.ID); got.Content != "<p>Old large.</p>" {
		t.Errorf("the large variation holds %q, want the owner's: it was declined", got.Content)
	}
	after, _ := h.server.Lookup(h.held.ID)
	if after.Excerpt != before.Excerpt {
		t.Errorf("the short description is %q, want the owner's %q: it was declined", after.Excerpt, before.Excerpt)
	}
	want := []field.Key{field.ProductShortDescription, field.SEODescription, field.SEOTitle, variationKey(large)}
	if !slices.Equal(published.Declined, want) {
		t.Errorf("declined = %v, want %v", published.Declined, want)
	}
	if len(published.PreviousVariations) != 1 || published.PreviousVariations[0].Key != variationKey(small) {
		t.Errorf("the result keeps %+v, want the small variation alone for a revert", published.PreviousVariations)
	}
	if published.PreviousProduct != nil && published.PreviousProduct.ShortWritten {
		t.Error("the product's snapshot says its short description was written, so a revert would take the owner's for ours")
	}
}
