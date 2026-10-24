//go:build !race

package scale_test

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"sync"
	"testing"
	"time"

	"go.uber.org/zap"

	"github.com/davidmovas/postulator/internal/adapters/llm/fake"
	"github.com/davidmovas/postulator/internal/adapters/sqlite"
	"github.com/davidmovas/postulator/internal/adapters/sqlite/sqlitetest"
	"github.com/davidmovas/postulator/internal/app"
	"github.com/davidmovas/postulator/internal/application/imports"
	"github.com/davidmovas/postulator/internal/application/observe"
	"github.com/davidmovas/postulator/internal/application/sites"
	"github.com/davidmovas/postulator/internal/domain/block"
	"github.com/davidmovas/postulator/internal/domain/field"
	"github.com/davidmovas/postulator/internal/domain/finding"
	"github.com/davidmovas/postulator/internal/domain/graph"
	"github.com/davidmovas/postulator/internal/domain/importmap"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/render"
	"github.com/davidmovas/postulator/internal/domain/site"
	"github.com/davidmovas/postulator/internal/kernel/id"
)

const (
	walkBatch = 25
	walkHours = 12
)

var (
	generatedAt = time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	modifiedAt  = generatedAt.Add(-time.Hour)
	observedAt  = generatedAt.Add(time.Hour)
)

type scaled struct {
	core    *app.Core
	counter *sqlitetest.Counter
	plan    *blueprint
	wp      *wordpress
	siteID  string
	blocks  int
	pageIDs map[string]string
	renders map[string]render.Extraction
	traced  map[string]struct{}
	path    string
	walks   int
	started time.Time
}

func (s *scaled) extract() error {
	install := pagemap.NewSite(baseURL)
	lookup := func(path string) (pagemap.WPType, bool) {
		target, known := s.plan.byPath[path]
		if !known {
			return "", false
		}
		return target.wpType, true
	}
	out := make([]render.Extraction, len(s.plan.specs))
	failed := make([]error, len(s.plan.specs))
	var wait sync.WaitGroup
	slots := make(chan struct{}, runtime.GOMAXPROCS(0))
	for i, held := range s.plan.specs {
		if held.kind == kindArchived {
			continue
		}
		wait.Go(func() {
			slots <- struct{}{}
			defer func() { <-slots }()
			raw := held.body(0)
			out[i], failed[i] = render.Extract(render.Input{
				Public: held.html(raw, 0), Body: block.Parse(raw), Site: install, Document: held.path, Lookup: lookup,
				Archive: held.wpType.Term() || held.kind == kindShop,
			})
		})
	}
	wait.Wait()
	s.renders = make(map[string]render.Extraction, len(out))
	for i, held := range s.plan.specs {
		if failed[i] != nil {
			return failed[i]
		}
		s.renders[held.path] = out[i]
	}
	return nil
}

func generate(t *testing.T, blocks int) *scaled {
	t.Helper()

	started := time.Now()
	home := t.TempDir()
	counter := sqlitetest.NewCounter()
	core, err := app.Open(t.Context(), app.Config{
		DatabasePath: filepath.Join(home, "postulator.db"), KeyDir: home, Provider: fake.New(),
		Open: func(cfg sqlite.Config) (*sqlite.Store, error) {
			cfg.Key = nil
			return counter.Open(cfg)
		},
	}, zap.NewNop())
	if err != nil {
		t.Fatalf("open the core: %v", err)
	}
	t.Cleanup(func() {
		if closeErr := core.Close(); closeErr != nil {
			t.Errorf("close the core: %v", closeErr)
		}
	})
	core.Scheduler.Stop()
	core.Engine.Stop()

	created, err := core.Sites.Create(t.Context(), sites.CreateRequest{
		Name: "Scale " + strconv.Itoa(blocks), BaseURL: baseURL, Username: "editor", Password: "abcd EFGH ijkl MNOP qrst UVWX",
	})
	if err != nil {
		t.Fatalf("create the site: %v", err)
	}
	s := &scaled{
		core: core, counter: counter, plan: plan(blocks), siteID: created.Site.ID, blocks: blocks, pageIDs: map[string]string{},
		started: started.UTC().Truncate(time.Second), traced: map[string]struct{}{}, path: filepath.Join(home, "postulator.db"),
	}
	s.wp = newWordPress(s.plan.specs, modifiedAt)
	opened := time.Now()
	if err := s.extract(); err != nil {
		t.Fatalf("extract the generated renders: %v", err)
	}
	if err := core.Store.Do(t.Context(), s.write); err != nil {
		t.Fatalf("write the generated site: %v", err)
	}
	written := time.Now()
	s.importRows(t)
	imported := time.Now()
	s.walk(t, nil)
	for _, sql := range counter.Distinct() {
		s.traced[sql] = struct{}{}
	}
	t.Logf("%d blocks, %d pages generated in %s (open %s, write %s, import %s, settle %s)", blocks, len(s.plan.specs)+2*blocks,
		time.Since(started).Round(time.Millisecond), opened.Sub(started).Round(time.Millisecond), written.Sub(opened).Round(time.Millisecond),
		imported.Sub(written).Round(time.Millisecond), time.Since(imported).Round(time.Millisecond))
	return s
}

func (s *scaled) write(ctx context.Context) error {
	store := s.core.Store
	entities, edges := sqlite.NewEntityRepo(store), sqlite.NewEdgeRepo(store)
	pages := sqlite.NewPageRepo(store)
	named := map[string]string{}
	for _, held := range s.plan.specs {
		if held.entity == "" {
			continue
		}
		entity := graph.Entity{
			ID: id.New(), SiteID: s.siteID, Name: held.entity, Definition: "The " + held.entity + " of the generated site.",
			Source: graph.SourceImport, CreatedAt: generatedAt, UpdatedAt: generatedAt,
		}
		if scope, scoped := named[held.scope]; scoped {
			entity.ScopeID = &scope
		}
		if err := entities.Insert(ctx, entity); err != nil {
			return err
		}
		named[held.entity] = entity.ID
		if entity.ScopeID != nil {
			if err := edges.Insert(ctx, edge(s.siteID, entity.ID, *entity.ScopeID, graph.EdgeParent)); err != nil {
				return err
			}
		}
		if other, related := named[held.related]; related {
			if err := edges.Insert(ctx, edge(s.siteID, entity.ID, other, graph.EdgeRelated)); err != nil {
				return err
			}
		}
	}
	for _, held := range s.plan.specs {
		s.pageIDs[held.path] = id.New()
	}
	for _, held := range s.plan.specs {
		if err := pages.Insert(ctx, s.page(held, named)); err != nil {
			return err
		}
	}
	for _, held := range s.plan.specs {
		if entityID, mapped := named[held.entity]; mapped {
			pageID := s.pageIDs[held.path]
			if err := entities.SetCanonicalPage(ctx, entityID, &pageID, generatedAt); err != nil {
				return err
			}
		}
	}
	for _, held := range s.plan.specs {
		if err := s.observed(ctx, held); err != nil {
			return err
		}
	}
	return sqlite.NewSiteRepo(store).UpdateObservation(ctx, s.siteID, s.wp.technical, site.ObservationState{
		LastStartedAt: &observedAt, LastFinishedAt: ref(observedAt.Add(time.Minute)), LastFullWalkAt: &observedAt,
		LastModifiedSeen: &generatedAt, SinceFullWalk: 1,
	}, observedAt)
}

func edge(siteID, from, to string, kind graph.EdgeKind) graph.Edge {
	built, err := graph.NewEdge(graph.Edge{
		ID: id.New(), SiteID: siteID, FromEntityID: from, ToEntityID: to, Kind: kind, Weight: 0.5,
		Source: graph.SourceImport, Status: graph.StatusApproved, CreatedAt: generatedAt,
	})
	if err != nil {
		panic(err)
	}
	return built
}

func ref[T any](value T) *T {
	return &value
}

func (s *scaled) page(held *spec, named map[string]string) pagemap.Page {
	page := pagemap.Page{
		ID: s.pageIDs[held.path], SiteID: s.siteID, Path: held.path, Slug: pagemap.Slug(held.path), WPType: held.wpType,
		Title: held.title, H1: held.title, Status: pagemap.StatusPublished, CreatedAt: generatedAt, UpdatedAt: observedAt,
		ParentReason: pagemap.ParentNone,
	}
	if held.kind == kindArchived {
		page.Status = pagemap.StatusArchived
	}
	if held.kind != kindArchived {
		wpID := held.wpID + held.termID
		page.WPID = &wpID
		page.WPModifiedAt, page.LastSyncedAt = &modifiedAt, &observedAt
	}
	if parentID, known := s.pageIDs[held.parent]; known && held.parent != "" {
		page.ParentPageID = &parentID
		page.ParentReason, page.ParentMethod = pagemap.ParentJSONLD, pagemap.MethodHref
	}
	if entityID, mapped := named[held.entity]; mapped {
		page.EntityID = &entityID
	}
	return page
}

func (s *scaled) observed(ctx context.Context, held *spec) error {
	if held.kind == kindArchived {
		return nil
	}
	store := s.core.Store
	pageID := s.pageIDs[held.path]
	identity := pagemap.Identity{
		PageID: pageID, URL: baseURL + held.path, Status: statusPublish, WPType: held.wpType, ModifiedGMT: &modifiedAt, ObservedAt: observedAt,
	}
	if held.parentWP != 0 {
		identity.WPParentID = ref(held.parentWP)
	}
	identities := sqlite.NewPageIdentityRepo(store)
	if err := identities.Put(ctx, identity); err != nil {
		return err
	}
	if len(held.terms) > 0 {
		if err := identities.ReplaceTerms(ctx, pageID, held.terms); err != nil {
			return err
		}
	}
	if err := sqlite.NewPageFieldRepo(store).Observe(ctx, pageID, s.values(held, pageID)); err != nil {
		return err
	}
	if err := s.findings(ctx, held, pageID); err != nil {
		return err
	}
	if !held.wpType.Term() {
		if err := s.bodies(ctx, held, pageID, held.body(0)); err != nil {
			return err
		}
	}
	return s.rendered(ctx, held, pageID)
}

func (s *scaled) values(held *spec, pageID string) []field.Value {
	item := (&entry{spec: held, content: held.body(0)}).item()
	values := []field.Value{field.NewValue(pageID, field.Title, item.Title, observedAt)}
	if held.wpType.Term() {
		return values
	}
	values = append(values,
		field.NewValue(pageID, field.Excerpt, item.Excerpt, observedAt),
		field.NewValue(pageID, field.H1, item.H1, observedAt),
		field.NewValue(pageID, field.SEOTitle, item.Meta.Title, observedAt),
		field.NewValue(pageID, field.SEODescription, item.Meta.Description, observedAt),
		field.NewValue(pageID, field.SEOCanonical, "", observedAt),
	)
	if held.edited {
		values[4].Base, values[4].BaseHash = "An older description.", field.Hash("An older description.")
	}
	if held.wpType != pagemap.WPProduct {
		return values
	}
	product := (&entry{spec: held, content: held.body(0), variations: s.wp.byURL[baseURL+held.path].variations}).product()
	values = append(values,
		field.NewValue(pageID, field.ProductDescription, product.Description, observedAt),
		field.NewValue(pageID, field.ProductShortDescription, product.ShortDescription, observedAt),
		field.NewValue(pageID, field.Price, product.Price, observedAt),
		field.NewValue(pageID, field.RegularPrice, product.RegularPrice, observedAt),
		field.NewValue(pageID, field.SalePrice, product.SalePrice, observedAt),
		field.NewValue(pageID, field.SKU, product.SKU, observedAt),
		field.NewValue(pageID, field.StockStatus, product.StockStatus, observedAt),
		field.NewValue(pageID, field.StockQuantity, "12", observedAt),
		field.NewValue(pageID, field.ManageStock, product.ManageStock, observedAt),
		field.NewValue(pageID, field.Attributes, string(product.Attributes), observedAt),
	)
	variations := s.wp.byURL[baseURL+held.path].variations
	for i := range variations {
		variation := &variations[i]
		instance := strconv.FormatInt(variation.ID, 10)
		values = append(values,
			field.NewValue(pageID, field.Instance(field.VariationDescription, instance), variation.Description, observedAt),
			field.NewValue(pageID, field.Instance(field.Variation, instance), `{"sku":"`+variation.SKU+`"}`, observedAt),
		)
	}
	return values
}

func (s *scaled) findings(ctx context.Context, held *spec, pageID string) error {
	repo := sqlite.NewFindingRepo(s.core.Store)
	if held.edited {
		subject := finding.SubjectPageField(pageID, string(field.SEODescription))
		if err := repo.Upsert(ctx, s.finding(finding.FieldChangedOnSite, pageID, subject, `{"key":"seo_description"}`)); err != nil {
			return err
		}
	}
	if held.drifted {
		if err := repo.Upsert(ctx, s.finding(finding.BodyChangedOnSite, pageID, finding.SubjectPage(pageID), `{"baseHash":"older"}`)); err != nil {
			return err
		}
	}
	return nil
}

func (s *scaled) finding(code finding.Code, pageID, subjectKey, subject string) finding.Finding {
	record, err := finding.New(finding.Finding{
		ID: id.New(), SiteID: s.siteID, PageID: &pageID, Code: code, SubjectKey: subjectKey, Subject: []byte(subject),
		CreatedAt: observedAt, UpdatedAt: observedAt,
	})
	if err != nil {
		panic(err)
	}
	return record
}

func (s *scaled) bodies(ctx context.Context, held *spec, pageID, raw string) error {
	repo := sqlite.NewPageBodyRepo(s.core.Store)
	parsed := block.Parse(raw)
	base := block.Snapshot{PageID: pageID, Side: block.Current, Raw: raw, Hash: block.BodyHash(raw), Kind: parsed.Kind, ObservedAt: observedAt}
	if held.drifted {
		older := "<!-- wp:paragraph -->\n<p>An older body of " + held.title + ".</p>\n<!-- /wp:paragraph -->"
		if err := repo.Observe(ctx, block.Snapshot{
			PageID: pageID, Side: block.Current, Raw: older, Hash: block.BodyHash(older), Kind: block.Parse(older).Kind, ObservedAt: observedAt,
		}); err != nil {
			return err
		}
	}
	if err := repo.Observe(ctx, base); err != nil {
		return err
	}
	written := []string{}
	if held.written {
		written = append(written, parsed.Blocks[0].Hash)
		if err := repo.ReplaceWritten(ctx, pageID, written, id.New(), observedAt); err != nil {
			return err
		}
	}
	stored := func(path string) (pagemap.Page, bool) {
		target, known := s.pageIDs[path]
		return pagemap.Page{ID: target, Path: path}, known
	}
	page := pagemap.Page{ID: pageID, SiteID: s.siteID, Path: held.path, WPType: held.wpType}
	links := observe.ContentLinks(page, pagemap.NewSite(baseURL), block.LinksOf(raw, written), nil, stored, observedAt)
	return sqlite.NewPageLinkRepo(s.core.Store).ReplaceForPage(ctx, pageID, pagemap.SourceRaw, links)
}

func (s *scaled) rendered(ctx context.Context, held *spec, pageID string) error {
	extracted := s.renders[held.path]
	if err := sqlite.NewPageRenderRepo(s.core.Store).Put(ctx, render.Record{PageID: pageID, Extraction: extracted, ExtractedAt: observedAt}); err != nil {
		return err
	}
	links := make([]pagemap.PageLink, 0, len(extracted.Links))
	for _, found := range extracted.Links {
		if found.Kind != pagemap.LinkPath || found.Path == "" || found.Region == render.RegionContent {
			continue
		}
		link := pagemap.PageLink{
			ID: id.New(), SiteID: s.siteID, FromPageID: pageID, ToURL: found.Path, AnchorText: found.Anchor,
			Type: found.Type, Region: string(found.Region), Source: pagemap.SourceRender, ObservedAt: observedAt,
		}
		if target, known := s.pageIDs[found.Path]; known {
			link.ToPageID = &target
		}
		if built, buildErr := pagemap.NewPageLink(link); buildErr == nil {
			links = append(links, built)
		}
	}
	return sqlite.NewPageLinkRepo(s.core.Store).ReplaceForPage(ctx, pageID, pagemap.SourceRender, links)
}

func (s *scaled) importRows(t *testing.T) {
	t.Helper()

	path := filepath.Join(t.TempDir(), "planned.csv")
	if err := os.WriteFile(path, []byte(plannedRows(s.blocks)), 0o600); err != nil {
		t.Fatalf("write the workbook: %v", err)
	}
	mapping := imports.Mapping{SiteID: s.siteID, Name: "planned", Columns: map[string]string{
		string(importmap.FieldPath): "path", string(importmap.FieldTitle): "title", string(importmap.FieldEntity): "entity",
	}}
	applied, err := s.core.Imports.Apply(t.Context(), imports.ApplyRequest{SiteID: s.siteID, Path: path, Mapping: mapping})
	if err != nil {
		t.Fatalf("import the workbook: %v", err)
	}
	if applied.Counts.PagesCreated != 2*s.blocks {
		t.Fatalf("the import planned %d pages, want %d: %+v", applied.Counts.PagesCreated, 2*s.blocks, applied.Counts)
	}
	listed, err := sqlite.NewPageRepo(s.core.Store).ListBySite(t.Context(), s.siteID)
	if err != nil {
		t.Fatalf("list the pages: %v", err)
	}
	for i := range listed {
		s.pageIDs[listed[i].Path] = listed[i].ID
	}
}

func (s *scaled) walk(t *testing.T, touched []string) observe.State {
	t.Helper()

	s.walks++
	if touched != nil {
		s.wp.touch(touched, s.started.Add(time.Duration(s.walks-walkHours)*time.Hour))
	}
	state, err := s.core.Observe.Run(t.Context(), s.siteID, s.wp.readers(), observe.Options{}, walkBatch)
	if err != nil {
		t.Fatalf("walk %d: %v", s.walks, err)
	}
	if state.Full {
		t.Fatalf("walk %d covered the whole site, want an incremental one", s.walks)
	}
	return state
}
