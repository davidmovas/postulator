package wp_test

import (
	"encoding/json"
	"net/http"
	"slices"
	"strconv"
	"testing"

	"github.com/davidmovas/postulator/internal/adapters/wp"
	"github.com/davidmovas/postulator/internal/adapters/wp/wptest"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

func seedProduct(server *wptest.Server) wptest.Item {
	client := server.Seed(wptest.Item{Type: wptest.TypeProductCategory, Title: "Angebote"})[0]
	return server.Seed(wptest.Item{
		Type:         wptest.TypeProduct,
		Title:        "Powder",
		Content:      "<p>long</p>",
		Excerpt:      "<p>short</p>",
		Status:       "publish",
		RegularPrice: "19.90",
		SKU:          "powder-1",
		Attributes: []wptest.Attribute{
			{ID: 7, Name: "Color", Options: []string{"Blue"}, Visible: true},
			{Name: "Origin", Options: []string{"Client"}, Position: 1, Visible: true},
		},
		Images:     []int64{41},
		Categories: []int64{client.ID},
	})[0]
}

func TestGetProductReadsWhatTheStoreHolds(t *testing.T) {
	t.Parallel()

	server := wptest.New(t)
	seeded := seedProduct(server)

	product, err := newClient(t, server).GetProduct(t.Context(), seeded.ID)
	if err != nil {
		t.Fatalf("GetProduct: %v", err)
	}

	recorded, _ := server.LastRequest()
	if recorded.Query.Get("context") != "edit" {
		t.Errorf("context = %q, want edit, or the store answers its rendering", recorded.Query.Get("context"))
	}
	if product.Name != "Powder" || product.Description != "<p>long</p>" || product.ShortDescription != "<p>short</p>" {
		t.Errorf("product = %+v", product)
	}
	if product.Type != "simple" || product.Status != "publish" || product.Slug != "powder" {
		t.Errorf("product = %+v", product)
	}
	if product.Permalink != server.URL()+"/product/powder/" || product.Modified.IsZero() {
		t.Errorf("permalink = %q, modified = %s", product.Permalink, product.Modified)
	}
	want := []wp.ProductAttribute{
		{ID: 7, Name: "Color", Options: []string{"Blue"}, Visible: true},
		{Name: "Origin", Options: []string{"Client"}, Position: 1, Visible: true},
	}
	if len(product.Attributes) != len(want) {
		t.Fatalf("attributes = %+v, want %+v", product.Attributes, want)
	}
	for i := range want {
		have := product.Attributes[i]
		if have.ID != want[i].ID || have.Name != want[i].Name || !slices.Equal(have.Options, want[i].Options) ||
			have.Position != want[i].Position || have.Visible != want[i].Visible || have.Variation != want[i].Variation {
			t.Errorf("attribute %d = %+v, want %+v", i, have, want[i])
		}
	}
	if len(product.Images) != 1 || product.Images[0].ID != 41 {
		t.Errorf("images = %+v", product.Images)
	}
	if !slices.Equal(product.Categories, seeded.Categories) {
		t.Errorf("categories = %v, want %v", product.Categories, seeded.Categories)
	}

	item := product.Item()
	if item.ID != seeded.ID || item.Type != wp.TypeProduct || item.Title != "Powder" || item.Content != "<p>long</p>" ||
		item.Excerpt != "<p>short</p>" || item.Slug != "powder" || item.Status != "publish" ||
		item.Link != product.Permalink || !item.Modified.Equal(product.Modified) || !slices.Equal(item.Categories, seeded.Categories) {
		t.Errorf("the product as an item = %+v", item)
	}
}

func TestGetProductTreatsTheTrashAndTheMissingAsNotFound(t *testing.T) {
	t.Parallel()

	server := wptest.New(t)
	trashed := server.Seed(wptest.Item{Type: wptest.TypeProduct, Title: "Gone", Status: "trash"})[0]
	page := server.Seed(wptest.Item{Type: wptest.TypePage, Title: "A page"})[0]
	client := newClient(t, server)

	for name, id := range map[string]int64{"trashed": trashed.ID, "a page": page.ID, "missing": 9999} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if _, err := client.GetProduct(t.Context(), id); !errors.IsCode(err, errors.NotFound) {
				t.Errorf("code = %q, want %q", errors.CodeOf(err), errors.NotFound)
			}
		})
	}
}

func TestUpdateProductWritesOnlyWhatItCarries(t *testing.T) {
	t.Parallel()

	attributes := []wp.ProductAttribute{
		{ID: 7, Name: "Color", Options: []string{"Blue"}, Visible: true},
		{Name: "Form", Options: []string{"Liquid"}, Position: 1, Visible: true},
	}
	images := []int64{55}

	keptCategories := func(t *testing.T, product wp.Product, client int64) {
		t.Helper()
		if !slices.Equal(product.Categories, []int64{client}) {
			t.Errorf("categories = %v, want the client's %d untouched", product.Categories, client)
		}
	}

	cases := []struct {
		check  func(t *testing.T, product wp.Product, client, ours int64)
		update func(client, ours int64) wp.UpdateProduct
		wire   func(client, ours int64) string
		name   string
		sent   []string
	}{
		{
			name: "the short description",
			update: func(int64, int64) wp.UpdateProduct {
				return wp.UpdateProduct{ShortDescription: pointerTo("<p>new short</p>")}
			},
			sent: []string{"short_description"},
			check: func(t *testing.T, product wp.Product, client, _ int64) {
				t.Helper()
				if product.ShortDescription != "<p>new short</p>" || len(product.Attributes) != 2 {
					t.Errorf("product = %+v", product)
				}
				keptCategories(t, product, client)
			},
		},
		{
			name:   "the whole attribute list",
			update: func(int64, int64) wp.UpdateProduct { return wp.UpdateProduct{Attributes: &attributes} },
			sent:   []string{"attributes"},
			check: func(t *testing.T, product wp.Product, client, _ int64) {
				t.Helper()
				if len(product.Attributes) != 2 || product.Attributes[1].Name != "Form" ||
					!slices.Equal(product.Attributes[1].Options, []string{"Liquid"}) || product.ShortDescription != "<p>short</p>" {
					t.Errorf("product = %+v", product)
				}
				keptCategories(t, product, client)
			},
		},
		{
			name:   "the images",
			update: func(int64, int64) wp.UpdateProduct { return wp.UpdateProduct{Images: &images} },
			sent:   []string{"images"},
			check: func(t *testing.T, product wp.Product, client, _ int64) {
				t.Helper()
				if len(product.Images) != 1 || product.Images[0].ID != 55 {
					t.Errorf("images = %+v", product.Images)
				}
				keptCategories(t, product, client)
			},
		},
		{
			name: "the client's categories with ours",
			update: func(client, ours int64) wp.UpdateProduct {
				return wp.UpdateProduct{Categories: &[]int64{client, ours}}
			},
			sent: []string{"categories"},
			wire: func(client, ours int64) string {
				return `[{"id":` + strconv.FormatInt(client, 10) + `},{"id":` + strconv.FormatInt(ours, 10) + `}]`
			},
			check: func(t *testing.T, product wp.Product, client, ours int64) {
				t.Helper()
				if !slices.Equal(product.Categories, []int64{client, ours}) {
					t.Errorf("categories = %v, want %d and %d", product.Categories, client, ours)
				}
			},
		},
		{
			name:   "no categories at all",
			update: func(int64, int64) wp.UpdateProduct { return wp.UpdateProduct{Categories: &[]int64{}} },
			sent:   []string{"categories"},
			wire:   func(int64, int64) string { return `[]` },
			check: func(t *testing.T, product wp.Product, _, _ int64) {
				t.Helper()
				if product.Categories == nil || len(product.Categories) != 0 {
					t.Errorf("categories = %#v, want an empty list", product.Categories)
				}
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			server := wptest.New(t)
			seeded := seedProduct(server)
			client := seeded.Categories[0]
			ours := server.Seed(wptest.Item{Type: wptest.TypeProductCategory, Title: "Koffein"})[0].ID

			product, err := newClient(t, server).UpdateProduct(t.Context(), seeded.ID, tc.update(client, ours))
			if err != nil {
				t.Fatalf("UpdateProduct: %v", err)
			}
			tc.check(t, product, client, ours)
			if product.Name != "Powder" || product.Description != "<p>long</p>" || product.Status != "publish" ||
				product.Slug != "powder" {
				t.Errorf("an absent field changed: %+v", product)
			}

			recorded, _ := server.LastRequest()
			var body map[string]json.RawMessage
			if err := json.Unmarshal(recorded.Body, &body); err != nil {
				t.Fatalf("decode the sent body: %v", err)
			}
			keys := make([]string, 0, len(body))
			for key := range body {
				keys = append(keys, key)
			}
			if !slices.Equal(keys, tc.sent) {
				t.Errorf("sent %v, want only %v", keys, tc.sent)
			}
			if tc.wire != nil {
				if want := tc.wire(client, ours); string(body["categories"]) != want {
					t.Errorf("categories went out as %s, want %s", body["categories"], want)
				}
			}

			stored, _ := server.Lookup(seeded.ID)
			if stored.RegularPrice != "19.90" || stored.SKU != "powder-1" {
				t.Errorf("price %q and SKU %q changed", stored.RegularPrice, stored.SKU)
			}
		})
	}
}

func TestUpdateProductRefusesAnEmptyUpdate(t *testing.T) {
	t.Parallel()

	server := wptest.New(t)
	seeded := seedProduct(server)

	if _, err := newClient(t, server).UpdateProduct(t.Context(), seeded.ID, wp.UpdateProduct{}); !errors.IsCode(err, errors.Invalid) {
		t.Errorf("code = %q, want %q", errors.CodeOf(err), errors.Invalid)
	}
	if requests := server.Requests(); len(requests) != 0 {
		t.Errorf("an empty update sent %d requests", len(requests))
	}
}

func TestCommerceSaysWhatTheStoreAllows(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		options []wptest.Option
		want    wp.Commerce
		code    errors.Code
	}{
		{name: "a store the user may edit", want: wp.CommerceReady},
		{name: "no store", options: []wptest.Option{wptest.WithoutCommerce()}, want: wp.CommerceAbsent},
		{name: "a user who may not edit products", options: []wptest.Option{wptest.WithoutProductEdit()}, want: wp.CommerceForbidden},
		{name: "a wrong password", options: []wptest.Option{wptest.WithCredentials("someone", "else")}, code: errors.Unauthorized},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			server := wptest.New(t, tc.options...)
			got, err := newClient(t, server).Commerce(t.Context())
			if tc.code != "" {
				if !errors.IsCode(err, tc.code) {
					t.Fatalf("code = %q, want %q", errors.CodeOf(err), tc.code)
				}
				return
			}
			if err != nil {
				t.Fatalf("Commerce: %v", err)
			}
			if got != tc.want {
				t.Errorf("commerce = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestCommerceOrKeepsWhatWasKnownWhenTheStoreCannotBeAsked(t *testing.T) {
	t.Parallel()

	cases := []struct {
		prepare func(server *wptest.Server)
		name    string
		options []wptest.Option
		want    wp.Commerce
	}{
		{name: "the store answers", want: wp.CommerceAbsent, options: []wptest.Option{wptest.WithoutCommerce()}},
		{
			name:    "the store is down",
			prepare: func(server *wptest.Server) { server.FailNext(http.StatusServiceUnavailable, 10) },
			want:    wp.CommerceReady,
		},
		{name: "the password is refused", options: []wptest.Option{wptest.WithCredentials("someone", "else")}, want: wp.CommerceReady},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			server := wptest.New(t, tc.options...)
			if tc.prepare != nil {
				tc.prepare(server)
			}
			got, err := newClient(t, server).CommerceOr(t.Context(), wp.CommerceReady)
			if err != nil {
				t.Fatalf("CommerceOr: %v", err)
			}
			if got != tc.want {
				t.Errorf("commerce = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestTheGenericMethodsLeaveTheShopToTheStoreMethods(t *testing.T) {
	t.Parallel()

	server := wptest.New(t)
	product := seedProduct(server)
	client := newClient(t, server)

	for _, itemType := range []wp.ItemType{wp.TypeProduct, wp.TypeProductCategory} {
		t.Run(string(itemType), func(t *testing.T) {
			t.Parallel()

			if _, err := client.ListItems(t.Context(), itemType, wp.ListQuery{}); !errors.IsCode(err, errors.Invalid) {
				t.Errorf("ListItems code = %q, want %q", errors.CodeOf(err), errors.Invalid)
			}
			if _, err := client.GetItem(t.Context(), itemType, product.ID); !errors.IsCode(err, errors.Invalid) {
				t.Errorf("GetItem code = %q, want %q", errors.CodeOf(err), errors.Invalid)
			}
		})
	}
}
