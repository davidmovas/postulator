package site

import (
	"Postulator/internal/config"
	"Postulator/internal/domain/entities"
	"Postulator/internal/infra/database"
	"Postulator/internal/infra/wp"
	"Postulator/pkg/di"
	"Postulator/pkg/logger"
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func setupTestService(t *testing.T) (*Service, func()) {
	t.Helper()

	db, dbCleanup := database.SetupTestDB(t)

	tempLogDir := filepath.Join(os.TempDir(), "postulator_test_logs", t.Name())
	_ = os.MkdirAll(tempLogDir, 0755)

	container := di.New()

	testLogger, err := logger.NewForTest(&config.Config{
		LogDir:      tempLogDir,
		AppLogFile:  "test.log",
		ErrLogFile:  "test_error.log",
		LogLevel:    "debug",
		ConsoleOut:  false,
		PrettyPrint: false,
	})
	if err != nil {
		dbCleanup()
		t.Fatalf("failed to create logger: %v", err)
	}

	container.MustRegister(di.Instance[*database.DB](db))
	container.MustRegister(di.Instance[*logger.Logger](testLogger))
	container.MustRegister(di.Instance[*wp.Client](wp.NewClient()))

	service, err := NewService(container)
	if err != nil {
		dbCleanup()
		t.Fatalf("failed to create service: %v", err)
	}

	cleanup := func() {
		_ = testLogger.Close()
		_ = os.RemoveAll(tempLogDir)
		dbCleanup()
	}

	return service, cleanup
}

func TestSiteService_CreateAndGet(t *testing.T) {
	service, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()

	t.Run("create site successfully", func(t *testing.T) {
		site := &entities.Site{
			Name:         "Test Site",
			URL:          "https://test.example.com",
			WPUsername:   "admin",
			WPPassword:   "password123",
			Status:       entities.StatusActive,
			HealthStatus: entities.HealthStatusUnknown,
		}

		err := service.CreateSite(ctx, site)
		if err != nil {
			t.Fatalf("failed to create site: %v", err)
		}
	})

	t.Run("get site by ID", func(t *testing.T) {
		site := &entities.Site{
			Name:         "Get Test Site",
			URL:          "https://gettest.example.com",
			WPUsername:   "admin",
			WPPassword:   "password123",
			Status:       entities.StatusActive,
			HealthStatus: entities.HealthStatusUnknown,
		}

		err := service.CreateSite(ctx, site)
		if err != nil {
			t.Fatalf("failed to create site: %v", err)
		}

		sites, err := service.ListSites(ctx)
		if err != nil {
			t.Fatalf("failed to list sites: %v", err)
		}

		if len(sites) == 0 {
			t.Fatal("no sites found")
		}

		retrievedSite, err := service.GetSite(ctx, sites[len(sites)-1].ID)
		if err != nil {
			t.Fatalf("failed to get site: %v", err)
		}

		if retrievedSite.Name != "Get Test Site" {
			t.Errorf("expected name 'Get Test Site', got '%s'", retrievedSite.Name)
		}
		if retrievedSite.URL != "https://gettest.example.com" {
			t.Errorf("expected URL 'https://gettest.example.com', got '%s'", retrievedSite.URL)
		}
	})

	t.Run("create site with duplicate URL should fail", func(t *testing.T) {
		site1 := &entities.Site{
			Name:         "Duplicate Test 1",
			URL:          "https://duplicate.example.com",
			WPUsername:   "admin",
			WPPassword:   "password123",
			Status:       entities.StatusActive,
			HealthStatus: entities.HealthStatusUnknown,
		}

		err := service.CreateSite(ctx, site1)
		if err != nil {
			t.Fatalf("failed to create first site: %v", err)
		}

		site2 := &entities.Site{
			Name:         "Duplicate Test 2",
			URL:          "https://duplicate.example.com",
			WPUsername:   "admin2",
			WPPassword:   "password456",
			Status:       entities.StatusActive,
			HealthStatus: entities.HealthStatusUnknown,
		}

		err = service.CreateSite(ctx, site2)
		if err == nil {
			t.Fatal("expected error when creating site with duplicate URL, got nil")
		}
	})

	t.Run("get non-existent site should fail", func(t *testing.T) {
		_, err := service.GetSite(ctx, 999999)
		if err == nil {
			t.Fatal("expected error when getting non-existent site, got nil")
		}
	})
}

func TestSiteService_ListSites(t *testing.T) {
	service, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()

	t.Run("list empty sites", func(t *testing.T) {
		sites, err := service.ListSites(ctx)
		if err != nil {
			t.Fatalf("failed to list sites: %v", err)
		}

		if len(sites) != 0 {
			t.Errorf("expected 0 sites, got %d", len(sites))
		}
	})

	t.Run("list multiple sites", func(t *testing.T) {
		for i := 1; i <= 3; i++ {
			site := &entities.Site{
				Name:         "List Test Site " + string(rune('0'+i)),
				URL:          "https://listtest" + string(rune('0'+i)) + ".example.com",
				WPUsername:   "admin",
				WPPassword:   "password123",
				Status:       entities.StatusActive,
				HealthStatus: entities.HealthStatusUnknown,
			}

			err := service.CreateSite(ctx, site)
			if err != nil {
				t.Fatalf("failed to create site %d: %v", i, err)
			}
		}

		sites, err := service.ListSites(ctx)
		if err != nil {
			t.Fatalf("failed to list sites: %v", err)
		}

		if len(sites) != 3 {
			t.Errorf("expected 3 sites, got %d", len(sites))
		}
	})
}

func TestSiteService_UpdateSite(t *testing.T) {
	service, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()

	t.Run("update site successfully", func(t *testing.T) {
		site := &entities.Site{
			Name:         "Update Test Site",
			URL:          "https://updatetest.example.com",
			WPUsername:   "admin",
			WPPassword:   "password123",
			Status:       entities.StatusActive,
			HealthStatus: entities.HealthStatusUnknown,
		}

		err := service.CreateSite(ctx, site)
		if err != nil {
			t.Fatalf("failed to create site: %v", err)
		}

		sites, err := service.ListSites(ctx)
		if err != nil {
			t.Fatalf("failed to list sites: %v", err)
		}

		if len(sites) == 0 {
			t.Fatal("no sites found")
		}

		updatedSite := sites[0]
		updatedSite.Name = "Updated Site Name"
		updatedSite.Status = entities.StatusInactive

		err = service.UpdateSite(ctx, updatedSite)
		if err != nil {
			t.Fatalf("failed to update site: %v", err)
		}

		retrievedSite, err := service.GetSite(ctx, updatedSite.ID)
		if err != nil {
			t.Fatalf("failed to get updated site: %v", err)
		}

		if retrievedSite.Name != "Updated Site Name" {
			t.Errorf("expected name 'Updated Site Name', got '%s'", retrievedSite.Name)
		}
		if retrievedSite.Status != entities.StatusInactive {
			t.Errorf("expected status 'inactive', got '%s'", retrievedSite.Status)
		}
	})

}

func TestSiteService_DeleteSite(t *testing.T) {
	service, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()

	t.Run("delete site successfully", func(t *testing.T) {
		site := &entities.Site{
			Name:         "Delete Test Site",
			URL:          "https://deletetest.example.com",
			WPUsername:   "admin",
			WPPassword:   "password123",
			Status:       entities.StatusActive,
			HealthStatus: entities.HealthStatusUnknown,
		}

		err := service.CreateSite(ctx, site)
		if err != nil {
			t.Fatalf("failed to create site: %v", err)
		}

		sites, err := service.ListSites(ctx)
		if err != nil {
			t.Fatalf("failed to list sites: %v", err)
		}

		if len(sites) == 0 {
			t.Fatal("no sites found")
		}

		siteID := sites[0].ID

		err = service.DeleteSite(ctx, siteID)
		if err != nil {
			t.Fatalf("failed to delete site: %v", err)
		}

		_, err = service.GetSite(ctx, siteID)
		if err == nil {
			t.Fatal("expected error when getting deleted site, got nil")
		}
	})

}

func TestSiteService_UpdateHealthStatus(t *testing.T) {
	service, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()

	t.Run("update health status", func(t *testing.T) {
		site := &entities.Site{
			Name:         "Health Test Site",
			URL:          "https://healthtest.example.com",
			WPUsername:   "admin",
			WPPassword:   "password123",
			Status:       entities.StatusActive,
			HealthStatus: entities.HealthStatusUnknown,
		}

		err := service.CreateSite(ctx, site)
		if err != nil {
			t.Fatalf("failed to create site: %v", err)
		}

		sites, err := service.ListSites(ctx)
		if err != nil {
			t.Fatalf("failed to list sites: %v", err)
		}

		if len(sites) == 0 {
			t.Fatal("no sites found")
		}

		siteID := sites[0].ID

		_ = service.CheckHealth(ctx, siteID)

		retrievedSite, err := service.GetSite(ctx, siteID)
		if err != nil {
			t.Fatalf("failed to get site: %v", err)
		}

		if retrievedSite.HealthStatus != entities.HealthStatusUnhealthy {
			t.Errorf("expected health status 'unhealthy', got '%s'", retrievedSite.HealthStatus)
		}

		if retrievedSite.LastHealthCheck == nil {
			t.Error("expected LastHealthCheck to be set, got nil")
		} else {
			timeSince := time.Since(*retrievedSite.LastHealthCheck)
			if timeSince > time.Minute {
				t.Errorf("LastHealthCheck is too old: %v", timeSince)
			}
		}
	})
}

func TestSiteService_Categories(t *testing.T) {
	service, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()

	t.Run("sync categories fails with unreachable site", func(t *testing.T) {
		site := &entities.Site{
			Name:         "Sync Test Site",
			URL:          "https://synctest.example.com",
			WPUsername:   "admin",
			WPPassword:   "password123",
			Status:       entities.StatusActive,
			HealthStatus: entities.HealthStatusUnknown,
		}

		err := service.CreateSite(ctx, site)
		if err != nil {
			t.Fatalf("failed to create site: %v", err)
		}

		sites, err := service.ListSites(ctx)
		if err != nil {
			t.Fatalf("failed to list sites: %v", err)
		}

		if len(sites) == 0 {
			t.Fatal("no sites found")
		}

		siteID := sites[0].ID

		err = service.SyncCategories(ctx, siteID)
		if err == nil {
			t.Fatal("expected error when syncing categories from unreachable site, got nil")
		}
	})
}
