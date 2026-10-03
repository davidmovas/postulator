package wptest_test

import (
	"net/http"
	"slices"
	"testing"

	"github.com/davidmovas/postulator/internal/adapters/wp/wptest"
)

type termBody struct {
	Name   string `json:"name"`
	Slug   string `json:"slug"`
	ID     int64  `json:"id"`
	Parent int64  `json:"parent"`
}

type refusalBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Data    struct {
		Status     int   `json:"status"`
		TermID     int64 `json:"term_id"`
		ResourceID int64 `json:"resource_id"`
	} `json:"data"`
}

func createCategory(t *testing.T, server *wptest.Server, body string) (status int, created termBody, refused refusalBody) {
	t.Helper()

	response, payload := call(t, server, http.MethodPost, "/wp-json/wp/v2/categories", []byte(body), true)
	if response.StatusCode == http.StatusCreated {
		decode(t, payload, &created)
		return response.StatusCode, created, refused
	}
	decode(t, payload, &refused)
	return response.StatusCode, created, refused
}

func TestACategoryNameIsUniqueUnderItsParent(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		body     func(koffein, powder wptest.Category) string
		existing func(koffein, powder wptest.Category) int64
		slug     string
	}{
		{
			name:     "the same name at the top level",
			body:     func(wptest.Category, wptest.Category) string { return `{"name":"Koffein"}` },
			existing: func(koffein, _ wptest.Category) int64 { return koffein.ID },
		},
		{
			name:     "the same name in another case",
			body:     func(wptest.Category, wptest.Category) string { return `{"name":"KOFFEIN"}` },
			existing: func(koffein, _ wptest.Category) int64 { return koffein.ID },
		},
		{
			name:     "the same name padded with spaces",
			body:     func(wptest.Category, wptest.Category) string { return `{"name":"  Koffein \t"}` },
			existing: func(koffein, _ wptest.Category) int64 { return koffein.ID },
		},
		{
			name:     "the same child under the same parent",
			body:     func(koffein, _ wptest.Category) string { return `{"name":"powder","parent":` + itoa(koffein.ID) + `}` },
			existing: func(_, powder wptest.Category) int64 { return powder.ID },
		},
		{
			name:     "the same name with the slug the term already has",
			body:     func(wptest.Category, wptest.Category) string { return `{"name":"Koffein","slug":"koffein"}` },
			existing: func(koffein, _ wptest.Category) int64 { return koffein.ID },
		},
		{
			name: "the same name under another parent",
			body: func(koffein, _ wptest.Category) string { return `{"name":"Koffein","parent":` + itoa(koffein.ID) + `}` },
			slug: "koffein-koffein",
		},
		{
			name: "the same name with a new slug",
			body: func(wptest.Category, wptest.Category) string { return `{"name":"Koffein","slug":"Koffein Two"}` },
			slug: "koffein-two",
		},
		{
			name: "a child name at the top level",
			body: func(wptest.Category, wptest.Category) string { return `{"name":"Powder"}` },
			slug: "powder-2",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			server := wptest.New(t)
			koffein := server.SeedCategory(wptest.Category{Name: "Koffein"})
			powder := server.SeedCategory(wptest.Category{Name: "Powder", Parent: koffein.ID})

			status, created, refused := createCategory(t, server, tc.body(koffein, powder))
			if tc.existing == nil {
				if status != http.StatusCreated {
					t.Fatalf("status = %d, want 201: %+v", status, refused)
				}
				if created.Slug != tc.slug {
					t.Errorf("slug = %q, want %q", created.Slug, tc.slug)
				}
				if len(server.Categories()) != 3 {
					t.Errorf("the store holds %d categories, want 3", len(server.Categories()))
				}
				return
			}

			want := tc.existing(koffein, powder)
			if status != http.StatusBadRequest || refused.Code != "term_exists" {
				t.Fatalf("status = %d, code = %q; want 400 term_exists", status, refused.Code)
			}
			if refused.Data.TermID != want || refused.Data.Status != http.StatusBadRequest {
				t.Errorf("data = %+v, want term_id %d with status 400", refused.Data, want)
			}
			if refused.Message != "A term with the name provided already exists with this parent." {
				t.Errorf("message = %q", refused.Message)
			}
			if len(server.Categories()) != 2 {
				t.Errorf("the store holds %d categories, want the refused one left out", len(server.Categories()))
			}
		})
	}
}

func TestACategoryNameIsStoredTheWayWordPressStoresIt(t *testing.T) {
	t.Parallel()

	server := wptest.New(t)

	status, created, _ := createCategory(t, server, `{"name":"R&D  <b>Lab</b> > 2"}`)
	if status != http.StatusCreated {
		t.Fatalf("status = %d, want 201", status)
	}
	if created.Name != "R&amp;D Lab &gt; 2" {
		t.Errorf("name = %q, want the stripped, collapsed and escaped form", created.Name)
	}
	if created.Slug != "r-d-lab-2" {
		t.Errorf("slug = %q, want r-d-lab-2", created.Slug)
	}

	response, payload := call(t, server, http.MethodGet, "/wp-json/wp/v2/categories/"+itoa(created.ID), nil, true)
	if response.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", response.StatusCode)
	}
	var read termBody
	decode(t, payload, &read)
	if read != created {
		t.Errorf("read = %+v, want %+v", read, created)
	}

	status, _, refused := createCategory(t, server, `{"name":"r&amp;d lab &gt; 2"}`)
	if status != http.StatusBadRequest || refused.Data.TermID != created.ID {
		t.Errorf("status = %d, refused = %+v; the escaped name is the same name", status, refused)
	}
}

func TestAChildSlugTakesItsParentsSlugThenANumber(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		seeds []wptest.Category
		want  string
	}{
		{name: "a free slug is kept", want: "pulver"},
		{
			name:  "a taken slug takes the parent's",
			seeds: []wptest.Category{{Name: "Pulver"}},
			want:  "pulver-extra",
		},
		{
			name:  "then the grandparent's",
			seeds: []wptest.Category{{Name: "Pulver"}, {Name: "Pulver Extra", Slug: "pulver-extra"}},
			want:  "pulver-extra-koffein",
		},
		{
			name: "then a number",
			seeds: []wptest.Category{
				{Name: "Pulver"}, {Name: "Pulver Extra", Slug: "pulver-extra"}, {Name: "Pulver Extra Koffein", Slug: "pulver-extra-koffein"},
			},
			want: "pulver-extra-koffein-2",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			server := wptest.New(t)
			koffein := server.SeedCategory(wptest.Category{Name: "Koffein"})
			extra := server.SeedCategory(wptest.Category{Name: "Extra", Parent: koffein.ID})
			for _, seed := range tc.seeds {
				server.SeedCategory(seed)
			}

			status, created, refused := createCategory(t, server, `{"name":"Pulver","parent":`+itoa(extra.ID)+`}`)
			if status != http.StatusCreated {
				t.Fatalf("status = %d, refused = %+v", status, refused)
			}
			if created.Slug != tc.want || created.Parent != extra.ID {
				t.Errorf("created = %+v, want slug %q under %d", created, tc.want, extra.ID)
			}
		})
	}
}

func TestACategoryIsRefusedWhatWordPressRefuses(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		options []wptest.Option
		body    string
		status  int
		code    string
	}{
		{name: "a parent that does not exist", body: `{"name":"Pulver","parent":999}`, status: http.StatusBadRequest, code: "rest_term_invalid"},
		{name: "no name", body: `{"parent":0}`, status: http.StatusBadRequest, code: "rest_missing_callback_param"},
		{name: "a broken body", body: `not json`, status: http.StatusBadRequest, code: "rest_invalid_json"},
		{
			name:    "a user who may not manage categories",
			options: []wptest.Option{wptest.WithoutTermEdit()},
			body:    `{"name":"Pulver"}`,
			status:  http.StatusForbidden,
			code:    "rest_cannot_create",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			server := wptest.New(t, tc.options...)
			status, _, refused := createCategory(t, server, tc.body)
			if status != tc.status || refused.Code != tc.code {
				t.Errorf("status = %d, code = %q; want %d %q", status, refused.Code, tc.status, tc.code)
			}
			if len(server.Categories()) != 0 {
				t.Errorf("the store holds %d categories after a refusal", len(server.Categories()))
			}
		})
	}

	server := wptest.New(t)
	status, created, _ := createCategory(t, server, `{"name":"Pulver","parent":0}`)
	if status != http.StatusCreated || created.Parent != 0 {
		t.Errorf("status = %d, created = %+v; a zero parent is the top level", status, created)
	}
}

func TestCategoriesAreFilteredLikeWordPress(t *testing.T) {
	t.Parallel()

	server := wptest.New(t)
	koffein := server.SeedCategory(wptest.Category{Name: "Koffein"})
	powder := server.SeedCategory(wptest.Category{Name: "Powder", Parent: koffein.ID})
	pills := server.SeedCategory(wptest.Category{Name: "Pills", Parent: koffein.ID})
	tee := server.SeedCategory(wptest.Category{Name: "Tee"})

	cases := []struct {
		name  string
		query string
		want  []int64
		total string
	}{
		{name: "everything in id order", query: "", want: []int64{koffein.ID, powder.ID, pills.ID, tee.ID}, total: "4"},
		{name: "the top level", query: "?parent=0", want: []int64{koffein.ID, tee.ID}, total: "2"},
		{name: "one parent's children", query: "?parent=" + itoa(koffein.ID), want: []int64{powder.ID, pills.ID}, total: "2"},
		{name: "some ids", query: "?include=" + itoa(tee.ID) + "," + itoa(powder.ID), want: []int64{powder.ID, tee.ID}, total: "2"},
		{name: "repeated ids", query: "?include[]=" + itoa(tee.ID) + "&include[]=" + itoa(pills.ID), want: []int64{pills.ID, tee.ID}, total: "2"},
		{name: "a slug", query: "?slug=pills", want: []int64{pills.ID}, total: "1"},
		{name: "a search over names", query: "?search=OWD", want: []int64{powder.ID}, total: "1"},
		{name: "a search over slugs", query: "?search=tee", want: []int64{tee.ID}, total: "1"},
		{name: "a second page", query: "?per_page=3&page=2", want: []int64{tee.ID}, total: "4"},
		{name: "past the end", query: "?per_page=3&page=9", want: []int64{}, total: "4"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			response, payload := call(t, server, http.MethodGet, "/wp-json/wp/v2/categories"+tc.query, nil, true)
			if response.StatusCode != http.StatusOK {
				t.Fatalf("status = %d, want 200", response.StatusCode)
			}
			if got := response.Header.Get("X-WP-Total"); got != tc.total {
				t.Errorf("X-WP-Total = %q, want %q", got, tc.total)
			}

			var listed []termBody
			decode(t, payload, &listed)
			ids := make([]int64, 0, len(listed))
			for _, term := range listed {
				ids = append(ids, term.ID)
			}
			if !slices.Equal(ids, tc.want) {
				t.Errorf("ids = %v, want %v", ids, tc.want)
			}
		})
	}

	for _, query := range []string{"?parent=none", "?include=none", "?per_page=0", "?per_page=101", "?page=0"} {
		response, payload := call(t, server, http.MethodGet, "/wp-json/wp/v2/categories"+query, nil, true)
		var refused refusalBody
		decode(t, payload, &refused)
		if response.StatusCode != http.StatusBadRequest || refused.Code != "rest_invalid_param" {
			t.Errorf("%s: status = %d, code = %q; want 400 rest_invalid_param", query, response.StatusCode, refused.Code)
		}
	}
}

func TestAMissingCategoryIsNotFound(t *testing.T) {
	t.Parallel()

	server := wptest.New(t)
	for _, path := range []string{"/wp-json/wp/v2/categories/999", "/wp-json/wp/v2/categories/none"} {
		response, payload := call(t, server, http.MethodGet, path, nil, true)
		var refused refusalBody
		decode(t, payload, &refused)
		if response.StatusCode != http.StatusNotFound || refused.Code != "rest_term_invalid" {
			t.Errorf("%s: status = %d, code = %q", path, response.StatusCode, refused.Code)
		}
	}
}
