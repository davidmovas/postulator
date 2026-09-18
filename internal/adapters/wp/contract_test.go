package wp_test

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
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
		"/manifest":         {"get"},
		"/content":          {"get"},
		"/seo-meta/{id}":    {"put"},
		"/content/{id}/raw": {"get", "put"},
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
