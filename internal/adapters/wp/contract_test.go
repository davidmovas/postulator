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
	"github.com/davidmovas/postulator/internal/domain/field"
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
		"/manifest":              {"get"},
		"/content":               {"get"},
		"/seo-meta/{id}":         {"get", "put"},
		"/content/{id}/raw":      {"get", "put"},
		"/content/{id}/preview":  {"post"},
		"/content/{id}/rendered": {"get"},
		"/meta/{id}":             {"get", "put"},
		"/terms/{id}":            {"get"},
		"/settings":              {"get"},
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
		{name: "seo state schema", snippet: "\n    SeoMetaState:\n"},
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
		{name: "fallback key", snippet: "_postulator_seo_title"},
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
		{name: "content posts only", snippet: "The id of a content post, a post of a public post type other than an\n        attachment"},
		{name: "href classification", snippet: "classified as\n            pagemap.Site.Resolve classifies it"},
		{name: "href host and port", snippet: "its host and port equal the site's host and port"},
		{name: "relative href against the document", snippet: "a relative href is internal and resolves\n            against the page's URL as a browser does: its path up to the\n            last slash."},
		{name: "preview link schema", snippet: "\n    PreviewLink:\n"},
		{name: "rendered content schema", snippet: "\n    RenderedContent:\n"},
		{name: "rendered capability", snippet: "\n              - rendered\n"},
		{name: "rendered inside a singular main query", snippet: "runs through the_content inside a singular main"},
		{name: "meta values schema", snippet: "\n    MetaValues:\n"},
		{name: "meta capability", snippet: "\n              - meta\n"},
		{name: "meta deny rule", snippet: "is_protected_meta says it is protected"},
		{name: "meta refusal code", snippet: "meta_key_denied"},
		{name: "meta key ceiling", snippet: "too_many_keys"},
		{name: "term schema", snippet: "\n    Term:\n"},
		{name: "post terms schema", snippet: "\n    PostTerms:\n"},
		{name: "terms capability", snippet: "\n              - terms\n"},
		{name: "technical pages schema", snippet: "\n    TechnicalPages:\n"},
		{name: "settings capability", snippet: "\n              - settings\n"},
		{name: "privacy page", snippet: "\n            privacy:\n"},
		{name: "excerpt property", snippet: "\n        excerpt:\n"},
		{name: "terms property", snippet: "\n        terms:\n"},
		{name: "post types property", snippet: "\n        postTypes:\n"},
		{name: "taxonomies property", snippet: "\n        taxonomies:\n"},
		{name: "unknown type code", snippet: "invalid_type"},
		{name: "expires at property", snippet: "\n        expiresAt:\n"},
		{name: "preview query argument", snippet: "preview_token query argument"},
		{name: "preview capability", snippet: "\n              - preview\n"},
		{name: "seo read capability", snippet: "\n              - seo_meta_read\n"},
		{name: "plugin version", snippet: "\n  version: 1.5.1\n"},
		{name: "meta write request schema", snippet: "\n    MetaWriteRequest:\n"},
		{name: "meta write result schema", snippet: "\n    MetaWriteResult:\n"},
		{name: "moved conflict schema", snippet: "\n    MovedConflict:\n"},
		{name: "meta write capability", snippet: "\n              - meta_write\n"},
		{name: "guarded write capability", snippet: "\n              - guarded_write\n"},
		{name: "unfiltered html property", snippet: "\n        unfilteredHtml:\n"},
		{name: "save stable property", snippet: "\n        saveStable:\n"},
		{name: "stored hashes property", snippet: "\n        hashes:\n"},
		{name: "moved property", snippet: "\n        moved:\n"},
		{name: "unguarded meta key code", snippet: "missing_expected"},
		{name: "save filters named", snippet: "pre_post_content and content_save_pre"},
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

func documentedDenySet(t *testing.T) (keys, prefixes []string) {
	t.Helper()

	var list *[]string
	inDeny := false
	for line := range strings.SplitSeq(contractDocument(t), "\n") {
		switch {
		case line == "x-denied-meta:":
			inDeny = true
		case inDeny && line != "" && !strings.HasPrefix(line, " "):
			inDeny = false
		case inDeny && line == "  keys:":
			list = &keys
		case inDeny && line == "  prefixes:":
			list = &prefixes
		case inDeny && list != nil && strings.HasPrefix(line, "    - "):
			*list = append(*list, strings.Trim(strings.TrimPrefix(line, "    - "), "\""))
		case inDeny && strings.HasPrefix(line, "  ") && !strings.HasPrefix(line, "   "):
			list = nil
		}
	}
	return keys, prefixes
}

func TestTheContractsDenySetIncludesTheCatalogs(t *testing.T) {
	t.Parallel()

	keys, prefixes := documentedDenySet(t)
	denied := field.DeniedMeta()
	slices.Sort(keys)
	slices.Sort(prefixes)
	slices.Sort(denied.Keys)
	slices.Sort(denied.Prefixes)
	if !slices.Equal(keys, denied.Keys) {
		t.Errorf("the contract denies the keys %v, and the catalog %v", keys, denied.Keys)
	}
	if !slices.Equal(prefixes, denied.Prefixes) {
		t.Errorf("the contract denies the prefixes %v, and the catalog %v", prefixes, denied.Prefixes)
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
	if _, err := client.SetSEOMeta(t.Context(), wp.TypePage, seeded[0].ID, wp.SEOMeta{Title: "x"}, nil); err != nil {
		t.Fatalf("SetSEOMeta: %v", err)
	}
	if _, err := client.GetSEOMeta(t.Context(), wp.TypePage, seeded[0].ID); err != nil {
		t.Fatalf("GetSEOMeta: %v", err)
	}
	if _, err := client.ReplaceSEOMeta(t.Context(), wp.TypePage, seeded[0].ID, wp.SEOMeta{}, []string{"title"}, nil); err != nil {
		t.Fatalf("ReplaceSEOMeta: %v", err)
	}
	raw, err := client.GetRaw(t.Context(), wp.TypePage, seeded[0].ID)
	if err != nil {
		t.Fatalf("GetRaw: %v", err)
	}
	if _, err = client.PutRaw(t.Context(), wp.TypePage, seeded[0].ID, "<p>y</p>", raw.ContentHash); err != nil {
		t.Fatalf("PutRaw: %v", err)
	}
	if _, err = client.PreviewLink(t.Context(), wp.TypePage, seeded[0].ID); err != nil {
		t.Fatalf("PreviewLink: %v", err)
	}
	if _, err = client.GetRendered(t.Context(), wp.TypePage, seeded[0].ID); err != nil {
		t.Fatalf("GetRendered: %v", err)
	}
	if _, err = client.GetMeta(t.Context(), wp.TypePage, seeded[0].ID, []string{"color"}); err != nil {
		t.Fatalf("GetMeta: %v", err)
	}
	if _, err = client.GetTerms(t.Context(), wp.TypePage, seeded[0].ID); err != nil {
		t.Fatalf("GetTerms: %v", err)
	}
	if _, err = client.PluginSettings(t.Context()); err != nil {
		t.Fatalf("PluginSettings: %v", err)
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
