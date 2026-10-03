package sqlite

import (
	"database/sql"
	"io/fs"
	"slices"
	"strings"
	"testing"
)

const latestMigration = 36

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
		"0011_model_catalog.sql", "0012_model_profiles.sql", "0013_llm_calls.sql",
		"0014_runs.sql", "0015_import_mappings.sql", "0016_run_items_target.sql", "0017_agent.sql", "0018_schedules.sql",
		"0019_edge_reason.sql",
		"0020_conversation_title_settled.sql",
		"0021_model_reasoning_effort.sql",
		"0022_cached_input_tokens.sql",
		"0023_run_item_seq.sql",
		"0024_page_observed.sql",
		"0025_run_kind_repair_revert.sql",
		"0026_run_item_note.sql",
		"0027_page_keywords.sql",
		"0028_run_item_blocked_by.sql",
		"0029_keyword_lists.sql",
		"0030_entity_scope.sql",
		"0031_site_commerce.sql",
		"0032_page_planned_path.sql",
		"0033_entity_site_category.sql",
		"0034_entity_terms.sql",
		"0035_llm_call_usage_detail.sql",
		"0036_model_catalog_tier_prices.sql",
	}
	if !slices.Equal(names, want) {
		t.Fatalf("embedded migrations = %v, want %v", names, want)
	}

	for _, name := range names {
		body, readErr := fs.ReadFile(fsys, name)
		if readErr != nil {
			t.Fatalf("read %s: %v", name, readErr)
		}
		markers := []string{"-- +goose Up", "-- +goose Down"}
		if strings.Contains(string(body), "CREATE TABLE") {
			markers = append(markers, "STRICT")
		}
		for _, marker := range markers {
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
	if version != latestMigration {
		t.Fatalf("version after up = %d, want %d", version, latestMigration)
	}

	if _, err = provider.DownTo(t.Context(), 0); err != nil {
		t.Fatalf("down: %v", err)
	}

	for _, table := range []string{"app_meta", "settings", "secrets", "sites", "link_policies", "templates", "entities", "entity_anchors", "entity_terms", "edges", "pages", "page_links", "template_overrides", "model_catalog", "model_profiles", "llm_calls", "runs", "run_items", "artifacts", "step_execs", "run_events", "conversations", "messages", "conversation_histories", "pending_actions", "tool_calls", "schedules"} {
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
	if version != latestMigration {
		t.Errorf("version after the second up = %d, want %d", version, latestMigration)
	}
}

func TestKeywordListsMigrationCarriesTheOldKeywordsOver(t *testing.T) {
	t.Parallel()

	store := openStore(t, nil)
	provider, err := store.provider()
	if err != nil {
		t.Fatalf("provider: %v", err)
	}
	if _, err = provider.DownTo(t.Context(), 28); err != nil {
		t.Fatalf("down to 28: %v", err)
	}

	exec := func(query string, args ...any) {
		t.Helper()
		if _, execErr := store.writer.ExecContext(t.Context(), query, args...); execErr != nil {
			t.Fatalf("%s: %v", query, execErr)
		}
	}
	text := func(query string, args ...any) string {
		t.Helper()
		var value string
		if scanErr := store.writer.QueryRowContext(t.Context(), query, args...).Scan(&value); scanErr != nil {
			t.Fatalf("%s: %v", query, scanErr)
		}
		return value
	}
	const at = "2026-09-18T09:00:00Z"

	cases := []struct {
		name      string
		id        string
		primary   string
		rest      string
		list      string
		backFirst string
		backRest  string
	}{
		{
			name: "the primary keyword leads and its repeat is dropped", id: "k1",
			primary: "running shoes", rest: `["trail shoes","Running Shoes"," road shoes ",""]`,
			list:      `[{"text":"running shoes"},{"text":"trail shoes"},{"text":"road shoes"}]`,
			backFirst: "running shoes", backRest: `["trail shoes","road shoes"]`,
		},
		{
			name: "secondary keywords alone keep their order", id: "k2",
			primary: "", rest: `["first","second"]`,
			list:      `[{"text":"first"},{"text":"second"}]`,
			backFirst: "first", backRest: `["second"]`,
		},
		{
			name: "a primary keyword alone is trimmed", id: "k3",
			primary: " solo ", rest: `[]`,
			list:      `[{"text":"solo"}]`,
			backFirst: "solo", backRest: `[]`,
		},
		{
			name: "no keywords at all", id: "k4",
			primary: "", rest: `[]`,
			list:      `[]`,
			backFirst: "", backRest: `[]`,
		},
	}

	exec(`INSERT INTO sites (id, name, base_url, secret_ref, status, created_at, updated_at) VALUES ('s1', 'Shop', 'https://shop', 'site:s1:wp_password', 'active', ?, ?)`, at, at)
	for _, tc := range cases {
		exec(`INSERT INTO entities (id, site_id, name, kind, primary_keyword, secondary_keywords, source, created_at, updated_at) VALUES (?, 's1', ?, 'topic', ?, ?, 'user', ?, ?)`,
			tc.id, tc.id, tc.primary, tc.rest, at, at)
		exec(`INSERT INTO pages (id, site_id, path, slug, wp_type, status, primary_keyword, keywords, created_at, updated_at) VALUES (?, 's1', ?, ?, 'page', 'planned', ?, ?, ?, ?)`,
			tc.id, "/"+tc.id+"/", tc.id, tc.primary, tc.rest, at, at)
	}

	if _, err = provider.Up(t.Context()); err != nil {
		t.Fatalf("up: %v", err)
	}
	for _, tc := range cases {
		if got := text(`SELECT keywords FROM entities WHERE id = ?`, tc.id); got != tc.list {
			t.Errorf("%s: entity keywords = %s, want %s", tc.name, got, tc.list)
		}
		if got := text(`SELECT keywords FROM pages WHERE id = ?`, tc.id); got != tc.list {
			t.Errorf("%s: page keywords = %s, want %s", tc.name, got, tc.list)
		}
		if got := text(`SELECT notes FROM pages WHERE id = ?`, tc.id); got != `[]` {
			t.Errorf("%s: page notes = %s, want an empty list", tc.name, got)
		}
	}

	exec(`UPDATE entities SET keywords = '[{"text":"measured","volume":900},{"text":"plain"}]' WHERE id = 'k4'`)
	exec(`UPDATE pages SET keywords = '[{"text":"measured","volume":900},{"text":"plain"}]', notes = '[{"label":"Notes","text":"kept"}]' WHERE id = 'k4'`)

	if _, err = provider.DownTo(t.Context(), 28); err != nil {
		t.Fatalf("down again: %v", err)
	}
	for _, tc := range cases {
		first, rest := tc.backFirst, tc.backRest
		if tc.id == "k4" {
			first, rest = "measured", `["plain"]`
		}
		if got := text(`SELECT primary_keyword FROM entities WHERE id = ?`, tc.id); got != first {
			t.Errorf("%s: entity primary keyword after the down = %q, want %q", tc.name, got, first)
		}
		if got := text(`SELECT secondary_keywords FROM entities WHERE id = ?`, tc.id); got != rest {
			t.Errorf("%s: entity secondary keywords after the down = %s, want %s", tc.name, got, rest)
		}
		if got := text(`SELECT primary_keyword FROM pages WHERE id = ?`, tc.id); got != first {
			t.Errorf("%s: page primary keyword after the down = %q, want %q", tc.name, got, first)
		}
		if got := text(`SELECT keywords FROM pages WHERE id = ?`, tc.id); got != rest {
			t.Errorf("%s: page keywords after the down = %s, want %s", tc.name, got, rest)
		}
	}

	if _, err = provider.Up(t.Context()); err != nil {
		t.Fatalf("up after the down: %v", err)
	}
}

func TestAnEntityNameIsUniqueUnderItsParent(t *testing.T) {
	t.Parallel()

	store := openStore(t, nil)
	exec := func(query string, args ...any) error {
		_, err := store.writer.ExecContext(t.Context(), query, args...)
		return err
	}
	const at = "2026-10-02T09:00:00Z"
	entity := func(id, name string, scope any) error {
		return exec(`INSERT INTO entities (id, site_id, name, kind, scope_entity_id, source, created_at, updated_at) VALUES (?, 's1', ?, 'topic', ?, 'import', ?, ?)`,
			id, name, scope, at, at)
	}

	if err := exec(`INSERT INTO sites (id, name, base_url, secret_ref, status, created_at, updated_at) VALUES ('s1', 'Shop', 'https://shop', 'site:s1:wp_password', 'active', ?, ?)`, at, at); err != nil {
		t.Fatalf("site: %v", err)
	}
	for _, step := range []struct {
		id, name string
		scope    any
		ok       bool
		why      string
	}{
		{id: "bpc", name: "BPC-157", ok: true, why: "a root"},
		{id: "tb", name: "TB-500", ok: true, why: "another root"},
		{id: "bpc-liquid", name: "Liquid", scope: "bpc", ok: true, why: "a name under one parent"},
		{id: "tb-liquid", name: "Liquid", scope: "tb", ok: true, why: "the same name under another parent"},
		{id: "bpc-liquid-2", name: "liquid", scope: "bpc", ok: false, why: "the same name twice under one parent"},
		{id: "bpc-root", name: "bpc-157", ok: false, why: "the same name twice at the top"},
		{id: "self", name: "Self", scope: "self", ok: false, why: "an entity under itself"},
	} {
		err := entity(step.id, step.name, step.scope)
		if step.ok && err != nil {
			t.Fatalf("%s: %v", step.why, err)
		}
		if !step.ok && err == nil {
			t.Fatalf("%s was accepted", step.why)
		}
	}

	if err := exec(`DELETE FROM entities WHERE id = 'bpc'`); err != nil {
		t.Fatalf("delete a parent: %v", err)
	}
	var scope sql.NullString
	if err := store.reader.QueryRowContext(t.Context(), `SELECT scope_entity_id FROM entities WHERE id = 'bpc-liquid'`).Scan(&scope); err != nil || scope.Valid {
		t.Fatalf("the scope of a child whose parent is gone = %v, %v; want NULL", scope, err)
	}
}

func TestEntityScopeMigrationTakesTheParentAnEntityHas(t *testing.T) {
	t.Parallel()

	store := openStore(t, nil)
	provider, err := store.provider()
	if err != nil {
		t.Fatalf("provider: %v", err)
	}
	if _, err = provider.DownTo(t.Context(), 29); err != nil {
		t.Fatalf("down to 29: %v", err)
	}
	exec := func(query string, args ...any) {
		t.Helper()
		if _, execErr := store.writer.ExecContext(t.Context(), query, args...); execErr != nil {
			t.Fatalf("%s: %v", query, execErr)
		}
	}
	text := func(query string) string {
		t.Helper()
		var value sql.NullString
		if scanErr := store.writer.QueryRowContext(t.Context(), query).Scan(&value); scanErr != nil {
			t.Fatalf("%s: %v", query, scanErr)
		}
		return value.String
	}
	const at = "2026-10-02T09:00:00Z"

	exec(`INSERT INTO sites (id, name, base_url, secret_ref, status, created_at, updated_at) VALUES ('s1', 'Shop', 'https://shop', 'site:s1:wp_password', 'active', ?, ?)`, at, at)
	for _, row := range [][2]string{{"a", "Peptides"}, {"b", "Healing"}, {"c", "BPC-157"}, {"d", "Lone"}} {
		exec(`INSERT INTO entities (id, site_id, name, kind, source, created_at, updated_at) VALUES (?, 's1', ?, 'topic', 'import', ?, ?)`, row[0], row[1], at, at)
	}
	exec(`INSERT INTO edges (id, site_id, from_entity_id, to_entity_id, kind, weight, source, status, created_at) VALUES ('g1', 's1', 'c', 'b', 'parent', 1, 'import', 'approved', '2026-10-02T09:05:00Z')`)
	exec(`INSERT INTO edges (id, site_id, from_entity_id, to_entity_id, kind, weight, source, status, created_at) VALUES ('g2', 's1', 'c', 'a', 'parent', 1, 'import', 'approved', '2026-10-02T09:01:00Z')`)
	exec(`INSERT INTO edges (id, site_id, from_entity_id, to_entity_id, kind, weight, source, status, created_at) VALUES ('g3', 's1', 'b', 'a', 'parent', 1, 'import', 'proposed', ?)`, at)

	if _, err = provider.Up(t.Context()); err != nil {
		t.Fatalf("up: %v", err)
	}
	if got := text(`SELECT scope_entity_id FROM entities WHERE id = 'c'`); got != "a" {
		t.Fatalf("the scope of an entity with two parents = %q, want the earlier one", got)
	}
	for _, id := range []string{"a", "b", "d"} {
		if got := text(`SELECT scope_entity_id FROM entities WHERE id = '` + id + `'`); got != "" {
			t.Fatalf("%s took the scope %q, want none", id, got)
		}
	}
	if got := text(`SELECT keywords FROM entities WHERE id = 'c'`); got != "[]" {
		t.Fatalf("the rebuild lost the keywords column: %q", got)
	}

	exec(`INSERT INTO entities (id, site_id, name, kind, scope_entity_id, source, created_at, updated_at) VALUES ('e', 's1', 'Healing', 'topic', 'c', 'import', ?, ?)`, at, at)
	if _, err = provider.DownTo(t.Context(), 29); err != nil {
		t.Fatalf("down with a shared name: %v", err)
	}
	if got := text(`SELECT count(DISTINCT lower(name)) FROM entities`); got != "5" {
		t.Fatalf("the down migration left %s distinct names for 5 entities", got)
	}
	if got := text(`SELECT name FROM entities WHERE id = 'b'`); got != "Healing" {
		t.Fatalf("the first of a shared name was renamed to %q", got)
	}
}

func TestSiteCategoryMigrationFlagsNoEntity(t *testing.T) {
	t.Parallel()

	store := openStore(t, nil)
	provider, err := store.provider()
	if err != nil {
		t.Fatalf("provider: %v", err)
	}
	if _, err = provider.DownTo(t.Context(), 32); err != nil {
		t.Fatalf("down to 32: %v", err)
	}
	exec := func(query string, args ...any) error {
		_, execErr := store.writer.ExecContext(t.Context(), query, args...)
		return execErr
	}
	count := func(query string) int {
		t.Helper()
		var n int
		if scanErr := store.writer.QueryRowContext(t.Context(), query).Scan(&n); scanErr != nil {
			t.Fatalf("%s: %v", query, scanErr)
		}
		return n
	}
	const at = "2026-10-03T09:00:00Z"

	if err = exec(`INSERT INTO sites (id, name, base_url, secret_ref, status, created_at, updated_at) VALUES ('s1', 'Shop', 'https://shop', 'site:s1:wp_password', 'active', ?, ?)`, at, at); err != nil {
		t.Fatalf("site: %v", err)
	}
	for _, row := range []struct {
		id, name, kind string
		scope          any
	}{
		{id: "root", name: "Peptides", kind: "category"},
		{id: "category", name: "Healing", kind: "category", scope: "root"},
		{id: "topic", name: "BPC-157", kind: "topic", scope: "category"},
	} {
		if err = exec(`INSERT INTO entities (id, site_id, name, kind, scope_entity_id, source, created_at, updated_at) VALUES (?, 's1', ?, ?, ?, 'import', ?, ?)`,
			row.id, row.name, row.kind, row.scope, at, at); err != nil {
			t.Fatalf("entity %s: %v", row.id, err)
		}
	}

	if _, err = provider.UpTo(t.Context(), 33); err != nil {
		t.Fatalf("up to 33: %v", err)
	}
	if got := count(`SELECT count(*) FROM entities WHERE site_category <> 0`); got != 0 {
		t.Fatalf("the migration made %d entities site categories, want none", got)
	}
	for _, step := range []struct {
		why   string
		value any
		ok    bool
	}{
		{why: "a flagged entity", value: 1, ok: true},
		{why: "an entity no longer flagged", value: 0, ok: true},
		{why: "a flag that is neither", value: 2},
		{why: "a negative flag", value: -1},
	} {
		err = exec(`UPDATE entities SET site_category = ? WHERE id = 'category'`, step.value)
		if step.ok && err != nil {
			t.Fatalf("%s: %v", step.why, err)
		}
		if !step.ok && err == nil {
			t.Fatalf("%s was accepted", step.why)
		}
	}

	if err = exec(`UPDATE entities SET site_category = 1 WHERE id = 'category'`); err != nil {
		t.Fatalf("flag: %v", err)
	}
	if _, err = provider.DownTo(t.Context(), 32); err != nil {
		t.Fatalf("down to 32 with a flagged entity: %v", err)
	}
	if got := count(`SELECT count(*) FROM entities`); got != 3 {
		t.Fatalf("the down migration left %d entities, want 3", got)
	}
	if err = exec(`UPDATE entities SET site_category = 1`); err == nil {
		t.Fatal("the column survived the down migration")
	}
	if _, err = provider.Up(t.Context()); err != nil {
		t.Fatalf("up again: %v", err)
	}
}

func TestEntityTermsSchema(t *testing.T) {
	t.Parallel()

	store := openStore(t, nil)
	exec := func(query string, args ...any) error {
		_, err := store.writer.ExecContext(t.Context(), query, args...)
		return err
	}
	count := func(query string) int {
		t.Helper()
		var n int
		if err := store.reader.QueryRowContext(t.Context(), query).Scan(&n); err != nil {
			t.Fatalf("%s: %v", query, err)
		}
		return n
	}
	const at = "2026-10-03T09:00:00Z"
	term := func(entityID, siteID, taxonomy string, termID, parentTermID int) error {
		return exec(`INSERT INTO entity_terms (entity_id, site_id, taxonomy, term_id, parent_term_id, name, run_id, seen_at) VALUES (?, ?, ?, ?, ?, 'Healing', '', ?)`,
			entityID, siteID, taxonomy, termID, parentTermID, at)
	}

	for _, setup := range []string{
		`INSERT INTO sites (id, name, base_url, secret_ref, status, created_at, updated_at) VALUES ('s1', 'Shop', 'https://shop', 'site:s1:wp_password', 'active', ?, ?)`,
		`INSERT INTO entities (id, site_id, name, kind, source, created_at, updated_at) VALUES ('e1', 's1', 'Healing', 'category', 'import', ?, ?)`,
		`INSERT INTO entities (id, site_id, name, kind, source, created_at, updated_at) VALUES ('e2', 's1', 'Recovery', 'category', 'import', ?, ?)`,
	} {
		if err := exec(setup, at, at); err != nil {
			t.Fatalf("%s: %v", setup, err)
		}
	}

	for _, step := range []struct {
		why                     string
		entityID, siteID, taxon string
		termID, parentTermID    int
		ok                      bool
	}{
		{why: "a category term", entityID: "e1", siteID: "s1", taxon: "category", termID: 5, ok: true},
		{why: "the same entity as a product category", entityID: "e1", siteID: "s1", taxon: "product_cat", termID: 5, parentTermID: 2, ok: true},
		{why: "a term another entity already maps to", entityID: "e2", siteID: "s1", taxon: "category", termID: 5, ok: true},
		{why: "a second term for one entity in one taxonomy", entityID: "e1", siteID: "s1", taxon: "category", termID: 6},
		{why: "a taxonomy Postulator does not write", entityID: "e2", siteID: "s1", taxon: "post_tag", termID: 7},
		{why: "a term id of zero", entityID: "e2", siteID: "s1", taxon: "product_cat", termID: 0},
		{why: "a negative parent", entityID: "e2", siteID: "s1", taxon: "product_cat", termID: 7, parentTermID: -1},
		{why: "an entity that does not exist", entityID: "e9", siteID: "s1", taxon: "product_cat", termID: 7},
		{why: "a site that does not exist", entityID: "e2", siteID: "s9", taxon: "product_cat", termID: 7},
	} {
		err := term(step.entityID, step.siteID, step.taxon, step.termID, step.parentTermID)
		if step.ok && err != nil {
			t.Fatalf("%s: %v", step.why, err)
		}
		if !step.ok && err == nil {
			t.Fatalf("%s was accepted", step.why)
		}
	}

	if got := count(`SELECT count(*) FROM sqlite_schema WHERE type = 'index' AND name = 'entity_terms_site_taxonomy_term'`); got != 1 {
		t.Fatalf("the lookup index by site, taxonomy and term is missing")
	}

	if err := exec(`DELETE FROM entities WHERE id = 'e2'`); err != nil {
		t.Fatalf("delete an entity: %v", err)
	}
	if got := count(`SELECT count(*) FROM entity_terms`); got != 2 {
		t.Fatalf("entity_terms after an entity delete = %d, want the other entity's 2", got)
	}
	if err := exec(`DELETE FROM sites WHERE id = 's1'`); err != nil {
		t.Fatalf("delete the site: %v", err)
	}
	if got := count(`SELECT count(*) FROM entity_terms`); got != 0 {
		t.Fatalf("entity_terms after a site delete = %d, want none", got)
	}
}

func TestSpendColumns(t *testing.T) {
	t.Parallel()

	store := openStore(t, nil)
	exec := func(query string, args ...any) error {
		_, err := store.writer.ExecContext(t.Context(), query, args...)
		return err
	}
	const at = "2026-10-03T09:00:00Z"

	if err := exec(`INSERT INTO llm_calls (id, run_id, item_id, step, conversation_id, provider, model, input_tokens, output_tokens, usd, latency_ms, status, error_code, created_at) VALUES ('c1', '', '', 'chat', 'conv', 'openai', 'gpt-5.6-terra', 10, 5, 0.1, 30, 'ok', '', ?)`, at); err != nil {
		t.Fatalf("a call written before the detail columns: %v", err)
	}
	var reasoning, written int
	var tier string
	if err := store.reader.QueryRowContext(t.Context(), `SELECT reasoning_tokens, cache_write_tokens, service_tier FROM llm_calls WHERE id = 'c1'`).Scan(&reasoning, &written, &tier); err != nil {
		t.Fatalf("read the detail columns: %v", err)
	}
	if reasoning != 0 || written != 0 || tier != "" {
		t.Fatalf("defaults = %d reasoning, %d written, tier %q; want zeros and no tier", reasoning, written, tier)
	}
	var index string
	if err := store.reader.QueryRowContext(t.Context(), `SELECT name FROM sqlite_schema WHERE type = 'index' AND name = 'llm_calls_created'`).Scan(&index); err != nil {
		t.Fatalf("the index the spend window reads is missing: %v", err)
	}

	model := `INSERT INTO model_catalog (provider, model, context_tokens, max_output_tokens, input_usd_per_m, output_usd_per_m, rpm, tpm, supports_structured, supports_images, reasoning, enabled, created_at, updated_at) VALUES ('openai', ?, 1000, 100, 2, 12, 1, 1, 1, 0, 1, 1, ?, ?)`
	for _, step := range []struct {
		model, column string
		price         float64
		ok            bool
	}{
		{model: "a", column: "cache_write_usd_per_m", price: 2.5, ok: true},
		{model: "b", column: "flex_input_usd_per_m", price: 1, ok: true},
		{model: "c", column: "flex_cached_input_usd_per_m", price: 0.1, ok: true},
		{model: "d", column: "flex_cache_write_usd_per_m", price: 1.25, ok: true},
		{model: "e", column: "flex_output_usd_per_m", price: 6, ok: true},
		{model: "f", column: "cache_write_usd_per_m", price: -1},
		{model: "g", column: "flex_input_usd_per_m", price: -1},
		{model: "h", column: "flex_cached_input_usd_per_m", price: -0.1},
		{model: "i", column: "flex_cache_write_usd_per_m", price: -1},
		{model: "j", column: "flex_output_usd_per_m", price: -6},
	} {
		if err := exec(model, step.model, at, at); err != nil {
			t.Fatalf("model %s: %v", step.model, err)
		}
		err := exec(`UPDATE model_catalog SET `+step.column+` = ? WHERE model = ?`, step.price, step.model)
		if step.ok && err != nil {
			t.Fatalf("%s = %v: %v", step.column, step.price, err)
		}
		if !step.ok && err == nil {
			t.Fatalf("%s = %v was accepted", step.column, step.price)
		}
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

func TestRunSchemaCascades(t *testing.T) {
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
	exec(`INSERT INTO pages (id, site_id, path, slug, wp_type, status, created_at, updated_at) VALUES ('pg1', 's1', '/shoes/', 'shoes', 'page', 'planned', ?, ?)`, at, at)
	exec(`INSERT INTO runs (id, site_id, kind, status, targets, recipe, publish_mode, created_by, deadline_at, created_at) VALUES ('r1', 's1', 'generate', 'pending', '["pg1"]', '[]', 'draft', 'user', ?, ?)`, at, at)
	exec(`INSERT INTO run_items (id, run_id, site_id, target_id, status, current_step, created_at, updated_at) VALUES ('i1', 'r1', 's1', 'pg1', 'pending', 'resolve_context', ?, ?)`, at, at)
	exec(`INSERT INTO artifacts (id, run_id, item_id, step, kind, blob, size, hash, created_at) VALUES ('a1', 'r1', 'i1', 'generate_body', 'body_html', x'3c703e', 3, 'hash', ?)`, at)
	exec(`INSERT INTO step_execs (id, run_id, item_id, step, attempt, status, input_hash, artifact_id, started_at) VALUES ('x1', 'r1', 'i1', 'generate_body', 1, 'done', 'ih', 'a1', ?)`, at)
	exec(`INSERT INTO run_events (id, run_id, seq, type, at, payload) VALUES ('v1', 'r1', 1, 'run.queued', ?, '{}')`, at)

	if _, err := store.writer.ExecContext(t.Context(), `INSERT INTO run_events (id, run_id, seq, type, at, payload) VALUES ('v2', 'r1', 1, 'run.started', ?, '{}')`, at); err == nil {
		t.Fatal("a run event sequence must be unique per run")
	}
	if _, err := store.writer.ExecContext(t.Context(), `INSERT INTO artifacts (id, run_id, item_id, step, kind, blob, size, hash, created_at) VALUES ('a2', 'r1', 'i1', 'generate_body', 'body_html', x'3c703e', 3, 'hash', ?)`, at); err == nil {
		t.Fatal("one artifact per item, step and kind")
	}
	if _, err := store.writer.ExecContext(t.Context(), `INSERT INTO step_execs (id, run_id, item_id, step, attempt, status, input_hash, started_at) VALUES ('x2', 'r1', 'i1', 'generate_body', 1, 'started', 'ih', ?)`, at); err == nil {
		t.Fatal("one step execution per item, step and attempt")
	}

	exec(`DELETE FROM artifacts WHERE id = 'a1'`)
	var artifact sql.NullString
	if err := store.reader.QueryRowContext(t.Context(), `SELECT artifact_id FROM step_execs WHERE id = 'x1'`).Scan(&artifact); err != nil || artifact.Valid {
		t.Errorf("artifact reference after artifact delete = %v, %v; want NULL", artifact, err)
	}

	exec(`DELETE FROM pages WHERE id = 'pg1'`)
	if got := count("run_items"); got != 1 {
		t.Errorf("run_items after page delete = %d, want the item kept as run history", got)
	}

	exec(`DELETE FROM sites WHERE id = 's1'`)
	for _, table := range []string{"runs", "run_items", "artifacts", "step_execs", "run_events"} {
		if got := count(table); got != 0 {
			t.Errorf("%s after site delete = %d", table, got)
		}
	}
}
