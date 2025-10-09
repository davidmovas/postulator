package prompt

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

func TestPromptService_CreateAndGet(t *testing.T) {
	service, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()

	t.Run("create prompt successfully", func(t *testing.T) {
		prompt := &entities.Prompt{
			Name:         "Test Prompt",
			SystemPrompt: "You are a helpful assistant.",
			UserPrompt:   "Generate text on topic {{title}} with {{words}} words.",
		}

		err := service.CreatePrompt(ctx, prompt)
		require.NoError(t, err)
		require.Equal(t, 2, len(prompt.Placeholders))
		require.Contains(t, prompt.Placeholders, "title")
		require.Contains(t, prompt.Placeholders, "words")
	})

	t.Run("get prompt by ID", func(t *testing.T) {
		prompt := &entities.Prompt{
			Name:         "Get Test Prompt",
			SystemPrompt: "System instructions",
			UserPrompt:   "User instructions with {{placeholder}}",
		}

		err := service.CreatePrompt(ctx, prompt)
		require.NoError(t, err)

		prompts, err := service.ListPrompts(ctx)
		require.NoError(t, err)
		require.Greater(t, len(prompts), 0)

		retrievedPrompt, err := service.GetPrompt(ctx, prompts[len(prompts)-1].ID)
		require.NoError(t, err)
		require.Equal(t, "Get Test Prompt", retrievedPrompt.Name)
		require.Contains(t, retrievedPrompt.Placeholders, "placeholder")
	})

	t.Run("create prompt without name should fail", func(t *testing.T) {
		prompt := &entities.Prompt{
			Name:         "",
			SystemPrompt: "System prompt",
			UserPrompt:   "User prompt",
		}

		err := service.CreatePrompt(ctx, prompt)
		require.Error(t, err)
	})

	t.Run("create prompt without any text should fail", func(t *testing.T) {
		prompt := &entities.Prompt{
			Name:         "Empty Prompt",
			SystemPrompt: "",
			UserPrompt:   "",
		}

		err := service.CreatePrompt(ctx, prompt)
		require.Error(t, err)
	})

	t.Run("get non-existent prompt should fail", func(t *testing.T) {
		_, err := service.GetPrompt(ctx, 999999)
		require.Error(t, err)
	})
}

func TestPromptService_ListPrompts(t *testing.T) {
	service, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()

	t.Run("list empty prompts", func(t *testing.T) {
		prompts, err := service.ListPrompts(ctx)
		require.NoError(t, err)
		require.Equal(t, 0, len(prompts))
	})

	t.Run("list multiple prompts", func(t *testing.T) {
		for i := 1; i <= 3; i++ {
			prompt := &entities.Prompt{
				Name:         "List Test Prompt " + string(rune('0'+i)),
				SystemPrompt: "System",
				UserPrompt:   "User with {{test}}",
			}

			err := service.CreatePrompt(ctx, prompt)
			require.NoError(t, err)
		}

		prompts, err := service.ListPrompts(ctx)
		require.NoError(t, err)
		require.Equal(t, 3, len(prompts))
	})
}

func TestPromptService_UpdatePrompt(t *testing.T) {
	service, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()

	t.Run("update prompt successfully", func(t *testing.T) {
		prompt := &entities.Prompt{
			Name:         "Original Prompt",
			SystemPrompt: "Original system",
			UserPrompt:   "Original user with {{old}}",
		}

		err := service.CreatePrompt(ctx, prompt)
		require.NoError(t, err)

		prompts, err := service.ListPrompts(ctx)
		require.NoError(t, err)
		require.Greater(t, len(prompts), 0)

		updatedPrompt := prompts[0]
		updatedPrompt.Name = "Updated Prompt"
		updatedPrompt.UserPrompt = "Updated user with {{new}}"

		err = service.UpdatePrompt(ctx, updatedPrompt)
		require.NoError(t, err)

		retrievedPrompt, err := service.GetPrompt(ctx, updatedPrompt.ID)
		require.NoError(t, err)
		require.Equal(t, "Updated Prompt", retrievedPrompt.Name)
		require.Contains(t, retrievedPrompt.Placeholders, "new")
	})
}

func TestPromptService_DeletePrompt(t *testing.T) {
	service, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()

	t.Run("delete prompt successfully", func(t *testing.T) {
		prompt := &entities.Prompt{
			Name:         "Delete Test Prompt",
			SystemPrompt: "System",
			UserPrompt:   "User",
		}

		err := service.CreatePrompt(ctx, prompt)
		require.NoError(t, err)

		prompts, err := service.ListPrompts(ctx)
		require.NoError(t, err)
		require.Greater(t, len(prompts), 0)

		promptID := prompts[0].ID

		err = service.DeletePrompt(ctx, promptID)
		require.NoError(t, err)

		_, err = service.GetPrompt(ctx, promptID)
		require.Error(t, err)
	})
}

func TestPromptService_RenderPrompt(t *testing.T) {
	service, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()

	t.Run("render prompt with all placeholders provided", func(t *testing.T) {
		prompt := &entities.Prompt{
			Name:         "Render Test Prompt",
			SystemPrompt: "You are an expert on {{topic}}.",
			UserPrompt:   "Write an article about {{title}} with {{words}} words for category {{category}}.",
		}

		err := service.CreatePrompt(ctx, prompt)
		require.NoError(t, err)

		prompts, err := service.ListPrompts(ctx)
		require.NoError(t, err)
		require.Greater(t, len(prompts), 0)

		promptID := prompts[len(prompts)-1].ID

		placeholders := map[string]string{
			"topic":    "technology",
			"title":    "AI Revolution",
			"words":    "1000",
			"category": "Tech News",
		}

		system, user, err := service.RenderPrompt(ctx, promptID, placeholders)
		require.NoError(t, err)
		require.Equal(t, "You are an expert on technology.", system)
		require.Equal(t, "Write an article about AI Revolution with 1000 words for category Tech News.", user)
	})

	t.Run("render prompt with missing placeholders should fail", func(t *testing.T) {
		prompt := &entities.Prompt{
			Name:         "Missing Placeholder Test",
			SystemPrompt: "System",
			UserPrompt:   "Generate text on {{title}} with {{words}} words.",
		}

		err := service.CreatePrompt(ctx, prompt)
		require.NoError(t, err)

		prompts, err := service.ListPrompts(ctx)
		require.NoError(t, err)
		require.Greater(t, len(prompts), 0)

		promptID := prompts[len(prompts)-1].ID

		placeholders := map[string]string{
			"title": "Test Title",
			// "words" is missing
		}

		_, _, err = service.RenderPrompt(ctx, promptID, placeholders)
		require.Error(t, err)
		require.Contains(t, err.Error(), "missing required placeholders")
	})

	t.Run("render prompt without placeholders", func(t *testing.T) {
		prompt := &entities.Prompt{
			Name:         "No Placeholder Test",
			SystemPrompt: "You are a helpful assistant.",
			UserPrompt:   "Generate creative content.",
		}

		err := service.CreatePrompt(ctx, prompt)
		require.NoError(t, err)

		prompts, err := service.ListPrompts(ctx)
		require.NoError(t, err)
		require.Greater(t, len(prompts), 0)

		promptID := prompts[len(prompts)-1].ID

		system, user, err := service.RenderPrompt(ctx, promptID, nil)
		require.NoError(t, err)
		require.Equal(t, "You are a helpful assistant.", system)
		require.Equal(t, "Generate creative content.", user)
	})

	t.Run("render prompt with extra placeholders", func(t *testing.T) {
		prompt := &entities.Prompt{
			Name:         "Extra Placeholder Test",
			SystemPrompt: "System",
			UserPrompt:   "Generate text on {{title}}.",
		}

		err := service.CreatePrompt(ctx, prompt)
		require.NoError(t, err)

		prompts, err := service.ListPrompts(ctx)
		require.NoError(t, err)
		require.Greater(t, len(prompts), 0)

		promptID := prompts[len(prompts)-1].ID

		placeholders := map[string]string{
			"title": "Test Title",
			"extra": "This is extra",
		}

		system, user, err := service.RenderPrompt(ctx, promptID, placeholders)

		require.NoError(t, err)
		require.Equal(t, "System", system)
		require.Equal(t, "Generate text on Test Title.", user)
	})

	t.Run("render non-existent prompt should fail", func(t *testing.T) {
		_, _, err := service.RenderPrompt(ctx, 999999, nil)
		require.Error(t, err)
	})
}

func TestPromptService_PlaceholderExtraction(t *testing.T) {
	service, cleanup := setupTestService(t)
	defer cleanup()

	ctx := context.Background()

	t.Run("extract multiple placeholders", func(t *testing.T) {
		prompt := &entities.Prompt{
			Name:         "Multi Placeholder Test",
			SystemPrompt: "System with {{a}} and {{b}}.",
			UserPrompt:   "User with {{c}}, {{d}}, and {{a}} again.",
		}

		err := service.CreatePrompt(ctx, prompt)
		require.NoError(t, err)
		require.Equal(t, 4, len(prompt.Placeholders))
		require.Contains(t, prompt.Placeholders, "a")
		require.Contains(t, prompt.Placeholders, "b")
		require.Contains(t, prompt.Placeholders, "c")
		require.Contains(t, prompt.Placeholders, "d")
	})

	t.Run("extract no placeholders", func(t *testing.T) {
		prompt := &entities.Prompt{
			Name:         "No Placeholder Test",
			SystemPrompt: "Plain system text.",
			UserPrompt:   "Plain user text.",
		}

		err := service.CreatePrompt(ctx, prompt)
		require.NoError(t, err)
		require.Equal(t, 0, len(prompt.Placeholders))
	})

	t.Run("extract placeholders with underscores and numbers", func(t *testing.T) {
		prompt := &entities.Prompt{
			Name:         "Complex Placeholder Test",
			SystemPrompt: "System",
			UserPrompt:   "Text with {{place_holder1}} and {{var2}}.",
		}

		err := service.CreatePrompt(ctx, prompt)
		require.NoError(t, err)
		require.Equal(t, 2, len(prompt.Placeholders))
		require.Contains(t, prompt.Placeholders, "place_holder1")
		require.Contains(t, prompt.Placeholders, "var2")
	})
}
