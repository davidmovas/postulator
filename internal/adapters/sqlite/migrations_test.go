package sqlite

import (
	"database/sql"
	"encoding/json"
	stderrors "errors"
	"fmt"
	"io/fs"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/davidmovas/postulator/internal/domain/field"
	"github.com/davidmovas/postulator/internal/domain/pagetype"
	"github.com/davidmovas/postulator/internal/domain/template"
	"github.com/davidmovas/postulator/internal/kernel/id"
)

const latestMigration = 82

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
		"0037_drop_model_reasoning_effort.sql",
		"0038_drop_removed_provider_profiles.sql",
		"0039_categories.sql",
		"0040_category_terms.sql",
		"0041_page_category.sql",
		"0042_drop_entity_terms.sql",
		"0043_drop_entity_site_category.sql",
		"0044_drop_categories.sql",
		"0045_pages_wp_type_open.sql",
		"0046_pages_model_columns.sql",
		"0047_page_identity.sql",
		"0048_page_terms.sql",
		"0049_page_fields.sql",
		"0050_page_bodies.sql",
		"0051_page_renders.sql",
		"0052_page_links_typed.sql",
		"0053_findings.sql",
		"0054_site_settings.sql",
		"0055_pages_parent_method.sql",
		"0056_front_page_not_technical.sql",
		"0057_page_renders_archive.sql",
		"0058_page_types.sql",
		"0059_field_scopes.sql",
		"0060_page_types_backfill.sql",
		"0061_entity_card.sql",
		"0062_link_rules_in_scopes.sql",
		"0063_kinds_become_types.sql",
		"0064_conversation_brief.sql",
		"0065_page_links_absolute.sql",
		"0066_page_types_default_templates.sql",
		"0067_template_specifications_retired.sql",
		"0068_pages_site_copies_dropped.sql",
		"0069_page_links_origin_dropped.sql",
		"0070_page_links_block_hash.sql",
		"0071_page_renders_theme_h1.sql",
		"0072_pages_workbook_hierarchy.sql",
		"0073_runs_kind_edit.sql",
		"0074_page_map.sql",
		"0075_page_renders_h1.sql",
		"0076_page_type_presets.sql",
		"0077_page_type_rules.sql",
		"0078_pages_type_rule.sql",
		"0079_pages_site_title.sql",
		"0080_page_map_type_trail.sql",
		"0081_page_map_depth.sql",
		"0082_run_brief.sql",
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

	for _, table := range []string{"app_meta", "settings", "secrets", "sites", "link_policies", "templates", "entities", "entity_anchors", "entity_terms", "categories", "category_terms", "edges", "pages", "page_links", "page_identity", "page_terms", "page_fields", "page_bodies", "page_blocks_written", "page_renders", "findings", "page_types", "field_scopes", "template_overrides", "model_catalog", "model_profiles", "llm_calls", "runs", "run_items", "artifacts", "step_execs", "run_events", "conversations", "messages", "conversation_histories", "pending_actions", "tool_calls", "schedules", "page_map", "page_map_sites"} {
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
		return exec(`INSERT INTO entities (id, site_id, name, scope_entity_id, source, created_at, updated_at) VALUES (?, 's1', ?, ?, 'import', ?, ?)`,
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
	for _, entityID := range []string{"a", "b", "d"} {
		if got := text(`SELECT scope_entity_id FROM entities WHERE id = '` + entityID + `'`); got != "" {
			t.Fatalf("%s took the scope %q, want none", entityID, got)
		}
	}
	if got := text(`SELECT keywords FROM entities WHERE id = 'c'`); got != "[]" {
		t.Fatalf("the rebuild lost the keywords column: %q", got)
	}

	exec(`INSERT INTO entities (id, site_id, name, scope_entity_id, source, created_at, updated_at) VALUES ('e', 's1', 'Healing', 'c', 'import', ?, ?)`, at, at)
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

func schemaOf(t *testing.T, store *Store, table string) []string {
	t.Helper()

	rows, err := store.writer.QueryContext(t.Context(),
		`SELECT type || ' ' || name || ': ' || sql FROM sqlite_schema WHERE tbl_name = ? AND sql IS NOT NULL ORDER BY type DESC, name`, table)
	if err != nil {
		t.Fatalf("read the schema of %s: %v", table, err)
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var entry string
		if scanErr := rows.Scan(&entry); scanErr != nil {
			t.Fatalf("scan the schema of %s: %v", table, scanErr)
		}
		out = append(out, entry)
	}
	if rowsErr := rows.Err(); rowsErr != nil {
		t.Fatalf("read the schema of %s: %v", table, rowsErr)
	}
	return out
}

func columnOf(t *testing.T, store *Store, table, column string) string {
	t.Helper()

	var described string
	err := store.writer.QueryRowContext(t.Context(),
		`SELECT type || ' notnull=' || "notnull" || ' default=' || coalesce(dflt_value, 'NULL') || ' pk=' || pk FROM pragma_table_xinfo(?) WHERE name = ?`,
		table, column).Scan(&described)
	if stderrors.Is(err, sql.ErrNoRows) {
		return ""
	}
	if err != nil {
		t.Fatalf("read the column %s.%s: %v", table, column, err)
	}
	return described
}

func storeAt(t *testing.T, version int64) *Store {
	t.Helper()

	store := openStore(t, nil)
	provider, err := store.provider()
	if err != nil {
		t.Fatalf("provider: %v", err)
	}
	if _, err = provider.DownTo(t.Context(), 0); err != nil {
		t.Fatalf("down to 0: %v", err)
	}
	if _, err = provider.UpTo(t.Context(), version); err != nil {
		t.Fatalf("up to %d: %v", version, err)
	}
	return store
}

func rowsOf(t *testing.T, store *Store, query string, args ...any) []string {
	t.Helper()

	rows, err := store.writer.QueryContext(t.Context(), query, args...)
	if err != nil {
		t.Fatalf("%s: %v", query, err)
	}
	defer rows.Close()
	columns, err := rows.Columns()
	if err != nil {
		t.Fatalf("the columns of %s: %v", query, err)
	}
	var out []string
	for rows.Next() {
		values := make([]any, len(columns))
		targets := make([]any, len(columns))
		for i := range values {
			targets[i] = &values[i]
		}
		if scanErr := rows.Scan(targets...); scanErr != nil {
			t.Fatalf("scan %s: %v", query, scanErr)
		}
		cells := make([]string, len(values))
		for i, value := range values {
			cells[i] = fmt.Sprint(value)
		}
		out = append(out, strings.Join(cells, " | "))
	}
	if rowsErr := rows.Err(); rowsErr != nil {
		t.Fatalf("%s: %v", query, rowsErr)
	}
	return out
}

func TestEntityTermsAreDroppedAndADownBringsThemBackAsMigration34MadeThem(t *testing.T) {
	t.Parallel()

	want := schemaOf(t, storeAt(t, 34), "entity_terms")
	if len(want) != 2 {
		t.Fatalf("migration 34 made %v, want the table and its index", want)
	}

	store := openStore(t, nil)
	provider, err := store.provider()
	if err != nil {
		t.Fatalf("provider: %v", err)
	}
	exec := func(query string, args ...any) {
		t.Helper()
		if _, execErr := store.writer.ExecContext(t.Context(), query, args...); execErr != nil {
			t.Fatalf("%s: %v", query, execErr)
		}
	}
	count := func(query string) int {
		t.Helper()
		var n int
		if scanErr := store.writer.QueryRowContext(t.Context(), query).Scan(&n); scanErr != nil {
			t.Fatalf("%s: %v", query, scanErr)
		}
		return n
	}
	const at = "2026-10-04T09:00:00Z"

	if got := schemaOf(t, store, "entity_terms"); len(got) != 0 {
		t.Fatalf("entity_terms survived the up migration: %v", got)
	}

	exec(`INSERT INTO sites (id, name, base_url, secret_ref, status, created_at, updated_at) VALUES ('s1', 'Shop', 'https://shop', 'site:s1:wp_password', 'active', ?, ?)`, at, at)
	exec(`INSERT INTO entities (id, site_id, name, source, created_at, updated_at) VALUES ('e1', 's1', 'Healing', 'import', ?, ?)`, at, at)

	if _, err = provider.DownTo(t.Context(), 41); err != nil {
		t.Fatalf("down to 41: %v", err)
	}
	if got := schemaOf(t, store, "entity_terms"); !slices.Equal(got, want) {
		t.Fatalf("the down migration made\n%s\nwant it as migration 34 made it\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
	exec(`INSERT INTO entity_terms (entity_id, site_id, taxonomy, term_id, name, seen_at) VALUES ('e1', 's1', 'category', 5, 'Healing', ?)`, at)

	if _, err = provider.Up(t.Context()); err != nil {
		t.Fatalf("up again with a term stored: %v", err)
	}
	if got := schemaOf(t, store, "entity_terms"); len(got) != 0 {
		t.Fatalf("entity_terms survived the second up: %v", got)
	}
	if got := count(`SELECT count(*) FROM entities WHERE id = 'e1'`); got != 1 {
		t.Fatalf("dropping the terms left %d of the one entity", got)
	}
}

func TestSiteCategoryIsDroppedAndADownBringsItBackAsMigration33MadeIt(t *testing.T) {
	t.Parallel()

	made := storeAt(t, 33)
	want := columnOf(t, made, "entities", "site_category")
	if want == "" {
		t.Fatal("migration 33 made no site_category column")
	}
	wantSchema := schemaOf(t, made, "entities")

	store := openStore(t, nil)
	provider, err := store.provider()
	if err != nil {
		t.Fatalf("provider: %v", err)
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
	const at = "2026-10-04T09:00:00Z"

	if got := columnOf(t, store, "entities", "site_category"); got != "" {
		t.Fatalf("entities still carries site_category (%s) after the up migration", got)
	}
	for _, setup := range []string{
		`INSERT INTO sites (id, name, base_url, secret_ref, status, created_at, updated_at) VALUES ('s1', 'Shop', 'https://shop', 'site:s1:wp_password', 'active', ?, ?)`,
		`INSERT INTO entities (id, site_id, name, source, created_at, updated_at) VALUES ('e1', 's1', 'Healing', 'import', ?, ?)`,
	} {
		if err = exec(setup, at, at); err != nil {
			t.Fatalf("%s: %v", setup, err)
		}
	}

	if _, err = provider.DownTo(t.Context(), 42); err != nil {
		t.Fatalf("down to 42: %v", err)
	}
	if got := columnOf(t, store, "entities", "site_category"); got != want {
		t.Fatalf("the down migration made site_category %q, want %q as migration 33 made it", got, want)
	}
	if got := schemaOf(t, store, "entities"); !slices.Equal(got, wantSchema) {
		t.Fatalf("the down migration left entities as\n%s\nwant it as migration 33 made it\n%s", strings.Join(got, "\n"), strings.Join(wantSchema, "\n"))
	}
	if got := count(`SELECT count(*) FROM entities WHERE id = 'e1' AND site_category = 0`); got != 1 {
		t.Fatal("an entity stored before the down is not left unflagged")
	}
	for _, step := range []struct {
		why   string
		value any
		ok    bool
	}{
		{why: "a flagged entity", value: 1, ok: true},
		{why: "an entity no longer flagged", value: 0, ok: true},
		{why: "a flag that is neither", value: 2},
		{why: "no flag at all", value: nil},
	} {
		err = exec(`UPDATE entities SET site_category = ? WHERE id = 'e1'`, step.value)
		if step.ok && err != nil {
			t.Fatalf("%s: %v", step.why, err)
		}
		if !step.ok && err == nil {
			t.Fatalf("%s was accepted", step.why)
		}
	}

	if err = exec(`UPDATE entities SET site_category = 1 WHERE id = 'e1'`); err != nil {
		t.Fatalf("flag the entity: %v", err)
	}
	if _, err = provider.Up(t.Context()); err != nil {
		t.Fatalf("up again with a flagged entity: %v", err)
	}
	if got := columnOf(t, store, "entities", "site_category"); got != "" {
		t.Fatalf("entities carries site_category (%s) after the second up", got)
	}
	if got := count(`SELECT count(*) FROM entities`); got != 1 {
		t.Fatalf("dropping the column left %d entities, want 1", got)
	}
}

func TestCategoriesSchema(t *testing.T) {
	t.Parallel()

	store := storeAt(t, 43)
	exec := func(query string, args ...any) error {
		_, err := store.writer.ExecContext(t.Context(), query, args...)
		return err
	}
	count := func(query string, args ...any) int {
		t.Helper()
		var n int
		if err := store.reader.QueryRowContext(t.Context(), query, args...).Scan(&n); err != nil {
			t.Fatalf("%s: %v", query, err)
		}
		return n
	}
	const at = "2026-10-04T09:00:00Z"
	insert := func(id, siteID, name, key string, parentID any) error {
		return exec(`INSERT INTO categories (id, site_id, name, name_key, parent_id, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)`,
			id, siteID, name, key, parentID, at, at)
	}

	for _, siteID := range []string{"s1", "s2"} {
		if err := exec(`INSERT INTO sites (id, name, base_url, secret_ref, status, created_at, updated_at) VALUES (?, 'Shop', 'https://shop', 'site:' || ? || ':wp_password', 'active', ?, ?)`, siteID, siteID, at, at); err != nil {
			t.Fatalf("site %s: %v", siteID, err)
		}
	}

	for _, step := range []struct {
		why                   string
		id, siteID, name, key string
		parentID              any
		ok                    bool
	}{
		{why: "a top-level category", id: "healing", siteID: "s1", name: "Healing", key: "healing", ok: true},
		{why: "another top-level category", id: "recovery", siteID: "s1", name: "Recovery", key: "recovery", ok: true},
		{why: "a category under another", id: "bpc", siteID: "s1", name: "BPC-157", key: "bpc-157", parentID: "healing", ok: true},
		{why: "a level under the second", id: "bpc-liquid", siteID: "s1", name: "Liquid", key: "liquid", parentID: "bpc", ok: true},
		{why: "the same key under another parent", id: "recovery-liquid", siteID: "s1", name: "Liquid", key: "liquid", parentID: "recovery", ok: true},
		{why: "the same key at the top of another site", id: "elsewhere", siteID: "s2", name: "Healing", key: "healing", ok: true},
		{why: "the same key twice at the top", id: "healing-2", siteID: "s1", name: "HEALING", key: "healing"},
		{why: "the same key twice under one parent", id: "bpc-2", siteID: "s1", name: "bpc-157", key: "bpc-157", parentID: "healing"},
		{why: "a category under itself", id: "self", siteID: "s1", name: "Self", key: "self", parentID: "self"},
		{why: "a parent that does not exist", id: "orphan", siteID: "s1", name: "Orphan", key: "orphan", parentID: "missing"},
		{why: "a site that does not exist", id: "stray", siteID: "s9", name: "Stray", key: "stray"},
	} {
		err := insert(step.id, step.siteID, step.name, step.key, step.parentID)
		if step.ok && err != nil {
			t.Fatalf("%s: %v", step.why, err)
		}
		if !step.ok && err == nil {
			t.Fatalf("%s was accepted", step.why)
		}
	}
	if err := exec(`INSERT INTO categories (id, site_id, name, name_key, created_at, updated_at) VALUES ('nulled', 's1', NULL, 'nulled', ?, ?)`, at, at); err == nil {
		t.Fatal("a category without a name was accepted")
	}
	if err := exec(`INSERT INTO categories (id, site_id, name, name_key, created_at, updated_at) VALUES ('unkeyed', 's1', 'Unkeyed', NULL, ?, ?)`, at, at); err == nil {
		t.Fatal("a category without a key was accepted")
	}

	for _, index := range []struct {
		name   string
		unique int
	}{
		{name: "categories_site_parent_key", unique: 1},
		{name: "categories_site_parent", unique: 0},
	} {
		if got := count(`SELECT count(*) FROM pragma_index_list('categories') WHERE name = ? AND "unique" = ?`, index.name, index.unique); got != 1 {
			t.Fatalf("the index %s (unique %d) is missing", index.name, index.unique)
		}
	}

	if err := exec(`DELETE FROM categories WHERE id = 'healing'`); err != nil {
		t.Fatalf("delete a parent: %v", err)
	}
	if got := count(`SELECT count(*) FROM categories WHERE site_id = 's1'`); got != 2 {
		t.Fatalf("categories after a parent delete = %d, want the other branch's 2", got)
	}
	if err := exec(`DELETE FROM sites WHERE id = 's1'`); err != nil {
		t.Fatalf("delete the site: %v", err)
	}
	if got := count(`SELECT count(*) FROM categories`); got != 1 {
		t.Fatalf("categories after a site delete = %d, want the other site's 1", got)
	}
}

func TestCategoryTermsSchema(t *testing.T) {
	t.Parallel()

	store := storeAt(t, 43)
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
	const at = "2026-10-04T09:00:00Z"
	term := func(categoryID, siteID, taxonomy string, termID, parentTermID int) error {
		return exec(`INSERT INTO category_terms (category_id, site_id, taxonomy, term_id, parent_term_id, name, run_id, seen_at) VALUES (?, ?, ?, ?, ?, 'Healing', '', ?)`,
			categoryID, siteID, taxonomy, termID, parentTermID, at)
	}

	for _, setup := range []string{
		`INSERT INTO sites (id, name, base_url, secret_ref, status, created_at, updated_at) VALUES ('s1', 'Shop', 'https://shop', 'site:s1:wp_password', 'active', ?, ?)`,
		`INSERT INTO categories (id, site_id, name, name_key, created_at, updated_at) VALUES ('c1', 's1', 'Healing', 'healing', ?, ?)`,
		`INSERT INTO categories (id, site_id, name, name_key, parent_id, created_at, updated_at) VALUES ('c2', 's1', 'BPC-157', 'bpc-157', 'c1', ?, ?)`,
		`INSERT INTO categories (id, site_id, name, name_key, created_at, updated_at) VALUES ('c3', 's1', 'Recovery', 'recovery', ?, ?)`,
	} {
		if err := exec(setup, at, at); err != nil {
			t.Fatalf("%s: %v", setup, err)
		}
	}

	for _, step := range []struct {
		why                       string
		categoryID, siteID, taxon string
		termID, parentTermID      int
		ok                        bool
	}{
		{why: "a category term", categoryID: "c1", siteID: "s1", taxon: "category", termID: 5, ok: true},
		{why: "the same category as a product category", categoryID: "c1", siteID: "s1", taxon: "product_cat", termID: 5, ok: true},
		{why: "a term under another", categoryID: "c2", siteID: "s1", taxon: "category", termID: 6, parentTermID: 5, ok: true},
		{why: "a term another category already maps to", categoryID: "c3", siteID: "s1", taxon: "category", termID: 5, ok: true},
		{why: "a second term for one category in one taxonomy", categoryID: "c1", siteID: "s1", taxon: "category", termID: 7},
		{why: "a taxonomy Postulator does not write", categoryID: "c3", siteID: "s1", taxon: "post_tag", termID: 7},
		{why: "a term id of zero", categoryID: "c3", siteID: "s1", taxon: "product_cat", termID: 0},
		{why: "a negative term id", categoryID: "c3", siteID: "s1", taxon: "product_cat", termID: -7},
		{why: "a negative parent", categoryID: "c3", siteID: "s1", taxon: "product_cat", termID: 7, parentTermID: -1},
		{why: "a category that does not exist", categoryID: "c9", siteID: "s1", taxon: "product_cat", termID: 7},
		{why: "a site that does not exist", categoryID: "c3", siteID: "s9", taxon: "product_cat", termID: 7},
	} {
		err := term(step.categoryID, step.siteID, step.taxon, step.termID, step.parentTermID)
		if step.ok && err != nil {
			t.Fatalf("%s: %v", step.why, err)
		}
		if !step.ok && err == nil {
			t.Fatalf("%s was accepted", step.why)
		}
	}

	var parentTermID int
	var runID string
	if err := exec(`INSERT INTO category_terms (category_id, site_id, taxonomy, term_id, name, seen_at) VALUES ('c3', 's1', 'product_cat', 9, 'Recovery', ?)`, at); err != nil {
		t.Fatalf("a term stored without a parent or a run: %v", err)
	}
	if err := store.reader.QueryRowContext(t.Context(), `SELECT parent_term_id, run_id FROM category_terms WHERE category_id = 'c3' AND taxonomy = 'product_cat'`).Scan(&parentTermID, &runID); err != nil {
		t.Fatalf("read the defaults: %v", err)
	}
	if parentTermID != 0 || runID != "" {
		t.Fatalf("defaults = parent %d, run %q; want no parent and no run", parentTermID, runID)
	}

	if got := count(`SELECT count(*) FROM sqlite_schema WHERE type = 'index' AND name = 'category_terms_site_taxonomy_term'`); got != 1 {
		t.Fatalf("the lookup index by site, taxonomy and term is missing")
	}

	if err := exec(`DELETE FROM categories WHERE id = 'c1'`); err != nil {
		t.Fatalf("delete a category: %v", err)
	}
	if got := count(`SELECT count(*) FROM category_terms`); got != 2 {
		t.Fatalf("category_terms after a category delete = %d, want the other category's 2", got)
	}
	if err := exec(`DELETE FROM sites WHERE id = 's1'`); err != nil {
		t.Fatalf("delete the site: %v", err)
	}
	if got := count(`SELECT count(*) FROM category_terms`); got != 0 {
		t.Fatalf("category_terms after a site delete = %d, want none", got)
	}
}

func TestPageCategoryColumn(t *testing.T) {
	t.Parallel()

	store := openStore(t, nil)
	provider, err := store.provider()
	if err != nil {
		t.Fatalf("provider: %v", err)
	}
	if _, err = provider.DownTo(t.Context(), 40); err != nil {
		t.Fatalf("down to 40: %v", err)
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
	const at = "2026-10-04T09:00:00Z"

	for _, setup := range []string{
		`INSERT INTO sites (id, name, base_url, secret_ref, status, created_at, updated_at) VALUES ('s1', 'Shop', 'https://shop', 'site:s1:wp_password', 'active', ?, ?)`,
		`INSERT INTO pages (id, site_id, path, slug, wp_type, status, created_at, updated_at) VALUES ('pg1', 's1', '/healing/', 'healing', 'page', 'planned', ?, ?)`,
		`INSERT INTO categories (id, site_id, name, name_key, created_at, updated_at) VALUES ('c1', 's1', 'Healing', 'healing', ?, ?)`,
	} {
		if err = exec(setup, at, at); err != nil {
			t.Fatalf("%s: %v", setup, err)
		}
	}

	if _, err = provider.UpTo(t.Context(), 43); err != nil {
		t.Fatalf("up to 43: %v", err)
	}
	if got := count(`SELECT count(*) FROM pages WHERE id = 'pg1' AND category_id = ''`); got != 1 {
		t.Fatal("a page stored before the column is not filed under no category")
	}
	if got := count(`SELECT count(*) FROM pragma_index_list('pages') WHERE name = 'pages_category'`); got != 1 {
		t.Fatal("the index the category filter reads is missing")
	}
	if got := count(`SELECT count(*) FROM pragma_foreign_key_list('pages') WHERE "from" = 'category_id'`); got != 0 {
		t.Fatal("category_id carries a foreign key, which SQLite's DROP COLUMN refuses")
	}
	for _, step := range []struct {
		why   string
		value any
		ok    bool
	}{
		{why: "a page filed under a category", value: "c1", ok: true},
		{why: "a page filed under no category", value: "", ok: true},
		{why: "a page with no value at all", value: nil},
	} {
		err = exec(`UPDATE pages SET category_id = ? WHERE id = 'pg1'`, step.value)
		if step.ok && err != nil {
			t.Fatalf("%s: %v", step.why, err)
		}
		if !step.ok && err == nil {
			t.Fatalf("%s was accepted", step.why)
		}
	}

	if err = exec(`UPDATE pages SET category_id = 'c1' WHERE id = 'pg1'`); err != nil {
		t.Fatalf("file the page: %v", err)
	}
	if _, err = provider.DownTo(t.Context(), 40); err != nil {
		t.Fatalf("down to 40 with a page filed: %v", err)
	}
	if err = exec(`UPDATE pages SET category_id = ''`); err == nil {
		t.Fatal("the column survived the down migration")
	}
	if got := count(`SELECT count(*) FROM pragma_index_list('pages') WHERE name = 'pages_category'`); got != 0 {
		t.Fatal("the index survived the down migration")
	}
	if got := count(`SELECT count(*) FROM pages`); got != 1 {
		t.Fatalf("the down migration left %d pages, want 1", got)
	}
	if _, err = provider.Up(t.Context()); err != nil {
		t.Fatalf("up again: %v", err)
	}
}

func TestCategoriesAreDroppedAndADownBringsThemBackAsMigrations39To41MadeThem(t *testing.T) {
	t.Parallel()

	made := storeAt(t, 43)
	tables := []string{"categories", "category_terms", "pages"}
	want := make(map[string][]string, len(tables))
	for _, table := range tables {
		want[table] = schemaOf(t, made, table)
	}
	if len(want["categories"]) != 3 {
		t.Fatalf("migration 39 made %v, want the table and its two indexes", want["categories"])
	}
	if len(want["category_terms"]) != 2 {
		t.Fatalf("migration 40 made %v, want the table and its index", want["category_terms"])
	}
	wantColumn := columnOf(t, made, "pages", "category_id")
	if wantColumn == "" {
		t.Fatal("migration 41 made no category_id column")
	}

	store := openStore(t, nil)
	provider, err := store.provider()
	if err != nil {
		t.Fatalf("provider: %v", err)
	}
	exec := func(query string, args ...any) {
		t.Helper()
		if _, execErr := store.writer.ExecContext(t.Context(), query, args...); execErr != nil {
			t.Fatalf("%s: %v", query, execErr)
		}
	}
	count := func(query string) int {
		t.Helper()
		var n int
		if scanErr := store.writer.QueryRowContext(t.Context(), query).Scan(&n); scanErr != nil {
			t.Fatalf("%s: %v", query, scanErr)
		}
		return n
	}
	requireDropped := func(when string) {
		t.Helper()
		for _, table := range []string{"categories", "category_terms"} {
			if got := schemaOf(t, store, table); len(got) != 0 {
				t.Fatalf("%s survived %s: %v", table, when, got)
			}
		}
		if got := columnOf(t, store, "pages", "category_id"); got != "" {
			t.Fatalf("pages still carries category_id (%s) after %s", got, when)
		}
		if got := count(`SELECT count(*) FROM sqlite_schema WHERE type = 'index' AND name = 'pages_category'`); got != 0 {
			t.Fatalf("the index pages_category survived %s", when)
		}
	}
	const at = "2026-10-04T09:00:00Z"

	requireDropped("the up migration")
	exec(`INSERT INTO sites (id, name, base_url, secret_ref, status, created_at, updated_at) VALUES ('s1', 'Shop', 'https://shop', 'site:s1:wp_password', 'active', ?, ?)`, at, at)
	exec(`INSERT INTO pages (id, site_id, path, slug, wp_type, status, created_at, updated_at) VALUES ('pg1', 's1', '/healing/', 'healing', 'page', 'planned', ?, ?)`, at, at)

	if _, err = provider.DownTo(t.Context(), 43); err != nil {
		t.Fatalf("down to 43: %v", err)
	}
	for _, table := range tables {
		if got := schemaOf(t, store, table); !slices.Equal(got, want[table]) {
			t.Fatalf("the down migration made %s as\n%s\nwant it as migrations 39 to 41 made it\n%s", table, strings.Join(got, "\n"), strings.Join(want[table], "\n"))
		}
	}
	if got := columnOf(t, store, "pages", "category_id"); got != wantColumn {
		t.Fatalf("the down migration made category_id %q, want %q as migration 41 made it", got, wantColumn)
	}
	if got := count(`SELECT count(*) FROM pages WHERE id = 'pg1' AND category_id = ''`); got != 1 {
		t.Fatal("a page stored before the down is not left filed under no category")
	}
	exec(`INSERT INTO categories (id, site_id, name, name_key, created_at, updated_at) VALUES ('c1', 's1', 'Healing', 'healing', ?, ?)`, at, at)
	exec(`INSERT INTO categories (id, site_id, name, name_key, parent_id, created_at, updated_at) VALUES ('c2', 's1', 'BPC-157', 'bpc-157', 'c1', ?, ?)`, at, at)
	exec(`INSERT INTO category_terms (category_id, site_id, taxonomy, term_id, name, seen_at) VALUES ('c2', 's1', 'category', 5, 'BPC-157', ?)`, at)
	exec(`INSERT INTO pages (id, site_id, path, slug, wp_type, status, category_id, created_at, updated_at) VALUES ('pg2', 's1', '/healing/bpc-157/', 'bpc-157', 'page', 'planned', 'c2', ?, ?)`, at, at)
	exec(`UPDATE pages SET category_id = 'c1' WHERE id = 'pg1'`)

	if _, err = provider.Up(t.Context()); err != nil {
		t.Fatalf("up again with a filed page, a category tree and a term stored: %v", err)
	}
	requireDropped("the second up")
	if got := count(`SELECT count(*) FROM pages WHERE id IN ('pg1', 'pg2')`); got != 2 {
		t.Fatalf("dropping the categories left %d of the two filed pages", got)
	}
}

func TestPagesRebuildKeepsEveryRowAndItsReferences(t *testing.T) {
	t.Parallel()

	store := storeAt(t, 44)
	provider, err := store.provider()
	if err != nil {
		t.Fatalf("provider: %v", err)
	}
	exec := func(query string, args ...any) error {
		_, execErr := store.writer.ExecContext(t.Context(), query, args...)
		return execErr
	}
	must := func(query string, args ...any) {
		t.Helper()
		if execErr := exec(query, args...); execErr != nil {
			t.Fatalf("%s: %v", query, execErr)
		}
	}
	count := func(query string) int {
		t.Helper()
		var n int
		if scanErr := store.writer.QueryRowContext(t.Context(), query).Scan(&n); scanErr != nil {
			t.Fatalf("%s: %v", query, scanErr)
		}
		return n
	}
	const at = "2026-10-17T09:00:00Z"

	must(`INSERT INTO sites (id, name, base_url, secret_ref, status, created_at, updated_at) VALUES ('s1', 'Shop', 'https://shop', 'site:s1:wp_password', 'active', ?, ?)`, at, at)
	must(`INSERT INTO templates (id, scope, site_id, name, page_kind, version, spec, created_at, updated_at) VALUES ('t1', 'global', NULL, 'Hub', 'hub', 1, '{}', ?, ?)`, at, at)
	must(`INSERT INTO entities (id, site_id, name, kind, source, created_at, updated_at) VALUES ('e1', 's1', 'Peptides', 'hub', 'import', ?, ?)`, at, at)
	must(`INSERT INTO pages (id, site_id, path, slug, wp_type, wp_id, title, h1, meta_title, meta_description, canonical, status, entity_id, template_id, content_hash, wp_modified_at, last_synced_at, drift, created_at, updated_at, wp_link, wp_slug, wp_status, wp_title, wp_h1, keywords, notes, planned_path)
		VALUES ('pg1', 's1', '/peptides/', 'peptides', 'page', 7, 'Peptides', 'All peptides', 'Peptides | Shop', 'Every peptide', 'https://shop/peptides/', 'published', 'e1', 't1', 'abc', ?, ?, 1, ?, ?, 'https://shop/peptides/', 'peptides', 'publish', 'Peptides', 'All peptides', '[{"text":"peptides"}]', '[{"label":"Notes","text":"kept"}]', 'https://shop/peptides/')`,
		at, at, at, at)
	must(`INSERT INTO pages (id, site_id, path, slug, parent_page_id, wp_type, wp_id, status, created_at, updated_at) VALUES ('pg2', 's1', '/peptides/bpc-157/', 'bpc-157', 'pg1', 'product_cat', 12, 'exists', ?, ?)`, at, at)
	must(`UPDATE entities SET canonical_page_id = 'pg1' WHERE id = 'e1'`)
	must(`INSERT INTO page_links (id, site_id, from_page_id, to_page_id, to_url, anchor_text, origin, observed_at) VALUES ('l1', 's1', 'pg2', 'pg1', '/peptides/', 'peptides', 'generated', ?)`, at)
	must(`INSERT INTO template_overrides (id, template_id, scope, site_id, page_id, patch, created_at, updated_at) VALUES ('o1', 't1', 'page', NULL, 'pg1', '{}', ?, ?)`, at, at)

	snapshot := func() []string {
		t.Helper()
		var out []string
		for _, table := range []string{"pages", "page_links", "entities", "template_overrides"} {
			out = append(out, rowsOf(t, store, `SELECT '`+table+`', * FROM `+table+` ORDER BY id`)...)
		}
		return append(out, rowsOf(t, store, `SELECT name, sql FROM sqlite_schema WHERE type = 'index' AND tbl_name = 'pages' AND sql IS NOT NULL ORDER BY name`)...)
	}
	wantReferences := []string{
		"entities | canonical_page_id | SET NULL",
		"page_links | from_page_id | CASCADE",
		"page_links | to_page_id | SET NULL",
		"pages | parent_page_id | SET NULL",
		"template_overrides | page_id | CASCADE",
	}
	requireIntact := func(when string, want []string) {
		t.Helper()
		if got := snapshot(); !slices.Equal(got, want) {
			t.Fatalf("after %s the rows and indexes are\n%s\nwant\n%s", when, strings.Join(got, "\n"), strings.Join(want, "\n"))
		}
		references := rowsOf(t, store, `SELECT m.name, f."from", f.on_delete FROM sqlite_schema AS m, pragma_foreign_key_list(m.name) AS f WHERE m.type = 'table' AND f."table" = 'pages' ORDER BY m.name, f."from"`)
		if !slices.Equal(references, wantReferences) {
			t.Fatalf("after %s the references into pages are\n%s\nwant\n%s", when, strings.Join(references, "\n"), strings.Join(wantReferences, "\n"))
		}
		if dangling := rowsOf(t, store, `PRAGMA foreign_key_check`); len(dangling) != 0 {
			t.Fatalf("after %s a reference dangles: %v", when, dangling)
		}
	}

	before := snapshot()
	requireIntact("the seed", before)

	if _, err = provider.UpTo(t.Context(), 45); err != nil {
		t.Fatalf("up to 45: %v", err)
	}
	requireIntact("the rebuild", before)
	for _, step := range []struct {
		why, id, path, wpType string
		ok                    bool
	}{
		{why: "a post type the old check refused", id: "pg3", path: "/events/launch/", wpType: "event", ok: true},
		{why: "an empty post type", id: "pg4", path: "/blank/", wpType: ""},
	} {
		err = exec(`INSERT INTO pages (id, site_id, path, slug, wp_type, status, created_at, updated_at) VALUES (?, 's1', ?, 'x', ?, 'exists', ?, ?)`, step.id, step.path, step.wpType, at, at)
		if step.ok && err != nil {
			t.Fatalf("%s: %v", step.why, err)
		}
		if !step.ok && err == nil {
			t.Fatalf("%s was accepted", step.why)
		}
	}

	if _, err = provider.DownTo(t.Context(), 44); err != nil {
		t.Fatalf("down to 44 with a page of an open type: %v", err)
	}
	requireIntact("the down", before)

	if _, err = provider.Up(t.Context()); err != nil {
		t.Fatalf("up again: %v", err)
	}
	must(`DELETE FROM pages WHERE id = 'pg1'`)
	for _, check := range []struct {
		why, query string
		want       int
	}{
		{why: "the child's parent is cleared", query: `SELECT count(*) FROM pages WHERE id = 'pg2' AND parent_page_id IS NULL`, want: 1},
		{why: "the link's target is cleared", query: `SELECT count(*) FROM page_links WHERE id = 'l1' AND to_page_id IS NULL`, want: 1},
		{why: "the entity's canonical page is cleared", query: `SELECT count(*) FROM entities WHERE id = 'e1' AND canonical_page_id IS NULL`, want: 1},
		{why: "the page's override is deleted", query: `SELECT count(*) FROM template_overrides WHERE id = 'o1'`, want: 0},
	} {
		if got := count(check.query); got != check.want {
			t.Fatalf("after a page delete, not so: %s", check.why)
		}
	}
	must(`DELETE FROM pages WHERE id = 'pg2'`)
	if got := count(`SELECT count(*) FROM page_links`); got != 0 {
		t.Fatalf("page_links after its source page is deleted = %d, want none", got)
	}
}

func TestADownRestoresThePagesCheck(t *testing.T) {
	t.Parallel()

	made := storeAt(t, 44)
	tables := []string{"pages", "page_links", "sites"}
	want := make(map[string][]string, len(tables))
	for _, table := range tables {
		want[table] = schemaOf(t, made, table)
	}

	store := openStore(t, nil)
	provider, err := store.provider()
	if err != nil {
		t.Fatalf("provider: %v", err)
	}
	exec := func(query string, args ...any) error {
		_, execErr := store.writer.ExecContext(t.Context(), query, args...)
		return execErr
	}
	const at = "2026-10-17T09:00:00Z"
	page := `INSERT INTO pages (id, site_id, path, slug, wp_type, status, created_at, updated_at) VALUES (?, 's1', ?, 'x', ?, 'exists', ?, ?)`

	if err = exec(`INSERT INTO sites (id, name, base_url, secret_ref, status, created_at, updated_at) VALUES ('s1', 'Shop', 'https://shop', 'site:s1:wp_password', 'active', ?, ?)`, at, at); err != nil {
		t.Fatalf("site: %v", err)
	}
	for _, row := range [][3]string{{"pg1", "/news/", "post"}, {"pg2", "/events/launch/", "event"}} {
		if err = exec(page, row[0], row[1], row[2], at, at); err != nil {
			t.Fatalf("page %s: %v", row[0], err)
		}
	}

	if _, err = provider.DownTo(t.Context(), 44); err != nil {
		t.Fatalf("down to 44 with a page of a type the old check refuses: %v", err)
	}
	for _, table := range tables {
		if got := schemaOf(t, store, table); !slices.Equal(got, want[table]) {
			t.Fatalf("the down migrations left %s as\n%s\nwant it as migration 44 left it\n%s", table, strings.Join(got, "\n"), strings.Join(want[table], "\n"))
		}
	}
	if got := rowsOf(t, store, `SELECT id FROM pages ORDER BY id`); !slices.Equal(got, []string{"pg1"}) {
		t.Fatalf("the down kept the pages %v, want the post alone", got)
	}
	if err = exec(page, "pg3", "/events/other/", "event", at, at); err == nil {
		t.Fatal("the restored check accepted a post type it never named")
	}

	if _, err = provider.Up(t.Context()); err != nil {
		t.Fatalf("up again: %v", err)
	}
}

func TestAnEditRunIsStoredAndADownTakesItAwayAlone(t *testing.T) {
	t.Parallel()

	made := storeAt(t, 72)
	want := schemaOf(t, made, "runs")

	store := openStore(t, nil)
	provider, err := store.provider()
	if err != nil {
		t.Fatalf("provider: %v", err)
	}
	exec := func(query string, args ...any) error {
		_, execErr := store.writer.ExecContext(t.Context(), query, args...)
		return execErr
	}
	const at = "2026-10-23T09:00:00Z"
	insert := `INSERT INTO runs (id, site_id, kind, status, targets, recipe, publish_mode, created_by, deadline_at, created_at)
		VALUES (?, 's1', ?, 'completed', '[]', '[]', 'publish', 'agent', ?, ?)`
	if err = exec(`INSERT INTO sites (id, name, base_url, secret_ref, status, created_at, updated_at) VALUES ('s1', 'Shop', 'https://shop', 'site:s1:wp_password', 'active', ?, ?)`, at, at); err != nil {
		t.Fatalf("site: %v", err)
	}
	for _, row := range [][2]string{{"r1", "edit"}, {"r2", "revert"}, {"r3", "generate"}} {
		if err = exec(insert, row[0], row[1], at, at); err != nil {
			t.Fatalf("a %s run: %v", row[1], err)
		}
	}
	if err = exec(insert, "r4", "dance", at, at); err == nil {
		t.Fatal("the check accepted a kind it never named")
	}

	if _, err = provider.DownTo(t.Context(), 72); err != nil {
		t.Fatalf("down to 72: %v", err)
	}
	if got := schemaOf(t, store, "runs"); !slices.Equal(got, want) {
		t.Fatalf("the down left runs as\n%s\nwant it as migration 72 left it\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
	if got := rowsOf(t, store, `SELECT id FROM runs ORDER BY id`); !slices.Equal(got, []string{"r2", "r3"}) {
		t.Fatalf("the down kept the runs %v, want every run but the edit", got)
	}
	if _, err = provider.Up(t.Context()); err != nil {
		t.Fatalf("up again: %v", err)
	}
}

func TestTheVisibleH1IsFilledFromTheStoredHeadingsWhereTheThemePrintsNone(t *testing.T) {
	t.Parallel()

	store := storeAt(t, 74)
	renders := schemaOf(t, store, "page_renders")
	provider, err := store.provider()
	if err != nil {
		t.Fatalf("provider: %v", err)
	}
	must := func(query string, args ...any) {
		t.Helper()
		if _, execErr := store.writer.ExecContext(t.Context(), query, args...); execErr != nil {
			t.Fatalf("%s: %v", query, execErr)
		}
	}
	const at = "2026-10-24T09:00:00Z"
	must(`INSERT INTO sites (id, name, base_url, secret_ref, status, created_at, updated_at) VALUES ('s1', 'Shop', 'https://shop', 'site:s1:wp_password', 'active', ?, ?)`, at, at)
	for _, render := range []struct{ id, themeH1, headings, final string }{
		{"pg1", "absent", `[{"level":2,"text":"Dosage","generated":false},{"level":1,"text":" Lab notes ","generated":false}]`, ""},
		{"pg2", "absent", `[{"level":2,"text":"Dosage","generated":false}]`, ""},
		{"pg3", "printed", `[]`, ""},
		{"pg4", "", `[{"level":1,"text":"Assumed","generated":false}]`, ""},
		{"pg5", "absent", `[{"level":1,"text":"Elsewhere","generated":false}]`, "/cart/"},
	} {
		must(`INSERT INTO pages (id, site_id, path, slug, wp_type, status, created_at, updated_at) VALUES (?, 's1', ?, ?, 'page', 'exists', ?, ?)`,
			render.id, "/"+render.id+"/", render.id, at, at)
		must(`INSERT INTO page_renders (page_id, render_hash, content_root, headings, theme_h1, final_path, extracted_at) VALUES (?, 'h', 'found', ?, ?, ?, ?)`,
			render.id, render.headings, render.themeH1, render.final, at)
	}

	if _, err = provider.UpTo(t.Context(), 75); err != nil {
		t.Fatalf("up to 75: %v", err)
	}
	want := []string{"pg1 | Lab notes | content", "pg2 |  | none", "pg3 |  | ", "pg4 |  | ", "pg5 |  | "}
	if got := rowsOf(t, store, `SELECT page_id, h1, h1_source FROM page_renders ORDER BY page_id`); !slices.Equal(got, want) {
		t.Fatalf("renders after 0075 =\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
	if _, err = store.writer.ExecContext(t.Context(), `UPDATE page_renders SET h1_source = 'guess' WHERE page_id = 'pg1'`); err == nil {
		t.Fatal("the check accepted a source the model does not name")
	}

	if _, err = provider.DownTo(t.Context(), 74); err != nil {
		t.Fatalf("down to 74: %v", err)
	}
	if got := schemaOf(t, store, "page_renders"); !slices.Equal(got, renders) {
		t.Fatalf("the down left page_renders as\n%s\nwant it as migration 74 left it\n%s", strings.Join(got, "\n"), strings.Join(renders, "\n"))
	}
	if _, err = provider.Up(t.Context()); err != nil {
		t.Fatalf("up again: %v", err)
	}
}

func TestPageLinksRebuildCopiesOldLinksAsContextualRawLinks(t *testing.T) {
	t.Parallel()

	store := storeAt(t, 51)
	want := schemaOf(t, store, "page_links")
	provider, err := store.provider()
	if err != nil {
		t.Fatalf("provider: %v", err)
	}
	exec := func(query string, args ...any) error {
		_, execErr := store.writer.ExecContext(t.Context(), query, args...)
		return execErr
	}
	must := func(query string, args ...any) {
		t.Helper()
		if execErr := exec(query, args...); execErr != nil {
			t.Fatalf("%s: %v", query, execErr)
		}
	}
	const at = "2026-10-17T09:00:00Z"
	const oldColumns = `SELECT id, site_id, from_page_id, to_page_id, to_url, anchor_text, origin, observed_at FROM page_links ORDER BY id`

	must(`INSERT INTO sites (id, name, base_url, secret_ref, status, created_at, updated_at) VALUES ('s1', 'Shop', 'https://shop', 'site:s1:wp_password', 'active', ?, ?)`, at, at)
	must(`INSERT INTO pages (id, site_id, path, slug, wp_type, status, created_at, updated_at) VALUES ('pg1', 's1', '/a/', 'a', 'page', 'exists', ?, ?)`, at, at)
	must(`INSERT INTO pages (id, site_id, path, slug, wp_type, status, created_at, updated_at) VALUES ('pg2', 's1', '/b/', 'b', 'page', 'exists', ?, ?)`, at, at)
	must(`INSERT INTO page_links (id, site_id, from_page_id, to_page_id, to_url, anchor_text, origin, observed_at) VALUES ('l1', 's1', 'pg1', 'pg2', '/b/', 'b', 'generated', ?)`, at)
	must(`INSERT INTO page_links (id, site_id, from_page_id, to_page_id, to_url, anchor_text, origin, observed_at) VALUES ('l2', 's1', 'pg2', NULL, 'https://elsewhere/', 'elsewhere', 'observed', ?)`, at)
	old := rowsOf(t, store, oldColumns)

	if _, err = provider.UpTo(t.Context(), 52); err != nil {
		t.Fatalf("up to 52: %v", err)
	}
	if got := rowsOf(t, store, oldColumns); !slices.Equal(got, old) {
		t.Fatalf("the rebuild changed the old columns to\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(old, "\n"))
	}
	typed := []string{
		"l1 | contextual |  | raw | <nil> | 1",
		"l2 | contextual |  | raw | <nil> | 0",
	}
	if got := rowsOf(t, store, `SELECT id, type, region, source, position, ours FROM page_links ORDER BY id`); !slices.Equal(got, typed) {
		t.Fatalf("the old links were typed as\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(typed, "\n"))
	}
	indexes := []string{"page_links_from", "page_links_from_source", "page_links_site", "page_links_to"}
	if got := rowsOf(t, store, `SELECT name FROM pragma_index_list('page_links') WHERE origin = 'c' ORDER BY name`); !slices.Equal(got, indexes) {
		t.Fatalf("page_links carries the indexes %v, want %v", got, indexes)
	}
	if got := rowsOf(t, store, `SELECT name FROM pragma_index_info('page_links_from_source') ORDER BY seqno`); !slices.Equal(got, []string{"from_page_id", "source"}) {
		t.Fatalf("page_links_from_source covers %v, want the source page and the source", got)
	}

	insert := `INSERT INTO page_links (id, site_id, from_page_id, to_page_id, to_url, anchor_text, origin, type, region, source, position, ours, observed_at) VALUES (?, 's1', 'pg1', NULL, '/c/', 'c', 'observed', ?, ?, ?, ?, ?, ?)`
	for _, step := range []struct {
		why, id, kind, region, source string
		position, ours                any
		ok                            bool
	}{
		{why: "a breadcrumb read from the render", id: "l3", kind: "breadcrumb", region: "breadcrumb", source: "render", position: -1, ours: 0, ok: true},
		{why: "a contextual link we wrote", id: "l4", kind: "contextual", source: "raw", position: 2, ours: 1, ok: true},
		{why: "a listing link", id: "l5", kind: "listing", region: "listing", source: "render", ours: 0, ok: true},
		{why: "a navigation link", id: "l6", kind: "navigation", region: "header", source: "render", ours: 0, ok: true},
		{why: "a link of no known region", id: "l7", kind: "unknown", source: "render", ours: 0, ok: true},
		{why: "a type the model does not name", id: "l8", kind: "footer", source: "render", ours: 0},
		{why: "a source the model does not name", id: "l9", kind: "contextual", source: "html", ours: 0},
		{why: "an ours that is neither", id: "l10", kind: "contextual", source: "raw", ours: 2},
		{why: "no ours at all", id: "l11", kind: "contextual", source: "raw"},
	} {
		err = exec(insert, step.id, step.kind, step.region, step.source, step.position, step.ours, at)
		if step.ok && err != nil {
			t.Fatalf("%s: %v", step.why, err)
		}
		if !step.ok && err == nil {
			t.Fatalf("%s was accepted", step.why)
		}
	}

	kept := rowsOf(t, store, oldColumns)
	if _, err = provider.DownTo(t.Context(), 51); err != nil {
		t.Fatalf("down to 51: %v", err)
	}
	if got := schemaOf(t, store, "page_links"); !slices.Equal(got, want) {
		t.Fatalf("the down left page_links as\n%s\nwant it as migration 9 made it\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
	if got := rowsOf(t, store, oldColumns); !slices.Equal(got, kept) {
		t.Fatalf("the down kept the links\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(kept, "\n"))
	}
	if _, err = provider.Up(t.Context()); err != nil {
		t.Fatalf("up again: %v", err)
	}
}

func TestTheParentMethodTheFrontPageAndTheArchiveState(t *testing.T) {
	t.Parallel()

	store := storeAt(t, 54)
	renders := schemaOf(t, store, "page_renders")
	provider, err := store.provider()
	if err != nil {
		t.Fatalf("provider: %v", err)
	}
	exec := func(query string, args ...any) error {
		_, execErr := store.writer.ExecContext(t.Context(), query, args...)
		return execErr
	}
	must := func(query string, args ...any) {
		t.Helper()
		if execErr := exec(query, args...); execErr != nil {
			t.Fatalf("%s: %v", query, execErr)
		}
	}
	const at = "2026-10-18T09:00:00Z"

	must(`INSERT INTO sites (id, name, base_url, secret_ref, status, created_at, updated_at) VALUES ('s1', 'Shop', 'https://shop', 'site:s1:wp_password', 'active', ?, ?)`, at, at)
	for _, page := range []struct{ id, path, reason, technical string }{
		{"pg1", "/a/", "jsonld_breadcrumb", ""},
		{"pg2", "/b/", "aria_breadcrumb", ""},
		{"pg3", "/", "none", "front_page"},
		{"pg4", "/shop/", "url", "shop"},
	} {
		must(`INSERT INTO pages (id, site_id, path, slug, wp_type, status, parent_reason, technical_reason, created_at, updated_at) VALUES (?, 's1', ?, ?, 'page', 'exists', ?, ?, ?, ?)`,
			page.id, page.path, page.id, page.reason, page.technical, at, at)
	}
	must(`INSERT INTO page_renders (page_id, render_hash, content_root, extracted_at) VALUES ('pg1', 'h', 'not_found', ?)`, at)

	if _, err = provider.UpTo(t.Context(), 57); err != nil {
		t.Fatalf("up to 57: %v", err)
	}
	want := []string{"pg1 | href | ", "pg2 | href | ", "pg3 |  | ", "pg4 |  | shop"}
	if got := rowsOf(t, store, `SELECT id, parent_method, technical_reason FROM pages ORDER BY id`); !slices.Equal(got, want) {
		t.Fatalf("pages after 0055 and 0056 =\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
	if got := rowsOf(t, store, `SELECT count(*) FROM page_renders`); !slices.Equal(got, []string{"0"}) {
		t.Fatalf("page_renders after 0057 holds %v rows, want the cache emptied", got)
	}

	for _, step := range []struct {
		why, query string
		ok         bool
	}{
		{why: "an archive's render", query: `INSERT INTO page_renders (page_id, render_hash, content_root, extracted_at) VALUES ('pg4', 'h', 'archive', '` + at + `')`, ok: true},
		{why: "a render that landed elsewhere", query: `INSERT INTO page_renders (page_id, render_hash, content_root, final_path, extracted_at) VALUES ('pg1', 'h', 'dynamic_only', '/cart/', '` + at + `')`, ok: true},
		{why: "a root state the model does not name", query: `UPDATE page_renders SET content_root = 'missing' WHERE page_id = 'pg4'`},
		{why: "a parent found through a redirect", query: `UPDATE pages SET parent_method = 'redirect' WHERE id = 'pg1'`, ok: true},
		{why: "a parent found by name", query: `UPDATE pages SET parent_method = 'name' WHERE id = 'pg2'`, ok: true},
		{why: "a method the model does not name", query: `UPDATE pages SET parent_method = 'guess' WHERE id = 'pg1'`},
		{why: "no method at all", query: `UPDATE pages SET parent_method = NULL WHERE id = 'pg1'`},
	} {
		err = exec(step.query)
		if step.ok && err != nil {
			t.Fatalf("%s: %v", step.why, err)
		}
		if !step.ok && err == nil {
			t.Fatalf("%s was accepted", step.why)
		}
	}
	if got := rowsOf(t, store, `SELECT page_id, final_path FROM page_renders ORDER BY page_id`); !slices.Equal(got, []string{"pg1 | /cart/", "pg4 | "}) {
		t.Fatalf("final paths = %v, want the landing kept and empty by default", got)
	}

	if _, err = provider.DownTo(t.Context(), 54); err != nil {
		t.Fatalf("down to 54: %v", err)
	}
	if got := schemaOf(t, store, "page_renders"); !slices.Equal(got, renders) {
		t.Fatalf("the down left page_renders as\n%s\nwant it as migration 51 made it\n%s", strings.Join(got, "\n"), strings.Join(renders, "\n"))
	}
	if got := columnOf(t, store, "pages", "parent_method"); got != "" {
		t.Fatalf("the down kept parent_method as %q", got)
	}
	if _, err = provider.Up(t.Context()); err != nil {
		t.Fatalf("up again: %v", err)
	}
}

func TestEverySiteGetsThePresetsAndEveryPageAType(t *testing.T) {
	t.Parallel()

	store := storeAt(t, 57)
	pagesBefore := schemaOf(t, store, "pages")
	provider, err := store.provider()
	if err != nil {
		t.Fatalf("provider: %v", err)
	}
	must := func(query string, args ...any) {
		t.Helper()
		if _, execErr := store.writer.ExecContext(t.Context(), query, args...); execErr != nil {
			t.Fatalf("%s: %v", query, execErr)
		}
	}
	const at = "2026-10-18T09:00:00Z"

	for _, site := range []string{"s1", "s2"} {
		must(`INSERT INTO sites (id, name, base_url, secret_ref, status, created_at, updated_at) VALUES (?, ?, ?, ?, 'active', ?, ?)`,
			site, site, "https://"+site, "site:"+site+":wp_password", at, at)
	}
	for _, tpl := range []struct{ id, kind string }{{"tpl-hub", "hub"}, {"tpl-category", "category"}, {"tpl-comparison", "comparison"}, {"tpl-product", "product"}} {
		must(`INSERT INTO templates (id, scope, name, page_kind, version, spec, created_at, updated_at) VALUES (?, 'global', ?, ?, 1, '{}', ?, ?)`, tpl.id, tpl.id, tpl.kind, at, at)
	}
	for _, entity := range []struct{ id, kind string }{{"e-hub", "hub"}, {"e-category", "category"}, {"e-product", "product"}, {"e-topic", "topic"}, {"e-custom", "custom"}} {
		must(`INSERT INTO entities (id, site_id, name, kind, source, created_at, updated_at) VALUES (?, 's1', ?, ?, 'import', ?, ?)`, entity.id, entity.id, entity.kind, at, at)
	}
	pages := []struct{ id, wpType, technical, template, entity string }{
		{id: "pg-product", wpType: "product", entity: "e-hub"},
		{id: "pg-term", wpType: "product_cat", template: "tpl-product"},
		{id: "pg-shop", wpType: "page", technical: "shop"},
		{id: "pg-cart", wpType: "page", technical: "cart", entity: "e-hub"},
		{id: "pg-hub-template", wpType: "page", template: "tpl-hub", entity: "e-topic"},
		{id: "pg-category-template", wpType: "page", template: "tpl-category"},
		{id: "pg-comparison-template", wpType: "page", template: "tpl-comparison", entity: "e-hub"},
		{id: "pg-hub-entity", wpType: "page", entity: "e-hub"},
		{id: "pg-category-entity", wpType: "page", entity: "e-category"},
		{id: "pg-product-entity", wpType: "page", entity: "e-product"},
		{id: "pg-topic-entity", wpType: "page", entity: "e-topic"},
		{id: "pg-custom-entity", wpType: "page", entity: "e-custom"},
		{id: "pg-plain", wpType: "post"},
	}
	for _, page := range pages {
		must(`INSERT INTO pages (id, site_id, path, slug, wp_type, status, technical_reason, template_id, entity_id, created_at, updated_at) VALUES (?, 's1', ?, ?, ?, 'exists', ?, nullif(?, ''), nullif(?, ''), ?, ?)`,
			page.id, "/"+page.id+"/", page.id, page.wpType, page.technical, page.template, page.entity, at, at)
	}

	if _, err = provider.UpTo(t.Context(), 60); err != nil {
		t.Fatalf("up to 60: %v", err)
	}

	want := []string{
		"pg-cart | technical | auto | technical",
		"pg-category-entity | hub | auto | kind",
		"pg-category-template | hub | auto | kind",
		"pg-comparison-template | entity | auto | kind",
		"pg-custom-entity | entity | auto | kind",
		"pg-hub-entity | hub | auto | kind",
		"pg-hub-template | hub | auto | kind",
		"pg-plain | entity | auto | default",
		"pg-product | product | auto | product",
		"pg-product-entity | product | auto | kind",
		"pg-shop | catalog | auto | shop",
		"pg-term | store_category | auto | store_category",
		"pg-topic-entity | entity | auto | kind",
	}
	got := rowsOf(t, store, `SELECT pages.id, page_types.preset, pages.page_type_source, pages.page_type_reason
		FROM pages JOIN page_types ON page_types.id = pages.page_type_id AND page_types.site_id = pages.site_id ORDER BY pages.id`)
	if !slices.Equal(got, want) {
		t.Fatalf("the backfill =\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
	if got := rowsOf(t, store, `SELECT site_id, group_concat(name, ',') FROM (SELECT site_id, name FROM page_types ORDER BY site_id, created_at) GROUP BY site_id`); !slices.Equal(got, []string{
		"s1 | Hub,Entity,Product,Store category,Catalog,Technical", "s2 | Hub,Entity,Product,Store category,Catalog,Technical",
	}) {
		t.Fatalf("the seeded types = %v, want the six presets on every site in their order", got)
	}
	for _, typeID := range rowsOf(t, store, `SELECT id FROM page_types`) {
		if !id.Valid(typeID) {
			t.Errorf("the seeded type id %q is not a canonical v4 UUID", typeID)
		}
	}
	if got := rowsOf(t, store, `SELECT page_types.preset, count(*) FROM field_scopes JOIN page_types ON page_types.id = field_scopes.page_type_id WHERE page_types.site_id = 's1' GROUP BY page_types.preset ORDER BY page_types.preset`); !slices.Equal(got, []string{
		"entity | 3", "hub | 3", "product | 4",
	}) {
		t.Fatalf("the seeded scopes per preset = %v", got)
	}

	for _, step := range []struct {
		why, query string
		ok         bool
	}{
		{why: "a type of the client's own", query: `INSERT INTO page_types (id, site_id, name, preset, created_at, updated_at) VALUES ('t-pillar', 's1', 'Pillar', 'hub', '` + at + `', '` + at + `')`, ok: true},
		{why: "a second type of one name in any case", query: `INSERT INTO page_types (id, site_id, name, preset, created_at, updated_at) VALUES ('t-pillar-2', 's1', 'PILLAR', 'entity', '` + at + `', '` + at + `')`},
		{why: "the same name on another site", query: `INSERT INTO page_types (id, site_id, name, preset, created_at, updated_at) VALUES ('t-pillar-s2', 's2', 'Pillar', 'hub', '` + at + `', '` + at + `')`, ok: true},
		{why: "the word category as a preset", query: `INSERT INTO page_types (id, site_id, name, preset, created_at, updated_at) VALUES ('t-cat', 's1', 'Cat', 'category', '` + at + `', '` + at + `')`},
		{why: "a blank name", query: `INSERT INTO page_types (id, site_id, name, preset, created_at, updated_at) VALUES ('t-blank', 's1', '  ', 'hub', '` + at + `', '` + at + `')`},
		{why: "a manual type", query: `UPDATE pages SET page_type_id = 't-pillar', page_type_source = 'manual', page_type_reason = 'user' WHERE id = 'pg-plain'`, ok: true},
		{why: "a source the model does not name", query: `UPDATE pages SET page_type_source = 'guess' WHERE id = 'pg-plain'`},
		{why: "a reason the model does not name", query: `UPDATE pages SET page_type_reason = 'hunch' WHERE id = 'pg-plain'`},
		{why: "a scope with its policy", query: `INSERT INTO field_scopes (page_type_id, field_key, link_policy, updated_at) VALUES ('t-pillar', 'excerpt', '{"links":false}', '` + at + `')`, ok: true},
		{why: "a policy that does not say whether it links", query: `INSERT INTO field_scopes (page_type_id, field_key, link_policy, updated_at) VALUES ('t-pillar', 'title', '{}', '` + at + `')`},
		{why: "a second scope of one field", query: `INSERT INTO field_scopes (page_type_id, field_key, link_policy, updated_at) VALUES ('t-pillar', 'excerpt', '{"links":false}', '` + at + `')`},
		{why: "a provider the model does not name", query: `INSERT INTO field_scopes (page_type_id, field_key, link_policy, provider, updated_at) VALUES ('t-pillar', 'meta:x', '{"links":false}', 'acf', '` + at + `')`},
		{why: "a deleted type takes its scopes", query: `DELETE FROM page_types WHERE id = 't-pillar-s2'`, ok: true},
	} {
		_, execErr := store.writer.ExecContext(t.Context(), step.query)
		if step.ok && execErr != nil {
			t.Fatalf("%s: %v", step.why, execErr)
		}
		if !step.ok && execErr == nil {
			t.Fatalf("%s was accepted", step.why)
		}
	}

	if _, err = provider.DownTo(t.Context(), 57); err != nil {
		t.Fatalf("down to 57: %v", err)
	}
	if got := schemaOf(t, store, "pages"); !slices.Equal(got, pagesBefore) {
		t.Fatalf("the down left pages as\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(pagesBefore, "\n"))
	}
	if got := schemaOf(t, store, "page_types"); len(got) != 0 {
		t.Fatalf("the down kept page_types: %v", got)
	}
	if _, err = provider.Up(t.Context()); err != nil {
		t.Fatalf("up again: %v", err)
	}
}

func TestTheMigrationSeedsWhatTheApplicationSeeds(t *testing.T) {
	t.Parallel()

	store := storeAt(t, 59)
	provider, err := store.provider()
	if err != nil {
		t.Fatalf("provider: %v", err)
	}
	const at = "2026-10-18T09:00:00Z"
	if _, err = store.writer.ExecContext(t.Context(), `INSERT INTO sites (id, name, base_url, secret_ref, status, created_at, updated_at) VALUES ('s1', 'Shop', 'https://shop', 'site:s1:wp_password', 'active', ?, ?)`, at, at); err != nil {
		t.Fatalf("insert the site: %v", err)
	}
	if _, err = provider.UpTo(t.Context(), 60); err != nil {
		t.Fatalf("up to 60: %v", err)
	}

	types, err := NewPageTypeRepo(store).ListBySite(t.Context(), "s1")
	if err != nil {
		t.Fatalf("ListBySite types: %v", err)
	}
	presets := []pagetype.Preset{pagetype.Hub, pagetype.Entity, pagetype.Product, pagetype.StoreCategory, pagetype.Catalog, pagetype.Technical}
	if len(types) != len(presets) {
		t.Fatalf("seeded %d types, want %d", len(types), len(presets))
	}
	want := make([]field.Scope, 0, 4*len(types))
	for i := range types {
		if types[i].Preset != presets[i] || types[i].Name != presets[i].DefaultName() || types[i].TemplateID != nil {
			t.Errorf("type %d = %+v, want the %s preset by its default name", i, types[i], presets[i])
		}
		want = append(want, pagetype.DefaultScopes(types[i].Preset, types[i].ID)...)
	}
	got, err := NewFieldScopeRepo(store).ListBySite(t.Context(), "s1")
	if err != nil {
		t.Fatalf("ListBySite scopes: %v", err)
	}
	for i := range got {
		got[i].UpdatedAt = time.Time{}
	}
	byKey := func(a, b field.Scope) int {
		return strings.Compare(a.PageTypeID+string(a.Key), b.PageTypeID+string(b.Key))
	}
	slices.SortFunc(got, byKey)
	slices.SortFunc(want, byKey)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("the migration seeded\n%+v\nwant what the application seeds\n%+v", got, want)
	}
}

func TestTheNewPresetsJoinEverySiteThatHasTypesAndNoPageMoves(t *testing.T) {
	t.Parallel()

	store := storeAt(t, 75)
	provider, err := store.provider()
	if err != nil {
		t.Fatalf("provider: %v", err)
	}
	must := func(query string, args ...any) {
		t.Helper()
		if _, execErr := store.writer.ExecContext(t.Context(), query, args...); execErr != nil {
			t.Fatalf("%s: %v", query, execErr)
		}
	}
	const at = "2026-10-26T09:00:00Z"

	for _, tpl := range []string{"Hub", "Guide", "Product"} {
		must(`INSERT INTO templates (id, scope, name, version, spec, created_at, updated_at) VALUES (?, 'global', ?, 1, '{}', ?, ?)`, "tpl-"+tpl, tpl, at, at)
	}
	must(`INSERT INTO templates (id, scope, name, version, spec, created_at, updated_at) VALUES ('tpl-own', 'global', 'Own', 1, '{}', ?, ?)`, at, at)
	for _, site := range []string{"s1", "s2", "s3"} {
		must(`INSERT INTO sites (id, name, base_url, secret_ref, status, created_at, updated_at) VALUES (?, ?, ?, ?, 'active', ?, ?)`,
			site, site, "https://"+site, "site:"+site+":wp_password", at, at)
	}
	must(`UPDATE sites SET default_template_id = 'tpl-own' WHERE id = 's2'`)
	types := []struct{ id, site, name, preset string }{
		{"s1-hub", "s1", "Hub", "hub"}, {"s1-entity", "s1", "Entity", "entity"}, {"s1-technical", "s1", "Technical", "technical"},
		{"s2-hub", "s2", "Category", "hub"}, {"s2-entity", "s2", "Entity", "entity"},
		{"s2-info", "s2", "INFO", "technical"}, {"s2-info-pages", "s2", "Info pages", "catalog"},
	}
	for _, kind := range types {
		must(`INSERT INTO page_types (id, site_id, name, preset, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`, kind.id, kind.site, kind.name, kind.preset, at, at)
	}
	must(`INSERT INTO pages (id, site_id, path, slug, wp_type, status, page_type_id, page_type_source, page_type_reason, created_at, updated_at)
		VALUES ('pg-hub', 's1', '/peptides/', 'peptides', 'page', 'exists', 's1-hub', 'auto', 'listing', ?, ?)`, at, at)
	must(`INSERT INTO pages (id, site_id, path, slug, wp_type, status, page_type_id, page_type_source, page_type_reason, created_at, updated_at)
		VALUES ('pg-mine', 's1', '/about/', 'about', 'page', 'exists', 's1-entity', 'manual', 'user', ?, ?)`, at, at)
	pagesBefore := rowsOf(t, store, `SELECT id, page_type_id, page_type_source, page_type_reason FROM pages ORDER BY id`)
	typesBefore := rowsOf(t, store, `SELECT id, name, preset FROM page_types ORDER BY id`)

	if _, err = provider.UpTo(t.Context(), 76); err != nil {
		t.Fatalf("up to 76: %v", err)
	}

	added := `SELECT page_types.site_id, page_types.name, page_types.preset, coalesce(page_types.template_id, '-')
		FROM page_types WHERE page_types.id NOT IN (` + "'" + strings.Join(func() []string {
		out := make([]string, 0, len(types))
		for _, kind := range types {
			out = append(out, kind.id)
		}
		return out
	}(), "','") + "'" + `) ORDER BY page_types.site_id, page_types.created_at, page_types.preset`
	if got := rowsOf(t, store, added); !slices.Equal(got, []string{
		"s1 | Landing | landing | tpl-Hub",
		"s1 | Category | category | tpl-Hub",
		"s1 | Guide | guide | tpl-Guide",
		"s1 | Info | info | tpl-Guide",
		"s2 | Landing | landing | -",
		"s2 | Guide | guide | -",
		"s2 | Info pages 2 | info | -",
	}) {
		t.Fatalf("the added presets =\n%s", strings.Join(got, "\n"))
	}
	for _, typeID := range rowsOf(t, store, `SELECT id FROM page_types WHERE preset IN ('landing', 'category', 'guide', 'info')`) {
		if !id.Valid(typeID) {
			t.Errorf("the added type id %q is not a canonical v4 UUID", typeID)
		}
	}
	if got := rowsOf(t, store, `SELECT id, page_type_id, page_type_source, page_type_reason FROM pages ORDER BY id`); !slices.Equal(got, pagesBefore) {
		t.Fatalf("a page moved: %v, was %v", got, pagesBefore)
	}

	added76, err := NewPageTypeRepo(store).ListBySite(t.Context(), "s1")
	if err != nil {
		t.Fatalf("ListBySite types: %v", err)
	}
	want := make([]field.Scope, 0)
	newTypes := make([]string, 0)
	for i := range added76 {
		switch added76[i].Preset {
		case pagetype.Landing, pagetype.Category, pagetype.Guide, pagetype.Info:
			want = append(want, pagetype.DefaultScopes(added76[i].Preset, added76[i].ID)...)
			newTypes = append(newTypes, added76[i].ID)
		}
	}
	scopes, err := NewFieldScopeRepo(store).ListBySite(t.Context(), "s1")
	if err != nil {
		t.Fatalf("ListBySite scopes: %v", err)
	}
	got := make([]field.Scope, 0)
	for i := range scopes {
		if slices.Contains(newTypes, scopes[i].PageTypeID) {
			scopes[i].UpdatedAt = time.Time{}
			got = append(got, scopes[i])
		}
	}
	byKey := func(a, b field.Scope) int {
		return strings.Compare(a.PageTypeID+string(a.Key), b.PageTypeID+string(b.Key))
	}
	slices.SortFunc(got, byKey)
	slices.SortFunc(want, byKey)
	if len(newTypes) != 4 || !reflect.DeepEqual(got, want) {
		t.Fatalf("the migration seeded the new presets' scopes\n%+v\nwant what the application seeds\n%+v", got, want)
	}

	for _, step := range []struct {
		why, query string
		ok         bool
	}{
		{why: "a landing type", query: `INSERT INTO page_types (id, site_id, name, preset, created_at, updated_at) VALUES ('t-landing', 's3', 'Top', 'landing', '` + at + `', '` + at + `')`, ok: true},
		{why: "an unknown preset", query: `INSERT INTO page_types (id, site_id, name, preset, created_at, updated_at) VALUES ('t-pillar', 's3', 'Pillar', 'pillar', '` + at + `', '` + at + `')`},
	} {
		_, execErr := store.writer.ExecContext(t.Context(), step.query)
		if step.ok && execErr != nil {
			t.Fatalf("%s: %v", step.why, execErr)
		}
		if !step.ok && execErr == nil {
			t.Fatalf("%s was accepted", step.why)
		}
	}
	if got := rowsOf(t, store, `SELECT count(*) FROM page_types WHERE site_id = 's3' AND id <> 't-landing'`); !slices.Equal(got, []string{"0"}) {
		t.Fatalf("a site without types got %v", got)
	}

	must(`UPDATE page_map_sites SET generation = 0`)
	must(`UPDATE page_types SET preset = 'category' WHERE id = 's1-hub'`)
	if got := rowsOf(t, store, `SELECT generation FROM page_map_sites WHERE site_id = 's1'`); !slices.Equal(got, []string{"1"}) {
		t.Fatalf("a preset that moved left the map's generation at %v", got)
	}
	must(`UPDATE page_types SET preset = 'hub' WHERE id = 's1-hub'`)

	if _, err = provider.DownTo(t.Context(), 75); err != nil {
		t.Fatalf("down to 75: %v", err)
	}
	if got := rowsOf(t, store, `SELECT preset, count(*) FROM page_types GROUP BY preset ORDER BY preset`); !slices.Equal(got, []string{
		"catalog | 1", "entity | 6", "hub | 6", "technical | 2",
	}) {
		t.Fatalf("the down mapped the presets to %v", got)
	}
	if got := rowsOf(t, store, `SELECT id, name, preset FROM page_types WHERE id IN ('s1-hub','s1-entity','s1-technical','s2-hub','s2-entity','s2-info','s2-info-pages') ORDER BY id`); !slices.Equal(got, typesBefore) {
		t.Fatalf("the down changed the client's types: %v, was %v", got, typesBefore)
	}
	if got := rowsOf(t, store, `SELECT name FROM sqlite_schema WHERE type IN ('index', 'trigger') AND tbl_name = 'page_types' AND name NOT LIKE 'sqlite_%' ORDER BY name`); !slices.Equal(got, []string{
		"page_map_type_added", "page_map_type_moved", "page_map_type_removed", "page_types_site_name", "page_types_site_preset",
	}) {
		t.Fatalf("the down left the indexes and triggers %v", got)
	}
	if _, err = provider.Up(t.Context()); err != nil {
		t.Fatalf("up again: %v", err)
	}
}

func TestTheMigrationSeedsTheRulesTheApplicationSeeds(t *testing.T) {
	t.Parallel()

	store := storeAt(t, 76)
	provider, err := store.provider()
	if err != nil {
		t.Fatalf("provider: %v", err)
	}
	must := func(query string, args ...any) {
		t.Helper()
		if _, execErr := store.writer.ExecContext(t.Context(), query, args...); execErr != nil {
			t.Fatalf("%s: %v", query, execErr)
		}
	}
	const at = "2026-10-26T09:00:00Z"
	for _, site := range []string{"fresh", "upgraded", "renamed", "bare"} {
		must(`INSERT INTO sites (id, name, base_url, secret_ref, status, created_at, updated_at) VALUES (?, ?, ?, ?, 'active', ?, ?)`,
			site, site, "https://"+site, "site:"+site+":wp_password", at, at)
	}
	seed := func(site string, presets []pagetype.Preset, names map[pagetype.Preset]string) {
		for i, preset := range presets {
			name := preset.DefaultName()
			if renamed, held := names[preset]; held {
				name = renamed
			}
			must(`INSERT INTO page_types (id, site_id, name, preset, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`,
				id.New(), site, name, string(preset), "2026-10-26T09:00:0"+strconv.Itoa(i)+"Z", at)
		}
	}
	seed("fresh", pagetype.Seeded(), nil)
	seed("upgraded", pagetype.Presets(), nil)
	seed("renamed", []pagetype.Preset{
		pagetype.Hub, pagetype.Entity, pagetype.Product, pagetype.StoreCategory, pagetype.Catalog, pagetype.Technical,
		pagetype.Landing, pagetype.Guide, pagetype.Info,
	}, map[pagetype.Preset]string{pagetype.Hub: "Category"})

	if _, err = provider.UpTo(t.Context(), 77); err != nil {
		t.Fatalf("up to 77: %v", err)
	}

	typesRepo, rulesRepo := NewPageTypeRepo(store), NewPageTypeRuleRepo(store)
	for _, site := range []string{"fresh", "upgraded", "renamed"} {
		types, listErr := typesRepo.ListBySite(t.Context(), site)
		if listErr != nil {
			t.Fatalf("ListBySite types: %v", listErr)
		}
		stored, listErr := rulesRepo.ListBySite(t.Context(), site)
		if listErr != nil {
			t.Fatalf("ListBySite rules: %v", listErr)
		}
		want := pagetype.DefaultRules(pagetype.NewSet(types, nil), site, time.Time{}, func() string { return "" })
		if len(stored) != len(want) {
			t.Fatalf("%s: the migration seeded %d rules, the application seeds %d", site, len(stored), len(want))
		}
		for i := range stored {
			if !id.Valid(stored[i].ID) {
				t.Errorf("%s: the seeded rule id %q is not a canonical v4 UUID", site, stored[i].ID)
			}
			stored[i].ID, stored[i].CreatedAt, stored[i].UpdatedAt = "", time.Time{}, time.Time{}
			if !reflect.DeepEqual(stored[i], want[i]) {
				t.Errorf("%s: rule %d =\n%+v\nwant what the application seeds\n%+v", site, i+1, stored[i], want[i])
			}
		}
	}
	renamedHub := rowsOf(t, store, `SELECT id FROM page_types WHERE site_id = 'renamed' AND name = 'Category'`)
	if got := rowsOf(t, store, `SELECT page_type_id FROM page_type_rules WHERE site_id = 'renamed' AND position = 7`); !slices.Equal(got, renamedHub) {
		t.Errorf("the category rule on the renamed site gives %v, want the client's Category %v", got, renamedHub)
	}
	if got := rowsOf(t, store, `SELECT count(*) FROM page_type_rules WHERE site_id = 'bare'`); !slices.Equal(got, []string{"0"}) {
		t.Errorf("a site without types got %v rules", got)
	}

	if _, err = provider.DownTo(t.Context(), 76); err != nil {
		t.Fatalf("down to 76: %v", err)
	}
	if got := schemaOf(t, store, "page_type_rules"); len(got) != 0 {
		t.Fatalf("the down kept the rules: %v", got)
	}
	if _, err = provider.Up(t.Context()); err != nil {
		t.Fatalf("up again: %v", err)
	}
}

func TestAPagesRuleAndItsReasonsComeAndGoWithTheirMigration(t *testing.T) {
	t.Parallel()

	store := storeAt(t, 77)
	provider, err := store.provider()
	if err != nil {
		t.Fatalf("provider: %v", err)
	}
	must := func(query string, args ...any) {
		t.Helper()
		if _, execErr := store.writer.ExecContext(t.Context(), query, args...); execErr != nil {
			t.Fatalf("%s: %v", query, execErr)
		}
	}
	const at = "2026-10-26T09:00:00Z"
	must(`INSERT INTO sites (id, name, base_url, secret_ref, status, created_at, updated_at) VALUES ('s1', 'Shop', 'https://shop', 'site:s1:wp_password', 'active', ?, ?)`, at, at)
	must(`INSERT INTO page_types (id, site_id, name, preset, created_at, updated_at) VALUES ('t-guide', 's1', 'Guide', 'guide', ?, ?)`, at, at)
	for _, page := range []struct{ id, source, reason string }{{"pg-user", "manual", "user"}, {"pg-listing", "auto", "listing"}, {"pg-rule", "auto", "default"}} {
		must(`INSERT INTO pages (id, site_id, path, slug, wp_type, status, page_type_id, page_type_source, page_type_reason, created_at, updated_at)
			VALUES (?, 's1', ?, ?, 'page', 'exists', 't-guide', ?, ?, ?, ?)`, page.id, "/"+page.id+"/", page.id, page.source, page.reason, at, at)
	}
	if got := rowsOf(t, store, `SELECT sql FROM sqlite_schema WHERE type = 'index' AND tbl_name = 'pages' AND sql IS NOT NULL ORDER BY name`); len(got) == 0 {
		t.Fatal("the pages table carries no index to keep")
	}
	indexes := rowsOf(t, store, `SELECT name FROM sqlite_schema WHERE type IN ('index', 'trigger') AND tbl_name = 'pages' ORDER BY name`)

	if _, err = provider.UpTo(t.Context(), 78); err != nil {
		t.Fatalf("up to 78: %v", err)
	}
	must(`UPDATE pages SET page_type_reason = 'rule', page_type_rule_id = 'rule-10' WHERE id = 'pg-rule'`)
	must(`UPDATE pages SET page_type_reason = 'unmatched' WHERE id = 'pg-listing'`)
	if _, execErr := store.writer.ExecContext(t.Context(), `UPDATE pages SET page_type_reason = 'hunch' WHERE id = 'pg-user'`); execErr == nil {
		t.Fatal("a reason the model does not name was accepted")
	}
	if got := rowsOf(t, store, `SELECT id, page_type_reason, page_type_rule_id FROM pages ORDER BY id`); !slices.Equal(got, []string{
		"pg-listing | unmatched | ", "pg-rule | rule | rule-10", "pg-user | user | ",
	}) {
		t.Fatalf("the pages after 78 = %v", got)
	}
	if got := rowsOf(t, store, `SELECT name FROM sqlite_schema WHERE type IN ('index', 'trigger') AND tbl_name = 'pages' ORDER BY name`); !slices.Equal(got, indexes) {
		t.Fatalf("78 changed the pages' indexes and triggers to %v, was %v", got, indexes)
	}

	if _, err = provider.DownTo(t.Context(), 77); err != nil {
		t.Fatalf("down to 77: %v", err)
	}
	if got := rowsOf(t, store, `SELECT id, page_type_source, page_type_reason FROM pages ORDER BY id`); !slices.Equal(got, []string{
		"pg-listing | auto | default", "pg-rule | auto | default", "pg-user | manual | user",
	}) {
		t.Fatalf("the down left the pages as %v, want a rule's reason taken back to the default", got)
	}
	if got := columnOf(t, store, "pages", "page_type_rule_id"); got != "" {
		t.Fatalf("the down kept page_type_rule_id: %q", got)
	}
	if _, err = provider.Up(t.Context()); err != nil {
		t.Fatalf("up again: %v", err)
	}
}

func TestTheLinkRulesMoveIntoTheScopesAndThePoliciesGo(t *testing.T) {
	t.Parallel()

	store := storeAt(t, 59)
	provider, err := store.provider()
	if err != nil {
		t.Fatalf("provider: %v", err)
	}
	must := func(query string, args ...any) {
		t.Helper()
		if _, execErr := store.writer.ExecContext(t.Context(), query, args...); execErr != nil {
			t.Fatalf("%s: %v", query, execErr)
		}
	}
	text := func(query string, args ...any) sql.NullString {
		t.Helper()
		var out sql.NullString
		if scanErr := store.reader.QueryRowContext(t.Context(), query, args...).Scan(&out); scanErr != nil {
			t.Fatalf("%s: %v", query, scanErr)
		}
		return out
	}
	const at = "2026-10-18T09:00:00Z"
	const hubRules = `{"upDepth":1,"downLinks":true,"siblingMinWeight":0.5,"maxLinks":20,"maxPerTarget":1,"parentLinkWithinParagraphs":2,"childrenSection":true}`

	for _, site := range []string{"s1", "s2"} {
		must(`INSERT INTO sites (id, name, base_url, secret_ref, status, created_at, updated_at) VALUES (?, ?, ?, ?, 'active', ?, ?)`,
			site, site, "https://"+site, "site:"+site+":wp_password", at, at)
	}
	if _, err = provider.UpTo(t.Context(), 61); err != nil {
		t.Fatalf("up to 61: %v", err)
	}
	must(`INSERT INTO link_policies (id, scope, site_id, name, rules, forbid_external, forbid_self, anchor_strategy, created_at, updated_at) VALUES ('p-default', 'global', NULL, 'Default', '{}', 1, 1, 'prefer_user', ?, ?)`, at, at)
	must(`INSERT INTO link_policies (id, scope, site_id, name, rules, forbid_external, forbid_self, anchor_strategy, created_at, updated_at) VALUES ('p-loose', 'site', 's1', 'Loose', '{}', 0, 1, 'rotate', ?, ?)`, at, at)
	must(`UPDATE sites SET default_link_policy_id = 'p-loose' WHERE id = 's1'`)

	seeds := template.Seed()
	hubAt := slices.IndexFunc(seeds, func(seed template.Template) bool { return seed.Name == "Hub" })
	if hubAt < 0 {
		t.Fatal("the Hub built-in is not seeded")
	}
	hub := seeds[hubAt]
	seedSpec, err := json.Marshal(hub.Spec)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	must(`INSERT INTO templates (id, scope, name, page_kind, version, spec, created_at, updated_at) VALUES ('tpl-hub', 'global', 'Hub', 'hub', 1, json_set(?, '$.linkRules', json(?)), ?, ?)`, string(seedSpec), hubRules, at, at)
	must(`INSERT INTO template_overrides (id, template_id, scope, site_id, page_id, patch, created_at, updated_at) VALUES ('o1', 'tpl-hub', 'site', 's1', NULL, '{"tone":"warm","linkRules":{"maxLinks":5}}', ?, ?)`, at, at)

	if _, err = provider.UpTo(t.Context(), 62); err != nil {
		t.Fatalf("up to 62: %v", err)
	}

	cases := []struct {
		site, preset, field string
		want                map[string]any
	}{
		{site: "s1", preset: "hub", field: "body", want: map[string]any{"links": true, "forbidExternal": false, "forbidSelf": true, "anchorStrategy": "rotate"}},
		{site: "s1", preset: "product", field: "product_description", want: map[string]any{"links": true, "forbidExternal": false, "forbidSelf": true, "anchorStrategy": "rotate"}},
		{site: "s1", preset: "hub", field: "seo_title", want: map[string]any{"links": false, "forbidExternal": false, "forbidSelf": false, "anchorStrategy": "prefer_user"}},
		{site: "s2", preset: "entity", field: "body", want: map[string]any{"links": true, "forbidExternal": true, "forbidSelf": true, "anchorStrategy": "prefer_user"}},
	}
	for _, tc := range cases {
		raw := text(`SELECT field_scopes.link_policy FROM field_scopes JOIN page_types ON page_types.id = field_scopes.page_type_id WHERE page_types.site_id = ? AND page_types.preset = ? AND field_scopes.field_key = ?`, tc.site, tc.preset, tc.field)
		var got map[string]any
		if decodeErr := json.Unmarshal([]byte(raw.String), &got); decodeErr != nil {
			t.Fatalf("decode %s: %v", raw.String, decodeErr)
		}
		for key, want := range tc.want {
			if got[key] != want {
				t.Errorf("%s %s %s: %s = %v, want %v (the site's chosen policy, else Default, carried once)", tc.site, tc.preset, tc.field, key, got[key], want)
			}
		}
	}

	var stored, seeded map[string]any
	if err = json.Unmarshal([]byte(text(`SELECT spec FROM templates WHERE id = 'tpl-hub'`).String), &stored); err != nil {
		t.Fatalf("decode the stored spec: %v", err)
	}
	if err = json.Unmarshal(seedSpec, &seeded); err != nil {
		t.Fatalf("decode the seed spec: %v", err)
	}
	if !reflect.DeepEqual(stored, seeded) {
		t.Errorf("a built-in stored before 2a = %v, want the new seed exactly", stored)
	}
	patch := text(`SELECT patch FROM template_overrides WHERE id = 'o1'`)
	resolved, err := template.Resolve(hub.Spec, json.RawMessage(patch.String), nil)
	if err != nil || resolved.Tone != "warm" {
		t.Errorf("a stored override after 0062 = %s resolves to %+v, %v; want its tone kept and its link rules gone", patch.String, resolved.Tone, err)
	}
	for _, gone := range []string{"link_policies"} {
		if name := text(`SELECT coalesce((SELECT name FROM sqlite_schema WHERE type = 'table' AND name = ?), '')`, gone); name.String != "" {
			t.Errorf("%s survived 0062", gone)
		}
	}
	if sites := strings.Join(schemaOf(t, store, "sites"), "\n"); !strings.Contains(sites, "default_template_id") || strings.Contains(sites, "default_link_policy_id") ||
		!strings.Contains(sites, "index sites_name") || !strings.Contains(sites, "index sites_created_at") {
		t.Errorf("sites after 0062 = %s, want the template default, both indexes and no policy column", sites)
	}
	must(`DELETE FROM sites WHERE id = 's1'`)
	if n := text(`SELECT count(*) FROM page_types WHERE site_id = 's1'`); n.String != "0" {
		t.Errorf("page types of a deleted site = %s, want the cascade the rebuilt sites table still drives", n.String)
	}

	if _, err = provider.DownTo(t.Context(), 61); err != nil {
		t.Fatalf("down to 61: %v", err)
	}
	if policy := text(`SELECT id FROM link_policies WHERE scope = 'global' AND name = 'Default'`); !id.Valid(policy.String) {
		t.Errorf("the down's Default policy id = %q, want a v4 UUID", policy.String)
	}
	if chosen := text(`SELECT default_link_policy_id FROM sites WHERE id = 's2'`); chosen.Valid {
		t.Errorf("the down's site policy = %v, want none", chosen)
	}
	if depth := text(`SELECT json_extract(spec, '$.linkRules.maxLinks') FROM templates WHERE id = 'tpl-hub'`); depth.String != "20" {
		t.Errorf("the down's Hub maxLinks = %q, want the built-in's 20 back", depth.String)
	}
}

func TestTheKindsBecomeTypesAndADownBringsThemBack(t *testing.T) {
	t.Parallel()

	store := storeAt(t, 62)
	provider, err := store.provider()
	if err != nil {
		t.Fatalf("provider: %v", err)
	}
	must := func(query string, args ...any) {
		t.Helper()
		if _, execErr := store.writer.ExecContext(t.Context(), query, args...); execErr != nil {
			t.Fatalf("%s: %v", query, execErr)
		}
	}
	text := func(query string) string {
		t.Helper()
		var out string
		if scanErr := store.reader.QueryRowContext(t.Context(), query).Scan(&out); scanErr != nil {
			t.Fatalf("%s: %v", query, scanErr)
		}
		return out
	}
	const at = "2026-10-18T09:00:00Z"

	for _, setup := range []string{
		`INSERT INTO sites (id, name, base_url, secret_ref, status, created_at, updated_at) VALUES ('s1', 'Shop', 'https://shop', 'site:s1:wp_password', 'active', ?, ?)`,
		`INSERT INTO page_types (id, site_id, name, preset, created_at, updated_at) VALUES ('t-product', 's1', 'Product', 'product', ?, ?)`,
	} {
		must(setup, at, at)
	}
	must(`INSERT INTO pages (id, site_id, path, slug, wp_type, status, page_type_id, page_type_source, page_type_reason, created_at, updated_at) VALUES ('pg1', 's1', '/bpc/', 'bpc', 'product', 'exists', 't-product', 'auto', 'product', ?, ?)`, at, at)
	must(`INSERT INTO entities (id, site_id, name, kind, canonical_page_id, source, created_at, updated_at) VALUES ('e-product', 's1', 'BPC', 'hub', 'pg1', 'import', ?, ?)`, at, at)
	must(`INSERT INTO entities (id, site_id, name, kind, source, created_at, updated_at) VALUES ('e-loose', 's1', 'Peptides', 'hub', 'import', ?, ?)`, at, at)
	for _, tpl := range []struct{ id, name, kind string }{{"tpl-hub", "Hub", "hub"}, {"tpl-mine", "Mine", "comparison"}} {
		must(`INSERT INTO templates (id, scope, name, page_kind, version, spec, created_at, updated_at) VALUES (?, 'global', ?, ?, 1, '{}', ?, ?)`, tpl.id, tpl.name, tpl.kind, at, at)
	}
	must(`INSERT INTO import_mappings (id, site_id, name, columns, options, created_at, updated_at) VALUES ('m-both', 's1', 'Both', '{"entity_kind":"Entity type","page_kind":"Kind","path":"URL"}', '{"rowType":"kind"}', ?, ?)`, at, at)
	must(`INSERT INTO import_mappings (id, site_id, name, columns, options, created_at, updated_at) VALUES ('m-page', 's1', 'Page', '{"page_kind":"Kind"}', '{"rowType":"products"}', ?, ?)`, at, at)
	entitiesAt62 := schemaOf(t, store, "entities")

	if _, err = provider.UpTo(t.Context(), 63); err != nil {
		t.Fatalf("up to 63: %v", err)
	}
	if got := columnOf(t, store, "entities", "kind"); got != "" {
		t.Errorf("entities.kind after 0063 = %q, want gone", got)
	}
	if got := columnOf(t, store, "templates", "page_kind"); got != "" {
		t.Errorf("templates.page_kind after 0063 = %q, want gone", got)
	}
	mappings := []struct{ id, columns, options string }{
		{id: "m-both", columns: `{"path":"URL","page_type":"Entity type"}`, options: `{"rowType":"type"}`},
		{id: "m-page", columns: `{"page_type":"Kind"}`, options: `{"rowType":"products"}`},
	}
	for _, want := range mappings {
		var columns, options map[string]any
		if err = json.Unmarshal([]byte(text(`SELECT columns FROM import_mappings WHERE id = '`+want.id+`'`)), &columns); err != nil {
			t.Fatalf("decode the columns of %s: %v", want.id, err)
		}
		if err = json.Unmarshal([]byte(text(`SELECT options FROM import_mappings WHERE id = '`+want.id+`'`)), &options); err != nil {
			t.Fatalf("decode the options of %s: %v", want.id, err)
		}
		var wantColumns, wantOptions map[string]any
		if err = json.Unmarshal([]byte(want.columns), &wantColumns); err != nil {
			t.Fatalf("decode %s: %v", want.columns, err)
		}
		if err = json.Unmarshal([]byte(want.options), &wantOptions); err != nil {
			t.Fatalf("decode %s: %v", want.options, err)
		}
		if !reflect.DeepEqual(columns, wantColumns) || !reflect.DeepEqual(options, wantOptions) {
			t.Errorf("mapping %s after 0063 = %v %v, want %v %v (an entity column wins over a page-kind column)", want.id, columns, options, wantColumns, wantOptions)
		}
	}

	if _, err = provider.DownTo(t.Context(), 62); err != nil {
		t.Fatalf("down to 62: %v", err)
	}
	if got := schemaOf(t, store, "entities"); !slices.Equal(got, entitiesAt62) {
		t.Errorf("the down left entities as\n%s\nwant it as 62 had it\n%s", strings.Join(got, "\n"), strings.Join(entitiesAt62, "\n"))
	}
	for query, want := range map[string]string{
		`SELECT kind FROM entities WHERE id = 'e-product'`:                                       "product",
		`SELECT kind FROM entities WHERE id = 'e-loose'`:                                         "topic",
		`SELECT page_kind FROM templates WHERE id = 'tpl-hub'`:                                   "hub",
		`SELECT page_kind FROM templates WHERE id = 'tpl-mine'`:                                  "guide",
		`SELECT json_extract(columns, '$.entity_kind') FROM import_mappings WHERE id = 'm-both'`: "Entity type",
		`SELECT json_extract(options, '$.rowType') FROM import_mappings WHERE id = 'm-both'`:     "kind",
	} {
		if got := text(query); got != want {
			t.Errorf("%s after the down = %q, want %q", query, got, want)
		}
	}
}

func typesBeforeTheBackfill(t *testing.T) *Store {
	t.Helper()

	store := storeAt(t, 65)
	const at = "2026-10-20T09:00:00Z"
	for _, setup := range []struct {
		query string
		args  []any
	}{
		{query: `INSERT INTO sites (id, name, base_url, secret_ref, status, created_at, updated_at) VALUES ('s1', 'Shop', 'https://shop', 'site:s1:wp_password', 'active', ?, ?)`},
		{query: `INSERT INTO sites (id, name, base_url, secret_ref, status, created_at, updated_at) VALUES ('s2', 'Kept', 'https://kept', 'site:s2:wp_password', 'active', ?, ?)`},
		{query: `INSERT INTO templates (id, scope, name, version, spec, created_at, updated_at) VALUES ('tpl-hub', 'global', 'Hub', 1, '{}', ?, ?)`},
		{query: `INSERT INTO templates (id, scope, name, version, spec, created_at, updated_at) VALUES ('tpl-guide', 'global', 'Guide', 1, '{}', ?, ?)`},
		{query: `INSERT INTO templates (id, scope, name, version, spec, created_at, updated_at) VALUES ('tpl-product', 'global', 'Product', 1, '{}', ?, ?)`},
		{query: `INSERT INTO templates (id, scope, site_id, name, version, spec, created_at, updated_at) VALUES ('tpl-site-guide', 'site', 's1', 'Guide', 1, '{}', ?, ?)`},
		{query: `INSERT INTO templates (id, scope, name, version, spec, created_at, updated_at) VALUES ('tpl-mine', 'global', 'Mine', 1, '{}', ?, ?)`},
		{query: `UPDATE sites SET default_template_id = 'tpl-mine', updated_at = ? WHERE id = 's2' AND created_at = ?`},
	} {
		if _, err := store.writer.ExecContext(t.Context(), setup.query, at, at); err != nil {
			t.Fatalf("%s: %v", setup.query, err)
		}
	}
	for _, kind := range []struct{ id, site, name, preset, template string }{
		{id: "t-hub", site: "s1", name: "Hub", preset: "hub"},
		{id: "t-entity", site: "s1", name: "Entity", preset: "entity"},
		{id: "t-product", site: "s1", name: "Product", preset: "product"},
		{id: "t-store", site: "s1", name: "Store category", preset: "store_category"},
		{id: "t-catalog", site: "s1", name: "Catalog", preset: "catalog"},
		{id: "t-technical", site: "s1", name: "Technical", preset: "technical"},
		{id: "t-chosen", site: "s1", name: "Pillar", preset: "hub", template: "tpl-mine"},
		{id: "t-kept", site: "s2", name: "Hub", preset: "hub"},
	} {
		var chosen any
		if kind.template != "" {
			chosen = kind.template
		}
		if _, err := store.writer.ExecContext(t.Context(),
			`INSERT INTO page_types (id, site_id, name, preset, template_id, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)`,
			kind.id, kind.site, kind.name, kind.preset, chosen, at, at); err != nil {
			t.Fatalf("insert the type %s: %v", kind.id, err)
		}
	}
	return store
}

func templatesOfTheTypes(t *testing.T, store *Store) map[string]string {
	t.Helper()

	out := map[string]string{}
	for _, row := range rowsOf(t, store, `SELECT id, coalesce(template_id, '') FROM page_types ORDER BY id`) {
		typeID, templateID, _ := strings.Cut(row, " | ")
		out[typeID] = templateID
	}
	return out
}

func TestTheBackfillFillsOnlyATypeWithoutATemplate(t *testing.T) {
	t.Parallel()

	store := typesBeforeTheBackfill(t)
	provider, err := store.provider()
	if err != nil {
		t.Fatalf("provider: %v", err)
	}
	if _, err = provider.UpTo(t.Context(), 66); err != nil {
		t.Fatalf("up to 66: %v", err)
	}

	want := map[string]string{
		"t-hub": "tpl-hub", "t-entity": "tpl-guide", "t-product": "tpl-product",
		"t-store": "", "t-catalog": "", "t-technical": "", "t-chosen": "tpl-mine", "t-kept": "",
	}
	if got := templatesOfTheTypes(t, store); !reflect.DeepEqual(got, want) {
		t.Errorf("the types' templates after 0066 = %v, want %v", got, want)
	}

	if _, err = provider.DownTo(t.Context(), 65); err != nil {
		t.Fatalf("down to 65: %v", err)
	}
	want["t-hub"], want["t-entity"], want["t-product"] = "", "", ""
	if got := templatesOfTheTypes(t, store); !reflect.DeepEqual(got, want) {
		t.Errorf("the types' templates after the down = %v, want %v", got, want)
	}
}

func TestTheBackfillLeavesASiteWithADefaultTemplate(t *testing.T) {
	t.Parallel()

	store := typesBeforeTheBackfill(t)
	provider, err := store.provider()
	if err != nil {
		t.Fatalf("provider: %v", err)
	}
	if _, err = provider.UpTo(t.Context(), 66); err != nil {
		t.Fatalf("up to 66: %v", err)
	}
	if got := templatesOfTheTypes(t, store)["t-kept"]; got != "" {
		t.Errorf("a type of a site whose default is chosen took %q, which would win over that default", got)
	}
}

func TestAStoredSpecWithSpecificationsStillDecodes(t *testing.T) {
	t.Parallel()

	var shipped template.TemplateSpec
	for _, seeded := range template.Seed() {
		if seeded.Name == "Product" {
			shipped = seeded.Spec
		}
	}
	encoded, err := json.Marshal(shipped)
	if err != nil {
		t.Fatalf("encode the product spec: %v", err)
	}
	var document map[string]any
	if err = json.Unmarshal(encoded, &document); err != nil {
		t.Fatalf("decode the product spec: %v", err)
	}
	product, ok := document["product"].(map[string]any)
	if !ok {
		t.Fatalf("the shipped product template carries no product block: %s", encoded)
	}
	product["specifications"] = []map[string]string{{"name": "Form", "intent": "As the notes say"}}
	stored, err := json.Marshal(document)
	if err != nil {
		t.Fatalf("encode the stored spec: %v", err)
	}
	const patch = `{"product":{"specifications":[{"name":"Size","intent":"As the notes say"}]},"tone":"plain"}`

	store := storeAt(t, 66)
	const at = "2026-10-20T09:00:00Z"
	for _, setup := range []struct {
		query string
		args  []any
	}{
		{query: `INSERT INTO sites (id, name, base_url, secret_ref, status, created_at, updated_at) VALUES ('s1', 'Shop', 'https://shop', 'site:s1:wp_password', 'active', ?, ?)`, args: []any{at, at}},
		{query: `INSERT INTO templates (id, scope, name, version, spec, created_at, updated_at) VALUES ('tpl-product', 'global', 'Product', 1, ?, ?, ?)`, args: []any{string(stored), at, at}},
		{query: `INSERT INTO template_overrides (id, template_id, scope, site_id, patch, created_at, updated_at) VALUES ('o1', 'tpl-product', 'site', 's1', ?, ?, ?)`, args: []any{patch, at, at}},
	} {
		if _, err = store.writer.ExecContext(t.Context(), setup.query, setup.args...); err != nil {
			t.Fatalf("%s: %v", setup.query, err)
		}
	}
	if _, err = template.Resolve(shipped, json.RawMessage(patch), nil); err == nil {
		t.Fatal("the strict decode took the retired key before 0067, so the test proves nothing")
	}

	provider, err := store.provider()
	if err != nil {
		t.Fatalf("provider: %v", err)
	}
	if _, err = provider.UpTo(t.Context(), 67); err != nil {
		t.Fatalf("up to 67: %v", err)
	}
	var spec, override string
	if err = store.writer.QueryRowContext(t.Context(), `SELECT spec FROM templates WHERE id = 'tpl-product'`).Scan(&spec); err != nil {
		t.Fatalf("read the spec: %v", err)
	}
	if err = store.writer.QueryRowContext(t.Context(), `SELECT patch FROM template_overrides WHERE id = 'o1'`).Scan(&override); err != nil {
		t.Fatalf("read the override: %v", err)
	}
	if strings.Contains(spec, "specifications") || strings.Contains(override, "specifications") {
		t.Fatalf("0067 left the retired key: %s / %s", spec, override)
	}
	var migrated template.TemplateSpec
	if err = json.Unmarshal([]byte(spec), &migrated); err != nil {
		t.Fatalf("decode the migrated spec: %v", err)
	}
	resolved, err := template.Resolve(migrated, json.RawMessage(override), nil)
	if err != nil {
		t.Fatalf("the migrated spec and override do not resolve: %v", err)
	}
	if resolved.Tone != "plain" || resolved.Product == nil || !resolved.Product.ShortDescription.Enabled {
		t.Errorf("the resolved spec lost what it held beside the retired key: %+v", resolved)
	}
}

func TestThePagesCopiesOfTheSiteAreDroppedAndADownBringsThemBackAsMigration45MadeThem(t *testing.T) {
	t.Parallel()

	copies := []string{"canonical", "content_hash", "drift", "wp_link", "wp_slug", "wp_status", "wp_title", "wp_h1"}
	made := storeAt(t, 67)
	want := make(map[string]string, len(copies))
	for _, column := range copies {
		if want[column] = columnOf(t, made, "pages", column); want[column] == "" {
			t.Fatalf("migration 67 holds no pages.%s", column)
		}
	}

	store := storeAt(t, 67)
	provider, err := store.provider()
	if err != nil {
		t.Fatalf("provider: %v", err)
	}
	must := func(query string, args ...any) {
		t.Helper()
		if _, execErr := store.writer.ExecContext(t.Context(), query, args...); execErr != nil {
			t.Fatalf("%s: %v", query, execErr)
		}
	}
	const (
		at       = "2026-10-17T09:00:00Z"
		synced   = "2026-10-18T09:00:00Z"
		observed = "2026-10-19T09:00:00Z"
		title    = "Peptides für Labore"
	)
	must(`INSERT INTO sites (id, name, base_url, secret_ref, status, created_at, updated_at) VALUES ('s1', 'Shop', 'https://shop', 'site:s1:wp_password', 'active', ?, ?)`, at, at)
	page := `INSERT INTO pages (id, site_id, path, slug, wp_type, wp_id, title, status, content_hash, drift, last_synced_at, wp_link, wp_slug, wp_status, wp_title, wp_h1, created_at, updated_at)
		VALUES (?, 's1', ?, ?, ?, ?, ?, 'published', ?, ?, ?, ?, ?, 'publish', ?, ?, ?, ?)`
	must(page, "pg1", "/peptides/", "peptides", "page", 1, "Peptides", "abc", 1, synced, "https://shop/peptides/", "peptides", title, "All peptides", at, at)
	must(page, "pg2", "/bpc-157/", "bpc-157", "page", 2, "BPC-157", "", 0, nil, "https://shop/bpc-157/", "bpc-157", "An old copy", "", at, at)
	must(page, "pg3", "/blank/", "blank", "page", 3, "Blank", "", 0, nil, "", "", "", "", at, at)
	must(page, "pg4", "/product-category/peptides/", "peptides", "product_cat", 4, "Peptides", "", 0, nil, "", "", "Peptides", "Peptides", at, at)
	must(`INSERT INTO page_fields (page_id, field_key, current_value, current_hash, base_value, base_hash, observed_at) VALUES ('pg2', 'title', 'Today''s title', ?, 'Today''s title', ?, ?)`,
		field.Hash("Today's title"), field.Hash("Today's title"), observed)

	if _, err = provider.UpTo(t.Context(), 68); err != nil {
		t.Fatalf("up to 68: %v", err)
	}
	for _, column := range copies {
		if got := columnOf(t, store, "pages", column); got != "" {
			t.Fatalf("pages still carries %s (%s) after the up migration", column, got)
		}
	}
	wantFields := []string{
		"pg1 | h1 | All peptides | " + field.Hash("All peptides") + " | All peptides | " + field.Hash("All peptides") + " | " + synced + " | <nil>",
		"pg1 | title | " + title + " | " + field.Hash(title) + " | " + title + " | " + field.Hash(title) + " | " + synced + " | <nil>",
		"pg2 | title | Today's title | " + field.Hash("Today's title") + " | Today's title | " + field.Hash("Today's title") + " | " + observed + " | <nil>",
		"pg4 | title | Peptides | " + field.Hash("Peptides") + " | Peptides | " + field.Hash("Peptides") + " | " + at + " | <nil>",
	}
	if got := rowsOf(t, store, `SELECT page_id, field_key, current_value, current_hash, base_value, base_hash, observed_at, written_run_id FROM page_fields ORDER BY page_id, field_key`); !slices.Equal(got, wantFields) {
		t.Fatalf("the backfilled fields are\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(wantFields, "\n"))
	}

	must(`INSERT INTO page_identity (page_id, url, status, wp_type, observed_at) VALUES ('pg1', 'https://shop/peptides/', 'draft', 'page', ?)`, observed)
	must(`UPDATE page_fields SET written_run_id = 'r1' WHERE page_id = 'pg1' AND field_key = 'title'`)
	body := `INSERT INTO page_bodies (page_id, side, raw, hash, kind, observed_at) VALUES (?, ?, ?, ?, 'classic', ?)`
	must(body, "pg1", "base", "<p>Ours.</p>", field.Hash("<p>Ours.</p>"), observed)
	must(body, "pg1", "current", "<p>Edited.</p>", field.Hash("<p>Edited.</p>"), observed)
	must(body, "pg2", "base", "<p>Theirs.</p>", field.Hash("<p>Theirs.</p>"), observed)
	must(body, "pg2", "current", "<p>Theirs, edited.</p>", field.Hash("<p>Theirs, edited.</p>"), observed)

	if _, err = provider.DownTo(t.Context(), 67); err != nil {
		t.Fatalf("down to 67: %v", err)
	}
	for _, column := range copies {
		if got := columnOf(t, store, "pages", column); got != want[column] {
			t.Fatalf("the down migration made pages.%s %q, want %q as migration 45 made it", column, got, want[column])
		}
	}
	wantPages := []string{
		"pg1 | " + title + " | All peptides | draft | https://shop/peptides/ | " + field.Hash("<p>Ours.</p>") + " | 1 |  | ",
		"pg2 | Today's title |  |  |  |  | 0 |  | ",
		"pg3 |  |  |  |  |  | 0 |  | ",
		"pg4 | Peptides |  |  |  |  | 0 |  | ",
	}
	if got := rowsOf(t, store, `SELECT id, wp_title, wp_h1, wp_status, wp_link, content_hash, drift, wp_slug, canonical FROM pages ORDER BY id`); !slices.Equal(got, wantPages) {
		t.Fatalf("the down refilled the pages as\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(wantPages, "\n"))
	}
	if _, execErr := store.writer.ExecContext(t.Context(), `UPDATE pages SET drift = 2 WHERE id = 'pg1'`); execErr == nil {
		t.Fatal("the restored drift check accepted a value that is neither 0 nor 1")
	}

	if _, err = provider.Up(t.Context()); err != nil {
		t.Fatalf("up again: %v", err)
	}
	if got := rowsOf(t, store, `SELECT count(*) FROM pages`); !slices.Equal(got, []string{"4"}) {
		t.Fatalf("the second up left %v pages, want all four", got)
	}
}

func TestALinkKeepsTheBlockItWasPlacedInAndADownDropsIt(t *testing.T) {
	t.Parallel()

	store := storeAt(t, 69)
	provider, err := store.provider()
	if err != nil {
		t.Fatalf("provider: %v", err)
	}
	must := func(query string, args ...any) {
		t.Helper()
		if _, execErr := store.writer.ExecContext(t.Context(), query, args...); execErr != nil {
			t.Fatalf("%s: %v", query, execErr)
		}
	}
	const at = "2026-10-22T09:00:00Z"
	must(`INSERT INTO sites (id, name, base_url, secret_ref, status, created_at, updated_at) VALUES ('s1', 'Shop', 'https://shop', 'site:s1:wp_password', 'active', ?, ?)`, at, at)
	page := `INSERT INTO pages (id, site_id, path, slug, wp_type, status, created_at, updated_at) VALUES (?, 's1', ?, ?, 'page', 'exists', ?, ?)`
	must(page, "pg1", "/a/", "a", at, at)
	must(page, "pg2", "/b/", "b", at, at)
	must(`INSERT INTO page_links (id, site_id, from_page_id, to_page_id, to_url, anchor_text, type, region, source, ours, observed_at) VALUES ('l1', 's1', 'pg1', 'pg2', '/b/', 'b', 'contextual', 'content', 'raw', 1, ?)`, at)
	if got := columnOf(t, store, "page_links", "block_hash"); got != "" {
		t.Fatalf("page_links carries block_hash (%s) before the up migration", got)
	}

	if _, err = provider.UpTo(t.Context(), 70); err != nil {
		t.Fatalf("up to 70: %v", err)
	}
	if got, want := columnOf(t, store, "page_links", "block_hash"), "TEXT notnull=0 default=NULL pk=0"; got != want {
		t.Fatalf("the up migration made page_links.block_hash %q, want %q", got, want)
	}
	must(`UPDATE page_links SET block_hash = 'h1' WHERE id = 'l1'`)
	must(`INSERT INTO page_links (id, site_id, from_page_id, to_page_id, to_url, anchor_text, type, region, source, ours, block_hash, observed_at) VALUES ('l2', 's1', 'pg1', NULL, '/c/', 'c', 'contextual', 'content', 'raw', 0, NULL, ?)`, at)
	keyed := []string{"l1 | h1", "l2 | -"}
	if got := rowsOf(t, store, `SELECT id, coalesce(block_hash, '-') FROM page_links ORDER BY id`); !slices.Equal(got, keyed) {
		t.Fatalf("the links carry\n%s\nwant the placed one keyed to its block and the client's unkeyed\n%s", strings.Join(got, "\n"), strings.Join(keyed, "\n"))
	}

	if _, err = provider.DownTo(t.Context(), 69); err != nil {
		t.Fatalf("down to 69: %v", err)
	}
	if got := columnOf(t, store, "page_links", "block_hash"); got != "" {
		t.Fatalf("page_links still carries block_hash (%s) after the down migration", got)
	}
	kept := []string{"l1 | 1", "l2 | 0"}
	if got := rowsOf(t, store, `SELECT id, ours FROM page_links ORDER BY id`); !slices.Equal(got, kept) {
		t.Fatalf("the down lost links:\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(kept, "\n"))
	}
}

func TestARenderSaysWhetherTheThemePrintsTheH1AndADownDropsIt(t *testing.T) {
	t.Parallel()

	store := storeAt(t, 70)
	provider, err := store.provider()
	if err != nil {
		t.Fatalf("provider: %v", err)
	}
	must := func(query string, args ...any) {
		t.Helper()
		if _, execErr := store.writer.ExecContext(t.Context(), query, args...); execErr != nil {
			t.Fatalf("%s: %v", query, execErr)
		}
	}
	const at = "2026-10-23T09:00:00Z"
	must(`INSERT INTO sites (id, name, base_url, secret_ref, status, created_at, updated_at) VALUES ('s1', 'Shop', 'https://shop', 'site:s1:wp_password', 'active', ?, ?)`, at, at)
	page := `INSERT INTO pages (id, site_id, path, slug, wp_type, status, created_at, updated_at) VALUES (?, 's1', ?, ?, 'page', 'exists', ?, ?)`
	must(page, "pg1", "/a/", "a", at, at)
	must(page, "pg2", "/b/", "b", at, at)
	must(`INSERT INTO page_renders (page_id, render_hash, content_root, extracted_at) VALUES ('pg1', 'h', 'found', ?)`, at)

	if _, err = provider.UpTo(t.Context(), 71); err != nil {
		t.Fatalf("up to 71: %v", err)
	}
	if got := rowsOf(t, store, `SELECT page_id, theme_h1 FROM page_renders ORDER BY page_id`); !slices.Equal(got, []string{"pg1 | "}) {
		t.Fatalf("an earlier render reads %v, want it unknown until the page is rendered again", got)
	}
	must(`INSERT INTO page_renders (page_id, render_hash, content_root, theme_h1, extracted_at) VALUES ('pg2', 'h', 'found', 'printed', ?)`, at)
	if _, execErr := store.writer.ExecContext(t.Context(), `UPDATE page_renders SET theme_h1 = 'maybe' WHERE page_id = 'pg1'`); execErr == nil {
		t.Fatal("a theme heading the model does not name was stored")
	}

	if _, err = provider.DownTo(t.Context(), 70); err != nil {
		t.Fatalf("down to 70: %v", err)
	}
	if got := columnOf(t, store, "page_renders", "theme_h1"); got != "" {
		t.Fatalf("page_renders still carries theme_h1 (%s) after the down migration", got)
	}
	if got := rowsOf(t, store, `SELECT page_id FROM page_renders ORDER BY page_id`); !slices.Equal(got, []string{"pg1", "pg2"}) {
		t.Fatalf("the down lost renders: %v", got)
	}
}

func TestAPageKeepsTheWorkbooksHierarchyAndADownDropsIt(t *testing.T) {
	t.Parallel()

	store := storeAt(t, 71)
	provider, err := store.provider()
	if err != nil {
		t.Fatalf("provider: %v", err)
	}
	must := func(query string, args ...any) {
		t.Helper()
		if _, execErr := store.writer.ExecContext(t.Context(), query, args...); execErr != nil {
			t.Fatalf("%s: %v", query, execErr)
		}
	}
	const at = "2026-10-23T09:00:00Z"
	must(`INSERT INTO sites (id, name, base_url, secret_ref, status, created_at, updated_at) VALUES ('s1', 'Shop', 'https://shop', 'site:s1:wp_password', 'active', ?, ?)`, at, at)
	must(`INSERT INTO pages (id, site_id, path, slug, wp_type, status, created_at, updated_at) VALUES ('pg1', 's1', '/a/', 'a', 'page', 'exists', ?, ?)`, at, at)

	if _, err = provider.UpTo(t.Context(), 72); err != nil {
		t.Fatalf("up to 72: %v", err)
	}
	if got := rowsOf(t, store, `SELECT planned_trail, row_key, parent_source FROM pages`); !slices.Equal(got, []string{"[] |  | site"}) {
		t.Fatalf("an earlier page reads %v, want no trail, no mapping and the site's parent", got)
	}
	must(`UPDATE pages SET planned_trail = '["Peptides"]', row_key = 'row:Groups:bpc-157', parent_source = 'workbook' WHERE id = 'pg1'`)
	if _, execErr := store.writer.ExecContext(t.Context(), `UPDATE pages SET parent_source = 'theirs' WHERE id = 'pg1'`); execErr == nil {
		t.Fatal("a parent source the model does not name was stored")
	}

	if _, err = provider.DownTo(t.Context(), 71); err != nil {
		t.Fatalf("down to 71: %v", err)
	}
	for _, column := range []string{"planned_trail", "row_key", "parent_source"} {
		if got := columnOf(t, store, "pages", column); got != "" {
			t.Fatalf("pages still carries %s (%s) after the down migration", column, got)
		}
	}
	if got := rowsOf(t, store, `SELECT id FROM pages`); !slices.Equal(got, []string{"pg1"}) {
		t.Fatalf("the down lost pages: %v", got)
	}
}

func TestTheLinksOriginIsDroppedAndADownBringsItBackFromTheirMark(t *testing.T) {
	t.Parallel()

	store := storeAt(t, 68)
	provider, err := store.provider()
	if err != nil {
		t.Fatalf("provider: %v", err)
	}
	exec := func(query string, args ...any) error {
		_, execErr := store.writer.ExecContext(t.Context(), query, args...)
		return execErr
	}
	must := func(query string, args ...any) {
		t.Helper()
		if execErr := exec(query, args...); execErr != nil {
			t.Fatalf("%s: %v", query, execErr)
		}
	}
	const at = "2026-10-21T09:00:00Z"
	must(`INSERT INTO sites (id, name, base_url, secret_ref, status, created_at, updated_at) VALUES ('s1', 'Shop', 'https://shop', 'site:s1:wp_password', 'active', ?, ?)`, at, at)
	page := `INSERT INTO pages (id, site_id, path, slug, wp_type, status, created_at, updated_at) VALUES (?, 's1', ?, ?, 'page', 'exists', ?, ?)`
	must(page, "pg1", "/a/", "a", at, at)
	must(page, "pg2", "/b/", "b", at, at)
	before := `INSERT INTO page_links (id, site_id, from_page_id, to_page_id, to_url, anchor_text, origin, type, region, source, ours, observed_at) VALUES (?, 's1', 'pg1', ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	must(before, "l1", "pg2", "/b/", "b", "generated", "contextual", "content", "raw", 1, at)
	must(before, "l2", nil, "/c/", "c", "observed", "contextual", "content", "raw", 0, at)
	must(before, "l3", "pg2", "/b/", "bee", "generated", "contextual", "content", "raw", 0, at)
	must(before, "l4", "pg2", "/b/", "B", "observed", "breadcrumb", "breadcrumb", "render", 0, at)

	if _, err = provider.UpTo(t.Context(), 69); err != nil {
		t.Fatalf("up to 69: %v", err)
	}
	if got := columnOf(t, store, "page_links", "origin"); got != "" {
		t.Fatalf("page_links still carries origin (%s) after the up migration", got)
	}
	marked := []string{"l1 | 1", "l2 | 0", "l3 | 1", "l4 | 0"}
	if got := rowsOf(t, store, `SELECT id, ours FROM page_links ORDER BY id`); !slices.Equal(got, marked) {
		t.Fatalf("the up marked the links\n%s\nwant every generated link ours\n%s", strings.Join(got, "\n"), strings.Join(marked, "\n"))
	}
	must(`INSERT INTO page_links (id, site_id, from_page_id, to_page_id, to_url, anchor_text, type, region, source, ours, observed_at) VALUES ('l5', 's1', 'pg2', 'pg1', '/a/', 'a', 'contextual', 'content', 'raw', 1, ?)`, at)

	if _, err = provider.DownTo(t.Context(), 68); err != nil {
		t.Fatalf("down to 68: %v", err)
	}
	if got, want := columnOf(t, store, "page_links", "origin"), "TEXT notnull=1 default='observed' pk=0"; got != want {
		t.Fatalf("the down migration made page_links.origin %q, want %q", got, want)
	}
	restored := []string{
		"l1 | generated | 1",
		"l2 | observed | 0",
		"l3 | generated | 1",
		"l4 | observed | 0",
		"l5 | generated | 1",
	}
	if got := rowsOf(t, store, `SELECT id, origin, ours FROM page_links ORDER BY id`); !slices.Equal(got, restored) {
		t.Fatalf("the down gave the links the origins\n%s\nwant each told by its mark\n%s", strings.Join(got, "\n"), strings.Join(restored, "\n"))
	}
	if execErr := exec(`UPDATE page_links SET origin = 'guessed' WHERE id = 'l1'`); execErr == nil {
		t.Fatal("the restored origin check accepted a value that is neither generated nor observed")
	}

	if _, err = provider.DownTo(t.Context(), 51); err != nil {
		t.Fatalf("down to 51 through the restored origin: %v", err)
	}
	old := []string{"l1 | generated", "l2 | observed", "l3 | generated", "l4 | observed", "l5 | generated"}
	if got := rowsOf(t, store, `SELECT id, origin FROM page_links ORDER BY id`); !slices.Equal(got, old) {
		t.Fatalf("the links reached migration 51 as\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(old, "\n"))
	}

	if _, err = provider.Up(t.Context()); err != nil {
		t.Fatalf("up again: %v", err)
	}
	again := []string{"l1 | 1", "l2 | 0", "l3 | 1", "l4 | 0", "l5 | 1"}
	if got := rowsOf(t, store, `SELECT id, ours FROM page_links ORDER BY id`); !slices.Equal(got, again) {
		t.Fatalf("the second up marked the links\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(again, "\n"))
	}
}

func TestFindingsSubjectIsUniquePerSite(t *testing.T) {
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
	const at = "2026-10-17T09:00:00Z"

	for _, setup := range []string{
		`INSERT INTO sites (id, name, base_url, secret_ref, status, created_at, updated_at) VALUES ('s1', 'Shop', 'https://shop', 'site:s1:wp_password', 'active', ?, ?)`,
		`INSERT INTO sites (id, name, base_url, secret_ref, status, created_at, updated_at) VALUES ('s2', 'Other', 'https://other', 'site:s2:wp_password', 'active', ?, ?)`,
		`INSERT INTO entities (id, site_id, name, source, created_at, updated_at) VALUES ('e1', 's1', 'Peptides', 'import', ?, ?)`,
		`INSERT INTO pages (id, site_id, path, slug, wp_type, status, created_at, updated_at) VALUES ('pg1', 's1', '/peptides/', 'peptides', 'page', 'exists', ?, ?)`,
	} {
		if err := exec(setup, at, at); err != nil {
			t.Fatalf("%s: %v", setup, err)
		}
	}

	for _, step := range []struct {
		why, id, siteID, code, key string
		pageID, entityID           any
		severity, status           string
		ok                         bool
	}{
		{why: "a finding on a page", id: "f1", siteID: "s1", code: "parent_unresolved", key: "page:pg1:crumb:/x/", pageID: "pg1", severity: "warning", status: "open", ok: true},
		{why: "the same subject under another code", id: "f2", siteID: "s1", code: "parent_differs", key: "page:pg1:crumb:/x/", pageID: "pg1", severity: "warning", status: "open", ok: true},
		{why: "the same code and subject on another site", id: "f3", siteID: "s2", code: "parent_unresolved", key: "page:pg1:crumb:/x/", severity: "warning", status: "open", ok: true},
		{why: "a resolved finding on an entity", id: "f4", siteID: "s1", code: "page_title_differs", key: "entity:e1", entityID: "e1", severity: "info", status: "resolved", ok: true},
		{why: "a dismissed finding on the site", id: "f5", siteID: "s1", code: "plugin_outdated", key: "site", severity: "error", status: "dismissed", ok: true},
		{why: "the same code and subject twice on one site", id: "f6", siteID: "s1", code: "parent_unresolved", key: "page:pg1:crumb:/x/", pageID: "pg1", severity: "warning", status: "open"},
		{why: "a severity the record does not name", id: "f7", siteID: "s1", code: "render_unreachable", key: "page:pg1", pageID: "pg1", severity: "fatal", status: "open"},
		{why: "a status the record does not name", id: "f8", siteID: "s1", code: "render_unreachable", key: "page:pg1", pageID: "pg1", severity: "warning", status: "closed"},
		{why: "a page that does not exist", id: "f9", siteID: "s1", code: "render_unreachable", key: "page:missing", pageID: "missing", severity: "warning", status: "open"},
		{why: "an entity that does not exist", id: "f10", siteID: "s1", code: "parent_differs", key: "entity:missing", entityID: "missing", severity: "warning", status: "open"},
		{why: "a site that does not exist", id: "f11", siteID: "s9", code: "plugin_outdated", key: "site", severity: "error", status: "open"},
	} {
		err := exec(`INSERT INTO findings (id, site_id, page_id, entity_id, code, severity, subject_key, status, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			step.id, step.siteID, step.pageID, step.entityID, step.code, step.severity, step.key, step.status, at, at)
		if step.ok && err != nil {
			t.Fatalf("%s: %v", step.why, err)
		}
		if !step.ok && err == nil {
			t.Fatalf("%s was accepted", step.why)
		}
	}

	if got := rowsOf(t, store, `SELECT subject, options, resolution, resolved_by FROM findings WHERE id = 'f1'`); !slices.Equal(got, []string{"{} | [] |  | "}) {
		t.Fatalf("a finding's defaults = %v, want an empty subject, no options and no resolution", got)
	}
	for _, index := range []struct {
		name    string
		columns []string
	}{
		{name: "findings_site_status", columns: []string{"site_id", "status", "created_at", "id"}},
		{name: "findings_page", columns: []string{"page_id"}},
	} {
		if got := rowsOf(t, store, `SELECT name FROM pragma_index_info(?) ORDER BY seqno`, index.name); !slices.Equal(got, index.columns) {
			t.Fatalf("the index %s covers %v, want %v", index.name, got, index.columns)
		}
	}

	if err := exec(`DELETE FROM pages WHERE id = 'pg1'`); err != nil {
		t.Fatalf("delete the page: %v", err)
	}
	if got := count(`SELECT count(*) FROM findings`); got != 3 {
		t.Fatalf("findings after a page delete = %d, want the 3 not on the page", got)
	}
	if err := exec(`DELETE FROM entities WHERE id = 'e1'`); err != nil {
		t.Fatalf("delete the entity: %v", err)
	}
	if got := count(`SELECT count(*) FROM findings`); got != 2 {
		t.Fatalf("findings after an entity delete = %d, want the site's and the other site's", got)
	}
	if err := exec(`DELETE FROM sites WHERE id = 's1'`); err != nil {
		t.Fatalf("delete the site: %v", err)
	}
	if got := rowsOf(t, store, `SELECT id FROM findings`); !slices.Equal(got, []string{"f3"}) {
		t.Fatalf("findings after a site delete = %v, want the other site's alone", got)
	}
}

func TestPageModelSchema(t *testing.T) {
	t.Parallel()

	store := openStore(t, nil)
	exec := func(query string, args ...any) error {
		_, err := store.writer.ExecContext(t.Context(), query, args...)
		return err
	}
	const at = "2026-10-17T09:00:00Z"

	for _, setup := range []string{
		`INSERT INTO sites (id, name, base_url, secret_ref, status, created_at, updated_at) VALUES ('s1', 'Shop', 'https://shop', 'site:s1:wp_password', 'active', ?, ?)`,
		`INSERT INTO pages (id, site_id, path, slug, wp_type, status, created_at, updated_at) VALUES ('pg1', 's1', '/tb-500/', 'tb-500', 'product', 'exists', ?, ?)`,
	} {
		if err := exec(setup, at, at); err != nil {
			t.Fatalf("%s: %v", setup, err)
		}
	}

	if got := rowsOf(t, store, `SELECT page_type_id, parent_reason, technical_reason, read_only_reason FROM pages WHERE id = 'pg1'`); !slices.Equal(got, []string{"<nil> | none |  | "}) {
		t.Fatalf("a page's model defaults = %v, want no page type, no parent reason, not technical and writable", got)
	}
	if got := rowsOf(t, store, `SELECT settings FROM sites WHERE id = 's1'`); !slices.Equal(got, []string{"{}"}) {
		t.Fatalf("a site's settings default = %v, want an empty object", got)
	}

	for _, step := range []struct {
		why, query string
		args       []any
		ok         bool
	}{
		{why: "a page type", query: `UPDATE pages SET page_type_id = 'pt1' WHERE id = 'pg1'`, ok: true},
		{why: "a parent reason the model does not name", query: `UPDATE pages SET parent_reason = 'guess' WHERE id = 'pg1'`},
		{why: "no parent reason at all", query: `UPDATE pages SET parent_reason = NULL WHERE id = 'pg1'`},
		{why: "a technical reason the model does not name", query: `UPDATE pages SET technical_reason = 'homepage' WHERE id = 'pg1'`},
		{why: "no technical reason at all", query: `UPDATE pages SET technical_reason = NULL WHERE id = 'pg1'`},
		{why: "a page a builder owns", query: `UPDATE pages SET read_only_reason = 'builder:elementor' WHERE id = 'pg1'`, ok: true},
		{why: "a site's settings", query: `UPDATE sites SET settings = '{"linkUrlForm":"auto"}' WHERE id = 's1'`, ok: true},
		{why: "no settings at all", query: `UPDATE sites SET settings = NULL WHERE id = 's1'`},
		{why: "an identity", query: `INSERT INTO page_identity (page_id, url, status, wp_type, wp_parent_id, modified_gmt, observed_at) VALUES ('pg1', 'https://shop/tb-500/', 'publish', 'product', NULL, ?, ?)`, args: []any{at, at}, ok: true},
		{why: "a second identity for the page", query: `INSERT INTO page_identity (page_id, url, status, wp_type, observed_at) VALUES ('pg1', 'https://shop/tb-500/', 'publish', 'product', ?)`, args: []any{at}},
		{why: "an identity of no page", query: `INSERT INTO page_identity (page_id, url, status, wp_type, observed_at) VALUES ('missing', 'https://shop/x/', 'publish', 'page', ?)`, args: []any{at}},
		{why: "a term", query: `INSERT INTO page_terms (page_id, taxonomy, term_id, name, slug) VALUES ('pg1', 'product_cat', 5, 'Peptides', 'peptides')`, ok: true},
		{why: "the same term id in another taxonomy", query: `INSERT INTO page_terms (page_id, taxonomy, term_id, name, slug, parent_term_id) VALUES ('pg1', 'product_tag', 5, 'Healing', 'healing', 0)`, ok: true},
		{why: "the same term twice", query: `INSERT INTO page_terms (page_id, taxonomy, term_id, name, slug) VALUES ('pg1', 'product_cat', 5, 'Peptides', 'peptides')`},
		{why: "a field", query: `INSERT INTO page_fields (page_id, field_key, current_value, current_hash, base_value, base_hash, observed_at, written_at, written_run_id) VALUES ('pg1', 'title', 'TB-500', 'h', 'TB-500', 'h', ?, ?, 'r1')`, args: []any{at, at}, ok: true},
		{why: "an instanced field with its defaults", query: `INSERT INTO page_fields (page_id, field_key, observed_at) VALUES ('pg1', 'variation_description:42', ?)`, args: []any{at}, ok: true},
		{why: "the same field twice", query: `INSERT INTO page_fields (page_id, field_key, observed_at) VALUES ('pg1', 'title', ?)`, args: []any{at}},
		{why: "the current body", query: `INSERT INTO page_bodies (page_id, side, raw, hash, kind, observed_at) VALUES ('pg1', 'current', '<p>x</p>', 'h', 'classic', ?)`, args: []any{at}, ok: true},
		{why: "the base body", query: `INSERT INTO page_bodies (page_id, side, raw, hash, kind, observed_at) VALUES ('pg1', 'base', '<!-- wp:paragraph --><p>x</p><!-- /wp:paragraph -->', 'h', 'blocks', ?)`, args: []any{at}, ok: true},
		{why: "a side the model does not name", query: `INSERT INTO page_bodies (page_id, side, raw, hash, kind, observed_at) VALUES ('pg1', 'draft', '', 'h', 'classic', ?)`, args: []any{at}},
		{why: "a body kind the model does not name", query: `INSERT INTO page_bodies (page_id, side, raw, hash, kind, observed_at) VALUES ('pg1', 'current', '', 'h', 'html', ?)`, args: []any{at}},
		{why: "a written block", query: `INSERT INTO page_blocks_written (page_id, block_hash, run_id, written_at) VALUES ('pg1', 'b1', 'r1', ?)`, args: []any{at}, ok: true},
		{why: "the same block written twice", query: `INSERT INTO page_blocks_written (page_id, block_hash, run_id, written_at) VALUES ('pg1', 'b1', 'r2', ?)`, args: []any{at}},
		{why: "a render", query: `INSERT INTO page_renders (page_id, render_hash, content_root, extracted_at) VALUES ('pg1', 'h', 'found', ?)`, args: []any{at}, ok: true},
		{why: "a second render for the page", query: `INSERT INTO page_renders (page_id, render_hash, content_root, extracted_at) VALUES ('pg1', 'h', 'not_found', ?)`, args: []any{at}},
		{why: "a root state the model does not name", query: `UPDATE page_renders SET content_root = 'missing' WHERE page_id = 'pg1'`},
	} {
		err := exec(step.query, step.args...)
		if step.ok && err != nil {
			t.Fatalf("%s: %v", step.why, err)
		}
		if !step.ok && err == nil {
			t.Fatalf("%s was accepted", step.why)
		}
	}

	for _, accepted := range []struct {
		query  string
		values []string
	}{
		{query: `UPDATE pages SET parent_reason = ? WHERE id = 'pg1'`, values: []string{"jsonld_breadcrumb", "aria_breadcrumb", "wp_parent", "url", "workbook", "none"}},
		{query: `UPDATE pages SET technical_reason = ? WHERE id = 'pg1'`, values: []string{"front_page", "posts_page", "privacy", "shop", "cart", "checkout", "account", "terms", "shortcode", "noindex", "workbook", "manual", ""}},
		{query: `UPDATE page_renders SET content_root = ? WHERE page_id = 'pg1'`, values: []string{"found", "empty_raw", "dynamic_only", "not_found"}},
	} {
		for _, value := range accepted.values {
			if err := exec(accepted.query, value); err != nil {
				t.Fatalf("%s with %q: %v", accepted.query, value, err)
			}
		}
	}

	for _, check := range []struct {
		why, query string
		want       []string
	}{
		{why: "a term's parent defaults to none", query: `SELECT parent_term_id FROM page_terms WHERE taxonomy = 'product_cat'`, want: []string{"0"}},
		{why: "a field starts empty on both sides", query: `SELECT current_value, current_hash, base_value, base_hash, written_at, written_run_id FROM page_fields WHERE field_key = 'variation_description:42'`, want: []string{" |  |  |  | <nil> | <nil>"}},
		{why: "a render starts with no regions, trail, headings or counts", query: `SELECT regions, jsonld, trail, headings, word_count_content, word_count_generated, builder, url_form, robots FROM page_renders`, want: []string{"[] | {} | [] | [] | 0 | 0 |  |  | "}},
		{why: "pages are found by their page type", query: `SELECT name FROM pragma_index_info('pages_page_type')`, want: []string{"page_type_id"}},
		{why: "terms are found by taxonomy and term", query: `SELECT name FROM pragma_index_info('page_terms_taxonomy_term') ORDER BY seqno`, want: []string{"taxonomy", "term_id"}},
	} {
		if got := rowsOf(t, store, check.query); !slices.Equal(got, check.want) {
			t.Fatalf("%s: got %v, want %v", check.why, got, check.want)
		}
	}

	if err := exec(`DELETE FROM pages WHERE id = 'pg1'`); err != nil {
		t.Fatalf("delete the page: %v", err)
	}
	for _, table := range []string{"page_identity", "page_terms", "page_fields", "page_bodies", "page_blocks_written", "page_renders"} {
		if got := rowsOf(t, store, `SELECT count(*) FROM `+table); !slices.Equal(got, []string{"0"}) {
			t.Fatalf("%s after its page is deleted = %v, want none", table, got)
		}
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

func TestTheCatalogRowCarriesNoReasoningEffort(t *testing.T) {
	t.Parallel()

	store := openStore(t, nil)
	provider, err := store.provider()
	if err != nil {
		t.Fatalf("provider: %v", err)
	}
	columns := func() []string {
		t.Helper()
		rows, queryErr := store.writer.QueryContext(t.Context(), `SELECT name FROM pragma_table_info('model_catalog')`)
		if queryErr != nil {
			t.Fatalf("read the columns: %v", queryErr)
		}
		defer rows.Close()
		var names []string
		for rows.Next() {
			var name string
			if scanErr := rows.Scan(&name); scanErr != nil {
				t.Fatalf("scan a column: %v", scanErr)
			}
			names = append(names, name)
		}
		if rowsErr := rows.Err(); rowsErr != nil {
			t.Fatalf("read the columns: %v", rowsErr)
		}
		return names
	}
	const at = "2026-10-03T09:00:00Z"
	insert := `INSERT INTO model_catalog (provider, model, context_tokens, max_output_tokens, input_usd_per_m, output_usd_per_m, rpm, tpm, supports_structured, supports_images, reasoning, enabled, created_at, updated_at) VALUES ('openai', 'gpt-5.6-terra', 1000, 100, 2, 12, 1, 1, 1, 0, 1, 1, ?, ?)`
	if _, err = store.writer.ExecContext(t.Context(), insert, at, at); err != nil {
		t.Fatalf("store a row: %v", err)
	}

	if slices.Contains(columns(), "reasoning_effort") {
		t.Fatal("model_catalog still carries reasoning_effort")
	}

	if _, err = provider.DownTo(t.Context(), 36); err != nil {
		t.Fatalf("down to 36: %v", err)
	}
	if !slices.Contains(columns(), "reasoning_effort") {
		t.Fatal("the down migration did not bring reasoning_effort back")
	}
	var effort string
	if err = store.writer.QueryRowContext(t.Context(), `SELECT reasoning_effort FROM model_catalog WHERE model = 'gpt-5.6-terra'`).Scan(&effort); err != nil || effort != "" {
		t.Fatalf("the restored effort = %q, %v; want the empty default", effort, err)
	}
	for _, value := range []struct {
		effort string
		ok     bool
	}{{effort: "medium", ok: true}, {effort: "xhigh", ok: true}, {effort: "minimal"}} {
		_, updateErr := store.writer.ExecContext(t.Context(), `UPDATE model_catalog SET reasoning_effort = ?`, value.effort)
		if value.ok != (updateErr == nil) {
			t.Errorf("reasoning_effort = %q: %v, want accepted %t", value.effort, updateErr, value.ok)
		}
	}

	if _, err = provider.Up(t.Context()); err != nil {
		t.Fatalf("up again: %v", err)
	}
	if slices.Contains(columns(), "reasoning_effort") {
		t.Fatal("model_catalog carries reasoning_effort after the second up")
	}
}

func TestAProfileOfARemovedProviderIsDropped(t *testing.T) {
	t.Parallel()

	store := openStore(t, nil)
	provider, err := store.provider()
	if err != nil {
		t.Fatalf("provider: %v", err)
	}
	if _, err = provider.DownTo(t.Context(), 37); err != nil {
		t.Fatalf("down to 37: %v", err)
	}

	const at = "2026-10-03T09:00:00Z"
	for _, row := range [][3]string{
		{"writer", "openai", "gpt-5.6-terra"},
		{"judge", "retired", "old-model"},
		{"chat", "elsewhere", "chat-model"},
		{"editor", "OpenAI", "gpt-5.6-luna"},
	} {
		if _, execErr := store.writer.ExecContext(t.Context(),
			`INSERT INTO model_profiles (role, provider, model, updated_at) VALUES (?, ?, ?, ?)`,
			row[0], row[1], row[2], at); execErr != nil {
			t.Fatalf("store the %s profile: %v", row[0], execErr)
		}
	}
	kept := func() []string {
		t.Helper()
		rows, queryErr := store.writer.QueryContext(t.Context(), `SELECT role || ':' || provider FROM model_profiles ORDER BY role`)
		if queryErr != nil {
			t.Fatalf("read the profiles: %v", queryErr)
		}
		defer rows.Close()
		var out []string
		for rows.Next() {
			var row string
			if scanErr := rows.Scan(&row); scanErr != nil {
				t.Fatalf("scan a profile: %v", scanErr)
			}
			out = append(out, row)
		}
		if rowsErr := rows.Err(); rowsErr != nil {
			t.Fatalf("read the profiles: %v", rowsErr)
		}
		return out
	}

	if _, err = provider.Up(t.Context()); err != nil {
		t.Fatalf("up: %v", err)
	}
	if got := kept(); !slices.Equal(got, []string{"writer:openai"}) {
		t.Fatalf("profiles after the up = %v, want the OpenAI one alone", got)
	}

	if _, err = provider.DownTo(t.Context(), 37); err != nil {
		t.Fatalf("the down of a removal must restore nothing and still succeed: %v", err)
	}
	if got := kept(); !slices.Equal(got, []string{"writer:openai"}) {
		t.Fatalf("profiles after the down = %v, want them untouched", got)
	}
	if _, err = provider.Up(t.Context()); err != nil {
		t.Fatalf("up again: %v", err)
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
	exec(`INSERT INTO templates (id, scope, site_id, name, version, spec, created_at, updated_at) VALUES ('t1', 'global', NULL, 'Hub', 1, '{}', ?, ?)`, at, at)
	exec(`INSERT INTO page_types (id, site_id, name, preset, created_at, updated_at) VALUES ('p1', 's1', 'Hub', 'hub', ?, ?)`, at, at)
	exec(`INSERT INTO field_scopes (page_type_id, field_key, link_policy, updated_at) VALUES ('p1', 'body', '{"links":true}', ?)`, at)
	exec(`UPDATE sites SET default_template_id = 't1' WHERE id = 's1'`)
	exec(`INSERT INTO entities (id, site_id, name, source, created_at, updated_at) VALUES ('e1', 's1', 'Shoes', 'user', ?, ?)`, at, at)
	exec(`INSERT INTO entities (id, site_id, name, source, created_at, updated_at) VALUES ('e2', 's1', 'Boots', 'user', ?, ?)`, at, at)
	exec(`INSERT INTO entity_anchors (entity_id, position, text, source, weight) VALUES ('e1', 0, 'shoes', 'user', 1)`)
	exec(`INSERT INTO edges (id, site_id, from_entity_id, to_entity_id, kind, weight, source, status, created_at) VALUES ('g1', 's1', 'e2', 'e1', 'parent', 1, 'user', 'approved', ?)`, at)
	exec(`INSERT INTO pages (id, site_id, path, slug, wp_type, status, entity_id, template_id, created_at, updated_at) VALUES ('pg1', 's1', '/shoes/', 'shoes', 'page', 'planned', 'e1', 't1', ?, ?)`, at, at)
	exec(`INSERT INTO pages (id, site_id, path, slug, parent_page_id, wp_type, status, entity_id, created_at, updated_at) VALUES ('pg2', 's1', '/shoes/boots/', 'boots', 'pg1', 'page', 'planned', 'e2', ?, ?)`, at, at)
	exec(`UPDATE entities SET canonical_page_id = 'pg1' WHERE id = 'e1'`)
	exec(`INSERT INTO page_links (id, site_id, from_page_id, to_page_id, to_url, anchor_text, type, source, ours, observed_at) VALUES ('l1', 's1', 'pg2', 'pg1', '/shoes/', 'shoes', 'contextual', 'raw', 1, ?)`, at)
	exec(`INSERT INTO page_links (id, site_id, from_page_id, to_page_id, to_url, anchor_text, type, source, ours, observed_at) VALUES ('l2', 's1', 'pg1', 'pg2', '/shoes/boots/', 'boots', 'contextual', 'raw', 1, ?)`, at)
	exec(`INSERT INTO template_overrides (id, template_id, scope, site_id, page_id, patch, created_at, updated_at) VALUES ('o1', 't1', 'site', 's1', NULL, '{}', ?, ?)`, at, at)
	exec(`INSERT INTO template_overrides (id, template_id, scope, site_id, page_id, patch, created_at, updated_at) VALUES ('o2', 't1', 'page', NULL, 'pg1', '{}', ?, ?)`, at, at)

	if _, err := store.writer.ExecContext(t.Context(), `INSERT INTO entities (id, site_id, name, source, created_at, updated_at) VALUES ('e3', 's1', 'shoes', 'user', ?, ?)`, at, at); err == nil {
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
	for _, table := range []string{"entities", "pages", "page_links", "page_types", "field_scopes", "template_overrides"} {
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
