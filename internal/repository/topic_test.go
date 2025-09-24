package repository

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"Postulator/internal/models"
	"Postulator/internal/schema"

	_ "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
)

func setupTestDB(t *testing.T) (*Repository, func()) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}

	if err := schema.InitSchema(db); err != nil {
		t.Fatalf("Failed to initialize test schema: %v", err)
	}

	repo := NewRepository(db)

	cleanup := func() {
		db.Close()
	}

	return repo, cleanup
}

func TestRepository_CreateTopic(t *testing.T) {
	repo, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	tests := []struct {
		name    string
		topic   *models.Topic
		wantErr bool
	}{
		{
			name: "Valid topic",
			topic: &models.Topic{
				Title:     "Test Topic",
				Keywords:  "test,topic,keywords",
				Category:  "Technology",
				Tags:      "test,topic",
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			wantErr: false,
		},
		{
			name: "Topic with minimal data",
			topic: &models.Topic{
				Title:     "Minimal Topic",
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			wantErr: false,
		},
		{
			name: "Empty title should fail",
			topic: &models.Topic{
				Title:     "",
				Keywords:  "test",
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := repo.CreateTopic(ctx, tt.topic)

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

			if result.ID <= 0 {
				t.Error("Result ID should be positive")
			}

			if result.Title != tt.topic.Title {
				t.Errorf("Expected title %s, got %s", tt.topic.Title, result.Title)
			}
		})
	}
}

func TestRepository_GetTopic(t *testing.T) {
	repo, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	// Create a test topic first
	topic := &models.Topic{
		Title:     "Test Topic",
		Keywords:  "test,topic",
		Category:  "Technology",
		Tags:      "test",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	created, err := repo.CreateTopic(ctx, topic)
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
			name:    "Non-existent topic ID",
			topicID: 999,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := repo.GetTopic(ctx, tt.topicID)

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

			if result.ID != created.ID {
				t.Errorf("Expected ID %d, got %d", created.ID, result.ID)
			}

			if result.Title != created.Title {
				t.Errorf("Expected title %s, got %s", created.Title, result.Title)
			}
		})
	}
}

func TestRepository_UpdateTopic(t *testing.T) {
	repo, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	// Create a test topic first
	topic := &models.Topic{
		Title:     "Original Title",
		Keywords:  "original,keywords",
		Category:  "Technology",
		Tags:      "original",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	created, err := repo.CreateTopic(ctx, topic)
	if err != nil {
		t.Fatalf("Failed to create test topic: %v", err)
	}

	// Update the topic
	created.Title = "Updated Title"
	created.Keywords = "updated,keywords"
	created.UpdatedAt = time.Now()

	updated, err := repo.UpdateTopic(ctx, created)
	if err != nil {
		t.Errorf("Failed to update topic: %v", err)
		return
	}

	if updated.Title != "Updated Title" {
		t.Errorf("Expected updated title 'Updated Title', got %s", updated.Title)
	}

	if updated.Keywords != "updated,keywords" {
		t.Errorf("Expected updated keywords 'updated,keywords', got %s", updated.Keywords)
	}

	// Verify the update by retrieving the topic
	retrieved, err := repo.GetTopic(ctx, created.ID)
	if err != nil {
		t.Errorf("Failed to retrieve updated topic: %v", err)
		return
	}

	if retrieved.Title != "Updated Title" {
		t.Errorf("Expected retrieved title 'Updated Title', got %s", retrieved.Title)
	}
}

func TestRepository_DeleteTopic(t *testing.T) {
	repo, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	// Create a test topic first
	topic := &models.Topic{
		Title:     "Test Topic",
		Keywords:  "test,topic",
		Category:  "Technology",
		Tags:      "test",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	created, err := repo.CreateTopic(ctx, topic)
	if err != nil {
		t.Fatalf("Failed to create test topic: %v", err)
	}

	// Delete the topic
	err = repo.DeleteTopic(ctx, created.ID)
	if err != nil {
		t.Errorf("Failed to delete topic: %v", err)
		return
	}

	// Verify the topic is deleted
	_, err = repo.GetTopic(ctx, created.ID)
	if err == nil {
		t.Error("Expected error when retrieving deleted topic, but got none")
	}
}

func TestRepository_GetTopics(t *testing.T) {
	repo, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	// Create test topics
	topics := []*models.Topic{
		{
			Title:     "Topic 1",
			Keywords:  "test1,topic1",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			Title:     "Topic 2",
			Keywords:  "test2,topic2",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			Title:     "Topic 3",
			Keywords:  "test3,topic3",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	}

	for _, topic := range topics {
		_, err := repo.CreateTopic(ctx, topic)
		if err != nil {
			t.Fatalf("Failed to create test topic: %v", err)
		}
	}

	// Test pagination
	result, err := repo.GetTopics(ctx, 2, 0)
	if err != nil {
		t.Errorf("Failed to get topics: %v", err)
		return
	}

	if len(result.Data) != 2 {
		t.Errorf("Expected 2 topics, got %d", len(result.Data))
	}

	if result.Total != 3 {
		t.Errorf("Expected total count 3, got %d", result.Total)
	}

	if result.Limit != 2 {
		t.Errorf("Expected limit 2, got %d", result.Limit)
	}

	if result.Offset != 0 {
		t.Errorf("Expected offset 0, got %d", result.Offset)
	}
}

func TestRepository_GetAllTopicsForRandomSelection(t *testing.T) {
	repo, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	// Create test topics
	topics := []*models.Topic{
		{
			Title:     "Random Topic 1",
			Keywords:  "random1",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			Title:     "Random Topic 2",
			Keywords:  "random2",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	}

	for _, topic := range topics {
		_, err := repo.CreateTopic(ctx, topic)
		if err != nil {
			t.Fatalf("Failed to create test topic: %v", err)
		}
	}

	// Test getting all topics for random selection
	result, err := repo.GetAllTopicsForRandomSelection(ctx)
	if err != nil {
		t.Errorf("Failed to get topics for random selection: %v", err)
		return
	}

	if len(result) != 2 {
		t.Errorf("Expected 2 topics, got %d", len(result))
	}

	// Verify the topics are correct
	for i, topic := range result {
		expectedTitle := topics[i].Title
		if topic.Title != expectedTitle {
			t.Errorf("Expected topic title %s, got %s", expectedTitle, topic.Title)
		}
	}
}
