package handlers

import (
	"testing"

	"Postulator/internal/dto"
	"Postulator/internal/testutil"
)

func TestHandler_CreateTopic(t *testing.T) {
	handler, db, cleanup := testutil.setupHandlerTest(t)
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
			},
			wantErr: true,
		},
		{
			name: "Valid topic with minimal data",
			request: dto.CreateTopicRequest{
				Title: "Minimal Topic",
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
		{Title: "Topic 1", Keywords: "key1", Category: "Cat1", Tags: "tag1"},
		{Title: "Topic 2", Keywords: "key2", Category: "Cat2", Tags: "tag2"},
		{Title: "Topic 3", Keywords: "key3", Category: "Cat3", Tags: "tag3"},
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
				Priority: 5,
			},
			wantErr: false,
		},
		{
			name: "Create site topic with minimal data",
			request: dto.CreateSiteTopicRequest{
				SiteID:   2, // From test data
				TopicID:  3, // Use topic 3 which isn't associated with site 2
				Priority: 1,
			},
			wantErr: false,
		},
		{
			name: "Invalid site ID should fail",
			request: dto.CreateSiteTopicRequest{
				SiteID:   0,
				TopicID:  3,
				Priority: 5,
			},
			wantErr: true,
		},
		{
			name: "Invalid topic ID should fail",
			request: dto.CreateSiteTopicRequest{
				SiteID:   1,
				TopicID:  0,
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

// Tests for TopicsImport functionality

func TestHandler_TopicsImport_TXT_Format(t *testing.T) {
	handler, testDB, cleanup := setupHandlerTest(t)
	defer cleanup()

	tests := []struct {
		name        string
		siteID      int64
		fileContent string
		previewOnly bool
		wantErr     bool
		expectedNew int
		expectedDup int
		expectedErr int
	}{
		{
			name:   "Valid TXT import preview",
			siteID: 1,
			fileContent: `New Topic 1
New Topic 2
Another Topic
Programming Guide`,
			previewOnly: true,
			wantErr:     false,
			expectedNew: 4,
			expectedDup: 0,
			expectedErr: 0,
		},
		{
			name:   "TXT import with duplicates within file",
			siteID: 1,
			fileContent: `Unique Topic
Duplicate Topic
Unique Topic 2
Duplicate Topic`,
			previewOnly: true,
			wantErr:     false,
			expectedNew: 2,
			expectedDup: 1,
			expectedErr: 0,
		},
		{
			name:   "TXT import with existing topics in DB",
			siteID: 1,
			fileContent: `AI Technology
New Unique Topic
Web Development`,
			previewOnly: true,
			wantErr:     false,
			expectedNew: 1,
			expectedDup: 0,
			expectedErr: 0,
		},
		{
			name:   "TXT import with empty lines",
			siteID: 1,
			fileContent: `Topic 1

Topic 2

Topic 3
`,
			previewOnly: true,
			wantErr:     false,
			expectedNew: 3,
			expectedDup: 0,
			expectedErr: 0,
		},
		{
			name:   "Actual TXT import",
			siteID: 1,
			fileContent: `Import Topic 1
Import Topic 2`,
			previewOnly: false,
			wantErr:     false,
			expectedNew: 2,
			expectedDup: 0,
			expectedErr: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testDB.ClearAllTables(t)
			testDB.InsertTestData(t)

			req := dto.TopicsImportRequest{
				SiteID:      tt.siteID,
				FileContent: tt.fileContent,
				FileFormat:  "txt",
				PreviewOnly: tt.previewOnly,
			}

			result, err := handler.TopicsImport(tt.siteID, req)

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

			if tt.previewOnly {
				preview, ok := result.(*dto.TopicsImportPreview)
				if !ok {
					t.Error("Expected TopicsImportPreview for preview mode")
					return
				}

				if preview.ValidTopics != tt.expectedNew {
					t.Errorf("Expected %d new topics, got %d", tt.expectedNew, preview.ValidTopics)
				}

				if preview.Duplicates != tt.expectedDup {
					t.Errorf("Expected %d duplicates, got %d", tt.expectedDup, preview.Duplicates)
				}

				if preview.Errors != tt.expectedErr {
					t.Errorf("Expected %d errors, got %d", tt.expectedErr, preview.Errors)
				}
			} else {
				importResult, ok := result.(*dto.TopicsImportResult)
				if !ok {
					t.Error("Expected TopicsImportResult for import mode")
					return
				}

				if importResult.CreatedTopics != tt.expectedNew {
					t.Errorf("Expected %d created topics, got %d", tt.expectedNew, importResult.CreatedTopics)
				}
			}
		})
	}
}

func TestHandler_TopicsImport_CSV_Format(t *testing.T) {
	handler, testDB, cleanup := setupHandlerTest(t)
	defer cleanup()

	tests := []struct {
		name        string
		siteID      int64
		fileContent string
		previewOnly bool
		wantErr     bool
		expectedNew int
		expectedErr int
	}{
		{
			name:   "Valid CSV import preview",
			siteID: 1,
			fileContent: `"Machine Learning Basics","ml,ai,basics","Technology","ml,tech"
"React Development","react,js,frontend","Programming","react,web"
"Python Guide","python,programming","Development","python,code"`,
			previewOnly: true,
			wantErr:     false,
			expectedNew: 3,
			expectedErr: 0,
		},
		{
			name:   "CSV with minimal columns",
			siteID: 1,
			fileContent: `"Topic 1"
"Topic 2"
"Topic 3"`,
			previewOnly: true,
			wantErr:     false,
			expectedNew: 3,
			expectedErr: 0,
		},
		{
			name:   "CSV with empty title",
			siteID: 1,
			fileContent: `"Valid Topic","keywords","category","tags"
"","invalid","invalid","invalid"
"Another Valid Topic","more","keywords","tags"`,
			previewOnly: true,
			wantErr:     false,
			expectedNew: 2,
			expectedErr: 1,
		},
		{
			name:   "Actual CSV import",
			siteID: 1,
			fileContent: `"CSV Topic 1","csv,keywords","CSV Category","csv,tags"
"CSV Topic 2","more,csv","Another Category","more,tags"`,
			previewOnly: false,
			wantErr:     false,
			expectedNew: 2,
			expectedErr: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testDB.ClearAllTables(t)
			testDB.InsertTestData(t)

			req := dto.TopicsImportRequest{
				SiteID:      tt.siteID,
				FileContent: tt.fileContent,
				FileFormat:  "csv",
				PreviewOnly: tt.previewOnly,
			}

			result, err := handler.TopicsImport(tt.siteID, req)

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

			if tt.previewOnly {
				preview, ok := result.(*dto.TopicsImportPreview)
				if !ok {
					t.Error("Expected TopicsImportPreview for preview mode")
					return
				}

				if preview.ValidTopics != tt.expectedNew {
					t.Errorf("Expected %d new topics, got %d", tt.expectedNew, preview.ValidTopics)
				}

				if preview.Errors != tt.expectedErr {
					t.Errorf("Expected %d errors, got %d", tt.expectedErr, preview.Errors)
				}
			} else {
				importResult, ok := result.(*dto.TopicsImportResult)
				if !ok {
					t.Error("Expected TopicsImportResult for import mode")
					return
				}

				if importResult.CreatedTopics != tt.expectedNew {
					t.Errorf("Expected %d created topics, got %d", tt.expectedNew, importResult.CreatedTopics)
				}
			}
		})
	}
}

func TestHandler_TopicsImport_JSONL_Format(t *testing.T) {
	handler, testDB, cleanup := setupHandlerTest(t)
	defer cleanup()

	tests := []struct {
		name        string
		siteID      int64
		fileContent string
		previewOnly bool
		wantErr     bool
		expectedNew int
		expectedErr int
	}{
		{
			name:   "Valid JSONL import preview",
			siteID: 1,
			fileContent: `{"title":"Docker Containers","keywords":"docker,containers,devops","category":"DevOps","tags":"docker,containers"}
{"title":"Kubernetes Guide","keywords":"k8s,kubernetes,orchestration","category":"DevOps","tags":"k8s,devops"}
{"title":"Minimal Topic"}`,
			previewOnly: true,
			wantErr:     false,
			expectedNew: 3,
			expectedErr: 0,
		},
		{
			name:   "JSONL with invalid JSON",
			siteID: 1,
			fileContent: `{"title":"Valid Topic","keywords":"valid"}
{invalid json}
{"title":"Another Valid Topic"}`,
			previewOnly: true,
			wantErr:     false,
			expectedNew: 2,
			expectedErr: 1,
		},
		{
			name:   "JSONL with empty title",
			siteID: 1,
			fileContent: `{"title":"","keywords":"invalid"}
{"title":"Valid Topic","keywords":"valid"}`,
			previewOnly: true,
			wantErr:     false,
			expectedNew: 1,
			expectedErr: 1,
		},
		{
			name:   "Actual JSONL import",
			siteID: 1,
			fileContent: `{"title":"JSONL Topic 1","keywords":"jsonl,test","category":"Test","tags":"jsonl"}
{"title":"JSONL Topic 2","keywords":"more,jsonl","category":"Test","tags":"test"}`,
			previewOnly: false,
			wantErr:     false,
			expectedNew: 2,
			expectedErr: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testDB.ClearAllTables(t)
			testDB.InsertTestData(t)

			req := dto.TopicsImportRequest{
				SiteID:      tt.siteID,
				FileContent: tt.fileContent,
				FileFormat:  "jsonl",
				PreviewOnly: tt.previewOnly,
			}

			result, err := handler.TopicsImport(tt.siteID, req)

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

			if tt.previewOnly {
				preview, ok := result.(*dto.TopicsImportPreview)
				if !ok {
					t.Error("Expected TopicsImportPreview for preview mode")
					return
				}

				if preview.ValidTopics != tt.expectedNew {
					t.Errorf("Expected %d new topics, got %d", tt.expectedNew, preview.ValidTopics)
				}

				if preview.Errors != tt.expectedErr {
					t.Errorf("Expected %d errors, got %d", tt.expectedErr, preview.Errors)
				}
			} else {
				importResult, ok := result.(*dto.TopicsImportResult)
				if !ok {
					t.Error("Expected TopicsImportResult for import mode")
					return
				}

				if importResult.CreatedTopics != tt.expectedNew {
					t.Errorf("Expected %d created topics, got %d", tt.expectedNew, importResult.CreatedTopics)
				}
			}
		})
	}
}

func TestHandler_TopicsImport_ErrorCases(t *testing.T) {
	handler, testDB, cleanup := setupHandlerTest(t)
	defer cleanup()

	tests := []struct {
		name    string
		siteID  int64
		request dto.TopicsImportRequest
		wantErr bool
		errMsg  string
	}{
		{
			name:   "Invalid site ID",
			siteID: 0,
			request: dto.TopicsImportRequest{
				SiteID:      0,
				FileContent: "Test Topic",
				FileFormat:  "txt",
				PreviewOnly: true,
			},
			wantErr: true,
			errMsg:  "invalid site ID",
		},
		{
			name:   "Empty file content",
			siteID: 1,
			request: dto.TopicsImportRequest{
				SiteID:      1,
				FileContent: "",
				FileFormat:  "txt",
				PreviewOnly: true,
			},
			wantErr: true,
			errMsg:  "file content is required",
		},
		{
			name:   "Empty file format",
			siteID: 1,
			request: dto.TopicsImportRequest{
				SiteID:      1,
				FileContent: "Test Topic",
				FileFormat:  "",
				PreviewOnly: true,
			},
			wantErr: true,
			errMsg:  "file format is required",
		},
		{
			name:   "Unsupported file format",
			siteID: 1,
			request: dto.TopicsImportRequest{
				SiteID:      1,
				FileContent: "Test Topic",
				FileFormat:  "xml",
				PreviewOnly: true,
			},
			wantErr: false, // Should return result with errors, not fail completely
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testDB.ClearAllTables(t)
			testDB.InsertTestData(t)

			result, err := handler.TopicsImport(tt.siteID, tt.request)

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error but got none")
				} else if tt.errMsg != "" && err.Error() != tt.errMsg {
					t.Errorf("Expected error message '%s', got '%s'", tt.errMsg, err.Error())
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if result == nil {
				t.Error("Result should not be nil")
			}
		})
	}
}

func TestHandler_TopicsReassign(t *testing.T) {
	handler, testDB, cleanup := setupHandlerTest(t)
	defer cleanup()

	tests := []struct {
		name               string
		request            dto.TopicsReassignRequest
		wantErr            bool
		expectedReassigned int
	}{
		{
			name: "Valid reassignment all topics",
			request: dto.TopicsReassignRequest{
				FromSiteID: 1,
				ToSiteID:   2,
				TopicIDs:   nil, // All topics
			},
			wantErr:            false,
			expectedReassigned: 2, // Site 1 has 2 topics from test data
		},
		{
			name: "Valid reassignment specific topics",
			request: dto.TopicsReassignRequest{
				FromSiteID: 1,
				ToSiteID:   3,
				TopicIDs:   []int64{1}, // Only topic 1
			},
			wantErr:            false,
			expectedReassigned: 1,
		},
		{
			name: "Invalid from site ID",
			request: dto.TopicsReassignRequest{
				FromSiteID: 0,
				ToSiteID:   2,
				TopicIDs:   nil,
			},
			wantErr: true,
		},
		{
			name: "Invalid to site ID",
			request: dto.TopicsReassignRequest{
				FromSiteID: 1,
				ToSiteID:   0,
				TopicIDs:   nil,
			},
			wantErr: true,
		},
		{
			name: "Same from and to site ID",
			request: dto.TopicsReassignRequest{
				FromSiteID: 1,
				ToSiteID:   1,
				TopicIDs:   nil,
			},
			wantErr: true,
		},
		{
			name: "Non-existent from site",
			request: dto.TopicsReassignRequest{
				FromSiteID: 999,
				ToSiteID:   1,
				TopicIDs:   nil,
			},
			wantErr: true,
		},
		{
			name: "Non-existent to site",
			request: dto.TopicsReassignRequest{
				FromSiteID: 1,
				ToSiteID:   999,
				TopicIDs:   nil,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testDB.ClearAllTables(t)
			testDB.InsertTestData(t)

			result, err := handler.TopicsReassign(tt.request)

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

			if result.FromSiteID != tt.request.FromSiteID {
				t.Errorf("Expected FromSiteID %d, got %d", tt.request.FromSiteID, result.FromSiteID)
			}

			if result.ToSiteID != tt.request.ToSiteID {
				t.Errorf("Expected ToSiteID %d, got %d", tt.request.ToSiteID, result.ToSiteID)
			}

			if result.ReassignedTopics != tt.expectedReassigned {
				t.Errorf("Expected %d reassigned topics, got %d", tt.expectedReassigned, result.ReassignedTopics)
			}

			if result.ErrorCount > 0 {
				t.Errorf("Expected no errors, got %d errors: %v", result.ErrorCount, result.ErrorMessages)
			}
		})
	}
}
