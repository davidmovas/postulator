//go:build !race

package scale_test

import (
	"cmp"
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/davidmovas/postulator/internal/application/observe"
	"github.com/davidmovas/postulator/internal/domain/block"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/site"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

const (
	baseURL       = "https://scale.example"
	statusPublish = "publish"
)

type crumb struct {
	name string
	path string
}

type entry struct {
	spec       *spec
	modified   time.Time
	revision   int
	content    string
	variations []observe.Variation
}

type wordpress struct {
	mu        sync.Mutex
	entries   []*entry
	byWPID    map[int64]*entry
	byURL     map[string]*entry
	technical site.TechnicalPages
}

func newWordPress(specs []*spec, at time.Time) *wordpress {
	wp := &wordpress{byWPID: map[int64]*entry{}, byURL: map[string]*entry{}}
	for _, s := range specs {
		if s.status != statusPublish {
			continue
		}
		held := &entry{spec: s, modified: at, content: s.body(0)}
		if s.wpType == pagemap.WPProduct {
			for v := range variationsPerProduct {
				held.variations = append(held.variations, observe.Variation{
					ID: s.wpID*10 + int64(v), ParentID: s.wpID, Status: statusPublish, SKU: fmt.Sprintf("%s-%d", s.sku(), v),
					Price: "19.00", RegularPrice: "19.00", StockStatus: "instock", ManageStock: "false",
					Attributes:  json.RawMessage(fmt.Sprintf(`[{"name":"Size","option":"%d ml"}]`, 10*(v+1))),
					Description: "<p>Variation " + strconv.Itoa(v) + " of " + s.title + ".</p>", Modified: at,
				})
			}
		}
		wp.entries = append(wp.entries, held)
		if !s.wpType.Term() {
			wp.byWPID[s.wpID] = held
		}
		wp.byURL[baseURL+s.path] = held
		switch s.kind {
		case kindFront:
			wp.technical.Front = s.wpID
		case kindShop:
			wp.technical.Shop = s.wpID
		case kindCart:
			wp.technical.Cart = s.wpID
		}
	}
	return wp
}

func (w *wordpress) touch(paths []string, at time.Time) {
	w.mu.Lock()
	defer w.mu.Unlock()
	for _, path := range append(slices.Clone(paths), sentinel) {
		held := w.byURL[baseURL+path]
		modified := at.Add(-time.Hour)
		if path == sentinel {
			modified = at
		}
		held.revision++
		held.modified = modified
		held.content = held.spec.body(held.revision)
		for v := range held.variations {
			held.variations[v].Modified = modified
		}
	}
}

func (w *wordpress) readers() observe.Readers {
	return observe.Readers{
		Core: coreReader{w}, Store: storeReader{w}, Plugin: pluginReader{w}, Render: renderReader{w},
		Manifest: observe.Manifest{
			Version: "1.5.2", Required: "1.5.1", Capabilities: []string{"bulk", "raw", "terms", "settings"},
			PostTypes: []string{"page", "post", "product"}, Taxonomies: []string{"product_cat"},
		},
	}
}

func (w *wordpress) since(after *time.Time, wanted func(*entry) bool) []*entry {
	w.mu.Lock()
	defer w.mu.Unlock()
	out := make([]*entry, 0)
	for _, held := range w.entries {
		if wanted(held) && (after == nil || held.modified.After(*after)) {
			out = append(out, held)
		}
	}
	slices.SortFunc(out, func(a, b *entry) int {
		return cmp.Or(a.modified.Compare(b.modified), cmp.Compare(a.spec.key(), b.spec.key()))
	})
	return out
}

type coreReader struct{ w *wordpress }

func (coreReader) PostTypes(context.Context) ([]observe.PostType, error) {
	return []observe.PostType{
		{Slug: "page", RestBase: "pages", Viewable: true, Hierarchical: true},
		{Slug: "post", RestBase: "posts", Viewable: true},
		{Slug: "product", RestBase: "product", Viewable: true},
		{Slug: "attachment", RestBase: "media", Viewable: true},
	}, nil
}

func (coreReader) ListItems(context.Context, string, observe.ItemQuery) (observe.ItemPage, error) {
	return observe.ItemPage{}, errors.New(errors.Invalid, "the generated site answers through its plugin")
}

func (r coreReader) SiteSettings(context.Context) (observe.SiteSettings, error) {
	return observe.SiteSettings{ShowOnFront: "page", FrontPageID: r.w.technical.Front}, nil
}

type pluginReader struct{ w *wordpress }

func (r pluginReader) ListContent(_ context.Context, q observe.ContentQuery) (observe.ContentPage, error) {
	matching := r.w.since(q.Since, func(held *entry) bool { return slices.Contains(q.Types, string(held.spec.wpType)) })
	start := 0
	if q.Cursor != "" {
		start = slices.IndexFunc(matching, func(held *entry) bool { return held.spec.key() == q.Cursor }) + 1
	}
	end := min(start+max(q.Limit, 1), len(matching))
	page := observe.ContentPage{Items: make([]observe.ContentItem, 0, end-start)}
	r.w.mu.Lock()
	defer r.w.mu.Unlock()
	for _, held := range matching[start:end] {
		page.Items = append(page.Items, held.item())
	}
	if end < len(matching) {
		page.NextCursor = matching[end-1].spec.key()
	}
	return page, nil
}

func (r pluginReader) GetRaw(_ context.Context, _ string, id int64) (observe.Raw, error) {
	r.w.mu.Lock()
	defer r.w.mu.Unlock()
	held, ok := r.w.byWPID[id]
	if !ok {
		return observe.Raw{}, errors.New(errors.NotFound, "no such post")
	}
	return observe.Raw{Content: held.content, Hash: block.BodyHash(held.content)}, nil
}

func (pluginReader) GetRendered(context.Context, string, int64) (observe.Rendered, error) {
	return observe.Rendered{}, errors.New(errors.NotFound, "the generated site renders publicly only")
}

func (pluginReader) GetSEOMeta(context.Context, string, int64) (observe.SEOMeta, error) {
	return observe.SEOMeta{}, nil
}

func (r pluginReader) GetTerms(_ context.Context, _ string, id int64) ([]pagemap.Term, error) {
	r.w.mu.Lock()
	defer r.w.mu.Unlock()
	if held, ok := r.w.byWPID[id]; ok {
		return slices.Clone(held.spec.terms), nil
	}
	return []pagemap.Term{}, nil
}

func (pluginReader) GetMeta(context.Context, string, int64, []string) (map[string]string, error) {
	return map[string]string{}, nil
}

func (r pluginReader) Settings(context.Context) (site.TechnicalPages, error) {
	return r.w.technical, nil
}

func (pluginReader) PreviewLink(context.Context, string, int64) (string, error) {
	return "", errors.New(errors.NotFound, "the generated site has no drafts")
}

type storeReader struct{ w *wordpress }

func (r storeReader) ListProducts(_ context.Context, q observe.ProductQuery) (observe.ProductPage, error) {
	matching := r.w.since(q.ModifiedAfter, func(held *entry) bool { return held.spec.wpType == pagemap.WPProduct })
	perPage := max(q.PerPage, 1)
	start := (max(q.Page, 1) - 1) * perPage
	if start >= len(matching) {
		return observe.ProductPage{}, nil
	}
	end := min(start+perPage, len(matching))
	page := observe.ProductPage{HasMore: end < len(matching)}
	r.w.mu.Lock()
	defer r.w.mu.Unlock()
	for _, held := range matching[start:end] {
		page.Items = append(page.Items, held.product())
	}
	return page, nil
}

func (r storeReader) ListVariations(_ context.Context, productID int64, page int) (observe.VariationPage, error) {
	r.w.mu.Lock()
	defer r.w.mu.Unlock()
	held, ok := r.w.byWPID[productID]
	if !ok || page > 1 {
		return observe.VariationPage{Items: []observe.Variation{}}, nil
	}
	return observe.VariationPage{Items: slices.Clone(held.variations)}, nil
}

func (r storeReader) StorePages(context.Context) (site.TechnicalPages, error) {
	return r.w.technical, nil
}

type renderReader struct{ w *wordpress }

func (r renderReader) Visit(_ context.Context, url string) (observe.Visit, error) {
	r.w.mu.Lock()
	defer r.w.mu.Unlock()
	held, ok := r.w.byURL[url]
	if !ok {
		return observe.Visit{Status: 404, Body: "<html><body><h1>Not found</h1></body></html>", URL: url}, nil
	}
	return observe.Visit{Status: 200, Body: held.spec.html(held.content, held.revision), URL: url}, nil
}

func (r renderReader) Resolve(_ context.Context, url string) (observe.Resolution, error) {
	r.w.mu.Lock()
	defer r.w.mu.Unlock()
	if _, ok := r.w.byURL[url]; ok {
		return observe.Resolution{URL: url, Status: 200}, nil
	}
	return observe.Resolution{URL: url, Status: 404}, nil
}

func (e *entry) item() observe.ContentItem {
	s := e.spec
	item := observe.ContentItem{
		ID: s.wpID, Parent: s.parentWP, Type: string(s.wpType), Path: s.path, Slug: pagemap.Slug(s.path), Status: statusPublish,
		Title: s.titleAt(e.revision), Modified: e.modified, Terms: slices.Clone(s.terms),
	}
	if s.wpType.Term() {
		item.ID = s.termID
		return item
	}
	item.H1 = item.Title
	item.Excerpt = s.excerpt()
	item.ContentHash = block.BodyHash(e.content)
	item.Meta = observe.SEOMeta{Title: item.Title + " | Scale", Description: s.description()}
	return item
}

func (e *entry) product() observe.Product {
	s := e.spec
	quantity := int64(12)
	ids := make([]int64, 0, len(e.variations))
	for i := range e.variations {
		ids = append(ids, e.variations[i].ID)
	}
	return observe.Product{
		ID: s.wpID, Permalink: baseURL + s.path, Slug: pagemap.Slug(s.path), Status: statusPublish, Name: s.titleAt(e.revision),
		Type: "variable", Description: e.content, ShortDescription: s.excerpt(), Price: "19.00", RegularPrice: "19.00",
		SKU: s.sku(), StockStatus: "instock", ManageStock: "true", StockQuantity: &quantity,
		Attributes: json.RawMessage(`[{"name":"Size","options":["10 ml","20 ml"]}]`), Categories: slices.Clone(s.terms),
		VariationIDs: ids, UpsellIDs: []int64{}, CrossSellIDs: []int64{},
		Images: []observe.ProductImage{{ID: s.wpID + 500000, Alt: s.title + " bottle", Title: s.title}}, Modified: e.modified,
	}
}

func (s *spec) html(content string, revision int) string {
	var b strings.Builder
	title := s.titleAt(revision)
	b.WriteString("<!doctype html><html><head><title>" + title + "</title>")
	if len(s.crumbs) > 0 {
		items := make([]map[string]any, 0, len(s.crumbs)+1)
		for i, c := range s.crumbs {
			items = append(items, map[string]any{"@type": "ListItem", "position": i + 1, "name": c.name, "item": baseURL + c.path})
		}
		items = append(items, map[string]any{"@type": "ListItem", "position": len(s.crumbs) + 1, "name": title})
		b.WriteString(`<script type="application/ld+json">` + mustJSON(map[string]any{
			"@context": "https://schema.org", "@type": "BreadcrumbList", "itemListElement": items,
		}) + "</script>")
	}
	b.WriteString(`</head><body><header><nav aria-label="Primary"><a href="/">Home</a> <a href="`)
	b.WriteString(s.section() + `">Section</a></nav></header><main>`)
	if len(s.crumbs) > 0 {
		b.WriteString(`<nav aria-label="Breadcrumb"><ol>`)
		for _, c := range s.crumbs {
			b.WriteString(`<li><a href="` + c.path + `">` + c.name + "</a></li>")
		}
		b.WriteString(`<li><span aria-current="page">` + title + "</span></li></ol></nav>")
	}
	b.WriteString("<article><h1>" + title + "</h1>" + content + "</article>")
	if len(s.listing) > 0 {
		b.WriteString("<section><h2>" + title + " listing</h2>")
		for _, path := range s.listing {
			b.WriteString(`<article><a href="` + path + `">` + strings.Trim(path, "/") + "</a></article>")
		}
		b.WriteString("</section>")
	}
	b.WriteString(`</main><aside><a href="` + s.section() + `">More from this section</a></aside>`)
	b.WriteString(`<footer><nav aria-label="Footer"><a href="/shop/">Shop</a></nav></footer></body></html>`)
	return b.String()
}

func mustJSON(value any) string {
	encoded, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	return string(encoded)
}
