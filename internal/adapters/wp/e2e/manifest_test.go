//go:build e2e

package e2e_test

import (
	"net/http"
	"regexp"
	"slices"
	"testing"
)

type manifest struct {
	Version      string   `json:"version"`
	Capabilities []string `json:"capabilities"`
	SEOPlugin    string   `json:"seoPlugin"`
	WPVersion    string   `json:"wpVersion"`
	Site         string   `json:"site"`
}

func readManifest(t *testing.T, c *client) manifest {
	t.Helper()

	var out manifest
	c.expect(t, http.MethodGet, "/wp-json/postulator/v1/manifest", nil, http.StatusOK, &out)
	return out
}

func TestManifestDescribesTheSite(t *testing.T) {
	c, env := newClient(t)

	got := readManifest(t, c)

	if got.Version != "1.2.0" {
		t.Errorf("version = %q, want 1.2.0", got.Version)
	}
	want := []string{"bulk", "seo_meta", "seo_meta_read", "content_hash", "raw", "preview"}
	if !slices.Equal(got.Capabilities, want) {
		t.Errorf("capabilities = %v, want %v", got.Capabilities, want)
	}
	if got.SEOPlugin != env.seo {
		t.Errorf("seoPlugin = %q, want %q", got.SEOPlugin, env.seo)
	}
	if !regexp.MustCompile(`^\d+\.\d+`).MatchString(got.WPVersion) {
		t.Errorf("wpVersion = %q, want a major.minor version", got.WPVersion)
	}
	if got.Site != env.baseURL {
		t.Errorf("site = %q, want %q", got.Site, env.baseURL)
	}
}

func TestManifestRequiresAuthentication(t *testing.T) {
	c, _ := newClient(t)

	anonymous := &client{base: c.base, http: c.http}
	status, body := anonymous.request(t, http.MethodGet, "/wp-json/postulator/v1/manifest", nil)
	if status != http.StatusUnauthorized {
		t.Fatalf("anonymous manifest: status %d, want 401, body %s", status, body)
	}
}
