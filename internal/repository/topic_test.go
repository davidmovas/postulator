package repository

import (
	"context"
	"testing"
	"time"

	"Postulator/internal/models"
	"Postulator/internal/testhelpers"
)

func TestTopicRepository_GetTopics(t *testing.T) {
	testDB := testhelpers.SetupTestDB(t)
	defer testDB.Close()

	repo := NewRepository(testDB.DB)
	testDB.InsertTestData(t)

	ctx := context.Background()

	t.Run("GetTopics with pagination", func(t *testing.T) {
		result, err := repo.GetTopics(ctx, 2, 0)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if len(result.Data) != 2 {
			t.Fatalf("Expected 2 topics, got %d", len(result.Data))
		}

		if result.Total != 3 {
			t.Fatalf("Expected total count 3, got %d", result.Total)
		}

		if result.Limit != 2 {
			t.Fatalf("Expected limit 2, got %d", result.Limit)
		}

		if result.Offset != 0 {
			t.Fatalf("Expected offset 0, got %d", result.Offset)
		}
	})

	t.Run("GetTopics with offset", func(t *testing.T) {
		result, err := repo.GetTopics(ctx, 1, 2)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if len(result.Data) != 1 {
			t.Fatalf("Expected 1 topic, got %d", len(result.Data))
		}

		if result.Offset != 2 {
			t.Fatalf("Expected offset 2, got %d", result.Offset)
		}
	})

	t.Run("GetTopics empty database", func(t *testing.T) {
		testDB.ClearAllTables(t)

		result, err := repo.GetTopics(ctx, 10, 0)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if len(result.Data) != 0 {
			t.Fatalf("Expected 0 topics, got %d", len(result.Data))
		}

		if result.Total != 0 {
			t.Fatalf("Expected total count 0, got %d", result.Total)
		}
	})
}

func TestTopicRepository_GetTopic(t *testing.T) {
	testDB := testhelpers.SetupTestDB(t)
	defer testDB.Close()

	repo := NewRepository(testDB.DB)
	testDB.InsertTestData(t)

	ctx := context.Background()

	t.Run("GetTopic existing", func(t *testing.T) {
		topic, err := repo.GetTopic(ctx, 1)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if topic.ID != 1 {
			t.Fatalf("Expected topic ID 1, got %d", topic.ID)
		}

		if topic.Title != "AI Technology" {
			t.Fatalf("Expected topic title 'AI Technology', got %s", topic.Title)
		}

		if topic.Keywords != "ai,ml,tech" {
			t.Fatalf("Expected keywords 'ai,ml,tech', got %s", topic.Keywords)
		}

		if topic.Category != "Technology" {
			t.Fatalf("Expected category 'Technology', got %s", topic.Category)
		}

		if !topic.IsActive {
			t.Fatal("Expected topic to be active")
		}
	})

	t.Run("GetTopic non-existing", func(t *testing.T) {
		_, err := repo.GetTopic(ctx, 999)
		if err == nil {
			t.Fatal("Expected error for non-existing topic, got nil")
		}
	})
}

func TestTopicRepository_GetTopicsBySiteID(t *testing.T) {
	testDB := testhelpers.SetupTestDB(t)
	defer testDB.Close()

	repo := NewRepository(testDB.DB)
	testDB.InsertTestData(t)

	ctx := context.Background()

	t.Run("GetTopicsBySiteID existing site", func(t *testing.T) {
		result, err := repo.GetTopicsBySiteID(ctx, 1, 10, 0)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if len(result.Data) != 2 {
			t.Fatalf("Expected 2 topics for site 1, got %d", len(result.Data))
		}

		if result.Total != 2 {
			t.Fatalf("Expected total count 2, got %d", result.Total)
		}

		// Check that we get the right topics (should be topics 1 and 2 based on test data)
		foundTopic1, foundTopic2 := false, false
		for _, topic := range result.Data {
			if topic.ID == 1 && topic.Title == "AI Technology" {
				foundTopic1 = true
			}
			if topic.ID == 2 && topic.Title == "Web Development" {
				foundTopic2 = true
			}
		}

		if !foundTopic1 || !foundTopic2 {
			t.Fatal("Expected to find both AI Technology and Web Development topics")
		}
	})

	t.Run("GetTopicsBySiteID with pagination", func(t *testing.T) {
		result, err := repo.GetTopicsBySiteID(ctx, 1, 1, 0)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if len(result.Data) != 1 {
			t.Fatalf("Expected 1 topic with limit 1, got %d", len(result.Data))
		}

		if result.Total != 2 {
			t.Fatalf("Expected total count 2, got %d", result.Total)
		}
	})

	t.Run("GetTopicsBySiteID non-existing site", func(t *testing.T) {
		result, err := repo.GetTopicsBySiteID(ctx, 999, 10, 0)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if len(result.Data) != 0 {
			t.Fatalf("Expected 0 topics for non-existing site, got %d", len(result.Data))
		}

		if result.Total != 0 {
			t.Fatalf("Expected total count 0, got %d", result.Total)
		}
	})

	t.Run("GetTopicsBySiteID only active associations", func(t *testing.T) {
		// Deactivate one site_topic association
		_, err := testDB.DB.Exec("UPDATE site_topics SET is_active = 0 WHERE site_id = 1 AND topic_id = 1")
		if err != nil {
			t.Fatalf("Failed to deactivate site_topic: %v", err)
		}

		result, err := repo.GetTopicsBySiteID(ctx, 1, 10, 0)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		// Should only get 1 topic now (topic 2)
		if len(result.Data) != 1 {
			t.Fatalf("Expected 1 active topic, got %d", len(result.Data))
		}

		if result.Data[0].ID != 2 {
			t.Fatalf("Expected topic ID 2, got %d", result.Data[0].ID)
		}
	})
}

func TestTopicRepository_CreateTopic(t *testing.T) {
	testDB := testhelpers.SetupTestDB(t)
	defer testDB.Close()

	repo := NewRepository(testDB.DB)

	ctx := context.Background()

	t.Run("CreateTopic success", func(t *testing.T) {
		now := time.Now()
		topic := &models.Topic{
			Title:     "New Test Topic",
			Keywords:  "test,new,topic",
			Category:  "Testing",
			Tags:      "test,new",
			IsActive:  true,
			CreatedAt: now,
			UpdatedAt: now,
		}

		createdTopic, err := repo.CreateTopic(ctx, topic)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if createdTopic.ID == 0 {
			t.Fatal("Expected topic ID to be set")
		}

		if createdTopic.Title != topic.Title {
			t.Fatalf("Expected title %s, got %s", topic.Title, createdTopic.Title)
		}

		// Verify topic was actually saved
		savedTopic, err := repo.GetTopic(ctx, createdTopic.ID)
		if err != nil {
			t.Fatalf("Failed to retrieve created topic: %v", err)
		}

		if savedTopic.Title != topic.Title {
			t.Fatalf("Expected saved title %s, got %s", topic.Title, savedTopic.Title)
		}

		if savedTopic.Keywords != "test,new,topic" {
			t.Fatalf("Expected keywords 'test,new,topic', got %s", savedTopic.Keywords)
		}
	})

	t.Run("CreateTopic with minimal data", func(t *testing.T) {
		topic := &models.Topic{
			Title:     "Minimal Topic",
			IsActive:  false,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		createdTopic, err := repo.CreateTopic(ctx, topic)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if createdTopic.ID == 0 {
			t.Fatal("Expected topic ID to be set")
		}

		if createdTopic.IsActive != false {
			t.Fatal("Expected IsActive to be false")
		}
	})
}

func TestTopicRepository_UpdateTopic(t *testing.T) {
	testDB := testhelpers.SetupTestDB(t)
	defer testDB.Close()

	repo := NewRepository(testDB.DB)
	testDB.InsertTestData(t)

	ctx := context.Background()

	t.Run("UpdateTopic success", func(t *testing.T) {
		// Get existing topic
		topic, err := repo.GetTopic(ctx, 1)
		if err != nil {
			t.Fatalf("Failed to get test topic: %v", err)
		}

		// Update fields
		topic.Title = "Updated AI Technology"
		topic.Keywords = "updated,ai,ml"
		topic.Category = "Updated Technology"
		topic.IsActive = false
		topic.UpdatedAt = time.Now()

		updatedTopic, err := repo.UpdateTopic(ctx, topic)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if updatedTopic.Title != "Updated AI Technology" {
			t.Fatalf("Expected updated title, got %s", updatedTopic.Title)
		}

		if updatedTopic.Keywords != "updated,ai,ml" {
			t.Fatalf("Expected updated keywords, got %s", updatedTopic.Keywords)
		}

		// Verify update persisted
		savedTopic, err := repo.GetTopic(ctx, 1)
		if err != nil {
			t.Fatalf("Failed to retrieve updated topic: %v", err)
		}

		if savedTopic.Title != "Updated AI Technology" {
			t.Fatalf("Expected persisted title 'Updated AI Technology', got %s", savedTopic.Title)
		}

		if savedTopic.IsActive != false {
			t.Fatal("Expected IsActive to be false")
		}
	})

	t.Run("UpdateTopic non-existing", func(t *testing.T) {
		topic := &models.Topic{
			ID:        999,
			Title:     "Non-existing",
			IsActive:  true,
			UpdatedAt: time.Now(),
		}

		_, err := repo.UpdateTopic(ctx, topic)
		// Update should not fail even if no rows affected
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
	})
}

func TestTopicRepository_DeleteTopic(t *testing.T) {
	testDB := testhelpers.SetupTestDB(t)
	defer testDB.Close()

	repo := NewRepository(testDB.DB)
	testDB.InsertTestData(t)

	ctx := context.Background()

	t.Run("DeleteTopic success", func(t *testing.T) {
		// Verify topic exists
		_, err := repo.GetTopic(ctx, 1)
		if err != nil {
			t.Fatalf("Topic should exist before deletion: %v", err)
		}

		// Delete topic
		err = repo.DeleteTopic(ctx, 1)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		// Verify topic is deleted
		_, err = repo.GetTopic(ctx, 1)
		if err == nil {
			t.Fatal("Expected error when getting deleted topic")
		}
	})

	t.Run("DeleteTopic non-existing", func(t *testing.T) {
		err := repo.DeleteTopic(ctx, 999)
		if err != nil {
			t.Fatalf("Expected no error for non-existing topic, got %v", err)
		}
	})

	t.Run("DeleteTopic cascades to site_topics", func(t *testing.T) {
		// Verify site_topic associations exist
		var count int
		err := testDB.DB.QueryRow("SELECT COUNT(*) FROM site_topics WHERE topic_id = 2").Scan(&count)
		if err != nil {
			t.Fatalf("Failed to count site_topics: %v", err)
		}
		if count == 0 {
			t.Fatal("Expected site_topic associations to exist before deletion")
		}

		// Delete topic
		err = repo.DeleteTopic(ctx, 2)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		// Verify site_topic associations are also deleted (CASCADE)
		err = testDB.DB.QueryRow("SELECT COUNT(*) FROM site_topics WHERE topic_id = 2").Scan(&count)
		if err != nil {
			t.Fatalf("Failed to count site_topics after deletion: %v", err)
		}
		if count != 0 {
			t.Fatal("Expected site_topic associations to be deleted due to CASCADE")
		}
	})
}

func TestTopicRepository_ActivateDeactivateTopic(t *testing.T) {
	testDB := testhelpers.SetupTestDB(t)
	defer testDB.Close()

	repo := NewRepository(testDB.DB)
	testDB.InsertTestData(t)

	ctx := context.Background()

	t.Run("DeactivateTopic", func(t *testing.T) {
		// Verify topic is active initially
		topic, err := repo.GetTopic(ctx, 1)
		if err != nil {
			t.Fatalf("Failed to get test topic: %v", err)
		}

		if !topic.IsActive {
			t.Fatal("Topic should be active initially")
		}

		// Deactivate topic
		err = repo.DeactivateTopic(ctx, 1)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		// Verify deactivation
		topic, err = repo.GetTopic(ctx, 1)
		if err != nil {
			t.Fatalf("Failed to get topic after deactivation: %v", err)
		}

		if topic.IsActive {
			t.Fatal("Topic should be inactive after deactivation")
		}
	})

	t.Run("ActivateTopic", func(t *testing.T) {
		// Use topic 3 which is inactive in test data
		topic, err := repo.GetTopic(ctx, 3)
		if err != nil {
			t.Fatalf("Failed to get test topic: %v", err)
		}

		if topic.IsActive {
			t.Fatal("Topic should be inactive initially")
		}

		// Activate topic
		err = repo.ActivateTopic(ctx, 3)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		// Verify activation
		topic, err = repo.GetTopic(ctx, 3)
		if err != nil {
			t.Fatalf("Failed to get topic after activation: %v", err)
		}

		if !topic.IsActive {
			t.Fatal("Topic should be active after activation")
		}
	})
}

func TestTopicRepository_GetActiveTopics(t *testing.T) {
	testDB := testhelpers.SetupTestDB(t)
	defer testDB.Close()

	repo := NewRepository(testDB.DB)
	testDB.InsertTestData(t)

	ctx := context.Background()

	t.Run("GetActiveTopics success", func(t *testing.T) {
		topics, err := repo.GetActiveTopics(ctx)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		// Should get 2 active topics from test data (topics 1 and 2)
		if len(topics) != 2 {
			t.Fatalf("Expected 2 active topics, got %d", len(topics))
		}

		// Verify all returned topics are active
		for _, topic := range topics {
			if !topic.IsActive {
				t.Fatalf("Expected all topics to be active, but topic %d is inactive", topic.ID)
			}
		}

		// Verify correct topics are returned (sorted by title)
		if topics[0].Title != "AI Technology" {
			t.Fatalf("Expected first topic to be 'AI Technology', got %s", topics[0].Title)
		}
		if topics[1].Title != "Web Development" {
			t.Fatalf("Expected second topic to be 'Web Development', got %s", topics[1].Title)
		}
	})

	t.Run("GetActiveTopics after deactivation", func(t *testing.T) {
		// Deactivate one topic
		err := repo.DeactivateTopic(ctx, 1)
		if err != nil {
			t.Fatalf("Failed to deactivate topic: %v", err)
		}

		topics, err := repo.GetActiveTopics(ctx)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		// Should get 1 active topic now
		if len(topics) != 1 {
			t.Fatalf("Expected 1 active topic, got %d", len(topics))
		}

		if topics[0].Title != "Web Development" {
			t.Fatalf("Expected remaining topic to be 'Web Development', got %s", topics[0].Title)
		}
	})

	t.Run("GetActiveTopics empty result", func(t *testing.T) {
		// Deactivate all topics
		err := repo.DeactivateTopic(ctx, 2)
		if err != nil {
			t.Fatalf("Failed to deactivate topic: %v", err)
		}

		topics, err := repo.GetActiveTopics(ctx)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if len(topics) != 0 {
			t.Fatalf("Expected 0 active topics, got %d", len(topics))
		}
	})
}

func TestTopicRepository_EdgeCases(t *testing.T) {
	testDB := testhelpers.SetupTestDB(t)
	defer testDB.Close()

	repo := NewRepository(testDB.DB)

	ctx := context.Background()

	t.Run("Large pagination values", func(t *testing.T) {
		result, err := repo.GetTopics(ctx, 1000, 0)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if len(result.Data) != 0 {
			t.Fatalf("Expected 0 topics in empty DB, got %d", len(result.Data))
		}
	})

	t.Run("Negative pagination values", func(t *testing.T) {
		result, err := repo.GetTopics(ctx, -1, -1)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if result == nil {
			t.Fatal("Expected result not to be nil")
		}
	})

	t.Run("Very large topic ID", func(t *testing.T) {
		_, err := repo.GetTopic(ctx, 9223372036854775807) // Max int64
		if err == nil {
			t.Fatal("Expected error for very large ID")
		}
	})

	t.Run("Empty string fields", func(t *testing.T) {
		topic := &models.Topic{
			Title:     "", // Empty title
			Keywords:  "",
			Category:  "",
			Tags:      "",
			IsActive:  true,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		// This should work as empty strings are valid
		createdTopic, err := repo.CreateTopic(ctx, topic)
		if err != nil {
			t.Fatalf("Expected no error for empty strings, got %v", err)
		}

		if createdTopic.ID == 0 {
			t.Fatal("Expected topic ID to be set")
		}
	})
}
