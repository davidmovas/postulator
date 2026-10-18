package pagetype_test

import (
	"slices"
	"testing"
	"time"

	"github.com/davidmovas/postulator/internal/domain/field"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/pagetype"
	"github.com/davidmovas/postulator/internal/domain/render"
)

var stamp = time.Date(2026, time.October, 18, 12, 0, 0, 0, time.UTC)

func seeded(siteID string) []pagetype.Type {
	out := make([]pagetype.Type, 0, len(pagetype.Presets()))
	for i, preset := range pagetype.Presets() {
		out = append(out, pagetype.Type{
			ID: siteID + "-" + string(preset), SiteID: siteID, Name: preset.DefaultName(), Preset: preset,
			CreatedAt: stamp.Add(time.Duration(i) * time.Second), UpdatedAt: stamp,
		})
	}
	return out
}

func TestThePresetsAndTheirNames(t *testing.T) {
	t.Parallel()

	want := []pagetype.Preset{pagetype.Hub, pagetype.Entity, pagetype.Product, pagetype.StoreCategory, pagetype.Catalog, pagetype.Technical}
	if got := pagetype.Presets(); !slices.Equal(got, want) {
		t.Fatalf("Presets = %v, want %v", got, want)
	}
	names := map[pagetype.Preset]string{
		pagetype.Hub: "Hub", pagetype.Entity: "Entity", pagetype.Product: "Product",
		pagetype.StoreCategory: "Store category", pagetype.Catalog: "Catalog", pagetype.Technical: "Technical",
	}
	for preset, name := range names {
		if got := preset.DefaultName(); got != name {
			t.Errorf("%s.DefaultName = %q, want %q", preset, got, name)
		}
	}
	for _, preset := range pagetype.Presets() {
		writable := preset == pagetype.Hub || preset == pagetype.Entity || preset == pagetype.Product
		if preset.Writable() != writable {
			t.Errorf("%s.Writable = %t, want %t", preset, preset.Writable(), writable)
		}
	}
	if pagetype.Preset("category").Valid() {
		t.Error("the word category names a preset; the archive preset is store_category")
	}
}

func TestATypeValidatesItsNameAndPreset(t *testing.T) {
	t.Parallel()

	valid := pagetype.Type{ID: "t1", SiteID: "s1", Name: "Pillar", Preset: pagetype.Hub, CreatedAt: stamp, UpdatedAt: stamp}
	if err := valid.Validate(); err != nil {
		t.Fatalf("Validate = %v", err)
	}
	for name, broken := range map[string]pagetype.Type{
		"no id":          {SiteID: "s1", Name: "Pillar", Preset: pagetype.Hub},
		"no site":        {ID: "t1", Name: "Pillar", Preset: pagetype.Hub},
		"a blank name":   {ID: "t1", SiteID: "s1", Name: "  ", Preset: pagetype.Hub},
		"no such preset": {ID: "t1", SiteID: "s1", Name: "Pillar", Preset: "category"},
	} {
		if err := broken.Validate(); err == nil {
			t.Errorf("%s: Validate passed", name)
		}
	}
}

func listing(inside bool) render.Record {
	return render.Record{Regions: []render.Region{{Kind: render.RegionListing, Inside: inside, Links: 6}}}
}

func TestAnUntypedPageIsTypedByWhatTheSiteShows(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		page     pagemap.Page
		record   *render.Record
		preset   pagetype.Preset
		reason   pagemap.TypeReason
		assigned bool
	}{
		"a product":                             {page: pagemap.Page{WPType: pagemap.WPProduct}, preset: pagetype.Product, reason: pagemap.TypeByProduct, assigned: true},
		"a store category":                      {page: pagemap.Page{WPType: pagemap.WPProductCategory}, preset: pagetype.StoreCategory, reason: pagemap.TypeByStoreCategory, assigned: true},
		"the shop":                              {page: pagemap.Page{WPType: pagemap.WPPage, TechnicalReason: pagemap.TechnicalShop}, preset: pagetype.Catalog, reason: pagemap.TypeByShop, assigned: true},
		"the cart":                              {page: pagemap.Page{WPType: pagemap.WPPage, TechnicalReason: pagemap.TechnicalCart}, preset: pagetype.Technical, reason: pagemap.TypeByTechnical, assigned: true},
		"a page the workbook calls technical":   {page: pagemap.Page{WPType: pagemap.WPPage, TechnicalReason: pagemap.TechnicalWorkbook}, preset: pagetype.Technical, reason: pagemap.TypeByTechnical, assigned: true},
		"a page whose content lists products":   {page: pagemap.Page{WPType: pagemap.WPPage}, record: new(listing(true)), preset: pagetype.Hub, reason: pagemap.TypeByListing, assigned: true},
		"a page with products in its sidebar":   {page: pagemap.Page{WPType: pagemap.WPPage}, record: new(listing(false)), preset: pagetype.Entity, reason: pagemap.TypeByDefault, assigned: true},
		"a page without a listing":              {page: pagemap.Page{WPType: pagemap.WPPost}, record: &render.Record{}, preset: pagetype.Entity, reason: pagemap.TypeByDefault, assigned: true},
		"a page not rendered yet waits":         {page: pagemap.Page{WPType: pagemap.WPPage}},
		"a technical page not rendered yet":     {page: pagemap.Page{WPType: pagemap.WPPage, TechnicalReason: pagemap.TechnicalNoindex}, preset: pagetype.Technical, reason: pagemap.TypeByTechnical, assigned: true},
		"a product with a listing is a product": {page: pagemap.Page{WPType: pagemap.WPProduct}, record: new(listing(true)), preset: pagetype.Product, reason: pagemap.TypeByProduct, assigned: true},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			preset, reason, assigned := pagetype.Assign(pagetype.FactsOf(tc.page, tc.record))
			if assigned != tc.assigned || preset != tc.preset || reason != tc.reason {
				t.Errorf("Assign = %s %s %t, want %s %s %t", preset, reason, assigned, tc.preset, tc.reason, tc.assigned)
			}
		})
	}
}

func TestACellNamesATypeByTheSitesNameFirstThenBySynonym(t *testing.T) {
	t.Parallel()

	types := seeded("s1")
	renamed := slices.Clone(types)
	renamed[0].Name = "Category"
	pillar := pagetype.Type{ID: "s1-pillar", SiteID: "s1", Name: "Pillar", Preset: pagetype.Hub, CreatedAt: stamp.Add(time.Hour), UpdatedAt: stamp}
	withPillar := append(slices.Clone(types), pillar)

	cases := map[string]struct {
		types  []pagetype.Type
		cell   string
		wpType pagemap.WPType
		want   string
		found  bool
	}{
		"the site's own name":                   {types: withPillar, cell: "pillar", wpType: pagemap.WPPage, want: "s1-pillar", found: true},
		"a preset's name":                       {types: types, cell: "Entity", wpType: pagemap.WPPage, want: "s1-entity", found: true},
		"Category on a regular page is a hub":   {types: types, cell: "Category", wpType: pagemap.WPPage, want: "s1-hub", found: true},
		"a renamed hub keeps the client's word": {types: renamed, cell: "category", wpType: pagemap.WPPage, want: "s1-hub", found: true},
		"the old kind topic":                    {types: types, cell: "topic", wpType: pagemap.WPPage, want: "s1-entity", found: true},
		"the old kind custom":                   {types: types, cell: "custom", wpType: pagemap.WPPage, want: "s1-entity", found: true},
		"GEO, the client's level":               {types: types, cell: "GEO", wpType: pagemap.WPPage, want: "s1-entity", found: true},
		"Compound/Product, the client's level":  {types: types, cell: "Compound/Product", wpType: pagemap.WPPage, want: "s1-product", found: true},
		"a commercial taxonomy":                 {types: types, cell: "Commercial Taxonomy", wpType: pagemap.WPPage, want: "s1-hub", found: true},
		"a pillar without a type of that name":  {types: types, cell: "pillar", wpType: pagemap.WPPage, want: "s1-hub", found: true},
		"the oldest hub takes a synonym":        {types: withPillar, cell: "hub", wpType: pagemap.WPPage, want: "s1-hub", found: true},
		"a store category named on a page":      {types: types, cell: "Store category", wpType: pagemap.WPPage, want: "s1-hub", found: true},
		"a term page whatever its cell":         {types: types, cell: "Product", wpType: pagemap.WPProductCategory, want: "s1-store_category", found: true},
		"a term page with an empty cell":        {types: types, cell: "", wpType: pagemap.WPProductCategory, want: "s1-store_category", found: true},
		"an empty cell on a page names nothing": {types: types, cell: " ", wpType: pagemap.WPPage},
		"an unknown word names nothing":         {types: types, cell: "landing", wpType: pagemap.WPPage},
		"the shop's word":                       {types: types, cell: "Shop", wpType: pagemap.WPPage, want: "s1-catalog", found: true},
		"technical":                             {types: types, cell: "technical", wpType: pagemap.WPPage, want: "s1-technical", found: true},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got, found := pagetype.Match(tc.cell, tc.types, tc.wpType)
			if found != tc.found || (found && got.ID != tc.want) {
				t.Errorf("Match(%q) = %s %t, want %s %t", tc.cell, got.ID, found, tc.want, tc.found)
			}
		})
	}
}

func enabledKeys(scopes []field.Scope) []field.Key {
	out := make([]field.Key, 0, len(scopes))
	for i := range scopes {
		if scopes[i].Enabled {
			out = append(out, scopes[i].Key)
		}
	}
	slices.Sort(out)
	return out
}

func TestEachPresetSeedsItsDefaultScopes(t *testing.T) {
	t.Parallel()

	want := map[pagetype.Preset][]field.Key{
		pagetype.Hub:           {field.Body, field.SEODescription, field.SEOTitle},
		pagetype.Entity:        {field.Body, field.SEODescription, field.SEOTitle},
		pagetype.Product:       {field.ProductDescription, field.ProductShortDescription, field.SEODescription, field.SEOTitle},
		pagetype.StoreCategory: {},
		pagetype.Catalog:       {},
		pagetype.Technical:     {},
	}
	for preset, keys := range want {
		scopes := pagetype.DefaultScopes(preset, "type-1")
		if got := enabledKeys(scopes); !slices.Equal(got, keys) {
			t.Errorf("%s enables %v, want %v", preset, got, keys)
		}
		for i := range scopes {
			if scopes[i].PageTypeID != "type-1" || !scopes[i].AIWrites || scopes[i].Rewrites {
				t.Errorf("%s %s = %+v, want the type's, written by the AI, rewrites off", preset, scopes[i].Key, scopes[i])
			}
			if err := scopes[i].Validate(); err != nil {
				t.Errorf("%s %s does not validate: %v", preset, scopes[i].Key, err)
			}
		}
	}
	for _, key := range []field.Key{field.VariationDescription, field.ImageAlt} {
		if scope := pagetype.DefaultScope(pagetype.Product, key, "type-1"); scope.Enabled {
			t.Errorf("product %s is on by default", key)
		}
	}
}

func TestEachPresetCarriesTheFormerRules(t *testing.T) {
	t.Parallel()

	flags := func(rules field.LinkRules) field.LinkPolicy {
		return field.LinkPolicy{Links: true, Rules: rules, ForbidExternal: true, ForbidSelf: true, AnchorStrategy: field.AnchorPreferUser}
	}
	category := flags(field.LinkRules{UpDepth: 1, DownLinks: true, SiblingMinWeight: 0.5, MaxLinks: 15, MaxPerTarget: 1, ParentLinkWithinParagraphs: 1, ChildrenSection: true})
	cases := map[string]struct {
		preset pagetype.Preset
		key    field.Key
		want   field.LinkPolicy
	}{
		"a hub's body, the hub built-in's":    {pagetype.Hub, field.Body, flags(field.LinkRules{UpDepth: 1, DownLinks: true, SiblingMinWeight: 0.5, MaxLinks: 20, MaxPerTarget: 1, ParentLinkWithinParagraphs: 2, ChildrenSection: true})},
		"an entity's body, the guide's":       {pagetype.Entity, field.Body, flags(field.LinkRules{UpDepth: 2, DownLinks: true, SiblingMinWeight: 0.5, MaxLinks: 12, MaxPerTarget: 1, ParentLinkWithinParagraphs: 2})},
		"a store category's body":             {pagetype.StoreCategory, field.Body, category},
		"a catalog's body":                    {pagetype.Catalog, field.Body, category},
		"a product's description, to its hub": {pagetype.Product, field.ProductDescription, flags(field.LinkRules{UpDepth: 1, SiblingMinWeight: 1, MaxLinks: 3, MaxPerTarget: 1, ParentLinkWithinParagraphs: 1})},
		"a product's short description":       {pagetype.Product, field.ProductShortDescription, field.NoLinks()},
		"a technical page's body":             {pagetype.Technical, field.Body, field.NoLinks()},
		"a hub's SEO title":                   {pagetype.Hub, field.SEOTitle, field.NoLinks()},
		"a hub's excerpt":                     {pagetype.Hub, field.Excerpt, field.NoLinks()},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if got := pagetype.DefaultScope(tc.preset, tc.key, "type-1").Links; got != tc.want {
				t.Errorf("policy = %+v, want %+v", got, tc.want)
			}
		})
	}
}

func TestTheSetResolvesEveryPageToATypeAndAPolicy(t *testing.T) {
	t.Parallel()

	types := seeded("s1")
	pillar := pagetype.Type{ID: "s1-pillar", SiteID: "s1", Name: "Pillar", Preset: pagetype.Hub, CreatedAt: stamp.Add(time.Hour), UpdatedAt: stamp}
	types = append(types, pillar)
	scopes := make([]field.Scope, 0, 4*len(types)+2)
	for _, kind := range types {
		scopes = append(scopes, pagetype.DefaultScopes(kind.Preset, kind.ID)...)
	}
	pillarBody := pagetype.DefaultScope(pagetype.Hub, field.Body, "s1-pillar")
	pillarBody.Links.Rules.MaxLinks = 5
	scopes = append(scopes, pillarBody)
	disabled := pagetype.DefaultScope(pagetype.Entity, field.Body, "s1-entity")
	disabled.Enabled = false
	scopes = append(scopes, disabled)
	set := pagetype.NewSet(types, scopes)

	typed := func(id string) *string { return &id }
	cases := map[string]struct {
		page     pagemap.Page
		typeID   string
		maxLinks int
		links    bool
	}{
		"a typed page":                     {page: pagemap.Page{SiteID: "s1", WPType: pagemap.WPPage, PageTypeID: typed("s1-pillar")}, typeID: "s1-pillar", maxLinks: 5, links: true},
		"an untyped page reads as Entity":  {page: pagemap.Page{SiteID: "s1", WPType: pagemap.WPPage}, typeID: "s1-entity"},
		"an untyped product":               {page: pagemap.Page{SiteID: "s1", WPType: pagemap.WPProduct}, typeID: "s1-product", maxLinks: 3, links: true},
		"an untyped shop":                  {page: pagemap.Page{SiteID: "s1", WPType: pagemap.WPPage, TechnicalReason: pagemap.TechnicalShop}, typeID: "s1-catalog"},
		"a dangling type":                  {page: pagemap.Page{SiteID: "s1", WPType: pagemap.WPPage, PageTypeID: typed("gone")}, typeID: "s1-entity"},
		"another site's type":              {page: pagemap.Page{SiteID: "s1", WPType: pagemap.WPProduct, PageTypeID: typed("s2-hub")}, typeID: "s1-product", maxLinks: 3, links: true},
		"a hub's body":                     {page: pagemap.Page{SiteID: "s1", WPType: pagemap.WPPage, PageTypeID: typed("s1-hub")}, typeID: "s1-hub", maxLinks: 20, links: true},
		"a disabled body carries no links": {page: pagemap.Page{SiteID: "s1", WPType: pagemap.WPPage, PageTypeID: typed("s1-entity")}, typeID: "s1-entity"},
		"a store category carries none":    {page: pagemap.Page{SiteID: "s1", WPType: pagemap.WPProductCategory, PageTypeID: typed("s1-store_category")}, typeID: "s1-store_category"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if got := set.TypeOf(tc.page); got.ID != tc.typeID {
				t.Errorf("TypeOf = %s, want %s", got.ID, tc.typeID)
			}
			policy := set.Policy(tc.page)
			if policy.Links != tc.links || (tc.links && policy.Rules.MaxLinks != tc.maxLinks) {
				t.Errorf("Policy = %+v, want links %t max %d", policy, tc.links, tc.maxLinks)
			}
		})
	}

	if oldest, found := set.ByPreset(pagetype.Hub); !found || oldest.ID != "s1-hub" {
		t.Errorf("ByPreset(hub) = %s %t, want the oldest hub", oldest.ID, found)
	}
	if scope, found := set.Scope("s1-hub", field.Instance(field.Meta, "spec_sheet")); found || scope.Enabled {
		t.Errorf("a custom field no scope names = %+v %t", scope, found)
	}
}
