package wp_test

import (
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/davidmovas/postulator/internal/adapters/wp"
	"github.com/davidmovas/postulator/internal/adapters/wp/wptest"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

const (
	koffeinBody = "<p>Koffein ist ein Alkaloid.</p>"
	koffeinHash = "119b7cff7356b21d2b00e64d2d3c0589b50f3270b302a94360fac92f1adc332b"
	powderHash  = "79db24b7a931978a1db05ccdb1ae1ed16b07aafa5277af75a7dc1b3b7ec6a509"
)

func TestCapabilitiesAreFetchedOnceAndCached(t *testing.T) {
	t.Parallel()

	server := wptest.New(t, wptest.WithSEOPlugin("rankmath"))
	client := newClient(t, server)

	first, err := client.Capabilities(t.Context())
	if err != nil {
		t.Fatalf("Capabilities: %v", err)
	}
	if first.SEOPlugin != "rankmath" || first.Version != "1.2.0" || first.Site != server.URL() {
		t.Errorf("capabilities = %+v", first)
	}
	if !first.Has("raw") || !first.Has("seo_meta") || !first.Has(wp.CapabilityPreview) || first.Has("telepathy") {
		t.Errorf("names = %v", first.Names)
	}
	if !first.Has(wp.CapabilitySEOMetaRead) {
		t.Errorf("names = %v, want the read of the SEO meta", first.Names)
	}

	if _, err = client.Capabilities(t.Context()); err != nil {
		t.Fatalf("Capabilities again: %v", err)
	}
	if got := len(server.Requests()); got != 1 {
		t.Errorf("the client asked for the manifest %d times, want 1", got)
	}
}

func TestEveryPluginMethodDegradesWhenThePluginIsAbsent(t *testing.T) {
	t.Parallel()

	server := wptest.New(t, wptest.WithoutPlugin())
	seeded := server.Seed(wptest.Item{Type: wptest.TypePage, Title: "Koffein", Content: koffeinBody})
	client := newClient(t, server)

	calls := []struct {
		name string
		call func() error
	}{
		{name: "manifest", call: func() error { _, err := client.Manifest(t.Context()); return err }},
		{name: "capabilities", call: func() error { _, err := client.Capabilities(t.Context()); return err }},
		{name: "content", call: func() error { _, err := client.ListContent(t.Context(), wp.ContentQuery{}); return err }},
		{name: "seo meta", call: func() error {
			_, err := client.SetSEOMeta(t.Context(), seeded[0].ID, wp.SEOMeta{Title: "x"})
			return err
		}},
		{name: "seo meta read", call: func() error {
			_, err := client.GetSEOMeta(t.Context(), seeded[0].ID)
			return err
		}},
		{name: "seo meta replace", call: func() error {
			_, err := client.ReplaceSEOMeta(t.Context(), seeded[0].ID, wp.SEOMeta{}, []string{"title"})
			return err
		}},
		{name: "raw read", call: func() error { _, err := client.GetRaw(t.Context(), seeded[0].ID); return err }},
		{name: "raw write", call: func() error {
			_, err := client.PutRaw(t.Context(), seeded[0].ID, "x", "")
			return err
		}},
		{name: "preview", call: func() error { _, err := client.PreviewLink(t.Context(), seeded[0].ID); return err }},
	}

	for _, tc := range calls {
		err := tc.call()
		if !errors.IsCode(err, errors.Invalid) {
			t.Errorf("%s code = %q, want %q", tc.name, errors.CodeOf(err), errors.Invalid)
		}
		if got := detailOf(t, err, "code"); got != "plugin_missing" {
			t.Errorf("%s detail = %q, want plugin_missing", tc.name, got)
		}
	}

	if got := len(server.Requests()); got != 1 {
		t.Errorf("the client probed the manifest %d times, want 1; absence is cached", got)
	}
}

func TestATransientManifestFailureIsNotCached(t *testing.T) {
	t.Parallel()

	server := wptest.New(t)
	server.FailNext(http.StatusInternalServerError, 1)
	client := newClient(t, server, wp.WithRetries(0))

	if _, err := client.Capabilities(t.Context()); !errors.IsCode(err, errors.External) {
		t.Fatalf("code = %q, want %q", errors.CodeOf(err), errors.External)
	}
	if _, err := client.Capabilities(t.Context()); err != nil {
		t.Errorf("a site that recovered must be usable again: %v", err)
	}
}

func TestListContentWalksTheOpaqueCursor(t *testing.T) {
	t.Parallel()

	server := wptest.New(t)
	server.Seed(
		wptest.Item{Type: wptest.TypePage, Title: "One"},
		wptest.Item{Type: wptest.TypePage, Title: "Two"},
		wptest.Item{Type: wptest.TypePage, Title: "Three"},
	)
	client := newClient(t, server)

	seen := 0
	query := wp.ContentQuery{Types: []wp.ItemType{wp.TypePage}, Limit: 2}
	for {
		page, err := client.ListContent(t.Context(), query)
		if err != nil {
			t.Fatalf("ListContent: %v", err)
		}
		seen += len(page.Items)
		if page.NextCursor == nil {
			break
		}
		query.Cursor = *page.NextCursor
	}

	if seen != 3 {
		t.Errorf("walked %d items, want 3", seen)
	}
}

func TestListContentCarriesTheWholeItem(t *testing.T) {
	t.Parallel()

	const powderContent = `<h1>Powder</h1>` + koffeinBody + `<p><a href="/koffein/">Koffein</a></p>`

	server := wptest.New(t)
	parent := server.Seed(wptest.Item{Type: wptest.TypePage, Title: "Koffein"})[0]
	server.Seed(wptest.Item{
		Type:    wptest.TypePage,
		Title:   "Powder",
		Parent:  parent.ID,
		Content: powderContent,
		Meta:    map[string]string{"_yoast_wpseo_title": "Powder | Koffein"},
	})

	page, err := newClient(t, server).ListContent(t.Context(), wp.ContentQuery{Types: []wp.ItemType{wp.TypePage}})
	if err != nil {
		t.Fatalf("ListContent: %v", err)
	}
	if len(page.Items) != 2 || page.NextCursor != nil {
		t.Fatalf("page = %+v", page)
	}

	child := page.Items[1]
	if child.Path != "/koffein/powder/" || child.Parent != parent.ID || child.Type != wp.TypePage {
		t.Errorf("item = %+v", child)
	}
	if child.Title != "Powder" || child.H1 != "Powder" || child.Status != "publish" {
		t.Errorf("item = %+v", child)
	}
	if child.ContentHash != wp.ContentHash(powderContent) {
		t.Errorf("contentHash = %q, want the hash of the raw content", child.ContentHash)
	}
	if child.Modified.IsZero() {
		t.Error("the item must carry a modification instant")
	}
	if child.Meta.Title != "Powder | Koffein" {
		t.Errorf("meta = %+v", child.Meta)
	}
	if len(child.Links) != 1 || child.Links[0].Href != "/koffein/" || child.Links[0].Anchor != "Koffein" {
		t.Errorf("links = %+v", child.Links)
	}
}

func TestTheContentQueryClampsTheLimitAndNamesTheTypes(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		query wp.ContentQuery
		limit string
	}{
		{name: "absent", query: wp.ContentQuery{}, limit: "100"},
		{name: "negative", query: wp.ContentQuery{Limit: -1}, limit: "100"},
		{name: "above the maximum", query: wp.ContentQuery{Limit: 99999}, limit: "500"},
		{name: "in range", query: wp.ContentQuery{Limit: 25}, limit: "25"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			server := wptest.New(t)
			if _, err := newClient(t, server).ListContent(t.Context(), tc.query); err != nil {
				t.Fatalf("ListContent: %v", err)
			}

			recorded, _ := server.LastRequest()
			if got := recorded.Query.Get("limit"); got != tc.limit {
				t.Errorf("limit = %q, want %q", got, tc.limit)
			}
		})
	}
}

func TestTheContentQuerySendsSinceAndTypes(t *testing.T) {
	t.Parallel()

	server := wptest.New(t)
	since := time.Date(2026, time.September, 18, 9, 0, 0, 0, time.UTC)

	if _, err := newClient(t, server).ListContent(t.Context(), wp.ContentQuery{
		Since: &since,
		Types: []wp.ItemType{wp.TypePage, wp.TypeProductCategory},
	}); err != nil {
		t.Fatalf("ListContent: %v", err)
	}

	recorded, _ := server.LastRequest()
	if got := recorded.Query.Get("since"); got != "2026-09-18T09:00:00Z" {
		t.Errorf("since = %q", got)
	}
	if got := recorded.Query.Get("types"); got != "page,product_cat" {
		t.Errorf("types = %q", got)
	}
}

func TestSetSEOMetaReportsWhatWasWritten(t *testing.T) {
	t.Parallel()

	server := wptest.New(t, wptest.WithSEOPlugin("yoast"))
	seeded := server.Seed(wptest.Item{Type: wptest.TypePage, Title: "Koffein"})
	client := newClient(t, server)

	result, err := client.SetSEOMeta(t.Context(), seeded[0].ID, wp.SEOMeta{
		Title:       "Koffein",
		Description: "about it",
	})
	if err != nil {
		t.Fatalf("SetSEOMeta: %v", err)
	}
	if result.SEOPlugin != "yoast" {
		t.Errorf("seoPlugin = %q", result.SEOPlugin)
	}
	if len(result.Applied) != 2 || result.Applied[0] != "title" || result.Applied[1] != "description" {
		t.Errorf("applied = %v", result.Applied)
	}

	stored, _ := server.Lookup(seeded[0].ID)
	if stored.Meta["_yoast_wpseo_metadesc"] != "about it" {
		t.Errorf("meta = %v", stored.Meta)
	}

	if _, err = client.SetSEOMeta(t.Context(), seeded[0].ID, wp.SEOMeta{}); !errors.IsCode(err, errors.Invalid) {
		t.Errorf("code = %q, want %q for an empty update", errors.CodeOf(err), errors.Invalid)
	}
}

func TestGetSEOMetaReadsTheFiveFieldsTheSiteHolds(t *testing.T) {
	t.Parallel()

	server := wptest.New(t, wptest.WithSEOPlugin("yoast"))
	seeded := server.Seed(wptest.Item{Type: wptest.TypePage, Title: "Koffein"})
	client := newClient(t, server)

	before, err := client.GetSEOMeta(t.Context(), seeded[0].ID)
	if err != nil {
		t.Fatalf("GetSEOMeta: %v", err)
	}
	if before != (wp.SEOMeta{}) {
		t.Fatalf("a post with no meta reads as %+v", before)
	}

	if _, err = client.SetSEOMeta(t.Context(), seeded[0].ID, wp.SEOMeta{
		Title: "Koffein", Description: "about it",
	}); err != nil {
		t.Fatalf("SetSEOMeta: %v", err)
	}

	after, err := client.GetSEOMeta(t.Context(), seeded[0].ID)
	if err != nil {
		t.Fatalf("GetSEOMeta again: %v", err)
	}
	if after.Title != "Koffein" || after.Description != "about it" || after.Canonical != "" {
		t.Fatalf("the read = %+v", after)
	}
}

func TestReplaceSEOMetaPutsBackTheFieldsItIsGiven(t *testing.T) {
	t.Parallel()

	server := wptest.New(t, wptest.WithSEOPlugin("yoast"))
	seeded := server.Seed(wptest.Item{Type: wptest.TypePage, Title: "Koffein"})
	client := newClient(t, server)

	if _, err := client.SetSEOMeta(t.Context(), seeded[0].ID, wp.SEOMeta{
		Title: "written by a run", Description: "written by a run", Canonical: "kept by a human",
	}); err != nil {
		t.Fatalf("SetSEOMeta: %v", err)
	}

	result, err := client.ReplaceSEOMeta(t.Context(), seeded[0].ID, wp.SEOMeta{Title: "what was there"},
		[]string{"title", "description"})
	if err != nil {
		t.Fatalf("ReplaceSEOMeta: %v", err)
	}
	if len(result.Applied) != 2 {
		t.Fatalf("applied = %v, want both named fields", result.Applied)
	}

	after, err := client.GetSEOMeta(t.Context(), seeded[0].ID)
	if err != nil {
		t.Fatalf("GetSEOMeta: %v", err)
	}
	if after.Title != "what was there" {
		t.Errorf("title = %q, want the value it is given", after.Title)
	}
	if after.Description != "" {
		t.Errorf("description = %q, want an empty value to clear the field", after.Description)
	}
	if after.Canonical != "kept by a human" {
		t.Errorf("canonical = %q, want a field nobody named left alone", after.Canonical)
	}

	if _, err = client.ReplaceSEOMeta(t.Context(), seeded[0].ID, wp.SEOMeta{}, []string{"shade"}); !errors.IsCode(err, errors.Invalid) {
		t.Errorf("a field the meta has no place for = %v, want invalid", err)
	}
	if _, err = client.ReplaceSEOMeta(t.Context(), seeded[0].ID, wp.SEOMeta{}, nil); !errors.IsCode(err, errors.Invalid) {
		t.Errorf("a replacement of nothing = %v, want invalid", err)
	}
}

func TestGetSEOMetaRefusesAPluginWithoutTheCapability(t *testing.T) {
	t.Parallel()

	server := wptest.New(t, wptest.WithCapabilities("bulk", "seo_meta", "content_hash", "raw", "preview"))
	seeded := server.Seed(wptest.Item{Type: wptest.TypePage, Title: "Koffein"})
	client := newClient(t, server)

	_, err := client.GetSEOMeta(t.Context(), seeded[0].ID)
	if !wp.IsPluginOutdated(err) {
		t.Fatalf("GetSEOMeta on a 1.1.0 plugin = %v, want plugin_outdated", err)
	}
	if got := detailOf(t, err, "capability"); got != wp.CapabilitySEOMetaRead {
		t.Errorf("the refusal names %q", got)
	}
	if got := len(server.Requests()); got != 1 {
		t.Errorf("the client asked the site %d times, want only the manifest", got)
	}
}

func TestTheRawRoundTripIsGuardedByTheHash(t *testing.T) {
	t.Parallel()

	server := wptest.New(t)
	seeded := server.Seed(wptest.Item{Type: wptest.TypePage, Title: "Koffein", Content: koffeinBody})
	client := newClient(t, server)

	raw, err := client.GetRaw(t.Context(), seeded[0].ID)
	if err != nil {
		t.Fatalf("GetRaw: %v", err)
	}
	if raw.Content != koffeinBody || raw.ContentHash != koffeinHash || raw.Type != wp.TypePage {
		t.Fatalf("raw = %+v", raw)
	}

	written, err := client.PutRaw(t.Context(), seeded[0].ID, "<p>Powder</p>", raw.ContentHash)
	if err != nil {
		t.Fatalf("PutRaw: %v", err)
	}
	if written != powderHash {
		t.Errorf("hash = %q, want %q", written, powderHash)
	}

	_, err = client.PutRaw(t.Context(), seeded[0].ID, "<p>again</p>", raw.ContentHash)
	if !errors.IsCode(err, errors.Conflict) {
		t.Fatalf("code = %q, want %q", errors.CodeOf(err), errors.Conflict)
	}
	if got := detailOf(t, err, "currentHash"); got != powderHash {
		t.Errorf("currentHash detail = %q, want %q", got, powderHash)
	}

	unconditional, err := client.PutRaw(t.Context(), seeded[0].ID, koffeinBody, "")
	if err != nil {
		t.Fatalf("PutRaw without a hash: %v", err)
	}
	if unconditional != koffeinHash {
		t.Errorf("hash = %q, want %q", unconditional, koffeinHash)
	}
}

func TestGetRawRefusesAHashThatDoesNotMatchTheContent(t *testing.T) {
	t.Parallel()

	server := wptest.New(t, wptest.WithBrokenContentHash())
	seeded := server.Seed(wptest.Item{Type: wptest.TypePage, Title: "Koffein", Content: koffeinBody})

	_, err := newClient(t, server).GetRaw(t.Context(), seeded[0].ID)
	if !errors.IsCode(err, errors.External) {
		t.Errorf("code = %q, want %q", errors.CodeOf(err), errors.External)
	}
}

func TestThePostOnlyRoutesReportATermAsMissing(t *testing.T) {
	t.Parallel()

	server := wptest.New(t)
	term := server.Seed(wptest.Item{Type: wptest.TypeProductCategory, Title: "Koffein"})[0]
	client := newClient(t, server)

	if _, err := client.GetRaw(t.Context(), term.ID); !errors.IsCode(err, errors.NotFound) {
		t.Errorf("raw read code = %q, want %q", errors.CodeOf(err), errors.NotFound)
	}
	if _, err := client.PutRaw(t.Context(), term.ID, "x", ""); !errors.IsCode(err, errors.NotFound) {
		t.Errorf("raw write code = %q, want %q", errors.CodeOf(err), errors.NotFound)
	}
	if _, err := client.SetSEOMeta(t.Context(), term.ID, wp.SEOMeta{Title: "x"}); !errors.IsCode(err, errors.NotFound) {
		t.Errorf("seo meta code = %q, want %q", errors.CodeOf(err), errors.NotFound)
	}
	if _, err := client.PreviewLink(t.Context(), term.ID); !errors.IsCode(err, errors.NotFound) {
		t.Errorf("preview code = %q, want %q", errors.CodeOf(err), errors.NotFound)
	}
}

func TestPreviewLinkIssuesTheURLAndTheExpiry(t *testing.T) {
	t.Parallel()

	server := wptest.New(t)
	draft := server.Seed(wptest.Item{Type: wptest.TypePage, Title: "Koffein", Status: "draft"})[0]
	client := newClient(t, server)

	link, err := client.PreviewLink(t.Context(), draft.ID)
	if err != nil {
		t.Fatalf("PreviewLink: %v", err)
	}
	parsed, err := url.Parse(link.URL)
	if err != nil {
		t.Fatalf("parse %s: %v", link.URL, err)
	}
	if !strings.HasPrefix(link.URL, server.URL()) || parsed.Query().Get("preview") != "true" ||
		parsed.Query().Get("postulator_preview") == "" {
		t.Errorf("url = %q", link.URL)
	}
	stored, _ := server.Lookup(draft.ID)
	if !link.ExpiresAt.Equal(stored.PreviewExpires) {
		t.Errorf("expires = %s, want %s", link.ExpiresAt, stored.PreviewExpires)
	}

	last, _ := server.LastRequest()
	if last.Method != http.MethodPost || last.Path != "/wp-json/postulator/v1/content/"+strconv.FormatInt(draft.ID, 10)+"/preview" {
		t.Errorf("last request = %s %s", last.Method, last.Path)
	}
}

func TestPreviewLinkRefusesAPluginWithoutTheCapability(t *testing.T) {
	t.Parallel()

	server := wptest.New(t, wptest.WithCapabilities("bulk", "seo_meta", "content_hash", "raw"))
	draft := server.Seed(wptest.Item{Type: wptest.TypePage, Title: "Koffein", Status: "draft"})[0]
	client := newClient(t, server)

	_, err := client.PreviewLink(t.Context(), draft.ID)
	if !errors.IsCode(err, errors.Invalid) || !wp.IsPluginOutdated(err) {
		t.Fatalf("PreviewLink = %v, want a plugin_outdated refusal", err)
	}
	if got := detailOf(t, err, "capability"); got != wp.CapabilityPreview {
		t.Errorf("capability = %q, want %q", got, wp.CapabilityPreview)
	}
	if wp.IsPluginMissing(err) {
		t.Errorf("an outdated plugin reads as a missing one")
	}
	if got := len(server.Requests()); got != 1 {
		t.Errorf("the client sent %d requests, want the manifest alone", got)
	}
}

func TestPreviewLinkRefusesAnUnreadableExpiry(t *testing.T) {
	t.Parallel()

	server := wptest.New(t, wptest.WithBrokenPreviewExpiry())
	draft := server.Seed(wptest.Item{Type: wptest.TypePage, Title: "Koffein", Status: "draft"})[0]
	client := newClient(t, server)

	if _, err := client.PreviewLink(t.Context(), draft.ID); !errors.IsCode(err, errors.External) {
		t.Fatalf("PreviewLink = %v, want an external failure", err)
	}
}

func TestInvalidateManifestReprobesASiteThatGainedThePlugin(t *testing.T) {
	t.Parallel()

	server := wptest.New(t, wptest.WithoutPlugin())
	client := newClient(t, server)

	if _, err := client.Capabilities(t.Context()); !wp.IsPluginMissing(err) {
		t.Fatalf("Capabilities without the plugin = %v, want a plugin_missing failure", err)
	}

	server.EnablePlugin()
	if _, err := client.Capabilities(t.Context()); !wp.IsPluginMissing(err) {
		t.Fatalf("Capabilities = %v, want the cached plugin_missing failure until it is invalidated", err)
	}

	client.InvalidateManifest()
	capabilities, err := client.Capabilities(t.Context())
	if err != nil {
		t.Fatalf("Capabilities after the plugin was installed: %v", err)
	}
	if !capabilities.Has("bulk") || !capabilities.Has("seo_meta") {
		t.Errorf("names = %v, want the companion capabilities", capabilities.Names)
	}
}

func TestInvalidateManifestDropsAManifestThatWasRead(t *testing.T) {
	t.Parallel()

	server := wptest.New(t)
	client := newClient(t, server)

	if _, err := client.Capabilities(t.Context()); err != nil {
		t.Fatalf("Capabilities: %v", err)
	}
	client.InvalidateManifest()
	if _, err := client.Capabilities(t.Context()); err != nil {
		t.Fatalf("Capabilities again: %v", err)
	}
	if got := len(server.Requests()); got != 2 {
		t.Errorf("the client asked for the manifest %d times, want 2", got)
	}
}
