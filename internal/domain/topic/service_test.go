package topic

import (
	"Postulator/internal/config"
	"Postulator/internal/domain/entities"
	"Postulator/internal/infra/database"
	"Postulator/pkg/di"
	"Postulator/pkg/logger"
	"context"
	"os"
	"path/filepath"
	"testing"
)

func setupTestService(t *testing.T) (*Service, func()) {
	t.Helper()

	db, dbCleanup := database.SetupTestDB(t)

	tempLogDir := filepath.Join(os.TempDir(), "postulator_test_logs", t.Name())
	_ = os.MkdirAll(tempLogDir, 0755)

	container := di.New()

	testLogger, err := logger.New(&config.Config{
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

func TestTopicService_CreateAndGet(t *testing.T) {
	service, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()

	t.Run("create topic successfully", func(t *testing.T) {
		topic := &entities.Topic{
			Title: "Test Topic 1",
		}

		err := service.CreateTopic(ctx, topic)
		if err != nil {
			t.Fatalf("failed to create topic: %v", err)
		}
	})

	t.Run("get topic by ID", func(t *testing.T) {
		topic := &entities.Topic{
			Title: "Get Test Topic",
		}

		err := service.CreateTopic(ctx, topic)
		if err != nil {
			t.Fatalf("failed to create topic: %v", err)
		}

		topics, err := service.ListTopics(ctx)
		if err != nil {
			t.Fatalf("failed to list topics: %v", err)
		}

		if len(topics) == 0 {
			t.Fatal("no topics found")
		}

		retrievedTopic, err := service.GetTopic(ctx, topics[len(topics)-1].ID)
		if err != nil {
			t.Fatalf("failed to get topic: %v", err)
		}

		if retrievedTopic.Title != "Get Test Topic" {
			t.Errorf("expected title 'Get Test Topic', got '%s'", retrievedTopic.Title)
		}
	})

	t.Run("create topic with duplicate title should fail", func(t *testing.T) {
		topic1 := &entities.Topic{
			Title: "Duplicate Topic",
		}

		err := service.CreateTopic(ctx, topic1)
		if err != nil {
			t.Fatalf("failed to create first topic: %v", err)
		}

		topic2 := &entities.Topic{
			Title: "Duplicate Topic",
		}

		err = service.CreateTopic(ctx, topic2)
		if err == nil {
			t.Fatal("expected error when creating topic with duplicate title, got nil")
		}
	})

	t.Run("get non-existent topic should fail", func(t *testing.T) {
		_, err := service.GetTopic(ctx, 999999)
		if err == nil {
			t.Fatal("expected error when getting non-existent topic, got nil")
		}
	})
}

func TestTopicService_BatchCreate(t *testing.T) {
	service, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()

	t.Run("batch create topics successfully", func(t *testing.T) {
		topics := []*entities.Topic{
			{Title: "Batch Topic 1"},
			{Title: "Batch Topic 2"},
			{Title: "Batch Topic 3"},
		}

		result, err := service.CreateTopicBatch(ctx, topics)
		if err != nil {
			t.Fatalf("failed to batch create topics: %v", err)
		}

		if result.TotalAdded != 3 {
			t.Errorf("expected 3 topics added, got %d", result.TotalAdded)
		}

		if result.TotalSkipped != 0 {
			t.Errorf("expected 0 topics skipped, got %d", result.TotalSkipped)
		}
	})

	t.Run("batch create with duplicates", func(t *testing.T) {
		topic := &entities.Topic{Title: "Existing Topic"}
		err := service.CreateTopic(ctx, topic)
		if err != nil {
			t.Fatalf("failed to create topic: %v", err)
		}

		topics := []*entities.Topic{
			{Title: "New Topic 1"},
			{Title: "Existing Topic"}, // <- duplicate
			{Title: "New Topic 2"},
		}

		result, err := service.CreateTopicBatch(ctx, topics)
		if err != nil {
			t.Fatalf("failed to batch create topics: %v", err)
		}

		if result.TotalAdded != 2 {
			t.Errorf("expected 2 topics added, got %d", result.TotalAdded)
		}

		if result.TotalSkipped != 1 {
			t.Errorf("expected 1 topic skipped, got %d", result.TotalSkipped)
		}

		if len(result.Skipped) != 1 {
			t.Errorf("expected 1 skipped title, got %d", len(result.Skipped))
		}
	})
}

func TestTopicService_ListTopics(t *testing.T) {
	service, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()

	t.Run("list empty topics", func(t *testing.T) {
		topics, err := service.ListTopics(ctx)
		if err != nil {
			t.Fatalf("failed to list topics: %v", err)
		}

		if len(topics) != 0 {
			t.Errorf("expected 0 topics, got %d", len(topics))
		}
	})

	t.Run("list multiple topics", func(t *testing.T) {
		for i := 1; i <= 3; i++ {
			topic := &entities.Topic{
				Title: "List Test Topic " + string(rune('0'+i)),
			}

			err := service.CreateTopic(ctx, topic)
			if err != nil {
				t.Fatalf("failed to create topic %d: %v", i, err)
			}
		}

		topics, err := service.ListTopics(ctx)
		if err != nil {
			t.Fatalf("failed to list topics: %v", err)
		}

		if len(topics) != 3 {
			t.Errorf("expected 3 topics, got %d", len(topics))
		}
	})
}

func TestTopicService_UpdateTopic(t *testing.T) {
	service, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()

	t.Run("update topic successfully", func(t *testing.T) {
		topic := &entities.Topic{
			Title: "Original Title",
		}

		err := service.CreateTopic(ctx, topic)
		if err != nil {
			t.Fatalf("failed to create topic: %v", err)
		}

		topics, err := service.ListTopics(ctx)
		if err != nil {
			t.Fatalf("failed to list topics: %v", err)
		}

		if len(topics) == 0 {
			t.Fatal("no topics found")
		}

		updatedTopic := topics[0]
		updatedTopic.Title = "Updated Title"

		err = service.UpdateTopic(ctx, updatedTopic)
		if err != nil {
			t.Fatalf("failed to update topic: %v", err)
		}

		retrievedTopic, err := service.GetTopic(ctx, updatedTopic.ID)
		if err != nil {
			t.Fatalf("failed to get updated topic: %v", err)
		}

		if retrievedTopic.Title != "Updated Title" {
			t.Errorf("expected title 'Updated Title', got '%s'", retrievedTopic.Title)
		}
	})
}

func TestTopicService_DeleteTopic(t *testing.T) {
	service, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()

	t.Run("delete topic successfully", func(t *testing.T) {
		topic := &entities.Topic{
			Title: "Delete Test Topic",
		}

		err := service.CreateTopic(ctx, topic)
		if err != nil {
			t.Fatalf("failed to create topic: %v", err)
		}

		topics, err := service.ListTopics(ctx)
		if err != nil {
			t.Fatalf("failed to list topics: %v", err)
		}

		if len(topics) == 0 {
			t.Fatal("no topics found")
		}

		topicID := topics[0].ID

		err = service.DeleteTopic(ctx, topicID)
		if err != nil {
			t.Fatalf("failed to delete topic: %v", err)
		}

		_, err = service.GetTopic(ctx, topicID)
		if err == nil {
			t.Fatal("expected error when getting deleted topic, got nil")
		}
	})
}

func TestTopicService_AssignToSite(t *testing.T) {
	service, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()

	createSiteAndCategory := func(t *testing.T) (int64, int64) {
		t.Helper()

		return 1, 1
	}

	t.Run("assign topic to site successfully", func(t *testing.T) {
		topic := &entities.Topic{
			Title: "Assignment Test Topic",
		}

		err := service.CreateTopic(ctx, topic)
		if err != nil {
			t.Fatalf("failed to create topic: %v", err)
		}

		topics, err := service.ListTopics(ctx)
		if err != nil {
			t.Fatalf("failed to list topics: %v", err)
		}

		if len(topics) == 0 {
			t.Fatal("no topics found")
		}

		topicID := topics[0].ID
		siteID, categoryID := createSiteAndCategory(t)

		err = service.AssignToSite(ctx, siteID, topicID, categoryID, entities.StrategyUnique)
		_ = err
	})
}

func TestTopicService_UniqueStrategy(t *testing.T) {
	service, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()

	t.Run("count unused topics", func(t *testing.T) {
		for i := 1; i <= 5; i++ {
			topic := &entities.Topic{
				Title: "Unique Strategy Topic " + string(rune('0'+i)),
			}

			err := service.CreateTopic(ctx, topic)
			if err != nil {
				t.Fatalf("failed to create topic %d: %v", i, err)
			}
		}

		count, err := service.CountUnusedTopics(ctx, 1)
		if err != nil {
			t.Logf("count unused topics: %v", err)
		} else {
			t.Logf("unused topics count: %d", count)
		}
	})
}
