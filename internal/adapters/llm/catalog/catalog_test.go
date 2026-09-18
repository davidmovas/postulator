package catalog_test

import (
	"context"
	"testing"
	"time"

	"github.com/davidmovas/postulator/internal/adapters/llm/catalog"
	"github.com/davidmovas/postulator/internal/adapters/sqlite"
	"github.com/davidmovas/postulator/internal/adapters/sqlite/sqlitetest"
	"github.com/davidmovas/postulator/internal/domain/llm"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

func newCatalog(t *testing.T) (*catalog.Catalog, *sqlite.ModelCatalogRepo) {
	t.Helper()

	repo := sqlite.NewModelCatalogRepo(sqlitetest.Open(t))
	built, err := catalog.New(repo)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return built, repo
}

func override(ref llm.ModelRef, enabled bool, price float64) llm.ModelOverride {
	at := time.Date(2026, 9, 18, 10, 0, 0, 0, time.UTC)
	return llm.ModelOverride{
		Info: llm.ModelInfo{
			Ref:             ref,
			ContextTokens:   64000,
			MaxOutputTokens: 4096,
			InputUSDPerM:    price,
			OutputUSDPerM:   price * 2,
			RPM:             10,
			TPM:             1000,
		},
		Enabled:   enabled,
		CreatedAt: at,
		UpdatedAt: at,
	}
}

func TestEmbeddedCatalogIsUsable(t *testing.T) {
	t.Parallel()

	built, _ := newCatalog(t)
	models, err := built.List(t.Context())
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(models) == 0 {
		t.Fatal("the embedded catalog is empty")
	}

	providers := map[string]bool{}
	for _, info := range models {
		providers[info.Ref.Provider] = true
		if err = info.Validate(); err != nil {
			t.Errorf("%s: %v", info.Ref, err)
		}
	}
	for _, want := range []string{"openai", "anthropic", "gemini"} {
		if !providers[want] {
			t.Errorf("the embedded catalog carries no %s model", want)
		}
	}
}

func TestDefaults(t *testing.T) {
	t.Parallel()

	built, _ := newCatalog(t)
	cases := []struct {
		name    string
		role    llm.Role
		wantErr bool
	}{
		{name: "writer", role: llm.RoleWriter},
		{name: "editor", role: llm.RoleEditor},
		{name: "linker", role: llm.RoleLinker},
		{name: "judge", role: llm.RoleJudge},
		{name: "chat", role: llm.RoleChat},
		{name: "image has no text default", role: llm.RoleImage, wantErr: true},
		{name: "an unknown role has none", role: "painter", wantErr: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ref, err := built.Default(tc.role)
			if tc.wantErr {
				if !errors.IsCode(err, errors.NotFound) {
					t.Fatalf("Default(%s) error = %v, want %s", tc.role, err, errors.NotFound)
				}
				return
			}
			if err != nil {
				t.Fatalf("Default(%s): %v", tc.role, err)
			}
			if _, err = built.Lookup(t.Context(), ref); err != nil {
				t.Errorf("the default for %s is not in the catalog: %v", tc.role, err)
			}
		})
	}
}

func TestOverrides(t *testing.T) {
	t.Parallel()

	built, repo := newCatalog(t)
	ctx := context.Background()

	known, err := built.List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	existing := known[0].Ref
	added := llm.ModelRef{Provider: "openai", Model: "gpt-house-blend"}

	if err = repo.Upsert(ctx, override(added, true, 3)); err != nil {
		t.Fatalf("Upsert: %v", err)
	}
	info, err := built.Lookup(ctx, added)
	if err != nil {
		t.Fatalf("Lookup added: %v", err)
	}
	if info.InputUSDPerM != 3 || info.MaxOutputTokens != 4096 {
		t.Errorf("added model = %+v, want the override values", info)
	}

	if err = repo.Upsert(ctx, override(existing, false, 0)); err != nil {
		t.Fatalf("Upsert disable: %v", err)
	}
	if _, err = built.Lookup(ctx, existing); !errors.IsCode(err, errors.NotFound) {
		t.Fatalf("Lookup disabled error = %v, want %s", err, errors.NotFound)
	}

	after, err := built.List(ctx)
	if err != nil {
		t.Fatalf("List after: %v", err)
	}
	if len(after) != len(known) {
		t.Errorf("models = %d, want %d after one addition and one removal", len(after), len(known))
	}

	revived := override(existing, true, 9)
	revived.Info.ContextTokens = 128000
	revived.Info.MaxOutputTokens = 8192
	if err = repo.Upsert(ctx, revived); err != nil {
		t.Fatalf("Upsert revive: %v", err)
	}
	info, err = built.Lookup(ctx, existing)
	if err != nil {
		t.Fatalf("Lookup after the model was switched back on: %v", err)
	}
	if info.InputUSDPerM != 9 {
		t.Errorf("revived model = %+v, want the new override values", info)
	}
}

func TestLookupRejectsAnUnknownModel(t *testing.T) {
	t.Parallel()

	built, _ := newCatalog(t)
	if _, err := built.Lookup(t.Context(), llm.ModelRef{Provider: "openai", Model: "nope"}); !errors.IsCode(err, errors.NotFound) {
		t.Fatalf("Lookup error = %v, want %s", err, errors.NotFound)
	}
}
