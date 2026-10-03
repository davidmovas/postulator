package wp_test

import (
	"encoding/json"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/davidmovas/postulator/internal/adapters/wp"
	"github.com/davidmovas/postulator/internal/adapters/wp/wptest"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

var taxonomies = []wp.Taxonomy{wp.TaxonomyCategory, wp.TaxonomyProductCategory}

func seedTerm(server *wptest.Server, taxonomy wp.Taxonomy, name string, parent int64) int64 {
	if taxonomy == wp.TaxonomyProductCategory {
		return server.Seed(wptest.Item{Type: wptest.TypeProductCategory, Title: name, Parent: parent})[0].ID
	}
	return server.SeedCategory(wptest.Category{Name: name, Parent: parent}).ID
}

func termRoute(taxonomy wp.Taxonomy) string {
	if taxonomy == wp.TaxonomyProductCategory {
		return "/wp-json/wc/v3/products/categories"
	}
	return "/wp-json/wp/v2/categories"
}

func sentBody(t *testing.T, recorded wptest.Request) map[string]any {
	t.Helper()

	var body map[string]any
	if err := json.Unmarshal(recorded.Body, &body); err != nil {
		t.Fatalf("decode the sent body %s: %v", recorded.Body, err)
	}
	return body
}

func requestsTo(server *wptest.Server, method, path string) int {
	count := 0
	for _, recorded := range server.Requests() {
		if recorded.Method == method && recorded.Path == path {
			count++
		}
	}
	return count
}

func termIDs(terms []wp.Term) []int64 {
	ids := make([]int64, 0, len(terms))
	for _, term := range terms {
		ids = append(ids, term.ID)
	}
	return ids
}

type termTree struct {
	koffein int64
	rd      int64
	pills   int64
	tee     int64
}

func seedTree(server *wptest.Server, taxonomy wp.Taxonomy) termTree {
	koffein := seedTerm(server, taxonomy, "Koffein", 0)
	return termTree{
		koffein: koffein,
		rd:      seedTerm(server, taxonomy, "R&D", koffein),
		pills:   seedTerm(server, taxonomy, "Pills", koffein),
		tee:     seedTerm(server, taxonomy, "Tee", 0),
	}
}

func TestListTermsReadsEitherTaxonomy(t *testing.T) {
	t.Parallel()

	zero := int64(0)
	cases := []struct {
		query func(tree termTree) wp.TermQuery
		want  func(tree termTree) []int64
		name  string
		total int
		more  bool
	}{
		{
			name:  "every term in id order",
			query: func(termTree) wp.TermQuery { return wp.TermQuery{} },
			want:  func(tree termTree) []int64 { return []int64{tree.koffein, tree.rd, tree.pills, tree.tee} },
			total: 4,
		},
		{
			name:  "one parent's children",
			query: func(tree termTree) wp.TermQuery { return wp.TermQuery{Parent: &tree.koffein} },
			want:  func(tree termTree) []int64 { return []int64{tree.rd, tree.pills} },
			total: 2,
		},
		{
			name:  "the top level",
			query: func(termTree) wp.TermQuery { return wp.TermQuery{Parent: &zero} },
			want:  func(tree termTree) []int64 { return []int64{tree.koffein, tree.tee} },
			total: 2,
		},
		{
			name:  "the stored ids",
			query: func(tree termTree) wp.TermQuery { return wp.TermQuery{Include: []int64{tree.tee, tree.rd, 999}} },
			want:  func(tree termTree) []int64 { return []int64{tree.rd, tree.tee} },
			total: 2,
		},
		{
			name:  "one page of several",
			query: func(termTree) wp.TermQuery { return wp.TermQuery{Page: 2, PerPage: 1} },
			want:  func(tree termTree) []int64 { return []int64{tree.rd} },
			total: 4,
			more:  true,
		},
		{
			name:  "past the end",
			query: func(termTree) wp.TermQuery { return wp.TermQuery{Page: 9, PerPage: 2} },
			want:  func(termTree) []int64 { return []int64{} },
			total: 4,
		},
	}

	for _, taxonomy := range taxonomies {
		for _, tc := range cases {
			t.Run(string(taxonomy)+"/"+tc.name, func(t *testing.T) {
				t.Parallel()

				server := wptest.New(t)
				tree := seedTree(server, taxonomy)

				page, err := newClient(t, server).ListTerms(t.Context(), taxonomy, tc.query(tree))
				if err != nil {
					t.Fatalf("ListTerms: %v", err)
				}
				if got := termIDs(page.Items); !slices.Equal(got, tc.want(tree)) {
					t.Errorf("ids = %v, want %v", got, tc.want(tree))
				}
				if page.Total != tc.total || page.HasMore != tc.more {
					t.Errorf("total = %d, more = %t; want %d, %t", page.Total, page.HasMore, tc.total, tc.more)
				}
				for _, term := range page.Items {
					if term.ID == tree.rd && (term.Name != "R&D" || term.Parent != tree.koffein || term.Slug != "r-d") {
						t.Errorf("the child reads %+v, want R&D under %d", term, tree.koffein)
					}
				}

				recorded, _ := server.LastRequest()
				if recorded.Path != termRoute(taxonomy) {
					t.Errorf("path = %q, want %q", recorded.Path, termRoute(taxonomy))
				}
			})
		}
	}
}

func TestListTermsAsksForWhatItNeeds(t *testing.T) {
	t.Parallel()

	parent := int64(7)
	cases := []struct {
		name    string
		query   wp.TermQuery
		page    string
		perPage string
		parent  string
		include string
	}{
		{name: "the defaults", query: wp.TermQuery{}, page: "1", perPage: "50"},
		{name: "a wide page is clamped", query: wp.TermQuery{PerPage: 500, Page: 3}, page: "3", perPage: "100"},
		{name: "a parent", query: wp.TermQuery{Parent: &parent}, page: "1", perPage: "50", parent: "7"},
		{name: "some ids", query: wp.TermQuery{Include: []int64{3, 1}}, page: "1", perPage: "50", include: "3,1"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			server := wptest.New(t)
			if _, err := newClient(t, server).ListTerms(t.Context(), wp.TaxonomyCategory, tc.query); err != nil {
				t.Fatalf("ListTerms: %v", err)
			}

			recorded, _ := server.LastRequest()
			query := recorded.Query
			if query.Get("page") != tc.page || query.Get("per_page") != tc.perPage {
				t.Errorf("page = %q, per_page = %q; want %q, %q", query.Get("page"), query.Get("per_page"), tc.page, tc.perPage)
			}
			if query.Get("orderby") != "id" || query.Get("order") != "asc" {
				t.Errorf("order = %q %q, want id asc", query.Get("orderby"), query.Get("order"))
			}
			if query.Get("parent") != tc.parent || query.Get("include") != tc.include {
				t.Errorf("parent = %q, include = %q; want %q, %q", query.Get("parent"), query.Get("include"), tc.parent, tc.include)
			}
			if query.Has("context") || query.Has("search") || query.Has("slug") {
				t.Errorf("query = %v; terms are read without an edit context or a name filter", query)
			}
		})
	}
}

func TestGetTermReadsOneTerm(t *testing.T) {
	t.Parallel()

	for _, taxonomy := range taxonomies {
		t.Run(string(taxonomy), func(t *testing.T) {
			t.Parallel()

			server := wptest.New(t)
			tree := seedTree(server, taxonomy)
			client := newClient(t, server)

			term, err := client.GetTerm(t.Context(), taxonomy, tree.rd)
			if err != nil {
				t.Fatalf("GetTerm: %v", err)
			}
			if term.ID != tree.rd || term.Name != "R&D" || term.Parent != tree.koffein {
				t.Errorf("term = %+v", term)
			}

			if _, err := client.GetTerm(t.Context(), taxonomy, 999); !errors.IsCode(err, errors.NotFound) {
				t.Errorf("code = %q, want %q", errors.CodeOf(err), errors.NotFound)
			}
		})
	}
}

func TestCreateTermSendsANameAndAParentOnly(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		termName string
		under    bool
		keys     []string
		want     string
		slug     string
	}{
		{name: "a top level term", termName: "Tee", keys: []string{"name"}, want: "Tee", slug: "tee"},
		{name: "a child", termName: "Powder", under: true, keys: []string{"name", "parent"}, want: "Powder", slug: "powder"},
		{name: "a name WordPress escapes", termName: "Salt & Pepper", keys: []string{"name"}, want: "Salt & Pepper", slug: "salt-pepper"},
	}

	for _, taxonomy := range taxonomies {
		for _, tc := range cases {
			t.Run(string(taxonomy)+"/"+tc.name, func(t *testing.T) {
				t.Parallel()

				server := wptest.New(t)
				koffein := seedTerm(server, taxonomy, "Koffein", 0)
				parent := int64(0)
				if tc.under {
					parent = koffein
				}

				term, err := newClient(t, server).CreateTerm(t.Context(), taxonomy, tc.termName, parent)
				if err != nil {
					t.Fatalf("CreateTerm: %v", err)
				}
				if term.ID == 0 || term.Name != tc.want || term.Slug != tc.slug || term.Parent != parent {
					t.Errorf("term = %+v, want %q (%s) under %d", term, tc.want, tc.slug, parent)
				}

				recorded, _ := server.LastRequest()
				body := sentBody(t, recorded)
				keys := make([]string, 0, len(body))
				for key := range body {
					keys = append(keys, key)
				}
				slices.Sort(keys)
				if !slices.Equal(keys, tc.keys) {
					t.Errorf("sent %v, want only %v", keys, tc.keys)
				}
				if body["name"] != tc.termName {
					t.Errorf("sent name %v, want %q", body["name"], tc.termName)
				}
			})
		}
	}
}

func TestCreateTermSaysWhyWordPressRefusedIt(t *testing.T) {
	t.Parallel()

	cases := []struct {
		check   func(t *testing.T, err error, existing int64)
		name    string
		options []wptest.Option
		term    string
		parent  int64
		code    map[wp.Taxonomy]errors.Code
		sent    bool
	}{
		{
			name: "an empty name",
			term: " \t ",
			code: map[wp.Taxonomy]errors.Code{wp.TaxonomyCategory: errors.Invalid, wp.TaxonomyProductCategory: errors.Invalid},
		},
		{
			name: "a name the parent already has",
			term: "koffein",
			code: map[wp.Taxonomy]errors.Code{wp.TaxonomyCategory: errors.Invalid, wp.TaxonomyProductCategory: errors.Invalid},
			sent: true,
			check: func(t *testing.T, err error, existing int64) {
				t.Helper()
				if id, exists := wp.TermExists(err); !exists || id != existing {
					t.Errorf("TermExists = %d, %t; want %d", id, exists, existing)
				}
			},
		},
		{
			name:   "a parent that does not exist",
			term:   "Pulver",
			parent: 999,
			code:   map[wp.Taxonomy]errors.Code{wp.TaxonomyCategory: errors.Invalid, wp.TaxonomyProductCategory: errors.NotFound},
			sent:   true,
			check: func(t *testing.T, err error, _ int64) {
				t.Helper()
				if _, exists := wp.TermExists(err); exists {
					t.Error("a missing parent is not a duplicate")
				}
			},
		},
		{
			name:    "a user who may not create terms",
			options: []wptest.Option{wptest.WithoutTermEdit()},
			term:    "Pulver",
			code:    map[wp.Taxonomy]errors.Code{wp.TaxonomyCategory: errors.Unauthorized, wp.TaxonomyProductCategory: errors.Unauthorized},
			sent:    true,
			check: func(t *testing.T, err error, _ int64) {
				t.Helper()
				if !wp.Forbidden(err) {
					t.Error("a refused permission must be told apart from a refused password")
				}
			},
		},
	}

	for _, taxonomy := range taxonomies {
		for _, tc := range cases {
			t.Run(string(taxonomy)+"/"+tc.name, func(t *testing.T) {
				t.Parallel()

				server := wptest.New(t, tc.options...)
				existing := seedTerm(server, taxonomy, "Koffein", 0)
				server.ResetRequests()

				_, err := newClient(t, server).CreateTerm(t.Context(), taxonomy, tc.term, tc.parent)
				if want := tc.code[taxonomy]; !errors.IsCode(err, want) {
					t.Fatalf("code = %q, want %q: %v", errors.CodeOf(err), want, err)
				}
				if sent := len(server.Requests()) > 0; sent != tc.sent {
					t.Errorf("a request was sent = %t, want %t", sent, tc.sent)
				}
				if tc.check != nil {
					tc.check(t, err, existing)
				}
			})
		}
	}
}

func TestFindTermMatchesTheNameWordPressStores(t *testing.T) {
	t.Parallel()

	cases := []struct {
		want   func(tree termTree) int64
		name   string
		term   string
		parent func(tree termTree) int64
	}{
		{name: "the same name", term: "Koffein", want: func(tree termTree) int64 { return tree.koffein }},
		{name: "another case", term: "KOFFEIN", want: func(tree termTree) int64 { return tree.koffein }},
		{name: "padded and doubled spaces", term: "  Koffein ", want: func(tree termTree) int64 { return tree.koffein }},
		{
			name:   "an ampersand WordPress stored as an entity",
			term:   "r&d",
			parent: func(tree termTree) int64 { return tree.koffein },
			want:   func(tree termTree) int64 { return tree.rd },
		},
		{
			name:   "the entity itself",
			term:   "R&amp;D",
			parent: func(tree termTree) int64 { return tree.koffein },
			want:   func(tree termTree) int64 { return tree.rd },
		},
		{name: "a child under the wrong parent", term: "Pills"},
		{name: "a name nobody carries", term: "Kaffee"},
	}

	for _, taxonomy := range taxonomies {
		for _, tc := range cases {
			t.Run(string(taxonomy)+"/"+tc.name, func(t *testing.T) {
				t.Parallel()

				server := wptest.New(t)
				tree := seedTree(server, taxonomy)
				parent := int64(0)
				if tc.parent != nil {
					parent = tc.parent(tree)
				}

				term, found, err := newClient(t, server).FindTerm(t.Context(), taxonomy, tc.term, parent)
				if err != nil {
					t.Fatalf("FindTerm: %v", err)
				}
				if tc.want == nil {
					if found {
						t.Errorf("found %+v, want nothing", term)
					}
					return
				}
				if !found || term.ID != tc.want(tree) || term.Parent != parent {
					t.Errorf("found = %t, term = %+v; want %d under %d", found, term, tc.want(tree), parent)
				}

				recorded, _ := server.LastRequest()
				if recorded.Query.Get("parent") != strconv.FormatInt(parent, 10) {
					t.Errorf("parent = %q; a name is looked for among one parent's children", recorded.Query.Get("parent"))
				}
			})
		}
	}
}

func TestFindTermPagesThroughEveryChild(t *testing.T) {
	t.Parallel()

	for _, taxonomy := range taxonomies {
		t.Run(string(taxonomy), func(t *testing.T) {
			t.Parallel()

			server := wptest.New(t)
			parent := seedTerm(server, taxonomy, "Koffein", 0)
			last := int64(0)
			for index := range 120 {
				last = seedTerm(server, taxonomy, "Child "+strconv.Itoa(index), parent)
			}
			server.ResetRequests()

			term, found, err := newClient(t, server).FindTerm(t.Context(), taxonomy, "child 119", parent)
			if err != nil {
				t.Fatalf("FindTerm: %v", err)
			}
			if !found || term.ID != last {
				t.Errorf("found = %t, term = %+v; want %d", found, term, last)
			}
			if got := len(server.Requests()); got != 2 {
				t.Errorf("the search took %d requests, want two pages of 100", got)
			}

			if _, found, err = newClient(t, server).FindTerm(t.Context(), taxonomy, "", parent); !errors.IsCode(err, errors.Invalid) || found {
				t.Errorf("an empty name: found = %t, code = %q; want %q", found, errors.CodeOf(err), errors.Invalid)
			}
		})
	}
}

func TestEnsureTermReusesCreatesOrAdoptsWhatWordPressHas(t *testing.T) {
	t.Parallel()

	cases := []struct {
		options []wptest.Option
		name    string
		term    string
		created bool
		reused  bool
		posts   int
		code    errors.Code
	}{
		{name: "a term the parent has", term: "koffein", reused: true},
		{name: "a term the parent lacks", term: "Tee", created: true, posts: 1},
		{name: "a name WordPress reduces to one it has", term: "<b>Koffein</b>", reused: true, posts: 1},
		{name: "a user who may not create terms", options: []wptest.Option{wptest.WithoutTermEdit()}, term: "Tee", posts: 1, code: errors.Unauthorized},
	}

	for _, taxonomy := range taxonomies {
		for _, tc := range cases {
			t.Run(string(taxonomy)+"/"+tc.name, func(t *testing.T) {
				t.Parallel()

				server := wptest.New(t, tc.options...)
				existing := seedTerm(server, taxonomy, "Koffein", 0)
				server.ResetRequests()

				term, created, err := newClient(t, server).EnsureTerm(t.Context(), taxonomy, tc.term, 0)
				if tc.code != "" {
					if !errors.IsCode(err, tc.code) || !wp.Forbidden(err) {
						t.Fatalf("code = %q, want a refused permission: %v", errors.CodeOf(err), err)
					}
					return
				}
				if err != nil {
					t.Fatalf("EnsureTerm: %v", err)
				}
				if created != tc.created {
					t.Errorf("created = %t, want %t", created, tc.created)
				}
				if tc.reused && term.ID != existing {
					t.Errorf("term = %+v, want the existing %d", term, existing)
				}
				if tc.created && (term.ID == existing || term.Name != tc.term) {
					t.Errorf("term = %+v, want a new %q", term, tc.term)
				}
				if got := requestsTo(server, http.MethodPost, termRoute(taxonomy)); got != tc.posts {
					t.Errorf("sent %d creates, want %d", got, tc.posts)
				}
			})
		}
	}
}

func TestTheTermMethodsKnowTwoTaxonomies(t *testing.T) {
	t.Parallel()

	server := wptest.New(t)
	client := newClient(t, server)
	unknown := wp.Taxonomy("post_tag")

	calls := map[string]func() error{
		"list":   func() error { _, err := client.ListTerms(t.Context(), unknown, wp.TermQuery{}); return err },
		"get":    func() error { _, err := client.GetTerm(t.Context(), unknown, 1); return err },
		"create": func() error { _, err := client.CreateTerm(t.Context(), unknown, "Koffein", 0); return err },
		"find":   func() error { _, _, err := client.FindTerm(t.Context(), unknown, "Koffein", 0); return err },
		"ensure": func() error { _, _, err := client.EnsureTerm(t.Context(), unknown, "Koffein", 0); return err },
	}
	for name, call := range calls {
		if err := call(); !errors.IsCode(err, errors.Invalid) {
			t.Errorf("%s: code = %q, want %q", name, errors.CodeOf(err), errors.Invalid)
		}
	}
	if got := len(server.Requests()); got != 0 {
		t.Errorf("an unknown taxonomy sent %d requests", got)
	}
}

func TestSameTermNameComparesWhatAPersonReads(t *testing.T) {
	t.Parallel()

	cases := []struct {
		left  string
		right string
		same  bool
	}{
		{left: "Koffein", right: "Koffein", same: true},
		{left: "Koffein", right: "kOFFEIN", same: true},
		{left: " Koffein\t", right: "Koffein", same: true},
		{left: "Koffein  Pulver", right: "Koffein Pulver", same: true},
		{left: "R&amp;D", right: "R&D", same: true},
		{left: "&lt;Lab&gt;", right: "<lab>", same: true},
		{left: "Café", right: "CAFÉ", same: true},
		{left: "Koffein", right: "Koffeine"},
		{left: "Koffein Pulver", right: "KoffeinPulver"},
		{left: "", right: " "},
	}

	for _, tc := range cases {
		t.Run(tc.left+" | "+tc.right, func(t *testing.T) {
			t.Parallel()

			if got := wp.SameTermName(tc.left, tc.right); got != tc.same {
				t.Errorf("SameTermName(%q, %q) = %t, want %t", tc.left, tc.right, got, tc.same)
			}
			if got := wp.SameTermName(tc.right, tc.left); got != tc.same {
				t.Errorf("SameTermName is not symmetric for %q and %q", tc.left, tc.right)
			}
		})
	}
}

func TestATermNameIsReadWithoutItsEntities(t *testing.T) {
	t.Parallel()

	server := wptest.New(t)
	id := server.SeedCategory(wptest.Category{Name: "Tee &amp; Kaffee &#8211; Mehr"}).ID

	term, err := newClient(t, server).GetTerm(t.Context(), wp.TaxonomyCategory, id)
	if err != nil {
		t.Fatalf("GetTerm: %v", err)
	}
	if term.Name != "Tee & Kaffee – Mehr" || strings.Contains(term.Name, "&amp;") {
		t.Errorf("name = %q, want the entities decoded", term.Name)
	}
}
