//go:build e2e

package e2e_test

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"
)

type previewLink struct {
	URL       string `json:"url"`
	ExpiresAt string `json:"expiresAt"`
}

var previewToken = regexp.MustCompile(`^[0-9a-f]{32}$`)

func previewPath(id int) string {
	return fmt.Sprintf("/wp-json/postulator/v1/content/%d/preview", id)
}

func issuePreview(t *testing.T, c *client, id int) previewLink {
	t.Helper()

	var link previewLink
	c.expect(t, http.MethodPost, previewPath(id), nil, http.StatusOK, &link)
	return link
}

func tokenOf(t *testing.T, link string) string {
	t.Helper()

	parsed, err := url.Parse(link)
	if err != nil {
		t.Fatalf("parse %s: %v", link, err)
	}
	return parsed.Query().Get("postulator_preview")
}

func withToken(t *testing.T, link, token string) string {
	t.Helper()

	parsed, err := url.Parse(link)
	if err != nil {
		t.Fatalf("parse %s: %v", link, err)
	}
	query := parsed.Query()
	query.Set("postulator_preview", token)
	parsed.RawQuery = query.Encode()
	return parsed.String()
}

func (c *client) anonymous(t *testing.T, target string) (status int, header http.Header, body string) {
	t.Helper()

	response, err := c.http.Get(target)
	if err != nil {
		t.Fatalf("call GET %s: %v", target, err)
	}
	defer response.Body.Close()

	raw, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read GET %s: %v", target, err)
	}
	return response.StatusCode, response.Header, string(raw)
}

func expirePreview(t *testing.T, env environment, id int) {
	t.Helper()

	if !strings.HasPrefix(env.baseURL, testStack) {
		t.Skipf("forcing an expiry needs wp-cli on the docker stack, and %s is another site", env.baseURL)
	}
	compose := filepath.Join(repoRoot(t), "docker", "e2e", "compose.yaml")
	command := exec.CommandContext(t.Context(), "docker", "compose", "-p", testProject, "-f", compose,
		"run", "--rm", "--no-deps", "-T",
		"--entrypoint", "wp", "bootstrap", "post", "meta", "update", fmt.Sprint(id), "_postulator_preview_expires", "1")
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("expire the preview through wp-cli: %v: %s", err, output)
	}
}

func TestPreviewLinkRendersADraftWithoutALogin(t *testing.T) {
	c, env := newClient(t)

	title := "Preview " + uniqueSlug("draft")
	id := createPage(t, c, pageSpec{title: title, slug: uniqueSlug("preview"), content: "<p>unpublished words</p>", status: "draft"})

	link := issuePreview(t, c, id)
	if !strings.HasPrefix(link.URL, env.baseURL) {
		t.Errorf("url = %q, want it under %s", link.URL, env.baseURL)
	}
	if !previewToken.MatchString(tokenOf(t, link.URL)) {
		t.Errorf("token in %q is not 32 lowercase hex characters", link.URL)
	}
	if !strings.Contains(link.URL, "preview=true") {
		t.Errorf("url = %q, want preview=true", link.URL)
	}
	expires, err := time.Parse(time.RFC3339, link.ExpiresAt)
	if err != nil {
		t.Fatalf("expiresAt %q is not RFC3339: %v", link.ExpiresAt, err)
	}
	if until := time.Until(expires); until < 50*time.Minute || until > 70*time.Minute {
		t.Errorf("expiresAt is %s away, want about an hour", until)
	}

	status, header, body := c.anonymous(t, link.URL)
	if status != http.StatusOK {
		t.Fatalf("anonymous GET of the preview: status %d, want 200", status)
	}
	if !strings.Contains(body, title) || !strings.Contains(body, "unpublished words") {
		t.Errorf("the preview does not render the draft")
	}
	if !strings.Contains(body, "noindex") {
		t.Errorf("the preview carries no noindex robots directive")
	}
	if !strings.Contains(header.Get("Cache-Control"), "no-cache") {
		t.Errorf("Cache-Control = %q, want no-cache", header.Get("Cache-Control"))
	}
	if !strings.Contains(header.Get("X-Robots-Tag"), "noindex") {
		t.Errorf("X-Robots-Tag = %q, want noindex", header.Get("X-Robots-Tag"))
	}
}

func TestPreviewLinkRefusesAWrongToken(t *testing.T) {
	c, _ := newClient(t)

	title := "Wrong " + uniqueSlug("draft")
	id := createPage(t, c, pageSpec{title: title, slug: uniqueSlug("wrong"), content: "<p>secret</p>", status: "draft"})
	link := issuePreview(t, c, id)

	status, _, body := c.anonymous(t, withToken(t, link.URL, strings.Repeat("0", 32)))
	if status != http.StatusNotFound {
		t.Fatalf("a wrong token: status %d, want 404", status)
	}
	if strings.Contains(body, title) {
		t.Fatalf("a wrong token rendered the draft")
	}
}

func TestPreviewLinkRefusesADraftThatWasNeverIssued(t *testing.T) {
	c, env := newClient(t)

	id := createPage(t, c, pageSpec{title: "Never", slug: uniqueSlug("never"), content: "<p>x</p>", status: "draft"})

	target := fmt.Sprintf("%s/?page_id=%d&preview=true&postulator_preview=%s", env.baseURL, id, strings.Repeat("a", 32))
	if status, _, _ := c.anonymous(t, target); status != http.StatusNotFound {
		t.Fatalf("a token for a draft that was never issued: status %d, want 404", status)
	}
}

func TestPreviewLinkRotatesOnEachCall(t *testing.T) {
	c, _ := newClient(t)

	id := createPage(t, c, pageSpec{title: "Rotate", slug: uniqueSlug("rotate"), content: "<p>x</p>", status: "draft"})

	first := issuePreview(t, c, id)
	second := issuePreview(t, c, id)
	if tokenOf(t, first.URL) == tokenOf(t, second.URL) {
		t.Fatalf("two calls issued the same token")
	}
	if status, _, _ := c.anonymous(t, first.URL); status != http.StatusNotFound {
		t.Errorf("the first link after a second was issued: status %d, want 404", status)
	}
	if status, _, _ := c.anonymous(t, second.URL); status != http.StatusOK {
		t.Errorf("the second link: status %d, want 200", status)
	}
}

func TestPreviewLinkExpires(t *testing.T) {
	c, env := newClient(t)

	id := createPage(t, c, pageSpec{title: "Expire", slug: uniqueSlug("expire"), content: "<p>x</p>", status: "draft"})
	link := issuePreview(t, c, id)

	expirePreview(t, env, id)

	if status, _, _ := c.anonymous(t, link.URL); status != http.StatusNotFound {
		t.Fatalf("an expired link: status %d, want 404", status)
	}
}

func TestPreviewLinkLeavesModifiedAndContentAlone(t *testing.T) {
	c, _ := newClient(t)

	id := createPage(t, c, pageSpec{title: "Quiet", slug: uniqueSlug("quiet"), content: "<p>x</p>", status: "draft"})
	before := findItem(t, c, "page", id)

	issuePreview(t, c, id)

	after := findItem(t, c, "page", id)
	if after.Modified != before.Modified || after.ContentHash != before.ContentHash {
		t.Fatalf("issuing a link changed the item: before %+v, after %+v", before, after)
	}
}

func TestPreviewLinkDoesNotChangeAPublishedPage(t *testing.T) {
	c, _ := newClient(t)

	title := "Live " + uniqueSlug("page")
	id := createPage(t, c, pageSpec{title: title, slug: uniqueSlug("live"), content: "<p>live words</p>"})
	link := issuePreview(t, c, id)

	for _, target := range []string{link.URL, withToken(t, link.URL, strings.Repeat("0", 32))} {
		status, _, body := c.anonymous(t, target)
		if status != http.StatusOK || !strings.Contains(body, "live words") {
			t.Errorf("GET %s of a published page: status %d", target, status)
		}
	}
}

func TestPreviewLinkRequiresAuthentication(t *testing.T) {
	c, _ := newClient(t)

	id := createPage(t, c, pageSpec{title: "Auth", slug: uniqueSlug("auth"), content: "<p>x</p>", status: "draft"})

	anonymous := &client{base: c.base, http: c.http}
	if status, body := anonymous.request(t, http.MethodPost, previewPath(id), nil); status != http.StatusUnauthorized {
		t.Fatalf("anonymous preview: status %d, want 401, body %s", status, body)
	}
}

func TestPreviewLinkRejectsAnUnknownID(t *testing.T) {
	c, _ := newClient(t)

	status, body := c.request(t, http.MethodPost, previewPath(999999999), nil)
	if status != http.StatusNotFound || !strings.Contains(string(body), "not_found") {
		t.Fatalf("an unknown id: status %d, body %s", status, body)
	}
}
