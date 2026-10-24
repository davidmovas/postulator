package sqlite_test

import (
	"reflect"
	"testing"
	"time"

	"github.com/davidmovas/postulator/internal/adapters/sqlite"
	"github.com/davidmovas/postulator/internal/adapters/sqlite/sqlitetest"
	"github.com/davidmovas/postulator/internal/domain/field"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

func TestPageFieldRepoObserveKeepsTheBase(t *testing.T) {
	t.Parallel()

	store := sqlitetest.Open(t)
	owner := sqlitetest.Site(t, store, "shop")
	other := sqlitetest.Site(t, store, "other")
	product := sqlitetest.Page(t, store, owner.ID, "/tb-500/")
	post := sqlitetest.Page(t, store, owner.ID, "/news/")
	elsewhere := sqlitetest.Page(t, store, other.ID, "/elsewhere/")
	repo := sqlite.NewPageFieldRepo(store)

	if got, err := repo.ListForPage(t.Context(), product.ID); err != nil || len(got) != 0 {
		t.Fatalf("ListForPage on a fresh page = %+v, %v", got, err)
	}

	variation := field.Instance(field.VariationDescription, "42")
	title := field.NewValue(product.ID, field.Title, "TB-500", sqlitetest.Stamp)
	excerpt := field.NewValue(product.ID, field.Excerpt, "", sqlitetest.Stamp)
	described := field.NewValue(product.ID, variation, "The 5 mg vial.", sqlitetest.Stamp)
	if err := repo.Observe(t.Context(), product.ID, []field.Value{title, excerpt, described}); err != nil {
		t.Fatalf("Observe: %v", err)
	}
	want := []field.Value{excerpt, title, described}
	if got, err := repo.ListForPage(t.Context(), product.ID); err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("ListForPage = %+v, %v\nwant %+v", got, err, want)
	}

	later := sqlitetest.Stamp.Add(time.Hour)
	edited := field.NewValue(product.ID, field.Title, "TB-500 Peptide", later)
	if err := repo.Observe(t.Context(), product.ID, []field.Value{edited}); err != nil {
		t.Fatalf("Observe again: %v", err)
	}
	moved := title
	moved.Current, moved.CurrentHash, moved.ObservedAt = edited.Current, edited.CurrentHash, later
	want = []field.Value{excerpt, moved, described}
	got, err := repo.ListForPage(t.Context(), product.ID)
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("after a second Observe = %+v, %v\nwant %+v", got, err, want)
	}
	if !got[1].Changed() || got[1].Base != "TB-500" {
		t.Fatalf("the title lost its base: %+v", got[1])
	}

	news := field.NewValue(post.ID, field.SEOTitle, "News | Shop", sqlitetest.Stamp)
	for _, step := range []struct {
		pageID string
		value  field.Value
	}{
		{pageID: post.ID, value: news},
		{pageID: elsewhere.ID, value: field.NewValue(elsewhere.ID, field.Title, "Elsewhere", sqlitetest.Stamp)},
	} {
		if err = repo.Observe(t.Context(), step.pageID, []field.Value{step.value}); err != nil {
			t.Fatalf("Observe another page: %v", err)
		}
	}
	bySite, err := repo.ListBySite(t.Context(), owner.ID)
	if err != nil || !reflect.DeepEqual(bySite, map[string][]field.Value{product.ID: want, post.ID: {news}}) {
		t.Fatalf("ListBySite = %+v, %v", bySite, err)
	}

	if err = repo.Observe(t.Context(), product.ID, []field.Value{news}); !errors.IsCode(err, errors.Invalid) {
		t.Errorf("a value of another page code = %q, want INVALID", errors.CodeOf(err))
	}
	if err = repo.Observe(t.Context(), "no-such-page", []field.Value{field.NewValue("no-such-page", field.Title, "x", later)}); !errors.IsCode(err, errors.Invalid) {
		t.Errorf("the fields of no page code = %q, want INVALID", errors.CodeOf(err))
	}

	if err = sqlite.NewPageRepo(store).Delete(t.Context(), product.ID); err != nil {
		t.Fatalf("delete the page: %v", err)
	}
	if got, err = repo.ListForPage(t.Context(), product.ID); err != nil || len(got) != 0 {
		t.Errorf("ListForPage after the page is deleted = %+v, %v", got, err)
	}
}

func TestPageFieldRepoAgreeMovesTheBase(t *testing.T) {
	t.Parallel()

	store := sqlitetest.Open(t)
	owner := sqlitetest.Site(t, store, "shop")
	page := sqlitetest.Page(t, store, owner.ID, "/tb-500/")
	repo := sqlite.NewPageFieldRepo(store)

	priced := field.NewValue(page.ID, field.Price, "39.00", sqlitetest.Stamp)
	sku := field.NewValue(page.ID, field.SKU, "TB-5", sqlitetest.Stamp)
	if err := repo.Observe(t.Context(), page.ID, []field.Value{priced, sku}); err != nil {
		t.Fatalf("Observe: %v", err)
	}
	repriced := field.NewValue(page.ID, field.Price, "35.00", sqlitetest.Stamp.Add(time.Minute))
	if err := repo.Observe(t.Context(), page.ID, []field.Value{repriced}); err != nil {
		t.Fatalf("Observe a new price: %v", err)
	}

	agreed := sqlitetest.Stamp.Add(time.Hour)
	if err := repo.Agree(t.Context(), page.ID, field.Price, agreed); err != nil {
		t.Fatalf("Agree: %v", err)
	}
	moved := repriced
	moved.ObservedAt = agreed
	want := []field.Value{moved, sku}
	got, err := repo.ListForPage(t.Context(), page.ID)
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("ListForPage after Agree = %+v, %v\nwant %+v", got, err, want)
	}
	if got[0].Changed() {
		t.Fatalf("an agreed field still reads as changed: %+v", got[0])
	}

	if err = repo.Agree(t.Context(), page.ID, field.StockStatus, agreed); !errors.IsCode(err, errors.NotFound) {
		t.Errorf("agreeing a field never observed code = %q, want NOT_FOUND", errors.CodeOf(err))
	}
	if err = repo.Agree(t.Context(), "no-such-page", field.Price, agreed); !errors.IsCode(err, errors.NotFound) {
		t.Errorf("agreeing a field of no page code = %q, want NOT_FOUND", errors.CodeOf(err))
	}
}

func TestPageFieldRepoRecordSetsBothSidesAndTheRun(t *testing.T) {
	t.Parallel()

	store := sqlitetest.Open(t)
	owner := sqlitetest.Site(t, store, "shop")
	page := sqlitetest.Page(t, store, owner.ID, "/tb-500/")
	repo := sqlite.NewPageFieldRepo(store)

	edited := sqlitetest.Stamp.Add(time.Minute)
	for _, value := range []field.Value{
		field.NewValue(page.ID, field.SEOTitle, "Theirs", sqlitetest.Stamp),
		field.NewValue(page.ID, field.SEOTitle, "Theirs, edited", edited),
	} {
		if err := repo.Observe(t.Context(), page.ID, []field.Value{value}); err != nil {
			t.Fatalf("Observe: %v", err)
		}
	}

	written := sqlitetest.Stamp.Add(time.Hour)
	run := "run-1"
	if err := repo.Record(t.Context(), page.ID, field.SEOTitle, "Ours", run, written); err != nil {
		t.Fatalf("Record: %v", err)
	}
	if err := repo.Record(t.Context(), page.ID, field.SEODescription, "Ours too", run, written); err != nil {
		t.Fatalf("Record a field never observed: %v", err)
	}
	hash := field.Hash("Ours")
	want := []field.Value{
		{PageID: page.ID, Key: field.SEODescription, Current: "Ours too", CurrentHash: field.Hash("Ours too"), Base: "Ours too", BaseHash: field.Hash("Ours too"),
			ObservedAt: written, WrittenAt: &written, WrittenRunID: &run},
		{PageID: page.ID, Key: field.SEOTitle, Current: "Ours", CurrentHash: hash, Base: "Ours", BaseHash: hash,
			ObservedAt: edited, WrittenAt: &written, WrittenRunID: &run},
	}
	got, err := repo.ListForPage(t.Context(), page.ID)
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("ListForPage after Record = %+v, %v\nwant %+v", got, err, want)
	}
	if got[1].Changed() {
		t.Fatalf("our own write reads as changed on the site: %+v", got[1])
	}

	if err = repo.Record(t.Context(), "no-such-page", field.SEOTitle, "Ours", run, written); !errors.IsCode(err, errors.Invalid) {
		t.Errorf("recording a field of no page code = %q, want INVALID", errors.CodeOf(err))
	}
}

func TestAFieldTheSiteChangedIsTakenAsTheClientsWhenAPersonAcceptsIt(t *testing.T) {
	t.Parallel()

	store := sqlitetest.Open(t)
	owner := sqlitetest.Site(t, store, "shop")
	page := sqlitetest.Page(t, store, owner.ID, "/tb-500/")
	repo := sqlite.NewPageFieldRepo(store)

	if err := repo.Record(t.Context(), page.ID, field.Title, "Ours", "run-1", sqlitetest.Stamp); err != nil {
		t.Fatalf("Record: %v", err)
	}
	edited := field.NewValue(page.ID, field.Title, "The client's own", sqlitetest.Stamp.Add(time.Hour))
	if err := repo.Observe(t.Context(), page.ID, []field.Value{edited}); err != nil {
		t.Fatalf("Observe: %v", err)
	}
	accepted := sqlitetest.Stamp.Add(2 * time.Hour)
	if err := repo.AcceptSite(t.Context(), page.ID, field.Title, accepted); err != nil {
		t.Fatalf("AcceptSite: %v", err)
	}
	got, err := repo.ListForPage(t.Context(), page.ID)
	if err != nil || len(got) != 1 {
		t.Fatalf("ListForPage = %+v, %v", got, err)
	}
	if title := got[0]; title.Base != "The client's own" || title.Changed() || title.WrittenRunID != nil || title.WrittenAt != nil {
		t.Errorf("after AcceptSite the title is %+v, want the client's value as the base and no run of ours behind it, or the next run writes over it", title)
	}
	if err = repo.AcceptSite(t.Context(), page.ID, field.Excerpt, accepted); !errors.IsCode(err, errors.NotFound) {
		t.Errorf("accepting a field the page never had code = %q, want NOT_FOUND", errors.CodeOf(err))
	}
}
