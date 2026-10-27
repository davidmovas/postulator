package graph_test

import (
	"context"
	"maps"
	"slices"
	"testing"

	"github.com/davidmovas/postulator/internal/adapters/sqlite"
	"github.com/davidmovas/postulator/internal/adapters/sqlite/sqlitetest"
	"github.com/davidmovas/postulator/internal/application/applicationtest"
	appgraph "github.com/davidmovas/postulator/internal/application/graph"
	"github.com/davidmovas/postulator/internal/domain/contextpack"
	"github.com/davidmovas/postulator/internal/domain/field"
	"github.com/davidmovas/postulator/internal/domain/finding"
	graphdomain "github.com/davidmovas/postulator/internal/domain/graph"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/pagetype"
	"github.com/davidmovas/postulator/internal/kernel/clock"
	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/kernel/id"
)

type storeMap struct {
	store *sqlite.Store
}

func (m storeMap) Map(ctx context.Context, siteID string) (contextpack.Map, error) {
	pages, err := sqlite.NewPageRepo(m.store).ListBySite(ctx, siteID)
	if err != nil {
		return contextpack.Map{}, err
	}
	entities, err := sqlite.NewEntityRepo(m.store).ListBySite(ctx, siteID)
	if err != nil {
		return contextpack.Map{}, err
	}
	edges, err := sqlite.NewEdgeRepo(m.store).ListBySite(ctx, siteID)
	if err != nil {
		return contextpack.Map{}, err
	}
	fields, err := sqlite.NewPageFieldRepo(m.store).ListBySite(ctx, siteID)
	if err != nil {
		return contextpack.Map{}, err
	}
	g, err := graphdomain.New(entities, edges)
	if err != nil {
		return contextpack.Map{}, err
	}
	kinds := make([]pagetype.Type, 0, len(pagetype.Presets()))
	for _, preset := range pagetype.Presets() {
		kinds = append(kinds, pagetype.Type{ID: string(preset), SiteID: siteID, Name: preset.DefaultName(), Preset: preset})
	}
	return contextpack.Snapshot{
		Index: pagemap.NewIndex(pages), Types: pagetype.NewSet(kinds, nil), Graph: g, Fields: fields,
	}.Map(nil), nil
}

type siteProposals struct {
	service  *appgraph.Service
	store    *sqlite.Store
	model    *scriptedModel
	pages    *sqlite.PageRepo
	findings *sqlite.FindingRepo
	siteID   string
	wpID     int64
}

func newSiteProposals(t *testing.T) *siteProposals {
	t.Helper()

	store := sqlitetest.Open(t)
	owner := sqlitetest.Site(t, store, "shop")
	model := &scriptedModel{}
	findings := sqlite.NewFindingRepo(store)
	return &siteProposals{
		service: appgraph.New(appgraph.Deps{
			Entities: sqlite.NewEntityRepo(store), Edges: sqlite.NewEdgeRepo(store), Sites: sqlite.NewSiteRepo(store),
			Pages: sqlite.NewPageRepo(store), Identities: sqlite.NewPageIdentityRepo(store), Fields: sqlite.NewPageFieldRepo(store),
			Findings: findings, Maps: storeMap{store: store}, Profiles: fixedProfiles{err: errors.New(errors.Invalid, "no model")}, LLM: model,
			UnitOfWork: store, Publisher: &applicationtest.Recorder{}, Clock: clock.NewFake(sqlitetest.Stamp),
		}),
		store: store, model: model, pages: sqlite.NewPageRepo(store), findings: findings, siteID: owner.ID, wpID: 100,
	}
}

func (f *siteProposals) onSite(t *testing.T, path, title string, wpType pagemap.WPType, parent *pagemap.Page, entityID *string) pagemap.Page {
	t.Helper()

	f.wpID++
	record := pagemap.Page{
		ID: id.New(), SiteID: f.siteID, Path: path, Slug: pagemap.Slug(path), WPType: wpType, WPID: new(f.wpID),
		Title: title, H1: title, Status: pagemap.StatusPublished, EntityID: entityID,
		CreatedAt: sqlitetest.Stamp, UpdatedAt: sqlitetest.Stamp,
	}
	if parent != nil {
		record.ParentPageID, record.ParentReason = new(parent.ID), pagemap.ParentJSONLD
	}
	if err := f.pages.Insert(t.Context(), record); err != nil {
		t.Fatalf("insert %s: %v", path, err)
	}
	return record
}

func (f *siteProposals) entity(t *testing.T, name string, scope *string) graphdomain.Entity {
	t.Helper()

	record, err := graphdomain.NewEntity(graphdomain.Entity{
		ID: id.New(), SiteID: f.siteID, Name: name, ScopeID: scope, Source: graphdomain.SourceUser,
		CreatedAt: sqlitetest.Stamp, UpdatedAt: sqlitetest.Stamp,
	})
	if err != nil {
		t.Fatalf("build %s: %v", name, err)
	}
	if err = sqlite.NewEntityRepo(f.store).Insert(t.Context(), record); err != nil {
		t.Fatalf("insert %s: %v", name, err)
	}
	return record
}

func (f *siteProposals) attributes(t *testing.T, page pagemap.Page, raw string) {
	t.Helper()

	if err := sqlite.NewPageFieldRepo(f.store).Observe(t.Context(), page.ID, []field.Value{field.NewValue(page.ID, field.Attributes, raw, sqlitetest.Stamp)}); err != nil {
		t.Fatalf("observe the attributes: %v", err)
	}
}

func TestASitePreviewProposesFromThePagesWithoutAModel(t *testing.T) {
	t.Parallel()

	f := newSiteProposals(t)
	coffee := f.entity(t, "Coffee", nil)
	hub := f.onSite(t, "/coffee/", "Coffee", pagemap.WPPage, nil, &coffee.ID)
	beans := f.onSite(t, "/product/beans/", "House Beans", pagemap.WPProduct, &hub, nil)
	f.attributes(t, beans, `[{"name":"Roast","options":["Medium","Dark"],"visible":true,"variation":true}]`)
	f.onSite(t, "/about/", "About Us", pagemap.WPPage, nil, nil)

	out, err := f.service.PreviewFromPages(t.Context(), appgraph.PreviewFromPagesRequest{SiteID: f.siteID, Source: appgraph.SourceSite})
	if err != nil {
		t.Fatalf("PreviewFromPages: %v", err)
	}
	if len(f.model.calls) != 0 || out.Tokens != 0 {
		t.Errorf("a site preview made %d model calls and %d tokens, want none", len(f.model.calls), out.Tokens)
	}
	paths := make([]string, 0, len(out.Entities))
	for i := range out.Entities {
		paths = append(paths, out.Entities[i].Path)
	}
	if !slices.Equal(paths, []string{"/about/", "/product/beans/"}) || out.Pages != 2 {
		t.Fatalf("proposals = %v over %d pages, want /about/ and the product", paths, out.Pages)
	}
	if !maps.Equal(out.PageTypes, map[string]string{out.Entities[0].PageID: string(pagetype.Guide), beans.ID: string(pagetype.Product)}) {
		t.Errorf("the source pages' types = %v, want each proposal's page by its type", out.PageTypes)
	}
	product := out.Entities[1]
	want := appgraph.ProposedEntity{
		PageID: beans.ID, Path: "/product/beans/", Name: "House Beans", Label: "House Beans", Parent: "/coffee/", ParentEntityID: coffee.ID,
		Facts: []appgraph.Fact{{Label: "Roast", Value: "Medium, Dark"}},
	}
	if product.PageID != want.PageID || product.Name != want.Name || product.Label != want.Label || product.Parent != want.Parent ||
		product.ParentEntityID != want.ParentEntityID || !slices.Equal(product.Facts, want.Facts) || product.Clash != "" {
		t.Errorf("the product's proposal = %+v, want %+v", product, want)
	}

	chosen, err := f.service.PreviewFromPages(t.Context(), appgraph.PreviewFromPagesRequest{
		SiteID: f.siteID, Source: appgraph.SourceSite, PageIDs: []string{beans.ID},
	})
	if err != nil || len(chosen.Entities) != 1 || chosen.Entities[0].PageID != beans.ID {
		t.Errorf("a site preview of one page = %+v %v", chosen.Entities, err)
	}
	if _, err := f.service.PreviewFromPages(t.Context(), appgraph.PreviewFromPagesRequest{SiteID: f.siteID, Source: "galaxy"}); !errors.IsCode(err, errors.Invalid) {
		t.Errorf("an unknown source = %v, want INVALID", err)
	}
	if _, err := f.service.PreviewFromPages(t.Context(), appgraph.PreviewFromPagesRequest{SiteID: f.siteID}); !errors.IsCode(err, errors.Invalid) {
		t.Errorf("a model preview with no model = %v, want INVALID as before", err)
	}
}

func (f *siteProposals) withoutEntity(t *testing.T, page pagemap.Page) {
	t.Helper()

	record, err := finding.New(finding.Finding{
		ID: id.New(), SiteID: f.siteID, PageID: new(page.ID), Code: finding.PageWithoutEntity, SubjectKey: finding.SubjectPage(page.ID),
		Subject: []byte(`{"path":"` + page.Path + `"}`), CreatedAt: sqlitetest.Stamp, UpdatedAt: sqlitetest.Stamp,
	})
	if err != nil {
		t.Fatalf("build the finding: %v", err)
	}
	if err = f.findings.Upsert(t.Context(), record); err != nil {
		t.Fatalf("raise the finding: %v", err)
	}
}

func (f *siteProposals) mappedTo(t *testing.T, page pagemap.Page) graphdomain.Entity {
	t.Helper()

	stored, err := f.pages.Get(t.Context(), page.ID)
	if err != nil || stored.EntityID == nil {
		t.Fatalf("%s carries no entity after the apply: %v", page.Path, err)
	}
	entity, err := sqlite.NewEntityRepo(f.store).Get(t.Context(), *stored.EntityID)
	if err != nil {
		t.Fatalf("read %s's entity: %v", page.Path, err)
	}
	return entity
}

func (f *siteProposals) approvedParent(t *testing.T, child, parent string) bool {
	t.Helper()

	edges, err := sqlite.NewEdgeRepo(f.store).ListBySite(t.Context(), f.siteID)
	if err != nil {
		t.Fatalf("list the edges: %v", err)
	}
	for i := range edges {
		if edges[i].FromEntityID == child && edges[i].ToEntityID == parent && edges[i].Kind == graphdomain.EdgeParent {
			return edges[i].Status == graphdomain.StatusApproved
		}
	}
	return false
}

func TestASiteProposalIsWrittenUnderItsParent(t *testing.T) {
	t.Parallel()

	f := newSiteProposals(t)
	coffee := f.entity(t, "Coffee", nil)
	hub := f.onSite(t, "/coffee/", "Coffee", pagemap.WPPage, nil, &coffee.ID)
	beans := f.onSite(t, "/product/beans/", "House Beans", pagemap.WPProduct, &hub, nil)
	f.attributes(t, beans, `[{"name":"Roast","options":["Medium"],"visible":true,"variation":false}]`)
	kit := f.onSite(t, "/product/kit/", "House Beans", pagemap.WPProduct, &hub, nil)
	same := f.onSite(t, "/product/coffee-sampler/", "Coffee", pagemap.WPProduct, &hub, nil)
	blends := f.onSite(t, "/blends/", "Blends", pagemap.WPPage, nil, nil)
	blend := f.onSite(t, "/product/blend-a/", "Blend A", pagemap.WPProduct, &blends, nil)
	f.withoutEntity(t, beans)

	preview, err := f.service.PreviewFromPages(t.Context(), appgraph.PreviewFromPagesRequest{SiteID: f.siteID, Source: appgraph.SourceSite})
	if err != nil {
		t.Fatalf("PreviewFromPages: %v", err)
	}
	proposals := slices.Clone(preview.Entities)
	slices.Reverse(proposals)
	out, err := f.service.ApplyProposals(t.Context(), appgraph.ApplyProposalsRequest{
		SiteID: f.siteID, Entities: proposals, Source: appgraph.SourceSite,
	})
	if err != nil {
		t.Fatalf("ApplyProposals: %v, want one clash to skip its proposal, never to refuse the batch", err)
	}
	if out.Mapped != 4 || out.Skipped != 1 {
		t.Errorf("mapped %d skipped %d, want 4 mapped and the clashing kit skipped", out.Mapped, out.Skipped)
	}

	written := f.mappedTo(t, beans)
	if written.ScopeID == nil || *written.ScopeID != coffee.ID || written.Source != graphdomain.SourceUser ||
		!slices.Equal(written.Facts, []graphdomain.Fact{{Label: "Roast", Value: "Medium"}}) || written.CanonicalPageID == nil {
		t.Errorf("the product's entity = %+v, want it under Coffee, the person's, with its store facts and its page", written)
	}
	if !f.approvedParent(t, written.ID, coffee.ID) {
		t.Errorf("the product's parent edge is not approved")
	}
	if stored, getErr := f.pages.Get(t.Context(), kit.ID); getErr != nil || stored.EntityID != nil {
		t.Errorf("the clashing kit = %v %v, want it left without an entity", stored.EntityID, getErr)
	}

	named := f.mappedTo(t, same)
	if named.ID == coffee.ID || named.ScopeID == nil || *named.ScopeID != coffee.ID {
		t.Errorf("a product named like its hub = %+v, want an entity of its own under the hub", named)
	}
	hubEntity := f.mappedTo(t, blends)
	child := f.mappedTo(t, blend)
	if hubEntity.ScopeID != nil || child.ScopeID == nil || *child.ScopeID != hubEntity.ID || !f.approvedParent(t, child.ID, hubEntity.ID) {
		t.Errorf("a hub and its product in one batch = %+v and %+v, want the product under the new hub", hubEntity, child)
	}

	cleared, err := f.findings.OpenBySite(t.Context(), f.siteID, finding.PageWithoutEntity)
	if err != nil || len(cleared) != 0 {
		t.Errorf("open page_without_entity findings after the apply = %d %v, want the mapped page's cleared", len(cleared), err)
	}
}
