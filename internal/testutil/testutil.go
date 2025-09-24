package testutil

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"

	"Postulator/internal/config"
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

// TestDB wraps a test database with helper methods
type TestDB struct {
	DB   *sql.DB
	Path string
}

// TestServices contains all services needed for testing
type TestServices struct {
	Repo                 *repository.Repository
	TopicStrategyService *topic_strategy.TopicStrategyService
	GPTService           *gpt.Service
	WordPressService     *wordpress.Service
	PipelineService      *pipeline.Service
}

// NewTestDB creates a new test database
func NewTestDB(t *testing.T) *TestDB {
	// Create temporary database file
	dbPath := t.TempDir() + "/test.db"

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}

	// Initialize schema
	if err := schema.InitSchema(db); err != nil {
		t.Fatalf("Failed to initialize test schema: %v", err)
	}

	return &TestDB{
		DB:   db,
		Path: dbPath,
	}
}

// Close closes the test database
func (tdb *TestDB) Close() error {
	err := tdb.DB.Close()
	if err == nil {
		os.Remove(tdb.Path)
	}
	return err
}

// ClearAllTables clears all tables in the test database
func (tdb *TestDB) ClearAllTables(t *testing.T) {
	tables := []string{
		"topic_usage",
		"site_prompts",
		"prompts",
		"posting_jobs",
		"articles",
		"schedules",
		"site_topics",
		"topics",
		"sites",
		"settings",
	}

	for _, table := range tables {
		query := "DELETE FROM " + table
		if _, err := tdb.DB.Exec(query); err != nil {
			t.Logf("Warning: Failed to clear table %s: %v", table, err)
		}
	}
}

// SetupTestServices creates all services with the test database
func SetupTestServices(t *testing.T, testDB *TestDB) *TestServices {
	repo := repository.NewRepository(testDB.DB)

	// Setup GPT service
	gptConfig := gpt.Config{
		APIKey:    "test-key",
		Model:     "gpt-3.5-turbo",
		MaxTokens: 4000,
		Timeout:   60 * time.Second,
	}
	gptService := gpt.NewService(gptConfig, repo)

	// Setup WordPress service
	wpConfig := wordpress.Config{
		Timeout: 30 * time.Second,
	}
	wpService := wordpress.NewService(wpConfig)

	// Setup topic strategy service
	topicStrategyService := topic_strategy.NewTopicStrategyService(repo)

	// Setup pipeline service
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

	return &TestServices{
		Repo:                 repo,
		TopicStrategyService: topicStrategyService,
		GPTService:           gptService,
		WordPressService:     wpService,
		PipelineService:      pipelineService,
	}
}

// SetupHandlerTest creates a handler with test database for testing
func SetupHandlerTest(t *testing.T) (*handlers.Handler, *TestDB, func()) {
	testDB := NewTestDB(t)
	services := SetupTestServices(t, testDB)

	ctx := context.Background()
	handler := handlers.NewHandler(
		ctx,
		services.GPTService,
		services.WordPressService,
		services.PipelineService,
		services.TopicStrategyService,
		services.Repo,
	)

	cleanup := func() {
		testDB.Close()
	}

	return handler, testDB, cleanup
}

// CreateTestConfig creates a test configuration
func CreateTestConfig() *config.AppConfig {
	return &config.AppConfig{
		DatabasePath: ":memory:",
		LogLevel:     "debug",
	}
}
