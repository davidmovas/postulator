//go:build e2e

package e2e_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
)

type rawContent struct {
	ID          int    `json:"id"`
	Type        string `json:"type"`
	Content     string `json:"content"`
	ContentHash string `json:"contentHash"`
}

const richBody = `<h2>Heading</h2><ul><li>first</li><li>second</li></ul>` +
	`<p>Link to <a href="/koffein/" title="up">Koffein</a>.</p>` +
	`<img src="/wp-content/uploads/powder.png" alt="Powder" width="800" height="600" />`

func rawPath(id int) string {
	return fmt.Sprintf("/wp-json/postulator/v1/content/%d/raw", id)
}

func TestRawRoundTripsWithoutFiltering(t *testing.T) {
	c, _ := newClient(t)

	original := "<p>original</p>"
	id := createPage(t, c, pageSpec{title: "Raw", slug: uniqueSlug("raw"), content: original})

	var got rawContent
	c.expect(t, http.MethodGet, rawPath(id), nil, http.StatusOK, &got)

	if got.ID != id {
		t.Errorf("id = %d, want %d", got.ID, id)
	}
	if got.Type != "page" {
		t.Errorf("type = %q, want page", got.Type)
	}
	if got.Content != original {
		t.Errorf("content = %q, want %q", got.Content, original)
	}
	if got.ContentHash != hashOf(original) {
		t.Errorf("contentHash = %q, want %q", got.ContentHash, hashOf(original))
	}

	var written struct {
		ContentHash string `json:"contentHash"`
	}
	c.expect(t, http.MethodPut, rawPath(id),
		map[string]string{"content": richBody, "expectedHash": got.ContentHash},
		http.StatusOK, &written)

	if written.ContentHash != hashOf(richBody) {
		t.Fatalf("contentHash = %q, want %q: WordPress altered the stored content", written.ContentHash, hashOf(richBody))
	}

	var reread rawContent
	c.expect(t, http.MethodGet, rawPath(id), nil, http.StatusOK, &reread)
	if reread.Content != richBody {
		t.Fatalf("re-read content = %q, want %q", reread.Content, richBody)
	}
}

func TestRawPreservesBackslashesAndQuotes(t *testing.T) {
	c, _ := newClient(t)

	id := createPage(t, c, pageSpec{title: "Slash", slug: uniqueSlug("slash"), content: "<p>seed</p>"})

	tricky := `<p>A path C:\Users\admin and a "quoted" word and an escaped \" quote.</p>`
	var written struct {
		ContentHash string `json:"contentHash"`
	}
	c.expect(t, http.MethodPut, rawPath(id), map[string]string{"content": tricky}, http.StatusOK, &written)

	if written.ContentHash != hashOf(tricky) {
		t.Fatalf("contentHash = %q, want %q: slashes were eaten on save", written.ContentHash, hashOf(tricky))
	}

	var reread rawContent
	c.expect(t, http.MethodGet, rawPath(id), nil, http.StatusOK, &reread)
	if reread.Content != tricky {
		t.Fatalf("re-read content = %q, want %q", reread.Content, tricky)
	}
}

func TestRawRejectsAStaleHash(t *testing.T) {
	c, _ := newClient(t)

	original := "<p>stale</p>"
	id := createPage(t, c, pageSpec{title: "Stale", slug: uniqueSlug("stale"), content: original})

	status, body := c.request(t, http.MethodPut, rawPath(id),
		map[string]string{"content": "<p>new</p>", "expectedHash": hashOf("something else")})

	if status != http.StatusConflict {
		t.Fatalf("status %d, want 409, body %s", status, body)
	}

	var conflict struct {
		Code        string `json:"code"`
		Message     string `json:"message"`
		CurrentHash string `json:"currentHash"`
	}
	if err := json.Unmarshal(body, &conflict); err != nil {
		t.Fatalf("decode conflict: %v, body %s", err, body)
	}
	if conflict.Code != "hash_mismatch" {
		t.Errorf("code = %q, want hash_mismatch", conflict.Code)
	}
	if conflict.Message == "" {
		t.Errorf("message is empty")
	}
	if conflict.CurrentHash != hashOf(original) {
		t.Errorf("currentHash = %q, want %q", conflict.CurrentHash, hashOf(original))
	}

	var unchanged rawContent
	c.expect(t, http.MethodGet, rawPath(id), nil, http.StatusOK, &unchanged)
	if unchanged.Content != original {
		t.Fatalf("a rejected write changed the content to %q", unchanged.Content)
	}
}

func TestRawWritesWithoutAnExpectedHash(t *testing.T) {
	c, _ := newClient(t)

	id := createPage(t, c, pageSpec{title: "Blind", slug: uniqueSlug("blind"), content: "<p>before</p>"})

	var written struct {
		ContentHash string `json:"contentHash"`
	}
	c.expect(t, http.MethodPut, rawPath(id), map[string]string{"content": "<p>after</p>"}, http.StatusOK, &written)

	if written.ContentHash != hashOf("<p>after</p>") {
		t.Errorf("contentHash = %q, want %q", written.ContentHash, hashOf("<p>after</p>"))
	}
}

func TestRawRejectsBadRequests(t *testing.T) {
	c, _ := newClient(t)

	id := createPage(t, c, pageSpec{title: "Bad", slug: uniqueSlug("bad"), content: "<p>b</p>"})

	cases := []struct {
		name   string
		path   string
		body   map[string]string
		status int
	}{
		{name: "unknown id", path: rawPath(98765432), body: map[string]string{"content": "x"}, status: http.StatusNotFound},
		{name: "missing content", path: rawPath(id), body: map[string]string{"expectedHash": "x"}, status: http.StatusBadRequest},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			status, body := c.request(t, http.MethodPut, testCase.path, testCase.body)
			if status != testCase.status {
				t.Fatalf("status %d, want %d, body %s", status, testCase.status, body)
			}
		})
	}
}

func TestRawRequiresAuthentication(t *testing.T) {
	c, _ := newClient(t)

	id := createPage(t, c, pageSpec{title: "Guarded", slug: uniqueSlug("guarded"), content: "<p>g</p>"})

	anonymous := &client{base: c.base, http: c.http}
	status, body := anonymous.request(t, http.MethodGet, rawPath(id), nil)
	if status != http.StatusUnauthorized {
		t.Fatalf("status %d, want 401, body %s", status, body)
	}
}
