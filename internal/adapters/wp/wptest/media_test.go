package wptest_test

import (
	"net/http"
	"testing"

	"github.com/davidmovas/postulator/internal/adapters/wp/wptest"
)

func TestARawUploadStoresTheBytesAndIgnoresAlternativeText(t *testing.T) {
	t.Parallel()

	server := wptest.New(t)
	request := newUpload(t, server, "koffein.png", "image/png", []byte{0x89, 0x50, 0x4e, 0x47})

	response, payload := send(t, request)
	if response.StatusCode != http.StatusCreated {
		t.Fatalf("status = %d, want 201", response.StatusCode)
	}

	var uploaded struct {
		ID        int64  `json:"id"`
		SourceURL string `json:"source_url"`
		AltText   string `json:"alt_text"`
		MimeType  string `json:"mime_type"`
	}
	decode(t, payload, &uploaded)

	if uploaded.ID == 0 || uploaded.SourceURL == "" || uploaded.MimeType != "image/png" {
		t.Errorf("uploaded = %+v", uploaded)
	}
	if uploaded.AltText != "" {
		t.Error("the raw upload cannot carry alternative text")
	}

	uploads := server.Uploads()
	if len(uploads) != 1 || uploads[0].Filename != "koffein.png" || len(uploads[0].Bytes) != 4 {
		t.Errorf("uploads = %+v", uploads)
	}

	_, payload = call(t, server, http.MethodPost, "/wp-json/wp/v2/media/"+itoa(uploaded.ID), []byte(`{"alt_text":"Koffein","title":"Koffein"}`), true)
	decode(t, payload, &uploaded)
	if uploaded.AltText != "Koffein" {
		t.Errorf("alt_text = %q, want Koffein after the second call", uploaded.AltText)
	}
}

func TestCategoriesListAndCreateWithUniqueSlugs(t *testing.T) {
	t.Parallel()

	server := wptest.New(t)
	server.SeedCategory(wptest.Category{Name: "Koffein", Description: "the hub"})

	response, payload := call(t, server, http.MethodPost, "/wp-json/wp/v2/categories", []byte(`{"name":"Koffein"}`), true)
	if response.StatusCode != http.StatusCreated {
		t.Fatalf("status = %d, want 201", response.StatusCode)
	}

	var created struct {
		ID   int64  `json:"id"`
		Slug string `json:"slug"`
	}
	decode(t, payload, &created)
	if created.Slug != "koffein-2" {
		t.Errorf("slug = %q, want koffein-2", created.Slug)
	}

	response, payload = call(t, server, http.MethodGet, "/wp-json/wp/v2/categories", nil, true)
	if response.Header.Get("X-WP-Total") != "2" {
		t.Errorf("X-WP-Total = %q, want 2", response.Header.Get("X-WP-Total"))
	}

	var categories []map[string]any
	decode(t, payload, &categories)
	if len(categories) != 2 {
		t.Errorf("got %d categories, want 2", len(categories))
	}
	if len(server.Categories()) != 2 {
		t.Errorf("the store holds %d categories", len(server.Categories()))
	}
}
