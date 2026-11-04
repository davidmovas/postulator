package sqlite

import (
	"encoding/json"
	"slices"
	"testing"

	"github.com/davidmovas/postulator/internal/domain/field"
)

func asUpgradedBy0092(t *testing.T, store *Store) {
	t.Helper()

	if _, err := store.writer.ExecContext(t.Context(),
		`UPDATE field_scopes SET link_policy = json_remove(link_policy, '$.forbidExternal')`); err != nil {
		t.Fatalf("leave the policies as 0092 leaves them: %v", err)
	}
}

func TestTheExternalSwitchLeavesThePoliciesAndAResearchingTemplateLinksItsSources(t *testing.T) {
	t.Parallel()

	store := storeAt(t, 91)
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
	one := func(query string, args ...any) string {
		t.Helper()
		got := rowsOf(t, store, query, args...)
		if len(got) != 1 {
			t.Fatalf("%s answered %v, want one row", query, got)
		}
		return got[0]
	}
	const (
		at          = "2026-11-03T08:00:00Z"
		linked      = `{"links":true,"rules":{"upDepth":1,"downLinks":true,"siblingMinWeight":0.5,"maxLinks":20,"maxPerTarget":1,"parentLinkWithinParagraphs":2,"childrenSection":true},"forbidExternal":true,"forbidSelf":true,"anchorStrategy":"prefer_user"}`
		unlinked    = `{"links":false,"rules":{"upDepth":0,"downLinks":false,"siblingMinWeight":0,"maxLinks":0,"maxPerTarget":0,"parentLinkWithinParagraphs":0,"childrenSection":false},"forbidExternal":false,"forbidSelf":false,"anchorStrategy":"prefer_user"}`
		researching = `{"sections":[],"recipe":[{"name":"resolve_context","enabled":true},{"name":"research","enabled":true,"params":{"focus":"doses"}}]}`
		resting     = `{"sections":[],"recipe":[{"name":"resolve_context","enabled":true},{"name":"research","enabled":false}]}`
		chosen      = `{"sections":[],"sourceLinks":{"max":1,"perDomain":0,"min":1},"recipe":[{"name":"research","enabled":true}]}`
		scopeArgs   = `{"op":"scope","field":"body","links":{"links":true,"forbidExternal":false,"forbidSelf":true,"anchorStrategy":"rotate"}}`
		planSteps   = `[{"id":"one","tool":"pagetypes","args":` + scopeArgs + `,"status":"pending"},{"id":"two","tool":"runs","args":{"op":"start"},"status":"pending"}]`
	)
	must(`INSERT INTO sites (id, name, base_url, secret_ref, status, created_at, updated_at) VALUES ('s1', 's1', 'https://s1', 'site:s1:wp_password', 'active', ?, ?)`, at, at)
	must(`INSERT INTO page_types (id, site_id, name, preset, created_at, updated_at) VALUES ('t-hub', 's1', 'Hub', 'hub', ?, ?)`, at, at)
	must(`INSERT INTO field_scopes (page_type_id, field_key, link_policy, updated_at) VALUES ('t-hub', 'body', ?, ?)`, linked, at)
	must(`INSERT INTO field_scopes (page_type_id, field_key, link_policy, updated_at) VALUES ('t-hub', 'seo_title', ?, ?)`, unlinked, at)
	for id, spec := range map[string]string{"tpl-researching": researching, "tpl-resting": resting, "tpl-chosen": chosen} {
		must(`INSERT INTO templates (id, scope, site_id, name, version, spec, created_at, updated_at) VALUES (?, 'global', NULL, ?, 1, ?, ?, ?)`, id, id, spec, at, at)
	}
	must(`INSERT INTO template_overrides (id, template_id, scope, site_id, page_id, patch, created_at, updated_at) VALUES ('o1', 'tpl-chosen', 'site', 's1', NULL, '{"sourceLinks":{"max":2}}', ?, ?)`, at, at)
	must(`INSERT INTO conversations (id, site_id, mode, created_at, updated_at) VALUES ('c1', 's1', 'confirm', ?, ?)`, at, at)
	must(`INSERT INTO pending_actions (id, conversation_id, tool, args, summary, status, created_at, updated_at) VALUES ('waiting', 'c1', 'pagetypes', ?, 'scope', 'pending', ?, ?)`, scopeArgs, at, at)
	must(`INSERT INTO pending_actions (id, conversation_id, tool, args, summary, status, created_at, updated_at) VALUES ('done', 'c1', 'pagetypes', ?, 'scope', 'executed', ?, ?)`, scopeArgs, at, at)
	must(`INSERT INTO agent_plans (id, conversation_id, site_id, title, status, steps, created_at, updated_at) VALUES ('p1', 'c1', 's1', 'Plan', 'proposed', ?, ?, ?)`, planSteps, at, at)

	if _, err = provider.UpTo(t.Context(), 92); err != nil {
		t.Fatalf("up to 92: %v", err)
	}
	for _, key := range []string{"body", "seo_title"} {
		raw := one(`SELECT link_policy FROM field_scopes WHERE page_type_id = 't-hub' AND field_key = ?`, key)
		if _, parseErr := field.ParsePolicy([]byte(raw)); parseErr != nil {
			t.Errorf("the %s policy after 92 is not readable: %v (%s)", key, parseErr, raw)
		}
	}
	if got := rowsOf(t, store, `SELECT id, coalesce(json_extract(spec, '$.sourceLinks'), 'none') FROM templates ORDER BY id`); !slices.Equal(got, []string{
		`tpl-chosen | {"max":1,"perDomain":0,"min":1}`, `tpl-researching | {"max":3,"perDomain":1,"min":0}`, "tpl-resting | none",
	}) {
		t.Errorf("the templates after 92 = %v, want a researching one to link three sources and the others as they were", got)
	}
	if got := one(`SELECT json_type(args, '$.links.forbidExternal') IS NULL FROM pending_actions WHERE id = 'waiting'`); got != "1" {
		t.Errorf("the waiting action still carries the external switch")
	}
	if got := one(`SELECT args FROM pending_actions WHERE id = 'done'`); got != scopeArgs {
		t.Errorf("the executed action = %s, want it as it was", got)
	}
	var steps []struct {
		ID   string          `json:"id"`
		Args json.RawMessage `json:"args"`
	}
	if err = json.Unmarshal([]byte(one(`SELECT steps FROM agent_plans WHERE id = 'p1'`)), &steps); err != nil {
		t.Fatalf("decode the plan steps: %v", err)
	}
	if len(steps) != 2 || steps[0].ID != "one" || steps[1].ID != "two" || string(steps[1].Args) != `{"op":"start"}` {
		t.Fatalf("the plan steps = %+v, want both in their order and the second as it was", steps)
	}
	var scoped struct {
		Links map[string]any `json:"links"`
	}
	if err = json.Unmarshal(steps[0].Args, &scoped); err != nil {
		t.Fatalf("decode the first step's arguments: %v", err)
	}
	if _, held := scoped.Links["forbidExternal"]; held || scoped.Links["anchorStrategy"] != "rotate" {
		t.Errorf("the plan's scope step = %v, want the external switch gone and the rest kept", scoped.Links)
	}

	if _, err = provider.DownTo(t.Context(), 91); err != nil {
		t.Fatalf("down to 91: %v", err)
	}
	if got := rowsOf(t, store, `SELECT field_key, json_extract(link_policy, '$.forbidExternal') FROM field_scopes ORDER BY field_key`); !slices.Equal(got, []string{
		"body | 1", "seo_title | 0",
	}) {
		t.Errorf("the policies after the down = %v, want a linked field refusing the outside again", got)
	}
	if got := rowsOf(t, store, `SELECT id FROM templates WHERE json_type(spec, '$.sourceLinks') IS NOT NULL UNION ALL SELECT id FROM template_overrides WHERE json_type(patch, '$.sourceLinks') IS NOT NULL`); len(got) != 0 {
		t.Errorf("the source links outlived the down in %v", got)
	}
	if _, err = provider.Up(t.Context()); err != nil {
		t.Fatalf("up again: %v", err)
	}
}
