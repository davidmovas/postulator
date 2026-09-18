//go:build e2e

package e2e_test

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type environment struct {
	baseURL string
	user    string
	pass    string
	seo     string
	woo     bool
}

type link struct {
	Href   string `json:"href"`
	Anchor string `json:"anchor"`
}

type itemMeta struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Canonical   string `json:"canonical"`
}

type contentItem struct {
	ID          int      `json:"id"`
	Type        string   `json:"type"`
	Slug        string   `json:"slug"`
	Path        string   `json:"path"`
	Parent      int      `json:"parent"`
	Status      string   `json:"status"`
	Modified    string   `json:"modified"`
	ContentHash string   `json:"contentHash"`
	Title       string   `json:"title"`
	H1          string   `json:"h1"`
	Meta        itemMeta `json:"meta"`
	Links       []link   `json:"links"`
}

type contentPage struct {
	Items      []contentItem `json:"items"`
	NextCursor *string       `json:"nextCursor"`
}

type pageSpec struct {
	title   string
	slug    string
	content string
	parent  int
	status  string
}

func repoRoot(t *testing.T) string {
	t.Helper()

	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("working directory: %v", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("no go.mod above %s", dir)
		}
		dir = parent
	}
}

func envPath(t *testing.T) string {
	t.Helper()

	if override := os.Getenv("POSTULATOR_E2E_ENV"); override != "" {
		return override
	}
	return filepath.Join(repoRoot(t), "docker", "e2e", ".env.generated")
}

func loadEnvironment(t *testing.T) environment {
	t.Helper()

	path := envPath(t)
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("%s is missing; run `task e2e:up` first: %v", path, err)
	}

	values := make(map[string]string)
	for _, line := range strings.Split(string(raw), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, found := strings.Cut(line, "=")
		if !found {
			t.Fatalf("%s has a line without '=': %q", path, line)
		}
		values[strings.TrimSpace(key)] = strings.TrimSpace(value)
	}

	env := environment{
		baseURL: strings.TrimSuffix(values["E2E_WP_URL"], "/"),
		user:    values["E2E_WP_USER"],
		pass:    values["E2E_WP_APP_PASSWORD"],
		seo:     values["E2E_SEO"],
		woo:     values["E2E_WOO"] == "1",
	}
	for name, value := range map[string]string{
		"E2E_WP_URL":          env.baseURL,
		"E2E_WP_USER":         env.user,
		"E2E_WP_APP_PASSWORD": env.pass,
		"E2E_SEO":             env.seo,
	} {
		if value == "" {
			t.Fatalf("%s does not set %s", path, name)
		}
	}
	return env
}

type client struct {
	base string
	user string
	pass string
	http *http.Client
}

func newClient(t *testing.T) (*client, environment) {
	t.Helper()

	env := loadEnvironment(t)
	return &client{
		base: env.baseURL,
		user: env.user,
		pass: env.pass,
		http: &http.Client{Timeout: 60 * time.Second},
	}, env
}

func (c *client) request(t *testing.T, method, path string, payload any) (status int, body []byte) {
	t.Helper()

	var sent io.Reader
	if payload != nil {
		encoded, err := json.Marshal(payload)
		if err != nil {
			t.Fatalf("encode %s %s: %v", method, path, err)
		}
		sent = bytes.NewReader(encoded)
	}

	request, err := http.NewRequest(method, c.base+path, sent)
	if err != nil {
		t.Fatalf("build %s %s: %v", method, path, err)
	}
	if c.user != "" {
		request.SetBasicAuth(c.user, c.pass)
	}
	if payload != nil {
		request.Header.Set("Content-Type", "application/json")
	}

	response, err := c.http.Do(request)
	if err != nil {
		t.Fatalf("call %s %s: %v", method, path, err)
	}
	defer response.Body.Close()

	raw, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read %s %s: %v", method, path, err)
	}
	return response.StatusCode, raw
}

func (c *client) expect(t *testing.T, method, path string, payload any, status int, out any) {
	t.Helper()

	code, raw := c.request(t, method, path, payload)
	if code != status {
		t.Fatalf("%s %s: status %d, want %d, body %s", method, path, code, status, raw)
	}
	if out == nil {
		return
	}
	if err := json.Unmarshal(raw, out); err != nil {
		t.Fatalf("decode %s %s: %v, body %s", method, path, err, raw)
	}
}

func (c *client) fetchPage(t *testing.T, target string) string {
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
	if response.StatusCode != http.StatusOK {
		t.Fatalf("GET %s: status %d, want 200", target, response.StatusCode)
	}
	return string(raw)
}

func hashOf(content string) string {
	sum := sha256.Sum256([]byte(content))
	return hex.EncodeToString(sum[:])
}

func uniqueSlug(prefix string) string {
	return fmt.Sprintf("%s-%d", prefix, time.Now().UnixNano())
}

func createPage(t *testing.T, c *client, spec pageSpec) int {
	t.Helper()

	status := spec.status
	if status == "" {
		status = "publish"
	}
	body := map[string]any{
		"title":   spec.title,
		"slug":    spec.slug,
		"content": spec.content,
		"status":  status,
	}
	if spec.parent != 0 {
		body["parent"] = spec.parent
	}

	var created struct {
		ID int `json:"id"`
	}
	c.expect(t, http.MethodPost, "/wp-json/wp/v2/pages", body, http.StatusCreated, &created)
	if created.ID == 0 {
		t.Fatalf("created page has id 0")
	}
	t.Cleanup(func() {
		c.request(t, http.MethodDelete, fmt.Sprintf("/wp-json/wp/v2/pages/%d?force=true", created.ID), nil)
	})
	return created.ID
}

func listContent(t *testing.T, c *client, query string) contentPage {
	t.Helper()

	var page contentPage
	c.expect(t, http.MethodGet, "/wp-json/postulator/v1/content?"+query, nil, http.StatusOK, &page)
	return page
}

func walkContent(t *testing.T, c *client, types string, limit int, visit func(contentItem) bool) {
	t.Helper()

	cursor := ""
	for range 200 {
		query := fmt.Sprintf("types=%s&limit=%d", types, limit)
		if cursor != "" {
			query += "&cursor=" + url.QueryEscape(cursor)
		}
		page := listContent(t, c, query)
		for i := range page.Items {
			if visit(page.Items[i]) {
				return
			}
		}
		if page.NextCursor == nil {
			return
		}
		cursor = *page.NextCursor
	}
	t.Fatalf("walking types=%s did not terminate", types)
}

func findItem(t *testing.T, c *client, types string, id int) contentItem {
	t.Helper()

	var found contentItem
	walkContent(t, c, types, 100, func(item contentItem) bool {
		if item.ID == id {
			found = item
			return true
		}
		return false
	})
	if found.ID == 0 {
		t.Fatalf("id %d was not listed under types=%s", id, types)
	}
	return found
}

func findBySlug(t *testing.T, c *client, types, slug string) contentItem {
	t.Helper()

	var found contentItem
	walkContent(t, c, types, 100, func(item contentItem) bool {
		if item.Slug == slug {
			found = item
			return true
		}
		return false
	})
	if found.ID == 0 {
		t.Fatalf("slug %q was not listed under types=%s", slug, types)
	}
	return found
}
