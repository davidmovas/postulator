package tools_test

import (
	"encoding/json"
	"testing"

	"github.com/davidmovas/postulator/internal/adapters/sqlite"
	"github.com/davidmovas/postulator/internal/application/templates"
	"github.com/davidmovas/postulator/internal/application/tools"
	domainagent "github.com/davidmovas/postulator/internal/domain/agent"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

func TestPagesGetAnswersThePageModelWithTheTemplateItsRunWouldUse(t *testing.T) {
	t.Parallel()

	registry, binding, seeded := wired(t)
	editable(t, seeded)
	repo := sqlite.NewPageRepo(seeded.store)
	page, err := repo.Get(t.Context(), seeded.page)
	if err != nil {
		t.Fatalf("read the page: %v", err)
	}
	page.TemplateID = &seeded.template
	if err = repo.Update(t.Context(), page); err != nil {
		t.Fatalf("give the page its template: %v", err)
	}
	binding.Mode = domainagent.ModeConfirm

	out, err := registry.Call(t.Context(), binding, "pages_get", json.RawMessage(`{"path":"/coffee/"}`))
	got, ok := out.(tools.PagesGetResponse)
	if err != nil || !ok {
		t.Fatalf("pages_get = %+v, %v, want the page's model and no card", out, err)
	}
	if got.Page.Page.PageID != seeded.page || got.Page.Body.Blocks != 2 || got.Page.Identity.WPID == nil {
		t.Errorf("the model = %+v, want the page on the site with its two blocks", got.Page)
	}
	if got.Template == nil || got.Template.ID != seeded.template || got.Template.Source != templates.SourcePage ||
		got.Template.Name == "" || len(got.Template.Headings) == 0 {
		t.Errorf("the template = %+v, want the page's own with its name, source and headings", got.Template)
	}
	if _, err = registry.Call(t.Context(), binding, "pages_get", json.RawMessage(`{"pageId":"missing"}`)); !errors.IsCode(err, errors.NotFound) {
		t.Errorf("a page that is not there = %v, want NotFound", err)
	}
	if _, known := registry.Lookup("templates_resolve_for_page"); known {
		t.Error("templates_resolve_for_page is still offered; pages_get answers which template applies")
	}
}
