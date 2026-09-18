package profiles_test

import (
	"context"
	"testing"
	"time"

	"github.com/davidmovas/postulator/internal/adapters/llm/profiles"
	"github.com/davidmovas/postulator/internal/adapters/sqlite"
	"github.com/davidmovas/postulator/internal/adapters/sqlite/sqlitetest"
	"github.com/davidmovas/postulator/internal/domain/llm"
	"github.com/davidmovas/postulator/internal/domain/site"
	"github.com/davidmovas/postulator/internal/kernel/clock"
	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/kernel/id"
)

type fixedDefaults map[llm.Role]llm.ModelRef

func (d fixedDefaults) Default(role llm.Role) (llm.ModelRef, error) {
	ref, ok := d[role]
	if !ok {
		return llm.ModelRef{}, errors.New(errors.NotFound, "no default for this role")
	}
	return ref, nil
}

func newProfiles(t *testing.T) (*profiles.Profiles, *sqlite.SiteRepo) {
	t.Helper()

	store := sqlitetest.Open(t)
	sites := sqlite.NewSiteRepo(store)
	resolver := profiles.New(
		sqlite.NewModelProfileRepo(store),
		sites,
		fixedDefaults{llm.RoleWriter: {Provider: "openai", Model: "catalog"}},
		clock.NewFake(time.Date(2026, 9, 18, 10, 0, 0, 0, time.UTC)),
	)
	return resolver, sites
}

func seedSite(t *testing.T, repo *sqlite.SiteRepo, refs map[llm.Role]llm.ModelRef) string {
	t.Helper()

	at := time.Date(2026, 9, 18, 9, 0, 0, 0, time.UTC)
	record := site.Site{
		ID:        id.New(),
		Name:      "Koffein",
		BaseURL:   "https://koffein.example",
		Status:    site.StatusActive,
		Plugin:    site.PluginState{Capabilities: []string{}},
		Defaults:  site.Defaults{ModelProfiles: refs},
		CreatedAt: at,
		UpdatedAt: at,
	}
	record.SecretRef = site.SecretRef(record.ID)
	if err := repo.Insert(t.Context(), record); err != nil {
		t.Fatalf("insert the site: %v", err)
	}
	return record.ID
}

func TestResolvePrecedence(t *testing.T) {
	t.Parallel()

	var (
		fromTemplate = llm.ModelRef{Provider: "openai", Model: "template"}
		fromSite     = llm.ModelRef{Provider: "openai", Model: "site"}
		fromGlobal   = llm.ModelRef{Provider: "openai", Model: "global"}
		fromCatalog  = llm.ModelRef{Provider: "openai", Model: "catalog"}
	)

	cases := []struct {
		name     string
		template map[llm.Role]llm.ModelRef
		siteRefs map[llm.Role]llm.ModelRef
		global   bool
		want     llm.ModelRef
	}{
		{
			name:     "the template wins",
			template: map[llm.Role]llm.ModelRef{llm.RoleWriter: fromTemplate},
			siteRefs: map[llm.Role]llm.ModelRef{llm.RoleWriter: fromSite},
			global:   true,
			want:     fromTemplate,
		},
		{
			name:     "the site comes next",
			siteRefs: map[llm.Role]llm.ModelRef{llm.RoleWriter: fromSite},
			global:   true,
			want:     fromSite,
		},
		{
			name:     "an incomplete site profile is ignored",
			siteRefs: map[llm.Role]llm.ModelRef{llm.RoleWriter: {Provider: "openai"}},
			global:   true,
			want:     fromGlobal,
		},
		{name: "the global profile comes next", global: true, want: fromGlobal},
		{name: "the catalog default is the floor", want: fromCatalog},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			resolver, sites := newProfiles(t)
			ctx := context.Background()
			if tc.global {
				if err := resolver.Set(ctx, llm.RoleWriter, fromGlobal); err != nil {
					t.Fatalf("Set: %v", err)
				}
			}
			siteID := seedSite(t, sites, tc.siteRefs)

			got, err := resolver.Resolve(ctx, siteID, llm.RoleWriter, tc.template)
			if err != nil {
				t.Fatalf("Resolve: %v", err)
			}
			if got != tc.want {
				t.Errorf("Resolve = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestResolveWithoutASite(t *testing.T) {
	t.Parallel()

	resolver, _ := newProfiles(t)
	got, err := resolver.Resolve(t.Context(), "", llm.RoleWriter, nil)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if got.Model != "catalog" {
		t.Errorf("Resolve = %v, want the catalog default", got)
	}
}

func TestResolveFailures(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		siteID string
		role   llm.Role
		want   errors.Code
	}{
		{name: "an unknown role", role: "painter", want: errors.Invalid},
		{name: "a missing site", siteID: id.New(), role: llm.RoleWriter, want: errors.NotFound},
		{name: "a role with no default anywhere", role: llm.RoleJudge, want: errors.NotFound},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			resolver, _ := newProfiles(t)
			if _, err := resolver.Resolve(t.Context(), tc.siteID, tc.role, nil); !errors.IsCode(err, tc.want) {
				t.Fatalf("Resolve error = %v, want %s", err, tc.want)
			}
		})
	}
}

func TestSetGlobalProfile(t *testing.T) {
	t.Parallel()

	resolver, _ := newProfiles(t)
	ctx := context.Background()

	if err := resolver.Set(ctx, "painter", llm.ModelRef{Provider: "openai", Model: "x"}); !errors.IsCode(err, errors.Invalid) {
		t.Fatalf("Set unknown role error = %v, want %s", err, errors.Invalid)
	}
	if err := resolver.Set(ctx, llm.RoleChat, llm.ModelRef{Provider: "openai"}); !errors.IsCode(err, errors.Invalid) {
		t.Fatalf("Set incomplete reference error = %v, want %s", err, errors.Invalid)
	}

	chat := llm.ModelRef{Provider: "anthropic", Model: "claude-sonnet-5"}
	if err := resolver.Set(ctx, llm.RoleChat, chat); err != nil {
		t.Fatalf("Set: %v", err)
	}

	global, err := resolver.Global(ctx)
	if err != nil {
		t.Fatalf("Global: %v", err)
	}
	if global[llm.RoleChat] != chat {
		t.Errorf("global = %v, want the chat profile", global)
	}
}
