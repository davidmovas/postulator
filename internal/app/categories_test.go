package app_test

import (
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	"github.com/davidmovas/postulator/internal/adapters/sqlite"
	"github.com/davidmovas/postulator/internal/app"
	"github.com/davidmovas/postulator/internal/application/graph"
	"github.com/davidmovas/postulator/internal/application/pages"
	"github.com/davidmovas/postulator/internal/application/sites"
	graphdomain "github.com/davidmovas/postulator/internal/domain/graph"
	"github.com/davidmovas/postulator/internal/kernel/dto"
)

func TestTheGraphAndThePagesNameTheCategoryTheSiteHas(t *testing.T) {
	t.Parallel()

	home := t.TempDir()
	core := openCore(t, app.Config{DatabasePath: filepath.Join(home, "postulator.db"), KeyDir: home})
	t.Cleanup(func() {
		if err := core.Close(); err != nil {
			t.Errorf("Close: %v", err)
		}
	})

	owner, err := core.Sites.Create(t.Context(), sites.CreateRequest{
		Name: "Peptides", BaseURL: "https://peptides.example", Username: "editor", Password: "hunter2",
	})
	if err != nil {
		t.Fatalf("create the site: %v", err)
	}
	created, err := core.Graph.CreateEntity(t.Context(), graph.CreateEntityRequest{
		SiteID: owner.Site.ID, Name: "Healing", Kind: "category", Keywords: []dto.Keyword{{Text: "healing peptides"}},
	})
	if err != nil {
		t.Fatalf("create the entity: %v", err)
	}
	if _, err = core.Graph.UpdateEntity(t.Context(), graph.UpdateEntityRequest{ID: created.Entity.ID, SiteCategory: new(true)}); err != nil {
		t.Fatalf("make the entity a category: %v", err)
	}
	page, err := core.Pages.Create(t.Context(), pages.CreateRequest{
		SiteID: owner.Site.ID, Path: "/healing/", Title: "Healing", WPType: "post", EntityID: &created.Entity.ID,
	})
	if err != nil {
		t.Fatalf("create the post: %v", err)
	}
	if err = sqlite.NewTermRepo(core.Store).Upsert(t.Context(), graphdomain.Term{
		EntityID: created.Entity.ID, SiteID: owner.Site.ID, Taxonomy: graphdomain.TaxonomyCategory, TermID: 14,
		Name: "Healing", SeenAt: time.Date(2026, time.October, 3, 9, 0, 0, 0, time.UTC),
	}); err != nil {
		t.Fatalf("store the term: %v", err)
	}

	entity, err := core.Graph.GetEntity(t.Context(), graph.GetEntityRequest{ID: created.Entity.ID})
	if err != nil {
		t.Fatalf("GetEntity: %v", err)
	}
	post, err := core.Pages.Get(t.Context(), pages.GetRequest{ID: page.Page.ID})
	if err != nil {
		t.Fatalf("Get the post: %v", err)
	}

	want := `[{"entityId":"` + created.Entity.ID + `","name":"Healing","termId":14}]`
	for _, got := range []struct {
		name       string
		categories []dto.Category
	}{
		{name: "the entity", categories: entity.Entity.Categories},
		{name: "the post", categories: post.Page.Categories},
	} {
		encoded, encodeErr := json.Marshal(got.categories)
		if encodeErr != nil {
			t.Fatalf("encode the categories of %s: %v", got.name, encodeErr)
		}
		if string(encoded) != want {
			t.Errorf("%s is filed in %s, want %s", got.name, encoded, want)
		}
	}
}
