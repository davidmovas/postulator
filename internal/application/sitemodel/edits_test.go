package sitemodel_test

import (
	"testing"

	"github.com/davidmovas/postulator/internal/application/sitemodel"
	"github.com/davidmovas/postulator/internal/domain/block"
	"github.com/davidmovas/postulator/internal/domain/field"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

func written(value field.Value, run string) field.Value {
	value.WrittenRunID = &run
	return value
}

func TestTheFieldsOfAPageAreReadWithTheirBaseAndOwner(t *testing.T) {
	t.Parallel()

	m := site1()
	moved := written(field.NewValue("bpc", field.SEOTitle, "Older title", stamp), "run-1")
	moved.Current, moved.CurrentHash = "The owner's title", field.Hash("The owner's title")
	m.Fields["bpc"] = []field.Value{
		written(field.NewValue("bpc", field.Excerpt, "Our excerpt.", stamp), "run-1"),
		moved,
		field.NewValue("bpc", field.Title, "BPC-157", stamp),
	}
	got, err := m.Service().Fields(t.Context(), sitemodel.FieldsRequest{SiteID: siteID, PageID: "bpc"})
	if err != nil {
		t.Fatalf("Fields: %v", err)
	}
	owners := map[string]string{}
	for _, row := range got.Fields.Items {
		owners[row.Key] = row.Owner
		if row.Key == string(field.SEOTitle) && (row.Base != "Older title" || row.Hash != field.Hash("The owner's title")) {
			t.Errorf("the SEO title reads %+v, want its base and the hash of what the site holds", row)
		}
	}
	want := map[string]string{"excerpt": "ours", "seo_title": "client", "title": "client"}
	for key, owner := range want {
		if owners[key] != owner {
			t.Errorf("the %s belongs to %q, want %q (%v)", key, owners[key], owner, owners)
		}
	}

	store, err := m.Service().Fields(t.Context(), sitemodel.FieldsRequest{SiteID: siteID, PageID: "tb", Keys: []string{"price"}})
	if err != nil || len(store.Fields.Items) != 1 || store.Fields.Items[0].Owner != "store" || store.Fields.Items[0].Writable {
		t.Errorf("the product's price = %+v, %v, want the store's, never written", store.Fields.Items, err)
	}
	first, err := m.Service().Fields(t.Context(), sitemodel.FieldsRequest{SiteID: siteID, PageID: "bpc", Limit: 1})
	if err != nil || len(first.Fields.Items) != 1 || !first.Fields.HasMore {
		t.Errorf("a page of one = %+v, %v, want a cursor to the rest", first.Fields, err)
	}
	if _, err := m.Service().Fields(t.Context(), sitemodel.FieldsRequest{SiteID: "elsewhere", PageID: "bpc"}); !errors.IsCode(err, errors.NotFound) {
		t.Errorf("another site's page = %v, want NotFound", err)
	}
}

func TestTheBlocksOfAPageAreListedWithTheirOwnersAndTheBodysHash(t *testing.T) {
	t.Parallel()

	m := site1()
	theirs, ours := "<p>The owner's own opening.</p>", "<p>Our paragraph on storage.</p>"
	raw := theirs + "\n\n" + ours
	m.Bodies[0].Raw, m.Bodies[0].Hash = raw, block.BodyHash(raw)
	m.Written = []block.Written{{PageID: "bpc", BlockHash: block.Parse(raw).Blocks[1].Hash, RunID: "run-1"}}

	got, err := m.Service().Blocks(t.Context(), sitemodel.BlocksRequest{SiteID: siteID, Path: "/bpc-157/"})
	if err != nil {
		t.Fatalf("Blocks: %v", err)
	}
	if got.BodyHash != block.BodyHash(raw) || got.BodyKind != "classic" || !got.OnSite || got.Rewrites {
		t.Fatalf("the body = %+v, want its hash, its kind, on the site and its owner's text kept", got)
	}
	rows := got.Blocks.Items
	if len(rows) != 2 || rows[0].Owner != "client" || rows[1].Owner != "ours" || rows[1].Hash != block.Parse(raw).Blocks[1].Hash ||
		rows[0].Excerpt != "The owner's own opening." {
		t.Fatalf("the blocks = %+v, want the owner's then ours, with hashes and excerpts", rows)
	}
	first, err := m.Service().Blocks(t.Context(), sitemodel.BlocksRequest{SiteID: siteID, PageID: "bpc", Limit: 1})
	if err != nil || len(first.Blocks.Items) != 1 || !first.Blocks.HasMore {
		t.Errorf("a page of one block = %+v, %v, want a cursor to the rest", first.Blocks, err)
	}
}
