package sqlite_test

import (
	"testing"
	"time"

	"github.com/davidmovas/postulator/internal/adapters/sqlite"
	"github.com/davidmovas/postulator/internal/adapters/sqlite/sqlitetest"
	"github.com/davidmovas/postulator/internal/domain/llm"
)

func TestModelProfileRepo(t *testing.T) {
	t.Parallel()

	repo := sqlite.NewModelProfileRepo(sqlitetest.Open(t))
	ctx := t.Context()
	at := time.Date(2026, 9, 18, 10, 0, 0, 0, time.UTC)

	profiles, err := repo.List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(profiles) != 0 {
		t.Fatalf("profiles = %v, want none", profiles)
	}

	writer := llm.ModelRef{Provider: "openai", Model: "gpt-5.6-terra"}
	if err = repo.Set(ctx, llm.RoleWriter, writer, at); err != nil {
		t.Fatalf("Set: %v", err)
	}
	if err = repo.Set(ctx, llm.RoleJudge, llm.ModelRef{Provider: "anthropic", Model: "claude-haiku-4-5"}, at); err != nil {
		t.Fatalf("Set judge: %v", err)
	}

	replacement := llm.ModelRef{Provider: "anthropic", Model: "claude-sonnet-5"}
	if err = repo.Set(ctx, llm.RoleWriter, replacement, at.Add(time.Hour)); err != nil {
		t.Fatalf("Set again: %v", err)
	}

	profiles, err = repo.List(ctx)
	if err != nil {
		t.Fatalf("List after: %v", err)
	}
	if len(profiles) != 2 || profiles[llm.RoleWriter] != replacement {
		t.Fatalf("profiles = %v, want the writer replaced", profiles)
	}

	if profiles[llm.RoleJudge].Model != "claude-haiku-4-5" {
		t.Errorf("profiles = %v, want the judge kept", profiles)
	}
}
