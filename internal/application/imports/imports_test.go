package imports_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/davidmovas/postulator/internal/adapters/importer"
	"github.com/davidmovas/postulator/internal/adapters/sqlite"
	"github.com/davidmovas/postulator/internal/adapters/sqlite/sqlitetest"
	"github.com/davidmovas/postulator/internal/application/applicationtest"
	"github.com/davidmovas/postulator/internal/application/imports"
	"github.com/davidmovas/postulator/internal/domain/graph"
	"github.com/davidmovas/postulator/internal/domain/importmap"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/kernel/clock"
)

type harness struct {
	service  *imports.Service
	store    *sqlite.Store
	recorder *applicationtest.Recorder
	siteID   string
	dir      string
}

func newHarness(t *testing.T) harness {
	t.Helper()
	return newHarnessWithRows(t, 0)
}

func newHarnessWithRows(t *testing.T, maxRows int) harness {
	t.Helper()

	store := sqlitetest.Open(t)
	owner := sqlitetest.Site(t, store, "shop")
	recorder := &applicationtest.Recorder{}

	return harness{
		service: imports.New(imports.Deps{
			Tables:     importer.New(),
			Entities:   sqlite.NewEntityRepo(store),
			Edges:      sqlite.NewEdgeRepo(store),
			Pages:      sqlite.NewPageRepo(store),
			Templates:  sqlite.NewTemplateRepo(store),
			Mappings:   sqlite.NewImportMappingRepo(store),
			Sites:      sqlite.NewSiteRepo(store),
			UnitOfWork: store,
			Publisher:  recorder,
			Clock:      clock.NewFake(time.Date(2026, time.September, 18, 9, 0, 0, 0, time.UTC)),
			MaxRows:    maxRows,
		}),
		store:    store,
		recorder: recorder,
		siteID:   owner.ID,
		dir:      t.TempDir(),
	}
}

func (h harness) file(t *testing.T, name, body string) string {
	t.Helper()

	path := filepath.Join(h.dir, name)
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
	return path
}

func (h harness) mapping(columns map[string]string) imports.Mapping {
	return imports.Mapping{SiteID: h.siteID, Name: "sheet", Columns: columns}
}

func (h harness) preview(t *testing.T, path string, mapping imports.Mapping) imports.PreviewReport {
	t.Helper()

	got, err := h.service.Preview(t.Context(), imports.PreviewRequest{SiteID: h.siteID, Path: path, Mapping: mapping})
	if err != nil {
		t.Fatalf("Preview: %v", err)
	}
	return got.Report
}

func (h harness) apply(t *testing.T, path string, mapping imports.Mapping) imports.ApplyResponse {
	t.Helper()

	got, err := h.service.Apply(t.Context(), imports.ApplyRequest{SiteID: h.siteID, Path: path, Mapping: mapping})
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	return got
}

func (h harness) entities(t *testing.T) []graph.Entity {
	t.Helper()

	listed, err := sqlite.NewEntityRepo(h.store).ListBySite(t.Context(), h.siteID)
	if err != nil {
		t.Fatalf("list the entities: %v", err)
	}
	return listed
}

func (h harness) edges(t *testing.T) []graph.Edge {
	t.Helper()

	listed, err := sqlite.NewEdgeRepo(h.store).ListBySite(t.Context(), h.siteID)
	if err != nil {
		t.Fatalf("list the edges: %v", err)
	}
	return listed
}

func (h harness) pages(t *testing.T) []pagemap.Page {
	t.Helper()

	listed, err := sqlite.NewPageRepo(h.store).ListBySite(t.Context(), h.siteID)
	if err != nil {
		t.Fatalf("list the pages: %v", err)
	}
	return listed
}

func findings(list []imports.Finding, code imports.FindingCode) []imports.Finding {
	out := make([]imports.Finding, 0, len(list))
	for i := range list {
		if list[i].Code == string(code) {
			out = append(out, list[i])
		}
	}
	return out
}

func page(report imports.PreviewReport, path string) (imports.PreviewPage, bool) {
	for i := range report.Pages {
		if report.Pages[i].Path == path {
			return report.Pages[i], true
		}
	}
	return imports.PreviewPage{}, false
}

func entity(report imports.PreviewReport, name string) (imports.PreviewEntity, bool) {
	for i := range report.Entities {
		if report.Entities[i].Name == name {
			return report.Entities[i], true
		}
	}
	return imports.PreviewEntity{}, false
}

const graphSheet = "path,title,entity,parent,related,keywords,anchors\n" +
	"/,Home,Shop,,,shop,shop here\n" +
	"/hosting/,Hosting,Hosting,Shop,Domains,hosting;servers,hosting plans\n" +
	"/domains/,Domains,Domains,Shop,,domains,buy a domain\n"

func graphMapping(h harness) imports.Mapping {
	mapping := h.mapping(map[string]string{
		string(importmap.FieldPath):     "path",
		string(importmap.FieldTitle):    "title",
		string(importmap.FieldEntity):   "entity",
		string(importmap.FieldKeywords): "keywords",
		string(importmap.FieldAnchors):  "anchors",
	})
	mapping.Columns[string(importmap.FieldParentEntity)] = "parent"
	mapping.Columns[string(importmap.FieldRelated)] = "related"
	mapping.Options.KeywordSeparator = ";"
	return mapping
}
