package sqlite

import (
	"database/sql"
	"io/fs"
	"slices"
	"strings"
	"testing"
)

func TestMigrationsAreEmbedded(t *testing.T) {
	t.Parallel()

	fsys, err := migrations()
	if err != nil {
		t.Fatalf("migrations: %v", err)
	}

	names, err := fs.Glob(fsys, "*.sql")
	if err != nil {
		t.Fatalf("glob: %v", err)
	}

	want := []string{
		"0001_app_meta.sql", "0002_settings.sql", "0003_secrets.sql",
		"0004_sites.sql", "0005_link_policies.sql", "0006_templates.sql", "0007_entities.sql",
		"0008_edges.sql", "0009_pages.sql", "0010_template_overrides.sql",
	}
	if !slices.Equal(names, want) {
		t.Fatalf("embedded migrations = %v, want %v", names, want)
	}

	for _, name := range names {
		body, readErr := fs.ReadFile(fsys, name)
		if readErr != nil {
			t.Fatalf("read %s: %v", name, readErr)
		}
		for _, marker := range []string{"-- +goose Up", "-- +goose Down", "STRICT"} {
			if !strings.Contains(string(body), marker) {
				t.Errorf("%s is missing %q", name, marker)
			}
		}
	}
}

func TestMigrationsRoundTrip(t *testing.T) {
	t.Parallel()

	store := openStore(t, nil)

	provider, err := store.provider()
	if err != nil {
		t.Fatalf("provider: %v", err)
	}

	version, err := provider.GetDBVersion(t.Context())
	if err != nil {
		t.Fatalf("version after up: %v", err)
	}
	if version != 10 {
		t.Fatalf("version after up = %d, want 10", version)
	}

	if _, err = provider.DownTo(t.Context(), 0); err != nil {
		t.Fatalf("down: %v", err)
	}

	for _, table := range []string{"app_meta", "settings", "secrets", "sites", "link_policies", "templates", "entities", "entity_anchors", "edges", "pages", "page_links", "template_overrides"} {
		var name string
		scanErr := store.writer.QueryRowContext(t.Context(),
			`SELECT name FROM sqlite_schema WHERE type = 'table' AND name = ?`, table).Scan(&name)
		if scanErr == nil {
			t.Errorf("%s survived the down migration", table)
		}
	}

	if _, err = provider.Up(t.Context()); err != nil {
		t.Fatalf("up again: %v", err)
	}

	version, err = provider.GetDBVersion(t.Context())
	if err != nil {
		t.Fatalf("version after the second up: %v", err)
	}
	if version != 10 {
		t.Errorf("version after the second up = %d, want 10", version)
	}
}

func TestSchemaCascades(t *testing.T) {
	t.Parallel()

	store := openStore(t, nil)
	exec := func(query string, args ...any) {
		t.Helper()
		if _, err := store.writer.ExecContext(t.Context(), query, args...); err != nil {
			t.Fatalf("%s: %v", query, err)
		}
	}
	count := func(table string) int {
		t.Helper()
		var n int
		if err := store.reader.QueryRowContext(t.Context(), "SELECT count(*) FROM "+table).Scan(&n); err != nil {
			t.Fatalf("count %s: %v", table, err)
		}
		return n
	}
	const at = "2026-09-18T09:00:00Z"

	exec(`INSERT INTO sites (id, name, base_url, secret_ref, status, created_at, updated_at) VALUES ('s1', 'Shop', 'https://shop', 'site:s1:wp_password', 'active', ?, ?)`, at, at)
	exec(`INSERT INTO templates (id, scope, site_id, name, page_kind, version, spec, created_at, updated_at) VALUES ('t1', 'global', NULL, 'Hub', 'hub', 1, '{}', ?, ?)`, at, at)
	exec(`INSERT INTO link_policies (id, scope, site_id, name, rules, forbid_external, forbid_self, anchor_strategy, created_at, updated_at) VALUES ('p1', 'site', 's1', 'Strict', '{}', 1, 1, 'rotate', ?, ?)`, at, at)
	exec(`UPDATE sites SET default_template_id = 't1', default_link_policy_id = 'p1' WHERE id = 's1'`)
	exec(`INSERT INTO entities (id, site_id, name, kind, source, created_at, updated_at) VALUES ('e1', 's1', 'Shoes', 'hub', 'user', ?, ?)`, at, at)
	exec(`INSERT INTO entities (id, site_id, name, kind, source, created_at, updated_at) VALUES ('e2', 's1', 'Boots', 'topic', 'user', ?, ?)`, at, at)
	exec(`INSERT INTO entity_anchors (entity_id, position, text, source, weight) VALUES ('e1', 0, 'shoes', 'user', 1)`)
	exec(`INSERT INTO edges (id, site_id, from_entity_id, to_entity_id, kind, weight, source, status, created_at) VALUES ('g1', 's1', 'e2', 'e1', 'parent', 1, 'user', 'approved', ?)`, at)
	exec(`INSERT INTO pages (id, site_id, path, slug, wp_type, status, entity_id, template_id, created_at, updated_at) VALUES ('pg1', 's1', '/shoes/', 'shoes', 'page', 'planned', 'e1', 't1', ?, ?)`, at, at)
	exec(`INSERT INTO pages (id, site_id, path, slug, parent_page_id, wp_type, status, entity_id, created_at, updated_at) VALUES ('pg2', 's1', '/shoes/boots/', 'boots', 'pg1', 'page', 'planned', 'e2', ?, ?)`, at, at)
	exec(`UPDATE entities SET canonical_page_id = 'pg1' WHERE id = 'e1'`)
	exec(`INSERT INTO page_links (id, site_id, from_page_id, to_page_id, to_url, anchor_text, origin, observed_at) VALUES ('l1', 's1', 'pg2', 'pg1', '/shoes/', 'shoes', 'generated', ?)`, at)
	exec(`INSERT INTO page_links (id, site_id, from_page_id, to_page_id, to_url, anchor_text, origin, observed_at) VALUES ('l2', 's1', 'pg1', 'pg2', '/shoes/boots/', 'boots', 'generated', ?)`, at)
	exec(`INSERT INTO template_overrides (id, template_id, scope, site_id, page_id, patch, created_at, updated_at) VALUES ('o1', 't1', 'site', 's1', NULL, '{}', ?, ?)`, at, at)
	exec(`INSERT INTO template_overrides (id, template_id, scope, site_id, page_id, patch, created_at, updated_at) VALUES ('o2', 't1', 'page', NULL, 'pg1', '{}', ?, ?)`, at, at)

	if _, err := store.writer.ExecContext(t.Context(), `INSERT INTO entities (id, site_id, name, kind, source, created_at, updated_at) VALUES ('e3', 's1', 'shoes', 'hub', 'user', ?, ?)`, at, at); err == nil {
		t.Fatal("entity names must be unique per site regardless of case")
	}
	if _, err := store.writer.ExecContext(t.Context(), `INSERT INTO template_overrides (id, template_id, scope, site_id, page_id, patch, created_at, updated_at) VALUES ('o3', 't1', 'site', 's1', NULL, '{}', ?, ?)`, at, at); err == nil {
		t.Fatal("one override per template and target")
	}
	if _, err := store.writer.ExecContext(t.Context(), `INSERT INTO edges (id, site_id, from_entity_id, to_entity_id, kind, weight, source, status, created_at) VALUES ('g2', 's1', 'e2', 'e1', 'related', 0.5, 'user', 'approved', ?)`, at); err == nil {
		t.Fatal("a related edge must be stored with the lower id first")
	}

	exec(`DELETE FROM pages WHERE id = 'pg1'`)
	var canonical, parent, toPage sql.NullString
	if err := store.reader.QueryRowContext(t.Context(), `SELECT canonical_page_id FROM entities WHERE id = 'e1'`).Scan(&canonical); err != nil || canonical.Valid {
		t.Errorf("canonical after page delete = %v, %v; want NULL", canonical, err)
	}
	if err := store.reader.QueryRowContext(t.Context(), `SELECT parent_page_id FROM pages WHERE id = 'pg2'`).Scan(&parent); err != nil || parent.Valid {
		t.Errorf("parent after page delete = %v, %v; want NULL", parent, err)
	}
	if err := store.reader.QueryRowContext(t.Context(), `SELECT to_page_id FROM page_links WHERE id = 'l1'`).Scan(&toPage); err != nil || toPage.Valid {
		t.Errorf("link target after page delete = %v, %v; want NULL", toPage, err)
	}
	if got := count("page_links"); got != 1 {
		t.Errorf("page_links after page delete = %d, want the incoming link only", got)
	}
	if got := count("template_overrides"); got != 1 {
		t.Errorf("template_overrides after page delete = %d, want the site override only", got)
	}

	exec(`DELETE FROM entities WHERE id = 'e1'`)
	if got := count("edges"); got != 0 {
		t.Errorf("edges after entity delete = %d", got)
	}
	if got := count("entity_anchors"); got != 0 {
		t.Errorf("anchors after entity delete = %d", got)
	}

	exec(`DELETE FROM templates WHERE id = 't1'`)
	var defaultTemplate sql.NullString
	if err := store.reader.QueryRowContext(t.Context(), `SELECT default_template_id FROM sites WHERE id = 's1'`).Scan(&defaultTemplate); err != nil || defaultTemplate.Valid {
		t.Errorf("default template after template delete = %v, %v; want NULL", defaultTemplate, err)
	}

	exec(`DELETE FROM sites WHERE id = 's1'`)
	for _, table := range []string{"entities", "pages", "page_links", "link_policies", "template_overrides"} {
		if got := count(table); got != 0 {
			t.Errorf("%s after site delete = %d", table, got)
		}
	}
}
