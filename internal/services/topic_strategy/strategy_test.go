package topic_strategy

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"Postulator/internal/models"
	"Postulator/internal/repository"
	"Postulator/internal/schema"

	_ "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
)

func setupTestStrategyService(t *testing.T) (*TopicStrategyService, *repository.Repository, func()) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}

	if err := schema.InitSchema(db); err != nil {
		t.Fatalf("Failed to initialize test schema: %v", err)
	}

	repo := repository.NewRepository(db)
	service := NewTopicStrategyService(repo)

	cleanup := func() {
		db.Close()
	}

	return service, repo, cleanup
}

func createTestSiteAndTopics(t *testing.T, repo *repository.Repository, ctx context.Context) (*models.Site, []*models.Topic, []*models.SiteTopic) {
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

	// Create test topics
	topics := []*models.Topic{
		{
			Title:     "Topic 1",
			Keywords:  "keyword1",
			Category:  "Category1",
			Tags:      "tag1",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			Title:     "Topic 2",
			Keywords:  "keyword2",
			Category:  "Category2",
			Tags:      "tag2",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			Title:     "Topic 3",
			Keywords:  "keyword3",
			Category:  "Category3",
			Tags:      "tag3",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	}

	var createdTopics []*models.Topic
	for _, topic := range topics {
		created, err := repo.CreateTopic(ctx, topic)
		if err != nil {
			t.Fatalf("Failed to create test topic: %v", err)
		}
		createdTopics = append(createdTopics, created)
	}

	// Create site-topic associations
	var siteTopics []*models.SiteTopic
	for i, topic := range createdTopics {
		siteTopic := &models.SiteTopic{
			SiteID:        createdSite.ID,
			TopicID:       topic.ID,
			Priority:      i + 1,
			UsageCount:    0,
			RoundRobinPos: i,
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
		}

		created, err := repo.CreateSiteTopic(ctx, siteTopic)
		if err != nil {
			t.Fatalf("Failed to create site topic: %v", err)
		}
		siteTopics = append(siteTopics, created)
	}

	return createdSite, createdTopics, siteTopics
}

func TestUniqueStrategy_SelectTopic(t *testing.T) {
	service, repo, cleanup := setupTestStrategyService(t)
	defer cleanup()

	ctx := context.Background()
	site, _, _ := createTestSiteAndTopics(t, repo, ctx)

	// Test unique strategy
	req := &models.TopicSelectionRequest{
		SiteID:   site.ID,
		Strategy: models.StrategyUnique,
	}

	// First selection should work
	result, err := service.SelectTopicForSite(ctx, req)
	if err != nil {
		t.Errorf("Failed to select topic with unique strategy: %v", err)
		return
	}

	if result == nil {
		t.Error("Result should not be nil")
		return
	}

	if result.Topic == nil {
		t.Error("Selected topic should not be nil")
		return
	}

	if result.SiteTopic == nil {
		t.Error("Site topic should not be nil")
		return
	}

	if result.Strategy != string(models.StrategyUnique) {
		t.Errorf("Expected strategy 'unique', got %s", result.Strategy)
	}

	firstSelectedID := result.Topic.ID

	// Second selection should select a different topic
	result2, err := service.SelectTopicForSite(ctx, req)
	if err != nil {
		t.Errorf("Failed to select second topic with unique strategy: %v", err)
		return
	}

	if result2.Topic.ID == firstSelectedID {
		t.Error("Second selection should return a different topic")
	}

	// Third selection should work (last unused topic)
	_, err = service.SelectTopicForSite(ctx, req)
	if err != nil {
		t.Errorf("Failed to select third topic with unique strategy: %v", err)
		return
	}

	// Fourth selection should fail (no more unused topics)
	_, err = service.SelectTopicForSite(ctx, req)
	if err == nil {
		t.Error("Expected error when no more unused topics available, but got none")
	}
}

func TestRoundRobinStrategy_SelectTopic(t *testing.T) {
	service, repo, cleanup := setupTestStrategyService(t)
	defer cleanup()

	ctx := context.Background()
	site, _, _ := createTestSiteAndTopics(t, repo, ctx)

	// Update site strategy to round_robin
	site.Strategy = "round_robin"
	_, err := repo.UpdateSite(ctx, site)
	if err != nil {
		t.Fatalf("Failed to update site strategy: %v", err)
	}

	req := &models.TopicSelectionRequest{
		SiteID:   site.ID,
		Strategy: models.StrategyRoundRobin,
	}

	// Test multiple selections - should cycle through topics
	var selectedTopicIDs []int64
	for i := 0; i < 6; i++ { // Test two full cycles
		result, err := service.SelectTopicForSite(ctx, req)
		if err != nil {
			t.Errorf("Failed to select topic with round_robin strategy (iteration %d): %v", i, err)
			return
		}

		selectedTopicIDs = append(selectedTopicIDs, result.Topic.ID)

		if result.Strategy != string(models.StrategyRoundRobin) {
			t.Errorf("Expected strategy 'round_robin', got %s", result.Strategy)
		}

		if !result.CanContinue {
			t.Error("Round robin strategy should always be able to continue")
		}
	}

	// Verify cycling behavior - first 3 should match last 3
	if len(selectedTopicIDs) != 6 {
		t.Fatalf("Expected 6 selections, got %d", len(selectedTopicIDs))
	}

	// After two cycles, the pattern should repeat
	for i := 0; i < 3; i++ {
		if selectedTopicIDs[i] != selectedTopicIDs[i+3] {
			t.Errorf("Round robin cycling failed: position %d (%d) != position %d (%d)",
				i, selectedTopicIDs[i], i+3, selectedTopicIDs[i+3])
		}
	}
}

func TestRandomStrategy_SelectTopic(t *testing.T) {
	service, repo, cleanup := setupTestStrategyService(t)
	defer cleanup()

	ctx := context.Background()
	site, _, _ := createTestSiteAndTopics(t, repo, ctx)

	// Update site strategy to random
	site.Strategy = "random"
	_, err := repo.UpdateSite(ctx, site)
	if err != nil {
		t.Fatalf("Failed to update site strategy: %v", err)
	}

	req := &models.TopicSelectionRequest{
		SiteID:   site.ID,
		Strategy: models.StrategyRandom,
	}

	// Test multiple selections
	selectedTopicIDs := make(map[int64]int)
	for i := 0; i < 10; i++ {
		result, err := service.SelectTopicForSite(ctx, req)
		if err != nil {
			t.Errorf("Failed to select topic with random strategy (iteration %d): %v", i, err)
			return
		}

		selectedTopicIDs[result.Topic.ID]++

		if result.Strategy != string(models.StrategyRandom) {
			t.Errorf("Expected strategy 'random', got %s", result.Strategy)
		}

		if !result.CanContinue {
			t.Error("Random strategy should always be able to continue")
		}

		if result.RemainingCount != 3 {
			t.Errorf("Expected remaining count 3, got %d", result.RemainingCount)
		}
	}

	// Verify that all topics were selected at least once (with high probability)
	topicCount := len(selectedTopicIDs)
	if topicCount < 2 {
		t.Errorf("Random strategy should select from multiple topics, only selected from %d topics", topicCount)
	}
}

func TestRandomAllStrategy_SelectTopic(t *testing.T) {
	service, repo, cleanup := setupTestStrategyService(t)
	defer cleanup()

	ctx := context.Background()
	site, _, _ := createTestSiteAndTopics(t, repo, ctx)

	// Create additional topics not associated with the site
	extraTopics := []*models.Topic{
		{
			Title:     "Extra Topic 1",
			Keywords:  "extra1",
			Category:  "Extra",
			Tags:      "extra",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			Title:     "Extra Topic 2",
			Keywords:  "extra2",
			Category:  "Extra",
			Tags:      "extra",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	}

	for _, topic := range extraTopics {
		_, err := repo.CreateTopic(ctx, topic)
		if err != nil {
			t.Fatalf("Failed to create extra topic: %v", err)
		}
	}

	req := &models.TopicSelectionRequest{
		SiteID:   site.ID,
		Strategy: models.StrategyRandomAll,
	}

	// Test multiple selections
	selectedTopicIDs := make(map[int64]int)
	for i := 0; i < 15; i++ {
		result, err := service.SelectTopicForSite(ctx, req)
		if err != nil {
			t.Errorf("Failed to select topic with random_all strategy (iteration %d): %v", i, err)
			return
		}

		selectedTopicIDs[result.Topic.ID]++

		if result.Strategy != string(models.StrategyRandomAll) {
			t.Errorf("Expected strategy 'random_all', got %s", result.Strategy)
		}

		if !result.CanContinue {
			t.Error("Random_all strategy should always be able to continue")
		}

		if result.RemainingCount != 5 { // 3 site topics + 2 extra topics
			t.Errorf("Expected remaining count 5, got %d", result.RemainingCount)
		}
	}

	// Verify that topics from both site-associated and non-site-associated were selected
	if len(selectedTopicIDs) < 4 {
		t.Errorf("Random_all strategy should select from all topics in system, only selected from %d topics", len(selectedTopicIDs))
	}
}

func TestTopicStrategyService_CanContinueWithStrategy(t *testing.T) {
	service, repo, cleanup := setupTestStrategyService(t)
	defer cleanup()

	ctx := context.Background()
	site, _, _ := createTestSiteAndTopics(t, repo, ctx)

	tests := []struct {
		name     string
		strategy string
		expected bool
	}{
		{
			name:     "Unique strategy with unused topics",
			strategy: "unique",
			expected: true,
		},
		{
			name:     "Round robin strategy",
			strategy: "round_robin",
			expected: true,
		},
		{
			name:     "Random strategy",
			strategy: "random",
			expected: true,
		},
		{
			name:     "Random all strategy",
			strategy: "random_all",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			canContinue, err := service.CanContinueWithStrategy(ctx, site.ID, tt.strategy)
			if err != nil {
				t.Errorf("Failed to check CanContinueWithStrategy: %v", err)
				return
			}

			if canContinue != tt.expected {
				t.Errorf("Expected CanContinueWithStrategy %v for strategy %s, got %v",
					tt.expected, tt.strategy, canContinue)
			}
		})
	}
}

func TestTopicStrategyService_GetTopicStatsForSite(t *testing.T) {
	service, repo, cleanup := setupTestStrategyService(t)
	defer cleanup()

	ctx := context.Background()
	site, _, siteTopics := createTestSiteAndTopics(t, repo, ctx)

	// Use one topic to create usage statistics
	err := repo.UpdateSiteTopicUsage(ctx, siteTopics[0].ID, "unique")
	if err != nil {
		t.Fatalf("Failed to update topic usage: %v", err)
	}

	stats, err := service.GetTopicStatsForSite(ctx, site.ID)
	if err != nil {
		t.Errorf("Failed to get topic stats: %v", err)
		return
	}

	if stats == nil {
		t.Error("Stats should not be nil")
		return
	}

	if stats.SiteID != site.ID {
		t.Errorf("Expected site ID %d, got %d", site.ID, stats.SiteID)
	}

	if stats.TotalTopics != 3 {
		t.Errorf("Expected total topics 3, got %d", stats.TotalTopics)
	}

	if stats.UsedTopics != 1 {
		t.Errorf("Expected used topics 1, got %d", stats.UsedTopics)
	}

	if stats.UnusedTopics != 2 {
		t.Errorf("Expected unused topics 2, got %d", stats.UnusedTopics)
	}
}
