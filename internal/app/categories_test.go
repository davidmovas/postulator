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
	"github.com/davidmovas/postulator/internal/domain/category"
	"github.com/davidmovas/postulator/internal/kernel/dto"
	"github.com/davidmovas/postulator/internal/kernel/id"
)

func storedCategory(t *testing.T, core *app.Core, siteID, name, parentID string) category.Category {
	t.Helper()
	at := time.Date(2026, time.October, 4, 9, 0, 0, 0, time.UTC)
	record, err := category.New(category.Category{ID: id.New(), SiteID: siteID, Name: name, ParentID: parentID, CreatedAt: at, UpdatedAt: at})
	if err != nil {
		t.Fatalf("category %s: %v", name, err)
	}
	if err = sqlite.NewCategoryRepo(core.Store).Insert(t.Context(), record); err != nil {
		t.Fatalf("store the category %s: %v", name, err)
	}
	return record
}

func encodedJSON(t *testing.T, v any) string {
	t.Helper()
	out, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	return string(out)
}

func TestTheGraphAndThePagesNameTheCategoryAPageIsFiledUnder(t *testing.T) {
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
	siteID := owner.Site.ID
	created, err := core.Graph.CreateEntity(t.Context(), graph.CreateEntityRequest{
		SiteID: siteID, Name: "Healing Peptides", Kind: "topic", Keywords: []dto.Keyword{{Text: "healing peptides"}},
	})
	if err != nil {
		t.Fatalf("create the entity: %v", err)
	}
	post, err := core.Pages.Create(t.Context(), pages.CreateRequest{
		SiteID: siteID, Path: "/healing/", Title: "Healing", WPType: "post", EntityID: &created.Entity.ID,
	})
	if err != nil {
		t.Fatalf("create the post: %v", err)
	}

	peptides := storedCategory(t, core, siteID, "Peptides", "")
	healing := storedCategory(t, core, siteID, "Healing", peptides.ID)
	pageRepo := sqlite.NewPageRepo(core.Store)
	filed, err := pageRepo.Get(t.Context(), post.Page.ID)
	if err != nil {
		t.Fatalf("read the post: %v", err)
	}
	filed.CategoryID = healing.ID
	if err = pageRepo.Update(t.Context(), filed); err != nil {
		t.Fatalf("file the post: %v", err)
	}
	if err = sqlite.NewCategoryTermRepo(core.Store).Upsert(t.Context(), category.Term{
		CategoryID: peptides.ID, SiteID: siteID, Taxonomy: category.TaxonomyCategory, TermID: 14,
		Name: "Peptides", SeenAt: time.Date(2026, time.October, 4, 9, 0, 0, 0, time.UTC),
	}); err != nil {
		t.Fatalf("store the term: %v", err)
	}

	entity, err := core.Graph.GetEntity(t.Context(), graph.GetEntityRequest{ID: created.Entity.ID})
	if err != nil {
		t.Fatalf("GetEntity: %v", err)
	}
	read, err := core.Pages.Get(t.Context(), pages.GetRequest{ID: post.Page.ID})
	if err != nil {
		t.Fatalf("Get the post: %v", err)
	}
	listed, err := core.Pages.List(t.Context(), pages.ListRequest{SiteID: siteID, CategoryID: peptides.ID})
	if err != nil {
		t.Fatalf("List under Peptides: %v", err)
	}
	tree, err := core.Pages.ListCategories(t.Context(), pages.ListCategoriesRequest{SiteID: siteID})
	if err != nil {
		t.Fatalf("ListCategories: %v", err)
	}

	chain := `[{"id":"` + peptides.ID + `","name":"Peptides","termId":14},{"id":"` + healing.ID + `","name":"Healing"}]`
	cases := []struct {
		name string
		got  any
		want string
	}{
		{name: "the entity", got: entity.Entity.Categories, want: chain},
		{name: "the post", got: read.Page.Categories, want: chain},
		{name: "the pages under the root category", got: len(listed.Items), want: "1"},
		{
			name: "the category tree", got: tree,
			want: `{"categories":[{"id":"` + peptides.ID + `","name":"Peptides","parentId":null,"pages":1,"termIds":{"category":14}},` +
				`{"id":"` + healing.ID + `","name":"Healing","parentId":"` + peptides.ID + `","pages":1,"termIds":{}}]}`,
		},
	}
	for _, tc := range cases {
		if got := encodedJSON(t, tc.got); got != tc.want {
			t.Errorf("%s = %s, want %s", tc.name, got, tc.want)
		}
	}
	if len(listed.Items) == 1 && listed.Items[0].ID != post.Page.ID {
		t.Errorf("the page under the root category is %s, want the post", listed.Items[0].Path)
	}
}
