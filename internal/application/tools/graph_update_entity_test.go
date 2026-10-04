package tools_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/davidmovas/postulator/internal/adapters/sqlite"
	domainagent "github.com/davidmovas/postulator/internal/domain/agent"
	"github.com/davidmovas/postulator/internal/domain/category"
	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/kernel/id"
)

func fileTheSeededPage(t *testing.T, seeded fixture) category.Category {
	t.Helper()

	coffee, err := category.New(category.Category{ID: id.New(), SiteID: seeded.site, Name: "Coffee", CreatedAt: stamp, UpdatedAt: stamp})
	if err != nil {
		t.Fatalf("category: %v", err)
	}
	if err = sqlite.NewCategoryRepo(seeded.store).Insert(t.Context(), coffee); err != nil {
		t.Fatalf("insert the category: %v", err)
	}
	pageRepo := sqlite.NewPageRepo(seeded.store)
	page, err := pageRepo.Get(t.Context(), seeded.page)
	if err != nil {
		t.Fatalf("Get the page: %v", err)
	}
	page.EntityID, page.CategoryID = &seeded.entity, coffee.ID
	if err = pageRepo.Update(t.Context(), page); err != nil {
		t.Fatalf("file the page: %v", err)
	}
	return coffee
}

func TestTheAgentsEntityUpdateAnswersWithTheCategoriesOfItsPage(t *testing.T) {
	t.Parallel()

	registry, binding, seeded := wired(t)
	binding.Mode = domainagent.ModeAutonomous
	coffee := fileTheSeededPage(t, seeded)

	out, err := registry.Call(t.Context(), binding, "graph_update_entity", json.RawMessage(`{"id":"`+seeded.entity+`","intent":"Find a coffee"}`))
	if err != nil {
		t.Fatalf("graph_update_entity: %v", err)
	}
	encoded, err := json.Marshal(out)
	if err != nil {
		t.Fatalf("encode the answer: %v", err)
	}
	for _, want := range []string{`"intent":"Find a coffee"`, `"categories":[{"id":"` + coffee.ID + `","name":"Coffee"}]`} {
		if !strings.Contains(string(encoded), want) {
			t.Errorf("the answer lacks %s: %s", want, encoded)
		}
	}
	if strings.Contains(string(encoded), "siteCategory") {
		t.Errorf("the answer still says whether the entity is a WordPress category: %s", encoded)
	}

	_, err = registry.Call(t.Context(), binding, "graph_update_entity", json.RawMessage(`{"id":"`+seeded.entity+`","siteCategory":true}`))
	if !errors.IsCode(err, errors.Invalid) {
		t.Fatalf("an entity made a WordPress category answered %v, want INVALID: a category is its own record now", err)
	}
}
