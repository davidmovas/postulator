package pipeline

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"Postulator/internal/dto"
	"Postulator/internal/models"
	"Postulator/internal/repository"
	"Postulator/internal/schema"
	"Postulator/internal/services/gpt"
	"Postulator/internal/services/topic_strategy"
	"Postulator/internal/services/wordpress"

	_ "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
)

func setupTestPipelineService(t *testing.T) (*Service, *repository.Repository, func()) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}

	if err := schema.InitSchema(db); err != nil {
		t.Fatalf("Failed to initialize test schema: %v", err)
	}

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

	pipelineConfig := Config{
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
	service := NewService(pipelineConfig, repo, gptService, wpService, topicStrategyService, ctx)

	cleanup := func() {
		db.Close()
	}

	return service, repo, cleanup
}

func createTestData(t *testing.T, repo *repository.Repository, ctx context.Context) (*models.Site, *models.Topic, *models.SiteTopic, *models.Prompt) {
	// Create test site
	site := &models.Site{
		Name:      "Test Site",
		URL:       "https://test.com",
		Username:  "testuser",
		Password:  "testpass",
		IsActive:  true,
		Strategy:  "unique",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	createdSite, err := repo.CreateSite(ctx, site)
	if err != nil {
		t.Fatalf("Failed to create test site: %v", err)
	}

	// Create test topic
	topic := &models.Topic{
		Title:     "Test Topic",
		Keywords:  "test,keywords",
		Category:  "Technology",
		Tags:      "test,tags",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	createdTopic, err := repo.CreateTopic(ctx, topic)
	if err != nil {
		t.Fatalf("Failed to create test topic: %v", err)
	}

	// Create site-topic association
	siteTopic := &models.SiteTopic{
		SiteID:        createdSite.ID,
		TopicID:       createdTopic.ID,
		Priority:      1,
		UsageCount:    0,
		RoundRobinPos: 0,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	createdSiteTopic, err := repo.CreateSiteTopic(ctx, siteTopic)
	if err != nil {
		t.Fatalf("Failed to create site topic: %v", err)
	}

	// Get default prompt (should exist from schema initialization)
	prompt, err := repo.GetDefaultPrompt(ctx)
	if err != nil {
		t.Fatalf("Failed to get default prompt: %v", err)
	}

	return createdSite, createdTopic, createdSiteTopic, prompt
}

func TestPipelineService_GenerateAndPublish(t *testing.T) {
	service, repo, cleanup := setupTestPipelineService(t)
	defer cleanup()

	ctx := context.Background()
	site, topic, _, _ := createTestData(t, repo, ctx)

	tests := []struct {
		name    string
		request dto.GeneratePublishRequest
		wantErr bool
	}{
		{
			name: "Valid request with specified topic",
			request: dto.GeneratePublishRequest{
				SiteID:   site.ID,
				TopicID:  &topic.ID,
				Title:    "Custom Title",
				Tone:     "professional",
				Style:    "informative",
				MinWords: 800,
			},
			wantErr: false,
		},
		{
			name: "Valid request with strategy selection",
			request: dto.GeneratePublishRequest{
				SiteID:   site.ID,
				Strategy: "unique",
				MinWords: 500,
			},
			wantErr: false,
		},
		{
			name: "Valid request with minimal data",
			request: dto.GeneratePublishRequest{
				SiteID: site.ID,
			},
			wantErr: false,
		},
		{
			name: "Invalid site ID",
			request: dto.GeneratePublishRequest{
				SiteID: 0,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := service.GenerateAndPublish(ctx, tt.request)

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if result == nil {
				t.Error("Result should not be nil")
				return
			}

			if result.ID <= 0 {
				t.Error("Article ID should be positive")
			}

			if result.SiteID != tt.request.SiteID {
				t.Errorf("Expected site ID %d, got %d", tt.request.SiteID, result.SiteID)
			}

			if result.Status != "published" {
				t.Errorf("Expected status 'published', got %s", result.Status)
			}

			if result.WordPressID <= 0 {
				t.Error("WordPress ID should be positive")
			}

			// Verify article was saved to database
			savedArticle, err := repo.GetArticle(ctx, result.ID)
			if err != nil {
				t.Errorf("Failed to retrieve saved article: %v", err)
			}

			if savedArticle.Title != result.Title {
				t.Errorf("Expected saved title %s, got %s", result.Title, savedArticle.Title)
			}
		})
	}
}

func TestPipelineService_GenerateAndPublish_Idempotency(t *testing.T) {
	service, repo, cleanup := setupTestPipelineService(t)
	defer cleanup()

	ctx := context.Background()
	site, topic, _, _ := createTestData(t, repo, ctx)

	request := dto.GeneratePublishRequest{
		SiteID:  site.ID,
		TopicID: &topic.ID,
		Title:   "Idempotency Test",
	}

	// First request
	result1, err := service.GenerateAndPublish(ctx, request)
	if err != nil {
		t.Fatalf("First request failed: %v", err)
	}

	// Second request with same parameters should return same result (idempotency)
	result2, err := service.GenerateAndPublish(ctx, request)
	if err != nil {
		t.Fatalf("Second request failed: %v", err)
	}

	if result1.ID != result2.ID {
		t.Errorf("Expected same article ID for idempotent requests, got %d and %d", result1.ID, result2.ID)
	}

	if result1.Title != result2.Title {
		t.Errorf("Expected same title for idempotent requests, got %s and %s", result1.Title, result2.Title)
	}
}

func TestPipelineService_CreatePublishJob(t *testing.T) {
	service, repo, cleanup := setupTestPipelineService(t)
	defer cleanup()

	ctx := context.Background()
	site, topic, _, _ := createTestData(t, repo, ctx)

	tests := []struct {
		name    string
		request dto.GeneratePublishRequest
		wantErr bool
	}{
		{
			name: "Valid job creation",
			request: dto.GeneratePublishRequest{
				SiteID:   site.ID,
				TopicID:  &topic.ID,
				Strategy: "unique",
			},
			wantErr: false,
		},
		{
			name: "Invalid site ID",
			request: dto.GeneratePublishRequest{
				SiteID: 0,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := service.CreatePublishJob(ctx, tt.request)

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if result == nil {
				t.Error("Result should not be nil")
				return
			}

			if result.ID <= 0 {
				t.Error("Job ID should be positive")
			}

			if result.SiteID != tt.request.SiteID {
				t.Errorf("Expected site ID %d, got %d", tt.request.SiteID, result.SiteID)
			}

			if result.Status != "pending" {
				t.Errorf("Expected status 'pending', got %s", result.Status)
			}

			if result.Type != "manual" {
				t.Errorf("Expected type 'manual', got %s", result.Type)
			}

			// Verify job was saved to database
			savedJob, err := repo.GetJob(ctx, result.ID)
			if err != nil {
				t.Errorf("Failed to retrieve saved job: %v", err)
			}

			if savedJob.SiteID != tt.request.SiteID {
				t.Errorf("Expected saved site ID %d, got %d", tt.request.SiteID, savedJob.SiteID)
			}
		})
	}
}

func TestPipelineService_GetJobs(t *testing.T) {
	service, repo, cleanup := setupTestPipelineService(t)
	defer cleanup()

	ctx := context.Background()
	site, _, _, _ := createTestData(t, repo, ctx)

	// Create test jobs
	testJobs := []dto.GeneratePublishRequest{
		{SiteID: site.ID, Strategy: "unique"},
		{SiteID: site.ID, Strategy: "random"},
		{SiteID: site.ID, Strategy: "round_robin"},
	}

	for _, jobReq := range testJobs {
		_, err := service.CreatePublishJob(ctx, jobReq)
		if err != nil {
			t.Fatalf("Failed to create test job: %v", err)
		}
	}

	tests := []struct {
		name    string
		request dto.PaginationRequest
		wantErr bool
	}{
		{
			name: "Get all jobs",
			request: dto.PaginationRequest{
				Page:  1,
				Limit: 10,
			},
			wantErr: false,
		},
		{
			name: "Get jobs with pagination",
			request: dto.PaginationRequest{
				Page:  1,
				Limit: 2,
			},
			wantErr: false,
		},
		{
			name:    "Get jobs with default pagination",
			request: dto.PaginationRequest{},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := service.GetJobs(ctx, tt.request)

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if result == nil {
				t.Error("Result should not be nil")
				return
			}

			if result.Jobs == nil {
				t.Error("Jobs list should not be nil")
				return
			}

			if result.Pagination == nil {
				t.Error("Pagination should not be nil")
				return
			}

			expectedCount := len(testJobs)
			if tt.request.Limit > 0 && tt.request.Limit < expectedCount {
				expectedCount = tt.request.Limit
			}

			if len(result.Jobs) > expectedCount {
				t.Errorf("Expected at most %d jobs, got %d", expectedCount, len(result.Jobs))
			}

			if result.Pagination.Total != int64(len(testJobs)) {
				t.Errorf("Expected total count %d, got %d", len(testJobs), result.Pagination.Total)
			}
		})
	}
}

func TestPipelineService_GetJob(t *testing.T) {
	service, repo, cleanup := setupTestPipelineService(t)
	defer cleanup()

	ctx := context.Background()
	site, _, _, _ := createTestData(t, repo, ctx)

	// Create a test job
	jobReq := dto.GeneratePublishRequest{
		SiteID:   site.ID,
		Strategy: "unique",
	}

	createdJob, err := service.CreatePublishJob(ctx, jobReq)
	if err != nil {
		t.Fatalf("Failed to create test job: %v", err)
	}

	tests := []struct {
		name    string
		jobID   int64
		wantErr bool
	}{
		{
			name:    "Valid job ID",
			jobID:   createdJob.ID,
			wantErr: false,
		},
		{
			name:    "Invalid job ID",
			jobID:   0,
			wantErr: true,
		},
		{
			name:    "Non-existent job ID",
			jobID:   999,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := service.GetJob(ctx, tt.jobID)

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if result == nil {
				t.Error("Result should not be nil")
				return
			}

			if result.ID != createdJob.ID {
				t.Errorf("Expected job ID %d, got %d", createdJob.ID, result.ID)
			}

			if result.SiteID != createdJob.SiteID {
				t.Errorf("Expected site ID %d, got %d", createdJob.SiteID, result.SiteID)
			}
		})
	}
}

func TestPipelineService_ProcessPromptPlaceholders(t *testing.T) {
	service, repo, cleanup := setupTestPipelineService(t)
	defer cleanup()

	ctx := context.Background()
	site, topic, _, _ := createTestData(t, repo, ctx)

	tests := []struct {
		name     string
		prompt   string
		request  dto.GeneratePublishRequest
		expected map[string]string // key: placeholder, value: expected replacement
	}{
		{
			name:   "Site placeholders",
			prompt: "Site: {site_name} at {site_url}",
			request: dto.GeneratePublishRequest{
				SiteID: site.ID,
			},
			expected: map[string]string{
				"{site_name}": site.Name,
				"{site_url}":  site.URL,
			},
		},
		{
			name:   "Topic placeholders",
			prompt: "Topic: {topic_title} with {keywords} in {category}",
			request: dto.GeneratePublishRequest{
				SiteID: site.ID,
			},
			expected: map[string]string{
				"{topic_title}": topic.Title,
				"{keywords}":    topic.Keywords,
				"{category}":    topic.Category,
			},
		},
		{
			name:   "Request placeholders",
			prompt: "Style: {style}, Tone: {tone}, Words: {min_words}",
			request: dto.GeneratePublishRequest{
				SiteID:   site.ID,
				Tone:     "casual",
				Style:    "conversational",
				MinWords: 1000,
			},
			expected: map[string]string{
				"{tone}":      "casual",
				"{style}":     "conversational",
				"{min_words}": "1000",
			},
		},
		{
			name:   "Default values",
			prompt: "Defaults: {tone}, {style}, {min_words}",
			request: dto.GeneratePublishRequest{
				SiteID: site.ID,
			},
			expected: map[string]string{
				"{tone}":      "professional",
				"{style}":     "informative",
				"{min_words}": "800",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.processPromptPlaceholders(tt.prompt, site, topic, tt.request)

			for placeholder, expectedValue := range tt.expected {
				if !containsString(result, expectedValue) {
					t.Errorf("Expected result to contain %s (from %s), but got: %s",
						expectedValue, placeholder, result)
				}
			}
		})
	}
}

func TestPipelineService_GenerateSlug(t *testing.T) {
	service, _, cleanup := setupTestPipelineService(t)
	defer cleanup()

	tests := []struct {
		name     string
		title    string
		expected string
	}{
		{
			name:     "Simple title",
			title:    "Hello World",
			expected: "hello-world",
		},
		{
			name:     "Title with special characters",
			title:    "Hello, World! How are you?",
			expected: "hello-world-how-are-you",
		},
		{
			name:     "Title with underscores",
			title:    "hello_world_test",
			expected: "hello-world-test",
		},
		{
			name:     "Title with numbers",
			title:    "Test 123 Article",
			expected: "test-123-article",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.generateSlug(tt.title)

			if result != tt.expected {
				t.Errorf("Expected slug %s, got %s", tt.expected, result)
			}
		})
	}
}

// Helper function to check if string contains substring
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) &&
		(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
			findInString(s, substr)))
}

func findInString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
