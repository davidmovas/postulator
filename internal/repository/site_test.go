package repository

import (
	"context"
	"testing"
	"time"

	"Postulator/internal/models"
	"Postulator/internal/testhelpers"
)

func TestSiteRepository_GetSites(t *testing.T) {
	testDB := testhelpers.SetupTestDB(t)
	defer testDB.Close()

	repo := NewRepository(testDB.DB)
	testDB.InsertTestData(t)

	ctx := context.Background()

	t.Run("GetSites with pagination", func(t *testing.T) {
		result, err := repo.GetSites(ctx, 2, 0)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if len(result.Data) != 2 {
			t.Fatalf("Expected 2 sites, got %d", len(result.Data))
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

	t.Run("GetSites with offset", func(t *testing.T) {
		result, err := repo.GetSites(ctx, 2, 2)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if len(result.Data) != 1 {
			t.Fatalf("Expected 1 site, got %d", len(result.Data))
		}

		if result.Offset != 2 {
			t.Fatalf("Expected offset 2, got %d", result.Offset)
		}
	})

	t.Run("GetSites empty database", func(t *testing.T) {
		testDB.ClearAllTables(t)

		result, err := repo.GetSites(ctx, 10, 0)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if len(result.Data) != 0 {
			t.Fatalf("Expected 0 sites, got %d", len(result.Data))
		}

		if result.Total != 0 {
			t.Fatalf("Expected total count 0, got %d", result.Total)
		}
	})
}

func TestSiteRepository_GetSite(t *testing.T) {
	testDB := testhelpers.SetupTestDB(t)
	defer testDB.Close()

	repo := NewRepository(testDB.DB)
	testDB.InsertTestData(t)

	ctx := context.Background()

	t.Run("GetSite existing", func(t *testing.T) {
		site, err := repo.GetSite(ctx, 1)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if site.ID != 1 {
			t.Fatalf("Expected site ID 1, got %d", site.ID)
		}

		if site.Name != "Test Site 1" {
			t.Fatalf("Expected site name 'Test Site 1', got %s", site.Name)
		}

		if site.URL != "https://test1.com" {
			t.Fatalf("Expected URL 'https://test1.com', got %s", site.URL)
		}

		if site.Strategy != "random" {
			t.Fatalf("Expected strategy 'random', got %s", site.Strategy)
		}
	})

	t.Run("GetSite non-existing", func(t *testing.T) {
		_, err := repo.GetSite(ctx, 999)
		if err == nil {
			t.Fatal("Expected error for non-existing site, got nil")
		}
	})
}

func TestSiteRepository_CreateSite(t *testing.T) {
	testDB := testhelpers.SetupTestDB(t)
	defer testDB.Close()

	repo := NewRepository(testDB.DB)

	ctx := context.Background()

	t.Run("CreateSite success", func(t *testing.T) {
		now := time.Now()
		site := &models.Site{
			Name:      "New Test Site",
			URL:       "https://newtest.com",
			Username:  "newuser",
			Password:  "newpass",
			IsActive:  true,
			Status:    "pending",
			Strategy:  "unique",
			CreatedAt: now,
			UpdatedAt: now,
		}

		createdSite, err := repo.CreateSite(ctx, site)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if createdSite.ID == 0 {
			t.Fatal("Expected site ID to be set")
		}

		if createdSite.Name != site.Name {
			t.Fatalf("Expected name %s, got %s", site.Name, createdSite.Name)
		}

		// Verify site was actually saved
		savedSite, err := repo.GetSite(ctx, createdSite.ID)
		if err != nil {
			t.Fatalf("Failed to retrieve created site: %v", err)
		}

		if savedSite.Name != site.Name {
			t.Fatalf("Expected saved name %s, got %s", site.Name, savedSite.Name)
		}

		if savedSite.Strategy != "unique" {
			t.Fatalf("Expected strategy 'unique', got %s", savedSite.Strategy)
		}
	})

	t.Run("CreateSite with minimal data", func(t *testing.T) {
		site := &models.Site{
			Name:      "Minimal Site",
			URL:       "https://minimal.com",
			Username:  "min",
			Password:  "pass",
			IsActive:  false,
			Status:    "pending",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		createdSite, err := repo.CreateSite(ctx, site)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if createdSite.ID == 0 {
			t.Fatal("Expected site ID to be set")
		}

		if createdSite.IsActive != false {
			t.Fatal("Expected IsActive to be false")
		}
	})
}

func TestSiteRepository_UpdateSite(t *testing.T) {
	testDB := testhelpers.SetupTestDB(t)
	defer testDB.Close()

	repo := NewRepository(testDB.DB)
	testDB.InsertTestData(t)

	ctx := context.Background()

	t.Run("UpdateSite success", func(t *testing.T) {
		// Get existing site
		site, err := repo.GetSite(ctx, 1)
		if err != nil {
			t.Fatalf("Failed to get test site: %v", err)
		}

		// Update fields
		site.Name = "Updated Site Name"
		site.URL = "https://updated.com"
		site.Strategy = "round_robin"
		site.IsActive = false
		site.UpdatedAt = time.Now()

		updatedSite, err := repo.UpdateSite(ctx, site)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if updatedSite.Name != "Updated Site Name" {
			t.Fatalf("Expected updated name, got %s", updatedSite.Name)
		}

		if updatedSite.Strategy != "round_robin" {
			t.Fatalf("Expected strategy 'round_robin', got %s", updatedSite.Strategy)
		}

		// Verify update persisted
		savedSite, err := repo.GetSite(ctx, 1)
		if err != nil {
			t.Fatalf("Failed to retrieve updated site: %v", err)
		}

		if savedSite.Name != "Updated Site Name" {
			t.Fatalf("Expected persisted name 'Updated Site Name', got %s", savedSite.Name)
		}

		if savedSite.IsActive != false {
			t.Fatal("Expected IsActive to be false")
		}
	})

	t.Run("UpdateSite non-existing", func(t *testing.T) {
		site := &models.Site{
			ID:        999,
			Name:      "Non-existing",
			URL:       "https://none.com",
			Username:  "none",
			Password:  "none",
			IsActive:  true,
			Status:    "pending",
			Strategy:  "random",
			UpdatedAt: time.Now(),
		}

		_, err := repo.UpdateSite(ctx, site)
		// Update should not fail even if no rows affected
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
	})
}

func TestSiteRepository_DeleteSite(t *testing.T) {
	testDB := testhelpers.SetupTestDB(t)
	defer testDB.Close()

	repo := NewRepository(testDB.DB)
	testDB.InsertTestData(t)

	ctx := context.Background()

	t.Run("DeleteSite success", func(t *testing.T) {
		// Verify site exists
		_, err := repo.GetSite(ctx, 1)
		if err != nil {
			t.Fatalf("Site should exist before deletion: %v", err)
		}

		// Delete site
		err = repo.DeleteSite(ctx, 1)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		// Verify site is deleted
		_, err = repo.GetSite(ctx, 1)
		if err == nil {
			t.Fatal("Expected error when getting deleted site")
		}
	})

	t.Run("DeleteSite non-existing", func(t *testing.T) {
		err := repo.DeleteSite(ctx, 999)
		if err != nil {
			t.Fatalf("Expected no error for non-existing site, got %v", err)
		}
	})
}

func TestSiteRepository_ActivateDeactivateSite(t *testing.T) {
	testDB := testhelpers.SetupTestDB(t)
	defer testDB.Close()

	repo := NewRepository(testDB.DB)
	testDB.InsertTestData(t)

	ctx := context.Background()

	t.Run("ActivateSite", func(t *testing.T) {
		// Make sure site 3 is inactive
		site, err := repo.GetSite(ctx, 3)
		if err != nil {
			t.Fatalf("Failed to get test site: %v", err)
		}

		if site.IsActive {
			t.Fatal("Site should be inactive initially")
		}

		// Activate site
		err = repo.ActivateSite(ctx, 3)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		// Verify activation
		site, err = repo.GetSite(ctx, 3)
		if err != nil {
			t.Fatalf("Failed to get site after activation: %v", err)
		}

		if !site.IsActive {
			t.Fatal("Site should be active after activation")
		}
	})

	t.Run("DeactivateSite", func(t *testing.T) {
		// Activate site first
		err := repo.ActivateSite(ctx, 1)
		if err != nil {
			t.Fatalf("Failed to activate site: %v", err)
		}

		// Deactivate site
		err = repo.DeactivateSite(ctx, 1)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		// Verify deactivation
		site, err := repo.GetSite(ctx, 1)
		if err != nil {
			t.Fatalf("Failed to get site after deactivation: %v", err)
		}

		if site.IsActive {
			t.Fatal("Site should be inactive after deactivation")
		}
	})
}

func TestSiteRepository_SetCheckStatus(t *testing.T) {
	testDB := testhelpers.SetupTestDB(t)
	defer testDB.Close()

	repo := NewRepository(testDB.DB)
	testDB.InsertTestData(t)

	ctx := context.Background()

	t.Run("SetCheckStatus success", func(t *testing.T) {
		checkTime := time.Now()
		status := "connected"

		err := repo.SetCheckStatus(ctx, 1, checkTime, status)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		// Verify status was set
		site, err := repo.GetSite(ctx, 1)
		if err != nil {
			t.Fatalf("Failed to get site after status update: %v", err)
		}

		if site.Status != "connected" {
			t.Fatalf("Expected status 'connected', got %s", site.Status)
		}

		// Check that last_check time was updated (allow some tolerance)
		timeDiff := site.LastCheck.Sub(checkTime)
		if timeDiff > time.Second || timeDiff < -time.Second {
			t.Fatalf("Expected last_check to be close to %v, got %v", checkTime, site.LastCheck)
		}
	})

	t.Run("SetCheckStatus different statuses", func(t *testing.T) {
		statuses := []string{"error", "pending", "maintenance"}

		for _, status := range statuses {
			checkTime := time.Now()

			err := repo.SetCheckStatus(ctx, 2, checkTime, status)
			if err != nil {
				t.Fatalf("Expected no error for status %s, got %v", status, err)
			}

			site, err := repo.GetSite(ctx, 2)
			if err != nil {
				t.Fatalf("Failed to get site: %v", err)
			}

			if site.Status != status {
				t.Fatalf("Expected status %s, got %s", status, site.Status)
			}
		}
	})
}

func TestSiteRepository_EdgeCases(t *testing.T) {
	testDB := testhelpers.SetupTestDB(t)
	defer testDB.Close()

	repo := NewRepository(testDB.DB)

	ctx := context.Background()

	t.Run("Large pagination values", func(t *testing.T) {
		result, err := repo.GetSites(ctx, 1000, 0)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if len(result.Data) != 0 {
			t.Fatalf("Expected 0 sites in empty DB, got %d", len(result.Data))
		}
	})

	t.Run("Negative pagination values", func(t *testing.T) {
		// The repository should handle negative values gracefully
		result, err := repo.GetSites(ctx, -1, -1)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		// SQLite should handle this gracefully
		if result == nil {
			t.Fatal("Expected result not to be nil")
		}
	})

	t.Run("Very large site ID", func(t *testing.T) {
		_, err := repo.GetSite(ctx, 9223372036854775807) // Max int64
		if err == nil {
			t.Fatal("Expected error for very large ID")
		}
	})
}
