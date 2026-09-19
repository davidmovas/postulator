package wp_test

import (
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/davidmovas/postulator/internal/adapters/wp"
	"github.com/davidmovas/postulator/internal/adapters/wp/wptest"
)

var httpMethods = []string{"get", "put", "post", "delete", "patch"}

func contractDocument(t *testing.T) string {
	t.Helper()

	raw, err := os.ReadFile(filepath.Join("..", "..", "..", "wp-plugin", "openapi.yaml"))
	if err != nil {
		t.Fatalf("read the plugin contract: %v", err)
	}
	return strings.ReplaceAll(string(raw), "\r\n", "\n")
}

func documentedRoutes(t *testing.T) map[string][]string {
	t.Helper()

	routes := make(map[string][]string)
	current := ""
	inPaths := false
	for _, line := range strings.Split(contractDocument(t), "\n") {
		switch {
		case line == "paths:":
			inPaths = true
		case inPaths && line != "" && !strings.HasPrefix(line, " "):
			inPaths = false
		case inPaths && strings.HasPrefix(line, "  /") && strings.HasSuffix(line, ":"):
			current = strings.TrimSuffix(strings.TrimSpace(line), ":")
			routes[current] = nil
		case inPaths && current != "" && strings.HasPrefix(line, "    ") && !strings.HasPrefix(line, "     "):
			name := strings.TrimSuffix(strings.TrimSpace(line), ":")
			if slices.Contains(httpMethods, name) {
				routes[current] = append(routes[current], name)
			}
		}
	}
	return routes
}

func TestThePluginContractDocumentsEveryRoute(t *testing.T) {
	t.Parallel()

	want := map[string][]string{
		"/manifest":             {"get"},
		"/content":              {"get"},
		"/seo-meta/{id}":        {"put"},
		"/content/{id}/raw":     {"get", "put"},
		"/content/{id}/preview": {"post"},
	}

	routes := documentedRoutes(t)
	if len(routes) != len(want) {
		t.Fatalf("the contract documents %d routes, want %d", len(routes), len(want))
	}
	for route, methods := range want {
		documented, ok := routes[route]
		if !ok {
			t.Errorf("the contract does not document %s", route)
			continue
		}
		slices.Sort(documented)
		if !slices.Equal(documented, methods) {
			t.Errorf("%s documents %v, want %v", route, documented, methods)
		}
	}
}

func TestThePluginContractDeclaresItsShapes(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		snippet string
	}{
		{name: "openapi version", snippet: "openapi: 3.1.0"},
		{name: "error schema", snippet: "\n    Error:\n"},
		{name: "manifest schema", snippet: "\n    Manifest:\n"},
		{name: "content page schema", snippet: "\n    ContentPage:\n"},
		{name: "content item schema", snippet: "\n    ContentItem:\n"},
		{name: "content link schema", snippet: "\n    ContentLink:\n"},
		{name: "seo request schema", snippet: "\n    SeoMetaRequest:\n"},
		{name: "seo result schema", snippet: "\n    SeoMetaResult:\n"},
		{name: "raw content schema", snippet: "\n    RawContent:\n"},
		{name: "raw update schema", snippet: "\n    RawUpdateRequest:\n"},
		{name: "raw result schema", snippet: "\n    RawUpdateResult:\n"},
		{name: "conflict schema", snippet: "\n    HashConflict:\n"},
		{name: "hash algorithm", snippet: "sha256"},
		{name: "conflict code", snippet: "hash_mismatch"},
		{name: "term rejection code", snippet: "not_found"},
		{name: "clamped limit", snippet: "maximum: 500"},
		{name: "nullable cursor", snippet: "- \"null\""},
		{name: "yoast key", snippet: "_yoast_wpseo_title"},
		{name: "rank math key", snippet: "rank_math_title"},
		{name: "fallback key", snippet: "_postulator_title"},
		{name: "basic auth", snippet: "scheme: basic"},
		{name: "content hash property", snippet: "\n        contentHash:\n"},
		{name: "next cursor property", snippet: "\n        nextCursor:\n"},
		{name: "capabilities property", snippet: "\n        capabilities:\n"},
		{name: "seo plugin property", snippet: "\n        seoPlugin:\n"},
		{name: "current hash property", snippet: "\n        currentHash:\n"},
		{name: "applied property", snippet: "\n        applied:\n"},
		{name: "links property", snippet: "\n        links:\n"},
		{name: "meta property", snippet: "\n        meta:\n"},
		{name: "heading property", snippet: "\n        h1:\n"},
		{name: "path property", snippet: "\n        path:\n"},
		{name: "modified property", snippet: "\n        modified:\n"},
		{name: "expected hash property", snippet: "\n        expectedHash:\n"},
		{name: "anchor property", snippet: "\n        anchor:\n"},
		{name: "href property", snippet: "\n        href:\n"},
		{name: "preview link schema", snippet: "\n    PreviewLink:\n"},
		{name: "expires at property", snippet: "\n        expiresAt:\n"},
		{name: "preview query argument", snippet: "postulator_preview"},
		{name: "preview capability", snippet: "\n              - preview\n"},
		{name: "plugin version", snippet: "\n  version: 1.1.0\n"},
	}

	document := contractDocument(t)
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if !strings.Contains(document, tc.snippet) {
				t.Errorf("the contract does not carry %q", tc.snippet)
			}
		})
	}
}

func TestTheClientHitsOnlyDocumentedPluginRoutes(t *testing.T) {
	t.Parallel()

	server := wptest.New(t)
	seeded := server.Seed(wptest.Item{Type: wptest.TypePage, Title: "Koffein", Content: "<p>x</p>"})
	client := newClient(t, server)

	if _, err := client.Capabilities(t.Context()); err != nil {
		t.Fatalf("Capabilities: %v", err)
	}
	if _, err := client.ListContent(t.Context(), wp.ContentQuery{}); err != nil {
		t.Fatalf("ListContent: %v", err)
	}
	if _, err := client.SetSEOMeta(t.Context(), seeded[0].ID, wp.SEOMeta{Title: "x"}); err != nil {
		t.Fatalf("SetSEOMeta: %v", err)
	}
	raw, err := client.GetRaw(t.Context(), seeded[0].ID)
	if err != nil {
		t.Fatalf("GetRaw: %v", err)
	}
	if _, err = client.PutRaw(t.Context(), seeded[0].ID, "<p>y</p>", raw.ContentHash); err != nil {
		t.Fatalf("PutRaw: %v", err)
	}
	if _, err = client.PreviewLink(t.Context(), seeded[0].ID); err != nil {
		t.Fatalf("PreviewLink: %v", err)
	}

	documented := documentedRoutes(t)
	id := strconv.FormatInt(seeded[0].ID, 10)
	replacer := strings.NewReplacer("/"+id+"/", "/{id}/", "/"+id, "/{id}")

	for _, recorded := range server.Requests() {
		route, found := strings.CutPrefix(recorded.Path, "/wp-json/postulator/v1")
		if !found {
			continue
		}
		route = replacer.Replace(route)
		if _, ok := documented[route]; !ok {
			t.Errorf("the client called %s, which the contract does not document", route)
		}
	}
}
