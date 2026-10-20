//go:build e2e

package e2e_test

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
)

const restrictedRole = "e2e_restricted"

type metaWrite struct {
	Meta     map[string]any `json:"meta"`
	Expected map[string]any `json:"expected"`
}

type storedMeta struct {
	ID   int `json:"id"`
	Meta map[string]struct {
		Value *string `json:"value"`
		Hash  *string `json:"hash"`
	} `json:"meta"`
}

type movedRefusal struct {
	Code  string             `json:"code"`
	Moved map[string]*string `json:"moved"`
}

type rawRead struct {
	Content     string `json:"content"`
	ContentHash string `json:"contentHash"`
	SaveStable  bool   `json:"saveStable"`
}

func putMetaPath(id int) string {
	return fmt.Sprintf("/wp-json/postulator/v1/meta/%d", id)
}

func restrictedClient(t *testing.T, c *client, env environment) *client {
	t.Helper()

	wpCLI(t, env, "eval", fmt.Sprintf(
		"if ( ! get_role( '%s' ) ) { add_role( '%s', 'E2E restricted', array_diff_key( get_role( 'editor' )->capabilities, array( 'unfiltered_html' => true ) ) ); }",
		restrictedRole, restrictedRole))
	login := uniqueSlug("restricted")
	id := wpCLI(t, env, "user", "create", login, login+"@example.test", "--role="+restrictedRole, "--porcelain")
	t.Cleanup(func() { wpCLI(t, env, "user", "delete", id, "--yes") })
	password := wpCLI(t, env, "user", "application-password", "create", id, "e2e", "--porcelain")
	return &client{base: c.base, user: login, pass: password, http: c.http}
}

func TestMetaWritesOnlyWhatTheUserMayEdit(t *testing.T) {
	c, env := newClient(t)

	page := createPage(t, c, pageSpec{title: "Meta writer", slug: uniqueSlug("meta-write"), content: "<p>x</p>"})
	wpCLI(t, env, "eval", fmt.Sprintf(
		"update_post_meta(%[1]d, 'subtitle', 'Old'); update_post_meta(%[1]d, 'spec_table', array('a' => 1)); "+
			"add_post_meta(%[1]d, 'twice', 'one'); add_post_meta(%[1]d, 'twice', 'two'); update_post_meta(%[1]d, '_secret', 'x');",
		page))

	var written storedMeta
	c.expect(t, http.MethodPut, putMetaPath(page), metaWrite{
		Meta:     map[string]any{"subtitle": "New", "tagline": "Fresh"},
		Expected: map[string]any{"subtitle": hashOf("Old"), "tagline": nil},
	}, http.StatusOK, &written)
	subtitle := written.Meta["subtitle"]
	if written.ID != page || subtitle.Value == nil || *subtitle.Value != "New" || subtitle.Hash == nil || *subtitle.Hash != hashOf("New") {
		t.Errorf("the write answered %+v", written)
	}
	var read metaValues
	c.expect(t, http.MethodGet, metaPath(page, "subtitle,tagline"), nil, http.StatusOK, &read)
	if read.Meta["subtitle"] != "New" || read.Meta["tagline"] != "Fresh" {
		t.Errorf("the site holds %+v", read.Meta)
	}

	var moved movedRefusal
	c.expect(t, http.MethodPut, putMetaPath(page), metaWrite{
		Meta:     map[string]any{"subtitle": "Again", "tagline": "Changed"},
		Expected: map[string]any{"subtitle": hashOf("Old"), "tagline": hashOf("Fresh")},
	}, http.StatusConflict, &moved)
	if moved.Code != "hash_mismatch" || moved.Moved["subtitle"] == nil || *moved.Moved["subtitle"] != hashOf("New") || len(moved.Moved) != 1 {
		t.Errorf("a moved value answers %+v, want hash_mismatch naming the subtitle alone", moved)
	}
	c.expect(t, http.MethodGet, metaPath(page, "tagline"), nil, http.StatusOK, &read)
	if read.Meta["tagline"] != "Fresh" {
		t.Errorf("a refused write changed the tagline to %v", read.Meta["tagline"])
	}

	for _, key := range []string{"total_sales", "TOTAL_SALES", "attribute_pa_color", "rank_math_title", "_price", "_secret", "spec_table", "twice"} {
		var refused metaRefusal
		c.expect(t, http.MethodPut, putMetaPath(page), metaWrite{
			Meta: map[string]any{key: "9.99"}, Expected: map[string]any{key: nil},
		}, http.StatusBadRequest, &refused)
		if refused.Code != "meta_key_denied" || len(refused.Data.Keys) != 1 || refused.Data.Keys[0] != key {
			t.Errorf("a write of %s answers %+v, want meta_key_denied naming it", key, refused)
		}
	}
	var unguarded metaRefusal
	c.expect(t, http.MethodPut, putMetaPath(page), metaWrite{Meta: map[string]any{"subtitle": "x"}, Expected: map[string]any{}},
		http.StatusBadRequest, &unguarded)
	if unguarded.Code != "missing_expected" {
		t.Errorf("an unguarded key answers %+v, want missing_expected", unguarded)
	}
	var notText metaRefusal
	c.expect(t, http.MethodPut, putMetaPath(page), metaWrite{
		Meta: map[string]any{"subtitle": []string{"a"}}, Expected: map[string]any{"subtitle": hashOf("New")},
	}, http.StatusBadRequest, &notText)
	if notText.Code != "meta_key_denied" {
		t.Errorf("a value that is no text answers %+v, want meta_key_denied", notText)
	}

	var deleted storedMeta
	c.expect(t, http.MethodPut, putMetaPath(page), metaWrite{
		Meta: map[string]any{"tagline": nil}, Expected: map[string]any{"tagline": hashOf("Fresh")},
	}, http.StatusOK, &deleted)
	if gone := deleted.Meta["tagline"]; gone.Value != nil || gone.Hash != nil {
		t.Errorf("a deleted key answers %+v", gone)
	}
	c.expect(t, http.MethodGet, metaPath(page, "tagline"), nil, http.StatusOK, &read)
	if held, present := read.Meta["tagline"]; !present || held != nil {
		t.Errorf("the deleted tagline reads %v", held)
	}

	if status, _ := (&client{base: c.base, http: c.http}).request(t, http.MethodPut, putMetaPath(page), metaWrite{
		Meta: map[string]any{"subtitle": "x"}, Expected: map[string]any{"subtitle": hashOf("New")},
	}); status != http.StatusUnauthorized {
		t.Errorf("an anonymous write = %d, want 401", status)
	}
}

func TestTheRawWriteAnswersWhatWasStored(t *testing.T) {
	c, env := newClient(t)
	restricted := restrictedClient(t, c, env)

	page := createPage(t, c, pageSpec{title: "Raw holder", slug: uniqueSlug("raw-stored"), content: "<p>Before</p>"})
	var before rawRead
	restricted.expect(t, http.MethodGet, rawPath(page), nil, http.StatusOK, &before)

	sent := `<p>Kept</p><iframe src="https://example.test/map"></iframe>`
	var written struct {
		Content     string `json:"content"`
		ContentHash string `json:"contentHash"`
	}
	restricted.expect(t, http.MethodPut, rawPath(page), map[string]any{"content": sent, "expectedHash": before.ContentHash}, http.StatusOK, &written)
	var after rawRead
	c.expect(t, http.MethodGet, rawPath(page), nil, http.StatusOK, &after)
	if written.Content != after.Content || written.ContentHash != after.ContentHash || written.ContentHash != hashOf(written.Content) {
		t.Errorf("the write answered %q under %s, and the site holds %q under %s", written.Content, written.ContentHash, after.Content, after.ContentHash)
	}
	if written.Content == sent || strings.Contains(written.Content, "<iframe") {
		t.Errorf("a user without unfiltered HTML stored %q, want kses to have taken the iframe", written.Content)
	}
}

func TestSaveStabilityFollowsTheUsersFilters(t *testing.T) {
	c, env := newClient(t)
	restricted := restrictedClient(t, c, env)

	framed := createPage(t, c, pageSpec{title: "Framed", slug: uniqueSlug("framed"), content: `<p>Map</p><iframe src="https://example.test/map"></iframe>`})
	plain := createPage(t, c, pageSpec{title: "Plain", slug: uniqueSlug("plain"), content: "<p>Only prose.</p>"})
	cases := []struct {
		name   string
		reader *client
		page   int
		stable bool
	}{
		{name: "an administrator over an iframe", reader: c, page: framed, stable: true},
		{name: "a restricted user over an iframe", reader: restricted, page: framed},
		{name: "a restricted user over plain prose", reader: restricted, page: plain, stable: true},
	}
	for _, tc := range cases {
		var raw rawRead
		tc.reader.expect(t, http.MethodGet, rawPath(tc.page), nil, http.StatusOK, &raw)
		if raw.SaveStable != tc.stable {
			t.Errorf("%s: saveStable = %t, want %t", tc.name, raw.SaveStable, tc.stable)
		}
	}

	var admin, limited struct {
		UnfilteredHTML bool `json:"unfilteredHtml"`
	}
	c.expect(t, http.MethodGet, "/wp-json/postulator/v1/manifest", nil, http.StatusOK, &admin)
	restricted.expect(t, http.MethodGet, "/wp-json/postulator/v1/manifest", nil, http.StatusOK, &limited)
	if !admin.UnfilteredHTML || limited.UnfilteredHTML {
		t.Errorf("unfilteredHtml = %t for the administrator and %t for the restricted user", admin.UnfilteredHTML, limited.UnfilteredHTML)
	}
}

func TestSEOMetaWritesUnderTheHashesRead(t *testing.T) {
	c, env := newClient(t)

	page := createPage(t, c, pageSpec{title: "SEO holder", slug: uniqueSlug("seo-guard"), content: "<p>x</p>"})
	var first struct {
		Applied []string          `json:"applied"`
		Hashes  map[string]string `json:"hashes"`
	}
	c.expect(t, http.MethodPut, fmt.Sprintf("/wp-json/postulator/v1/seo-meta/%d", page), map[string]any{
		"title": "Ours", "expected": map[string]string{"title": hashOf("")},
	}, http.StatusOK, &first)
	if len(first.Applied) != 1 || first.Hashes["title"] != hashOf("Ours") {
		t.Errorf("the guarded write answered %+v", first)
	}

	key := map[string]string{"yoast": "_yoast_wpseo_title", "rankmath": "rank_math_title"}[standOf(t, env).seoPlugin]
	if key == "" {
		key = "_postulator_seo_title"
	}
	wpCLI(t, env, "post", "meta", "update", fmt.Sprint(page), key, "The client's own")
	var moved movedRefusal
	c.expect(t, http.MethodPut, fmt.Sprintf("/wp-json/postulator/v1/seo-meta/%d", page), map[string]any{
		"title": "Again", "description": "About", "expected": map[string]string{"title": hashOf("Ours"), "description": hashOf("")},
	}, http.StatusConflict, &moved)
	if moved.Code != "hash_mismatch" || len(moved.Moved) != 1 || moved.Moved["title"] == nil || *moved.Moved["title"] != hashOf("The client's own") {
		t.Errorf("a moved title answers %+v", moved)
	}
	var state struct {
		Title       string `json:"title"`
		Description string `json:"description"`
	}
	c.expect(t, http.MethodGet, fmt.Sprintf("/wp-json/postulator/v1/seo-meta/%d", page), nil, http.StatusOK, &state)
	if state.Title != "The client's own" || state.Description != "" {
		t.Errorf("a refused write changed the SEO meta: %+v", state)
	}
}
