package handlers

import (
	"testing"

	"Postulator/internal/dto"
)

func TestHandler_CreateTopic(t *testing.T) {
	handler, testDB, cleanup := setupHandlerTest(t)
	defer cleanup()

	tests := []struct {
		name    string
		request dto.CreateTopicRequest
		wantErr bool
		errMsg  string
	}{
		{
			name: "Valid topic creation",
			request: dto.CreateTopicRequest{
				Title:    "Test Topic",
				Keywords: "test,topic,keywords",
				Category: "Technology",
				Tags:     "test,topic",
				IsActive: true,
			},
			wantErr: false,
		},
		{
			name: "Empty title should fail",
			request: dto.CreateTopicRequest{
				Title:    "",
				Keywords: "test,keywords",
				Category: "Technology",
				Tags:     "test",
				IsActive: true,
			},
			wantErr: true,
		},
		{
			name: "Valid topic with minimal data",
			request: dto.CreateTopicRequest{
				Title:    "Minimal Topic",
				IsActive: false,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testDB.ClearAllTables(t)

			response, err := handler.CreateTopic(tt.request)

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

			if response.Title != tt.request.Title {
				t.Errorf("Expected title '%s', got '%s'", tt.request.Title, response.Title)
			}

			if response.Keywords != tt.request.Keywords {
				t.Errorf("Expected keywords '%s', got '%s'", tt.request.Keywords, response.Keywords)
			}

			if response.Category != tt.request.Category {
				t.Errorf("Expected category '%s', got '%s'", tt.request.Category, response.Category)
			}
		})
	}
}

func TestHandler_GetTopic(t *testing.T) {
	handler, testDB, cleanup := setupHandlerTest(t)
	defer cleanup()

	testDB.ClearAllTables(t)

	// Create a test topic first
	createReq := dto.CreateTopicRequest{
		Title:    "Test Topic",
		Keywords: "test,topic",
		Category: "Technology",
		Tags:     "test",
		IsActive: true,
	}

	created, err := handler.CreateTopic(createReq)
	if err != nil {
		t.Fatalf("Failed to create test topic: %v", err)
	}

	tests := []struct {
		name    string
		topicID int64
		wantErr bool
	}{
		{
			name:    "Valid topic ID",
			topicID: created.ID,
			wantErr: false,
		},
		{
			name:    "Invalid topic ID",
			topicID: 0,
			wantErr: true,
		},
		{
			name:    "Non-existent topic ID",
			topicID: 999,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response, err := handler.GetTopic(tt.topicID)

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

			if response.Title != created.Title {
				t.Errorf("Expected title '%s', got '%s'", created.Title, response.Title)
			}
		})
	}
}

func TestHandler_GetTopics(t *testing.T) {
	handler, testDB, cleanup := setupHandlerTest(t)
	defer cleanup()

	testDB.ClearAllTables(t)

	// Create multiple test topics
	topics := []dto.CreateTopicRequest{
		{Title: "Topic 1", Keywords: "key1", Category: "Cat1", Tags: "tag1", IsActive: true},
		{Title: "Topic 2", Keywords: "key2", Category: "Cat2", Tags: "tag2", IsActive: true},
		{Title: "Topic 3", Keywords: "key3", Category: "Cat3", Tags: "tag3", IsActive: false},
	}

	for _, topic := range topics {
		_, err := handler.CreateTopic(topic)
		if err != nil {
			t.Fatalf("Failed to create test topic: %v", err)
		}
	}

	tests := []struct {
		name       string
		pagination dto.PaginationRequest
		wantCount  int
	}{
		{
			name:       "Get all topics",
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
			response, err := handler.GetTopics(tt.pagination)

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if response == nil {
				t.Error("Response should not be nil")
				return
			}

			if len(response.Topics) != tt.wantCount {
				t.Errorf("Expected %d topics, got %d", tt.wantCount, len(response.Topics))
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

func TestHandler_UpdateTopic(t *testing.T) {
	handler, testDB, cleanup := setupHandlerTest(t)
	defer cleanup()

	testDB.ClearAllTables(t)

	// Create a test topic first
	createReq := dto.CreateTopicRequest{
		Title:    "Original Topic",
		Keywords: "original",
		Category: "Original",
		Tags:     "original",
		IsActive: true,
	}

	created, err := handler.CreateTopic(createReq)
	if err != nil {
		t.Fatalf("Failed to create test topic: %v", err)
	}

	tests := []struct {
		name    string
		request dto.UpdateTopicRequest
		wantErr bool
	}{
		{
			name: "Valid update",
			request: dto.UpdateTopicRequest{
				ID:       created.ID,
				Title:    "Updated Topic",
				Keywords: "updated,keywords",
				Category: "Updated",
				Tags:     "updated,tags",
				IsActive: false,
			},
			wantErr: false,
		},
		{
			name: "Invalid ID should fail",
			request: dto.UpdateTopicRequest{
				ID:       0,
				Title:    "Test",
				Keywords: "test",
				Category: "Test",
				Tags:     "test",
				IsActive: true,
			},
			wantErr: true,
		},
		{
			name: "Empty title should fail",
			request: dto.UpdateTopicRequest{
				ID:       created.ID,
				Title:    "",
				Keywords: "test",
				Category: "Test",
				Tags:     "test",
				IsActive: true,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response, err := handler.UpdateTopic(tt.request)

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

			if response.Title != tt.request.Title {
				t.Errorf("Expected title '%s', got '%s'", tt.request.Title, response.Title)
			}

			if response.Keywords != tt.request.Keywords {
				t.Errorf("Expected keywords '%s', got '%s'", tt.request.Keywords, response.Keywords)
			}
		})
	}
}

func TestHandler_DeleteTopic(t *testing.T) {
	handler, testDB, cleanup := setupHandlerTest(t)
	defer cleanup()

	testDB.ClearAllTables(t)

	// Create a test topic first
	createReq := dto.CreateTopicRequest{
		Title:    "Test Topic",
		Keywords: "test",
		Category: "Test",
		Tags:     "test",
		IsActive: true,
	}

	created, err := handler.CreateTopic(createReq)
	if err != nil {
		t.Fatalf("Failed to create test topic: %v", err)
	}

	tests := []struct {
		name    string
		topicID int64
		wantErr bool
	}{
		{
			name:    "Valid deletion",
			topicID: created.ID,
			wantErr: false,
		},
		{
			name:    "Invalid ID should fail",
			topicID: 0,
			wantErr: true,
		},
		{
			name:    "Non-existent ID should not fail",
			topicID: 999,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err = handler.DeleteTopic(tt.topicID)

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
	_, err = handler.GetTopic(created.ID)
	if err == nil {
		t.Error("Expected error when getting deleted topic")
	}
}

func TestHandler_ActivateDeactivateTopic(t *testing.T) {
	handler, testDB, cleanup := setupHandlerTest(t)
	defer cleanup()

	testDB.ClearAllTables(t)

	// Create a test topic first
	createReq := dto.CreateTopicRequest{
		Title:    "Test Topic",
		Keywords: "test",
		Category: "Test",
		Tags:     "test",
		IsActive: false,
	}

	created, err := handler.CreateTopic(createReq)
	if err != nil {
		t.Fatalf("Failed to create test topic: %v", err)
	}

	t.Run("Activate topic", func(t *testing.T) {
		err := handler.ActivateTopic(created.ID)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}

		// Verify activation by getting the topic
		response, err := handler.GetTopic(created.ID)
		if err != nil {
			t.Errorf("Failed to get topic after activation: %v", err)
			return
		}

		if !response.IsActive {
			t.Error("Expected topic to be active after activation")
		}
	})

	t.Run("Deactivate topic", func(t *testing.T) {
		err := handler.DeactivateTopic(created.ID)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}

		// Verify deactivation by getting the topic
		response, err := handler.GetTopic(created.ID)
		if err != nil {
			t.Errorf("Failed to get topic after deactivation: %v", err)
			return
		}

		if response.IsActive {
			t.Error("Expected topic to be inactive after deactivation")
		}
	})

	t.Run("Invalid topic ID should fail", func(t *testing.T) {
		err := handler.ActivateTopic(0)
		if err == nil {
			t.Error("Expected error for invalid topic ID")
		}

		err = handler.DeactivateTopic(0)
		if err == nil {
			t.Error("Expected error for invalid topic ID")
		}
	})
}

func TestHandler_GetActiveTopics(t *testing.T) {
	handler, testDB, cleanup := setupHandlerTest(t)
	defer cleanup()

	testDB.ClearAllTables(t)

	// Create multiple topics with different active status
	topics := []dto.CreateTopicRequest{
		{Title: "Active Topic 1", IsActive: true},
		{Title: "Active Topic 2", IsActive: true},
		{Title: "Inactive Topic", IsActive: false},
	}

	var activeCount int
	for _, topic := range topics {
		_, err := handler.CreateTopic(topic)
		if err != nil {
			t.Fatalf("Failed to create test topic: %v", err)
		}
		if topic.IsActive {
			activeCount++
		}
	}

	t.Run("Get active topics", func(t *testing.T) {
		response, err := handler.GetActiveTopics()

		if err != nil {
			t.Errorf("Unexpected error: %v", err)
			return
		}

		if len(response) != activeCount {
			t.Errorf("Expected %d active topics, got %d", activeCount, len(response))
		}

		// Verify all returned topics are active
		for _, topic := range response {
			if !topic.IsActive {
				t.Errorf("Expected all topics to be active, but topic %d is inactive", topic.ID)
			}
		}
	})
}

func TestHandler_CreateSiteTopic(t *testing.T) {
	handler, testDB, cleanup := setupHandlerTest(t)
	defer cleanup()

	tests := []struct {
		name    string
		request dto.CreateSiteTopicRequest
		wantErr bool
		errMsg  string
	}{
		{
			name: "Valid site topic creation",
			request: dto.CreateSiteTopicRequest{
				SiteID:   1, // From test data
				TopicID:  3, // Use topic 3 which isn't associated with site 1
				IsActive: true,
				Priority: 5,
			},
			wantErr: false,
		},
		{
			name: "Create site topic with minimal data",
			request: dto.CreateSiteTopicRequest{
				SiteID:   2, // From test data
				TopicID:  3, // Use topic 3 which isn't associated with site 2
				IsActive: false,
				Priority: 1,
			},
			wantErr: false,
		},
		{
			name: "Invalid site ID should fail",
			request: dto.CreateSiteTopicRequest{
				SiteID:   0,
				TopicID:  3,
				IsActive: true,
				Priority: 5,
			},
			wantErr: true,
		},
		{
			name: "Invalid topic ID should fail",
			request: dto.CreateSiteTopicRequest{
				SiteID:   1,
				TopicID:  0,
				IsActive: true,
				Priority: 5,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testDB.ClearAllTables(t)
			testDB.InsertTestData(t)

			response, err := handler.CreateSiteTopic(tt.request)

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

			if response.SiteID != tt.request.SiteID {
				t.Errorf("Expected site ID %d, got %d", tt.request.SiteID, response.SiteID)
			}

			if response.TopicID != tt.request.TopicID {
				t.Errorf("Expected topic ID %d, got %d", tt.request.TopicID, response.TopicID)
			}

			if response.Priority != tt.request.Priority {
				t.Errorf("Expected priority %d, got %d", tt.request.Priority, response.Priority)
			}

			if response.IsActive != tt.request.IsActive {
				t.Errorf("Expected IsActive %v, got %v", tt.request.IsActive, response.IsActive)
			}
		})
	}
}

func TestHandler_GetSiteTopics(t *testing.T) {
	handler, testDB, cleanup := setupHandlerTest(t)
	defer cleanup()

	testDB.ClearAllTables(t)
	testDB.InsertTestData(t)

	tests := []struct {
		name       string
		siteID     int64
		pagination dto.PaginationRequest
		wantErr    bool
		wantCount  int
	}{
		{
			name:       "Get site topics for existing site",
			siteID:     1, // From test data
			pagination: dto.PaginationRequest{Page: 1, Limit: 10},
			wantErr:    false,
			wantCount:  2, // From test data site 1 has 2 topic associations
		},
		{
			name:       "Get site topics with pagination",
			siteID:     1,
			pagination: dto.PaginationRequest{Page: 1, Limit: 1},
			wantErr:    false,
			wantCount:  1,
		},
		{
			name:       "Get site topics with offset",
			siteID:     1,
			pagination: dto.PaginationRequest{Page: 2, Limit: 1},
			wantErr:    false,
			wantCount:  1,
		},
		{
			name:       "Get site topics for non-existing site",
			siteID:     999,
			pagination: dto.PaginationRequest{Page: 1, Limit: 10},
			wantErr:    false,
			wantCount:  0,
		},
		{
			name:       "Invalid site ID should fail",
			siteID:     0,
			pagination: dto.PaginationRequest{Page: 1, Limit: 10},
			wantErr:    true,
		},
		{
			name:       "Default pagination",
			siteID:     1,
			pagination: dto.PaginationRequest{},
			wantErr:    false,
			wantCount:  2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response, err := handler.GetSiteTopics(tt.siteID, tt.pagination)

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

			if len(response.SiteTopics) != tt.wantCount {
				t.Errorf("Expected %d site topics, got %d", tt.wantCount, len(response.SiteTopics))
			}

			if response.Pagination == nil {
				t.Error("Pagination should not be nil")
				return
			}

			// Verify all returned site topics belong to the requested site
			for _, st := range response.SiteTopics {
				if st.SiteID != tt.siteID {
					t.Errorf("Expected site ID %d, got %d", tt.siteID, st.SiteID)
				}
			}
		})
	}
}

func TestHandler_GetTopicSites(t *testing.T) {
	handler, testDB, cleanup := setupHandlerTest(t)
	defer cleanup()

	testDB.ClearAllTables(t)
	testDB.InsertTestData(t)

	tests := []struct {
		name       string
		topicID    int64
		pagination dto.PaginationRequest
		wantErr    bool
		wantCount  int
	}{
		{
			name:       "Get topic sites for existing topic",
			topicID:    1, // From test data
			pagination: dto.PaginationRequest{Page: 1, Limit: 10},
			wantErr:    false,
			wantCount:  2, // From test data topic 1 is associated with sites 1 and 2
		},
		{
			name:       "Get topic sites with pagination",
			topicID:    1,
			pagination: dto.PaginationRequest{Page: 1, Limit: 1},
			wantErr:    false,
			wantCount:  1,
		},
		{
			name:       "Get topic sites for non-existing topic",
			topicID:    999,
			pagination: dto.PaginationRequest{Page: 1, Limit: 10},
			wantErr:    false,
			wantCount:  0,
		},
		{
			name:       "Invalid topic ID should fail",
			topicID:    0,
			pagination: dto.PaginationRequest{Page: 1, Limit: 10},
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response, err := handler.GetTopicSites(tt.topicID, tt.pagination)

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

			if len(response.SiteTopics) != tt.wantCount {
				t.Errorf("Expected %d topic sites, got %d", tt.wantCount, len(response.SiteTopics))
			}

			// Verify all returned site topics belong to the requested topic
			for _, st := range response.SiteTopics {
				if st.TopicID != tt.topicID {
					t.Errorf("Expected topic ID %d, got %d", tt.topicID, st.TopicID)
				}
			}
		})
	}
}

func TestHandler_UpdateSiteTopic(t *testing.T) {
	handler, testDB, cleanup := setupHandlerTest(t)
	defer cleanup()

	testDB.ClearAllTables(t)
	testDB.InsertTestData(t)

	// Create a test site topic first using unique combination
	createReq := dto.CreateSiteTopicRequest{
		SiteID:   1,
		TopicID:  3, // Use topic 3 which isn't associated with site 1
		IsActive: true,
		Priority: 5,
	}

	created, err := handler.CreateSiteTopic(createReq)
	if err != nil {
		t.Fatalf("Failed to create test site topic: %v", err)
	}

	tests := []struct {
		name    string
		request dto.UpdateSiteTopicRequest
		wantErr bool
	}{
		{
			name: "Valid update",
			request: dto.UpdateSiteTopicRequest{
				ID:       created.ID,
				SiteID:   1,
				TopicID:  3,
				IsActive: false,
				Priority: 8,
			},
			wantErr: false,
		},
		{
			name: "Invalid ID should fail",
			request: dto.UpdateSiteTopicRequest{
				ID:       0,
				SiteID:   1,
				TopicID:  3,
				IsActive: true,
				Priority: 5,
			},
			wantErr: true,
		},
		{
			name: "Invalid site ID should fail",
			request: dto.UpdateSiteTopicRequest{
				ID:       created.ID,
				SiteID:   0,
				TopicID:  3,
				IsActive: true,
				Priority: 5,
			},
			wantErr: true,
		},
		{
			name: "Invalid topic ID should fail",
			request: dto.UpdateSiteTopicRequest{
				ID:       created.ID,
				SiteID:   1,
				TopicID:  0,
				IsActive: true,
				Priority: 5,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response, err := handler.UpdateSiteTopic(tt.request)

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

			if response.Priority != tt.request.Priority {
				t.Errorf("Expected priority %d, got %d", tt.request.Priority, response.Priority)
			}

			if response.IsActive != tt.request.IsActive {
				t.Errorf("Expected IsActive %v, got %v", tt.request.IsActive, response.IsActive)
			}
		})
	}
}

func TestHandler_DeleteSiteTopic(t *testing.T) {
	handler, testDB, cleanup := setupHandlerTest(t)
	defer cleanup()

	testDB.ClearAllTables(t)
	testDB.InsertTestData(t)

	// Create a test site topic first using unique combination
	createReq := dto.CreateSiteTopicRequest{
		SiteID:   1,
		TopicID:  3, // Use topic 3 which isn't associated with site 1
		IsActive: true,
		Priority: 5,
	}

	created, err := handler.CreateSiteTopic(createReq)
	if err != nil {
		t.Fatalf("Failed to create test site topic: %v", err)
	}

	tests := []struct {
		name        string
		siteTopicID int64
		wantErr     bool
	}{
		{
			name:        "Valid deletion",
			siteTopicID: created.ID,
			wantErr:     false,
		},
		{
			name:        "Invalid ID should fail",
			siteTopicID: 0,
			wantErr:     true,
		},
		{
			name:        "Non-existent ID should not fail",
			siteTopicID: 999,
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := handler.DeleteSiteTopic(tt.siteTopicID)

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
}

func TestHandler_DeleteSiteTopicBySiteAndTopic(t *testing.T) {
	handler, testDB, cleanup := setupHandlerTest(t)
	defer cleanup()

	testDB.ClearAllTables(t)
	testDB.InsertTestData(t)

	tests := []struct {
		name    string
		siteID  int64
		topicID int64
		wantErr bool
	}{
		{
			name:    "Valid deletion by site and topic",
			siteID:  1, // From test data
			topicID: 1, // From test data
			wantErr: false,
		},
		{
			name:    "Invalid site ID should fail",
			siteID:  0,
			topicID: 1,
			wantErr: true,
		},
		{
			name:    "Invalid topic ID should fail",
			siteID:  1,
			topicID: 0,
			wantErr: true,
		},
		{
			name:    "Non-existent association should not fail",
			siteID:  999,
			topicID: 999,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := handler.DeleteSiteTopicBySiteAndTopic(tt.siteID, tt.topicID)

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
}

func TestHandler_ActivateDeactivateSiteTopic(t *testing.T) {
	handler, testDB, cleanup := setupHandlerTest(t)
	defer cleanup()

	testDB.ClearAllTables(t)
	testDB.InsertTestData(t)

	// Create a test site topic first using unique combination
	createReq := dto.CreateSiteTopicRequest{
		SiteID:   1,
		TopicID:  3, // Use topic 3 which isn't associated with site 1
		IsActive: false,
		Priority: 5,
	}

	created, err := handler.CreateSiteTopic(createReq)
	if err != nil {
		t.Fatalf("Failed to create test site topic: %v", err)
	}

	t.Run("Activate site topic", func(t *testing.T) {
		err := handler.ActivateSiteTopic(created.ID)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
	})

	t.Run("Deactivate site topic", func(t *testing.T) {
		err := handler.DeactivateSiteTopic(created.ID)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
	})

	t.Run("Invalid site topic ID should fail for activation", func(t *testing.T) {
		err := handler.ActivateSiteTopic(0)
		if err == nil {
			t.Error("Expected error for invalid site topic ID")
		}
	})

	t.Run("Invalid site topic ID should fail for deactivation", func(t *testing.T) {
		err := handler.DeactivateSiteTopic(0)
		if err == nil {
			t.Error("Expected error for invalid site topic ID")
		}
	})
}

func TestHandler_TopicStats(t *testing.T) {
	handler, testDB, cleanup := setupHandlerTest(t)
	defer cleanup()

	testDB.ClearAllTables(t)
	testDB.InsertTestData(t)

	t.Run("Get topic stats", func(t *testing.T) {
		response, err := handler.GetTopicStats(1) // Site ID from test data
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
			return
		}

		if response == nil {
			t.Error("Response should not be nil")
			return
		}

		if response.SiteID != 1 {
			t.Errorf("Expected site ID 1, got %d", response.SiteID)
		}

		// Basic validation - exact values depend on test data
		if response.TotalTopics < 0 {
			t.Error("Total topics should not be negative")
		}

		if response.ActiveTopics < 0 {
			t.Error("Active topics should not be negative")
		}
	})

	t.Run("Invalid site ID should fail", func(t *testing.T) {
		_, err := handler.GetTopicStats(0)
		if err == nil {
			t.Error("Expected error for invalid site ID")
		}
	})
}

func TestHandler_EdgeCases(t *testing.T) {
	handler, testDB, cleanup := setupHandlerTest(t)
	defer cleanup()

	testDB.ClearAllTables(t)

	tests := []struct {
		name string
		test func(t *testing.T)
	}{
		{
			name: "Create site topic with invalid site ID",
			test: func(t *testing.T) {
				req := dto.CreateSiteTopicRequest{
					SiteID:   0,
					TopicID:  1,
					IsActive: true,
					Priority: 5,
				}

				_, err := handler.CreateSiteTopic(req)
				if err == nil {
					t.Error("Expected error for invalid site ID")
				}
			},
		},
		{
			name: "Create site topic with invalid topic ID",
			test: func(t *testing.T) {
				req := dto.CreateSiteTopicRequest{
					SiteID:   1,
					TopicID:  0,
					IsActive: true,
					Priority: 5,
				}

				_, err := handler.CreateSiteTopic(req)
				if err == nil {
					t.Error("Expected error for invalid topic ID")
				}
			},
		},
		{
			name: "Get topics by site with invalid site ID",
			test: func(t *testing.T) {
				_, err := handler.GetTopicsBySiteID(0, dto.PaginationRequest{})
				if err == nil {
					t.Error("Expected error for invalid site ID")
				}
			},
		},
		{
			name: "Delete site topic by site and topic with invalid IDs",
			test: func(t *testing.T) {
				err := handler.DeleteSiteTopicBySiteAndTopic(0, 0)
				if err == nil {
					t.Error("Expected error for invalid IDs")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, tt.test)
	}
}
