package handlers

import (
	"testing"

	"Postulator/internal/dto"
)

func TestHandler_CreateSite(t *testing.T) {
	handler, testDB, cleanup := setupHandlerTest(t)
	defer cleanup()

	tests := []struct {
		name    string
		request dto.CreateSiteRequest
		wantErr bool
		errMsg  string
	}{
		{
			name: "Valid site creation",
			request: dto.CreateSiteRequest{
				Name:     "Test Site",
				URL:      "https://test.com",
				Username: "testuser",
				Password: "testpass",
				IsActive: true,
				Strategy: "random",
			},
			wantErr: false,
		},
		{
			name: "Empty name should fail",
			request: dto.CreateSiteRequest{
				Name:     "",
				URL:      "https://test.com",
				Username: "testuser",
				Password: "testpass",
				IsActive: true,
				Strategy: "random",
			},
			wantErr: true,
		},
		{
			name: "Invalid URL should fail",
			request: dto.CreateSiteRequest{
				Name:     "Test Site",
				URL:      "invalid-url",
				Username: "testuser",
				Password: "testpass",
				IsActive: true,
				Strategy: "random",
			},
			wantErr: true,
		},
		{
			name: "Empty username should fail",
			request: dto.CreateSiteRequest{
				Name:     "Test Site",
				URL:      "https://test.com",
				Username: "",
				Password: "testpass",
				IsActive: true,
				Strategy: "random",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testDB.ClearAllTables(t)

			response, err := handler.CreateSite(tt.request)

			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected error but got none")
				} else if tt.errMsg != "" && err.Error() != tt.errMsg {
					t.Errorf("Expected error message '%s', got '%s'", tt.errMsg, err.Error())
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if response == nil {
				t.Error("Response should not be nil")
				return
			}

			if response.ID <= 0 {
				t.Error("Response ID should be positive")
			}

			if response.Name != tt.request.Name {
				t.Errorf("Expected name '%s', got '%s'", tt.request.Name, response.Name)
			}

			if response.URL != tt.request.URL {
				t.Errorf("Expected URL '%s', got '%s'", tt.request.URL, response.URL)
			}

			if response.Strategy != tt.request.Strategy {
				t.Errorf("Expected strategy '%s', got '%s'", tt.request.Strategy, response.Strategy)
			}
		})
	}
}

func TestHandler_GetSite(t *testing.T) {
	handler, testDB, cleanup := setupHandlerTest(t)
	defer cleanup()

	testDB.ClearAllTables(t)

	// Create a test site first
	createReq := dto.CreateSiteRequest{
		Name:     "Test Site",
		URL:      "https://test.com",
		Username: "testuser",
		Password: "testpass",
		IsActive: true,
		Strategy: "random",
	}

	created, err := handler.CreateSite(createReq)
	if err != nil {
		t.Fatalf("Failed to create test site: %v", err)
	}

	tests := []struct {
		name    string
		siteID  int64
		wantErr bool
	}{
		{
			name:    "Valid site ID",
			siteID:  created.ID,
			wantErr: false,
		},
		{
			name:    "Invalid site ID",
			siteID:  0,
			wantErr: true,
		},
		{
			name:    "Non-existent site ID",
			siteID:  999,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response, err := handler.GetSite(tt.siteID)

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

			if response == nil {
				t.Error("Response should not be nil")
				return
			}

			if response.ID != created.ID {
				t.Errorf("Expected ID %d, got %d", created.ID, response.ID)
			}

			if response.Name != created.Name {
				t.Errorf("Expected name '%s', got '%s'", created.Name, response.Name)
			}
		})
	}
}

func TestHandler_GetSites(t *testing.T) {
	handler, testDB, cleanup := setupHandlerTest(t)
	defer cleanup()

	testDB.ClearAllTables(t)

	// Create multiple test sites
	sites := []dto.CreateSiteRequest{
		{Name: "Site 1", URL: "https://site1.com", Username: "user1", Password: "pass1", Strategy: "random"},
		{Name: "Site 2", URL: "https://site2.com", Username: "user2", Password: "pass2", Strategy: "unique"},
		{Name: "Site 3", URL: "https://site3.com", Username: "user3", Password: "pass3", Strategy: "round_robin"},
	}

	for _, site := range sites {
		_, err := handler.CreateSite(site)
		if err != nil {
			t.Fatalf("Failed to create test site: %v", err)
		}
	}

	tests := []struct {
		name       string
		pagination dto.PaginationRequest
		wantCount  int
	}{
		{
			name:       "Get all sites",
			pagination: dto.PaginationRequest{Page: 1, Limit: 10},
			wantCount:  3,
		},
		{
			name:       "Get with limit",
			pagination: dto.PaginationRequest{Page: 1, Limit: 2},
			wantCount:  2,
		},
		{
			name:       "Get second page",
			pagination: dto.PaginationRequest{Page: 2, Limit: 2},
			wantCount:  1,
		},
		{
			name:       "Default pagination",
			pagination: dto.PaginationRequest{},
			wantCount:  3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response, err := handler.GetSites(tt.pagination)

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if response == nil {
				t.Error("Response should not be nil")
				return
			}

			if len(response.Sites) != tt.wantCount {
				t.Errorf("Expected %d sites, got %d", tt.wantCount, len(response.Sites))
			}

			if response.Pagination == nil {
				t.Error("Pagination should not be nil")
				return
			}

			if response.Pagination.Total != 3 {
				t.Errorf("Expected total 3, got %d", response.Pagination.Total)
			}
		})
	}
}

func TestHandler_UpdateSite(t *testing.T) {
	handler, testDB, cleanup := setupHandlerTest(t)
	defer cleanup()

	testDB.ClearAllTables(t)

	// Create a test site first
	createReq := dto.CreateSiteRequest{
		Name:     "Original Site",
		URL:      "https://original.com",
		Username: "original",
		Password: "original",
		IsActive: true,
		Strategy: "random",
	}

	created, err := handler.CreateSite(createReq)
	if err != nil {
		t.Fatalf("Failed to create test site: %v", err)
	}

	tests := []struct {
		name    string
		request dto.UpdateSiteRequest
		wantErr bool
	}{
		{
			name: "Valid update",
			request: dto.UpdateSiteRequest{
				ID:       created.ID,
				Name:     "Updated Site",
				URL:      "https://updated.com",
				Username: "updated",
				Password: "updated",
				IsActive: false,
				Strategy: "unique",
			},
			wantErr: false,
		},
		{
			name: "Invalid ID should fail",
			request: dto.UpdateSiteRequest{
				ID:       0,
				Name:     "Test",
				URL:      "https://test.com",
				Username: "test",
				Password: "test",
				IsActive: true,
				Strategy: "random",
			},
			wantErr: true,
		},
		{
			name: "Empty name should fail",
			request: dto.UpdateSiteRequest{
				ID:       created.ID,
				Name:     "",
				URL:      "https://test.com",
				Username: "test",
				Password: "test",
				IsActive: true,
				Strategy: "random",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response, err := handler.UpdateSite(tt.request)

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

			if response == nil {
				t.Error("Response should not be nil")
				return
			}

			if response.ID != tt.request.ID {
				t.Errorf("Expected ID %d, got %d", tt.request.ID, response.ID)
			}

			if response.Name != tt.request.Name {
				t.Errorf("Expected name '%s', got '%s'", tt.request.Name, response.Name)
			}

			if response.Strategy != tt.request.Strategy {
				t.Errorf("Expected strategy '%s', got '%s'", tt.request.Strategy, response.Strategy)
			}
		})
	}
}

func TestHandler_DeleteSite(t *testing.T) {
	handler, testDB, cleanup := setupHandlerTest(t)
	defer cleanup()

	testDB.ClearAllTables(t)

	// Create a test site first
	createReq := dto.CreateSiteRequest{
		Name:     "Test Site",
		URL:      "https://test.com",
		Username: "test",
		Password: "test",
		IsActive: true,
		Strategy: "random",
	}

	created, err := handler.CreateSite(createReq)
	if err != nil {
		t.Fatalf("Failed to create test site: %v", err)
	}

	tests := []struct {
		name    string
		siteID  int64
		wantErr bool
	}{
		{
			name:    "Valid deletion",
			siteID:  created.ID,
			wantErr: false,
		},
		{
			name:    "Invalid ID should fail",
			siteID:  0,
			wantErr: true,
		},
		{
			name:    "Non-existent ID should not fail",
			siteID:  999,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := handler.DeleteSite(tt.siteID)

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}

	// Verify deletion
	_, err = handler.GetSite(created.ID)
	if err == nil {
		t.Error("Expected error when getting deleted site")
	}
}

func TestHandler_ActivateDeactivateSite(t *testing.T) {
	handler, testDB, cleanup := setupHandlerTest(t)
	defer cleanup()

	testDB.ClearAllTables(t)

	// Create a test site first
	createReq := dto.CreateSiteRequest{
		Name:     "Test Site",
		URL:      "https://test.com",
		Username: "test",
		Password: "test",
		IsActive: false,
		Strategy: "random",
	}

	created, err := handler.CreateSite(createReq)
	if err != nil {
		t.Fatalf("Failed to create test site: %v", err)
	}

	t.Run("Activate site", func(t *testing.T) {
		err := handler.ActivateSite(created.ID)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}

		// Verify activation by getting the site
		response, err := handler.GetSite(created.ID)
		if err != nil {
			t.Errorf("Failed to get site after activation: %v", err)
			return
		}

		if !response.IsActive {
			t.Error("Expected site to be active after activation")
		}
	})

	t.Run("Deactivate site", func(t *testing.T) {
		err := handler.DeactivateSite(created.ID)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}

		// Verify deactivation by getting the site
		response, err := handler.GetSite(created.ID)
		if err != nil {
			t.Errorf("Failed to get site after deactivation: %v", err)
			return
		}

		if response.IsActive {
			t.Error("Expected site to be inactive after deactivation")
		}
	})

	t.Run("Invalid site ID should fail", func(t *testing.T) {
		err := handler.ActivateSite(0)
		if err == nil {
			t.Error("Expected error for invalid site ID")
		}

		err = handler.DeactivateSite(0)
		if err == nil {
			t.Error("Expected error for invalid site ID")
		}
	})
}

func TestHandler_SetSiteCheckStatus(t *testing.T) {
	handler, testDB, cleanup := setupHandlerTest(t)
	defer cleanup()

	testDB.ClearAllTables(t)

	// Create a test site first
	createReq := dto.CreateSiteRequest{
		Name:     "Test Site",
		URL:      "https://test.com",
		Username: "test",
		Password: "test",
		IsActive: true,
		Strategy: "random",
	}

	created, err := handler.CreateSite(createReq)
	if err != nil {
		t.Fatalf("Failed to create test site: %v", err)
	}

	tests := []struct {
		name    string
		siteID  int64
		status  string
		wantErr bool
	}{
		{
			name:    "Set connected status",
			siteID:  created.ID,
			status:  "connected",
			wantErr: false,
		},
		{
			name:    "Set error status",
			siteID:  created.ID,
			status:  "error",
			wantErr: false,
		},
		{
			name:    "Invalid site ID should fail",
			siteID:  0,
			status:  "connected",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := handler.SetSiteCheckStatus(tt.siteID, tt.status)

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			// Verify status was set by getting the site
			if !tt.wantErr {
				response, err := handler.GetSite(tt.siteID)
				if err != nil {
					t.Errorf("Failed to get site after status update: %v", err)
					return
				}

				if response.Status != tt.status {
					t.Errorf("Expected status '%s', got '%s'", tt.status, response.Status)
				}
			}
		})
	}
}

func TestHandler_TestSiteConnection(t *testing.T) {
	handler, testDB, cleanup := setupHandlerTest(t)
	defer cleanup()

	testDB.ClearAllTables(t)

	// Create a test site first
	createReq := dto.CreateSiteRequest{
		Name:     "Test Site",
		URL:      "https://test.com",
		Username: "test",
		Password: "test",
		IsActive: true,
		Strategy: "random",
	}

	created, err := handler.CreateSite(createReq)
	if err != nil {
		t.Fatalf("Failed to create test site: %v", err)
	}

	tests := []struct {
		name    string
		request dto.TestSiteConnectionRequest
		wantErr bool
	}{
		{
			name: "Valid connection test",
			request: dto.TestSiteConnectionRequest{
				SiteID: created.ID,
			},
			wantErr: false, // Will pass since pipeline service returns nil
		},
		{
			name: "Invalid site ID should fail",
			request: dto.TestSiteConnectionRequest{
				SiteID: 0,
			},
			wantErr: true,
		},
		{
			name: "Non-existent site ID",
			request: dto.TestSiteConnectionRequest{
				SiteID: 999,
			},
			wantErr: false, // Pipeline service returns nil, so no error expected
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response, err := handler.TestSiteConnection(tt.request)

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

			if response == nil {
				t.Error("Response should not be nil")
				return
			}
		})
	}
}
