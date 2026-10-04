package sqlite_test

import (
	"testing"
	"time"

	"github.com/davidmovas/postulator/internal/adapters/sqlite"
	"github.com/davidmovas/postulator/internal/adapters/sqlite/sqlitetest"
	"github.com/davidmovas/postulator/internal/domain/llm"
)

func TestModelCatalogRepoKeepsEveryPrice(t *testing.T) {
	t.Parallel()

	created := time.Date(2026, 10, 3, 9, 0, 0, 0, time.UTC)
	base := llm.ModelInfo{
		Ref: llm.ModelRef{Provider: "openai", Model: "gpt-5.6-terra"}, ContextTokens: 1050000, MaxOutputTokens: 128000,
		InputUSDPerM: 2, OutputUSDPerM: 12, RPM: 500, TPM: 500000, SupportsStructured: true, Reasoning: true,
	}
	priced := func(change func(info *llm.ModelInfo)) llm.ModelInfo {
		info := base
		change(&info)
		return info
	}

	cases := []struct {
		name    string
		info    llm.ModelInfo
		enabled bool
	}{
		{name: "standard prices only", info: base, enabled: true},
		{
			name: "a cache read and a cache write price",
			info: priced(func(info *llm.ModelInfo) {
				info.CachedInputUSDPerM, info.CacheWriteUSDPerM = 0.2, 2.5
			}),
			enabled: true,
		},
		{
			name: "every flex price",
			info: priced(func(info *llm.ModelInfo) {
				info.CachedInputUSDPerM, info.CacheWriteUSDPerM = 0.2, 2.5
				info.FlexInputUSDPerM, info.FlexCachedInputUSDPerM = 1, 0.1
				info.FlexCacheWriteUSDPerM, info.FlexOutputUSDPerM = 1.25, 6
			}),
			enabled: true,
		},
		{
			name: "a disabled model keeps its prices",
			info: priced(func(info *llm.ModelInfo) {
				info.FlexInputUSDPerM, info.FlexOutputUSDPerM = 1, 6
			}),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			repo := sqlite.NewModelCatalogRepo(sqlitetest.Open(t))
			first := llm.ModelOverride{Info: base, Enabled: true, CreatedAt: created, UpdatedAt: created}
			if err := repo.Upsert(t.Context(), first); err != nil {
				t.Fatalf("Upsert: %v", err)
			}
			second := llm.ModelOverride{
				Info: tc.info, Enabled: tc.enabled, CreatedAt: created.Add(time.Hour), UpdatedAt: created.Add(time.Hour),
			}
			if err := repo.Upsert(t.Context(), second); err != nil {
				t.Fatalf("Upsert again: %v", err)
			}

			listed, err := repo.List(t.Context())
			if err != nil {
				t.Fatalf("List: %v", err)
			}
			if len(listed) != 1 {
				t.Fatalf("overrides = %d, want one row per model", len(listed))
			}
			got := listed[0]
			if got.Info != tc.info {
				t.Errorf("info = %+v, want %+v", got.Info, tc.info)
			}
			if got.Enabled != tc.enabled || !got.CreatedAt.Equal(created) || !got.UpdatedAt.Equal(second.UpdatedAt) {
				t.Errorf("override = %+v, want enabled %t, the first creation and the last update", got, tc.enabled)
			}
		})
	}
}
