//go:build e2e

package e2e_test

import (
	"net/http"
	"slices"
	"strings"
	"testing"
)

func TestSiteAnswersAndCredentialsAuthenticate(t *testing.T) {
	c, env := newClient(t)

	var root struct {
		Name       string   `json:"name"`
		Namespaces []string `json:"namespaces"`
	}
	c.expect(t, http.MethodGet, "/wp-json/", nil, http.StatusOK, &root)

	if !slices.Contains(root.Namespaces, "postulator/v1") {
		t.Fatalf("postulator/v1 is not registered; namespaces are %v", root.Namespaces)
	}

	var me struct {
		Slug string `json:"slug"`
	}
	c.expect(t, http.MethodGet, "/wp-json/wp/v2/users/me", nil, http.StatusOK, &me)
	if !strings.EqualFold(me.Slug, env.user) {
		t.Fatalf("authenticated as %q, want %q", me.Slug, env.user)
	}
}
