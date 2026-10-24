package sqlite_test

import (
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/davidmovas/postulator/internal/adapters/sqlite"
	"github.com/davidmovas/postulator/internal/adapters/sqlite/sqlitetest"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

func TestPageIdentityRepoRoundTrip(t *testing.T) {
	t.Parallel()

	store := sqlitetest.Open(t)
	owner := sqlitetest.Site(t, store, "shop")
	other := sqlitetest.Site(t, store, "other")
	hub := sqlitetest.Page(t, store, owner.ID, "/peptides/")
	product := sqlitetest.Page(t, store, owner.ID, "/peptides/tb-500/")
	elsewhere := sqlitetest.Page(t, store, other.ID, "/elsewhere/")
	repo := sqlite.NewPageIdentityRepo(store)

	if _, err := repo.Get(t.Context(), hub.ID); !errors.IsCode(err, errors.NotFound) {
		t.Fatalf("Get before Put code = %q, want NOT_FOUND", errors.CodeOf(err))
	}

	parent := int64(12)
	modified := sqlitetest.Stamp.Add(-time.Hour)
	identities := []pagemap.Identity{
		{PageID: hub.ID, URL: "https://shop.example.com/peptides/", Status: "publish", WPType: pagemap.WPPage, ModifiedGMT: &modified, ObservedAt: sqlitetest.Stamp},
		{PageID: product.ID, URL: "https://shop.example.com/peptides/tb-500/", Status: "draft", WPType: pagemap.WPProduct, WPParentID: &parent, ObservedAt: sqlitetest.Stamp},
		{PageID: elsewhere.ID, URL: "https://other.example.com/elsewhere/", Status: "publish", WPType: "event", ObservedAt: sqlitetest.Stamp},
	}
	for i := range identities {
		if err := repo.Put(t.Context(), identities[i]); err != nil {
			t.Fatalf("Put %s: %v", identities[i].URL, err)
		}
	}
	got, err := repo.Get(t.Context(), hub.ID)
	if err != nil || !reflect.DeepEqual(got, identities[0]) {
		t.Fatalf("Get = %+v, %v\nwant %+v", got, err, identities[0])
	}

	moved := identities[0]
	moved.Status = "private"
	moved.ModifiedGMT = nil
	moved.ObservedAt = sqlitetest.Stamp.Add(time.Minute)
	if err = repo.Put(t.Context(), moved); err != nil {
		t.Fatalf("Put again: %v", err)
	}
	if got, err = repo.Get(t.Context(), hub.ID); err != nil || !reflect.DeepEqual(got, moved) {
		t.Fatalf("Get after a second Put = %+v, %v\nwant %+v", got, err, moved)
	}

	want := []pagemap.Identity{moved, identities[1]}
	slices.SortFunc(want, func(a, b pagemap.Identity) int { return strings.Compare(a.PageID, b.PageID) })
	listed, err := repo.ListBySite(t.Context(), owner.ID)
	if err != nil || !reflect.DeepEqual(listed, want) {
		t.Fatalf("ListBySite = %+v, %v\nwant %+v", listed, err, want)
	}

	stray := identities[0]
	stray.PageID = "no-such-page"
	if err = repo.Put(t.Context(), stray); !errors.IsCode(err, errors.Invalid) {
		t.Errorf("an identity of no page code = %q, want INVALID", errors.CodeOf(err))
	}

	if err = sqlite.NewPageRepo(store).Delete(t.Context(), hub.ID); err != nil {
		t.Fatalf("delete the page: %v", err)
	}
	if _, err = repo.Get(t.Context(), hub.ID); !errors.IsCode(err, errors.NotFound) {
		t.Errorf("Get after the page is deleted code = %q, want NOT_FOUND", errors.CodeOf(err))
	}
}

func TestPageIdentityRepoReplacesTheTermsOfAPage(t *testing.T) {
	t.Parallel()

	store := sqlitetest.Open(t)
	owner := sqlitetest.Site(t, store, "shop")
	other := sqlitetest.Site(t, store, "other")
	product := sqlitetest.Page(t, store, owner.ID, "/tb-500/")
	post := sqlitetest.Page(t, store, owner.ID, "/news/")
	elsewhere := sqlitetest.Page(t, store, other.ID, "/elsewhere/")
	repo := sqlite.NewPageIdentityRepo(store)

	terms := []pagemap.Term{
		{Taxonomy: "product_tag", TermID: 7, Name: "New", Slug: "new"},
		{Taxonomy: "product_cat", TermID: 9, Name: "Healing", Slug: "healing", ParentTermID: 7},
		{Taxonomy: "product_cat", TermID: 7, Name: "Peptides", Slug: "peptides"},
	}
	sorted := []pagemap.Term{terms[2], terms[1], terms[0]}
	news := []pagemap.Term{{Taxonomy: "category", TermID: 3, Name: "News", Slug: "news"}}
	for _, step := range []struct {
		pageID string
		terms  []pagemap.Term
	}{
		{pageID: product.ID, terms: terms},
		{pageID: post.ID, terms: news},
		{pageID: elsewhere.ID, terms: news},
	} {
		if err := repo.ReplaceTerms(t.Context(), step.pageID, step.terms); err != nil {
			t.Fatalf("ReplaceTerms: %v", err)
		}
	}

	got, err := repo.ListTerms(t.Context(), product.ID)
	if err != nil || !reflect.DeepEqual(got, sorted) {
		t.Fatalf("ListTerms = %+v, %v\nwant %+v", got, err, sorted)
	}
	bySite, err := repo.ListTermsBySite(t.Context(), owner.ID)
	if err != nil || !reflect.DeepEqual(bySite, map[string][]pagemap.Term{product.ID: sorted, post.ID: news}) {
		t.Fatalf("ListTermsBySite = %+v, %v", bySite, err)
	}

	if err = repo.ReplaceTerms(t.Context(), product.ID, terms[2:]); err != nil {
		t.Fatalf("ReplaceTerms with fewer: %v", err)
	}
	if got, err = repo.ListTerms(t.Context(), product.ID); err != nil || !reflect.DeepEqual(got, terms[2:]) {
		t.Fatalf("ListTerms after fewer = %+v, %v", got, err)
	}

	twice := []pagemap.Term{news[0], news[0]}
	if err = repo.ReplaceTerms(t.Context(), post.ID, twice); !errors.IsCode(err, errors.Conflict) {
		t.Fatalf("the same term twice code = %q, want CONFLICT", errors.CodeOf(err))
	}
	if got, err = repo.ListTerms(t.Context(), post.ID); err != nil || !reflect.DeepEqual(got, news) {
		t.Fatalf("a refused replace changed the terms to %+v, %v", got, err)
	}
	if err = repo.ReplaceTerms(t.Context(), "no-such-page", news); !errors.IsCode(err, errors.Invalid) {
		t.Errorf("the terms of no page code = %q, want INVALID", errors.CodeOf(err))
	}

	if err = repo.ReplaceTerms(t.Context(), product.ID, nil); err != nil {
		t.Fatalf("ReplaceTerms with none: %v", err)
	}
	if got, err = repo.ListTerms(t.Context(), product.ID); err != nil || len(got) != 0 {
		t.Fatalf("ListTerms after clearing = %+v, %v", got, err)
	}
	if bySite, err = repo.ListTermsBySite(t.Context(), owner.ID); err != nil || !reflect.DeepEqual(bySite, map[string][]pagemap.Term{post.ID: news}) {
		t.Fatalf("ListTermsBySite after clearing = %+v, %v", bySite, err)
	}

	if err = sqlite.NewPageRepo(store).Delete(t.Context(), post.ID); err != nil {
		t.Fatalf("delete the page: %v", err)
	}
	if got, err = repo.ListTerms(t.Context(), post.ID); err != nil || len(got) != 0 {
		t.Errorf("ListTerms after the page is deleted = %+v, %v", got, err)
	}
}
