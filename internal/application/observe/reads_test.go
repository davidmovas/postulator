package observe_test

import (
	"context"
	stderrors "errors"
	"fmt"
	"slices"
	"testing"
	"time"

	"github.com/davidmovas/postulator/internal/application/observe"
	"github.com/davidmovas/postulator/internal/domain/block"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

func TestABulkWalkTakesTermsFromItsListAndFetchesNoEmptyBody(t *testing.T) {
	t.Parallel()
	s := newScenario(t)

	s.run(t, observe.Options{})

	raws, terms, _ := s.plugin.reads()
	if !slices.Equal(terms, []int64{productID}) {
		t.Errorf("the terms route was asked for %v, want the store's product alone: the bulk list carries every post's terms", terms)
	}
	for _, empty := range []int64{peptidesID, shopID} {
		if slices.Contains(raws, empty) {
			t.Errorf("the raw route was asked for post %d, whose list hash is the empty body's", empty)
		}
	}
	for _, written := range []int64{homeID, bpcID, tb500ID, accountID, productID} {
		if !slices.Contains(raws, written) {
			t.Errorf("the raw route was never asked for post %d, whose body is new", written)
		}
	}
	for _, path := range []string{"/peptides/", "/shop/"} {
		body := s.stores.bodies[s.pageAt(t, path).ID]
		if body[block.Current].Raw != "" || body[block.Current].Hash != block.BodyHash("") || body[block.Base].Hash != block.BodyHash("") {
			t.Errorf("%s body = %+v, want the empty body kept as its own snapshot", path, body)
		}
	}
	if got := s.stores.terms[s.pageAt(t, "/product/bpc-157-peptide/").ID]; len(got) != 2 || got[0].Taxonomy != "product_cat" || got[1].Taxonomy != "entity" {
		t.Errorf("the product's terms = %+v, want the two its list item carries", got)
	}
}

func TestAFailedReadNamesItsPageOrStageAndKeepsItsCode(t *testing.T) {
	t.Parallel()

	answered := "the WordPress site failed on its side: WordPress answered 500 on GET /wp-json/postulator/v1/content/4/raw"
	cases := []struct {
		name   string
		broken func(s *scenario)
		want   string
	}{
		{
			name: "a page's body",
			broken: func(s *scenario) {
				s.plugin.failRaw = map[int64]error{tb500ID: errors.New(errors.External, answered).WithRetry(3 * time.Second)}
			},
			want: "reading the body of /tb-500/: " + answered,
		},
		{
			name: "the list",
			broken: func(s *scenario) {
				s.plugin.failList = errors.New(errors.External, answered).WithRetry(3 * time.Second)
			},
			want: "reading the site's pages: " + answered,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			s := newScenario(t)
			tc.broken(s)

			_, err := s.service.Run(context.Background(), siteID, s.readers, observe.Options{}, 50)

			if err == nil || err.Error() != tc.want {
				t.Fatalf("err = %v, want %q", err, tc.want)
			}
			var kernel *errors.Error
			if !stderrors.As(err, &kernel) || kernel.Code != errors.External || kernel.Retry == nil || kernel.Retry.After != 3*time.Second {
				t.Errorf("err = %#v, want the site's code and its retry kept", kernel)
			}
		})
	}
}

func TestASimpleProductIsNotAskedForVariations(t *testing.T) {
	t.Parallel()
	s := newScenario(t)
	simple := s.store.products[0]
	simple.ID, simple.Permalink, simple.Slug, simple.Type, simple.SKU, simple.VariationIDs = 20, baseURL+"/product/tb-500-vial/", "tb-500-vial", "simple", "TB500", nil
	s.store.products = append(s.store.products, simple)

	s.run(t, observe.Options{})

	if asked := s.store.askedVariations(); !slices.Equal(asked, []int64{productID}) {
		t.Errorf("variations were asked for %v, want the variable product alone", asked)
	}
}

func TestAListWithoutItsTermsIsCompletedByTheTermsRoute(t *testing.T) {
	t.Parallel()
	s := newScenario(t)
	s.readers.Manifest.PostTypes, s.readers.Manifest.Taxonomies = nil, nil
	for i := range s.plugin.items {
		s.plugin.items[i].Terms = nil
	}

	s.run(t, observe.Options{})

	_, terms, _ := s.plugin.reads()
	if !slices.Contains(terms, productID) {
		t.Errorf("the terms route was asked for %v, want the product's terms read from it", terms)
	}
	if got := s.stores.terms[s.pageAt(t, "/product/bpc-157-peptide/").ID]; len(got) != 2 {
		t.Errorf("the product's terms = %+v, want the two the terms route answers", got)
	}
}

func TestTheContentStageReadsBodiesInParallelAndEachPageKeepsItsOwn(t *testing.T) {
	t.Parallel()
	s := newScenario(t)
	s.plugin.slow = 3 * time.Millisecond
	const extra = 24
	for i := range extra {
		wpID := int64(500 + i)
		raw := fmt.Sprintf("The body of article %d.", i)
		s.plugin.items = append(s.plugin.items, contentItem(wpID, "page", fmt.Sprintf("/article-%d/", i), fmt.Sprintf("Article %d", i), raw))
		s.plugin.raws[wpID] = rawOf(raw)
	}

	state, err := s.service.Run(context.Background(), siteID, s.readers, observe.Options{}, 50)
	if err != nil || state.Stage != observe.StageDone {
		t.Fatalf("observe: stage %s, %v", state.Stage, err)
	}

	_, _, peak := s.plugin.reads()
	limit := s.stores.sites[siteID].Settings.Observe.RenderConcurrency
	if peak < 2 || peak > limit {
		t.Errorf("the plugin's reads peaked at %d at once, want between 2 and the site's %d", peak, limit)
	}
	for i := range extra {
		page := s.pageAt(t, fmt.Sprintf("/article-%d/", i))
		if got, want := s.stores.bodies[page.ID][block.Current].Raw, fmt.Sprintf("The body of article %d.", i); got != want {
			t.Errorf("/article-%d/ holds %q, want %q", i, got, want)
		}
	}
	if got := s.pageAt(t, "/article-3/").WPType; got != pagemap.WPPage {
		t.Errorf("/article-3/ is a %s", got)
	}
}
