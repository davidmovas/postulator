package wp_test

import (
	"testing"
	"time"

	"github.com/davidmovas/postulator/internal/adapters/wp"
	"github.com/davidmovas/postulator/internal/adapters/wp/wptest"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

func TestListItemsWalksEveryPage(t *testing.T) {
	t.Parallel()

	server := wptest.New(t)
	for index := range 5 {
		server.Seed(wptest.Item{Type: wptest.TypePage, Title: "Page " + string(rune('A'+index))})
	}

	client := newClient(t, server)
	seen := 0
	for page := 1; ; page++ {
		result, err := client.ListItems(t.Context(), wp.TypePage, wp.ListQuery{Page: page, PerPage: 2})
		if err != nil {
			t.Fatalf("ListItems page %d: %v", page, err)
		}
		if result.Total != 5 || result.TotalPages != 3 {
			t.Fatalf("page %d reports total %d over %d pages", page, result.Total, result.TotalPages)
		}
		seen += len(result.Items)
		if !result.HasMore {
			break
		}
	}
	if seen != 5 {
		t.Errorf("walked %d items, want 5", seen)
	}
}

func TestListItemsTreatsThePageNumberErrorAsTheEndOfTheList(t *testing.T) {
	t.Parallel()

	server := wptest.New(t)
	server.Seed(wptest.Item{Type: wptest.TypePage, Title: "Only"})

	result, err := newClient(t, server).ListItems(t.Context(), wp.TypePage, wp.ListQuery{Page: 9, PerPage: 2})
	if err != nil {
		t.Fatalf("a page past the end is the end of the list, not a failure: %v", err)
	}
	if len(result.Items) != 0 || result.HasMore {
		t.Errorf("result = %+v, want an empty final page", result)
	}
}

func TestListItemsStopsAtTheEndOfAWooCommerceCollection(t *testing.T) {
	t.Parallel()

	server := wptest.New(t)
	server.Seed(
		wptest.Item{Type: wptest.TypeProduct, Title: "One"},
		wptest.Item{Type: wptest.TypeProduct, Title: "Two"},
	)

	result, err := newClient(t, server).ListItems(t.Context(), wp.TypeProduct, wp.ListQuery{Page: 9, PerPage: 1})
	if err != nil {
		t.Fatalf("ListItems: %v", err)
	}
	if len(result.Items) != 0 || result.HasMore {
		t.Errorf("result = %+v, want an empty final page", result)
	}
}

func TestTheQueryMatchesTheNamespace(t *testing.T) {
	t.Parallel()

	cut := time.Date(2026, time.September, 18, 9, 0, 0, 0, time.UTC)
	cases := []struct {
		name          string
		itemType      wp.ItemType
		query         wp.ListQuery
		wantContext   string
		wantStatus    string
		wantModified  string
		wantFieldList string
	}{
		{
			name:          "core takes a status list and an edit context",
			itemType:      wp.TypePage,
			query:         wp.ListQuery{Status: []string{"publish", "draft"}, ModifiedAfter: &cut, Fields: []string{"slug"}},
			wantContext:   "edit",
			wantStatus:    "publish,draft",
			wantModified:  "2026-09-18T09:00:00",
			wantFieldList: "id,slug",
		},
		{
			name:         "woocommerce takes one status and no context",
			itemType:     wp.TypeProduct,
			query:        wp.ListQuery{Status: []string{"publish", "draft"}, ModifiedAfter: &cut},
			wantStatus:   "publish",
			wantModified: "2026-09-18T09:00:00",
		},
		{
			name:     "a product category has no modification date to filter on",
			itemType: wp.TypeProductCategory,
			query:    wp.ListQuery{ModifiedAfter: &cut},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			server := wptest.New(t)
			if _, err := newClient(t, server).ListItems(t.Context(), tc.itemType, tc.query); err != nil {
				t.Fatalf("ListItems: %v", err)
			}

			recorded, ok := server.LastRequest()
			if !ok {
				t.Fatal("no request reached the site")
			}
			if got := recorded.Query.Get("context"); got != tc.wantContext {
				t.Errorf("context = %q, want %q", got, tc.wantContext)
			}
			if got := recorded.Query.Get("status"); got != tc.wantStatus {
				t.Errorf("status = %q, want %q", got, tc.wantStatus)
			}
			if got := recorded.Query.Get("modified_after"); got != tc.wantModified {
				t.Errorf("modified_after = %q, want %q", got, tc.wantModified)
			}
			if got := recorded.Query.Get("_fields"); got != tc.wantFieldList {
				t.Errorf("_fields = %q, want %q", got, tc.wantFieldList)
			}
		})
	}
}

func TestTheQueryClampsTheWindow(t *testing.T) {
	t.Parallel()

	server := wptest.New(t)
	if _, err := newClient(t, server).ListItems(t.Context(), wp.TypePage, wp.ListQuery{Page: 0, PerPage: 9999}); err != nil {
		t.Fatalf("ListItems: %v", err)
	}

	recorded, _ := server.LastRequest()
	if got := recorded.Query.Get("page"); got != "1" {
		t.Errorf("page = %q, want 1", got)
	}
	if got := recorded.Query.Get("per_page"); got != "100" {
		t.Errorf("per_page = %q, want the WordPress maximum of 100", got)
	}
}

func TestGetItemReadsTheRawPostRatherThanTheRendering(t *testing.T) {
	t.Parallel()

	server := wptest.New(t)
	seeded := server.Seed(wptest.Item{
		Type:    wptest.TypePage,
		Title:   "Koffein",
		Content: "<p>Koffein ist ein Alkaloid.</p>",
		Excerpt: "short",
		Status:  "publish",
	})

	item, err := newClient(t, server).GetItem(t.Context(), wp.TypePage, seeded[0].ID)
	if err != nil {
		t.Fatalf("GetItem: %v", err)
	}

	if item.Title != "Koffein" || item.Content != "<p>Koffein ist ein Alkaloid.</p>" || item.Excerpt != "short" {
		t.Errorf("item = %+v", item)
	}
	if item.Type != wp.TypePage || item.Slug != "koffein" || item.Link == "" {
		t.Errorf("item = %+v", item)
	}
	if item.Modified.IsZero() || item.Modified.Location() != time.UTC {
		t.Errorf("modified = %s, want a UTC instant parsed from a suffix-less WordPress stamp", item.Modified)
	}
	if item.Meta != nil {
		t.Errorf("meta = %v, want nil when the site sends the empty array", item.Meta)
	}
}

func TestGetItemMapsAWooCommerceProduct(t *testing.T) {
	t.Parallel()

	server := wptest.New(t)
	category := server.Seed(wptest.Item{Type: wptest.TypeProductCategory, Title: "Koffein"})[0]
	seeded := server.Seed(wptest.Item{
		Type:       wptest.TypeProduct,
		Title:      "Powder",
		Content:    "<p>long</p>",
		Excerpt:    "short",
		Categories: []int64{category.ID},
	})

	item, err := newClient(t, server).GetItem(t.Context(), wp.TypeProduct, seeded[0].ID)
	if err != nil {
		t.Fatalf("GetItem: %v", err)
	}

	if item.Title != "Powder" || item.Content != "<p>long</p>" || item.Excerpt != "short" {
		t.Errorf("item = %+v", item)
	}
	if item.Type != wp.TypeProduct || item.Link == "" {
		t.Errorf("item = %+v", item)
	}
	if len(item.Categories) != 1 || item.Categories[0] != category.ID {
		t.Errorf("categories = %v", item.Categories)
	}
}

func TestGetItemMapsAProductCategory(t *testing.T) {
	t.Parallel()

	server := wptest.New(t)
	seeded := server.Seed(wptest.Item{Type: wptest.TypeProductCategory, Title: "Koffein", Content: "the hub"})

	item, err := newClient(t, server).GetItem(t.Context(), wp.TypeProductCategory, seeded[0].ID)
	if err != nil {
		t.Fatalf("GetItem: %v", err)
	}
	if item.Title != "Koffein" || item.Content != "the hub" || item.Slug != "koffein" {
		t.Errorf("item = %+v", item)
	}
	if !item.Modified.IsZero() {
		t.Error("a product category is a term and carries no modification date")
	}
}

func TestTheReadMethodsRejectTheUnknownAndTheMissing(t *testing.T) {
	t.Parallel()

	server := wptest.New(t)
	client := newClient(t, server)

	if _, err := client.ListItems(t.Context(), wp.ItemType("attachment"), wp.ListQuery{}); !errors.IsCode(err, errors.Invalid) {
		t.Errorf("code = %q, want %q", errors.CodeOf(err), errors.Invalid)
	}
	if _, err := client.GetItem(t.Context(), wp.TypePage, 404); !errors.IsCode(err, errors.NotFound) {
		t.Errorf("code = %q, want %q", errors.CodeOf(err), errors.NotFound)
	}
}
