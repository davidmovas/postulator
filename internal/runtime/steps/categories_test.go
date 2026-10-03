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
	"github.com/davidmovas/postulator/internal/adapters/sqlite/sqlitetest"
	"github.com/davidmovas/postulator/internal/adapters/wp"
	"github.com/davidmovas/postulator/internal/adapters/wp/wptest"
	"github.com/davidmovas/postulator/internal/domain/category"
	"github.com/davidmovas/postulator/internal/domain/content"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/run"
	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/kernel/id"
	"github.com/davidmovas/postulator/internal/runtime/steps"
)

const (
	categoryDrinks = "cat-drinks"
	categoryCoffee = "cat-coffee"
)

type categoryList struct {
	err   error
	items []category.Category
}

func (c categoryList) ListBySite(context.Context, string) ([]category.Category, error) {
	return c.items, c.err
}

type termMemory struct {
	held map[string]category.Term
	mu   sync.Mutex
}

func newTermMemory() *termMemory {
	return &termMemory{held: make(map[string]category.Term)}
}

func termKey(categoryID string, taxonomy category.Taxonomy) string {
	return categoryID + "/" + string(taxonomy)
}

func (m *termMemory) ListBySite(_ context.Context, siteID string) ([]category.Term, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	out := make([]category.Term, 0, len(m.held))
	for key := range m.held {
		if m.held[key].SiteID == siteID {
			out = append(out, m.held[key])
		}
	}
	slices.SortFunc(out, func(left, right category.Term) int {
		return cmp.Or(
			strings.Compare(string(left.Taxonomy), string(right.Taxonomy)),
			cmp.Compare(left.TermID, right.TermID),
			strings.Compare(left.CategoryID, right.CategoryID),
		)
	})
	return out, nil
}

func (m *termMemory) Upsert(_ context.Context, t category.Term) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.held[termKey(t.CategoryID, t.Taxonomy)] = t
	return nil
}

func (m *termMemory) Delete(_ context.Context, categoryID string, taxonomy category.Taxonomy) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := termKey(categoryID, taxonomy)
	if _, held := m.held[key]; !held {
		return errors.New(errors.NotFound, "the category has no term in this taxonomy")
	}
	delete(m.held, key)
	return nil
}

func (m *termMemory) of(categoryID string, taxonomy category.Taxonomy) (category.Term, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	held, found := m.held[termKey(categoryID, taxonomy)]
	return held, found
}

func filedCategories() []category.Category {
	return []category.Category{
		{ID: categoryCoffee, SiteID: "site", Name: "Coffee", Key: category.Key("Coffee"), ParentID: categoryDrinks},
		{ID: categoryDrinks, SiteID: "site", Name: "Drinks", Key: category.Key("Drinks")},
	}
}

func filed(deps *steps.Deps) *termMemory {
	terms := newTermMemory()
	deps.Categories = categoryList{items: filedCategories()}
	deps.CategoryTerms = terms
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
	refuseWith(t, w, status, "internal_server_error", "the relay failed the request")
}

func refuseWith(t *testing.T, w http.ResponseWriter, status int, code, message string) {
	t.Helper()

	body, err := json.Marshal(map[string]any{"code": code, "message": message, "data": map[string]any{"status": status}})
	if err != nil {
		t.Errorf("encode the relayed refusal: %v", err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if _, err = w.Write(body); err != nil {
		t.Errorf("answer the relayed request: %v", err)
	}
}

type failingTerms struct {
	*termMemory
	err error
}

func (f failingTerms) Upsert(context.Context, category.Term) error {
	return f.err
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

func storedCategory(t *testing.T, store *sqlite.Store, siteID, name, parentID string) category.Category {
	t.Helper()

	record, err := category.New(category.Category{
		ID: id.New(), SiteID: siteID, Name: name, ParentID: parentID, CreatedAt: sqlitetest.Stamp, UpdatedAt: sqlitetest.Stamp,
	})
	if err != nil {
		t.Fatalf("the category %s: %v", name, err)
	}
	if err = sqlite.NewCategoryRepo(store).Insert(t.Context(), record); err != nil {
		t.Fatalf("insert the category %s: %v", name, err)
	}
	return record
}

func fileThePipeline(t *testing.T, p *pipeline) {
	t.Helper()

	drinks := storedCategory(t, p.store, p.siteID, "Drinks", "")
	coffee := storedCategory(t, p.store, p.siteID, "Coffee", drinks.ID)
	espresso, err := p.pages.Get(t.Context(), p.pageID)
	if err != nil {
		t.Fatalf("read the page: %v", err)
	}
	espresso.CategoryID = coffee.ID
	if err = p.pages.Update(t.Context(), espresso); err != nil {
		t.Fatalf("file the page under %s: %v", coffee.Name, err)
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

	stored, err := sqlite.NewCategoryTermRepo(p.store).ListBySite(t.Context(), p.siteID)
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
	if written.Taxonomy != category.TaxonomyCategory || !written.Taken || !slices.Equal(written.Added, chain) ||
		written.Previous == nil || len(written.Previous) != 0 {
		t.Errorf("categories = %+v, want the chain %v added to nothing and taken", written, chain)
	}
	want := []steps.AssignedTerm{
		{CategoryID: categoryDrinks, Name: "Drinks", TermID: drinks.ID, Created: true},
		{CategoryID: categoryCoffee, Name: "Coffee", TermID: coffee.ID, ParentID: drinks.ID, Created: true},
	}
	if !reflect.DeepEqual(written.Terms, want) {
		t.Errorf("terms = %+v, want %+v", written.Terms, want)
	}
	if stored, _ := server.Lookup(first.WPID); !sameSet(stored.Categories, chain) {
		t.Errorf("the page is filed under %v, want %v", stored.Categories, chain)
	}
	for _, level := range want {
		kept, found := terms.of(level.CategoryID, category.TaxonomyCategory)
		if !found || kept.TermID != level.TermID || kept.ParentTermID != level.ParentID || kept.RunID != "run" ||
			kept.SiteID != "site" || kept.Name != level.Name || kept.SeenAt.IsZero() {
			t.Errorf("the term of %s is kept as %+v (%t), want %d under %d by the run", level.CategoryID, kept, found,
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
	if kept, _ := terms.of(categoryDrinks, category.TaxonomyCategory); kept.RunID != "run" {
		t.Errorf("the second publish took the term over: %+v", kept)
	}
}

func TestPublishFilesAPageByItsCategoryWhateverItsEntity(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name       string
		entityID   *string
		categoryID string
		filed      bool
	}{
		{name: "a page with no entity under a category", categoryID: categoryCoffee, filed: true},
		{name: "a page whose entity is gone under a category", entityID: pointer("deleted"), categoryID: categoryCoffee, filed: true},
		{name: "a page with an entity and no category", entityID: pointer("child")},
		{name: "a page under a category the site no longer holds", entityID: pointer("child"), categoryID: "cat-gone"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			deps, server := imageDeps(t)
			terms := filed(&deps)
			sc := publishContext(t)
			sc.Page.EntityID = tc.entityID
			sc.Page.CategoryID = tc.categoryID

			published := runPublish(t, deps, sc)
			stored, _ := server.Lookup(published.WPID)
			if !tc.filed {
				held, err := terms.ListBySite(t.Context(), "site")
				if err != nil {
					t.Fatalf("list the terms: %v", err)
				}
				if published.Categories != nil || len(stored.Categories) != 0 || len(server.Categories()) != 0 || len(held) != 0 {
					t.Fatalf("categories = %+v, the page carries %v among %+v and %+v is kept; want nothing filed",
						published.Categories, stored.Categories, server.Categories(), held)
				}
				return
			}
			drinks := siteCategory(t, server, "Drinks", 0)
			coffee := siteCategory(t, server, "Coffee", drinks.ID)
			if published.Categories == nil || !published.Categories.Taken ||
				!sameSet(stored.Categories, []int64{drinks.ID, coffee.ID}) {
				t.Fatalf("categories = %+v and the page carries %v, want Drinks and Coffee", published.Categories, stored.Categories)
			}
			if kept, found := terms.of(categoryCoffee, category.TaxonomyCategory); !found || kept.TermID != coffee.ID {
				t.Errorf("the term of Coffee is kept as %+v (%t), want %d", kept, found, coffee.ID)
			}
		})
	}
}

func TestThePublishResultNamesTheCategoryOfEachTerm(t *testing.T) {
	t.Parallel()

	deps, _ := imageDeps(t)
	filed(&deps)
	result, err := steps.Publish(deps).Run(t.Context(), publishContext(t))
	if err != nil || len(result.Artifacts) != 1 {
		t.Fatalf("Publish = %+v, %v", result, err)
	}

	var shape struct {
		Categories struct {
			Terms []map[string]any `json:"terms"`
		} `json:"categories"`
	}
	if err = json.Unmarshal(result.Artifacts[0].Blob, &shape); err != nil {
		t.Fatalf("decode the publish result: %v", err)
	}
	if len(shape.Categories.Terms) != 2 {
		t.Fatalf("terms = %+v, want two", shape.Categories.Terms)
	}
	for i, want := range []string{categoryDrinks, categoryCoffee} {
		term := shape.Categories.Terms[i]
		keys := make([]string, 0, len(term))
		for key := range term {
			keys = append(keys, key)
		}
		slices.Sort(keys)
		if !slices.Equal(keys, []string{"categoryId", "created", "name", "parentId", "termId"}) || term["categoryId"] != want {
			t.Errorf("term %d = %+v, want the fields of a term filed for the category %s", i, term, want)
		}
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

func TestPublishWritesAPageWordPressWillNotFileAndSaysWhy(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		code    string
		message string
		status  int
	}{
		{name: "a name WordPress rejects", status: http.StatusBadRequest, code: "rest_invalid_param", message: "Invalid parameter(s): name"},
		{name: "a name that sanitizes to nothing", status: http.StatusInternalServerError, code: "empty_term_name", message: "A name is required for this term."},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			deps, server := imageDeps(t)
			terms := filed(&deps)
			deps.WordPress = oneClient{client: behind(t, server, func(w http.ResponseWriter, r *http.Request, forward http.Handler) {
				if r.Method == http.MethodPost && r.URL.Path == "/wp-json/wp/v2/categories" {
					refuseWith(t, w, tc.status, tc.code, tc.message)
					return
				}
				forward.ServeHTTP(w, r)
			})}

			published := runPublish(t, deps, publishContext(t))
			if published.WPID == 0 || published.Categories != nil {
				t.Fatalf("publish = %+v, want the page written without categories", published)
			}
			if got := categoryFindings(published.Findings); !slices.Equal(got, []string{steps.CodeCategoryRefused}) {
				t.Fatalf("findings = %v, want %s", got, steps.CodeCategoryRefused)
			}
			for i := range published.Findings {
				if published.Findings[i].Code == steps.CodeCategoryRefused && !strings.Contains(published.Findings[i].Message, tc.message) {
					t.Errorf("the finding reads %q, want WordPress's own words %q", published.Findings[i].Message, tc.message)
				}
			}
			if held, err := terms.ListBySite(t.Context(), "site"); err != nil || len(held) != 0 {
				t.Errorf("the refused walk kept %+v (%v)", held, err)
			}
		})
	}
}

func TestPublishNamesAStoreItWasNotGiven(t *testing.T) {
	t.Parallel()

	noTerms := func(deps *steps.Deps) { deps.CategoryTerms = nil }
	noCategories := func(deps *steps.Deps) { deps.Categories = nil }
	cases := []struct {
		name       string
		missing    func(*steps.Deps)
		says       string
		categoryID string
		filed      bool
	}{
		{name: "no term store for a page with categories", missing: noTerms, categoryID: categoryCoffee, filed: true, says: "term store"},
		{name: "no term store for a page under a category the site lacks", missing: noTerms, categoryID: categoryCoffee},
		{name: "no term store for a page with no category", missing: noTerms, filed: true},
		{name: "no category reader for a page with a category", missing: noCategories, categoryID: categoryCoffee, filed: true, says: "category reader"},
		{name: "no category reader for a page with no category", missing: noCategories, filed: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			deps, server := imageDeps(t)
			if tc.filed {
				filed(&deps)
			}
			tc.missing(&deps)
			sc := publishContext(t)
			sc.Page.CategoryID = tc.categoryID

			_, err := steps.Publish(deps).Run(t.Context(), sc)
			if tc.says == "" {
				if err != nil || len(server.Items()) != 1 {
					t.Fatalf("Publish = %v over %+v, want a page that needs no such store written", err, server.Items())
				}
				return
			}
			if !errors.IsCode(err, errors.Internal) || !strings.Contains(err.Error(), tc.says) {
				t.Fatalf("Publish = %v, want an error naming the missing %s", err, tc.says)
			}
			if len(server.Items()) != 0 || len(server.Categories()) != 0 {
				t.Errorf("the site holds %+v and %+v, want nothing written", server.Items(), server.Categories())
			}
		})
	}
}

func TestPublishStopsWhenItCannotKeepATerm(t *testing.T) {
	t.Parallel()

	deps, server := imageDeps(t)
	terms := filed(&deps)
	deps.CategoryTerms = failingTerms{termMemory: terms, err: errors.New(errors.Invalid, "the term row was refused")}

	_, err := steps.Publish(deps).Run(t.Context(), publishContext(t))
	if !errors.IsCode(err, errors.Invalid) {
		t.Fatalf("Publish = %v, want the store's refusal handed back rather than told as WordPress's", err)
	}
	if len(server.Items()) != 0 {
		t.Errorf("the site holds %+v, want nothing written past a term the run could not keep", server.Items())
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
