package repository

import (
	"context"
	"testing"
	"time"

	"Postulator/internal/models"
	"Postulator/internal/testhelpers"
)

func TestSiteTopicRepository_CreateSiteTopic(t *testing.T) {
	testDB := testhelpers.SetupTestDB(t)
	defer testDB.Close()

	repo := NewRepository(testDB.DB)
	testDB.InsertTestData(t)

	ctx := context.Background()

	t.Run("CreateSiteTopic success", func(t *testing.T) {
		siteTopic := &models.SiteTopic{
			SiteID:        1,
			TopicID:       3, // Use topic 3 which isn't associated with site 1
			IsActive:      true,
			Priority:      5,
			UsageCount:    0,
			RoundRobinPos: 0,
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
		}

		createdSiteTopic, err := repo.CreateSiteTopic(ctx, siteTopic)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if createdSiteTopic.ID == 0 {
			t.Fatal("Expected siteTopic ID to be set")
		}

		if createdSiteTopic.SiteID != 1 {
			t.Fatalf("Expected site ID 1, got %d", createdSiteTopic.SiteID)
		}

		if createdSiteTopic.TopicID != 3 {
			t.Fatalf("Expected topic ID 3, got %d", createdSiteTopic.TopicID)
		}
	})

	t.Run("CreateSiteTopic duplicate association", func(t *testing.T) {
		siteTopic := &models.SiteTopic{
			SiteID:    1,
			TopicID:   1, // This association already exists in test data
			IsActive:  true,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		_, err := repo.CreateSiteTopic(ctx, siteTopic)
		// Should fail due to UNIQUE constraint
		if err == nil {
			t.Fatal("Expected error for duplicate association")
		}
	})

	t.Run("CreateSiteTopic with minimal data", func(t *testing.T) {
		siteTopic := &models.SiteTopic{
			SiteID:   2,
			TopicID:  3,
			IsActive: false,
		}

		createdSiteTopic, err := repo.CreateSiteTopic(ctx, siteTopic)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if createdSiteTopic.ID == 0 {
			t.Fatal("Expected siteTopic ID to be set")
		}

		if createdSiteTopic.IsActive {
			t.Fatal("Expected IsActive to be false")
		}
	})
}

func TestSiteTopicRepository_GetSiteTopics(t *testing.T) {
	testDB := testhelpers.SetupTestDB(t)
	defer testDB.Close()

	repo := NewRepository(testDB.DB)
	testDB.InsertTestData(t)

	ctx := context.Background()

	t.Run("GetSiteTopics existing site", func(t *testing.T) {
		result, err := repo.GetSiteTopics(ctx, 1, 10, 0)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if len(result.Data) != 2 {
			t.Fatalf("Expected 2 site topics, got %d", len(result.Data))
		}

		if result.Total != 2 {
			t.Fatalf("Expected total count 2, got %d", result.Total)
		}

		// Verify the associations are correct
		foundAssoc1, foundAssoc2 := false, false
		for _, st := range result.Data {
			if st.SiteID == 1 && st.TopicID == 1 {
				foundAssoc1 = true
			}
			if st.SiteID == 1 && st.TopicID == 2 {
				foundAssoc2 = true
			}
		}

		if !foundAssoc1 || !foundAssoc2 {
			t.Fatal("Expected to find both topic associations for site 1")
		}
	})

	t.Run("GetSiteTopics with pagination", func(t *testing.T) {
		result, err := repo.GetSiteTopics(ctx, 1, 1, 0)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if len(result.Data) != 1 {
			t.Fatalf("Expected 1 site topic with limit 1, got %d", len(result.Data))
		}

		if result.Total != 2 {
			t.Fatalf("Expected total count 2, got %d", result.Total)
		}
	})

	t.Run("GetSiteTopics non-existing site", func(t *testing.T) {
		result, err := repo.GetSiteTopics(ctx, 999, 10, 0)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if len(result.Data) != 0 {
			t.Fatalf("Expected 0 site topics for non-existing site, got %d", len(result.Data))
		}
	})
}

func TestSiteTopicRepository_GetTopicSites(t *testing.T) {
	testDB := testhelpers.SetupTestDB(t)
	defer testDB.Close()

	repo := NewRepository(testDB.DB)
	testDB.InsertTestData(t)

	ctx := context.Background()

	t.Run("GetTopicSites existing topic", func(t *testing.T) {
		result, err := repo.GetTopicSites(ctx, 1, 10, 0)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		// Topic 1 is associated with sites 1 and 2 based on test data
		if len(result.Data) != 2 {
			t.Fatalf("Expected 2 topic sites, got %d", len(result.Data))
		}

		if result.Total != 2 {
			t.Fatalf("Expected total count 2, got %d", result.Total)
		}

		// Verify the associations are correct
		foundSite1, foundSite2 := false, false
		for _, st := range result.Data {
			if st.SiteID == 1 && st.TopicID == 1 {
				foundSite1 = true
			}
			if st.SiteID == 2 && st.TopicID == 1 {
				foundSite2 = true
			}
		}

		if !foundSite1 || !foundSite2 {
			t.Fatal("Expected to find both site associations for topic 1")
		}
	})

	t.Run("GetTopicSites with pagination", func(t *testing.T) {
		result, err := repo.GetTopicSites(ctx, 1, 1, 1)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if len(result.Data) != 1 {
			t.Fatalf("Expected 1 topic site with limit 1, got %d", len(result.Data))
		}

		if result.Total != 2 {
			t.Fatalf("Expected total count 2, got %d", result.Total)
		}
	})

	t.Run("GetTopicSites non-existing topic", func(t *testing.T) {
		result, err := repo.GetTopicSites(ctx, 999, 10, 0)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if len(result.Data) != 0 {
			t.Fatalf("Expected 0 topic sites for non-existing topic, got %d", len(result.Data))
		}
	})
}

func TestSiteTopicRepository_GetSiteTopic(t *testing.T) {
	testDB := testhelpers.SetupTestDB(t)
	defer testDB.Close()

	repo := NewRepository(testDB.DB)
	testDB.InsertTestData(t)

	ctx := context.Background()

	t.Run("GetSiteTopic existing association", func(t *testing.T) {
		siteTopic, err := repo.GetSiteTopic(ctx, 1, 1)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if siteTopic.SiteID != 1 {
			t.Fatalf("Expected site ID 1, got %d", siteTopic.SiteID)
		}

		if siteTopic.TopicID != 1 {
			t.Fatalf("Expected topic ID 1, got %d", siteTopic.TopicID)
		}

		if !siteTopic.IsActive {
			t.Fatal("Expected association to be active")
		}
	})

	t.Run("GetSiteTopic non-existing association", func(t *testing.T) {
		_, err := repo.GetSiteTopic(ctx, 1, 999)
		if err == nil {
			t.Fatal("Expected error for non-existing association")
		}
	})
}

func TestSiteTopicRepository_UpdateSiteTopic(t *testing.T) {
	testDB := testhelpers.SetupTestDB(t)
	defer testDB.Close()

	repo := NewRepository(testDB.DB)
	testDB.InsertTestData(t)

	ctx := context.Background()

	t.Run("UpdateSiteTopic success", func(t *testing.T) {
		// Get existing site topic
		siteTopic, err := repo.GetSiteTopic(ctx, 1, 1)
		if err != nil {
			t.Fatalf("Failed to get test site topic: %v", err)
		}

		// Update fields
		siteTopic.IsActive = false
		siteTopic.Priority = 10
		siteTopic.UsageCount = 5
		siteTopic.UpdatedAt = time.Now()

		updatedSiteTopic, err := repo.UpdateSiteTopic(ctx, siteTopic)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if updatedSiteTopic.IsActive {
			t.Fatal("Expected IsActive to be false")
		}

		// Verify update persisted
		savedSiteTopic, err := repo.GetSiteTopic(ctx, 1, 1)
		if err != nil {
			t.Fatalf("Failed to retrieve updated site topic: %v", err)
		}

		if savedSiteTopic.IsActive {
			t.Fatal("Expected persisted IsActive to be false")
		}
	})

	t.Run("UpdateSiteTopic non-existing", func(t *testing.T) {
		siteTopic := &models.SiteTopic{
			ID:       999,
			SiteID:   999,
			TopicID:  999,
			IsActive: true,
		}

		_, err := repo.UpdateSiteTopic(ctx, siteTopic)
		// Update should not fail even if no rows affected
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
	})
}

func TestSiteTopicRepository_DeleteSiteTopic(t *testing.T) {
	testDB := testhelpers.SetupTestDB(t)
	defer testDB.Close()

	repo := NewRepository(testDB.DB)
	testDB.InsertTestData(t)

	ctx := context.Background()

	t.Run("DeleteSiteTopic success", func(t *testing.T) {
		// Get site topic ID first
		siteTopic, err := repo.GetSiteTopic(ctx, 1, 1)
		if err != nil {
			t.Fatalf("Failed to get site topic: %v", err)
		}

		// Delete site topic
		err = repo.DeleteSiteTopic(ctx, siteTopic.ID)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		// Verify deletion
		_, err = repo.GetSiteTopic(ctx, 1, 1)
		if err == nil {
			t.Fatal("Expected error when getting deleted site topic")
		}
	})

	t.Run("DeleteSiteTopic non-existing", func(t *testing.T) {
		err := repo.DeleteSiteTopic(ctx, 999)
		if err != nil {
			t.Fatalf("Expected no error for non-existing site topic, got %v", err)
		}
	})
}

func TestSiteTopicRepository_DeleteSiteTopicBySiteAndTopic(t *testing.T) {
	testDB := testhelpers.SetupTestDB(t)
	defer testDB.Close()

	repo := NewRepository(testDB.DB)
	testDB.InsertTestData(t)

	ctx := context.Background()

	t.Run("DeleteSiteTopicBySiteAndTopic success", func(t *testing.T) {
		// Verify association exists
		_, err := repo.GetSiteTopic(ctx, 1, 2)
		if err != nil {
			t.Fatalf("Association should exist before deletion: %v", err)
		}

		// Delete by site and topic IDs
		err = repo.DeleteSiteTopicBySiteAndTopic(ctx, 1, 2)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		// Verify deletion
		_, err = repo.GetSiteTopic(ctx, 1, 2)
		if err == nil {
			t.Fatal("Expected error when getting deleted association")
		}
	})

	t.Run("DeleteSiteTopicBySiteAndTopic non-existing", func(t *testing.T) {
		err := repo.DeleteSiteTopicBySiteAndTopic(ctx, 999, 999)
		if err != nil {
			t.Fatalf("Expected no error for non-existing association, got %v", err)
		}
	})
}

func TestSiteTopicRepository_ActivateDeactivateSiteTopic(t *testing.T) {
	testDB := testhelpers.SetupTestDB(t)
	defer testDB.Close()

	repo := NewRepository(testDB.DB)
	testDB.InsertTestData(t)

	ctx := context.Background()

	t.Run("DeactivateSiteTopic", func(t *testing.T) {
		// Get site topic ID
		siteTopic, err := repo.GetSiteTopic(ctx, 1, 1)
		if err != nil {
			t.Fatalf("Failed to get site topic: %v", err)
		}

		if !siteTopic.IsActive {
			t.Fatal("Site topic should be active initially")
		}

		// Deactivate
		err = repo.DeactivateSiteTopic(ctx, siteTopic.ID)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		// Verify deactivation
		siteTopic, err = repo.GetSiteTopic(ctx, 1, 1)
		if err != nil {
			t.Fatalf("Failed to get site topic after deactivation: %v", err)
		}

		if siteTopic.IsActive {
			t.Fatal("Site topic should be inactive after deactivation")
		}
	})

	t.Run("ActivateSiteTopic", func(t *testing.T) {
		// Get site topic ID that we just deactivated
		siteTopic, err := repo.GetSiteTopic(ctx, 1, 1)
		if err != nil {
			t.Fatalf("Failed to get site topic: %v", err)
		}

		if siteTopic.IsActive {
			t.Fatal("Site topic should be inactive initially")
		}

		// Activate
		err = repo.ActivateSiteTopic(ctx, siteTopic.ID)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		// Verify activation
		siteTopic, err = repo.GetSiteTopic(ctx, 1, 1)
		if err != nil {
			t.Fatalf("Failed to get site topic after activation: %v", err)
		}

		if !siteTopic.IsActive {
			t.Fatal("Site topic should be active after activation")
		}
	})
}

func TestSiteTopicRepository_GetSiteTopicsForSelection(t *testing.T) {
	testDB := testhelpers.SetupTestDB(t)
	defer testDB.Close()

	repo := NewRepository(testDB.DB)
	testDB.InsertTestData(t)

	ctx := context.Background()

	t.Run("GetSiteTopicsForSelection active topics only", func(t *testing.T) {
		siteTopics, err := repo.GetSiteTopicsForSelection(ctx, 1, "random")
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if len(siteTopics) != 2 {
			t.Fatalf("Expected 2 site topics, got %d", len(siteTopics))
		}

		// Verify all returned site topics are active
		for _, st := range siteTopics {
			if !st.IsActive {
				t.Fatalf("Expected all site topics to be active, but site topic %d is inactive", st.ID)
			}
			if st.SiteID != 1 {
				t.Fatalf("Expected all site topics to have site ID 1, but got %d", st.SiteID)
			}
		}

		// Verify ordering by priority DESC, created_at ASC
		if len(siteTopics) >= 2 {
			// Topic 2 has higher priority (2) than topic 1 (1) in test data
			if siteTopics[0].TopicID != 2 {
				t.Fatalf("Expected first topic to be topic 2 (higher priority), got %d", siteTopics[0].TopicID)
			}
		}
	})

	t.Run("GetSiteTopicsForSelection after deactivation", func(t *testing.T) {
		// Deactivate one site topic
		siteTopic, err := repo.GetSiteTopic(ctx, 1, 1)
		if err != nil {
			t.Fatalf("Failed to get site topic: %v", err)
		}

		err = repo.DeactivateSiteTopic(ctx, siteTopic.ID)
		if err != nil {
			t.Fatalf("Failed to deactivate site topic: %v", err)
		}

		siteTopics, err := repo.GetSiteTopicsForSelection(ctx, 1, "random")
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		// Should only get 1 active site topic now
		if len(siteTopics) != 1 {
			t.Fatalf("Expected 1 active site topic, got %d", len(siteTopics))
		}

		if siteTopics[0].TopicID != 2 {
			t.Fatalf("Expected remaining site topic to have topic ID 2, got %d", siteTopics[0].TopicID)
		}
	})

	t.Run("GetSiteTopicsForSelection non-existing site", func(t *testing.T) {
		siteTopics, err := repo.GetSiteTopicsForSelection(ctx, 999, "random")
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if len(siteTopics) != 0 {
			t.Fatalf("Expected 0 site topics for non-existing site, got %d", len(siteTopics))
		}
	})
}

func TestSiteTopicRepository_UpdateSiteTopicUsage(t *testing.T) {
	testDB := testhelpers.SetupTestDB(t)
	defer testDB.Close()

	repo := NewRepository(testDB.DB)
	testDB.InsertTestData(t)

	ctx := context.Background()

	t.Run("UpdateSiteTopicUsage random strategy", func(t *testing.T) {
		// Get site topic before update
		siteTopic, err := repo.GetSiteTopic(ctx, 1, 1)
		if err != nil {
			t.Fatalf("Failed to get site topic: %v", err)
		}

		initialUsageCount := siteTopic.UsageCount
		initialLastUsedAt := siteTopic.LastUsedAt

		// Update usage with random strategy
		err = repo.UpdateSiteTopicUsage(ctx, siteTopic.ID, "random")
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		// Verify updates
		updatedSiteTopic, err := repo.GetSiteTopic(ctx, 1, 1)
		if err != nil {
			t.Fatalf("Failed to get updated site topic: %v", err)
		}

		if updatedSiteTopic.UsageCount != initialUsageCount+1 {
			t.Fatalf("Expected usage count to increment by 1, got %d", updatedSiteTopic.UsageCount)
		}

		if !updatedSiteTopic.LastUsedAt.After(initialLastUsedAt) {
			t.Fatal("Expected LastUsedAt to be updated")
		}
	})

	t.Run("UpdateSiteTopicUsage round_robin strategy", func(t *testing.T) {
		// Get site topic before update
		siteTopic, err := repo.GetSiteTopic(ctx, 2, 1)
		if err != nil {
			t.Fatalf("Failed to get site topic: %v", err)
		}

		initialRoundRobinPos := siteTopic.RoundRobinPos

		// Update usage with round_robin strategy
		err = repo.UpdateSiteTopicUsage(ctx, siteTopic.ID, "round_robin")
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		// Verify round-robin position was incremented
		updatedSiteTopic, err := repo.GetSiteTopic(ctx, 2, 1)
		if err != nil {
			t.Fatalf("Failed to get updated site topic: %v", err)
		}

		if updatedSiteTopic.RoundRobinPos != initialRoundRobinPos+1 {
			t.Fatalf("Expected round-robin position to increment by 1, got %d", updatedSiteTopic.RoundRobinPos)
		}
	})
}

func TestSiteTopicRepository_GetTopicStats(t *testing.T) {
	testDB := testhelpers.SetupTestDB(t)
	defer testDB.Close()

	repo := NewRepository(testDB.DB)
	testDB.InsertTestData(t)

	ctx := context.Background()

	t.Run("GetTopicStats success", func(t *testing.T) {
		stats, err := repo.GetTopicStats(ctx, 1)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if stats.SiteID != 1 {
			t.Fatalf("Expected site ID 1, got %d", stats.SiteID)
		}

		if stats.TotalTopics != 2 {
			t.Fatalf("Expected 2 total topics, got %d", stats.TotalTopics)
		}

		if stats.ActiveTopics != 2 {
			t.Fatalf("Expected 2 active topics, got %d", stats.ActiveTopics)
		}

		// Based on test data, both topics have been used (usage_count > 0)
		if stats.UsedTopics != 2 {
			t.Fatalf("Expected 2 used topics, got %d", stats.UsedTopics)
		}

		if stats.UnusedTopics != 0 {
			t.Fatalf("Expected 0 unused topics, got %d", stats.UnusedTopics)
		}

		if stats.UniqueTopicsLeft != 0 {
			t.Fatalf("Expected 0 unique topics left, got %d", stats.UniqueTopicsLeft)
		}
	})

	t.Run("GetTopicStats after usage update", func(t *testing.T) {
		// Reset one topic's usage count to 0
		_, err := testDB.DB.Exec("UPDATE site_topics SET usage_count = 0 WHERE site_id = 2 AND topic_id = 2")
		if err != nil {
			t.Fatalf("Failed to reset usage count: %v", err)
		}

		stats, err := repo.GetTopicStats(ctx, 2)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if stats.UsedTopics != 1 {
			t.Fatalf("Expected 1 used topic, got %d", stats.UsedTopics)
		}

		if stats.UnusedTopics != 1 {
			t.Fatalf("Expected 1 unused topic, got %d", stats.UnusedTopics)
		}

		if stats.UniqueTopicsLeft != 1 {
			t.Fatalf("Expected 1 unique topic left, got %d", stats.UniqueTopicsLeft)
		}
	})

	t.Run("GetTopicStats non-existing site", func(t *testing.T) {
		stats, err := repo.GetTopicStats(ctx, 999)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if stats.TotalTopics != 0 {
			t.Fatalf("Expected 0 total topics, got %d", stats.TotalTopics)
		}

		if stats.ActiveTopics != 0 {
			t.Fatalf("Expected 0 active topics, got %d", stats.ActiveTopics)
		}
	})
}

func TestSiteTopicRepository_EdgeCases(t *testing.T) {
	testDB := testhelpers.SetupTestDB(t)
	defer testDB.Close()

	repo := NewRepository(testDB.DB)

	ctx := context.Background()

	t.Run("Large pagination values", func(t *testing.T) {
		result, err := repo.GetSiteTopics(ctx, 1, 1000, 0)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if len(result.Data) != 0 {
			t.Fatalf("Expected 0 site topics in empty DB, got %d", len(result.Data))
		}
	})

	t.Run("Negative pagination values", func(t *testing.T) {
		result, err := repo.GetSiteTopics(ctx, 1, -1, -1)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if result == nil {
			t.Fatal("Expected result not to be nil")
		}
	})

	t.Run("Very large site/topic IDs", func(t *testing.T) {
		_, err := repo.GetSiteTopic(ctx, 9223372036854775807, 9223372036854775807)
		if err == nil {
			t.Fatal("Expected error for very large IDs")
		}
	})
}
