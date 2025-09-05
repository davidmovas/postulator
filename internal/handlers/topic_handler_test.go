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
			err := handler.DeleteTopic(tt.topicID)

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

func TestHandler_SiteTopic_CRUD(t *testing.T) {
	handler, testDB, cleanup := setupHandlerTest(t)
	defer cleanup()

	testDB.ClearAllTables(t)
	testDB.InsertTestData(t)

	// Create a test topic first
	topic, err := handler.CreateTopic(dto.CreateTopicRequest{
		Title:    "Site Topic Test",
		Keywords: "site,topic",
		Category: "Test",
		Tags:     "test",
		IsActive: true,
	})
	if err != nil {
		t.Fatalf("Failed to create test topic: %v", err)
	}

	var createdSiteTopic *dto.SiteTopicResponse

	// Test create site topic
	t.Run("Create site topic", func(t *testing.T) {
		req := dto.CreateSiteTopicRequest{
			SiteID:   1, // From test data
			TopicID:  topic.ID,
			IsActive: true,
			Priority: 5,
		}

		response, err := handler.CreateSiteTopic(req)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
			return
		}

		if response == nil {
			t.Error("Response should not be nil")
			return
		}

		createdSiteTopic = response

		if response.SiteID != req.SiteID {
			t.Errorf("Expected site ID %d, got %d", req.SiteID, response.SiteID)
		}

		if response.TopicID != req.TopicID {
			t.Errorf("Expected topic ID %d, got %d", req.TopicID, response.TopicID)
		}

		if response.Priority != req.Priority {
			t.Errorf("Expected priority %d, got %d", req.Priority, response.Priority)
		}
	})

	// Test get site topics
	t.Run("Get site topics", func(t *testing.T) {
		response, err := handler.GetSiteTopics(1, dto.PaginationRequest{Page: 1, Limit: 10})
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
			return
		}

		if response == nil {
			t.Error("Response should not be nil")
			return
		}

		// Should have at least the one we created plus any from test data
		if len(response.SiteTopics) == 0 {
			t.Error("Expected at least 1 site topic")
		}
	})

	// Test update site topic
	t.Run("Update site topic", func(t *testing.T) {
		req := dto.UpdateSiteTopicRequest{
			ID:       createdSiteTopic.ID,
			SiteID:   1,
			TopicID:  topic.ID,
			IsActive: false,
			Priority: 8,
		}

		response, err := handler.UpdateSiteTopic(req)
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

		if response.Priority != 8 {
			t.Errorf("Expected priority 8, got %d", response.Priority)
		}
	})

	// Test activate/deactivate site topic
	t.Run("Activate site topic", func(t *testing.T) {
		err := handler.ActivateSiteTopic(createdSiteTopic.ID)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
	})

	t.Run("Deactivate site topic", func(t *testing.T) {
		err := handler.DeactivateSiteTopic(createdSiteTopic.ID)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
	})

	// Test delete site topic
	t.Run("Delete site topic", func(t *testing.T) {
		err := handler.DeleteSiteTopic(createdSiteTopic.ID)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
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
