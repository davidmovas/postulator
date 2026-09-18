package wptest_test

import (
	"net/http"
	"testing"

	"github.com/davidmovas/postulator/internal/adapters/wp/wptest"
)

const koffeinHash = "119b7cff7356b21d2b00e64d2d3c0589b50f3270b302a94360fac92f1adc332b"

func TestTheManifestReportsTheFrozenCapabilities(t *testing.T) {
	t.Parallel()

	server := wptest.New(t, wptest.WithSEOPlugin("rankmath"))
	response, payload := call(t, server, http.MethodGet, "/wp-json/postulator/v1/manifest", nil, true)
	if response.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", response.StatusCode)
	}

	var manifest struct {
		Version      string   `json:"version"`
		Capabilities []string `json:"capabilities"`
		SEOPlugin    string   `json:"seoPlugin"`
		WPVersion    string   `json:"wpVersion"`
		Site         string   `json:"site"`
	}
	decode(t, payload, &manifest)

	want := []string{"bulk", "seo_meta", "content_hash", "raw"}
	if manifest.Version != "1.0.0" || len(manifest.Capabilities) != len(want) {
		t.Fatalf("manifest = %+v", manifest)
	}
	for index, capability := range want {
		if manifest.Capabilities[index] != capability {
			t.Errorf("capability %d = %q, want %q", index, manifest.Capabilities[index], capability)
		}
	}
	if manifest.SEOPlugin != "rankmath" || manifest.WPVersion == "" || manifest.Site != server.URL() {
		t.Errorf("manifest = %+v", manifest)
	}
}

func TestASiteWithoutThePluginHasNoRoutes(t *testing.T) {
	t.Parallel()

	server := wptest.New(t, wptest.WithoutPlugin())
	seeded := server.Seed(wptest.Item{Type: wptest.TypePage, Title: "Koffein"})

	cases := []struct {
		name   string
		method string
		path   string
		body   []byte
	}{
		{name: "manifest", method: http.MethodGet, path: "/wp-json/postulator/v1/manifest"},
		{name: "content", method: http.MethodGet, path: "/wp-json/postulator/v1/content"},
		{name: "seo meta", method: http.MethodPut, path: "/wp-json/postulator/v1/seo-meta/" + itoa(seeded[0].ID), body: []byte(`{"title":"x"}`)},
		{name: "raw read", method: http.MethodGet, path: "/wp-json/postulator/v1/content/" + itoa(seeded[0].ID) + "/raw"},
		{name: "raw write", method: http.MethodPut, path: "/wp-json/postulator/v1/content/" + itoa(seeded[0].ID) + "/raw", body: []byte(`{"content":"x"}`)},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			response, payload := call(t, server, tc.method, tc.path, tc.body, true)
			if response.StatusCode != http.StatusNotFound {
				t.Fatalf("status = %d, want 404", response.StatusCode)
			}

			var failure struct {
				Code string `json:"code"`
			}
			decode(t, payload, &failure)
			if failure.Code != "rest_no_route" {
				t.Errorf("code = %q, want rest_no_route", failure.Code)
			}
		})
	}
}

func TestContentCarriesPathsHashesHeadingsAndInternalLinks(t *testing.T) {
	t.Parallel()

	server := wptest.New(t)
	parent := server.Seed(wptest.Item{Type: wptest.TypePage, Title: "Koffein"})[0]
	server.Seed(wptest.Item{
		Type:    wptest.TypePage,
		Title:   "Powder",
		Parent:  parent.ID,
		Content: `<h1>Powder</h1><p>Koffein ist ein Alkaloid.</p><p><a href="/Koffein//">Koffein</a> and <a href="https://example.com/x">away</a> and <a href="mailto:a@b.c">mail</a></p>`,
	})

	_, payload := call(t, server, http.MethodGet, "/wp-json/postulator/v1/content?types=page", nil, true)

	var page struct {
		Items []struct {
			Path        string `json:"path"`
			Title       string `json:"title"`
			H1          string `json:"h1"`
			ContentHash string `json:"contentHash"`
			Modified    string `json:"modified"`
			Links       []struct {
				Href   string `json:"href"`
				Anchor string `json:"anchor"`
			} `json:"links"`
		} `json:"items"`
		NextCursor *string `json:"nextCursor"`
	}
	decode(t, payload, &page)

	if len(page.Items) != 2 {
		t.Fatalf("got %d items, want 2", len(page.Items))
	}
	if page.NextCursor != nil {
		t.Errorf("nextCursor = %v, want null on the last page", *page.NextCursor)
	}

	child := page.Items[1]
	if child.Path != "/koffein/powder/" {
		t.Errorf("path = %q, want /koffein/powder/", child.Path)
	}
	if child.H1 != "Powder" || child.Title != "Powder" {
		t.Errorf("item = %+v", child)
	}
	if len(child.Modified) < 20 || child.Modified[len(child.Modified)-1] != 'Z' {
		t.Errorf("modified = %q, want RFC3339 with a UTC offset", child.Modified)
	}
	if len(child.Links) != 1 {
		t.Fatalf("links = %+v, want only the internal one", child.Links)
	}
	if child.Links[0].Href != "/koffein/" || child.Links[0].Anchor != "Koffein" {
		t.Errorf("link = %+v, want a collapsed and slash-wrapped path", child.Links[0])
	}
}

func TestContentPagesWithAnOpaqueCursorAndAnExclusiveSince(t *testing.T) {
	t.Parallel()

	server := wptest.New(t)
	seeded := server.Seed(
		wptest.Item{Type: wptest.TypePage, Title: "One"},
		wptest.Item{Type: wptest.TypePage, Title: "Two"},
		wptest.Item{Type: wptest.TypePage, Title: "Three"},
	)

	var first struct {
		Items      []map[string]any `json:"items"`
		NextCursor *string          `json:"nextCursor"`
	}
	_, payload := call(t, server, http.MethodGet, "/wp-json/postulator/v1/content?limit=2", nil, true)
	decode(t, payload, &first)

	if len(first.Items) != 2 || first.NextCursor == nil {
		t.Fatalf("first page = %+v", first)
	}

	var second struct {
		Items      []map[string]any `json:"items"`
		NextCursor *string          `json:"nextCursor"`
	}
	_, payload = call(t, server, http.MethodGet, "/wp-json/postulator/v1/content?limit=2&cursor="+*first.NextCursor, nil, true)
	decode(t, payload, &second)

	if len(second.Items) != 1 || second.NextCursor != nil {
		t.Fatalf("second page = %+v", second)
	}

	since := seeded[1].Modified.UTC().Format("2006-01-02T15:04:05Z07:00")
	_, payload = call(t, server, http.MethodGet, "/wp-json/postulator/v1/content?since="+since, nil, true)
	decode(t, payload, &first)
	if len(first.Items) != 1 {
		t.Errorf("since returned %d items, want 1; the bound is exclusive", len(first.Items))
	}
}

func TestTheContentLimitIsClampedRatherThanRejected(t *testing.T) {
	t.Parallel()

	server := wptest.New(t)
	for index := range 3 {
		server.Seed(wptest.Item{Type: wptest.TypePage, Title: "Page " + string(rune('A'+index))})
	}

	cases := []struct {
		name  string
		query string
	}{
		{name: "above the maximum", query: "?limit=99999"},
		{name: "zero", query: "?limit=0"},
		{name: "not a number", query: "?limit=lots"},
		{name: "absent", query: ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			response, payload := call(t, server, http.MethodGet, "/wp-json/postulator/v1/content"+tc.query, nil, true)
			if response.StatusCode != http.StatusOK {
				t.Fatalf("status = %d, want 200", response.StatusCode)
			}

			var page struct {
				Items []map[string]any `json:"items"`
			}
			decode(t, payload, &page)
			if len(page.Items) != 3 {
				t.Errorf("got %d items, want all 3", len(page.Items))
			}
		})
	}
}

func TestSeoMetaWritesTheKeysOfTheDetectedPlugin(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		plugin string
		key    string
	}{
		{name: "yoast", plugin: "yoast", key: "_yoast_wpseo_title"},
		{name: "rank math", plugin: "rankmath", key: "rank_math_title"},
		{name: "no plugin", plugin: "none", key: "_postulator_title"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			server := wptest.New(t, wptest.WithSEOPlugin(tc.plugin))
			seeded := server.Seed(wptest.Item{Type: wptest.TypePage, Title: "Koffein"})

			body := []byte(`{"title":"Koffein","description":"about it","ogTitle":"Koffein og"}`)
			response, payload := call(t, server, http.MethodPut, "/wp-json/postulator/v1/seo-meta/"+itoa(seeded[0].ID), body, true)
			if response.StatusCode != http.StatusOK {
				t.Fatalf("status = %d, want 200", response.StatusCode)
			}

			var result struct {
				Applied   []string `json:"applied"`
				SEOPlugin string   `json:"seoPlugin"`
			}
			decode(t, payload, &result)

			if result.SEOPlugin != tc.plugin {
				t.Errorf("seoPlugin = %q, want %q", result.SEOPlugin, tc.plugin)
			}
			want := []string{"title", "description", "ogTitle"}
			if len(result.Applied) != len(want) {
				t.Fatalf("applied = %v, want %v", result.Applied, want)
			}
			for index, field := range want {
				if result.Applied[index] != field {
					t.Errorf("applied[%d] = %q, want %q", index, result.Applied[index], field)
				}
			}

			stored, _ := server.Lookup(seeded[0].ID)
			if stored.Meta[tc.key] != "Koffein" {
				t.Errorf("meta[%s] = %q, want Koffein", tc.key, stored.Meta[tc.key])
			}
		})
	}
}

func TestTheRawRoutesRoundTripAndGuardTheHash(t *testing.T) {
	t.Parallel()

	server := wptest.New(t)
	seeded := server.Seed(wptest.Item{Type: wptest.TypePage, Title: "Koffein", Content: "<p>Koffein ist ein Alkaloid.</p>"})

	_, payload := call(t, server, http.MethodGet, "/wp-json/postulator/v1/content/"+itoa(seeded[0].ID)+"/raw", nil, true)
	var raw struct {
		Content     string `json:"content"`
		ContentHash string `json:"contentHash"`
		Type        string `json:"type"`
	}
	decode(t, payload, &raw)

	if raw.Content != "<p>Koffein ist ein Alkaloid.</p>" || raw.ContentHash != koffeinHash {
		t.Fatalf("raw = %+v", raw)
	}
	if raw.Type != wptest.TypePage {
		t.Errorf("type = %q", raw.Type)
	}

	response, payload := call(t, server, http.MethodPut, "/wp-json/postulator/v1/content/"+itoa(seeded[0].ID)+"/raw",
		[]byte(`{"content":"<p>Powder</p>","expectedHash":"`+koffeinHash+`"}`), true)
	if response.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", response.StatusCode)
	}

	var written struct {
		ContentHash string `json:"contentHash"`
	}
	decode(t, payload, &written)
	if written.ContentHash != "79db24b7a931978a1db05ccdb1ae1ed16b07aafa5277af75a7dc1b3b7ec6a509" {
		t.Errorf("contentHash = %q", written.ContentHash)
	}

	response, payload = call(t, server, http.MethodPut, "/wp-json/postulator/v1/content/"+itoa(seeded[0].ID)+"/raw",
		[]byte(`{"content":"<p>again</p>","expectedHash":"`+koffeinHash+`"}`), true)
	if response.StatusCode != http.StatusConflict {
		t.Fatalf("status = %d, want 409", response.StatusCode)
	}

	var conflict struct {
		Code        string `json:"code"`
		CurrentHash string `json:"currentHash"`
	}
	decode(t, payload, &conflict)
	if conflict.Code != "hash_mismatch" || conflict.CurrentHash != written.ContentHash {
		t.Errorf("conflict = %+v", conflict)
	}
}

func TestThePostOnlyRoutesRefuseATerm(t *testing.T) {
	t.Parallel()

	server := wptest.New(t)
	term := server.Seed(wptest.Item{Type: wptest.TypeProductCategory, Title: "Koffein"})[0]

	cases := []struct {
		name   string
		method string
		path   string
		body   []byte
	}{
		{name: "seo meta", method: http.MethodPut, path: "/wp-json/postulator/v1/seo-meta/" + itoa(term.ID), body: []byte(`{"title":"x"}`)},
		{name: "raw read", method: http.MethodGet, path: "/wp-json/postulator/v1/content/" + itoa(term.ID) + "/raw"},
		{name: "raw write", method: http.MethodPut, path: "/wp-json/postulator/v1/content/" + itoa(term.ID) + "/raw", body: []byte(`{"content":"x"}`)},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			response, payload := call(t, server, tc.method, tc.path, tc.body, true)
			if response.StatusCode != http.StatusNotFound {
				t.Fatalf("status = %d, want 404", response.StatusCode)
			}

			var failure struct {
				Code string `json:"code"`
			}
			decode(t, payload, &failure)
			if failure.Code != "not_found" {
				t.Errorf("code = %q, want not_found", failure.Code)
			}
		})
	}
}

func TestATermStillAppearsInTheContentListing(t *testing.T) {
	t.Parallel()

	server := wptest.New(t)
	server.Seed(wptest.Item{Type: wptest.TypeProductCategory, Title: "Koffein", Content: "the hub"})

	_, payload := call(t, server, http.MethodGet, "/wp-json/postulator/v1/content?types=product_cat", nil, true)

	var page struct {
		Items []struct {
			Type     string `json:"type"`
			Modified string `json:"modified"`
			Path     string `json:"path"`
		} `json:"items"`
	}
	decode(t, payload, &page)

	if len(page.Items) != 1 {
		t.Fatalf("got %d items, want 1", len(page.Items))
	}
	if page.Items[0].Type != wptest.TypeProductCategory || page.Items[0].Modified == "" {
		t.Errorf("item = %+v; a term carries the modification date the plugin maintains", page.Items[0])
	}
	if page.Items[0].Path != "/koffein/" {
		t.Errorf("path = %q", page.Items[0].Path)
	}
}

func TestABrokenSiteCanReportTheWrongHash(t *testing.T) {
	t.Parallel()

	server := wptest.New(t, wptest.WithBrokenContentHash())
	seeded := server.Seed(wptest.Item{Type: wptest.TypePage, Title: "Koffein", Content: "<p>Koffein ist ein Alkaloid.</p>"})

	_, payload := call(t, server, http.MethodGet, "/wp-json/postulator/v1/content/"+itoa(seeded[0].ID)+"/raw", nil, true)

	var raw struct {
		ContentHash string `json:"contentHash"`
	}
	decode(t, payload, &raw)
	if raw.ContentHash == koffeinHash {
		t.Error("the broken site must not report the correct hash")
	}
}

func TestThePluginRoutesRejectBadInput(t *testing.T) {
	t.Parallel()

	server := wptest.New(t)
	seeded := server.Seed(wptest.Item{Type: wptest.TypePage, Title: "Koffein", Content: "<p>x</p>"})[0]

	cases := []struct {
		name   string
		method string
		path   string
		body   []byte
		status int
	}{
		{name: "cursor that is not base64", method: http.MethodGet, path: "/wp-json/postulator/v1/content?cursor=!!!", status: http.StatusBadRequest},
		{name: "cursor that is not json", method: http.MethodGet, path: "/wp-json/postulator/v1/content?cursor=bm90anNvbg", status: http.StatusBadRequest},
		{name: "seo meta with a broken body", method: http.MethodPut, path: "/wp-json/postulator/v1/seo-meta/" + itoa(seeded.ID), body: []byte("not json"), status: http.StatusBadRequest},
		{name: "seo meta for a malformed id", method: http.MethodPut, path: "/wp-json/postulator/v1/seo-meta/0", body: []byte(`{"title":"x"}`), status: http.StatusNotFound},
		{name: "raw read for a malformed id", method: http.MethodGet, path: "/wp-json/postulator/v1/content/0/raw", status: http.StatusNotFound},
		{name: "raw write for a malformed id", method: http.MethodPut, path: "/wp-json/postulator/v1/content/0/raw", body: []byte(`{"content":"x"}`), status: http.StatusNotFound},
		{name: "raw write without content", method: http.MethodPut, path: "/wp-json/postulator/v1/content/" + itoa(seeded.ID) + "/raw", body: []byte(`{}`), status: http.StatusBadRequest},
		{name: "raw write with a broken body", method: http.MethodPut, path: "/wp-json/postulator/v1/content/" + itoa(seeded.ID) + "/raw", body: []byte("not json"), status: http.StatusBadRequest},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			response, _ := call(t, server, tc.method, tc.path, tc.body, true)
			if response.StatusCode != tc.status {
				t.Errorf("status = %d, want %d", response.StatusCode, tc.status)
			}
		})
	}
}

func TestAnEditJustBeforeARawWriteIsAConflict(t *testing.T) {
	t.Parallel()

	server := wptest.New(t)
	seeded := server.Seed(wptest.Item{Type: wptest.TypePage, Title: "Coffee", Content: "<p>One.</p>"})
	path := "/wp-json/postulator/v1/content/" + itoa(seeded[0].ID) + "/raw"

	var read struct {
		ContentHash string `json:"contentHash"`
	}
	_, payload := call(t, server, http.MethodGet, path, nil, true)
	decode(t, payload, &read)

	server.EditBeforeNextRawWrite(seeded[0].ID, "<p>Edited by a human.</p>")

	response, body := call(t, server, http.MethodPut, path,
		[]byte(`{"content":"<p>Two.</p>","expectedHash":"`+read.ContentHash+`"}`), true)
	if response.StatusCode != http.StatusConflict {
		t.Fatalf("status = %d, want 409", response.StatusCode)
	}

	var conflict struct {
		Code        string `json:"code"`
		CurrentHash string `json:"currentHash"`
	}
	decode(t, body, &conflict)
	if conflict.Code != "hash_mismatch" || conflict.CurrentHash == read.ContentHash {
		t.Fatalf("conflict = %+v", conflict)
	}

	stored, _ := server.Lookup(seeded[0].ID)
	if stored.Content != "<p>Edited by a human.</p>" {
		t.Fatalf("the refused write landed: %q", stored.Content)
	}
}

func TestRewriteReplacesTheStoredContent(t *testing.T) {
	t.Parallel()

	server := wptest.New(t)
	seeded := server.Seed(wptest.Item{Type: wptest.TypePage, Title: "Coffee", Content: "<p>One.</p>"})

	if !server.Rewrite(seeded[0].ID, "<p>Two.</p>") {
		t.Fatal("Rewrite reported that the item is absent")
	}
	if server.Rewrite(9999, "<p>Three.</p>") {
		t.Fatal("Rewrite reported that an absent item exists")
	}

	stored, _ := server.Lookup(seeded[0].ID)
	if stored.Content != "<p>Two.</p>" || !stored.Modified.After(seeded[0].Modified) {
		t.Fatalf("the item holds %+v", stored)
	}
}
