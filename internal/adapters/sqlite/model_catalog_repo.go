package sqlite

import (
	"context"
	"database/sql"

	"github.com/davidmovas/postulator/internal/domain/llm"
)

const (
	modelColumns = `provider, model, context_tokens, max_output_tokens, input_usd_per_m, cached_input_usd_per_m,
		cache_write_usd_per_m, output_usd_per_m, flex_input_usd_per_m, flex_cached_input_usd_per_m,
		flex_cache_write_usd_per_m, flex_output_usd_per_m, rpm, tpm,
		supports_structured, supports_images, reasoning, reasoning_effort, enabled, created_at, updated_at`
	upsertModel = `INSERT INTO model_catalog (` + modelColumns + `)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT (provider, model) DO UPDATE SET
			context_tokens = excluded.context_tokens,
			max_output_tokens = excluded.max_output_tokens,
			input_usd_per_m = excluded.input_usd_per_m,
			cached_input_usd_per_m = excluded.cached_input_usd_per_m,
			cache_write_usd_per_m = excluded.cache_write_usd_per_m,
			output_usd_per_m = excluded.output_usd_per_m,
			flex_input_usd_per_m = excluded.flex_input_usd_per_m,
			flex_cached_input_usd_per_m = excluded.flex_cached_input_usd_per_m,
			flex_cache_write_usd_per_m = excluded.flex_cache_write_usd_per_m,
			flex_output_usd_per_m = excluded.flex_output_usd_per_m,
			rpm = excluded.rpm,
			tpm = excluded.tpm,
			supports_structured = excluded.supports_structured,
			supports_images = excluded.supports_images,
			reasoning = excluded.reasoning,
			reasoning_effort = excluded.reasoning_effort,
			enabled = excluded.enabled,
			updated_at = excluded.updated_at`
	selectModels = `SELECT ` + modelColumns + ` FROM model_catalog ORDER BY provider, model`
)

type ModelCatalogRepo struct {
	store *Store
}

func NewModelCatalogRepo(store *Store) *ModelCatalogRepo {
	return &ModelCatalogRepo{store: store}
}

func (r *ModelCatalogRepo) Upsert(ctx context.Context, override llm.ModelOverride) error {
	info := override.Info
	_, err := execWrite(ctx, r.store.writeFrom(ctx), upsertModel, []any{
		info.Ref.Provider, info.Ref.Model, info.ContextTokens, info.MaxOutputTokens, info.InputUSDPerM,
		info.CachedInputUSDPerM, info.CacheWriteUSDPerM, info.OutputUSDPerM, info.FlexInputUSDPerM,
		info.FlexCachedInputUSDPerM, info.FlexCacheWriteUSDPerM, info.FlexOutputUSDPerM,
		info.RPM, info.TPM, boolInt(info.SupportsStructured), boolInt(info.SupportsImages), boolInt(info.Reasoning),
		string(info.ReasoningEffort),
		boolInt(override.Enabled), formatTime(override.CreatedAt), formatTime(override.UpdatedAt),
	}, nil, "store the model override")
	return err
}

func (r *ModelCatalogRepo) List(ctx context.Context) ([]llm.ModelOverride, error) {
	return selectAll(ctx, r.store.execFrom(ctx), selectModels, nil, scanModelOverride, "list the model overrides")
}

func scanModelOverride(rows *sql.Rows) (llm.ModelOverride, error) {
	var (
		override                                  llm.ModelOverride
		structured, images, reasoning, enabled    int64
		createdAt, updatedAt, effort              string
		contextTokens, maxOutput, requests, count int
	)
	info := &override.Info
	if err := rows.Scan(
		&info.Ref.Provider, &info.Ref.Model, &contextTokens, &maxOutput,
		&info.InputUSDPerM, &info.CachedInputUSDPerM, &info.CacheWriteUSDPerM, &info.OutputUSDPerM,
		&info.FlexInputUSDPerM, &info.FlexCachedInputUSDPerM, &info.FlexCacheWriteUSDPerM, &info.FlexOutputUSDPerM,
		&requests, &count,
		&structured, &images, &reasoning, &effort, &enabled, &createdAt, &updatedAt,
	); err != nil {
		return llm.ModelOverride{}, err
	}

	info.ContextTokens = contextTokens
	info.MaxOutputTokens = maxOutput
	info.RPM = requests
	info.TPM = count
	info.SupportsStructured = structured == 1
	info.SupportsImages = images == 1
	info.Reasoning = reasoning == 1
	info.ReasoningEffort = llm.ReasoningEffort(effort)
	override.Enabled = enabled == 1

	var err error
	if override.CreatedAt, err = parseTime(createdAt); err != nil {
		return llm.ModelOverride{}, err
	}
	if override.UpdatedAt, err = parseTime(updatedAt); err != nil {
		return llm.ModelOverride{}, err
	}
	return override, nil
}
