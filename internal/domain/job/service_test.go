package job

import (
	"Postulator/internal/config"
	"Postulator/internal/domain/article"
	"Postulator/internal/domain/entities"
	"Postulator/internal/domain/prompt"
	"Postulator/internal/domain/site"
	"Postulator/internal/domain/topic"
	"Postulator/internal/infra/ai"
	"Postulator/internal/infra/database"
	"Postulator/internal/infra/wp"
	"Postulator/pkg/di"
	"Postulator/pkg/logger"
	"os"
	"path/filepath"
	"reflect"
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
	container.MustRegister(di.Instance[*ai.Client](ai.NewClient()))

	execRepo, err := NewExecutionRepository(container)
	if err != nil {
		dbCleanup()
		t.Fatalf("failed to create service: %v", err)
	}

	container.MustRegister(&di.Registration[IExecutionRepository]{
		Provider:      di.Must[IExecutionRepository](execRepo),
		Lifecycle:     di.Singleton,
		InterfaceType: reflect.TypeOf((*IExecutionRepository)(nil)).Elem(),
	})

	articleRepo, err := article.NewRepository(container)
	if err != nil {
		dbCleanup()
		t.Fatalf("failed to create service: %v", err)
	}

	container.MustRegister(&di.Registration[article.IRepository]{
		Provider:      di.Must[article.IRepository](articleRepo),
		Lifecycle:     di.Singleton,
		InterfaceType: reflect.TypeOf((*article.IRepository)(nil)).Elem(),
	})

	topicService, err := topic.NewService(container)
	if err != nil {
		dbCleanup()
		t.Fatalf("failed to create service: %v", err)
	}

	container.MustRegister(&di.Registration[topic.IService]{
		Provider:      di.Must[topic.IService](topicService),
		Lifecycle:     di.Singleton,
		InterfaceType: reflect.TypeOf((*topic.IService)(nil)).Elem(),
	})

	promptService, err := prompt.NewService(container)
	if err != nil {
		dbCleanup()
		t.Fatalf("failed to create service: %v", err)
	}

	container.MustRegister(&di.Registration[prompt.IService]{
		Provider:      di.Must[prompt.IService](promptService),
		Lifecycle:     di.Singleton,
		InterfaceType: reflect.TypeOf((*prompt.IService)(nil)).Elem(),
	})

	siteService, err := site.NewService(container)
	if err != nil {
		dbCleanup()
		t.Fatalf("failed to create service: %v", err)
	}

	container.MustRegister(&di.Registration[site.IService]{
		Provider:      di.Must[site.IService](siteService),
		Lifecycle:     di.Singleton,
		InterfaceType: reflect.TypeOf((*site.IService)(nil)).Elem(),
	})

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
