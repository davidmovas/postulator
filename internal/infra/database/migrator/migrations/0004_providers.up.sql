-- +goose Up
-- =========================================================================
-- AI PROVIDERS
-- =========================================================================

CREATE TABLE ai_providers (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL UNIQUE,
    provider TEXT NOT NULL,
    model TEXT NOT NULL,
    api_key TEXT NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT 1,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CHECK (provider IN ('openai', 'anthropic', 'google'))
);

CREATE INDEX idx_ai_providers_active ON ai_providers(is_active);

-- =========================================================================
-- PROMPTS
-- =========================================================================

CREATE TABLE prompts (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    system_prompt TEXT NOT NULL,
    user_prompt TEXT NOT NULL,
    placeholders TEXT,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- +goose Down
DROP TABLE IF EXISTS prompts;

DROP INDEX IF EXISTS idx_ai_providers_active;
DROP TABLE IF EXISTS ai_providers;
