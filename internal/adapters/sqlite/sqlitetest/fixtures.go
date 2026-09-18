package sqlitetest

import (
	"testing"
	"time"

	"github.com/davidmovas/postulator/internal/adapters/sqlite"
	"github.com/davidmovas/postulator/internal/domain/graph"
	"github.com/davidmovas/postulator/internal/domain/llm"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/site"
	"github.com/davidmovas/postulator/internal/domain/template"
	"github.com/davidmovas/postulator/internal/kernel/id"
)

var Stamp = time.Date(2026, time.September, 18, 9, 0, 0, 0, time.UTC)

func Site(t testing.TB, store *sqlite.Store, name string) site.Site {
	t.Helper()

	record := site.Site{
		ID:        id.New(),
		Name:      name,
		BaseURL:   "https://" + name + ".example.com",
		Username:  "editor",
		Status:    site.StatusActive,
		Plugin:    site.PluginState{Capabilities: []string{}},
		Defaults:  site.Defaults{ModelProfiles: map[llm.Role]llm.ModelRef{}},
		CreatedAt: Stamp,
		UpdatedAt: Stamp,
	}
	record.SecretRef = site.SecretRef(record.ID)
	if err := sqlite.NewSiteRepo(store).Insert(t.Context(), record); err != nil {
		t.Fatalf("insert the site fixture: %v", err)
	}
	return record
}

func Entity(t testing.TB, store *sqlite.Store, siteID, name string) graph.Entity {
	t.Helper()

	record := graph.Entity{
		ID: id.New(), SiteID: siteID, Name: name, Kind: graph.KindTopic, PrimaryKeyword: name,
		SecondaryKeywords: []string{}, Anchors: []graph.Anchor{{Text: name, Source: graph.AnchorUser, Weight: 1}},
		Source: graph.SourceUser, CreatedAt: Stamp, UpdatedAt: Stamp,
	}
	if err := sqlite.NewEntityRepo(store).Insert(t.Context(), record); err != nil {
		t.Fatalf("insert the entity fixture: %v", err)
	}
	return record
}

func Page(t testing.TB, store *sqlite.Store, siteID, path string) pagemap.Page {
	t.Helper()

	record := pagemap.Page{
		ID: id.New(), SiteID: siteID, Path: path, Slug: pagemap.Slug(path), WPType: pagemap.WPPage,
		Title: path, H1: path, Status: pagemap.StatusPlanned, CreatedAt: Stamp, UpdatedAt: Stamp,
	}
	if err := sqlite.NewPageRepo(store).Insert(t.Context(), record); err != nil {
		t.Fatalf("insert the page fixture: %v", err)
	}
	return record
}

func Template(t testing.TB, store *sqlite.Store, name string) template.Template {
	t.Helper()

	seed := template.Seed()[3]
	record := template.Template{ID: id.New(), Scope: template.ScopeGlobal, Name: name, PageKind: seed.PageKind, Version: 1, Spec: seed.Spec, CreatedAt: Stamp, UpdatedAt: Stamp}
	if err := sqlite.NewTemplateRepo(store).Insert(t.Context(), record); err != nil {
		t.Fatalf("insert the template fixture: %v", err)
	}
	return record
}
