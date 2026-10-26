package sitemodel_test

import (
	"strings"
	"testing"
	"unicode/utf8"

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

func TestABodyReadsAsTextBlockByBlockAndALongBlockInParts(t *testing.T) {
	t.Parallel()

	m := site1()
	long := strings.Repeat("Пептид хранят в холоде. ", 400)
	raw := `<p>Read the <a href="/tb-500/">TB-500 guide</a> and <a href="https://elsewhere.example.org/">a study</a>.</p>` +
		"\n\n<p>" + long + "</p>\n\n<p>Closing words.</p>"
	m.Bodies[0].Raw, m.Bodies[0].Hash = raw, block.BodyHash(raw)

	var rows []sitemodel.BlockRow
	req := sitemodel.BlocksRequest{SiteID: siteID, PageID: "bpc", Text: true, Limit: 2}
	for {
		got, err := m.Service().Blocks(t.Context(), req)
		if err != nil {
			t.Fatalf("Blocks: %v", err)
		}
		rows = append(rows, got.Blocks.Items...)
		if !got.Blocks.HasMore {
			break
		}
		req.Cursor = string(got.Blocks.Next)
	}

	parts := map[int][]sitemodel.BlockRow{}
	for _, row := range rows {
		parts[row.Index] = append(parts[row.Index], row)
	}
	if len(parts) != 3 || len(parts[0]) != 1 || len(parts[2]) != 1 || len(parts[1]) < 3 {
		t.Fatalf("the rows by block = %d, %d, %d; want the long block read in parts", len(parts[0]), len(parts[1]), len(parts[2]))
	}
	opening := parts[0][0]
	if opening.Text != "Read the TB-500 guide and a study." || len(opening.Links) != 2 {
		t.Fatalf("the opening reads %q with links %+v", opening.Text, opening.Links)
	}
	if guide := opening.Links[0]; guide.Anchor != "TB-500 guide" || guide.Path != "/tb-500/" {
		t.Errorf("the internal link = %+v, want its anchor and the page it points at", guide)
	}
	if study := opening.Links[1]; study.Path != "" || study.Href != "https://elsewhere.example.org/" {
		t.Errorf("the external link = %+v, want its href and no page", study)
	}
	var joined strings.Builder
	at := 0
	for _, part := range parts[1] {
		if part.From != at || part.Runes != utf8.RuneCountInString(strings.TrimSpace(long)) {
			t.Fatalf("a part starts at %d of %d runes, want %d", part.From, part.Runes, at)
		}
		joined.WriteString(part.Text)
		at += utf8.RuneCountInString(part.Text)
	}
	if joined.String() != strings.TrimSpace(long) {
		t.Errorf("the parts of the long block do not join back into its text")
	}
	if plain, err := m.Service().Blocks(t.Context(), sitemodel.BlocksRequest{SiteID: siteID, PageID: "bpc"}); err != nil ||
		len(plain.Blocks.Items) != 3 || plain.Blocks.Items[1].Text != "" || plain.Blocks.Items[1].Excerpt == "" {
		t.Errorf("without text the blocks = %+v, %v, want one row each with its excerpt", plain.Blocks.Items, err)
	}
}
