package steps_test

import (
	"bytes"
	"cmp"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"reflect"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/davidmovas/postulator/internal/adapters/sqlite"
	"github.com/davidmovas/postulator/internal/adapters/wp"
	"github.com/davidmovas/postulator/internal/adapters/wp/wptest"
	"github.com/davidmovas/postulator/internal/domain/content"
	"github.com/davidmovas/postulator/internal/domain/graph"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/run"
	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/runtime/steps"
)

type termMemory struct {
	held map[string]graph.Term
	mu   sync.Mutex
}

func newTermMemory() *termMemory {
	return &termMemory{held: make(map[string]graph.Term)}
}

func termKey(entityID string, taxonomy graph.Taxonomy) string {
	return entityID + "/" + string(taxonomy)
}

func (m *termMemory) ListBySite(_ context.Context, siteID string) ([]graph.Term, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	out := make([]graph.Term, 0, len(m.held))
	for key := range m.held {
		if m.held[key].SiteID == siteID {
			out = append(out, m.held[key])
		}
	}
	slices.SortFunc(out, func(left, right graph.Term) int {
		return cmp.Or(
			strings.Compare(string(left.Taxonomy), string(right.Taxonomy)),
			cmp.Compare(left.TermID, right.TermID),
			strings.Compare(left.EntityID, right.EntityID),
		)
	})
	return out, nil
}

func (m *termMemory) Upsert(_ context.Context, t graph.Term) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.held[termKey(t.EntityID, t.Taxonomy)] = t
	return nil
}

func (m *termMemory) Delete(_ context.Context, entityID string, taxonomy graph.Taxonomy) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := termKey(entityID, taxonomy)
	if _, held := m.held[key]; !held {
		return errors.New(errors.NotFound, "the entity has no term in this taxonomy")
	}
	delete(m.held, key)
	return nil
}

func (m *termMemory) of(entityID string, taxonomy graph.Taxonomy) (graph.Term, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	held, found := m.held[termKey(entityID, taxonomy)]
	return held, found
}

func filedEntities() []graph.Entity {
	entities := unitEntities()
	entities[0].SiteCategory = true
	entities[0].ScopeID = pointer("drinks")
	entities[1].ScopeID = pointer("parent")
	return append([]graph.Entity{{
		ID: "drinks", SiteID: "site", Name: "Drinks", Kind: graph.KindCategory, SiteCategory: true,
		Source: graph.SourceImport,
	}}, entities...)
}

func filed(deps *steps.Deps) *termMemory {
	terms := newTermMemory()
	deps.Entities = entityList{items: filedEntities()}
	deps.Terms = terms
	return terms
}

func sameSet(left, right []int64) bool {
	return reflect.DeepEqual(sortedIDs(left), sortedIDs(right))
}

func sortedIDs(ids []int64) []int64 {
	sorted := append(make([]int64, 0, len(ids)), ids...)
	slices.Sort(sorted)
	return sorted
}

func categoryFindings(findings []content.Finding) []string {
	watched := []string{
		steps.CodeCategoriesForbidden, steps.CodeCategoryRefused, steps.CodeCategoriesNotTaken,
		steps.CodePageCategoriesNeedPlugin,
	}
	out := make([]string, 0)
	for i := range findings {
		if slices.Contains(watched, findings[i].Code) {
			out = append(out, findings[i].Code)
		}
	}
	return out
}

func siteCategory(t *testing.T, server *wptest.Server, name string, parent int64) wptest.Category {
	t.Helper()

	for _, held := range server.Categories() {
		if strings.EqualFold(held.Name, name) && held.Parent == parent {
			return held
		}
	}
	t.Fatalf("the site holds no category %q under %d: %+v", name, parent, server.Categories())
	return wptest.Category{}
}

func clientAt(t *testing.T, base string) *wp.Client {
	t.Helper()

	client, err := wp.New(wp.Config{
		BaseURL: base, Username: wptest.DefaultUser, AppPassword: wptest.DefaultPassword, AllowInsecure: true,
	}, wp.WithRateLimit(0), wp.WithBackoff(func(int) time.Duration { return 0 }))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return client
}

func behind(t *testing.T, server *wptest.Server, handle func(http.ResponseWriter, *http.Request, http.Handler)) *wp.Client {
	t.Helper()

	upstream, err := url.Parse(server.URL())
	if err != nil {
		t.Fatalf("parse the site address: %v", err)
	}
	forward := httputil.NewSingleHostReverseProxy(upstream)
	front := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handle(w, r, forward)
	}))
	t.Cleanup(front.Close)
	return clientAt(t, front.URL)
}

func failWith(t *testing.T, w http.ResponseWriter, status int) {
	t.Helper()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if _, err := w.Write([]byte(`{"code":"internal_server_error","message":"the relay failed the request"}`)); err != nil {
		t.Errorf("answer the relayed request: %v", err)
	}
}

func withoutCategories(t *testing.T, r *http.Request) {
	t.Helper()

	var body map[string]any
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		t.Errorf("read the relayed body: %v", err)
		return
	}
	delete(body, "categories")
	encoded, err := json.Marshal(body)
	if err != nil {
		t.Errorf("encode the relayed body: %v", err)
		return
	}
	r.Body = io.NopCloser(bytes.NewReader(encoded))
	r.ContentLength = int64(len(encoded))
}

func fileThePipeline(t *testing.T, p *pipeline) {
	t.Helper()

	repo := sqlite.NewEntityRepo(p.store)
	held, err := repo.ListBySite(t.Context(), p.siteID)
	if err != nil {
		t.Fatalf("list the entities: %v", err)
	}
	byName := make(map[string]graph.Entity, len(held))
	for i := range held {
		byName[held[i].Name] = held[i]
	}

	drinks, coffee, espresso := byName["Drinks"], byName["Coffee"], byName["Espresso"]
	drinks.SiteCategory = true
	coffee.SiteCategory, coffee.ScopeID = true, &drinks.ID
	espresso.ScopeID = &coffee.ID
	filedOnes := []graph.Entity{drinks, coffee, espresso}
	for i := range filedOnes {
		if err = repo.Update(t.Context(), filedOnes[i]); err != nil {
			t.Fatalf("file %s: %v", filedOnes[i].Name, err)
		}
	}
}

func TestAGenerateRunFilesThePageUnderItsCategoriesOnce(t *testing.T) {
	t.Parallel()

	p := newPipeline(t)
	fileThePipeline(t, p)
	p.start(t)

	first := p.generate(t)
	var published steps.PublishResult
	p.artifactOf(t, first.ID, run.ArtifactPublishResult, &published)
	drinks := siteCategory(t, p.server, "Drinks", 0)
	coffee := siteCategory(t, p.server, "Coffee", drinks.ID)
	if published.Categories == nil || !published.Categories.Taken ||
		!slices.Equal(published.Categories.Added, []int64{drinks.ID, coffee.ID}) {
		t.Fatalf("categories = %+v, want Drinks and Coffee", published.Categories)
	}
	if page, _ := p.server.Lookup(published.WPID); !sameSet(page.Categories, []int64{drinks.ID, coffee.ID}) {
		t.Fatalf("the draft carries %v", page.Categories)
	}

	stored, err := sqlite.NewTermRepo(p.store).ListBySite(t.Context(), p.siteID)
	if err != nil {
		t.Fatalf("list the terms: %v", err)
	}
	if len(stored) != 2 || stored[0].RunID != first.RunID || stored[1].RunID != first.RunID {
		t.Fatalf("the terms kept are %+v, want both created by run %s", stored, first.RunID)
	}

	second := p.generate(t)
	var again steps.PublishResult
	p.artifactOf(t, second.ID, run.ArtifactPublishResult, &again)
	if len(p.server.Categories()) != 2 || again.Categories == nil || len(again.Categories.Added) != 0 {
		t.Fatalf("the second run left %+v and reports %+v", p.server.Categories(), again.Categories)
	}
	for i := range again.Categories.Terms {
		if again.Categories.Terms[i].Created {
			t.Errorf("the second run reports %+v created", again.Categories.Terms[i])
		}
	}
}

func TestPublishFilesAPageUnderItsWholeCategoryChainOnce(t *testing.T) {
	t.Parallel()

	deps, server := imageDeps(t)
	terms := filed(&deps)
	sc := publishContext(t)

	first := runPublish(t, deps, sc)
	if len(server.Categories()) != 2 {
		t.Fatalf("the site holds the categories %+v, want two", server.Categories())
	}
	drinks := siteCategory(t, server, "Drinks", 0)
	coffee := siteCategory(t, server, "Coffee", drinks.ID)
	chain := []int64{drinks.ID, coffee.ID}

	written := first.Categories
	if written == nil {
		t.Fatalf("the publish result says nothing of the categories: %+v", first)
	}
	if written.Taxonomy != graph.TaxonomyCategory || !written.Taken || !slices.Equal(written.Added, chain) ||
		written.Previous == nil || len(written.Previous) != 0 {
		t.Errorf("categories = %+v, want the chain %v added to nothing and taken", written, chain)
	}
	want := []steps.AssignedTerm{
		{EntityID: "drinks", Name: "Drinks", TermID: drinks.ID, Created: true},
		{EntityID: "parent", Name: "Coffee", TermID: coffee.ID, ParentID: drinks.ID, Created: true},
	}
	if !reflect.DeepEqual(written.Terms, want) {
		t.Errorf("terms = %+v, want %+v", written.Terms, want)
	}
	if stored, _ := server.Lookup(first.WPID); !sameSet(stored.Categories, chain) {
		t.Errorf("the page is filed under %v, want %v", stored.Categories, chain)
	}
	for _, level := range want {
		kept, found := terms.of(level.EntityID, graph.TaxonomyCategory)
		if !found || kept.TermID != level.TermID || kept.ParentTermID != level.ParentID || kept.RunID != "run" ||
			kept.SiteID != "site" || kept.Name != level.Name || kept.SeenAt.IsZero() {
			t.Errorf("the term of %s is kept as %+v (%t), want %d under %d by the run", level.EntityID, kept, found,
				level.TermID, level.ParentID)
		}
	}

	server.ResetRequests()
	sc.Run.ID = "run-2"
	sc.Page.WPID = &first.WPID
	second := runPublish(t, deps, sc)
	if len(server.Categories()) != 2 {
		t.Fatalf("a second publish left the categories %+v, want the same two", server.Categories())
	}
	again := second.Categories
	if again == nil || !again.Taken || again.Added == nil || len(again.Added) != 0 || !sameSet(again.Previous, chain) {
		t.Fatalf("second categories = %+v, want nothing added over %v", again, chain)
	}
	for i := range again.Terms {
		if again.Terms[i].Created {
			t.Errorf("the second publish reports %+v created", again.Terms[i])
		}
	}
	for _, request := range server.Requests() {
		if request.Method != http.MethodPost {
			continue
		}
		if request.Path == "/wp-json/wp/v2/categories" {
			t.Errorf("the second publish created a category: %s", request.Body)
		}
		if strings.HasPrefix(request.Path, "/wp-json/wp/v2/pages/") && bytes.Contains(request.Body, []byte(`"categories"`)) {
			t.Errorf("the second publish sent the categories the page already carries: %s", request.Body)
		}
	}
	if kept, _ := terms.of("drinks", graph.TaxonomyCategory); kept.RunID != "run" {
		t.Errorf("the second publish took the term over: %+v", kept)
	}
}

func TestPublishFilesWhereTheSiteCarriesCategories(t *testing.T) {
	t.Parallel()

	older := wptest.WithCapabilities("bulk", "seo_meta", "seo_meta_read", "content_hash", "raw", "preview")
	cases := []struct {
		name     string
		opts     []wptest.Option
		wpType   pagemap.WPType
		findings []string
		seeded   bool
		filed    bool
	}{
		{name: "a post on a site without the plugin", opts: []wptest.Option{wptest.WithoutPlugin()}, wpType: pagemap.WPPost, filed: true},
		{name: "a page on a site with the plugin", wpType: pagemap.WPPage, filed: true},
		{
			name: "a page on a site without the plugin", opts: []wptest.Option{wptest.WithoutPlugin()}, wpType: pagemap.WPPage,
			findings: []string{steps.CodePageCategoriesNeedPlugin},
		},
		{
			name: "a page under a plugin older than page categories", opts: []wptest.Option{older}, wpType: pagemap.WPPage,
			findings: []string{steps.CodePageCategoriesNeedPlugin},
		},
		{
			name: "a page for a user who may not create categories", opts: []wptest.Option{wptest.WithoutTermEdit()},
			wpType: pagemap.WPPage, findings: []string{steps.CodeCategoriesForbidden},
		},
		{
			name: "a post for a user who may not create categories", opts: []wptest.Option{wptest.WithoutTermEdit()},
			wpType: pagemap.WPPost, findings: []string{steps.CodeCategoriesForbidden},
		},
		{
			name: "a page for a user who may only use the categories there are", opts: []wptest.Option{wptest.WithoutTermEdit()},
			wpType: pagemap.WPPage, seeded: true, filed: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			deps, server := imageDepsWith(t, tc.opts...)
			filed(&deps)
			if tc.seeded {
				drinks := server.SeedCategory(wptest.Category{Name: "Drinks"})
				server.SeedCategory(wptest.Category{Name: "Coffee", Parent: drinks.ID})
			}
			sc := publishContext(t)
			sc.Page.WPType = tc.wpType

			published := runPublish(t, deps, sc)
			if published.WPID == 0 {
				t.Fatalf("publish = %+v, want the item written whatever its categories", published)
			}
			if got := categoryFindings(published.Findings); !slices.Equal(got, orNone(tc.findings)) {
				t.Errorf("findings = %v, want %v", got, tc.findings)
			}
			stored, _ := server.Lookup(published.WPID)
			if !tc.filed {
				if published.Categories != nil || len(stored.Categories) != 0 {
					t.Errorf("categories = %+v and the item carries %v, want none", published.Categories, stored.Categories)
				}
				if !tc.seeded && len(server.Categories()) != 0 {
					t.Errorf("the site gained the categories %+v for an item that carries none", server.Categories())
				}
				return
			}

			drinks := siteCategory(t, server, "Drinks", 0)
			coffee := siteCategory(t, server, "Coffee", drinks.ID)
			if len(server.Categories()) != 2 || !sameSet(stored.Categories, []int64{drinks.ID, coffee.ID}) {
				t.Errorf("the item carries %v among %+v, want Drinks and Coffee", stored.Categories, server.Categories())
			}
			if published.Categories == nil || !published.Categories.Taken {
				t.Fatalf("categories = %+v, want them taken", published.Categories)
			}
			for i := range published.Categories.Terms {
				if published.Categories.Terms[i].Created == tc.seeded {
					t.Errorf("term %+v reports created %t over a site seeded %t", published.Categories.Terms[i],
						published.Categories.Terms[i].Created, tc.seeded)
				}
			}
		})
	}
}

func TestPublishAddsTheChainToTheCategoriesAnItemCarries(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		wpType  string
		held    func(*wptest.Server) []int64
		kept    []string
		created []bool
	}{
		{
			name:   "a page filed elsewhere",
			wpType: wptest.TypePage,
			held: func(server *wptest.Server) []int64 {
				return []int64{server.SeedCategory(wptest.Category{Name: "News"}).ID}
			},
			kept:    []string{"News"},
			created: []bool{true, true},
		},
		{
			name:   "a post already under the root category",
			wpType: wptest.TypePost,
			held: func(server *wptest.Server) []int64 {
				return []int64{server.SeedCategory(wptest.Category{Name: "drinks"}).ID}
			},
			created: []bool{false, true},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			deps, server := imageDeps(t)
			filed(&deps)
			previous := tc.held(server)
			item := server.Seed(wptest.Item{
				Type: tc.wpType, Title: "Espresso", Slug: "espresso", Content: "<p>before</p>", Categories: previous,
			})[0]
			sc := publishContext(t)
			sc.Page.WPType = pagemap.WPType(tc.wpType)

			published := runPublish(t, deps, sc)
			if published.Created || published.WPID != item.ID {
				t.Fatalf("publish = %+v, want an update of %d", published, item.ID)
			}
			drinks := siteCategory(t, server, "drinks", 0)
			coffee := siteCategory(t, server, "Coffee", drinks.ID)
			written := published.Categories
			if written == nil || !written.Taken || !slices.Equal(written.Previous, previous) {
				t.Fatalf("categories = %+v, want the item's own %v kept as previous", written, previous)
			}
			added := []int64{drinks.ID, coffee.ID}
			if !tc.created[0] {
				added = added[1:]
			}
			if !slices.Equal(written.Added, added) {
				t.Errorf("added = %v, want exactly %v", written.Added, added)
			}
			for i := range written.Terms {
				if written.Terms[i].Created != tc.created[i] {
					t.Errorf("term %+v reports created %t, want %t", written.Terms[i], written.Terms[i].Created, tc.created[i])
				}
			}
			stored, _ := server.Lookup(item.ID)
			if !sameSet(stored.Categories, append(slices.Clone(previous), added...)) {
				t.Errorf("the item carries %v, want %v with %v", stored.Categories, previous, added)
			}
			for _, name := range tc.kept {
				if held := siteCategory(t, server, name, 0); !slices.Contains(stored.Categories, held.ID) {
					t.Errorf("the item lost its own category %s", name)
				}
			}
		})
	}
}

func TestPublishSaysSoWhenTheSiteDropsTheCategories(t *testing.T) {
	t.Parallel()

	deps, server := imageDeps(t)
	filed(&deps)
	deps.WordPress = oneClient{client: behind(t, server, func(w http.ResponseWriter, r *http.Request, forward http.Handler) {
		if r.Method == http.MethodPost && strings.HasPrefix(r.URL.Path, "/wp-json/wp/v2/pages") {
			withoutCategories(t, r)
		}
		forward.ServeHTTP(w, r)
	})}

	published := runPublish(t, deps, publishContext(t))
	if published.WPID == 0 {
		t.Fatalf("publish = %+v, want the page written", published)
	}
	if published.Categories == nil || published.Categories.Taken {
		t.Errorf("categories = %+v, want them reported as not taken", published.Categories)
	}
	if got := categoryFindings(published.Findings); !slices.Equal(got, []string{steps.CodeCategoriesNotTaken}) {
		t.Errorf("findings = %v, want %s", got, steps.CodeCategoriesNotTaken)
	}
}

func TestPublishRetriedAfterAFaultMidChainCreatesNoTermTwice(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		created []bool
		written bool
	}{
		{name: "a create refused before it was written", created: []bool{true, true}},
		{name: "a create written and its answer lost", written: true, created: []bool{true, false}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			deps, server := imageDeps(t)
			filed(&deps)
			var creates atomic.Int32
			deps.WordPress = oneClient{client: behind(t, server, func(w http.ResponseWriter, r *http.Request, forward http.Handler) {
				if r.Method == http.MethodPost && r.URL.Path == "/wp-json/wp/v2/categories" && creates.Add(1) == 2 {
					if tc.written {
						forward.ServeHTTP(httptest.NewRecorder(), r)
					}
					failWith(t, w, http.StatusInternalServerError)
					return
				}
				forward.ServeHTTP(w, r)
			})}

			sc := publishContext(t)
			if _, err := steps.Publish(deps).Run(t.Context(), sc); !errors.IsCode(err, errors.External) {
				t.Fatalf("Publish over a failing site = %v, want the fault handed to the step's retry", err)
			}
			if items := server.Items(); len(items) != 0 {
				t.Fatalf("the failed attempt wrote %+v, want nothing written before the categories", items)
			}

			published := runPublish(t, deps, sc)
			if len(server.Categories()) != 2 {
				t.Fatalf("the site holds %+v after the retry, want each category once", server.Categories())
			}
			if published.Categories == nil || !published.Categories.Taken || len(published.Categories.Terms) != 2 {
				t.Fatalf("categories = %+v", published.Categories)
			}
			for i, level := range published.Categories.Terms {
				if level.Created != tc.created[i] {
					t.Errorf("term %+v reports created %t, want %t", level, level.Created, tc.created[i])
				}
			}
		})
	}
}
