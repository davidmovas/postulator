package job

import (
	"Postulator/internal/config"
	"Postulator/internal/infra/database"
	"Postulator/pkg/di"
	"Postulator/pkg/logger"
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func setupTestExecutionRepository(t *testing.T) (*ExecRepository, func()) {
	t.Helper()

	db, dbCleanup := database.SetupTestDB(t)

	// Disable foreign key constraints for testing
	ctx := context.Background()
	_, err := db.ExecContext(ctx, "PRAGMA foreign_keys = OFF")
	require.NoError(t, err)

	tempLogDir := filepath.Join(os.TempDir(), "postulator_test_logs", t.Name())
	require.NoError(t, os.MkdirAll(tempLogDir, 0755))

	container := di.New()

	testLogger, err := logger.NewForTest(&config.Config{
		LogDir:      tempLogDir,
		AppLogFile:  "test.log",
		ErrLogFile:  "test_error.log",
		LogLevel:    "debug",
		ConsoleOut:  false,
		PrettyPrint: false,
	})
	require.NoError(t, err)

	container.MustRegister(di.Instance[*database.DB](db))
	container.MustRegister(di.Instance[*logger.Logger](testLogger))

	repo, err := NewExecutionRepository(container)
	require.NoError(t, err)

	cleanup := func() {
		_ = testLogger.Close()
		_ = os.RemoveAll(tempLogDir)
		dbCleanup()
	}

	return repo, cleanup
}

func TestExecutionRepository_CreateAndGet(t *testing.T) {
	repo, cleanup := setupTestExecutionRepository(t)
	defer cleanup()

	ctx := context.Background()

	t.Run("create execution successfully", func(t *testing.T) {
		exec := &Execution{
			JobID:     1,
			TopicID:   1,
			Status:    ExecutionPending,
			StartedAt: time.Now(),
		}

		err := repo.Create(ctx, exec)
		require.NoError(t, err)
	})

	t.Run("get execution by ID", func(t *testing.T) {
		title := "Test Article Title"
		content := "Test Article Content"
		exec := &Execution{
			JobID:            1,
			TopicID:          2,
			GeneratedTitle:   &title,
			GeneratedContent: &content,
			Status:           ExecutionGenerating,
			StartedAt:        time.Now(),
		}

		err := repo.Create(ctx, exec)
		require.NoError(t, err)

		// Get all executions to find the ID
		// GetByJobID orders by started_at DESC, so the first element is the most recent
		allExecs, err := repo.GetByJobID(ctx, 1)
		require.NoError(t, err)
		require.Greater(t, len(allExecs), 0)

		retrievedExec, err := repo.GetByID(ctx, allExecs[0].ID)
		require.NoError(t, err)
		require.Equal(t, int64(2), retrievedExec.TopicID)
		require.NotNil(t, retrievedExec.GeneratedTitle)
		require.Equal(t, "Test Article Title", *retrievedExec.GeneratedTitle)
		require.Equal(t, ExecutionGenerating, retrievedExec.Status)
	})

	t.Run("get non-existent execution should fail", func(t *testing.T) {
		_, err := repo.GetByID(ctx, 999999)
		require.Error(t, err)
	})
}

func TestExecutionRepository_GetByJobID(t *testing.T) {
	repo, cleanup := setupTestExecutionRepository(t)
	defer cleanup()

	ctx := context.Background()

	t.Run("get executions by job ID", func(t *testing.T) {
		// Create executions for job 1
		for i := 1; i <= 3; i++ {
			exec := &Execution{
				JobID:     1,
				TopicID:   int64(i),
				Status:    ExecutionPending,
				StartedAt: time.Now(),
			}
			err := repo.Create(ctx, exec)
			require.NoError(t, err)
		}

		// Create executions for job 2
		for i := 1; i <= 2; i++ {
			exec := &Execution{
				JobID:     2,
				TopicID:   int64(i + 10),
				Status:    ExecutionPending,
				StartedAt: time.Now(),
			}
			err := repo.Create(ctx, exec)
			require.NoError(t, err)
		}

		// Get executions for job 1
		job1Execs, err := repo.GetByJobID(ctx, 1)
		require.NoError(t, err)
		require.Equal(t, 3, len(job1Execs))

		// Verify all belong to job 1
		for _, exec := range job1Execs {
			require.Equal(t, int64(1), exec.JobID)
		}

		// Get executions for job 2
		job2Execs, err := repo.GetByJobID(ctx, 2)
		require.NoError(t, err)
		require.Equal(t, 2, len(job2Execs))

		// Verify all belong to job 2
		for _, exec := range job2Execs {
			require.Equal(t, int64(2), exec.JobID)
		}
	})

	t.Run("get executions for job with no executions", func(t *testing.T) {
		execs, err := repo.GetByJobID(ctx, 999)
		require.NoError(t, err)
		require.Equal(t, 0, len(execs))
	})
}

func TestExecutionRepository_GetPendingValidation(t *testing.T) {
	repo, cleanup := setupTestExecutionRepository(t)
	defer cleanup()

	ctx := context.Background()

	t.Run("get pending validation executions", func(t *testing.T) {
		// Create executions with different statuses
		statuses := []ExecutionStatus{
			ExecutionPending,
			ExecutionGenerating,
			ExecutionPendingValidation,
			ExecutionPendingValidation,
			ExecutionValidated,
			ExecutionPublished,
			ExecutionFailed,
		}

		for i, status := range statuses {
			exec := &Execution{
				JobID:     int64(i + 1),
				TopicID:   int64(i + 1),
				Status:    status,
				StartedAt: time.Now(),
			}
			err := repo.Create(ctx, exec)
			require.NoError(t, err)
		}

		// Get pending validation executions
		pendingExecs, err := repo.GetPendingValidation(ctx)
		require.NoError(t, err)
		require.Equal(t, 2, len(pendingExecs))

		// Verify all have pending_validation status
		for _, exec := range pendingExecs {
			require.Equal(t, ExecutionPendingValidation, exec.Status)
		}
	})

	t.Run("get pending validation with no pending", func(t *testing.T) {
		// After previous test, create only non-pending executions
		exec := &Execution{
			JobID:     100,
			TopicID:   100,
			Status:    ExecutionPublished,
			StartedAt: time.Now(),
		}
		err := repo.Create(ctx, exec)
		require.NoError(t, err)

		pendingExecs, err := repo.GetPendingValidation(ctx)
		require.NoError(t, err)
		// Should still have 2 from previous test
		require.GreaterOrEqual(t, len(pendingExecs), 2)
	})
}

func TestExecutionRepository_Update(t *testing.T) {
	repo, cleanup := setupTestExecutionRepository(t)
	defer cleanup()

	ctx := context.Background()

	t.Run("update execution successfully", func(t *testing.T) {
		exec := &Execution{
			JobID:     1,
			TopicID:   1,
			Status:    ExecutionPending,
			StartedAt: time.Now(),
		}

		err := repo.Create(ctx, exec)
		require.NoError(t, err)

		execs, err := repo.GetByJobID(ctx, 1)
		require.NoError(t, err)
		require.Greater(t, len(execs), 0)

		// Update execution
		updatedExec := execs[len(execs)-1]
		title := "Generated Title"
		content := "Generated Content"
		now := time.Now()
		updatedExec.GeneratedTitle = &title
		updatedExec.GeneratedContent = &content
		updatedExec.Status = ExecutionGenerating
		updatedExec.GeneratedAt = &now

		err = repo.Update(ctx, updatedExec)
		require.NoError(t, err)

		retrievedExec, err := repo.GetByID(ctx, updatedExec.ID)
		require.NoError(t, err)
		require.Equal(t, ExecutionGenerating, retrievedExec.Status)
		require.NotNil(t, retrievedExec.GeneratedTitle)
		require.Equal(t, "Generated Title", *retrievedExec.GeneratedTitle)
		require.NotNil(t, retrievedExec.GeneratedAt)
	})

	t.Run("update execution status progression", func(t *testing.T) {
		exec := &Execution{
			JobID:     2,
			TopicID:   2,
			Status:    ExecutionPending,
			StartedAt: time.Now(),
		}

		err := repo.Create(ctx, exec)
		require.NoError(t, err)

		execs, err := repo.GetByJobID(ctx, 2)
		require.NoError(t, err)
		require.Greater(t, len(execs), 0)

		currentExec := execs[len(execs)-1]

		// Progress through statuses
		statusProgression := []ExecutionStatus{
			ExecutionGenerating,
			ExecutionPendingValidation,
			ExecutionValidated,
			ExecutionPublishing,
			ExecutionPublished,
		}

		for _, status := range statusProgression {
			currentExec.Status = status
			err = repo.Update(ctx, currentExec)
			require.NoError(t, err)

			retrievedExec, err := repo.GetByID(ctx, currentExec.ID)
			require.NoError(t, err)
			require.Equal(t, status, retrievedExec.Status)
		}
	})
}

func TestExecutionRepository_Delete(t *testing.T) {
	repo, cleanup := setupTestExecutionRepository(t)
	defer cleanup()

	ctx := context.Background()

	t.Run("delete execution successfully", func(t *testing.T) {
		exec := &Execution{
			JobID:     1,
			TopicID:   1,
			Status:    ExecutionPending,
			StartedAt: time.Now(),
		}

		err := repo.Create(ctx, exec)
		require.NoError(t, err)

		execs, err := repo.GetByJobID(ctx, 1)
		require.NoError(t, err)
		require.Greater(t, len(execs), 0)

		execID := execs[len(execs)-1].ID

		err = repo.Delete(ctx, execID)
		require.NoError(t, err)

		_, err = repo.GetByID(ctx, execID)
		require.Error(t, err)
	})

	t.Run("delete non-existent execution should fail", func(t *testing.T) {
		err := repo.Delete(ctx, 999999)
		require.Error(t, err)
	})
}

func TestExecutionRepository_ExecutionStatuses(t *testing.T) {
	repo, cleanup := setupTestExecutionRepository(t)
	defer cleanup()

	ctx := context.Background()

	t.Run("create executions with all statuses", func(t *testing.T) {
		statuses := []ExecutionStatus{
			ExecutionPending,
			ExecutionGenerating,
			ExecutionPendingValidation,
			ExecutionValidated,
			ExecutionPublishing,
			ExecutionPublished,
			ExecutionFailed,
		}

		for i, status := range statuses {
			exec := &Execution{
				JobID:     int64(i + 1),
				TopicID:   int64(i + 1),
				Status:    status,
				StartedAt: time.Now(),
			}

			err := repo.Create(ctx, exec)
			require.NoError(t, err)
		}

		// Verify all statuses are preserved
		for i, expectedStatus := range statuses {
			execs, err := repo.GetByJobID(ctx, int64(i+1))
			require.NoError(t, err)
			require.Greater(t, len(execs), 0)
			require.Equal(t, expectedStatus, execs[len(execs)-1].Status)
		}
	})
}

func TestExecutionRepository_WithErrorMessage(t *testing.T) {
	repo, cleanup := setupTestExecutionRepository(t)
	defer cleanup()

	ctx := context.Background()

	t.Run("create execution with error message", func(t *testing.T) {
		errorMsg := "Failed to generate article: timeout"
		exec := &Execution{
			JobID:        1,
			TopicID:      1,
			Status:       ExecutionFailed,
			ErrorMessage: &errorMsg,
			StartedAt:    time.Now(),
		}

		err := repo.Create(ctx, exec)
		require.NoError(t, err)

		execs, err := repo.GetByJobID(ctx, 1)
		require.NoError(t, err)
		require.Greater(t, len(execs), 0)

		retrievedExec := execs[len(execs)-1]
		require.NotNil(t, retrievedExec.ErrorMessage)
		require.Equal(t, "Failed to generate article: timeout", *retrievedExec.ErrorMessage)
	})
}

func TestExecutionRepository_WithArticleID(t *testing.T) {
	repo, cleanup := setupTestExecutionRepository(t)
	defer cleanup()

	ctx := context.Background()

	t.Run("create execution with article ID", func(t *testing.T) {
		articleID := int64(42)
		title := "Published Article"
		content := "Article content"
		now := time.Now()

		exec := &Execution{
			JobID:            1,
			TopicID:          1,
			GeneratedTitle:   &title,
			GeneratedContent: &content,
			Status:           ExecutionPublished,
			ArticleID:        &articleID,
			StartedAt:        now,
			PublishedAt:      &now,
		}

		err := repo.Create(ctx, exec)
		require.NoError(t, err)

		execs, err := repo.GetByJobID(ctx, 1)
		require.NoError(t, err)
		require.Greater(t, len(execs), 0)

		retrievedExec := execs[len(execs)-1]
		require.NotNil(t, retrievedExec.ArticleID)
		require.Equal(t, int64(42), *retrievedExec.ArticleID)
		require.NotNil(t, retrievedExec.PublishedAt)
	})
}

func TestExecutionRepository_Timestamps(t *testing.T) {
	repo, cleanup := setupTestExecutionRepository(t)
	defer cleanup()

	ctx := context.Background()

	t.Run("execution timestamps progression", func(t *testing.T) {
		now := time.Now()
		title := "Test Title"
		content := "Test Content"

		exec := &Execution{
			JobID:     1,
			TopicID:   1,
			Status:    ExecutionPending,
			StartedAt: now,
		}

		err := repo.Create(ctx, exec)
		require.NoError(t, err)

		execs, err := repo.GetByJobID(ctx, 1)
		require.NoError(t, err)
		require.Greater(t, len(execs), 0)

		currentExec := execs[len(execs)-1]

		// Update with generated timestamp
		generatedAt := now.Add(1 * time.Minute)
		currentExec.GeneratedTitle = &title
		currentExec.GeneratedContent = &content
		currentExec.Status = ExecutionGenerating
		currentExec.GeneratedAt = &generatedAt

		err = repo.Update(ctx, currentExec)
		require.NoError(t, err)

		// Update with validated timestamp
		validatedAt := now.Add(2 * time.Minute)
		currentExec.Status = ExecutionValidated
		currentExec.ValidatedAt = &validatedAt

		err = repo.Update(ctx, currentExec)
		require.NoError(t, err)

		// Update with published timestamp
		publishedAt := now.Add(3 * time.Minute)
		articleID := int64(1)
		currentExec.Status = ExecutionPublished
		currentExec.ArticleID = &articleID
		currentExec.PublishedAt = &publishedAt

		err = repo.Update(ctx, currentExec)
		require.NoError(t, err)

		// Verify all timestamps are present
		finalExec, err := repo.GetByID(ctx, currentExec.ID)
		require.NoError(t, err)
		require.NotNil(t, finalExec.GeneratedAt)
		require.NotNil(t, finalExec.ValidatedAt)
		require.NotNil(t, finalExec.PublishedAt)
		require.NotNil(t, finalExec.ArticleID)
	})
}
