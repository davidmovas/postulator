package handlers

import (
	"context"
	"testing"

	"Postulator/internal/dto"
	"Postulator/internal/repository"
	"Postulator/internal/testhelpers"
)

func setupHandlerTest(t *testing.T) (*Handler, *testhelpers.TestDB, func()) {
	testDB := testhelpers.SetupTestDB(t)

	repo := repository.NewRepository(testDB.DB)

	handler := &Handler{
		repo: repo,
		ctx:  context.Background(),
	}

	cleanup := func() {
		_ = testDB.Close()
	}

	return handler, testDB, cleanup
}

func TestHandler_CreatePrompt(t *testing.T) {
	handler, testDB, cleanup := setupHandlerTest(t)
	defer cleanup()

	tests := []struct {
		name    string
		request dto.CreatePromptRequest
		wantErr bool
		errMsg  string
	}{
		{
			name: "Valid prompt creation",
			request: dto.CreatePromptRequest{
				Name:   "Test Prompt",
				System: "This is a test system prompt with {{title}} placeholder",
				User:   "This is a test user prompt with {{title}} placeholder",
			},
			wantErr: false,
		},
		{
			name: "Valid user prompt creation",
			request: dto.CreatePromptRequest{
				Name:   "User Prompt",
				System: "This is a test system prompt with {{title}} placeholder",
				User:   "This is a test user prompt with {{title}} placeholder",
			},
			wantErr: false,
		},
		{
			name: "Empty name should fail",
			request: dto.CreatePromptRequest{
				Name:   "",
				System: "This is a test system prompt with {{title}} placeholder",
				User:   "This is a test user prompt with {{title}} placeholder",
			},
			wantErr: true,
			errMsg:  "name is required",
		},
		{
			name: "Empty system prompt should fail",
			request: dto.CreatePromptRequest{
				Name:   "Test",
				System: "",
				User:   "This is a test user prompt with {{title}} placeholder",
			},
			wantErr: true,
			errMsg:  "system is required",
		},
		{
			name: "Empty user prompt should fail",
			request: dto.CreatePromptRequest{
				Name:   "Test",
				System: "This is a test system prompt with {{title}} placeholder",
				User:   "",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testDB.ClearAllTables(t)

			response, err := handler.CreatePrompt(tt.request)

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

			if response.System != tt.request.System {
				t.Errorf("Expected system prompt '%s', got '%s'", tt.request.System, response.System)
			}

			if response.User != tt.request.User {
				t.Errorf("Expected user prompt '%s', got '%s'", tt.request.User, response.User)
			}
		})
	}
}

func TestHandler_GetPrompt(t *testing.T) {
	handler, testDB, cleanup := setupHandlerTest(t)
	defer cleanup()

	testDB.ClearAllTables(t)

	// Create a test prompt first
	createReq := dto.CreatePromptRequest{
		Name:   "Test Prompt",
		System: "Test system content with {{placeholder}}",
		User:   "Test user content with {{placeholder}}",
	}

	created, err := handler.CreatePrompt(createReq)
	if err != nil {
		t.Fatalf("Failed to create test prompt: %v", err)
	}

	tests := []struct {
		name     string
		promptID int64
		wantErr  bool
	}{
		{
			name:     "Valid prompt ID",
			promptID: created.ID,
			wantErr:  false,
		},
		{
			name:     "Invalid prompt ID",
			promptID: 0,
			wantErr:  true,
		},
		{
			name:     "Non-existent prompt ID",
			promptID: 999,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var response *dto.PromptResponse
			response, err = handler.GetPrompt(tt.promptID)

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

func TestHandler_GetPrompts(t *testing.T) {
	handler, testDB, cleanup := setupHandlerTest(t)
	defer cleanup()

	testDB.ClearAllTables(t)

	// Create multiple test prompts
	prompts := []dto.CreatePromptRequest{
		{Name: "Prompt 1", System: "System content 1", User: "User content 1"},
		{Name: "Prompt 2", System: "System content 2", User: "User content 2"},
		{Name: "Prompt 3", System: "System content 3", User: "User content 3"},
	}

	for _, prompt := range prompts {
		_, err := handler.CreatePrompt(prompt)
		if err != nil {
			t.Fatalf("Failed to create test prompt: %v", err)
		}
	}

	tests := []struct {
		name       string
		pagination dto.PaginationRequest
		wantCount  int
	}{
		{
			name:       "Get all prompts",
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
			response, err := handler.GetPrompts(tt.pagination)

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if response == nil {
				t.Error("Response should not be nil")
				return
			}

			if len(response.Prompts) != tt.wantCount {
				t.Errorf("Expected %d prompts, got %d", tt.wantCount, len(response.Prompts))
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

func TestHandler_UpdatePrompt(t *testing.T) {
	handler, testDB, cleanup := setupHandlerTest(t)
	defer cleanup()

	testDB.ClearAllTables(t)

	// Create a test prompt first
	createReq := dto.CreatePromptRequest{
		Name:   "Original Prompt",
		System: "Original system content with {{placeholder}}",
		User:   "Original user content with {{placeholder}}",
	}

	created, err := handler.CreatePrompt(createReq)
	if err != nil {
		t.Fatalf("Failed to create test prompt: %v", err)
	}

	tests := []struct {
		name    string
		request dto.UpdatePromptRequest
		wantErr bool
	}{
		{
			name: "Valid update",
			request: dto.UpdatePromptRequest{
				ID:      created.ID,
				Name:    "Updated Prompt",
				Content: "Updated content with {{new_placeholder}}",
				Type:    "user",
			},
			wantErr: false,
		},
		{
			name: "Invalid ID should fail",
			request: dto.UpdatePromptRequest{
				ID:      0,
				Name:    "Test",
				Content: "Content",
				Type:    "system",
			},
			wantErr: true,
		},
		{
			name: "Empty name should fail",
			request: dto.UpdatePromptRequest{
				ID:      created.ID,
				Name:    "",
				Content: "Content",
				Type:    "system",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var response *dto.PromptResponse
			response, err = handler.UpdatePrompt(tt.request)

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
		})
	}
}

func TestHandler_DeletePrompt(t *testing.T) {
	handler, testDB, cleanup := setupHandlerTest(t)
	defer cleanup()

	testDB.ClearAllTables(t)

	// Create a test prompt first
	createReq := dto.CreatePromptRequest{
		Name:   "Test Prompt",
		System: "Test system content with {{placeholder}}",
		User:   "Test user content with {{placeholder}}",
	}

	created, err := handler.CreatePrompt(createReq)
	if err != nil {
		t.Fatalf("Failed to create test prompt: %v", err)
	}

	tests := []struct {
		name     string
		promptID int64
		wantErr  bool
	}{
		{
			name:     "Valid deletion",
			promptID: created.ID,
			wantErr:  false,
		},
		{
			name:     "Invalid ID should fail",
			promptID: 0,
			wantErr:  true,
		},
		{
			name:     "Non-existent ID should not fail",
			promptID: 999,
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err = handler.DeletePrompt(tt.promptID)

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
	_, err = handler.GetPrompt(created.ID)
	if err == nil {
		t.Error("Expected error when getting deleted prompt")
	}
}

func TestHandler_DefaultPrompt(t *testing.T) {
	handler, testDB, cleanup := setupHandlerTest(t)
	defer cleanup()

	testDB.ClearAllTables(t)

	// Create test prompts
	prompt1, err := handler.CreatePrompt(dto.CreatePromptRequest{
		Name:   "Prompt 1",
		System: "System content 1",
		User:   "User content 1",
	})
	if err != nil {
		t.Fatalf("Failed to create test prompt: %v", err)
	}

	// Test setting default prompt
	t.Run("Set default prompt", func(t *testing.T) {
		err = handler.SetDefaultPrompt(dto.SetDefaultPromptRequest{ID: prompt1.ID})
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}

		// Get default prompt
		var defaultPrompt *dto.PromptResponse
		defaultPrompt, err = handler.GetDefaultPrompt()
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
			return
		}

		if defaultPrompt.ID != prompt1.ID {
			t.Errorf("Expected default prompt ID %d, got %d", prompt1.ID, defaultPrompt.ID)
		}
	})

	// Test invalid default prompt ID
	t.Run("Invalid default prompt ID", func(t *testing.T) {
		err = handler.SetDefaultPrompt(dto.SetDefaultPromptRequest{ID: 0})
		if err == nil {
			t.Error("Expected error for invalid prompt ID")
		}
	})
}

func TestHandler_SitePrompt_CRUD(t *testing.T) {
	handler, testDB, cleanup := setupHandlerTest(t)
	defer cleanup()

	testDB.ClearAllTables(t)
	testDB.InsertTestData(t)

	// Create a test prompt first
	prompt, err := handler.CreatePrompt(dto.CreatePromptRequest{
		Name:   "Site Prompt",
		System: "System content with {{placeholder}}",
		User:   "User content with {{placeholder}}",
	})
	if err != nil {
		t.Fatalf("Failed to create test prompt: %v", err)
	}

	var createdSitePrompt *dto.SitePromptResponse

	// Test create site prompt
	t.Run("Create site prompt", func(t *testing.T) {
		req := dto.CreateSitePromptRequest{
			SiteID:   1, // From test data
			PromptID: prompt.ID,
			IsActive: true,
		}

		var response *dto.SitePromptResponse
		response, err = handler.CreateSitePrompt(req)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
			return
		}

		if response == nil {
			t.Error("Response should not be nil")
			return
		}

		createdSitePrompt = response

		if response.SiteID != req.SiteID {
			t.Errorf("Expected site ID %d, got %d", req.SiteID, response.SiteID)
		}

		if response.PromptID != req.PromptID {
			t.Errorf("Expected prompt ID %d, got %d", req.PromptID, response.PromptID)
		}

		if response.IsActive != req.IsActive {
			t.Errorf("Expected IsActive %v, got %v", req.IsActive, response.IsActive)
		}
	})

	// Test get site prompt
	t.Run("Get site prompt", func(t *testing.T) {
		var response *dto.SitePromptResponse
		response, err = handler.GetSitePrompt(1)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
			return
		}

		if response == nil {
			t.Error("Response should not be nil")
			return
		}

		if response.ID != createdSitePrompt.ID {
			t.Errorf("Expected ID %d, got %d", createdSitePrompt.ID, response.ID)
		}
	})

	// Test update site prompt
	t.Run("Update site prompt", func(t *testing.T) {
		req := dto.UpdateSitePromptRequest{
			ID:       createdSitePrompt.ID,
			SiteID:   1,
			PromptID: prompt.ID,
			IsActive: false,
		}

		response, err := handler.UpdateSitePrompt(req)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
			return
		}

		if response == nil {
			t.Error("Response should not be nil")
			return
		}

		if response.IsActive != false {
			t.Error("Expected IsActive to be false")
		}
	})

	// Test activate/deactivate site prompt
	t.Run("Activate site prompt", func(t *testing.T) {
		err := handler.ActivateSitePrompt(createdSitePrompt.ID)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
	})

	t.Run("Deactivate site prompt", func(t *testing.T) {
		err := handler.DeactivateSitePrompt(createdSitePrompt.ID)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
	})

	// Test get prompt sites
	t.Run("Get prompt sites", func(t *testing.T) {
		var response *dto.SitePromptListResponse
		response, err = handler.GetPromptSites(prompt.ID, dto.PaginationRequest{Page: 1, Limit: 10})
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
			return
		}

		if response == nil {
			t.Error("Response should not be nil")
			return
		}

		if len(response.SitePrompts) != 1 {
			t.Errorf("Expected 1 site prompt, got %d", len(response.SitePrompts))
		}
	})

	// Test delete site prompt
	t.Run("Delete site prompt", func(t *testing.T) {
		err = handler.DeleteSitePrompt(createdSitePrompt.ID)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}

		// Verify deletion
		_, err = handler.GetSitePrompt(1)
		if err == nil {
			t.Error("Expected error when getting deleted site prompt")
		}
	})
}

func TestHandler_SitePrompt_EdgeCases(t *testing.T) {
	handler, testDB, cleanup := setupHandlerTest(t)
	defer cleanup()

	testDB.ClearAllTables(t)

	tests := []struct {
		name string
		test func(t *testing.T)
	}{
		{
			name: "Create site prompt with invalid site ID",
			test: func(t *testing.T) {
				req := dto.CreateSitePromptRequest{
					SiteID:   0,
					PromptID: 1,
					IsActive: true,
				}

				_, err := handler.CreateSitePrompt(req)
				if err == nil {
					t.Error("Expected error for invalid site ID")
				}
			},
		},
		{
			name: "Create site prompt with invalid prompt ID",
			test: func(t *testing.T) {
				req := dto.CreateSitePromptRequest{
					SiteID:   1,
					PromptID: 0,
					IsActive: true,
				}

				_, err := handler.CreateSitePrompt(req)
				if err == nil {
					t.Error("Expected error for invalid prompt ID")
				}
			},
		},
		{
			name: "Get site prompt with invalid site ID",
			test: func(t *testing.T) {
				_, err := handler.GetSitePrompt(0)
				if err == nil {
					t.Error("Expected error for invalid site ID")
				}
			},
		},
		{
			name: "Update site prompt with invalid ID",
			test: func(t *testing.T) {
				req := dto.UpdateSitePromptRequest{
					ID:       0,
					SiteID:   1,
					PromptID: 1,
					IsActive: true,
				}

				_, err := handler.UpdateSitePrompt(req)
				if err == nil {
					t.Error("Expected error for invalid site prompt ID")
				}
			},
		},
		{
			name: "Delete site prompt by site with invalid site ID",
			test: func(t *testing.T) {
				err := handler.DeleteSitePromptBySite(0)
				if err == nil {
					t.Error("Expected error for invalid site ID")
				}
			},
		},
		{
			name: "Activate site prompt with invalid ID",
			test: func(t *testing.T) {
				err := handler.ActivateSitePrompt(0)
				if err == nil {
					t.Error("Expected error for invalid site prompt ID")
				}
			},
		},
		{
			name: "Get prompt sites with invalid prompt ID",
			test: func(t *testing.T) {
				_, err := handler.GetPromptSites(0, dto.PaginationRequest{})
				if err == nil {
					t.Error("Expected error for invalid prompt ID")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, tt.test)
	}
}
