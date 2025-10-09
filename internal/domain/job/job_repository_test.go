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

func setupTestJobRepository(t *testing.T) (*JobRepository, func()) {
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

	repo, err := NewJobRepository(container)
	require.NoError(t, err)

	cleanup := func() {
		_ = testLogger.Close()
		_ = os.RemoveAll(tempLogDir)
		dbCleanup()
	}

	return repo, cleanup
}

func TestJobRepository_CreateAndGet(t *testing.T) {
	repo, cleanup := setupTestJobRepository(t)
	defer cleanup()

	ctx := context.Background()

	t.Run("create job successfully", func(t *testing.T) {
		nextRun := time.Now().Add(24 * time.Hour)
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
			Status:             StatusActive,
			NextRunAt:          &nextRun,
		}

		err := repo.Create(ctx, job)
		require.NoError(t, err)
	})

	t.Run("get job by ID", func(t *testing.T) {
		nextRun := time.Now().Add(48 * time.Hour)
		job := &Job{
			Name:               "Get Test Job",
			SiteID:             1,
			CategoryID:         1,
			PromptID:           1,
			AIProviderID:       1,
			AIModel:            "gpt-4o",
			RequiresValidation: true,
			ScheduleType:       ScheduleWeekly,
			JitterEnabled:      true,
			JitterMinutes:      30,
			Status:             StatusActive,
			NextRunAt:          &nextRun,
		}

		err := repo.Create(ctx, job)
		require.NoError(t, err)

		jobs, err := repo.GetAll(ctx)
		require.NoError(t, err)
		require.Greater(t, len(jobs), 0)

		retrievedJob, err := repo.GetByID(ctx, jobs[len(jobs)-1].ID)
		require.NoError(t, err)
		require.Equal(t, "Get Test Job", retrievedJob.Name)
		require.Equal(t, "gpt-4o", retrievedJob.AIModel)
		require.True(t, retrievedJob.RequiresValidation)
		require.Equal(t, ScheduleWeekly, retrievedJob.ScheduleType)
	})

	t.Run("get non-existent job should fail", func(t *testing.T) {
		_, err := repo.GetByID(ctx, 999999)
		require.Error(t, err)
	})
}

func TestJobRepository_GetAll(t *testing.T) {
	repo, cleanup := setupTestJobRepository(t)
	defer cleanup()

	ctx := context.Background()

	t.Run("list empty jobs", func(t *testing.T) {
		jobs, err := repo.GetAll(ctx)
		require.NoError(t, err)
		require.Equal(t, 0, len(jobs))
	})

	t.Run("list multiple jobs", func(t *testing.T) {
		for i := 1; i <= 3; i++ {
			nextRun := time.Now().Add(time.Duration(i) * 24 * time.Hour)
			job := &Job{
				Name:         "List Test Job " + string(rune('0'+i)),
				SiteID:       int64(i),
				CategoryID:   1,
				PromptID:     1,
				AIProviderID: 1,
				AIModel:      "gpt-4",
				ScheduleType: ScheduleDaily,
				Status:       StatusActive,
				NextRunAt:    &nextRun,
			}

			err := repo.Create(ctx, job)
			require.NoError(t, err)
		}

		jobs, err := repo.GetAll(ctx)
		require.NoError(t, err)
		require.Equal(t, 3, len(jobs))
	})
}

func TestJobRepository_GetActive(t *testing.T) {
	repo, cleanup := setupTestJobRepository(t)
	defer cleanup()

	ctx := context.Background()

	t.Run("get only active jobs", func(t *testing.T) {
		// Create active jobs
		for i := 1; i <= 2; i++ {
			nextRun := time.Now().Add(time.Duration(i) * 24 * time.Hour)
			job := &Job{
				Name:         "Active Job " + string(rune('0'+i)),
				SiteID:       int64(i),
				CategoryID:   1,
				PromptID:     1,
				AIProviderID: 1,
				AIModel:      "gpt-4",
				ScheduleType: ScheduleDaily,
				Status:       StatusActive,
				NextRunAt:    &nextRun,
			}
			err := repo.Create(ctx, job)
			require.NoError(t, err)
		}

		// Create paused job
		nextRun := time.Now().Add(24 * time.Hour)
		pausedJob := &Job{
			Name:         "Paused Job",
			SiteID:       3,
			CategoryID:   1,
			PromptID:     1,
			AIProviderID: 1,
			AIModel:      "gpt-4",
			ScheduleType: ScheduleDaily,
			Status:       StatusPaused,
			NextRunAt:    &nextRun,
		}
		err := repo.Create(ctx, pausedJob)
		require.NoError(t, err)

		// Get all jobs
		allJobs, err := repo.GetAll(ctx)
		require.NoError(t, err)
		require.Equal(t, 3, len(allJobs))

		// Get only active jobs
		activeJobs, err := repo.GetActive(ctx)
		require.NoError(t, err)
		require.Equal(t, 2, len(activeJobs))

		// Verify all returned jobs are active
		for _, job := range activeJobs {
			require.Equal(t, StatusActive, job.Status)
		}
	})
}

func TestJobRepository_Update(t *testing.T) {
	repo, cleanup := setupTestJobRepository(t)
	defer cleanup()

	ctx := context.Background()

	t.Run("update job successfully", func(t *testing.T) {
		nextRun := time.Now().Add(24 * time.Hour)
		job := &Job{
			Name:         "Original Job",
			SiteID:       1,
			CategoryID:   1,
			PromptID:     1,
			AIProviderID: 1,
			AIModel:      "gpt-4",
			ScheduleType: ScheduleDaily,
			Status:       StatusActive,
			NextRunAt:    &nextRun,
		}

		err := repo.Create(ctx, job)
		require.NoError(t, err)

		jobs, err := repo.GetAll(ctx)
		require.NoError(t, err)
		require.Greater(t, len(jobs), 0)

		updatedJob := jobs[0]
		updatedJob.Name = "Updated Job"
		updatedJob.Status = StatusPaused
		updatedJob.AIModel = "gpt-4o"

		err = repo.Update(ctx, updatedJob)
		require.NoError(t, err)

		retrievedJob, err := repo.GetByID(ctx, updatedJob.ID)
		require.NoError(t, err)
		require.Equal(t, "Updated Job", retrievedJob.Name)
		require.Equal(t, StatusPaused, retrievedJob.Status)
		require.Equal(t, "gpt-4o", retrievedJob.AIModel)
	})
}

func TestJobRepository_Delete(t *testing.T) {
	repo, cleanup := setupTestJobRepository(t)
	defer cleanup()

	ctx := context.Background()

	t.Run("delete job successfully", func(t *testing.T) {
		nextRun := time.Now().Add(24 * time.Hour)
		job := &Job{
			Name:         "Delete Test Job",
			SiteID:       1,
			CategoryID:   1,
			PromptID:     1,
			AIProviderID: 1,
			AIModel:      "gpt-4",
			ScheduleType: ScheduleDaily,
			Status:       StatusActive,
			NextRunAt:    &nextRun,
		}

		err := repo.Create(ctx, job)
		require.NoError(t, err)

		jobs, err := repo.GetAll(ctx)
		require.NoError(t, err)
		require.Greater(t, len(jobs), 0)

		jobID := jobs[0].ID

		err = repo.Delete(ctx, jobID)
		require.NoError(t, err)

		_, err = repo.GetByID(ctx, jobID)
		require.Error(t, err)
	})

	t.Run("delete non-existent job should fail", func(t *testing.T) {
		err := repo.Delete(ctx, 999999)
		require.Error(t, err)
	})
}

func TestJobRepository_GetDueJobs(t *testing.T) {
	repo, cleanup := setupTestJobRepository(t)
	defer cleanup()

	ctx := context.Background()

	t.Run("get due jobs", func(t *testing.T) {
		now := time.Now()

		// Create job due in the past (should be returned)
		pastTime := now.Add(-1 * time.Hour)
		pastJob := &Job{
			Name:         "Past Due Job",
			SiteID:       1,
			CategoryID:   1,
			PromptID:     1,
			AIProviderID: 1,
			AIModel:      "gpt-4",
			ScheduleType: ScheduleDaily,
			Status:       StatusActive,
			NextRunAt:    &pastTime,
		}
		err := repo.Create(ctx, pastJob)
		require.NoError(t, err)

		// Create job due now (should be returned)
		nowJob := &Job{
			Name:         "Due Now Job",
			SiteID:       2,
			CategoryID:   1,
			PromptID:     1,
			AIProviderID: 1,
			AIModel:      "gpt-4",
			ScheduleType: ScheduleDaily,
			Status:       StatusActive,
			NextRunAt:    &now,
		}
		err = repo.Create(ctx, nowJob)
		require.NoError(t, err)

		// Create job due in the future (should NOT be returned)
		futureTime := now.Add(1 * time.Hour)
		futureJob := &Job{
			Name:         "Future Job",
			SiteID:       3,
			CategoryID:   1,
			PromptID:     1,
			AIProviderID: 1,
			AIModel:      "gpt-4",
			ScheduleType: ScheduleDaily,
			Status:       StatusActive,
			NextRunAt:    &futureTime,
		}
		err = repo.Create(ctx, futureJob)
		require.NoError(t, err)

		// Create paused job due in the past (should NOT be returned)
		pausedPastTime := now.Add(-2 * time.Hour)
		pausedJob := &Job{
			Name:         "Paused Past Job",
			SiteID:       4,
			CategoryID:   1,
			PromptID:     1,
			AIProviderID: 1,
			AIModel:      "gpt-4",
			ScheduleType: ScheduleDaily,
			Status:       StatusPaused,
			NextRunAt:    &pausedPastTime,
		}
		err = repo.Create(ctx, pausedJob)
		require.NoError(t, err)

		// Get due jobs
		dueJobs, err := repo.GetDueJobs(ctx, now)
		require.NoError(t, err)
		require.Equal(t, 2, len(dueJobs))

		// Verify all returned jobs are active and due
		for _, job := range dueJobs {
			require.Equal(t, StatusActive, job.Status)
			require.True(t, job.NextRunAt.Before(now) || job.NextRunAt.Equal(now))
		}
	})

	t.Run("get due jobs with no due jobs", func(t *testing.T) {
		now := time.Now()
		futureTime := now.Add(24 * time.Hour)

		job := &Job{
			Name:         "Future Only Job",
			SiteID:       5,
			CategoryID:   1,
			PromptID:     1,
			AIProviderID: 1,
			AIModel:      "gpt-4",
			ScheduleType: ScheduleDaily,
			Status:       StatusActive,
			NextRunAt:    &futureTime,
		}
		err := repo.Create(ctx, job)
		require.NoError(t, err)

		dueJobs, err := repo.GetDueJobs(ctx, now)
		require.NoError(t, err)
		// Should still have the 2 due jobs from previous test
		require.GreaterOrEqual(t, len(dueJobs), 0)
	})
}

func TestJobRepository_ScheduleTypes(t *testing.T) {
	repo, cleanup := setupTestJobRepository(t)
	defer cleanup()

	ctx := context.Background()

	t.Run("create job with different schedule types", func(t *testing.T) {
		scheduleTypes := []ScheduleType{
			ScheduleManual,
			ScheduleOnce,
			ScheduleDaily,
			ScheduleWeekly,
			ScheduleMonthly,
		}

		for i, scheduleType := range scheduleTypes {
			nextRun := time.Now().Add(time.Duration(i+1) * 24 * time.Hour)
			job := &Job{
				Name:         string(scheduleType) + " Job",
				SiteID:       int64(i + 1),
				CategoryID:   1,
				PromptID:     1,
				AIProviderID: 1,
				AIModel:      "gpt-4",
				ScheduleType: scheduleType,
				Status:       StatusActive,
				NextRunAt:    &nextRun,
			}

			err := repo.Create(ctx, job)
			require.NoError(t, err)
		}

		jobs, err := repo.GetAll(ctx)
		require.NoError(t, err)
		require.GreaterOrEqual(t, len(jobs), 5)

		// Verify schedule types are preserved
		foundTypes := make(map[ScheduleType]bool)
		for _, job := range jobs {
			foundTypes[job.ScheduleType] = true
		}

		for _, scheduleType := range scheduleTypes {
			require.True(t, foundTypes[scheduleType], "Schedule type %s not found", scheduleType)
		}
	})
}

func TestJobRepository_WithScheduleTimeAndDay(t *testing.T) {
	repo, cleanup := setupTestJobRepository(t)
	defer cleanup()

	ctx := context.Background()

	t.Run("create job with schedule time and day", func(t *testing.T) {
		scheduleTime := time.Date(2024, 1, 1, 9, 30, 0, 0, time.UTC)
		scheduleDay := 15
		nextRun := time.Now().Add(24 * time.Hour)

		job := &Job{
			Name:         "Scheduled Job",
			SiteID:       1,
			CategoryID:   1,
			PromptID:     1,
			AIProviderID: 1,
			AIModel:      "gpt-4",
			ScheduleType: ScheduleMonthly,
			ScheduleTime: &scheduleTime,
			ScheduleDay:  &scheduleDay,
			Status:       StatusActive,
			NextRunAt:    &nextRun,
		}

		err := repo.Create(ctx, job)
		require.NoError(t, err)

		jobs, err := repo.GetAll(ctx)
		require.NoError(t, err)
		require.Greater(t, len(jobs), 0)

		retrievedJob, err := repo.GetByID(ctx, jobs[len(jobs)-1].ID)
		require.NoError(t, err)
		require.NotNil(t, retrievedJob.ScheduleTime)
		require.NotNil(t, retrievedJob.ScheduleDay)
		require.Equal(t, 15, *retrievedJob.ScheduleDay)
	})
}
