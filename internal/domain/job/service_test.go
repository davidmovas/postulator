package job

import (
	"Postulator/internal/config"
	"Postulator/internal/domain/article"
	"Postulator/internal/domain/entities"
	"Postulator/internal/infra/database"
	"Postulator/internal/infra/wp"
	"Postulator/pkg/di"
	"Postulator/pkg/logger"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func setupTestService(t *testing.T) (*Service, func()) {
	t.Helper()

	db, dbCleanup := database.SetupTestDB(t)

	tempLogDir := filepath.Join(os.TempDir(), "postulator_test_logs", t.Name())
	_ = os.MkdirAll(tempLogDir, 0755)

	container := di.New()

	testLogger, err := logger.NewForTest(&config.Config{
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
	container.MustRegister(di.Instance[*wp.Client](wp.NewClient()))

	execRepo, err := NewExecutionRepository(container)
	if err != nil {
		dbCleanup()
		t.Fatalf("failed to create service: %v", err)
	}
	container.MustRegister(di.Instance[IExecutionRepository](execRepo))

	articleRepo, err := article.NewRepository(container)
	if err != nil {
		return nil,
	}

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

func TestJobService_CreateAndGet(t *testing.T) {
	service, cleanup := setupTestService(t)
	defer cleanup()

	t.Run("create job successfully", func(t *testing.T) {
		interval := 1000

		job := &Job{
			Name:               "test_job_1",
			SiteID:             1,
			CategoryID:         1,
			PromptID:           1,
			AIProviderID:       1,
			AIModel:            string(entities.ModelGPT4OMini),
			RequiresValidation: false,
			ScheduleType:       ScheduleManual,
			IntervalValue:      &interval,
			JitterEnabled:      true,
			JitterMinutes:      30,
			Status:             StatusActive,
		}

		err := service.CreateJob(t.Context(), job)
		require.NoError(t, err)
	})
}
