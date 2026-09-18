package wptest_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/davidmovas/postulator/internal/adapters/wp/wptest"
)

func seedPages(t *testing.T, server *wptest.Server, count int) []wptest.Item {
	t.Helper()

	items := make([]wptest.Item, 0, count)
	for index := range count {
		items = append(items, wptest.Item{Type: wptest.TypePage, Title: "Page " + string(rune('A'+index)), Content: "<p>body</p>"})
	}
	return server.Seed(items...)
}

func TestListReportsTheWordPressPagingHeaders(t *testing.T) {
	t.Parallel()

	server := wptest.New(t)
	seedPages(t, server, 5)

	response, payload := call(t, server, http.MethodGet, "/wp-json/wp/v2/pages?page=2&per_page=2", nil, true)
	if response.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", response.StatusCode)
	}
	if got := response.Header.Get("X-WP-Total"); got != "5" {
		t.Errorf("X-WP-Total = %q, want 5", got)
	}
	if got := response.Header.Get("X-WP-TotalPages"); got != "3" {
		t.Errorf("X-WP-TotalPages = %q, want 3", got)
	}

	var items []map[string]any
	decode(t, payload, &items)
	if len(items) != 2 {
		t.Errorf("page 2 holds %d items, want 2", len(items))
	}
}

func TestAPageNumberPastTheEndIsAFourHundred(t *testing.T) {
	t.Parallel()

	server := wptest.New(t)
	seedPages(t, server, 2)

	response, payload := call(t, server, http.MethodGet, "/wp-json/wp/v2/pages?page=9&per_page=2", nil, true)
	if response.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", response.StatusCode)
	}

	var failure struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	}
	decode(t, payload, &failure)
	if failure.Code != "rest_post_invalid_page_number" {
		t.Errorf("code = %q", failure.Code)
	}
}

func TestListFiltersByStatusAndModification(t *testing.T) {
	t.Parallel()

	server := wptest.New(t)
	seeded := server.Seed(
		wptest.Item{Type: wptest.TypePage, Title: "Draft", Status: "draft"},
		wptest.Item{Type: wptest.TypePage, Title: "Live", Status: "publish"},
	)

	_, payload := call(t, server, http.MethodGet, "/wp-json/wp/v2/pages?status=draft", nil, true)
	var drafts []map[string]any
	decode(t, payload, &drafts)
	if len(drafts) != 1 {
		t.Fatalf("status filter returned %d items, want 1", len(drafts))
	}

	cut := seeded[0].Modified.UTC().Format("2006-01-02T15:04:05")
	_, payload = call(t, server, http.MethodGet, "/wp-json/wp/v2/pages?modified_after="+cut, nil, true)
	var recent []map[string]any
	decode(t, payload, &recent)
	if len(recent) != 1 {
		t.Errorf("modified_after returned %d items, want 1; the bound is exclusive", len(recent))
	}
}

func TestFieldsNarrowsThePayload(t *testing.T) {
	t.Parallel()

	server := wptest.New(t)
	seedPages(t, server, 1)

	_, payload := call(t, server, http.MethodGet, "/wp-json/wp/v2/pages?_fields=id,slug", nil, true)
	var items []map[string]any
	decode(t, payload, &items)
	if len(items) != 1 {
		t.Fatalf("got %d items", len(items))
	}
	if len(items[0]) != 2 {
		t.Errorf("item = %v, want only id and slug", items[0])
	}
}

func TestAnItemCarriesRawAndRenderedTextAndAnEmptyMetaArray(t *testing.T) {
	t.Parallel()

	server := wptest.New(t)
	seeded := server.Seed(wptest.Item{Type: wptest.TypePage, Title: "Koffein", Content: "<p>Koffein ist ein Alkaloid.</p>"})

	_, payload := call(t, server, http.MethodGet, "/wp-json/wp/v2/pages/"+itoa(seeded[0].ID), nil, true)

	var item struct {
		Title struct {
			Raw      string `json:"raw"`
			Rendered string `json:"rendered"`
		} `json:"title"`
		ModifiedGMT string          `json:"modified_gmt"`
		Meta        json.RawMessage `json:"meta"`
	}
	decode(t, payload, &item)

	if item.Title.Raw != "Koffein" || item.Title.Rendered != "Koffein" {
		t.Errorf("title = %+v", item.Title)
	}
	if len(item.ModifiedGMT) != 19 {
		t.Errorf("modified_gmt = %q, want a WordPress timestamp with no offset", item.ModifiedGMT)
	}
	if string(item.Meta) != "[]" {
		t.Errorf("meta = %s, want the empty array WordPress actually sends", item.Meta)
	}
}

func TestAnUnknownItemIsAFourOhFour(t *testing.T) {
	t.Parallel()

	server := wptest.New(t)
	response, _ := call(t, server, http.MethodGet, "/wp-json/wp/v2/pages/404", nil, true)
	if response.StatusCode != http.StatusNotFound {
		t.Errorf("status = %d, want 404", response.StatusCode)
	}
}

func TestCreateRewritesACollidingSlug(t *testing.T) {
	t.Parallel()

	server := wptest.New(t)
	server.Seed(wptest.Item{Type: wptest.TypePage, Title: "Powder"})

	response, payload := call(t, server, http.MethodPost, "/wp-json/wp/v2/pages", []byte(`{"title":"Powder","content":"<p>x</p>","slug":"powder","status":"draft"}`), true)
	if response.StatusCode != http.StatusCreated {
		t.Fatalf("status = %d, want 201", response.StatusCode)
	}

	var created struct {
		ID     int64  `json:"id"`
		Slug   string `json:"slug"`
		Link   string `json:"link"`
		Status string `json:"status"`
	}
	decode(t, payload, &created)
	if created.Slug != "powder-2" {
		t.Errorf("slug = %q, want powder-2", created.Slug)
	}
	if created.Status != "draft" {
		t.Errorf("status = %q, want draft", created.Status)
	}
	if created.Link == "" {
		t.Error("a created item must carry its permalink")
	}
}

func TestUpdateAppliesOnlyThePresentFields(t *testing.T) {
	t.Parallel()

	server := wptest.New(t)
	parent := server.Seed(wptest.Item{Type: wptest.TypePage, Title: "Koffein"})[0]
	child := server.Seed(wptest.Item{Type: wptest.TypePage, Title: "Powder", Parent: parent.ID, Status: "draft"})[0]

	_, payload := call(t, server, http.MethodPost, "/wp-json/wp/v2/pages/"+itoa(child.ID), []byte(`{"status":"publish"}`), true)
	var updated struct {
		Status string `json:"status"`
		Parent int64  `json:"parent"`
		Title  struct {
			Raw string `json:"raw"`
		} `json:"title"`
	}
	decode(t, payload, &updated)
	if updated.Status != "publish" || updated.Parent != parent.ID || updated.Title.Raw != "Powder" {
		t.Errorf("update = %+v, want only the status changed", updated)
	}

	_, payload = call(t, server, http.MethodPost, "/wp-json/wp/v2/pages/"+itoa(child.ID), []byte(`{"parent":0}`), true)
	decode(t, payload, &updated)
	if updated.Parent != 0 {
		t.Errorf("parent = %d, want 0; zero is a value, not an absence", updated.Parent)
	}
}

func TestDeleteTrashesUnlessItIsForced(t *testing.T) {
	t.Parallel()

	server := wptest.New(t)
	seeded := seedPages(t, server, 2)

	_, payload := call(t, server, http.MethodDelete, "/wp-json/wp/v2/pages/"+itoa(seeded[0].ID), nil, true)
	var trashed struct {
		Status string `json:"status"`
	}
	decode(t, payload, &trashed)
	if trashed.Status != "trash" {
		t.Errorf("status = %q, want trash", trashed.Status)
	}
	if _, ok := server.Lookup(seeded[0].ID); !ok {
		t.Error("a trashed item stays in the database")
	}

	call(t, server, http.MethodDelete, "/wp-json/wp/v2/pages/"+itoa(seeded[1].ID)+"?force=true", nil, true)
	if _, ok := server.Lookup(seeded[1].ID); ok {
		t.Error("a forced delete removes the item")
	}
}
