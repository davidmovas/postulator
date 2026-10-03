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

	for _, info := range models {
		if info.Ref.Provider != llm.ProviderOpenAI {
			t.Errorf("the embedded catalog carries %s, which is not an OpenAI model", info.Ref)
		}
		if err = info.Validate(); err != nil {
			t.Errorf("%s: %v", info.Ref, err)
		}
	}
}

func TestAnOverrideOfARemovedProviderIsNeverOffered(t *testing.T) {
	t.Parallel()

	retired := llm.ModelRef{Provider: "retired", Model: "old-model"}
	cases := []struct {
		name    string
		enabled bool
	}{
		{name: "a model added under a removed provider", enabled: true},
		{name: "a model switched off under a removed provider", enabled: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			built, repo := newCatalog(t)
			before, err := built.List(t.Context())
			if err != nil {
				t.Fatalf("List: %v", err)
			}
			if err = repo.Upsert(t.Context(), override(retired, tc.enabled, 3)); err != nil {
				t.Fatalf("Upsert: %v", err)
			}

			after, err := built.List(t.Context())
			if err != nil {
				t.Fatalf("List: %v", err)
			}
			if len(after) != len(before) {
				t.Errorf("models = %d, want the %d the catalog offered before", len(after), len(before))
			}
			for _, info := range after {
				if info.Ref.Provider != llm.ProviderOpenAI {
					t.Errorf("the catalog offers %s", info.Ref)
				}
			}
			if _, err = built.Lookup(t.Context(), retired); !errors.IsCode(err, errors.NotFound) {
				t.Errorf("Lookup of the removed provider's model = %v, want %s", err, errors.NotFound)
			}
		})
	}
}

func TestEveryEmbeddedModelPricesACacheRead(t *testing.T) {
	t.Parallel()

	built, _ := newCatalog(t)
	models, err := built.List(t.Context())
	if err != nil {
		t.Fatalf("List: %v", err)
	}

	for _, info := range models {
		if info.CachedInputUSDPerM <= 0 {
			t.Errorf("%s prices no cache read, so a reused prompt is charged as a fresh one", info.Ref)
		}
		if info.CachedInputUSDPerM > info.InputUSDPerM {
			t.Errorf("%s prices a cache read above a fresh token", info.Ref)
		}
	}
}

func TestTheOpenAIModelsPriceTheFlexTierAndTheCacheWrite(t *testing.T) {
	t.Parallel()

	built, _ := newCatalog(t)
	cases := []struct {
		model  string
		input  float64
		write  float64
		flexIn float64
		cached float64
		flexWr float64
		flexOt float64
	}{
		{model: "gpt-5.6-sol", input: 4.00, write: 5.00, flexIn: 2.00, cached: 0.20, flexWr: 2.50, flexOt: 10.00},
		{model: "gpt-5.6-terra", input: 2.00, write: 2.50, flexIn: 1.00, cached: 0.10, flexWr: 1.25, flexOt: 6.00},
		{model: "gpt-5.6-luna", input: 0.20, write: 0.25, flexIn: 0.10, cached: 0.01, flexWr: 0.125, flexOt: 0.60},
	}

	for _, tc := range cases {
		t.Run(tc.model, func(t *testing.T) {
			t.Parallel()

			info, err := built.Lookup(t.Context(), llm.ModelRef{Provider: "openai", Model: tc.model})
			if err != nil {
				t.Fatalf("Lookup: %v", err)
			}
			if !info.OffersFlex() {
				t.Fatalf("%s offers no flex tier", tc.model)
			}
			if info.InputUSDPerM != tc.input || info.CacheWriteUSDPerM != tc.write {
				t.Errorf("%s prices input %v and a cache write %v, want %v and %v",
					tc.model, info.InputUSDPerM, info.CacheWriteUSDPerM, tc.input, tc.write)
			}
			if info.FlexInputUSDPerM != tc.flexIn || info.FlexCachedInputUSDPerM != tc.cached ||
				info.FlexCacheWriteUSDPerM != tc.flexWr || info.FlexOutputUSDPerM != tc.flexOt {
				t.Errorf("%s prices flex at %v / %v / %v / %v, want %v / %v / %v / %v", tc.model,
					info.FlexInputUSDPerM, info.FlexCachedInputUSDPerM, info.FlexCacheWriteUSDPerM, info.FlexOutputUSDPerM,
					tc.flexIn, tc.cached, tc.flexWr, tc.flexOt)
			}
			if info.FlexInputUSDPerM*2 != info.InputUSDPerM || info.FlexOutputUSDPerM*2 != info.OutputUSDPerM {
				t.Errorf("%s does not price flex at half the standard tier", tc.model)
			}
		})
	}
}

func TestEveryEmbeddedCacheWriteCostsAQuarterMoreThanAFreshToken(t *testing.T) {
	t.Parallel()

	built, _ := newCatalog(t)
	models, err := built.List(t.Context())
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	for _, info := range models {
		if info.CacheWriteUSDPerM == 0 {
			continue
		}
		if want := info.InputUSDPerM * 1.25; info.CacheWriteUSDPerM != want {
			t.Errorf("%s prices a cache write at %v, want %v", info.Ref, info.CacheWriteUSDPerM, want)
		}
		if info.OffersFlex() && info.FlexCacheWriteUSDPerM != info.FlexInputUSDPerM*1.25 {
			t.Errorf("%s prices a flex cache write at %v, want %v", info.Ref, info.FlexCacheWriteUSDPerM, info.FlexInputUSDPerM*1.25)
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
		{name: "image", role: llm.RoleImage},
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

func TestAnOverrideTakesTheBasePriceForEveryPriceItLeftAtZero(t *testing.T) {
	t.Parallel()

	terra := llm.ModelRef{Provider: "openai", Model: "gpt-5.6-terra"}
	base := llm.ModelInfo{
		InputUSDPerM: 2, CachedInputUSDPerM: 0.2, CacheWriteUSDPerM: 2.5, OutputUSDPerM: 12,
		FlexInputUSDPerM: 1, FlexCachedInputUSDPerM: 0.1, FlexCacheWriteUSDPerM: 1.25, FlexOutputUSDPerM: 6,
	}
	prices := func(info llm.ModelInfo) llm.ModelInfo {
		return llm.ModelInfo{
			InputUSDPerM: info.InputUSDPerM, CachedInputUSDPerM: info.CachedInputUSDPerM,
			CacheWriteUSDPerM: info.CacheWriteUSDPerM, OutputUSDPerM: info.OutputUSDPerM,
			FlexInputUSDPerM: info.FlexInputUSDPerM, FlexCachedInputUSDPerM: info.FlexCachedInputUSDPerM,
			FlexCacheWriteUSDPerM: info.FlexCacheWriteUSDPerM, FlexOutputUSDPerM: info.FlexOutputUSDPerM,
		}
	}

	cases := []struct {
		name  string
		ref   llm.ModelRef
		saved llm.ModelInfo
		want  llm.ModelInfo
	}{
		{
			name:  "a row saved before the flex and cache write prices existed",
			ref:   terra,
			saved: llm.ModelInfo{InputUSDPerM: 2, CachedInputUSDPerM: 0.2, OutputUSDPerM: 12},
			want:  base,
		},
		{
			name:  "a row saved without its cached price",
			ref:   terra,
			saved: llm.ModelInfo{InputUSDPerM: 2, OutputUSDPerM: 12},
			want:  base,
		},
		{
			name:  "a row with no price at all",
			ref:   terra,
			saved: llm.ModelInfo{},
			want:  base,
		},
		{
			name: "prices the row sets are its own",
			ref:  terra,
			saved: llm.ModelInfo{
				InputUSDPerM: 3, CachedInputUSDPerM: 0.3, CacheWriteUSDPerM: 3.75, OutputUSDPerM: 15,
				FlexInputUSDPerM: 1.5, FlexCachedInputUSDPerM: 0.15, FlexCacheWriteUSDPerM: 1.875, FlexOutputUSDPerM: 7.5,
			},
			want: llm.ModelInfo{
				InputUSDPerM: 3, CachedInputUSDPerM: 0.3, CacheWriteUSDPerM: 3.75, OutputUSDPerM: 15,
				FlexInputUSDPerM: 1.5, FlexCachedInputUSDPerM: 0.15, FlexCacheWriteUSDPerM: 1.875, FlexOutputUSDPerM: 7.5,
			},
		},
		{
			name:  "only the prices left at zero are filled",
			ref:   terra,
			saved: llm.ModelInfo{InputUSDPerM: 2.5, OutputUSDPerM: 14, FlexInputUSDPerM: 1.25, FlexOutputUSDPerM: 7},
			want: llm.ModelInfo{
				InputUSDPerM: 2.5, CachedInputUSDPerM: 0.2, CacheWriteUSDPerM: 2.5, OutputUSDPerM: 14,
				FlexInputUSDPerM: 1.25, FlexCachedInputUSDPerM: 0.1, FlexCacheWriteUSDPerM: 1.25, FlexOutputUSDPerM: 7,
			},
		},
		{
			name:  "a model the embedded catalog does not carry keeps its zeros",
			ref:   llm.ModelRef{Provider: "openai", Model: "gpt-house-blend"},
			saved: llm.ModelInfo{InputUSDPerM: 2, OutputUSDPerM: 12},
			want:  llm.ModelInfo{InputUSDPerM: 2, OutputUSDPerM: 12},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			built, repo := newCatalog(t)
			stored := override(tc.ref, true, 0)
			stored.Info.InputUSDPerM, stored.Info.CachedInputUSDPerM = tc.saved.InputUSDPerM, tc.saved.CachedInputUSDPerM
			stored.Info.CacheWriteUSDPerM, stored.Info.OutputUSDPerM = tc.saved.CacheWriteUSDPerM, tc.saved.OutputUSDPerM
			stored.Info.FlexInputUSDPerM, stored.Info.FlexCachedInputUSDPerM = tc.saved.FlexInputUSDPerM, tc.saved.FlexCachedInputUSDPerM
			stored.Info.FlexCacheWriteUSDPerM, stored.Info.FlexOutputUSDPerM = tc.saved.FlexCacheWriteUSDPerM, tc.saved.FlexOutputUSDPerM
			if err := repo.Upsert(t.Context(), stored); err != nil {
				t.Fatalf("Upsert: %v", err)
			}

			info, err := built.Lookup(t.Context(), tc.ref)
			if err != nil {
				t.Fatalf("Lookup: %v", err)
			}
			if got := prices(info); got != tc.want {
				t.Errorf("prices = %+v, want %+v", got, tc.want)
			}
			if info.MaxOutputTokens != stored.Info.MaxOutputTokens || info.RPM != stored.Info.RPM {
				t.Errorf("info = %+v, want the override's limits kept", info)
			}

			listed, err := built.List(t.Context())
			if err != nil {
				t.Fatalf("List: %v", err)
			}
			for _, model := range listed {
				if model.Ref == tc.ref && prices(model) != tc.want {
					t.Errorf("listed prices = %+v, want %+v", prices(model), tc.want)
				}
			}
		})
	}
}

func TestLookupRejectsAnUnknownModel(t *testing.T) {
	t.Parallel()

	built, _ := newCatalog(t)
	if _, err := built.Lookup(t.Context(), llm.ModelRef{Provider: "openai", Model: "nope"}); !errors.IsCode(err, errors.NotFound) {
		t.Fatalf("Lookup error = %v, want %s", err, errors.NotFound)
	}
}

func TestAnOverrideKeepsWhetherTheModelReasons(t *testing.T) {
	t.Parallel()

	for _, reasons := range []bool{true, false} {
		built, repo := newCatalog(t)
		ref := llm.ModelRef{Provider: "openai", Model: "gpt-5.6-luna"}
		stored := override(ref, true, 1)
		stored.Info.Reasoning = reasons

		if err := repo.Upsert(t.Context(), stored); err != nil {
			t.Fatalf("Upsert: %v", err)
		}

		info, err := built.Lookup(t.Context(), ref)
		if err != nil {
			t.Fatalf("Lookup: %v", err)
		}
		if info.Reasoning != reasons {
			t.Errorf("reasoning = %t, want the override's %t", info.Reasoning, reasons)
		}
	}
}
