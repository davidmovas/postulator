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

func TestMediaCanBeReadBackAndGuardsItsInput(t *testing.T) {
	t.Parallel()

	server := wptest.New(t)
	_, payload := send(t, newUpload(t, server, "koffein.png", "image/png", []byte{0x89}))

	var uploaded struct {
		ID int64 `json:"id"`
	}
	decode(t, payload, &uploaded)

	response, body := call(t, server, http.MethodGet, "/wp-json/wp/v2/media/"+itoa(uploaded.ID), nil, true)
	if response.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", response.StatusCode)
	}

	var read struct {
		MimeType string `json:"mime_type"`
	}
	decode(t, body, &read)
	if read.MimeType != "image/png" {
		t.Errorf("mime_type = %q, want image/png", read.MimeType)
	}

	cases := []struct {
		name   string
		method string
		path   string
		body   []byte
	}{
		{name: "read a missing upload", method: http.MethodGet, path: "/wp-json/wp/v2/media/404"},
		{name: "describe a missing upload", method: http.MethodPost, path: "/wp-json/wp/v2/media/404", body: []byte(`{"alt_text":"x"}`)},
		{name: "read a malformed id", method: http.MethodGet, path: "/wp-json/wp/v2/media/none"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			missing, _ := call(t, server, tc.method, tc.path, tc.body, true)
			if missing.StatusCode != http.StatusNotFound {
				t.Errorf("status = %d, want 404", missing.StatusCode)
			}
		})
	}
}

func TestAnUploadNeedsADispositionAndBytes(t *testing.T) {
	t.Parallel()

	server := wptest.New(t)

	response, _ := call(t, server, http.MethodPost, "/wp-json/wp/v2/media", []byte{0x89}, true)
	if response.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want 400 without a Content-Disposition", response.StatusCode)
	}

	response, _ = send(t, newUpload(t, server, "koffein.png", "image/png", nil))
	if response.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want 400 without any bytes", response.StatusCode)
	}
}

func TestTheMediaLibraryIsSearchable(t *testing.T) {
	t.Parallel()

	server := wptest.New(t)
	for _, name := range []string{"espresso-cup.png", "kettle.png"} {
		response, _ := send(t, newUpload(t, server, name, "image/png", []byte{0x89}))
		if response.StatusCode != http.StatusCreated {
			t.Fatalf("upload %s: status %d", name, response.StatusCode)
		}
	}

	var listed []struct {
		SourceURL string `json:"source_url"`
		ID        int64  `json:"id"`
	}
	response, payload := call(t, server, http.MethodGet, "/wp-json/wp/v2/media?search=espresso", nil, true)
	if response.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", response.StatusCode)
	}
	decode(t, payload, &listed)
	if len(listed) != 1 {
		t.Fatalf("the search returned %d items, want 1", len(listed))
	}
	if response.Header.Get("X-WP-Total") != "1" {
		t.Errorf("X-WP-Total = %q, want 1", response.Header.Get("X-WP-Total"))
	}
}

func TestTheSlugFilterNarrowsAListing(t *testing.T) {
	t.Parallel()

	server := wptest.New(t)
	server.Seed(
		wptest.Item{Type: wptest.TypePage, Title: "Espresso"},
		wptest.Item{Type: wptest.TypePage, Title: "Kettle"},
	)

	var listed []struct {
		Slug string `json:"slug"`
	}
	_, payload := call(t, server, http.MethodGet, "/wp-json/wp/v2/pages?slug=kettle", nil, true)
	decode(t, payload, &listed)
	if len(listed) != 1 || listed[0].Slug != "kettle" {
		t.Fatalf("the slug filter returned %+v", listed)
	}
}
