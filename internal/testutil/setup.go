package testutil

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"Postulator/internal/handlers"
	"Postulator/internal/repository"
	"Postulator/internal/schema"
	"Postulator/internal/services/gpt"
	"Postulator/internal/services/pipeline"
	"Postulator/internal/services/topic_strategy"
	"Postulator/internal/services/wordpress"

	_ "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
)

// setupHandlerTest creates a handler with test database for testing
func setupHandlerTest(t *testing.T) (*handlers.Handler, *sql.DB, func()) {
	// Create temporary in-memory database
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}

	// Initialize schema
	if err := schema.InitSchema(db); err != nil {
		t.Fatalf("Failed to initialize test schema: %v", err)
	}

	// Create repository
	repo := repository.NewRepository(db)

	// Setup services
	gptConfig := gpt.Config{
		APIKey:    "test-key",
		Model:     "gpt-3.5-turbo",
		MaxTokens: 4000,
		Timeout:   60 * time.Second,
	}
	gptService := gpt.NewService(gptConfig, repo)

	wpConfig := wordpress.Config{
		Timeout: 30 * time.Second,
	}
	wpService := wordpress.NewService(wpConfig)

	topicStrategyService := topic_strategy.NewTopicStrategyService(repo)

	pipelineConfig := pipeline.Config{
		MaxWorkers:       5,
		JobTimeout:       900 * time.Second,
		RetryCount:       3,
		RetryDelay:       3 * time.Second,
		MinContentWords:  500,
		MaxDailyPosts:    10,
		WordPressTimeout: 30 * time.Second,
		GPTTimeout:       60 * time.Second,
	}
	ctx := context.Background()
	pipelineService := pipeline.NewService(pipelineConfig, repo, gptService, wpService, topicStrategyService, ctx)

	// Create handler
	handler := handlers.NewHandler(
		ctx,
		gptService,
		wpService,
		pipelineService,
		topicStrategyService,
		repo,
	)

	cleanup := func() {
		db.Close()
	}

	return handler, db, cleanup
}

// ClearAllTables clears all tables in the test database
func ClearAllTables(t *testing.T, db *sql.DB) {
	tables := []string{
		"topic_usage", "site_prompts", "prompts", "posting_jobs",
		"articles", "schedules", "site_topics", "topics", "sites", "settings",
	}

	for _, table := range tables {
		if _, err := db.Exec("DELETE FROM " + table); err != nil {
			t.Logf("Warning: Failed to clear table %s: %v", table, err)
		}
	}
}
