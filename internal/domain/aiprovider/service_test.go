package aiprovider

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

	"github.com/stretchr/testify/require"
)

func setupTestService(t *testing.T) (*Service, func()) {
	t.Helper()

	db, dbCleanup := database.SetupTestDB(t)

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

	service, err := NewService(container)
	require.NoError(t, err)

	cleanup := func() {
		_ = testLogger.Close()
		_ = os.RemoveAll(tempLogDir)
		dbCleanup()
	}

	return service, cleanup
}

func TestAIProviderService_CreateAndGet(t *testing.T) {
	service, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()

	t.Run("create provider successfully", func(t *testing.T) {
		provider := &entities.AIProvider{
			Name:     "OpenAI",
			APIKey:   "sk-test-key-123",
			Model:    "gpt-4",
			IsActive: true,
		}

		err := service.CreateProvider(ctx, provider)
		require.NoError(t, err)
	})

	t.Run("get provider by ID", func(t *testing.T) {
		provider := &entities.AIProvider{
			Name:     "Anthropic",
			APIKey:   "sk-ant-test-key",
			Model:    "claude-3-opus-20240229",
			IsActive: true,
		}

		err := service.CreateProvider(ctx, provider)
		require.NoError(t, err)

		providers, err := service.ListProviders(ctx)
		require.NoError(t, err)
		require.Greater(t, len(providers), 0)

		retrievedProvider, err := service.GetProvider(ctx, providers[len(providers)-1].ID)
		require.NoError(t, err)
		require.Equal(t, "Anthropic", retrievedProvider.Name)
		require.Equal(t, "claude-3-opus-20240229", retrievedProvider.Model)
		require.True(t, retrievedProvider.IsActive)
	})

	t.Run("create provider with duplicate name should fail", func(t *testing.T) {
		provider1 := &entities.AIProvider{
			Name:     "Google",
			APIKey:   "key1",
			Model:    "gemini-pro",
			IsActive: true,
		}

		err := service.CreateProvider(ctx, provider1)
		require.NoError(t, err)

		provider2 := &entities.AIProvider{
			Name:     "Google",
			APIKey:   "key2",
			Model:    "gemini-1.5-pro",
			IsActive: true,
		}

		err = service.CreateProvider(ctx, provider2)
		require.Error(t, err)
	})

	t.Run("create provider with empty name should fail", func(t *testing.T) {
		provider := &entities.AIProvider{
			Name:     "",
			APIKey:   "key",
			Model:    "gpt-4",
			IsActive: true,
		}

		err := service.CreateProvider(ctx, provider)
		require.Error(t, err)
		require.Contains(t, err.Error(), "name cannot be empty")
	})

	t.Run("create provider with empty API key should fail", func(t *testing.T) {
		provider := &entities.AIProvider{
			Name:     "openai",
			APIKey:   "",
			Model:    "gpt-4",
			IsActive: true,
		}

		err := service.CreateProvider(ctx, provider)
		require.Error(t, err)
		require.Contains(t, err.Error(), "API key cannot be empty")
	})

	t.Run("create provider with empty model should fail", func(t *testing.T) {
		provider := &entities.AIProvider{
			Name:     "TestProvider",
			APIKey:   "key",
			Model:    "",
			IsActive: true,
		}

		err := service.CreateProvider(ctx, provider)
		require.Error(t, err)
		require.Contains(t, err.Error(), "model cannot be empty")
	})

	t.Run("get non-existent provider should fail", func(t *testing.T) {
		_, err := service.GetProvider(ctx, 999999)
		require.Error(t, err)
	})
}

func TestAIProviderService_ListProviders(t *testing.T) {
	service, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()

	t.Run("list empty providers", func(t *testing.T) {
		providers, err := service.ListProviders(ctx)
		require.NoError(t, err)
		require.Equal(t, 0, len(providers))
	})

	t.Run("list multiple providers", func(t *testing.T) {
		models := []string{"gpt-4", "gpt-4o", "gpt-3.5-turbo"}
		for i := 1; i <= 3; i++ {
			provider := &entities.AIProvider{
				Name:     "openai",
				APIKey:   "key" + string(rune('0'+i)),
				Model:    models[i-1],
				IsActive: true,
			}

			err := service.CreateProvider(ctx, provider)
			// First one should succeed, others will fail due to duplicate name
			if i == 1 {
				require.NoError(t, err)
			}
		}

		providers, err := service.ListProviders(ctx)
		require.NoError(t, err)
		require.GreaterOrEqual(t, len(providers), 1)
	})
}

func TestAIProviderService_ListActiveProviders(t *testing.T) {
	service, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()

	t.Run("list only active providers", func(t *testing.T) {
		// Create 2 active providers
		activeProvider1 := &entities.AIProvider{
			Name:     "openai",
			APIKey:   "key1",
			Model:    "gpt-4",
			IsActive: true,
		}
		err := service.CreateProvider(ctx, activeProvider1)
		require.NoError(t, err)

		activeProvider2 := &entities.AIProvider{
			Name:     "anthropic",
			APIKey:   "key2",
			Model:    "claude-3-opus-20240229",
			IsActive: true,
		}
		err = service.CreateProvider(ctx, activeProvider2)
		require.NoError(t, err)

		// Create 1 inactive provider
		inactiveProvider := &entities.AIProvider{
			Name:     "google",
			APIKey:   "key-inactive",
			Model:    "gemini-pro",
			IsActive: false,
		}
		err = service.CreateProvider(ctx, inactiveProvider)
		require.NoError(t, err)

		// List all providers
		allProviders, err := service.ListProviders(ctx)
		require.NoError(t, err)
		require.Equal(t, 3, len(allProviders))

		// List only active providers
		activeProviders, err := service.ListActiveProviders(ctx)
		require.NoError(t, err)
		require.Equal(t, 2, len(activeProviders))

		// Verify all returned providers are active
		for _, p := range activeProviders {
			require.True(t, p.IsActive)
		}
	})
}

func TestAIProviderService_UpdateProvider(t *testing.T) {
	service, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()

	t.Run("update provider successfully", func(t *testing.T) {
		provider := &entities.AIProvider{
			Name:     "openai",
			APIKey:   "original-key",
			Model:    "gpt-4",
			IsActive: true,
		}

		err := service.CreateProvider(ctx, provider)
		require.NoError(t, err)

		providers, err := service.ListProviders(ctx)
		require.NoError(t, err)
		require.Greater(t, len(providers), 0)

		updatedProvider := providers[0]
		updatedProvider.Model = "gpt-4o"
		updatedProvider.IsActive = false

		err = service.UpdateProvider(ctx, updatedProvider)
		require.NoError(t, err)

		retrievedProvider, err := service.GetProvider(ctx, updatedProvider.ID)
		require.NoError(t, err)
		require.Equal(t, "openai", retrievedProvider.Name)
		require.Equal(t, "gpt-4o", retrievedProvider.Model)
		require.False(t, retrievedProvider.IsActive)
	})

	t.Run("update provider with invalid data should fail", func(t *testing.T) {
		provider := &entities.AIProvider{
			Name:     "anthropic",
			APIKey:   "valid-key",
			Model:    "claude-3-haiku-20240307",
			IsActive: true,
		}

		err := service.CreateProvider(ctx, provider)
		require.NoError(t, err)

		providers, err := service.ListProviders(ctx)
		require.NoError(t, err)
		require.Greater(t, len(providers), 0)

		invalidProvider := providers[len(providers)-1]
		invalidProvider.Name = ""

		err = service.UpdateProvider(ctx, invalidProvider)
		require.Error(t, err)
	})
}

func TestAIProviderService_DeleteProvider(t *testing.T) {
	service, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()

	t.Run("delete provider successfully", func(t *testing.T) {
		provider := &entities.AIProvider{
			Name:     "google",
			APIKey:   "delete-key",
			Model:    "gemini-1.5-flash",
			IsActive: true,
		}

		err := service.CreateProvider(ctx, provider)
		require.NoError(t, err)

		providers, err := service.ListProviders(ctx)
		require.NoError(t, err)
		require.Greater(t, len(providers), 0)

		providerID := providers[0].ID

		err = service.DeleteProvider(ctx, providerID)
		require.NoError(t, err)

		_, err = service.GetProvider(ctx, providerID)
		require.Error(t, err)
	})

	t.Run("delete non-existent provider should fail", func(t *testing.T) {
		err := service.DeleteProvider(ctx, 999999)
		require.Error(t, err)
	})
}

func TestAIProviderService_SetProviderStatus(t *testing.T) {
	service, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()

	t.Run("set provider status successfully", func(t *testing.T) {
		provider := &entities.AIProvider{
			Name:     "anthropic",
			APIKey:   "status-key",
			Model:    "claude-3-sonnet-20240229",
			IsActive: true,
		}

		err := service.CreateProvider(ctx, provider)
		require.NoError(t, err)

		providers, err := service.ListProviders(ctx)
		require.NoError(t, err)
		require.Greater(t, len(providers), 0)

		providerID := providers[len(providers)-1].ID

		// Deactivate provider
		err = service.SetProviderStatus(ctx, providerID, false)
		require.NoError(t, err)

		retrievedProvider, err := service.GetProvider(ctx, providerID)
		require.NoError(t, err)
		require.False(t, retrievedProvider.IsActive)

		// Reactivate provider
		err = service.SetProviderStatus(ctx, providerID, true)
		require.NoError(t, err)

		retrievedProvider, err = service.GetProvider(ctx, providerID)
		require.NoError(t, err)
		require.True(t, retrievedProvider.IsActive)
	})

	t.Run("set status for non-existent provider should fail", func(t *testing.T) {
		err := service.SetProviderStatus(ctx, 999999, false)
		require.Error(t, err)
	})
}

func TestAIProviderService_Validation(t *testing.T) {
	service, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()

	t.Run("create provider with whitespace-only name should fail", func(t *testing.T) {
		provider := &entities.AIProvider{
			Name:     "   ",
			APIKey:   "key",
			Model:    "gpt-4o-mini",
			IsActive: true,
		}

		err := service.CreateProvider(ctx, provider)
		require.Error(t, err)
	})

	t.Run("create provider trims whitespace from name and model", func(t *testing.T) {
		provider := &entities.AIProvider{
			Name:     "  openai  ",
			APIKey:   "key",
			Model:    "  gpt-4  ",
			IsActive: true,
		}

		err := service.CreateProvider(ctx, provider)
		require.NoError(t, err)

		providers, err := service.ListProviders(ctx)
		require.NoError(t, err)
		require.Greater(t, len(providers), 0)

		lastProvider := providers[len(providers)-1]
		require.Equal(t, "openai", lastProvider.Name)
		require.Equal(t, "gpt-4", lastProvider.Model)
	})
}

func TestAIProviderService_GetAvailableModels(t *testing.T) {
	service, cleanup := setupTestService(t)
	defer cleanup()

	t.Run("get OpenAI models", func(t *testing.T) {
		models := service.GetAvailableModels("openai")
		require.Greater(t, len(models), 0)
		require.Contains(t, models, entities.ModelGPT4)
		require.Contains(t, models, entities.ModelGPT4O)
		require.Contains(t, models, entities.ModelGPT35Turbo)
	})

	t.Run("get Anthropic models", func(t *testing.T) {
		models := service.GetAvailableModels("anthropic")
		require.Greater(t, len(models), 0)
		require.Contains(t, models, entities.ModelClaude3Opus)
		require.Contains(t, models, entities.ModelClaude35Sonnet)
	})

	t.Run("get Google models", func(t *testing.T) {
		models := service.GetAvailableModels("google")
		require.Greater(t, len(models), 0)
		require.Contains(t, models, entities.ModelGeminiPro)
		require.Contains(t, models, entities.ModelGemini15Pro)
	})

	t.Run("get models for unknown provider returns empty list", func(t *testing.T) {
		models := service.GetAvailableModels("unknown-provider")
		require.Equal(t, 0, len(models))
	})

	t.Run("provider name is case insensitive", func(t *testing.T) {
		models1 := service.GetAvailableModels("OpenAI")
		models2 := service.GetAvailableModels("OPENAI")
		models3 := service.GetAvailableModels("openai")
		require.Equal(t, len(models1), len(models2))
		require.Equal(t, len(models1), len(models3))
	})
}

func TestAIProviderService_ValidateModel(t *testing.T) {
	service, cleanup := setupTestService(t)
	defer cleanup()

	t.Run("validate valid OpenAI model", func(t *testing.T) {
		err := service.ValidateModel("openai", "gpt-4")
		require.NoError(t, err)

		err = service.ValidateModel("openai", "gpt-4o")
		require.NoError(t, err)

		err = service.ValidateModel("openai", "gpt-3.5-turbo")
		require.NoError(t, err)
	})

	t.Run("validate valid Anthropic model", func(t *testing.T) {
		err := service.ValidateModel("anthropic", "claude-3-opus-20240229")
		require.NoError(t, err)

		err = service.ValidateModel("anthropic", "claude-3-5-sonnet-20241022")
		require.NoError(t, err)
	})

	t.Run("validate valid Google model", func(t *testing.T) {
		err := service.ValidateModel("google", "gemini-pro")
		require.NoError(t, err)

		err = service.ValidateModel("google", "gemini-1.5-pro")
		require.NoError(t, err)
	})

	t.Run("validate invalid model for OpenAI", func(t *testing.T) {
		err := service.ValidateModel("openai", "claude-3-opus-20240229")
		require.Error(t, err)
		require.Contains(t, err.Error(), "not valid for provider")
	})

	t.Run("validate invalid model for Anthropic", func(t *testing.T) {
		err := service.ValidateModel("anthropic", "gpt-4")
		require.Error(t, err)
		require.Contains(t, err.Error(), "not valid for provider")
	})

	t.Run("validate model with empty model should fail", func(t *testing.T) {
		err := service.ValidateModel("openai", "")
		require.Error(t, err)
		require.Contains(t, err.Error(), "model cannot be empty")
	})

	t.Run("validate model with unknown provider should fail", func(t *testing.T) {
		err := service.ValidateModel("unknown-provider", "gpt-4")
		require.Error(t, err)
		require.Contains(t, err.Error(), "unknown provider")
	})

	t.Run("provider name is case insensitive", func(t *testing.T) {
		err1 := service.ValidateModel("OpenAI", "gpt-4")
		err2 := service.ValidateModel("OPENAI", "gpt-4")
		err3 := service.ValidateModel("openai", "gpt-4")
		require.NoError(t, err1)
		require.NoError(t, err2)
		require.NoError(t, err3)
	})
}

func TestAIProviderService_CreateWithModelValidation(t *testing.T) {
	service, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()

	t.Run("create OpenAI provider with valid model", func(t *testing.T) {
		provider := &entities.AIProvider{
			Name:     "openai",
			APIKey:   "sk-test-key",
			Model:    "gpt-4o",
			IsActive: true,
		}

		err := service.CreateProvider(ctx, provider)
		require.NoError(t, err)
	})

	t.Run("create Anthropic provider with valid model", func(t *testing.T) {
		provider := &entities.AIProvider{
			Name:     "anthropic",
			APIKey:   "sk-ant-test-key",
			Model:    "claude-3-5-sonnet-20241022",
			IsActive: true,
		}

		err := service.CreateProvider(ctx, provider)
		require.NoError(t, err)
	})

	t.Run("create Google provider with invalid Anthropic model should fail", func(t *testing.T) {
		provider := &entities.AIProvider{
			Name:     "google",
			APIKey:   "sk-test-key",
			Model:    "claude-3-opus-20240229",
			IsActive: true,
		}

		err := service.CreateProvider(ctx, provider)
		require.Error(t, err)
		require.Contains(t, err.Error(), "not valid for provider")
	})

	t.Run("create OpenAI provider with invalid Anthropic model should fail", func(t *testing.T) {
		provider := &entities.AIProvider{
			Name:     "openai",
			APIKey:   "sk-test-key-2",
			Model:    "claude-3-haiku-20240307",
			IsActive: true,
		}

		err := service.CreateProvider(ctx, provider)
		require.Error(t, err)
		require.Contains(t, err.Error(), "not valid for provider")
	})

	t.Run("update provider with invalid model should fail", func(t *testing.T) {
		provider := &entities.AIProvider{
			Name:     "google",
			APIKey:   "sk-test-key-3",
			Model:    "gemini-pro",
			IsActive: true,
		}

		err := service.CreateProvider(ctx, provider)
		require.NoError(t, err)

		providers, err := service.ListProviders(ctx)
		require.NoError(t, err)
		require.Greater(t, len(providers), 0)

		// Try to update with invalid model (OpenAI model for Google provider)
		lastProvider := providers[len(providers)-1]
		lastProvider.Model = "gpt-4"

		err = service.UpdateProvider(ctx, lastProvider)
		require.Error(t, err)
		require.Contains(t, err.Error(), "not valid for provider")
	})
}
