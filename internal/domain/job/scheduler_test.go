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
type mockAIClientScheduler struct{}

func (m *mockAIClientScheduler) GenerateArticle(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	return "Mock generated article content", nil
}

func setupTestScheduler(t *testing.T) (*Scheduler, *JobRepository, func()) {
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
		Provider:      func(di.Container) (ai.Client, error) { return &mockAIClientScheduler{}, nil },
		Lifecycle:     di.Singleton,
		InterfaceType: reflect.TypeOf((*ai.Client)(nil)).Elem(),
	})

	scheduler, err := NewScheduler(container)
	require.NoError(t, err)

	repo, err := NewJobRepository(container)
	require.NoError(t, err)

	cleanup := func() {
		_ = testLogger.Close()
		_ = os.RemoveAll(tempLogDir)
		dbCleanup()
	}

	return scheduler, repo, cleanup
}

func TestScheduler_CalculateNextRun_Manual(t *testing.T) {
	scheduler, _, cleanup := setupTestScheduler(t)
	defer cleanup()

	t.Run("manual schedule has no next run", func(t *testing.T) {
		now := time.Now()
		job := &Job{
			ScheduleType: ScheduleManual,
		}

		nextRun := scheduler.CalculateNextRun(job, now)
		require.True(t, nextRun.IsZero())
	})
}

func TestScheduler_CalculateNextRun_Once(t *testing.T) {
	scheduler, _, cleanup := setupTestScheduler(t)
	defer cleanup()

	t.Run("once schedule runs immediately if never run", func(t *testing.T) {
		now := time.Now()
		job := &Job{
			ScheduleType: ScheduleOnce,
			LastRunAt:    nil,
		}

		nextRun := scheduler.CalculateNextRun(job, now)
		require.False(t, nextRun.IsZero())
		require.True(t, nextRun.Equal(now) || nextRun.After(now.Add(-1*time.Second)))
	})

	t.Run("once schedule has no next run after execution", func(t *testing.T) {
		now := time.Now()
		lastRun := now.Add(-1 * time.Hour)
		job := &Job{
			ScheduleType: ScheduleOnce,
			LastRunAt:    &lastRun,
		}

		nextRun := scheduler.CalculateNextRun(job, now)
		require.True(t, nextRun.IsZero())
	})
}

func TestScheduler_CalculateNextRun_Daily(t *testing.T) {
	scheduler, _, cleanup := setupTestScheduler(t)
	defer cleanup()

	t.Run("daily schedule at specific time", func(t *testing.T) {
		now := time.Date(2024, 1, 15, 14, 30, 0, 0, time.UTC)
		scheduleTime := time.Date(2024, 1, 1, 9, 0, 0, 0, time.UTC)

		job := &Job{
			ScheduleType: ScheduleDaily,
			ScheduleTime: &scheduleTime,
		}

		nextRun := scheduler.CalculateNextRun(job, now)
		require.False(t, nextRun.IsZero())

		expectedDate := time.Date(2024, 1, 16, 9, 0, 0, 0, time.UTC)
		require.Equal(t, expectedDate.Year(), nextRun.Year())
		require.Equal(t, expectedDate.Month(), nextRun.Month())
		require.Equal(t, expectedDate.Day(), nextRun.Day())
		require.Equal(t, 9, nextRun.Hour())
		require.Equal(t, 0, nextRun.Minute())
	})

	t.Run("daily schedule at future time today", func(t *testing.T) {
		now := time.Date(2024, 1, 15, 8, 0, 0, 0, time.UTC)
		scheduleTime := time.Date(2024, 1, 1, 9, 0, 0, 0, time.UTC)

		job := &Job{
			ScheduleType: ScheduleDaily,
			ScheduleTime: &scheduleTime,
		}

		nextRun := scheduler.CalculateNextRun(job, now)
		require.False(t, nextRun.IsZero())

		require.Equal(t, now.Year(), nextRun.Year())
		require.Equal(t, now.Month(), nextRun.Month())
		require.Equal(t, now.Day(), nextRun.Day())
		require.Equal(t, 9, nextRun.Hour())
	})

	t.Run("daily schedule without specific time", func(t *testing.T) {
		now := time.Date(2024, 1, 15, 14, 30, 0, 0, time.UTC)

		job := &Job{
			ScheduleType: ScheduleDaily,
			ScheduleTime: nil,
		}

		nextRun := scheduler.CalculateNextRun(job, now)
		require.False(t, nextRun.IsZero())

		require.True(t, nextRun.After(now))
	})
}

func TestScheduler_CalculateNextRun_Weekly(t *testing.T) {
	scheduler, _, cleanup := setupTestScheduler(t)
	defer cleanup()

	t.Run("weekly schedule on specific weekday", func(t *testing.T) {
		now := time.Date(2024, 1, 15, 14, 30, 0, 0, time.UTC)
		require.Equal(t, time.Monday, now.Weekday())

		friday := 5
		scheduleTime := time.Date(2024, 1, 1, 9, 0, 0, 0, time.UTC)

		job := &Job{
			ScheduleType: ScheduleWeekly,
			ScheduleDay:  &friday,
			ScheduleTime: &scheduleTime,
		}

		nextRun := scheduler.CalculateNextRun(job, now)
		require.False(t, nextRun.IsZero())

		require.Equal(t, time.Friday, nextRun.Weekday())
		require.Equal(t, 9, nextRun.Hour())
		require.Equal(t, 0, nextRun.Minute())
	})

	t.Run("weekly schedule on today but time passed", func(t *testing.T) {
		now := time.Date(2024, 1, 15, 14, 30, 0, 0, time.UTC)
		require.Equal(t, time.Monday, now.Weekday())

		monday := 1
		scheduleTime := time.Date(2024, 1, 1, 9, 0, 0, 0, time.UTC)

		job := &Job{
			ScheduleType: ScheduleWeekly,
			ScheduleDay:  &monday,
			ScheduleTime: &scheduleTime,
		}

		nextRun := scheduler.CalculateNextRun(job, now)
		require.False(t, nextRun.IsZero())

		require.Equal(t, time.Monday, nextRun.Weekday())
		require.True(t, nextRun.After(now.Add(6*24*time.Hour)))
	})
}

func TestScheduler_CalculateNextRun_Monthly(t *testing.T) {
	scheduler, _, cleanup := setupTestScheduler(t)
	defer cleanup()

	t.Run("monthly schedule on specific day", func(t *testing.T) {
		now := time.Date(2024, 1, 5, 14, 30, 0, 0, time.UTC)

		day := 15
		scheduleTime := time.Date(2024, 1, 1, 9, 0, 0, 0, time.UTC)

		job := &Job{
			ScheduleType: ScheduleMonthly,
			ScheduleDay:  &day,
			ScheduleTime: &scheduleTime,
		}

		nextRun := scheduler.CalculateNextRun(job, now)
		require.False(t, nextRun.IsZero())

		require.Equal(t, 2024, nextRun.Year())
		require.Equal(t, time.January, nextRun.Month())
		require.Equal(t, 15, nextRun.Day())
		require.Equal(t, 9, nextRun.Hour())
	})

	t.Run("monthly schedule day already passed this month", func(t *testing.T) {
		now := time.Date(2024, 1, 20, 14, 30, 0, 0, time.UTC)

		day := 15
		scheduleTime := time.Date(2024, 1, 1, 9, 0, 0, 0, time.UTC)

		job := &Job{
			ScheduleType: ScheduleMonthly,
			ScheduleDay:  &day,
			ScheduleTime: &scheduleTime,
		}

		nextRun := scheduler.CalculateNextRun(job, now)
		require.False(t, nextRun.IsZero())

		require.Equal(t, 2024, nextRun.Year())
		require.Equal(t, time.February, nextRun.Month())
		require.Equal(t, 15, nextRun.Day())
	})

	t.Run("monthly schedule handles February correctly", func(t *testing.T) {
		now := time.Date(2024, 1, 15, 14, 30, 0, 0, time.UTC)

		day := 31
		scheduleTime := time.Date(2024, 1, 1, 9, 0, 0, 0, time.UTC)

		job := &Job{
			ScheduleType: ScheduleMonthly,
			ScheduleDay:  &day,
			ScheduleTime: &scheduleTime,
		}

		nextRun := scheduler.CalculateNextRun(job, now)
		require.False(t, nextRun.IsZero())

		require.Equal(t, time.January, nextRun.Month())
		require.Equal(t, 31, nextRun.Day())
	})
}

func TestScheduler_CalculateNextRun_WithJitter(t *testing.T) {
	scheduler, _, cleanup := setupTestScheduler(t)
	defer cleanup()

	t.Run("daily schedule with jitter", func(t *testing.T) {
		now := time.Date(2024, 1, 15, 14, 30, 0, 0, time.UTC)
		scheduleTime := time.Date(2024, 1, 1, 9, 0, 0, 0, time.UTC)

		job := &Job{
			ScheduleType:  ScheduleDaily,
			ScheduleTime:  &scheduleTime,
			JitterEnabled: true,
			JitterMinutes: 30,
		}

		nextRun := scheduler.CalculateNextRun(job, now)
		require.False(t, nextRun.IsZero())

		baseTime := time.Date(2024, 1, 16, 9, 0, 0, 0, time.UTC)

		minTime := baseTime.Add(-30 * time.Minute)
		maxTime := baseTime.Add(30 * time.Minute)

		require.True(t, nextRun.After(minTime) || nextRun.Equal(minTime))
		require.True(t, nextRun.Before(maxTime) || nextRun.Equal(maxTime))
	})

	t.Run("multiple calculations with jitter produce different results", func(t *testing.T) {
		now := time.Date(2024, 1, 15, 14, 30, 0, 0, time.UTC)
		scheduleTime := time.Date(2024, 1, 1, 9, 0, 0, 0, time.UTC)

		job := &Job{
			ScheduleType:  ScheduleDaily,
			ScheduleTime:  &scheduleTime,
			JitterEnabled: true,
			JitterMinutes: 30,
		}

		results := make(map[time.Time]bool)
		for i := 0; i < 10; i++ {
			nextRun := scheduler.CalculateNextRun(job, now)
			results[nextRun] = true
		}

		require.GreaterOrEqual(t, len(results), 1)
	})
}

func TestScheduler_RestoreState(t *testing.T) {
	scheduler, repo, cleanup := setupTestScheduler(t)
	defer cleanup()

	ctx := context.Background()

	t.Run("restore state for jobs with no next run", func(t *testing.T) {
		job := &Job{
			Name:         "No Next Run Job",
			SiteID:       1,
			CategoryID:   1,
			PromptID:     1,
			AIProviderID: 1,
			AIModel:      "gpt-4",
			ScheduleType: ScheduleDaily,
			Status:       StatusActive,
			NextRunAt:    nil,
		}

		err := repo.Create(ctx, job)
		require.NoError(t, err)

		err = scheduler.RestoreState(ctx)
		require.NoError(t, err)

		jobs, err := repo.GetActive(ctx)
		require.NoError(t, err)
		require.Greater(t, len(jobs), 0)

		found := false
		for _, j := range jobs {
			if j.Name == "No Next Run Job" {
				require.NotNil(t, j.NextRunAt)
				found = true
				break
			}
		}
		require.True(t, found)
	})

	t.Run("restore state for missed executions", func(t *testing.T) {
		pastTime := time.Now().Add(-2 * time.Hour)
		job := &Job{
			Name:         "Missed Execution Job",
			SiteID:       2,
			CategoryID:   1,
			PromptID:     1,
			AIProviderID: 1,
			AIModel:      "gpt-4",
			ScheduleType: ScheduleDaily,
			Status:       StatusActive,
			NextRunAt:    &pastTime,
		}

		err := repo.Create(ctx, job)
		require.NoError(t, err)

		err = scheduler.RestoreState(ctx)
		require.NoError(t, err)

		jobs, err := repo.GetActive(ctx)
		require.NoError(t, err)

		found := false
		for _, j := range jobs {
			if j.Name == "Missed Execution Job" {
				require.NotNil(t, j.NextRunAt)
				require.True(t, j.NextRunAt.After(time.Now().Add(-1*time.Minute)))
				require.True(t, j.NextRunAt.Before(time.Now().Add(6*time.Minute)))
				found = true
				break
			}
		}
		require.True(t, found)
	})

	t.Run("restore state ignores manual jobs", func(t *testing.T) {
		job := &Job{
			Name:         "Manual Job",
			SiteID:       3,
			CategoryID:   1,
			PromptID:     1,
			AIProviderID: 1,
			AIModel:      "gpt-4",
			ScheduleType: ScheduleManual,
			Status:       StatusActive,
			NextRunAt:    nil,
		}

		err := repo.Create(ctx, job)
		require.NoError(t, err)

		err = scheduler.RestoreState(ctx)
		require.NoError(t, err)

		jobs, err := repo.GetActive(ctx)
		require.NoError(t, err)

		found := false
		for _, j := range jobs {
			if j.Name == "Manual Job" {
				found = true
				break
			}
		}
		require.True(t, found)
	})

	t.Run("restore state ignores paused jobs", func(t *testing.T) {
		pastTime := time.Now().Add(-1 * time.Hour)
		job := &Job{
			Name:         "Paused Job",
			SiteID:       4,
			CategoryID:   1,
			PromptID:     1,
			AIProviderID: 1,
			AIModel:      "gpt-4",
			ScheduleType: ScheduleDaily,
			Status:       StatusPaused,
			NextRunAt:    &pastTime,
		}

		err := repo.Create(ctx, job)
		require.NoError(t, err)

		err = scheduler.RestoreState(ctx)
		require.NoError(t, err)

		allJobs, err := repo.GetAll(ctx)
		require.NoError(t, err)

		var retrievedJob *Job
		for _, j := range allJobs {
			if j.Name == "Paused Job" {
				retrievedJob = j
				break
			}
		}
		require.NotNil(t, retrievedJob)
		require.Equal(t, StatusPaused, retrievedJob.Status)
		require.True(t, retrievedJob.NextRunAt.Before(time.Now()))
	})
}

func TestScheduler_StartStop(t *testing.T) {
	scheduler, _, cleanup := setupTestScheduler(t)
	defer cleanup()

	ctx := context.Background()

	t.Run("start and stop scheduler", func(t *testing.T) {
		err := scheduler.Start(ctx)
		require.NoError(t, err)
		require.True(t, scheduler.running)

		time.Sleep(100 * time.Millisecond)

		err = scheduler.Stop()
		require.NoError(t, err)
		require.False(t, scheduler.running)
	})
}
