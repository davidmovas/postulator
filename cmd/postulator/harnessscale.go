//go:build uiharness

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/davidmovas/postulator/internal/adapters/sqlite"
	"github.com/davidmovas/postulator/internal/app"
	"github.com/davidmovas/postulator/internal/application/pagetypes"
	"github.com/davidmovas/postulator/internal/application/sites"
	"github.com/davidmovas/postulator/internal/domain/block"
	domaingraph "github.com/davidmovas/postulator/internal/domain/graph"
	"github.com/davidmovas/postulator/internal/domain/keyword"
	"github.com/davidmovas/postulator/internal/domain/note"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/render"
	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/kernel/id"
)

const (
	scaleVariable   = "POSTULATOR_HARNESS_PAGES"
	scaleSiteName   = "Kiln and Co. Wholesale"
	scaleSiteURL    = "http://kiln-wholesale.invalid"
	scaleGuides     = 400
	guideKept       = 4
	productUnowned  = 40
	categoryUnowned = 6
	scaleFirstWPID  = 5000
)

var (
	scaleDepartments = []string{
		"Stoneware", "Porcelain", "Earthenware", "Raku", "Terracotta", "Celadon", "Majolica", "Bone China", "Salt Glaze", "Slipware",
	}
	scaleShapes = []string{
		"Mugs", "Bowls", "Plates", "Vases", "Teapots", "Cups", "Platters", "Jugs", "Planters", "Trays", "Jars", "Dishes",
	}
	scaleInfo      = []string{"about-us", "contact", "shipping", "returns", "faq"}
	scaleLeftovers = []string{"lookbook", "press", "trade-fairs", "studio-visits", "gift-cards"}
	scaleAt        = time.Date(2026, time.October, 20, 9, 0, 0, 0, time.UTC)
)

type scalePage struct {
	path      string
	title     string
	parent    string
	scope     string
	schema    string
	wpType    pagemap.WPType
	technical pagemap.TechnicalReason
	words     int
	h2        int
	entity    bool
	listing   bool
	rendered  bool
}

func scalePages() (int, error) {
	raw := strings.TrimSpace(os.Getenv(scaleVariable))
	if raw == "" {
		return 0, nil
	}
	total, err := strconv.Atoi(raw)
	if err != nil || total < len(scaleShape(0)) {
		return 0, errors.New(errors.Invalid, scaleVariable+" takes a count of pages of at least "+strconv.Itoa(len(scaleShape(0)))).
			WithDetail("value", raw)
	}
	return total, nil
}

func sluggedName(name string) string {
	return strings.ReplaceAll(strings.ToLower(name), " ", "-")
}

func titleOf(slug string) string {
	words := strings.Split(slug, "-")
	for i := range words {
		words[i] = strings.ToUpper(words[i][:1]) + words[i][1:]
	}
	return strings.Join(words, " ")
}

func scaleShape(total int) []scalePage {
	out := []scalePage{
		{path: pagemap.RootPath, title: "Kiln and Co.", wpType: pagemap.WPPage, listing: true, words: 420, h2: 3, rendered: true},
		{path: "/cart/", title: "Cart", wpType: pagemap.WPPage, technical: pagemap.TechnicalCart},
		{path: "/checkout/", title: "Checkout", wpType: pagemap.WPPage, technical: pagemap.TechnicalCheckout},
		{path: "/my-account/", title: "My account", wpType: pagemap.WPPage, technical: pagemap.TechnicalAccount},
		{path: "/shop/", title: "Shop", wpType: pagemap.WPPage, technical: pagemap.TechnicalShop, listing: true, rendered: true},
		{path: "/guides/", title: "Guides", wpType: pagemap.WPPage, entity: true, words: 600, h2: 2, rendered: true},
	}
	for _, slug := range scaleInfo {
		out = append(out, scalePage{path: "/" + slug + "/", title: titleOf(slug), wpType: pagemap.WPPage, words: 300, h2: 1, rendered: true})
	}
	for _, slug := range scaleLeftovers {
		out = append(out, scalePage{path: "/" + slug + "/", title: titleOf(slug), wpType: pagemap.WPPage, words: 200, rendered: true})
	}
	categories := make([]scalePage, 0, len(scaleDepartments)*len(scaleShapes))
	for d, department := range scaleDepartments {
		home := "/" + sluggedName(department) + "/"
		out = append(out, scalePage{
			path: home, title: department, wpType: pagemap.WPPage, entity: true, listing: true, words: 500, h2: 2, rendered: true,
		})
		for c, shape := range scaleShapes {
			name := department + " " + shape
			category := scalePage{
				path: home + sluggedName(shape) + "/", title: name, parent: home, scope: department, wpType: pagemap.WPPage,
				entity: (d*len(scaleShapes)+c)%categoryUnowned != 0, listing: true, words: 350, h2: 2, rendered: true,
			}
			categories = append(categories, category)
			out = append(out, category,
				scalePage{path: "/product-category/" + sluggedName(name) + "/", title: name, wpType: pagemap.WPProductCategory})
		}
	}
	for g := 1; g <= scaleGuides; g++ {
		number := fmt.Sprintf("%03d", g)
		out = append(out, scalePage{
			path: "/guides/glaze-notes-" + number + "/", title: "Glaze notes " + number, parent: "/guides/", scope: "Guides",
			wpType: pagemap.WPPost, entity: g%guideKept == 0, schema: "BlogPosting", words: 900, h2: 4, rendered: true,
		})
	}
	for p := 0; len(out) < total; p++ {
		category := &categories[p%len(categories)]
		scope := category.title
		if !category.entity {
			scope = category.scope
		}
		number := fmt.Sprintf("%04d", p+1)
		out = append(out, scalePage{
			path: "/product/" + sluggedName(category.title) + "-" + number + "/", title: category.title + " No. " + number,
			parent: category.path, scope: scope, wpType: pagemap.WPProduct, entity: p%productUnowned != 0,
		})
	}
	return out
}

func (p *scalePage) extraction() render.Extraction {
	extracted := render.Extraction{
		Hash: block.BodyHash(p.path), Root: render.RootFound, Regions: []render.Region{}, Trails: []pagemap.Trail{},
		JSONLD: map[string][]json.RawMessage{}, Links: []render.Link{}, Headings: []render.Heading{}, WordsContent: p.words,
	}
	if p.listing {
		extracted.Regions = append(extracted.Regions, render.Region{Kind: render.RegionListing, Inside: true, Links: 12})
	}
	for i := range p.h2 {
		extracted.Headings = append(extracted.Headings, render.Heading{Level: 2, Text: p.title + " " + strconv.Itoa(i+1)})
	}
	if p.schema != "" {
		extracted.JSONLD[p.schema] = []json.RawMessage{json.RawMessage(`{"@type":"` + p.schema + `"}`)}
	}
	return extracted
}

func seedScale(ctx context.Context, core *app.Core, total int) error {
	created, err := core.Sites.Create(ctx, sites.CreateRequest{
		Name: scaleSiteName, BaseURL: scaleSiteURL, Username: harnessUser, AllowInsecure: true,
	})
	if err != nil {
		return err
	}
	siteID := created.Site.ID
	shape := scaleShape(total)
	if writeErr := core.Store.Do(ctx, func(unit context.Context) error { return writeScale(unit, core.Store, siteID, shape) }); writeErr != nil {
		return writeErr
	}
	preview, err := core.Classification.Preview(ctx, pagetypes.PreviewRequest{SiteID: siteID})
	if err != nil {
		return err
	}
	_, err = core.Classification.Apply(ctx, pagetypes.ApplyRequest{SiteID: siteID, Hash: preview.Hash})
	return err
}

func writeScale(ctx context.Context, store *sqlite.Store, siteID string, shape []scalePage) error {
	entities, edges := sqlite.NewEntityRepo(store), sqlite.NewEdgeRepo(store)
	pages, renders := sqlite.NewPageRepo(store), sqlite.NewPageRenderRepo(store)
	named := make(map[string]string, len(shape))
	for i := range shape {
		held := &shape[i]
		if !held.entity {
			continue
		}
		entity := domaingraph.Entity{
			ID: id.New(), SiteID: siteID, Name: held.title, Definition: "The " + held.title + " of the wholesale range.",
			Source: domaingraph.SourceImport, CreatedAt: scaleAt, UpdatedAt: scaleAt,
		}
		if scope, scoped := named[held.scope]; scoped {
			entity.ScopeID = &scope
		}
		if err := entities.Insert(ctx, entity); err != nil {
			return err
		}
		named[held.title] = entity.ID
		if entity.ScopeID == nil {
			continue
		}
		parent, err := domaingraph.NewEdge(domaingraph.Edge{
			ID: id.New(), SiteID: siteID, FromEntityID: entity.ID, ToEntityID: *entity.ScopeID, Kind: domaingraph.EdgeParent, Weight: 1,
			Source: domaingraph.SourceImport, Status: domaingraph.StatusApproved, CreatedAt: scaleAt,
		})
		if err != nil {
			return err
		}
		if insertErr := edges.Insert(ctx, parent); insertErr != nil {
			return insertErr
		}
	}
	pageIDs := make(map[string]string, len(shape))
	for i := range shape {
		held := &shape[i]
		pageIDs[held.path] = id.New()
		wpID := int64(scaleFirstWPID + i)
		page := pagemap.Page{
			ID: pageIDs[held.path], SiteID: siteID, Path: held.path, Slug: pagemap.Slug(held.path), WPType: held.wpType, WPID: &wpID,
			Title: held.title, H1: held.title, Keywords: keyword.Of(), Notes: []note.Note{}, Status: pagemap.StatusPublished,
			TechnicalReason: held.technical, ParentReason: pagemap.ParentNone, CreatedAt: scaleAt, UpdatedAt: scaleAt,
		}
		if parentID, known := pageIDs[held.parent]; known {
			page.ParentPageID, page.ParentReason, page.ParentMethod = &parentID, pagemap.ParentJSONLD, pagemap.MethodHref
		}
		entityID, mapped := named[held.title]
		if held.entity && mapped {
			page.EntityID = &entityID
		}
		if err := pages.Insert(ctx, page); err != nil {
			return err
		}
		if page.EntityID != nil {
			if err := entities.SetCanonicalPage(ctx, entityID, &page.ID, scaleAt); err != nil {
				return err
			}
		}
		if held.rendered {
			if err := renders.Put(ctx, render.Record{PageID: page.ID, Extraction: held.extraction(), ExtractedAt: scaleAt}); err != nil {
				return err
			}
		}
	}
	return nil
}
