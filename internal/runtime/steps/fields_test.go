package steps_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/davidmovas/postulator/internal/adapters/wp/wptest"
	"github.com/davidmovas/postulator/internal/domain/field"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/pagetype"
	"github.com/davidmovas/postulator/internal/domain/run"
	"github.com/davidmovas/postulator/internal/domain/template"
	"github.com/davidmovas/postulator/internal/runtime/steps"
)

type scopedTypes struct {
	keys []field.Key
}

func (s scopedTypes) Set(_ context.Context, siteID string) (pagetype.Set, error) {
	kind := pagetype.Type{ID: "type-entity", SiteID: siteID, Name: "Entity", Preset: pagetype.Entity}
	scopes := pagetype.DefaultScopes(pagetype.Entity, kind.ID)
	for _, key := range s.keys {
		scope := pagetype.DefaultScope(pagetype.Entity, key, kind.ID)
		scope.Enabled, scope.AIWrites, scope.MaxLength = true, true, 40
		scope.Instruction = "One sentence for the archive."
		scopes = append(scopes, scope)
	}
	return pagetype.NewSet([]pagetype.Type{kind}, scopes), nil
}

func withFields(t *testing.T, sc *run.StepContext, values ...steps.FieldValue) {
	t.Helper()

	blob, err := json.Marshal(steps.Fields{Values: values, Skipped: []steps.FieldSkip{}})
	if err != nil {
		t.Fatalf("encode the fields: %v", err)
	}
	sc.Artifacts[run.ArtifactFields] = run.Artifact{Kind: run.ArtifactFields, Blob: blob}
}

func fieldsOf(t *testing.T, result run.Result) steps.Fields {
	t.Helper()

	if len(result.Artifacts) != 1 || result.Artifacts[0].Kind != run.ArtifactFields {
		t.Fatalf("artifacts = %+v, want the fields", result.Artifacts)
	}
	var fields steps.Fields
	if err := json.Unmarshal(result.Artifacts[0].Blob, &fields); err != nil {
		t.Fatalf("decode the fields: %v", err)
	}
	return fields
}

func TestGenerateFieldsAsksOnlyForWritableFields(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		wpType  pagemap.WPType
		keys    []field.Key
		asked   bool
		want    string
		skipped []string
	}{
		{name: "a post's excerpt is written", wpType: pagemap.WPPost, keys: []field.Key{field.Excerpt}, asked: true, want: "Espresso, pulled under pressure…"},
		{name: "a page's post type carries no excerpt", wpType: pagemap.WPPage, keys: []field.Key{field.Excerpt}, skipped: []string{"excerpt"}},
		{name: "no field the scope enables", wpType: pagemap.WPPost},
		{name: "a custom field nothing writes yet", wpType: pagemap.WPPost, keys: []field.Key{field.Instance(field.Meta, "hero_subtitle")}, skipped: []string{"meta:hero_subtitle"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			deps, _ := imageDeps(t)
			deps.Types = scopedTypes{keys: tc.keys}
			recorder := &promptRecorder{reply: `{"fields":[{"key":"excerpt","value":"Espresso, pulled under pressure, served small and strong."}]}`}
			deps.LLM = recorder
			sc := unitContext(t, nil)
			sc.Page.WPType = tc.wpType

			result, err := steps.GenerateFields(deps).Run(t.Context(), sc)
			if err != nil {
				t.Fatalf("GenerateFields: %v", err)
			}
			fields := fieldsOf(t, result)
			if asked := recorder.last != ""; asked != tc.asked {
				t.Fatalf("the model was asked: %t, want %t", asked, tc.asked)
			}
			if tc.asked {
				if got, held := fields.Value(field.Excerpt); !held || got != tc.want {
					t.Errorf("excerpt = %q, %t; want %q clipped to the scope's 40 characters", got, held, tc.want)
				}
				for _, want := range []string{"THE FIELDS TO WRITE", "Excerpt (key excerpt)", "Instruction: One sentence for the archive.", "Length: at most 40 characters"} {
					if !strings.Contains(recorder.last, want) {
						t.Errorf("the prompt lacks %q:\n%s", want, recorder.last)
					}
				}
			}
			if len(fields.Skipped) != len(tc.skipped) {
				t.Fatalf("skipped = %+v, want %v", fields.Skipped, tc.skipped)
			}
			for i := range tc.skipped {
				if fields.Skipped[i].Key != tc.skipped[i] || fields.Skipped[i].Reason == "" {
					t.Errorf("skipped = %+v, want %s with its reason", fields.Skipped[i], tc.skipped[i])
				}
			}
		})
	}
}

func TestTheShortDescriptionIsAFieldOfItsOwn(t *testing.T) {
	t.Parallel()

	briefed := &template.Product{ShortDescription: template.ProductShortDescription{
		Enabled: true, Intent: "Say what it is for", TargetWords: 30, PrimaryKeyword: true,
	}}
	cases := []struct {
		name    string
		wpType  pagemap.WPType
		product *template.Product
		asked   bool
		skipped []string
	}{
		{name: "the scope and the template both write it", wpType: pagemap.WPProduct, product: briefed, asked: true},
		{name: "the template switches it off", wpType: pagemap.WPProduct, product: &template.Product{}, skipped: []string{"product_short_description"}},
		{name: "the template declares no product outputs", wpType: pagemap.WPProduct, skipped: []string{"product_short_description"}},
		{name: "a page has none", wpType: pagemap.WPPage, product: briefed},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			deps, _ := imageDeps(t)
			deps.Types = typesStub{}
			recorder := &promptRecorder{reply: `{"fields":[{"key":"product_short_description","value":"A compact machine & grinder\nfor the counter."}]}`}
			deps.LLM = recorder
			sc := unitContext(t, nil)
			sc.Page.WPType = tc.wpType
			sc.Spec.Product = tc.product

			result, err := steps.GenerateFields(deps).Run(t.Context(), sc)
			if err != nil {
				t.Fatalf("GenerateFields: %v", err)
			}
			fields := fieldsOf(t, result)
			if asked := recorder.last != ""; asked != tc.asked {
				t.Fatalf("the model was asked: %t, want %t", asked, tc.asked)
			}
			if tc.asked {
				got, held := fields.Value(field.ProductShortDescription)
				if want := "<p>A compact machine &amp; grinder for the counter.</p>"; !held || got != want {
					t.Errorf("short description = %q, %t; want the plain answer as one escaped paragraph %q", got, held, want)
				}
				for _, want := range []string{
					"Short description (key product_short_description)", "The template asks: Say what it is for", "About 30 words",
					"It carries the primary keyword",
				} {
					if !strings.Contains(recorder.last, want) {
						t.Errorf("the prompt lacks %q:\n%s", want, recorder.last)
					}
				}
				if strings.Contains(recorder.last, "key excerpt") {
					t.Errorf("a product was asked for an excerpt, which its short description is:\n%s", recorder.last)
				}
			}
			if len(fields.Skipped) != len(tc.skipped) {
				t.Fatalf("skipped = %+v, want %v", fields.Skipped, tc.skipped)
			}
			for i := range tc.skipped {
				if fields.Skipped[i].Key != tc.skipped[i] || fields.Skipped[i].Reason == "" {
					t.Errorf("skipped = %+v, want %s with its reason", fields.Skipped[i], tc.skipped[i])
				}
			}
		})
	}
}

func TestTheExcerptIsWrittenAndReverted(t *testing.T) {
	t.Parallel()

	stand := newRevertStand(t)
	deps := stand.deps
	deps.Types = scopedTypes{keys: []field.Key{field.Excerpt}}
	seeded := stand.server.Seed(wptest.Item{Type: wptest.TypePost, Title: "Espresso", Slug: "espresso-post", Status: "publish", Content: oursBody, Excerpt: "The owner's excerpt."})
	sc := rewritten(publishContext(t))
	sc.Page.WPType, sc.Page.WPID = pagemap.WPPost, &seeded[0].ID
	blob, err := json.Marshal(steps.Fields{Values: []steps.FieldValue{{Key: "excerpt", Value: "Pull a shot."}}})
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	sc.Artifacts[run.ArtifactFields] = run.Artifact{Kind: run.ArtifactFields, Blob: blob}

	published := runPublish(t, deps, sc)
	stored, _ := stand.server.Lookup(seeded[0].ID)
	if stored.Excerpt != "Pull a shot." || published.PreviousExcerpt == nil || *published.PreviousExcerpt != "The owner's excerpt." {
		t.Fatalf("excerpt on the site %q, kept %v; want ours written and the owner's kept for a revert", stored.Excerpt, published.PreviousExcerpt)
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
	if back, _ := stand.server.Lookup(seeded[0].ID); back.Excerpt != "The owner's excerpt." {
		t.Errorf("after the revert the excerpt is %q, want the owner's back", back.Excerpt)
	}

	edited := stand.server.Seed(wptest.Item{Type: wptest.TypePost, Title: "Ristretto", Slug: "ristretto-post", Status: "publish", Content: oursBody, Excerpt: "Theirs."})
	sc.Page.WPID, sc.Page.Slug = &edited[0].ID, "ristretto-post"
	again := runPublish(t, deps, sc)
	stand.server.Restore(wptest.Item{ID: edited[0].ID, Type: wptest.TypePost, Title: "Ristretto", Slug: "ristretto-post", Status: "publish",
		Content: oursBody, Excerpt: "Edited by the owner after the run."})
	second := pagemap.Page{ID: "page-ristretto-post", SiteID: "site", Path: "/ristretto-post/", Slug: "ristretto-post", WPType: pagemap.WPPost,
		Status: pagemap.StatusPublished, WPID: &edited[0].ID}
	stand.pages.items[second.ID] = second
	stand.wpIDs[second.ID] = edited[0].ID
	stand.source.record(t, second.ID, map[run.ArtifactKind]any{run.ArtifactPublishResult: again})
	refused, _ := stand.revert(t, second.ID)
	if _, found := findingOf(refused, steps.CodeRevertExcerptKept); !found {
		t.Errorf("a revert over an excerpt the owner edited = %+v, want it kept and said", refused)
	}
	if back, _ := stand.server.Lookup(edited[0].ID); back.Excerpt != "Edited by the owner after the run." {
		t.Errorf("the owner's later excerpt became %q", back.Excerpt)
	}
}
