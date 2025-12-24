-- +goose Up
-- AI Usage Logs for tracking token consumption and costs
CREATE TABLE IF NOT EXISTS ai_usage_logs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    site_id INTEGER NOT NULL,
    operation_type TEXT NOT NULL,
    provider_name TEXT NOT NULL,
    model_name TEXT NOT NULL,
    input_tokens INTEGER NOT NULL DEFAULT 0,
    output_tokens INTEGER NOT NULL DEFAULT 0,
    total_tokens INTEGER NOT NULL DEFAULT 0,
    cost_usd REAL NOT NULL DEFAULT 0,
    duration_ms INTEGER NOT NULL DEFAULT 0,
    success INTEGER NOT NULL DEFAULT 1,
    error_message TEXT,
    metadata TEXT,
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

-- Indexes for efficient aggregation queries
CREATE INDEX IF NOT EXISTS idx_ai_usage_site_id ON ai_usage_logs(site_id);
CREATE INDEX IF NOT EXISTS idx_ai_usage_created_at ON ai_usage_logs(created_at);
CREATE INDEX IF NOT EXISTS idx_ai_usage_operation ON ai_usage_logs(operation_type);
CREATE INDEX IF NOT EXISTS idx_ai_usage_provider ON ai_usage_logs(provider_name);

-- Composite index for common dashboard queries
CREATE INDEX IF NOT EXISTS idx_ai_usage_site_date ON ai_usage_logs(site_id, created_at);
