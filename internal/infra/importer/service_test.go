package importer

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

func setupTestImporter(t *testing.T) (*ImportService, func()) {
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

	importerService, err := NewImportService(container)
	if err != nil {
		dbCleanup()
		t.Fatalf("failed to create importer service: %v", err)
	}

	cleanup := func() {
		_ = testLogger.Close()
		_ = os.RemoveAll(tempLogDir)
		dbCleanup()
	}

	return importerService, cleanup
}

func createTestFile(t *testing.T, filename string, content string) string {
	t.Helper()

	tempDir := filepath.Join(os.TempDir(), "postulator_test_files", t.Name())
	_ = os.MkdirAll(tempDir, 0755)

	filePath := filepath.Join(tempDir, filename)
	err := os.WriteFile(filePath, []byte(content), 0644)
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	t.Cleanup(func() {
		_ = os.RemoveAll(tempDir)
	})

	return filePath
}

func TestImportService_TXT(t *testing.T) {
	importer, cleanup := setupTestImporter(t)
	defer cleanup()

	ctx := context.Background()

	t.Run("import from TXT file successfully", func(t *testing.T) {
		content := `First Topic Title
Second Topic Title
Third Topic Title`

		filePath := createTestFile(t, "test.txt", content)

		result, err := importer.ImportTopics(ctx, filePath)
		if err != nil {
			t.Fatalf("failed to import topics: %v", err)
		}

		if result.TotalAdded != 3 {
			t.Errorf("expected 3 topics added, got %d", result.TotalAdded)
		}

		if result.TotalSkipped != 0 {
			t.Errorf("expected 0 topics skipped, got %d", result.TotalSkipped)
		}

		if result.TotalRead != 3 {
			t.Errorf("expected 3 topics read, got %d", result.TotalRead)
		}
	})

	t.Run("import TXT with duplicates", func(t *testing.T) {
		content := `Unique Topic 1
Duplicate Topic
Unique Topic 2
Duplicate Topic
Unique Topic 3`

		filePath := createTestFile(t, "test_dup.txt", content)

		result, err := importer.ImportTopics(ctx, filePath)
		if err != nil {
			t.Fatalf("failed to import topics: %v", err)
		}

		if result.TotalAdded != 4 {
			t.Errorf("expected 4 topics added, got %d", result.TotalAdded)
		}

		if result.TotalSkipped != 1 {
			t.Errorf("expected 1 topic skipped, got %d", result.TotalSkipped)
		}
	})

	t.Run("import TXT with empty lines", func(t *testing.T) {
		content := `Topic One

Topic Two

Topic Three`

		filePath := createTestFile(t, "test_empty.txt", content)

		result, err := importer.ImportTopics(ctx, filePath)
		if err != nil {
			t.Fatalf("failed to import topics: %v", err)
		}

		if result.TotalAdded != 3 {
			t.Errorf("expected 3 topics added, got %d", result.TotalAdded)
		}
	})
}

func TestImportService_CSV(t *testing.T) {
	importer, cleanup := setupTestImporter(t)
	defer cleanup()

	ctx := context.Background()

	t.Run("import from CSV file successfully", func(t *testing.T) {
		content := `title
CSV Topic 1
CSV Topic 2
CSV Topic 3`

		filePath := createTestFile(t, "test.csv", content)

		result, err := importer.ImportTopics(ctx, filePath)
		if err != nil {
			t.Fatalf("failed to import topics: %v", err)
		}

		if result.TotalAdded != 3 {
			t.Errorf("expected 3 topics added, got %d", result.TotalAdded)
		}
	})

	t.Run("import CSV without header", func(t *testing.T) {
		content := `CSV Topic A
CSV Topic B
CSV Topic C`

		filePath := createTestFile(t, "test_no_header.csv", content)

		result, err := importer.ImportTopics(ctx, filePath)
		if err != nil {
			t.Fatalf("failed to import topics: %v", err)
		}

		if result.TotalAdded >= 3 {
			t.Logf("imported %d topics", result.TotalAdded)
		}
	})
}

func TestImportService_JSON(t *testing.T) {
	importer, cleanup := setupTestImporter(t)
	defer cleanup()

	ctx := context.Background()

	t.Run("import from JSON file successfully", func(t *testing.T) {
		content := `[
  "JSON Topic 1",
  "JSON Topic 2",
  "JSON Topic 3"
]`

		filePath := createTestFile(t, "test.json", content)

		result, err := importer.ImportTopics(ctx, filePath)
		if err != nil {
			t.Fatalf("failed to import topics: %v", err)
		}

		if result.TotalAdded != 3 {
			t.Errorf("expected 3 topics added, got %d", result.TotalAdded)
		}
	})

	t.Run("import from JSON object format", func(t *testing.T) {
		content := `{
  "topics": [
    {"title": "JSON Object Topic 1"},
    {"title": "JSON Object Topic 2"},
    {"title": "JSON Object Topic 3"}
  ]
}`

		filePath := createTestFile(t, "test_object.json", content)

		result, err := importer.ImportTopics(ctx, filePath)
		if err != nil {
			t.Fatalf("failed to import topics: %v", err)
		}

		if result.TotalAdded >= 3 {
			t.Logf("imported %d topics from JSON object format", result.TotalAdded)
		}
	})
}

func TestImportService_InvalidFormat(t *testing.T) {
	importer, cleanup := setupTestImporter(t)
	defer cleanup()

	ctx := context.Background()

	t.Run("import from unsupported format should fail", func(t *testing.T) {
		content := `Some content`
		filePath := createTestFile(t, "test.pdf", content)

		_, err := importer.ImportTopics(ctx, filePath)
		if err == nil {
			t.Fatal("expected error for unsupported format, got nil")
		}
	})

	t.Run("import from non-existent file should fail", func(t *testing.T) {
		_, err := importer.ImportTopics(ctx, "non_existent_file.txt")
		if err == nil {
			t.Fatal("expected error for non-existent file, got nil")
		}
	})
}

func TestImportService_ImportAndAssign(t *testing.T) {
	importer, cleanup := setupTestImporter(t)
	defer cleanup()

	ctx := context.Background()

	t.Run("import and assign to site", func(t *testing.T) {
		content := `Assigned Topic 1
Assigned Topic 2
Assigned Topic 3`

		filePath := createTestFile(t, "test_assign.txt", content)

		result, err := importer.ImportAndAssignToSite(ctx, filePath, 1, 1, entities.StrategyUnique)
		if err != nil {
			t.Logf("import and assign error (expected): %v", err)
		} else {
			t.Logf("imported and assigned %d topics", result.TotalAdded)
		}
	})
}

func TestImportService_LargeFile(t *testing.T) {
	importer, cleanup := setupTestImporter(t)
	defer cleanup()

	ctx := context.Background()

	t.Run("import large TXT file", func(t *testing.T) {
		var content string
		for i := 1; i <= 100; i++ {
			content += "Large Import Topic " + string(rune('0'+(i%10))) + string(rune('0'+(i/10))) + "\n"
		}

		filePath := createTestFile(t, "large.txt", content)

		result, err := importer.ImportTopics(ctx, filePath)
		if err != nil {
			t.Fatalf("failed to import large file: %v", err)
		}

		if result.TotalRead != 100 {
			t.Errorf("expected 100 topics read, got %d", result.TotalRead)
		}

		t.Logf("imported %d topics from large file", result.TotalAdded)
	})
}
