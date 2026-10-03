package wptest_test

import (
	"net/http"
	"slices"
	"testing"

	"github.com/davidmovas/postulator/internal/adapters/wp/wptest"
)

func createProductCategory(t *testing.T, server *wptest.Server, body string) (status int, created termBody, refused refusalBody) {
	t.Helper()

	response, payload := call(t, server, http.MethodPost, "/wp-json/wc/v3/products/categories", []byte(body), true)
	if response.StatusCode == http.StatusCreated {
		decode(t, payload, &created)
		return response.StatusCode, created, refused
	}
	decode(t, payload, &refused)
	return response.StatusCode, created, refused
}

func TestAProductCategoryIsCreatedOnceUnderItsParent(t *testing.T) {
	t.Parallel()

	server := wptest.New(t)
	koffein := server.Seed(wptest.Item{Type: wptest.TypeProductCategory, Title: "Koffein"})[0]

	status, created, refused := createProductCategory(t, server, `{"name":"Pulver & Co","parent":`+itoa(koffein.ID)+`}`)
	if status != http.StatusCreated {
		t.Fatalf("status = %d, refused = %+v", status, refused)
	}
	if created.Name != "Pulver &amp; Co" || created.Slug != "pulver-co" || created.Parent != koffein.ID {
		t.Errorf("created = %+v", created)
	}
	if stored, ok := server.LookupTerm(created.ID); !ok || stored.Type != wptest.TypeProductCategory {
		t.Errorf("the created category is not a stored product category: %+v", stored)
	}

	cases := []struct {
		name   string
		body   string
		status int
		code   string
		id     int64
	}{
		{name: "the same name under the same parent", body: `{"name":"pulver &amp; co","parent":` + itoa(koffein.ID) + `}`, status: http.StatusBadRequest, code: "term_exists", id: created.ID},
		{name: "the same name at the top level", body: `{"name":"KOFFEIN"}`, status: http.StatusBadRequest, code: "term_exists", id: koffein.ID},
		{name: "a parent that does not exist", body: `{"name":"Tee","parent":999}`, status: http.StatusNotFound, code: "woocommerce_rest_term_invalid"},
		{name: "no name", body: `{}`, status: http.StatusBadRequest, code: "rest_missing_callback_param"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			status, _, refused := createProductCategory(t, server, tc.body)
			if status != tc.status || refused.Code != tc.code {
				t.Fatalf("status = %d, code = %q; want %d %q", status, refused.Code, tc.status, tc.code)
			}
			if refused.Data.ResourceID != tc.id || refused.Data.TermID != 0 {
				t.Errorf("data = %+v, want resource_id %d and no term_id", refused.Data, tc.id)
			}
		})
	}
}

func TestAProductCategoryIsRefusedWithoutTheStoreOrThePermission(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		options []wptest.Option
		status  int
		code    string
	}{
		{name: "no store", options: []wptest.Option{wptest.WithoutCommerce()}, status: http.StatusNotFound, code: "rest_no_route"},
		{name: "no permission", options: []wptest.Option{wptest.WithoutTermEdit()}, status: http.StatusForbidden, code: "woocommerce_rest_cannot_create"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			server := wptest.New(t, tc.options...)
			status, _, refused := createProductCategory(t, server, `{"name":"Koffein"}`)
			if status != tc.status || refused.Code != tc.code {
				t.Errorf("status = %d, code = %q; want %d %q", status, refused.Code, tc.status, tc.code)
			}
		})
	}
}

func TestProductCategoriesAreFilteredLikeWooCommerce(t *testing.T) {
	t.Parallel()

	server := wptest.New(t)
	koffein := server.Seed(wptest.Item{Type: wptest.TypeProductCategory, Title: "Koffein"})[0]
	powder := server.Seed(wptest.Item{Type: wptest.TypeProductCategory, Title: "Powder", Parent: koffein.ID})[0]
	tee := server.Seed(wptest.Item{Type: wptest.TypeProductCategory, Title: "Tee"})[0]

	cases := []struct {
		name  string
		query string
		want  []int64
	}{
		{name: "everything", query: "", want: []int64{koffein.ID, powder.ID, tee.ID}},
		{name: "the top level", query: "?parent=0", want: []int64{koffein.ID, tee.ID}},
		{name: "one parent's children", query: "?parent=" + itoa(koffein.ID), want: []int64{powder.ID}},
		{name: "some ids", query: "?include=" + itoa(tee.ID) + "," + itoa(koffein.ID), want: []int64{koffein.ID, tee.ID}},
		{name: "a slug", query: "?slug=tee", want: []int64{tee.ID}},
		{name: "a search", query: "?search=pow", want: []int64{powder.ID}},
		{name: "past the end", query: "?page=4&per_page=1", want: []int64{}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			response, payload := call(t, server, http.MethodGet, "/wp-json/wc/v3/products/categories"+tc.query, nil, true)
			if response.StatusCode != http.StatusOK {
				t.Fatalf("status = %d, want 200", response.StatusCode)
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
}

func TestAProductsCategoriesAreReplacedWholesale(t *testing.T) {
	t.Parallel()

	origin := wptest.New(t)
	tee := origin.Seed(wptest.Item{Type: wptest.TypeProductCategory, Title: "Tee"})[0]
	koffein := origin.Seed(wptest.Item{Type: wptest.TypeProductCategory, Title: "Koffein"})[0]
	client := origin.Seed(wptest.Item{Type: wptest.TypeProductCategory, Title: "Angebote"})[0]
	product := origin.Seed(wptest.Item{Type: wptest.TypeProduct, Title: "Powder", Categories: []int64{client.ID}})[0]

	cases := []struct {
		name string
		body string
		want []int64
	}{
		{
			name: "the union in the order the store names them",
			body: `{"categories":[{"id":` + itoa(tee.ID) + `},{"id":` + itoa(client.ID) + `},{"id":` + itoa(koffein.ID) + `}]}`,
			want: []int64{client.ID, koffein.ID, tee.ID},
		},
		{
			name: "an unknown id is dropped",
			body: `{"categories":[{"id":` + itoa(tee.ID) + `},{"id":999}]}`,
			want: []int64{tee.ID},
		},
		{name: "an empty list clears them", body: `{"categories":[]}`, want: []int64{}},
		{name: "an absent list keeps them", body: `{"short_description":"x"}`, want: []int64{client.ID}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			server := wptest.New(t)
			server.Restore(tee, koffein, client, product)

			_, payload := call(t, server, http.MethodPost, "/wp-json/wc/v3/products/"+itoa(product.ID), []byte(tc.body), true)
			var updated productBody
			decode(t, payload, &updated)

			ids := make([]int64, 0, len(updated.Categories))
			for _, category := range updated.Categories {
				ids = append(ids, category.ID)
			}
			if !slices.Equal(ids, tc.want) {
				t.Errorf("categories = %v, want %v", ids, tc.want)
			}
			if stored, _ := server.Lookup(product.ID); !slices.Equal(stored.Categories, tc.want) {
				t.Errorf("stored categories = %v, want %v", stored.Categories, tc.want)
			}
		})
	}
}
