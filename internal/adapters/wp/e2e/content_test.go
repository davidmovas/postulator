//go:build e2e

package e2e_test

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestContentReportsPathHashLinksAndH1(t *testing.T) {
	c, env := newClient(t)

	parentSlug := uniqueSlug("koffein")
	parentID := createPage(t, c, pageSpec{title: "Koffein", slug: parentSlug, content: "<p>Parent</p>"})

	childSlug := uniqueSlug("powder")
	body := fmt.Sprintf(
		`<h1>Powder</h1><h2>Section</h2><ul><li>one</li></ul>`+
			`<p>Up to <a href="/%s/">Koffein</a> and out to <a href="https://example.com/x/">Example</a>.</p>`+
			`<img src="/wp-content/uploads/x.png" alt="x" />`,
		parentSlug,
	)
	childID := createPage(t, c, pageSpec{title: "Powder", slug: childSlug, content: body, parent: parentID})

	item := findItem(t, c, "page", childID)

	if item.Type != "page" {
		t.Errorf("type = %q, want page", item.Type)
	}
	if item.Slug != childSlug {
		t.Errorf("slug = %q, want %q", item.Slug, childSlug)
	}
	wantPath := "/" + parentSlug + "/" + childSlug + "/"
	if item.Path != wantPath {
		t.Errorf("path = %q, want %q", item.Path, wantPath)
	}
	if item.Parent != parentID {
		t.Errorf("parent = %d, want %d", item.Parent, parentID)
	}
	if item.Status != "publish" {
		t.Errorf("status = %q, want publish", item.Status)
	}
	if item.Title != "Powder" {
		t.Errorf("title = %q, want Powder", item.Title)
	}
	if item.H1 != "Powder" {
		t.Errorf("h1 = %q, want Powder", item.H1)
	}
	if item.ContentHash != hashOf(body) {
		t.Errorf("contentHash = %q, want %q", item.ContentHash, hashOf(body))
	}
	if _, err := time.Parse(time.RFC3339, item.Modified); err != nil {
		t.Errorf("modified = %q is not RFC3339: %v", item.Modified, err)
	}

	want := link{Href: "/" + parentSlug + "/", Anchor: "Koffein"}
	if len(item.Links) != 1 || item.Links[0] != want {
		t.Fatalf("links = %v, want exactly [%v]", item.Links, want)
	}

	if env.seo == "none" && item.Meta != (itemMeta{}) {
		t.Errorf("meta = %v, want empty for a page with no SEO meta", item.Meta)
	}
}

func TestContentLowercasesPathsAndHrefs(t *testing.T) {
	c, _ := newClient(t)

	mixedSlug := strings.ToUpper(uniqueSlug("MixedCase"))
	body := fmt.Sprintf(`<p><a href="/%s/?utm=x#frag">Anchor</a></p>`, mixedSlug)
	id := createPage(t, c, pageSpec{title: "Mixed", slug: mixedSlug, content: body})

	item := findItem(t, c, "page", id)

	wantPath := strings.ToLower("/" + mixedSlug + "/")
	if item.Path != wantPath {
		t.Errorf("path = %q, want %q", item.Path, wantPath)
	}
	if len(item.Links) != 1 {
		t.Fatalf("links = %v, want exactly one", item.Links)
	}
	if item.Links[0].Href != wantPath {
		t.Errorf("links[0].href = %q, want %q", item.Links[0].Href, wantPath)
	}
	if item.Links[0].Anchor != "Anchor" {
		t.Errorf("links[0].anchor = %q, want Anchor", item.Links[0].Anchor)
	}
}

func TestContentIncludesDraftsAndExcludesTrash(t *testing.T) {
	c, _ := newClient(t)

	draftID := createPage(t, c, pageSpec{title: "Draft", slug: uniqueSlug("draft"), content: "<p>d</p>", status: "draft"})

	item := findItem(t, c, "page", draftID)
	if item.Status != "draft" {
		t.Errorf("status = %q, want draft", item.Status)
	}

	c.expect(t, http.MethodDelete, fmt.Sprintf("/wp-json/wp/v2/pages/%d", draftID), nil, http.StatusOK, nil)

	trashed := false
	walkContent(t, c, "page", 100, func(listed contentItem) bool {
		if listed.ID == draftID {
			trashed = true
			return true
		}
		return false
	})
	if trashed {
		t.Fatalf("trashed page %d is still listed", draftID)
	}
}

func TestContentPagesByKeysetCursor(t *testing.T) {
	c, _ := newClient(t)

	for i := range 3 {
		createPage(t, c, pageSpec{title: fmt.Sprintf("Cursor %d", i), slug: uniqueSlug("cursor"), content: "<p>c</p>"})
	}

	seen := make(map[int]int)
	cursor := ""
	pages := 0
	for range 200 {
		query := "types=page&limit=1"
		if cursor != "" {
			query += "&cursor=" + url.QueryEscape(cursor)
		}
		page := listContent(t, c, query)
		pages++
		if len(page.Items) > 1 {
			t.Fatalf("limit=1 returned %d items", len(page.Items))
		}
		for _, item := range page.Items {
			seen[item.ID]++
		}
		if page.NextCursor == nil {
			break
		}
		cursor = *page.NextCursor
	}
	if pages < 3 {
		t.Fatalf("walked only %d pages, want at least 3", pages)
	}
	for id, count := range seen {
		if count != 1 {
			t.Errorf("id %d appeared %d times", id, count)
		}
	}
}

func TestContentFiltersBySince(t *testing.T) {
	c, _ := newClient(t)

	before := time.Now().UTC().Format(time.RFC3339)
	time.Sleep(2 * time.Second)
	freshID := createPage(t, c, pageSpec{title: "Fresh", slug: uniqueSlug("fresh"), content: "<p>f</p>"})

	page := listContent(t, c, "types=page&limit=500&since="+url.QueryEscape(before))
	found := false
	for _, item := range page.Items {
		if item.ID == freshID {
			found = true
		}
		if item.Modified < before {
			t.Errorf("item %d modified %q is older than since %q", item.ID, item.Modified, before)
		}
	}
	if !found {
		t.Fatalf("page %d created after %q is not listed", freshID, before)
	}
}

func TestContentRejectsBadParameters(t *testing.T) {
	c, _ := newClient(t)

	cases := []struct {
		name  string
		query string
	}{
		{name: "unknown type", query: "types=widget"},
		{name: "unparseable since", query: "since=yesterday"},
		{name: "undecodable cursor", query: "cursor=%21%21%21"},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			status, body := c.request(t, http.MethodGet, "/wp-json/postulator/v1/content?"+testCase.query, nil)
			if status != http.StatusBadRequest {
				t.Fatalf("status %d, want 400, body %s", status, body)
			}
			if !strings.Contains(string(body), `"code"`) {
				t.Fatalf("body %s has no code field", body)
			}
		})
	}
}

func TestContentClampsLimit(t *testing.T) {
	c, _ := newClient(t)

	page := listContent(t, c, "types=page&limit=100000")
	if len(page.Items) > 500 {
		t.Fatalf("limit=100000 returned %d items, want at most 500", len(page.Items))
	}
}

func TestContentReportsNullCursorAtEndOfList(t *testing.T) {
	c, _ := newClient(t)

	page := listContent(t, c, "types=page&limit=500")
	if page.NextCursor != nil {
		t.Fatalf("nextCursor = %q, want null at the end of the list", *page.NextCursor)
	}
}

func TestContentSinceIsExclusive(t *testing.T) {
	c, _ := newClient(t)

	id := createPage(t, c, pageSpec{title: "Exclusive", slug: uniqueSlug("exclusive"), content: "<p>e</p>"})
	item := findItem(t, c, "page", id)

	atItsOwnModified := listContent(t, c, "types=page&limit=500&since="+url.QueryEscape(item.Modified))
	for _, listed := range atItsOwnModified.Items {
		if listed.ID == id {
			t.Fatalf("since=%s returned the item whose modified equals it; since must be exclusive", item.Modified)
		}
	}

	before, err := time.Parse(time.RFC3339, item.Modified)
	if err != nil {
		t.Fatalf("parse modified %q: %v", item.Modified, err)
	}
	oneSecondEarlier := before.Add(-time.Second).Format(time.RFC3339)

	found := false
	walkContent(t, c, "page", 500, func(listed contentItem) bool {
		if listed.ID == id {
			found = true
			return true
		}
		return false
	})
	if !found {
		t.Fatalf("page %d is not listed at all", id)
	}

	page := listContent(t, c, "types=page&limit=500&since="+url.QueryEscape(oneSecondEarlier))
	found = false
	for _, listed := range page.Items {
		if listed.ID == id {
			found = true
		}
	}
	if !found {
		t.Fatalf("since=%s (one second before modified) did not return the item", oneSecondEarlier)
	}
}

func TestContentGivesDraftsAUniquePath(t *testing.T) {
	c, _ := newClient(t)

	slug := uniqueSlug("dev5-target")
	title := strings.ReplaceAll(slug, "-", " ")

	publishedID := createPage(t, c, pageSpec{title: title, slug: slug, content: "<p>live</p>"})
	draftID := createPage(t, c, pageSpec{title: title, content: "<p>draft</p>", status: "draft"})

	published := findItem(t, c, "page", publishedID)
	draft := findItem(t, c, "page", draftID)

	if draft.Status != "draft" {
		t.Fatalf("draft status = %q, want draft", draft.Status)
	}
	if draft.Slug != "" {
		t.Fatalf("draft slug = %q, want empty so the path has to be synthesised", draft.Slug)
	}
	if published.Path != "/"+slug+"/" {
		t.Fatalf("published path = %q, want /%s/", published.Path, slug)
	}
	if draft.Path == published.Path {
		t.Fatalf("draft and published page both report %q; the synthesised draft slug must be unique", draft.Path)
	}
	if !strings.HasPrefix(draft.Path, "/") || !strings.HasSuffix(draft.Path, "/") {
		t.Errorf("draft path = %q, want one leading and one trailing slash", draft.Path)
	}
}

func TestContentRejectsMalformedParameters(t *testing.T) {
	c, _ := newClient(t)

	cases := []struct {
		name  string
		query string
		code  string
	}{
		{name: "array types", query: "types[]=page", code: "invalid_param"},
		{name: "array since", query: "since[]=2026-01-01T00:00:00Z", code: "invalid_param"},
		{name: "array cursor", query: "cursor[]=x", code: "invalid_param"},
		{name: "cursor without a phase", query: "cursor=" + base64URL(`{"i":1}`), code: "invalid_cursor"},
		{name: "post cursor without modified", query: "cursor=" + base64URL(`{"p":"post","i":1}`), code: "invalid_cursor"},
		{name: "post cursor with a string id", query: "cursor=" + base64URL(`{"p":"post","m":"2026-01-01 00:00:00","i":"1"}`), code: "invalid_cursor"},
		{name: "term cursor without an id", query: "cursor=" + base64URL(`{"p":"term"}`), code: "invalid_cursor"},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			status, body := c.request(t, http.MethodGet, "/wp-json/postulator/v1/content?"+testCase.query, nil)
			if status != http.StatusBadRequest {
				t.Fatalf("status %d, want 400, body %s", status, body)
			}
			if !strings.Contains(string(body), `"`+testCase.code+`"`) {
				t.Fatalf("body %s does not carry the %s code", body, testCase.code)
			}
		})
	}
}
