package findings_test

import (
	"context"
	"encoding/json"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/davidmovas/postulator/internal/application/events"
	"github.com/davidmovas/postulator/internal/application/findings"
	"github.com/davidmovas/postulator/internal/application/graph"
	"github.com/davidmovas/postulator/internal/application/imports"
	"github.com/davidmovas/postulator/internal/domain/contextpack"
	"github.com/davidmovas/postulator/internal/domain/field"
	"github.com/davidmovas/postulator/internal/domain/finding"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/pagetype"
	"github.com/davidmovas/postulator/internal/domain/run"
	"github.com/davidmovas/postulator/internal/kernel/clock"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

const (
	siteID = "site-1"
	pageID = "page-1"
)

var at = time.Date(2026, time.October, 23, 12, 0, 0, 0, time.UTC)

type store struct {
	held  map[string]finding.Finding
	calls []string
}

func (s *store) Get(_ context.Context, id string) (finding.Finding, error) {
	held, found := s.held[id]
	if !found {
		return finding.Finding{}, errors.New(errors.NotFound, "finding not found")
	}
	return held, nil
}

func (s *store) Settle(_ context.Context, id string, status finding.Status, resolution, resolvedBy string, _ time.Time) error {
	held := s.held[id]
	if held.Status != finding.Open {
		return errors.New(errors.Conflict, "the finding is no longer open")
	}
	held.Status, held.Resolution, held.ResolvedBy = status, resolution, resolvedBy
	s.held[id] = held
	return nil
}

func (s *store) ListBySite(_ context.Context, site string, status finding.Status) ([]finding.Finding, error) {
	out := make([]finding.Finding, 0, len(s.held))
	for id := range s.held {
		if held := s.held[id]; held.SiteID == site && (status == "" || held.Status == status) {
			out = append(out, held)
		}
	}
	slices.SortFunc(out, func(a, b finding.Finding) int { return a.CreatedAt.Compare(b.CreatedAt) })
	return out, nil
}

func (s *store) AcceptSite(_ context.Context, page string, key field.Key, _ time.Time) error {
	s.calls = append(s.calls, "field:"+page+":"+string(key))
	return nil
}

type bodies struct{ s *store }

func (b bodies) AcceptSite(_ context.Context, page string, _ time.Time) error {
	b.s.calls = append(b.s.calls, "body:"+page)
	return nil
}

type pageRows struct{ s *store }

func (p pageRows) SetParentSource(_ context.Context, page string, source pagemap.ParentSource, _ time.Time) error {
	p.s.calls = append(p.s.calls, "parent:"+page+":"+string(source))
	return nil
}

func (pageRows) Get(_ context.Context, id string) (pagemap.Page, error) {
	return pagemap.Page{ID: id, SiteID: siteID, Path: "/" + id + "/"}, nil
}

func (p pageRows) Update(_ context.Context, page pagemap.Page) error {
	switch {
	case page.Title != "":
		p.s.calls = append(p.s.calls, "title:"+page.ID+":"+page.Title)
	case page.H1 != "":
		p.s.calls = append(p.s.calls, "h1:"+page.ID+":"+page.H1)
	}
	return nil
}

func (p pageRows) SetType(_ context.Context, pageID, typeID string, reason pagemap.TypeReason, _ time.Time) error {
	p.s.calls = append(p.s.calls, "type:"+pageID+":"+typeID+":"+string(reason))
	return nil
}

type counter struct {
	mu        sync.Mutex
	published []events.Type
}

func (c *counter) Publish(eventType events.Type, _ any) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.published = append(c.published, eventType)
	return nil
}

func (s *store) AcceptCard(_ context.Context, req graph.AcceptCardRequest) (graph.AcceptCardResponse, error) {
	s.calls = append(s.calls, "card:"+req.FindingID+":"+req.ProposalHash+":"+req.By)
	return graph.AcceptCardResponse{}, s.Settle(context.Background(), req.FindingID, finding.Resolved, "accepted", req.By, at)
}

func (s *store) MapRow(_ context.Context, req imports.MapRowRequest) (imports.MapRowResponse, error) {
	s.calls = append(s.calls, "row:"+req.FindingID+":"+req.PageID+":"+req.By)
	return imports.MapRowResponse{}, s.Settle(context.Background(), req.FindingID, finding.Resolved, "mapped", req.By, at)
}

type unit struct{}

func (unit) Do(ctx context.Context, fn func(context.Context) error) error {
	return fn(ctx)
}

type places struct{}

func sitePage(id, path, title, parent, typeID string) pagemap.Page {
	page := pagemap.Page{
		ID: id, SiteID: siteID, Path: path, Slug: pagemap.Slug(path), WPType: pagemap.WPPage, WPID: new(int64(len(path))),
		Status: pagemap.StatusPublished, Title: title, CreatedAt: at, UpdatedAt: at,
	}
	if parent != "" {
		page.ParentPageID, page.ParentReason = new(parent), pagemap.ParentJSONLD
	}
	if typeID != "" {
		page.PageTypeID = new(typeID)
	}
	return page
}

func (places) Map(_ context.Context, site string) (contextpack.Map, error) {
	if site != siteID {
		return contextpack.Map{}, errors.New(errors.NotFound, "site not found")
	}
	kinds := []pagetype.Type{
		{ID: "type-hub", SiteID: siteID, Name: "Hub", Preset: pagetype.Hub},
		{ID: "type-entity", SiteID: siteID, Name: "Guide", Preset: pagetype.Entity},
	}
	held := []pagemap.Page{
		sitePage("page-root", "/", "Home", "", ""),
		sitePage(pageID, "/a/", "Coffee", "page-root", "type-hub"),
		sitePage("page-2", "/a/b/", "Beans", pageID, "type-entity"),
		sitePage("page-3", "/a/b/c/", "Roasts", "page-2", "type-entity"),
		sitePage("page-4", "/a/x/", "Kettles", "page-5", "type-entity"),
		sitePage("page-5", "/z/", "Tools", "page-root", "type-hub"),
	}
	return contextpack.Snapshot{Index: pagemap.NewIndex(held), Types: pagetype.NewSet(kinds, nil)}.Map(nil), nil
}

func newService(t *testing.T, held ...finding.Finding) (*findings.Service, *store) {
	t.Helper()
	service, s, _ := newCounted(t, held...)
	return service, s
}

func newCounted(t *testing.T, held ...finding.Finding) (*findings.Service, *store, *counter) {
	t.Helper()
	s := &store{held: make(map[string]finding.Finding)}
	for i := range held {
		record, err := finding.New(held[i])
		if err != nil {
			t.Fatalf("seed %s: %v", held[i].Code, err)
		}
		s.held[record.ID] = record
	}
	published := &counter{}
	return findings.New(findings.Deps{
		Findings: s, Fields: s, Bodies: bodies{s}, Pages: pageRows{s}, Cards: s, Rows: s, Places: places{},
		UnitOfWork: unit{}, Publisher: published, Clock: clock.NewFake(at),
	}), s, published
}

func raised(code finding.Code, subject string) finding.Finding {
	page := pageID
	return finding.Finding{
		ID: "f-" + string(code), SiteID: siteID, PageID: &page, Code: code, SubjectKey: finding.SubjectPage(page),
		Subject: json.RawMessage(subject), CreatedAt: at, UpdatedAt: at,
	}
}

func TestEachResolutionOptionPerCode(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		raised  finding.Finding
		req     findings.ResolveRequest
		calls   []string
		status  finding.Status
		settles bool
	}{
		{
			name: "a field the client changed is taken as his", raised: raised(finding.FieldChangedOnSite, `{"key":"seo_title"}`),
			req: findings.ResolveRequest{Option: "accept_site"}, calls: []string{"field:page-1:seo_title"}, status: finding.Resolved, settles: true,
		},
		{
			name: "a body the client changed is taken as his", raised: raised(finding.BodyChangedOnSite, `{}`),
			req: findings.ResolveRequest{Option: "accept_site"}, calls: []string{"body:page-1"}, status: finding.Resolved, settles: true,
		},
		{
			name: "the site's hierarchy wins", raised: raised(finding.ParentDiffers, `{"sheet":["A"],"site":["B"]}`),
			req: findings.ResolveRequest{Option: "accept_site"}, calls: []string{"parent:page-1:"}, status: finding.Resolved, settles: true,
		},
		{
			name: "the workbook's hierarchy is pinned", raised: raised(finding.ParentDiffers, `{"sheet":["A"],"site":["B"]}`),
			req: findings.ResolveRequest{Option: "accept_workbook"}, calls: []string{"parent:page-1:workbook"}, status: finding.Resolved, settles: true,
		},
		{
			name: "the plan takes the site's title", raised: raised(finding.PageTitleDiffers, `{"planned":"Mugs","site":"Ceramic mugs"}`),
			req: findings.ResolveRequest{Option: "accept_site"}, calls: []string{"title:page-1:Ceramic mugs"}, status: finding.Resolved, settles: true,
		},
		{
			name: "the plan takes the site's H1", raised: raised(finding.PageH1Differs, `{"planned":"Mugs","site":"Our mugs"}`),
			req: findings.ResolveRequest{Option: "accept_site"}, calls: []string{"h1:page-1:Our mugs"}, status: finding.Resolved, settles: true,
		},
		{
			name: "a technical page is typed by the agent", raised: raised(finding.TechnicalDetected, `{"reason":"shortcode"}`),
			req:   findings.ResolveRequest{Option: "set_type", PageTypeID: "type-entity", By: finding.ResolvedByAgent},
			calls: []string{"type:page-1:type-entity:ai"}, status: finding.Resolved, settles: true,
		},
		{
			name: "a card is accepted as shown", raised: raised(finding.EntityCardProposed, `{}`),
			req: findings.ResolveRequest{Option: "accept", ProposalHash: "3f1a"}, calls: []string{"card:f-entity_card_proposed:3f1a:person"},
			status: finding.Resolved, settles: true,
		},
		{
			name: "an ambiguous row is mapped to a page", raised: raised(finding.RowAmbiguous, `{"sheet":"Groups"}`),
			req: findings.ResolveRequest{Option: "map_to_page", PageID: "page-2"}, calls: []string{"row:f-row_ambiguous:page-2:person"},
			status: finding.Resolved, settles: true,
		},
		{
			name: "a row without a page is ignored", raised: raised(finding.RowWithoutPage, `{"sheet":"Groups"}`),
			req: findings.ResolveRequest{Option: "ignore"}, status: finding.Dismissed, settles: true,
		},
		{
			name: "a field is rewritten by a run", raised: raised(finding.FieldChangedOnSite, `{"key":"excerpt"}`),
			req: findings.ResolveRequest{Option: "rewrite_field"}, status: finding.Open,
		},
		{
			name: "a body is relinked by a run", raised: raised(finding.BodyChangedOnSite, `{}`),
			req: findings.ResolveRequest{Option: "relink_page"}, status: finding.Open,
		},
		{
			name: "an unreachable page is fetched again", raised: raised(finding.RenderUnreachable, `{"status":403}`),
			req: findings.ResolveRequest{Option: "rescan"}, status: finding.Open,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			service, s := newService(t, tc.raised)
			tc.req.FindingID = tc.raised.ID
			resolved, err := service.Resolve(t.Context(), tc.req)
			if err != nil {
				t.Fatalf("Resolve: %v", err)
			}
			if !slices.Equal(s.calls, tc.calls) {
				t.Errorf("model changes = %v, want %v", s.calls, tc.calls)
			}
			if resolved.Finding.Status != string(tc.status) || s.held[tc.raised.ID].Status != tc.status {
				t.Errorf("the finding is %s (held %s), want %s", resolved.Finding.Status, s.held[tc.raised.ID].Status, tc.status)
			}
			if want := tc.req.By; tc.settles && want != "" && s.held[tc.raised.ID].ResolvedBy != want {
				t.Errorf("resolved by %q, want %q", s.held[tc.raised.ID].ResolvedBy, want)
			}
			if tc.settles && (resolved.Start != nil || resolved.Sync != nil) {
				t.Errorf("a model change prepared a run: %+v", resolved)
			}
		})
	}
}

func TestARunOptionOnlyPreparesTheStart(t *testing.T) {
	t.Parallel()

	service, _ := newService(t,
		raised(finding.FieldChangedOnSite, `{"key":"seo_description"}`),
		raised(finding.PageTitleDiffers, `{"planned":"Mugs","site":"Ceramic mugs"}`),
		raised(finding.BodyChangedOnSite, `{}`),
		raised(finding.RenderRedirected, `{"target":"/x/"}`),
	)
	rewrite, err := service.Resolve(t.Context(), findings.ResolveRequest{FindingID: "f-field_changed_on_site", Option: "rewrite_field"})
	if err != nil || rewrite.Start == nil || !slices.Equal(rewrite.Start.Fields, []string{"seo_description"}) ||
		!slices.Equal(rewrite.Start.Rewrite, []string{pageID}) || !slices.Equal(rewrite.Start.PageIDs, []string{pageID}) {
		t.Errorf("rewrite_field = %+v, %v; want a start scoped to the field, rewriting the client's value", rewrite.Start, err)
	}
	title, err := service.Resolve(t.Context(), findings.ResolveRequest{FindingID: "f-page_title_differs", Option: "rewrite_field"})
	if err != nil || title.Start == nil || !slices.Equal(title.Start.Fields, []string{string(field.Title)}) {
		t.Errorf("rewrite_field on the title = %+v, %v", title.Start, err)
	}
	relink, err := service.Resolve(t.Context(), findings.ResolveRequest{FindingID: "f-body_changed_on_site", Option: "relink_page"})
	if err != nil || relink.Start == nil || relink.Start.Kind != string(run.KindRelink) || len(relink.Start.Fields) != 0 {
		t.Errorf("relink_page = %+v, %v; want a relink start", relink.Start, err)
	}
	rescan, err := service.Resolve(t.Context(), findings.ResolveRequest{FindingID: "f-render_redirected", Option: "rescan"})
	if err != nil || rescan.Sync == nil || !rescan.Sync.Full || !rescan.Sync.RenderRescan || rescan.Sync.SiteID != siteID {
		t.Errorf("rescan = %+v, %v; want a full walk that renders every page again", rescan.Sync, err)
	}
}

func TestAResolutionIsRefusedWhenItCannotApply(t *testing.T) {
	t.Parallel()

	settled := raised(finding.ParentDiffers, `{}`)
	settled.ID, settled.Status = "f-settled", finding.Dismissed
	product := raised(finding.ParentDiffers, `{}`)
	product.ID, product.Options = "f-product", pagemap.TrailOptions(pagemap.WPProduct)
	cases := []struct {
		name string
		req  findings.ResolveRequest
		code errors.Code
	}{
		{name: "an option the code does not offer", req: findings.ResolveRequest{FindingID: "f-parent_differs", Option: "set_type"}, code: errors.Invalid},
		{name: "an option nobody knows", req: findings.ResolveRequest{FindingID: "f-parent_differs", Option: "merge"}, code: errors.Invalid},
		{name: "someone who is no person and no agent", req: findings.ResolveRequest{FindingID: "f-parent_differs", Option: "ignore", By: "cron"}, code: errors.Invalid},
		{name: "a finding settled already", req: findings.ResolveRequest{FindingID: "f-settled", Option: "ignore"}, code: errors.Conflict},
		{name: "a finding nobody raised", req: findings.ResolveRequest{FindingID: "f-none", Option: "ignore"}, code: errors.NotFound},
		{name: "a product pinned to the workbook", req: findings.ResolveRequest{FindingID: "f-product", Option: "accept_workbook"}, code: errors.Invalid},
		{name: "a type without a type", req: findings.ResolveRequest{FindingID: "f-technical_detected", Option: "set_type"}, code: errors.Invalid},
		{name: "a mapping without a page", req: findings.ResolveRequest{FindingID: "f-row_ambiguous", Option: "map_to_page"}, code: errors.Invalid},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			service, s := newService(t, raised(finding.ParentDiffers, `{}`), settled, product,
				raised(finding.TechnicalDetected, `{}`), raised(finding.RowAmbiguous, `{}`))
			if _, err := service.Resolve(t.Context(), tc.req); !errors.IsCode(err, tc.code) {
				t.Errorf("Resolve(%+v) = %v, want %s", tc.req, err, tc.code)
			}
			if len(s.calls) != 0 {
				t.Errorf("a refused resolution changed the model: %v", s.calls)
			}
		})
	}
}

func TestFindingsAreListedByCursor(t *testing.T) {
	t.Parallel()

	held := make([]finding.Finding, 0, 3)
	for i, code := range []finding.Code{finding.ParentDiffers, finding.BodyChangedOnSite, finding.RowWithoutPage} {
		f := raised(code, `{}`)
		f.CreatedAt = at.Add(time.Duration(i) * time.Minute)
		held = append(held, f)
	}
	held[2].PageID = nil
	service, _ := newService(t, held...)

	first, err := service.List(t.Context(), findings.ListRequest{SiteID: siteID, Limit: 2})
	if err != nil || len(first.Items) != 2 || !first.HasMore || first.Items[0].Code != "parent_differs" {
		t.Fatalf("the first page = %+v, %v", first, err)
	}
	if kinds := first.Items[0].Options; len(kinds) != 3 || kinds[0].Kind != "model" {
		t.Errorf("options = %+v, want each with its kind", kinds)
	}
	rest, err := service.List(t.Context(), findings.ListRequest{SiteID: siteID, Limit: 2, Cursor: string(first.Next)})
	if err != nil || len(rest.Items) != 1 || rest.Items[0].Code != "row_without_page" || rest.HasMore {
		t.Errorf("the next page = %+v, %v", rest, err)
	}
	onPage, err := service.List(t.Context(), findings.ListRequest{SiteID: siteID, PageID: pageID, Code: "body_changed_on_site"})
	if err != nil || len(onPage.Items) != 1 {
		t.Errorf("filtered = %+v, %v", onPage, err)
	}
	if _, err := service.List(t.Context(), findings.ListRequest{SiteID: siteID, Code: "made_up"}); !errors.IsCode(err, errors.Invalid) {
		t.Errorf("an unknown code = %v, want Invalid", err)
	}
}

func onPage(code finding.Code, page string) finding.Finding {
	f := raised(code, `{}`)
	f.ID, f.PageID, f.SubjectKey = "f-"+string(code)+"-"+page, new(page), finding.SubjectPage(page)
	return f
}

func spread() []finding.Finding {
	loose := raised(finding.PluginOutdated, `{}`)
	loose.PageID, loose.SubjectKey = nil, finding.SubjectSite
	return []finding.Finding{
		onPage(finding.FieldChangedOnSite, pageID), onPage(finding.ParentDiffers, "page-2"),
		onPage(finding.BodyChangedOnSite, "page-3"), onPage(finding.TechnicalDetected, "page-4"), loose,
	}
}

func idsOf(listed []findings.Finding) []string {
	out := make([]string, 0, len(listed))
	for i := range listed {
		out = append(out, listed[i].ID)
	}
	slices.Sort(out)
	return out
}

func TestFindingsAreListedByPageTypeAndSubtree(t *testing.T) {
	t.Parallel()

	service, _ := newService(t, spread()...)
	cases := []struct {
		name string
		req  findings.ListRequest
		want []string
	}{
		{
			name: "a page and its descendants by derived parent", req: findings.ListRequest{Under: pageID},
			want: []string{"f-body_changed_on_site-page-3", "f-field_changed_on_site-page-1", "f-parent_differs-page-2"},
		},
		{name: "a deeper subtree", req: findings.ListRequest{Under: "page-2"}, want: []string{"f-body_changed_on_site-page-3", "f-parent_differs-page-2"}},
		{name: "a path neighbor under another parent", req: findings.ListRequest{Under: "page-5"}, want: []string{"f-technical_detected-page-4"}},
		{name: "the page's type", req: findings.ListRequest{PageTypeID: "type-hub"}, want: []string{"f-field_changed_on_site-page-1"}},
		{name: "type and subtree", req: findings.ListRequest{Under: pageID, PageTypeID: "type-entity"}, want: []string{"f-body_changed_on_site-page-3", "f-parent_differs-page-2"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			tc.req.SiteID = siteID
			listed, err := service.List(t.Context(), tc.req)
			if err != nil {
				t.Fatalf("List: %v", err)
			}
			if got := idsOf(listed.Items); !slices.Equal(got, tc.want) {
				t.Errorf("List(%+v) = %v, want %v", tc.req, got, tc.want)
			}
		})
	}

	if _, err := service.List(t.Context(), findings.ListRequest{SiteID: siteID, Under: "page-nobody"}); !errors.IsCode(err, errors.NotFound) {
		t.Errorf("a subtree under no page = %v, want NotFound", err)
	}
}

func TestAFindingNamesItsPageInWords(t *testing.T) {
	t.Parallel()

	service, _ := newService(t, spread()...)
	listed, err := service.List(t.Context(), findings.ListRequest{SiteID: siteID})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	for _, held := range listed.Items {
		switch held.ID {
		case "f-body_changed_on_site-page-3":
			if held.PagePath != "/a/b/c/" || held.PageTitle != "Roasts" {
				t.Errorf("the body finding names %q %q, want /a/b/c/ Roasts", held.PagePath, held.PageTitle)
			}
		case "f-plugin_outdated":
			if held.PagePath != "" || held.PageTitle != "" {
				t.Errorf("a site's finding names a page: %q %q", held.PagePath, held.PageTitle)
			}
		}
	}
	got, err := service.Get(t.Context(), findings.GetRequest{SiteID: siteID, FindingID: "f-parent_differs-page-2"})
	if err != nil || got.PagePath != "/a/b/" || got.PageTitle != "Beans" {
		t.Errorf("Get = %+v, %v; want it to name /a/b/ Beans", got, err)
	}
}

func TestFindingsAreCountedByCodeUnderTheListsFilters(t *testing.T) {
	t.Parallel()

	service, _ := newService(t, spread()...)
	all, err := service.Counts(t.Context(), findings.CountsRequest{SiteID: siteID})
	if err != nil {
		t.Fatalf("Counts: %v", err)
	}
	if all.Total != 5 || len(all.Codes) != 5 {
		t.Errorf("counts = %+v, want five codes of one", all)
	}
	for i := 1; i < len(all.Codes); i++ {
		if all.Codes[i-1].Severity == string(finding.Info) && all.Codes[i].Severity == string(finding.Warning) {
			t.Errorf("counts = %+v, want warnings before notes", all.Codes)
		}
	}
	under, err := service.Counts(t.Context(), findings.CountsRequest{SiteID: siteID, Under: pageID, PageTypeID: "type-entity"})
	if err != nil || under.Total != 2 {
		t.Errorf("counts under /a/ typed Guide = %+v, %v; want 2", under, err)
	}
	if _, err := service.Counts(t.Context(), findings.CountsRequest{SiteID: siteID, Status: "lost"}); !errors.IsCode(err, errors.Invalid) {
		t.Errorf("an unknown status = %v, want Invalid", err)
	}
}

func TestOneSentenceExplainsWhatAResolutionDoes(t *testing.T) {
	t.Parallel()

	technical := func(page string) finding.Finding { return onPage(finding.TechnicalDetected, page) }
	changed := onPage(finding.FieldChangedOnSite, "page-5")
	changed.Subject = json.RawMessage(`{"key":"seo_description"}`)
	service, _ := newService(t, technical(pageID), technical("page-2"), technical("page-3"), technical("page-4"),
		changed, onPage(finding.RowAmbiguous, "page-2"), onPage(finding.RenderRedirected, pageID))
	cases := []struct {
		name string
		req  findings.ExplainRequest
		says []string
		kind finding.OptionKind
	}{
		{
			name: "a dismissal", req: findings.ExplainRequest{FindingIDs: []string{"f-technical_detected-page-1"}, Option: "ignore"},
			says: []string{"Dismiss the finding technical_detected on /a/", "nothing is written to the site"}, kind: finding.ModelChange,
		},
		{
			name: "a type named", req: findings.ExplainRequest{FindingIDs: []string{"f-technical_detected-page-2"}, Option: "set_type", PageTypeID: "type-hub"},
			says: []string{"Type /a/b/ as Hub"}, kind: finding.ModelChange,
		},
		{
			name: "a row mapped onto a page", req: findings.ExplainRequest{FindingIDs: []string{"f-row_ambiguous-page-2"}, Option: "map_to_page", PageID: "page-5"},
			says: []string{"onto /z/"}, kind: finding.ModelChange,
		},
		{
			name: "a field rewritten by a run", req: findings.ExplainRequest{FindingIDs: []string{"f-field_changed_on_site-page-5"}, Option: "rewrite_field"},
			says: []string{"run start that rewrites the seo_description of /z/", "estimate"}, kind: finding.Run,
		},
		{
			name: "a rescan", req: findings.ExplainRequest{FindingIDs: []string{"f-render_redirected-page-1"}, Option: "rescan"},
			says: []string{"full sync", "renders every page again"}, kind: finding.Run,
		},
		{
			name: "four findings dismissed at once",
			req: findings.ExplainRequest{Option: "ignore", FindingIDs: []string{
				"f-technical_detected-page-1", "f-technical_detected-page-2", "f-technical_detected-page-3", "f-technical_detected-page-4",
			}},
			says: []string{"Dismiss the 4 findings technical_detected on /a/, /a/b/, /a/b/c/ and 1 more", "nothing is written to the site"},
			kind: finding.ModelChange,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			tc.req.SiteID = siteID
			said, err := service.Explain(t.Context(), tc.req)
			if err != nil {
				t.Fatalf("Explain: %v", err)
			}
			for _, part := range tc.says {
				if !strings.Contains(said.Sentence, part) {
					t.Errorf("the sentence %q does not say %q", said.Sentence, part)
				}
			}
			if said.Kind != string(tc.kind) {
				t.Errorf("kind = %q, want %q", said.Kind, tc.kind)
			}
		})
	}

	refused := []struct {
		name string
		req  findings.ExplainRequest
		code errors.Code
	}{
		{name: "no finding", req: findings.ExplainRequest{Option: "ignore"}, code: errors.Invalid},
		{name: "two codes at once", req: findings.ExplainRequest{Option: "ignore", FindingIDs: []string{"f-technical_detected-page-1", "f-row_ambiguous-page-2"}}, code: errors.Invalid},
		{name: "a run option at once", req: findings.ExplainRequest{Option: "rescan", FindingIDs: []string{"f-render_redirected-page-1", "f-render_redirected-page-1"}}, code: errors.Invalid},
		{name: "a type the site lacks", req: findings.ExplainRequest{Option: "set_type", PageTypeID: "type-nobody", FindingIDs: []string{"f-technical_detected-page-1"}}, code: errors.NotFound},
	}
	for _, tc := range refused {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			tc.req.SiteID = siteID
			if _, err := service.Explain(t.Context(), tc.req); !errors.IsCode(err, tc.code) {
				t.Errorf("Explain(%+v) = %v, want %s", tc.req, err, tc.code)
			}
		})
	}
}

func TestEveryFindingOfACodeIsResolvedInOneConfirmedAction(t *testing.T) {
	t.Parallel()

	ids := []string{"f-technical_detected-page-1", "f-technical_detected-page-2", "f-technical_detected-page-3"}
	held := []finding.Finding{
		onPage(finding.TechnicalDetected, pageID), onPage(finding.TechnicalDetected, "page-2"), onPage(finding.TechnicalDetected, "page-3"),
	}

	t.Run("each is typed and the window hears once", func(t *testing.T) {
		t.Parallel()

		service, s, published := newCounted(t, held...)
		resolved, err := service.ResolveMany(t.Context(), findings.ResolveManyRequest{
			SiteID: siteID, Code: "technical_detected", Option: "set_type", PageTypeID: "type-entity", FindingIDs: ids,
		})
		if err != nil {
			t.Fatalf("ResolveMany: %v", err)
		}
		want := []string{"type:page-1:type-entity:user", "type:page-2:type-entity:user", "type:page-3:type-entity:user"}
		if !slices.Equal(s.calls, want) {
			t.Errorf("model changes = %v, want %v", s.calls, want)
		}
		if len(resolved.Findings) != 3 {
			t.Fatalf("answered %d findings, want 3", len(resolved.Findings))
		}
		for _, shown := range resolved.Findings {
			if shown.Status != string(finding.Resolved) || shown.ResolvedBy != finding.ResolvedByPerson || shown.PagePath == "" {
				t.Errorf("%s = %+v, want it resolved by the person and named by its page", shown.ID, shown)
			}
		}
		if !slices.Equal(published.published, []events.Type{events.PagesChanged}) {
			t.Errorf("published %v, want one pages.changed after the unit", published.published)
		}
	})

	t.Run("each is dismissed", func(t *testing.T) {
		t.Parallel()

		service, s, _ := newCounted(t, held...)
		if _, err := service.ResolveMany(t.Context(), findings.ResolveManyRequest{
			SiteID: siteID, Code: "technical_detected", Option: "ignore", FindingIDs: ids,
		}); err != nil {
			t.Fatalf("ResolveMany: %v", err)
		}
		for _, id := range ids {
			if s.held[id].Status != finding.Dismissed {
				t.Errorf("%s is %s, want dismissed", id, s.held[id].Status)
			}
		}
	})

	settled := onPage(finding.TechnicalDetected, "page-4")
	settled.Status = finding.Dismissed
	product := onPage(finding.ParentDiffers, "page-5")
	product.Options = pagemap.TrailOptions(pagemap.WPProduct)
	refused := []struct {
		name string
		req  findings.ResolveManyRequest
		code errors.Code
	}{
		{name: "a run option", req: findings.ResolveManyRequest{Code: "field_changed_on_site", Option: "rewrite_field", FindingIDs: []string{"f-field_changed_on_site-page-1"}}, code: errors.Invalid},
		{name: "a mapping, which names a page per row", req: findings.ResolveManyRequest{Code: "row_ambiguous", Option: "map_to_page", FindingIDs: []string{"f-row_ambiguous-page-2"}}, code: errors.Invalid},
		{name: "a card, read one by one", req: findings.ResolveManyRequest{Code: "entity_card_proposed", Option: "accept", FindingIDs: []string{"f-entity_card_proposed"}}, code: errors.Invalid},
		{name: "a finding settled since", req: findings.ResolveManyRequest{Code: "technical_detected", Option: "ignore", FindingIDs: append(slices.Clone(ids), "f-technical_detected-page-4")}, code: errors.Conflict},
		{name: "another code among them", req: findings.ResolveManyRequest{Code: "technical_detected", Option: "ignore", FindingIDs: append(slices.Clone(ids), "f-row_ambiguous-page-2")}, code: errors.Invalid},
		{name: "a code that is not theirs", req: findings.ResolveManyRequest{Code: "builder_detected", Option: "ignore", FindingIDs: ids}, code: errors.Invalid},
		{name: "a finding named twice", req: findings.ResolveManyRequest{Code: "technical_detected", Option: "ignore", FindingIDs: append(slices.Clone(ids), ids[0])}, code: errors.Invalid},
		{name: "an option one of them does not offer", req: findings.ResolveManyRequest{Code: "parent_differs", Option: "accept_workbook", FindingIDs: []string{"f-parent_differs-page-2", "f-parent_differs-page-5"}}, code: errors.Invalid},
		{name: "a type the site lacks", req: findings.ResolveManyRequest{Code: "technical_detected", Option: "set_type", PageTypeID: "type-nobody", FindingIDs: ids}, code: errors.NotFound},
		{name: "nothing named", req: findings.ResolveManyRequest{Code: "technical_detected", Option: "ignore"}, code: errors.Invalid},
	}
	for _, tc := range refused {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			service, s, published := newCounted(t, append(slices.Clone(held), settled, product, onPage(finding.ParentDiffers, "page-2"),
				onPage(finding.FieldChangedOnSite, pageID), onPage(finding.RowAmbiguous, "page-2"), raised(finding.EntityCardProposed, `{}`))...)
			tc.req.SiteID = siteID
			if _, err := service.ResolveMany(t.Context(), tc.req); !errors.IsCode(err, tc.code) {
				t.Errorf("ResolveMany(%+v) = %v, want %s", tc.req, err, tc.code)
			}
			if len(s.calls) != 0 || len(published.published) != 0 {
				t.Errorf("a refused bulk changed %v and published %v", s.calls, published.published)
			}
			for _, id := range ids {
				if s.held[id].Status != finding.Open {
					t.Errorf("a refused bulk settled %s", id)
				}
			}
		})
	}
}

func TestAFindingOfAnotherSiteIsNeitherReadNorResolved(t *testing.T) {
	t.Parallel()

	service, s := newService(t, raised(finding.TechnicalDetected, `{"reason":"shortcode"}`))
	held, err := service.Get(t.Context(), findings.GetRequest{SiteID: siteID, FindingID: "f-technical_detected"})
	if err != nil || held.Code != "technical_detected" || len(held.Options) != 2 {
		t.Fatalf("Get on its own site = %+v, %v", held, err)
	}
	if _, err := service.Get(t.Context(), findings.GetRequest{SiteID: "site-2", FindingID: "f-technical_detected"}); !errors.IsCode(err, errors.NotFound) {
		t.Errorf("Get from another site = %v, want NotFound", err)
	}
	_, err = service.Resolve(t.Context(), findings.ResolveRequest{
		SiteID: "site-2", FindingID: "f-technical_detected", Option: "ignore", By: finding.ResolvedByAgent,
	})
	if !errors.IsCode(err, errors.NotFound) {
		t.Errorf("Resolve from another site = %v, want NotFound", err)
	}
	if s.held["f-technical_detected"].Status != finding.Open || len(s.calls) != 0 {
		t.Errorf("a resolution from another site changed %+v, %v", s.held["f-technical_detected"], s.calls)
	}
}

func TestProposingEntitiesOnlyPreparesTheSitePreview(t *testing.T) {
	t.Parallel()

	ids := []string{"f-page_without_entity-page-1", "f-page_without_entity-page-2", "f-page_without_entity-page-3"}
	service, s, published := newCounted(t,
		onPage(finding.PageWithoutEntity, pageID), onPage(finding.PageWithoutEntity, "page-2"), onPage(finding.PageWithoutEntity, "page-3"))

	one, err := service.Resolve(t.Context(), findings.ResolveRequest{FindingID: ids[0], Option: "propose_entity"})
	want := graph.PreviewFromPagesRequest{SiteID: siteID, PageIDs: []string{pageID}, Source: graph.SourceSite}
	if err != nil || one.Proposals == nil || one.Proposals.SiteID != want.SiteID || !slices.Equal(one.Proposals.PageIDs, want.PageIDs) ||
		one.Proposals.Source != want.Source || one.Finding.PagePath != "/a/" {
		t.Errorf("propose_entity = %+v %v, want the site preview of its page", one, err)
	}

	many, err := service.ResolveMany(t.Context(), findings.ResolveManyRequest{
		SiteID: siteID, Code: "page_without_entity", Option: "propose_entity", FindingIDs: ids,
	})
	if err != nil || many.Proposals == nil || !slices.Equal(many.Proposals.PageIDs, []string{pageID, "page-2", "page-3"}) ||
		many.Proposals.Source != graph.SourceSite || len(many.Findings) != 3 {
		t.Errorf("propose_entity in bulk = %+v %v, want one site preview of every page", many, err)
	}
	for _, id := range ids {
		if s.held[id].Status != finding.Open {
			t.Errorf("%s is %s after a proposal, want it open until an entity carries its page", id, s.held[id].Status)
		}
	}
	if len(s.calls) != 0 || len(published.published) != 0 {
		t.Errorf("a proposal changed the model %v and published %v, want neither", s.calls, published.published)
	}

	said, err := service.Explain(t.Context(), findings.ExplainRequest{FindingIDs: ids, Option: "propose_entity"})
	if err != nil || said.Kind != string(finding.Proposal) || !strings.Contains(said.Sentence, "entity proposals") ||
		!strings.HasSuffix(said.Sentence, "nothing is written to the site") {
		t.Errorf("Explain(propose_entity) = %+v %v", said, err)
	}
}
