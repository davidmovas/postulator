package steps

import (
	"encoding/json"
	"net/http"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/davidmovas/postulator/internal/adapters/wp"
	"github.com/davidmovas/postulator/internal/adapters/wp/wptest"
	"github.com/davidmovas/postulator/internal/application/edit"
	"github.com/davidmovas/postulator/internal/domain/block"
	"github.com/davidmovas/postulator/internal/domain/field"
)

var deniedOnTheWire = []string{
	"slug", "status", "parent", "menu_order", "template", "categories", "tags", "meta", "date", "author",
	"name", "price", "regular_price", "sale_price", "sku", "stock_status", "stock_quantity", "manage_stock",
	"attributes", "default_attributes", "variations", "type",
}

func writersOver(t *testing.T, server *wptest.Server) edit.Service {
	t.Helper()

	client, err := wp.New(wp.Config{
		BaseURL: server.URL(), Username: wptest.DefaultUser, AppPassword: wptest.DefaultPassword, AllowInsecure: true,
	}, wp.WithRateLimit(0), wp.WithBackoff(func(int) time.Duration { return 0 }))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return editorFor(client)
}

func hashText(value string) *string {
	return new(block.BodyHash(value))
}

func wireKeys(t *testing.T, body []byte) []string {
	t.Helper()

	var sent map[string]any
	if err := json.Unmarshal(body, &sent); err != nil {
		t.Fatalf("decode a write %s: %v", body, err)
	}
	keys := make([]string, 0, len(sent))
	for key, value := range sent {
		keys = append(keys, key)
		if nested, ok := value.(map[string]any); ok && key == "meta" {
			for inner := range nested {
				keys = append(keys, "meta:"+inner)
			}
		}
	}
	return keys
}

func TestNothingOnTheDenyListLeavesOnAnUpdate(t *testing.T) {
	t.Parallel()

	server := wptest.New(t, wptest.WithSEOPlugin("yoast"))
	page := server.Seed(wptest.Item{
		Type: wptest.TypePage, Title: "Holder", Slug: "holder", Status: "publish", Content: "<p>Old</p>",
		Meta: map[string]string{"subtitle": "Old", "_yoast_wpseo_title": "Old SEO"},
	})[0]
	product := server.Seed(wptest.Item{
		Type: wptest.TypeProduct, Title: "Kettle", Slug: "kettle", Status: "publish", Content: "<p>Old description</p>",
		Excerpt: "Old short", RegularPrice: "9.99", SKU: "K-1", Categories: []int64{3},
	})[0]
	service := writersOver(t, server)
	server.ResetRequests()

	pageWrites := []edit.Write{
		{Key: field.Title, Value: new("New title"), Expected: hashText("Holder")},
		{Key: field.Excerpt, Value: new("New excerpt")},
		{Key: field.FeaturedImage, Value: new("12")},
		{Key: field.Instance(field.Meta, "subtitle"), Value: new("New"), Expected: hashText("Old")},
		{Key: field.SEOTitle, Value: new("New SEO"), Expected: hashText("Old SEO"), Social: &edit.Social{Value: "New OG", Expected: *hashText("")}},
		{Key: field.SEODescription, Value: new("New description")},
		{Key: field.Body, Value: new("<p>New</p>"), Expected: hashText("<p>Old</p>")},
	}
	productWrites := []edit.Write{
		{Key: field.ProductShortDescription, Value: new("New short"), Expected: hashText("Old short")},
		{Key: field.ProductDescription, Value: new("<p>New description</p>"), Expected: hashText("<p>Old description</p>")},
	}
	for name, run := range map[string]func() ([]edit.Result, error){
		"page": func() ([]edit.Result, error) {
			return service.Write(t.Context(), edit.Target{Type: "page", ID: page.ID}, pageWrites)
		},
		"product": func() ([]edit.Result, error) {
			return service.Write(t.Context(), edit.Target{Type: "product", ID: product.ID, Product: true}, productWrites)
		},
	} {
		results, err := run()
		if err != nil {
			t.Fatalf("%s: Write: %v", name, err)
		}
		for _, result := range results {
			if result.Outcome != edit.OutcomeWritten {
				t.Errorf("%s: %s is %s, want written", name, result.Key, result.Outcome)
			}
		}
	}

	writes := 0
	for _, request := range server.Requests() {
		if request.Method == http.MethodGet || len(request.Body) == 0 {
			continue
		}
		writes++
		metaRoute := strings.Contains(request.Path, "/postulator/v1/meta/")
		for _, key := range wireKeys(t, request.Body) {
			if metaRoute && (key == "meta" || key == "expected") {
				continue
			}
			if slices.Contains(deniedOnTheWire, key) || (strings.HasPrefix(key, "meta:") && !field.MetaKeyAllowed(strings.TrimPrefix(key, "meta:"))) {
				t.Errorf("%s %s sent %s", request.Method, request.Path, key)
			}
		}
	}
	if writes < 6 {
		t.Errorf("only %d writes were recorded, too few to cover every writer", writes)
	}
	heldPage, _ := server.Lookup(page.ID)
	heldProduct, _ := server.Lookup(product.ID)
	if heldPage.Slug != "holder" || heldPage.Status != "publish" || heldProduct.Title != "Kettle" || heldProduct.RegularPrice != "9.99" ||
		heldProduct.SKU != "K-1" || len(heldProduct.Categories) != 1 || heldProduct.Status != "publish" {
		t.Errorf("a write moved what the client owns: %+v %+v", heldPage, heldProduct)
	}
	if heldPage.Title != "New title" || heldPage.Content != "<p>New</p>" || heldProduct.Excerpt != "New short" || heldProduct.Content != "<p>New description</p>" {
		t.Errorf("the writable fields did not land: %+v %+v", heldPage, heldProduct)
	}
}

func TestTheWritersReportAValueTheClientMoved(t *testing.T) {
	t.Parallel()

	server := wptest.New(t, wptest.WithSEOPlugin("yoast"))
	page := server.Seed(wptest.Item{
		Type: wptest.TypePage, Title: "Holder", Content: "<p>Old</p>",
		Meta: map[string]string{"subtitle": "Old", "_yoast_wpseo_title": "Old SEO"},
	})[0]
	service := writersOver(t, server)
	target := edit.Target{Type: "page", ID: page.ID}
	server.Rewrite(page.ID, "<p>His</p>")
	server.EditMeta(page.ID, "subtitle", "His")
	server.EditMeta(page.ID, "_yoast_wpseo_title", "His SEO")

	for _, write := range []edit.Write{
		{Key: field.Title, Value: new("New"), Expected: hashText("Another title")},
		{Key: field.Body, Value: new("<p>New</p>"), Expected: hashText("<p>Old</p>")},
		{Key: field.Instance(field.Meta, "subtitle"), Value: new("New"), Expected: hashText("Old")},
		{Key: field.SEOTitle, Value: new("New"), Expected: hashText("Old SEO")},
	} {
		results, err := service.Write(t.Context(), target, []edit.Write{write})
		if err != nil {
			t.Fatalf("%s: Write: %v", write.Key, err)
		}
		if len(results) != 1 || results[0].Outcome != edit.OutcomeMoved || results[0].Hash == nil {
			t.Errorf("%s: results %+v, want moved with the live hash", write.Key, results)
		}
	}
	held, _ := server.Lookup(page.ID)
	if held.Title != "Holder" || held.Content != "<p>His</p>" || held.Meta["subtitle"] != "His" || held.Meta["_yoast_wpseo_title"] != "His SEO" {
		t.Errorf("a moved value was written over: %+v", held)
	}
}
