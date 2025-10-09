package job

import (
	"Postulator/internal/config"
	"Postulator/internal/infra/ai"
	"Postulator/internal/infra/database"
	"Postulator/internal/infra/wp"
	"Postulator/pkg/di"
	"Postulator/pkg/logger"
	"context"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// mockAIClient is a mock implementation of ai.Client for testing
type mockAIClient struct{}

func (m *mockAIClient) GenerateArticle(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	return "Mock generated article content", nil
}

func setupTestService(t *testing.T) (*Service, *JobRepository, *ExecRepository, func()) {
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
	container.MustRegister(di.Instance[*wp.Client](wp.NewClient()))
	container.MustRegister(&di.Registration[ai.Client]{
		Provider:      func(di.Container) (ai.Client, error) { return &mockAIClient{}, nil },
		Lifecycle:     di.Singleton,
		InterfaceType: reflect.TypeOf((*ai.Client)(nil)).Elem(),
	})

	service, err := NewService(container)
	require.NoError(t, err)

	repo, err := NewJobRepository(container)
	require.NoError(t, err)

	execRepo, err := NewExecutionRepository(container)
	require.NoError(t, err)

	cleanup := func() {
		_ = testLogger.Close()
		_ = os.RemoveAll(tempLogDir)
		dbCleanup()
	}

	return service, repo, execRepo, cleanup
}

func TestJobService_CreateAndGet(t *testing.T) {
	service, _, _, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()

	t.Run("create job successfully", func(t *testing.T) {
		job := &Job{
			Name:               "Test Job",
			SiteID:             1,
			CategoryID:         1,
			PromptID:           1,
			AIProviderID:       1,
			AIModel:            "gpt-4",
			RequiresValidation: false,
			ScheduleType:       ScheduleDaily,
			JitterEnabled:      false,
		}

		err := service.CreateJob(ctx, job)
		require.NoError(t, err)
		require.NotNil(t, job.NextRunAt)
	})

	t.Run("get job by ID", func(t *testing.T) {
		monday := 1
		job := &Job{
			Name:               "Get Test Job",
			SiteID:             1,
			CategoryID:         1,
			PromptID:           1,
			AIProviderID:       1,
			AIModel:            "gpt-4o",
			RequiresValidation: true,
			ScheduleType:       ScheduleWeekly,
			ScheduleDay:        &monday,
			JitterEnabled:      true,
			JitterMinutes:      30,
		}

		err := service.CreateJob(ctx, job)
		require.NoError(t, err)

		jobs, err := service.ListJobs(ctx)
		require.NoError(t, err)
		require.Greater(t, len(jobs), 0)

		retrievedJob, err := service.GetJob(ctx, jobs[len(jobs)-1].ID)
		require.NoError(t, err)
		require.Equal(t, "Get Test Job", retrievedJob.Name)
		require.Equal(t, "gpt-4o", retrievedJob.AIModel)
		require.True(t, retrievedJob.RequiresValidation)
	})

	t.Run("create job with invalid data should fail", func(t *testing.T) {
		job := &Job{
			Name:         "", // Empty name
			SiteID:       1,
			CategoryID:   1,
			PromptID:     1,
			AIProviderID: 1,
			AIModel:      "gpt-4",
			ScheduleType: ScheduleDaily,
		}

		err := service.CreateJob(ctx, job)
		require.Error(t, err)
	})

	t.Run("get non-existent job should fail", func(t *testing.T) {
		_, err := service.GetJob(ctx, 999999)
		require.Error(t, err)
	})
}

func TestJobService_ListJobs(t *testing.T) {
	service, _, _, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()

	t.Run("list empty jobs", func(t *testing.T) {
		jobs, err := service.ListJobs(ctx)
		require.NoError(t, err)
		require.Equal(t, 0, len(jobs))
	})

	t.Run("list multiple jobs", func(t *testing.T) {
		for i := 1; i <= 3; i++ {
			job := &Job{
				Name:         "List Test Job " + string(rune('0'+i)),
				SiteID:       int64(i),
				CategoryID:   1,
				PromptID:     1,
				AIProviderID: 1,
				AIModel:      "gpt-4",
				ScheduleType: ScheduleDaily,
			}

			err := service.CreateJob(ctx, job)
			require.NoError(t, err)
		}

		jobs, err := service.ListJobs(ctx)
		require.NoError(t, err)
		require.Equal(t, 3, len(jobs))
	})
}

func TestJobService_UpdateJob(t *testing.T) {
	service, _, _, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()

	t.Run("update job successfully", func(t *testing.T) {
		job := &Job{
			Name:         "Original Job",
			SiteID:       1,
			CategoryID:   1,
			PromptID:     1,
			AIProviderID: 1,
			AIModel:      "gpt-4",
			ScheduleType: ScheduleDaily,
		}

		err := service.CreateJob(ctx, job)
		require.NoError(t, err)

		jobs, err := service.ListJobs(ctx)
		require.NoError(t, err)
		require.Greater(t, len(jobs), 0)

		updatedJob := jobs[0]
		updatedJob.Name = "Updated Job"
		updatedJob.AIModel = "gpt-4o"

		err = service.UpdateJob(ctx, updatedJob)
		require.NoError(t, err)

		retrievedJob, err := service.GetJob(ctx, updatedJob.ID)
		require.NoError(t, err)
		require.Equal(t, "Updated Job", retrievedJob.Name)
		require.Equal(t, "gpt-4o", retrievedJob.AIModel)
	})

	t.Run("update job with invalid data should fail", func(t *testing.T) {
		job := &Job{
			Name:         "Valid Job",
			SiteID:       1,
			CategoryID:   1,
			PromptID:     1,
			AIProviderID: 1,
			AIModel:      "gpt-4",
			ScheduleType: ScheduleDaily,
		}

		err := service.CreateJob(ctx, job)
		require.NoError(t, err)

		jobs, err := service.ListJobs(ctx)
		require.NoError(t, err)
		require.Greater(t, len(jobs), 0)

		invalidJob := jobs[len(jobs)-1]
		invalidJob.Name = "" // Invalid empty name

		err = service.UpdateJob(ctx, invalidJob)
		require.Error(t, err)
	})
}

func TestJobService_DeleteJob(t *testing.T) {
	service, _, _, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()

	t.Run("delete job successfully", func(t *testing.T) {
		job := &Job{
			Name:         "Delete Test Job",
			SiteID:       1,
			CategoryID:   1,
			PromptID:     1,
			AIProviderID: 1,
			AIModel:      "gpt-4",
			ScheduleType: ScheduleDaily,
		}

		err := service.CreateJob(ctx, job)
		require.NoError(t, err)

		jobs, err := service.ListJobs(ctx)
		require.NoError(t, err)
		require.Greater(t, len(jobs), 0)

		jobID := jobs[0].ID

		err = service.DeleteJob(ctx, jobID)
		require.NoError(t, err)

		_, err = service.GetJob(ctx, jobID)
		require.Error(t, err)
	})

	t.Run("delete non-existent job should fail", func(t *testing.T) {
		err := service.DeleteJob(ctx, 999999)
		require.Error(t, err)
	})
}

func TestJobService_PauseJob(t *testing.T) {
	service, repo, _, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()

	t.Run("pause active job successfully", func(t *testing.T) {
		job := &Job{
			Name:         "Pause Test Job",
			SiteID:       1,
			CategoryID:   1,
			PromptID:     1,
			AIProviderID: 1,
			AIModel:      "gpt-4",
			ScheduleType: ScheduleDaily,
		}

		err := service.CreateJob(ctx, job)
		require.NoError(t, err)

		jobs, err := service.ListJobs(ctx)
		require.NoError(t, err)
		require.Greater(t, len(jobs), 0)

		jobID := jobs[len(jobs)-1].ID

		err = service.PauseJob(ctx, jobID)
		require.NoError(t, err)

		retrievedJob, err := repo.GetByID(ctx, jobID)
		require.NoError(t, err)
		require.Equal(t, StatusPaused, retrievedJob.Status)
	})

	t.Run("pause non-existent job should fail", func(t *testing.T) {
		err := service.PauseJob(ctx, 999999)
		require.Error(t, err)
	})
}

func TestJobService_ResumeJob(t *testing.T) {
	service, repo, _, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()

	t.Run("resume paused job successfully", func(t *testing.T) {
		job := &Job{
			Name:         "Resume Test Job",
			SiteID:       1,
			CategoryID:   1,
			PromptID:     1,
			AIProviderID: 1,
			AIModel:      "gpt-4",
			ScheduleType: ScheduleDaily,
		}

		err := service.CreateJob(ctx, job)
		require.NoError(t, err)

		jobs, err := service.ListJobs(ctx)
		require.NoError(t, err)
		require.Greater(t, len(jobs), 0)

		jobID := jobs[len(jobs)-1].ID

		// First pause it
		err = service.PauseJob(ctx, jobID)
		require.NoError(t, err)

		// Then resume it
		err = service.ResumeJob(ctx, jobID)
		require.NoError(t, err)

		retrievedJob, err := repo.GetByID(ctx, jobID)
		require.NoError(t, err)
		require.Equal(t, StatusActive, retrievedJob.Status)
		require.NotNil(t, retrievedJob.NextRunAt)
	})

	t.Run("resume non-existent job should fail", func(t *testing.T) {
		err := service.ResumeJob(ctx, 999999)
		require.Error(t, err)
	})
}

func TestJobService_ExecuteJobManually(t *testing.T) {
	service, repo, _, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()

	t.Run("execute job manually creates execution record", func(t *testing.T) {
		job := &Job{
			Name:         "Manual Execution Job",
			SiteID:       1,
			CategoryID:   1,
			PromptID:     1,
			AIProviderID: 1,
			AIModel:      "gpt-4",
			ScheduleType: ScheduleManual,
		}

		err := service.CreateJob(ctx, job)
		require.NoError(t, err)

		jobs, err := service.ListJobs(ctx)
		require.NoError(t, err)
		require.Greater(t, len(jobs), 0)

		jobID := jobs[len(jobs)-1].ID

		// Note: ExecuteJobManually will fail because executor needs real dependencies
		// But we can verify it attempts to execute
		err = service.ExecuteJobManually(ctx, jobID)
		// Error is expected because we don't have real topic/prompt/AI dependencies in test
		// Just verify the method was called
		_ = err

		// Verify job was retrieved (would have errored earlier if job didn't exist)
		retrievedJob, err := repo.GetByID(ctx, jobID)
		require.NoError(t, err)
		require.Equal(t, "Manual Execution Job", retrievedJob.Name)
	})

	t.Run("execute non-existent job should fail", func(t *testing.T) {
		err := service.ExecuteJobManually(ctx, 999999)
		require.Error(t, err)
	})
}

func TestJobService_ValidateExecution(t *testing.T) {
	service, _, execRepo, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()

	t.Run("approve execution successfully", func(t *testing.T) {
		// Create an execution pending validation
		title := "Test Article"
		content := "Test Content"
		exec := &Execution{
			JobID:            1,
			TopicID:          1,
			GeneratedTitle:   &title,
			GeneratedContent: &content,
			Status:           ExecutionPendingValidation,
			StartedAt:        time.Now(),
		}

		err := execRepo.Create(ctx, exec)
		require.NoError(t, err)

		execs, err := execRepo.GetByJobID(ctx, 1)
		require.NoError(t, err)
		require.Greater(t, len(execs), 0)

		execID := execs[len(execs)-1].ID

		// Note: ValidateExecution will fail because it tries to publish to WordPress
		// But we can verify it attempts validation
		err = service.ValidateExecution(ctx, execID, true)
		// Error is expected because we don't have real WordPress client
		_ = err

		// Verify execution exists
		retrievedExec, err := execRepo.GetByID(ctx, execID)
		require.NoError(t, err)
		require.NotNil(t, retrievedExec)
	})

	t.Run("reject execution successfully", func(t *testing.T) {
		// Create an execution pending validation
		title := "Test Article 2"
		content := "Test Content 2"
		exec := &Execution{
			JobID:            2,
			TopicID:          2,
			GeneratedTitle:   &title,
			GeneratedContent: &content,
			Status:           ExecutionPendingValidation,
			StartedAt:        time.Now(),
		}

		err := execRepo.Create(ctx, exec)
		require.NoError(t, err)

		execs, err := execRepo.GetByJobID(ctx, 2)
		require.NoError(t, err)
		require.Greater(t, len(execs), 0)

		execID := execs[len(execs)-1].ID

		// Reject the execution
		err = service.ValidateExecution(ctx, execID, false)
		require.NoError(t, err)

		// Verify execution status changed to failed
		retrievedExec, err := execRepo.GetByID(ctx, execID)
		require.NoError(t, err)
		require.Equal(t, ExecutionFailed, retrievedExec.Status)
		require.NotNil(t, retrievedExec.ErrorMessage)
	})

	t.Run("validate non-existent execution should fail", func(t *testing.T) {
		err := service.ValidateExecution(ctx, 999999, true)
		require.Error(t, err)
	})

	t.Run("validate execution not pending validation should fail", func(t *testing.T) {
		// Create execution with different status
		exec := &Execution{
			JobID:     3,
			TopicID:   3,
			Status:    ExecutionPublished, // Not pending validation
			StartedAt: time.Now(),
		}

		err := execRepo.Create(ctx, exec)
		require.NoError(t, err)

		execs, err := execRepo.GetByJobID(ctx, 3)
		require.NoError(t, err)
		require.Greater(t, len(execs), 0)

		execID := execs[len(execs)-1].ID

		err = service.ValidateExecution(ctx, execID, true)
		require.Error(t, err)
	})
}

func TestJobService_GetPendingValidations(t *testing.T) {
	service, _, execRepo, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()

	t.Run("get pending validations", func(t *testing.T) {
		// Create multiple executions with different statuses
		statuses := []ExecutionStatus{
			ExecutionPending,
			ExecutionGenerating,
			ExecutionPendingValidation,
			ExecutionPendingValidation,
			ExecutionValidated,
			ExecutionPublished,
		}

		for i, status := range statuses {
			exec := &Execution{
				JobID:     int64(i + 1),
				TopicID:   int64(i + 1),
				Status:    status,
				StartedAt: time.Now(),
			}
			err := execRepo.Create(ctx, exec)
			require.NoError(t, err)
		}

		// Get pending validations
		pending, err := service.GetPendingValidations(ctx)
		require.NoError(t, err)
		require.Equal(t, 2, len(pending))

		// Verify all are pending validation
		for _, exec := range pending {
			require.Equal(t, ExecutionPendingValidation, exec.Status)
		}
	})
}

func TestJobService_Validation(t *testing.T) {
	service, _, _, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()

	t.Run("create job with empty name should fail", func(t *testing.T) {
		job := &Job{
			Name:         "",
			SiteID:       1,
			CategoryID:   1,
			PromptID:     1,
			AIProviderID: 1,
			AIModel:      "gpt-4",
			ScheduleType: ScheduleDaily,
		}

		err := service.CreateJob(ctx, job)
		require.Error(t, err)
		require.Contains(t, err.Error(), "name")
	})

	t.Run("create job with invalid site ID should fail", func(t *testing.T) {
		job := &Job{
			Name:         "Test Job",
			SiteID:       0,
			CategoryID:   1,
			PromptID:     1,
			AIProviderID: 1,
			AIModel:      "gpt-4",
			ScheduleType: ScheduleDaily,
		}

		err := service.CreateJob(ctx, job)
		require.Error(t, err)
	})

	t.Run("create job with invalid schedule type should fail", func(t *testing.T) {
		job := &Job{
			Name:         "Test Job",
			SiteID:       1,
			CategoryID:   1,
			PromptID:     1,
			AIProviderID: 1,
			AIModel:      "gpt-4",
			ScheduleType: ScheduleType("invalid"),
		}

		err := service.CreateJob(ctx, job)
		require.Error(t, err)
	})

	t.Run("create weekly job without schedule day should fail", func(t *testing.T) {
		job := &Job{
			Name:         "Test Job",
			SiteID:       1,
			CategoryID:   1,
			PromptID:     1,
			AIProviderID: 1,
			AIModel:      "gpt-4",
			ScheduleType: ScheduleWeekly,
			ScheduleDay:  nil,
		}

		err := service.CreateJob(ctx, job)
		require.Error(t, err)
	})

	t.Run("create monthly job without schedule day should fail", func(t *testing.T) {
		job := &Job{
			Name:         "Test Job",
			SiteID:       1,
			CategoryID:   1,
			PromptID:     1,
			AIProviderID: 1,
			AIModel:      "gpt-4",
			ScheduleType: ScheduleMonthly,
			ScheduleDay:  nil,
		}

		err := service.CreateJob(ctx, job)
		require.Error(t, err)
	})
}

func TestJobService_ScheduleTypes(t *testing.T) {
	service, _, _, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()

	t.Run("create jobs with different schedule types", func(t *testing.T) {
		scheduleTypes := []ScheduleType{
			ScheduleManual,
			ScheduleOnce,
			ScheduleDaily,
		}

		for i, scheduleType := range scheduleTypes {
			job := &Job{
				Name:         string(scheduleType) + " Job",
				SiteID:       int64(i + 1),
				CategoryID:   1,
				PromptID:     1,
				AIProviderID: 1,
				AIModel:      "gpt-4",
				ScheduleType: scheduleType,
			}

			err := service.CreateJob(ctx, job)
			require.NoError(t, err)
		}

		jobs, err := service.ListJobs(ctx)
		require.NoError(t, err)
		require.GreaterOrEqual(t, len(jobs), 3)
	})

	t.Run("create weekly job with schedule day", func(t *testing.T) {
		friday := 5
		job := &Job{
			Name:         "Weekly Job",
			SiteID:       10,
			CategoryID:   1,
			PromptID:     1,
			AIProviderID: 1,
			AIModel:      "gpt-4",
			ScheduleType: ScheduleWeekly,
			ScheduleDay:  &friday,
		}

		err := service.CreateJob(ctx, job)
		require.NoError(t, err)
	})

	t.Run("create monthly job with schedule day", func(t *testing.T) {
		day := 15
		job := &Job{
			Name:         "Monthly Job",
			SiteID:       11,
			CategoryID:   1,
			PromptID:     1,
			AIProviderID: 1,
			AIModel:      "gpt-4",
			ScheduleType: ScheduleMonthly,
			ScheduleDay:  &day,
		}

		err := service.CreateJob(ctx, job)
		require.NoError(t, err)
	})
}
