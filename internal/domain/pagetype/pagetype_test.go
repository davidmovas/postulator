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

func TestThePresetsTheirNamesAndTheirRoles(t *testing.T) {
	t.Parallel()

	want := []pagetype.Preset{
		pagetype.Landing, pagetype.Category, pagetype.Guide, pagetype.Info, pagetype.Product,
		pagetype.StoreCategory, pagetype.Catalog, pagetype.Technical, pagetype.Hub, pagetype.Entity,
	}
	if got := pagetype.Presets(); !slices.Equal(got, want) {
		t.Fatalf("Presets = %v, want %v", got, want)
	}
	if got := pagetype.Seeded(); !slices.Equal(got, want[:8]) {
		t.Fatalf("Seeded = %v, want the client's eight, without Hub and Entity", got)
	}
	for _, preset := range want {
		if legacy := preset == pagetype.Hub || preset == pagetype.Entity; preset.Legacy() != legacy {
			t.Errorf("%s Legacy = %t, want %t: only a site from before 2.6 holds Hub and Entity, and no new type starts from them",
				preset, preset.Legacy(), legacy)
		}
	}
	cases := map[pagetype.Preset]struct {
		name     string
		role     pagetype.Role
		writable bool
	}{
		pagetype.Landing:       {"Landing", pagetype.RoleHub, true},
		pagetype.Category:      {"Category", pagetype.RoleHub, true},
		pagetype.Guide:         {"Guide", pagetype.RoleEntity, true},
		pagetype.Info:          {"Info", pagetype.RoleEntity, true},
		pagetype.Product:       {"Product", pagetype.RoleProduct, true},
		pagetype.StoreCategory: {"Store category", pagetype.RoleArchive, false},
		pagetype.Catalog:       {"Catalog", pagetype.RoleCatalog, false},
		pagetype.Technical:     {"Technical", pagetype.RoleTechnical, false},
		pagetype.Hub:           {"Hub", pagetype.RoleHub, true},
		pagetype.Entity:        {"Entity", pagetype.RoleEntity, true},
	}
	if len(cases) != len(pagetype.Presets()) {
		t.Fatalf("the table names %d presets, Presets has %d", len(cases), len(pagetype.Presets()))
	}
	for preset, tc := range cases {
		if got := preset.DefaultName(); got != tc.name {
			t.Errorf("%s.DefaultName = %q, want %q", preset, got, tc.name)
		}
		if got := preset.Role(); got != tc.role {
			t.Errorf("%s.Role = %q, want %q", preset, got, tc.role)
		}
		if preset.Writable() != tc.writable {
			t.Errorf("%s.Writable = %t, want %t", preset, preset.Writable(), tc.writable)
		}
	}
	if pagetype.Preset("pillar").Valid() {
		t.Error("an unknown word names a preset")
	}
	if !slices.Equal(pagetype.Roles(), []pagetype.Role{
		pagetype.RoleHub, pagetype.RoleEntity, pagetype.RoleProduct, pagetype.RoleArchive, pagetype.RoleCatalog, pagetype.RoleTechnical,
	}) {
		t.Errorf("Roles = %v", pagetype.Roles())
	}
}

func TestATypeIsFoundByItsRoleThePreferredPresetFirst(t *testing.T) {
	t.Parallel()

	legacy := []pagetype.Type{
		{ID: "hub", SiteID: "s1", Name: "Hub", Preset: pagetype.Hub, CreatedAt: stamp},
		{ID: "entity", SiteID: "s1", Name: "Entity", Preset: pagetype.Entity, CreatedAt: stamp},
	}
	upgraded := append(slices.Clone(legacy),
		pagetype.Type{ID: "category", SiteID: "s1", Name: "Category", Preset: pagetype.Category, CreatedAt: stamp.Add(time.Hour)},
		pagetype.Type{ID: "landing", SiteID: "s1", Name: "Landing", Preset: pagetype.Landing, CreatedAt: stamp.Add(time.Hour)},
		pagetype.Type{ID: "guide", SiteID: "s1", Name: "Guide", Preset: pagetype.Guide, CreatedAt: stamp.Add(time.Hour)},
		pagetype.Type{ID: "info", SiteID: "s1", Name: "Info", Preset: pagetype.Info, CreatedAt: stamp.Add(time.Hour)},
	)
	landingOnly := []pagetype.Type{{ID: "landing", SiteID: "s1", Name: "Landing", Preset: pagetype.Landing, CreatedAt: stamp}}
	cases := map[string]struct {
		types []pagetype.Type
		role  pagetype.Role
		want  string
	}{
		"a site before 2.6 keeps its hub":       {legacy, pagetype.RoleHub, "hub"},
		"a site before 2.6 keeps its entity":    {legacy, pagetype.RoleEntity, "entity"},
		"an upgraded site prefers Category":     {upgraded, pagetype.RoleHub, "category"},
		"an upgraded site prefers Guide":        {upgraded, pagetype.RoleEntity, "guide"},
		"the last hub-like type stands for all": {landingOnly, pagetype.RoleHub, "landing"},
		"a role the site lacks finds nothing":   {legacy, pagetype.RoleProduct, ""},
	}
	for name, tc := range cases {
		got, found := pagetype.ByRole(tc.types, tc.role)
		if found != (tc.want != "") || got.ID != tc.want {
			t.Errorf("%s: ByRole = %q %t, want %q", name, got.ID, found, tc.want)
		}
	}
}

func TestEachWritablePresetNamesItsBuiltInTemplate(t *testing.T) {
	t.Parallel()

	want := map[pagetype.Preset]string{
		pagetype.Landing: "Hub", pagetype.Category: "Hub", pagetype.Guide: "Guide", pagetype.Info: "Guide",
		pagetype.Hub: "Hub", pagetype.Entity: "Guide", pagetype.Product: "Product",
		pagetype.StoreCategory: "", pagetype.Catalog: "", pagetype.Technical: "",
	}
	for _, preset := range pagetype.Presets() {
		if got := preset.DefaultTemplate(); got != want[preset] {
			t.Errorf("%s.DefaultTemplate = %q, want %q", preset, got, want[preset])
		}
		if (preset.DefaultTemplate() != "") != preset.Writable() {
			t.Errorf("%s names a template %q while writable is %v", preset, preset.DefaultTemplate(), preset.Writable())
		}
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
		"no such preset": {ID: "t1", SiteID: "s1", Name: "Pillar", Preset: "pillar"},
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
		role     pagetype.Role
		reason   pagemap.TypeReason
		assigned bool
	}{
		"a product":                             {page: pagemap.Page{WPType: pagemap.WPProduct}, role: pagetype.RoleProduct, reason: pagemap.TypeByProduct, assigned: true},
		"a store category":                      {page: pagemap.Page{WPType: pagemap.WPProductCategory}, role: pagetype.RoleArchive, reason: pagemap.TypeByStoreCategory, assigned: true},
		"the shop":                              {page: pagemap.Page{WPType: pagemap.WPPage, TechnicalReason: pagemap.TechnicalShop}, role: pagetype.RoleCatalog, reason: pagemap.TypeByShop, assigned: true},
		"the cart":                              {page: pagemap.Page{WPType: pagemap.WPPage, TechnicalReason: pagemap.TechnicalCart}, role: pagetype.RoleTechnical, reason: pagemap.TypeByTechnical, assigned: true},
		"a page the workbook calls technical":   {page: pagemap.Page{WPType: pagemap.WPPage, TechnicalReason: pagemap.TechnicalWorkbook}, role: pagetype.RoleTechnical, reason: pagemap.TypeByTechnical, assigned: true},
		"a page whose content lists products":   {page: pagemap.Page{WPType: pagemap.WPPage}, record: new(listing(true)), role: pagetype.RoleHub, reason: pagemap.TypeByListing, assigned: true},
		"a page with products in its sidebar":   {page: pagemap.Page{WPType: pagemap.WPPage}, record: new(listing(false)), role: pagetype.RoleEntity, reason: pagemap.TypeByDefault, assigned: true},
		"a page without a listing":              {page: pagemap.Page{WPType: pagemap.WPPost}, record: &render.Record{}, role: pagetype.RoleEntity, reason: pagemap.TypeByDefault, assigned: true},
		"a page not rendered yet waits":         {page: pagemap.Page{WPType: pagemap.WPPage}},
		"a technical page not rendered yet":     {page: pagemap.Page{WPType: pagemap.WPPage, TechnicalReason: pagemap.TechnicalNoindex}, role: pagetype.RoleTechnical, reason: pagemap.TypeByTechnical, assigned: true},
		"a product with a listing is a product": {page: pagemap.Page{WPType: pagemap.WPProduct}, record: new(listing(true)), role: pagetype.RoleProduct, reason: pagemap.TypeByProduct, assigned: true},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			var rendered *pagetype.Rendered
			if tc.record != nil {
				rendered = new(pagetype.RenderedOf(tc.record))
			}
			role, reason, assigned := pagetype.Assign(pagetype.FactsOf(tc.page, rendered))
			if assigned != tc.assigned || role != tc.role || reason != tc.reason {
				t.Errorf("Assign = %s %s %t, want %s %s %t", role, reason, assigned, tc.role, tc.reason, tc.assigned)
			}
		})
	}
}

func legacy(siteID string) []pagetype.Type {
	out := make([]pagetype.Type, 0, 6)
	for i, preset := range []pagetype.Preset{pagetype.Hub, pagetype.Entity, pagetype.Product, pagetype.StoreCategory, pagetype.Catalog, pagetype.Technical} {
		out = append(out, pagetype.Type{
			ID: siteID + "-" + string(preset), SiteID: siteID, Name: preset.DefaultName(), Preset: preset,
			CreatedAt: stamp.Add(time.Duration(i) * time.Second), UpdatedAt: stamp,
		})
	}
	return out
}

func fresh(siteID string) []pagetype.Type {
	out := make([]pagetype.Type, 0, len(pagetype.Seeded()))
	for i, preset := range pagetype.Seeded() {
		out = append(out, pagetype.Type{
			ID: siteID + "-" + string(preset), SiteID: siteID, Name: preset.DefaultName(), Preset: preset,
			CreatedAt: stamp.Add(time.Duration(i) * time.Second), UpdatedAt: stamp,
		})
	}
	return out
}

func TestACellNamesATypeByTheSitesNameFirstThenBySynonym(t *testing.T) {
	t.Parallel()

	types := legacy("s1")
	renamed := slices.Clone(types)
	renamed[0].Name = "Category"
	pillar := pagetype.Type{ID: "s1-pillar", SiteID: "s1", Name: "Pillar", Preset: pagetype.Hub, CreatedAt: stamp.Add(time.Hour), UpdatedAt: stamp}
	withPillar := append(slices.Clone(types), pillar)
	client := fresh("s1")

	cases := map[string]struct {
		types  []pagetype.Type
		cell   string
		wpType pagemap.WPType
		want   string
		found  bool
	}{
		"the site's own name":                      {types: withPillar, cell: "pillar", wpType: pagemap.WPPage, want: "s1-pillar", found: true},
		"a preset's name":                          {types: types, cell: "Entity", wpType: pagemap.WPPage, want: "s1-entity", found: true},
		"Category on a site before 2.6 is its hub": {types: types, cell: "Category", wpType: pagemap.WPPage, want: "s1-hub", found: true},
		"a renamed hub keeps the client's word":    {types: renamed, cell: "category", wpType: pagemap.WPPage, want: "s1-hub", found: true},
		"the old kind topic":                       {types: types, cell: "topic", wpType: pagemap.WPPage, want: "s1-entity", found: true},
		"the old kind custom":                      {types: types, cell: "custom", wpType: pagemap.WPPage, want: "s1-entity", found: true},
		"GEO, the client's level":                  {types: types, cell: "GEO", wpType: pagemap.WPPage, want: "s1-entity", found: true},
		"Compound/Product, the client's level":     {types: types, cell: "Compound/Product", wpType: pagemap.WPPage, want: "s1-product", found: true},
		"a commercial taxonomy":                    {types: types, cell: "Commercial Taxonomy", wpType: pagemap.WPPage, want: "s1-hub", found: true},
		"a pillar without a type of that name":     {types: types, cell: "pillar", wpType: pagemap.WPPage, want: "s1-hub", found: true},
		"the oldest hub takes a synonym":           {types: withPillar, cell: "hub", wpType: pagemap.WPPage, want: "s1-hub", found: true},
		"a store category named on a page":         {types: types, cell: "Store category", wpType: pagemap.WPPage, want: "s1-hub", found: true},
		"a term page whatever its cell":            {types: types, cell: "Product", wpType: pagemap.WPProductCategory, want: "s1-store_category", found: true},
		"a term page with an empty cell":           {types: types, cell: "", wpType: pagemap.WPProductCategory, want: "s1-store_category", found: true},
		"an empty cell on a page names nothing":    {types: types, cell: " ", wpType: pagemap.WPPage},
		"an unknown word names nothing":            {types: types, cell: "brochure", wpType: pagemap.WPPage},
		"the shop's word":                          {types: types, cell: "Shop", wpType: pagemap.WPPage, want: "s1-catalog", found: true},
		"technical":                                {types: types, cell: "technical", wpType: pagemap.WPPage, want: "s1-technical", found: true},
		"landing on a site before 2.6 is its hub":  {types: types, cell: "Landing", wpType: pagemap.WPPage, want: "s1-hub", found: true},
		"Category on a 2.6 site":                   {types: client, cell: "Category", wpType: pagemap.WPPage, want: "s1-category", found: true},
		"hub on a 2.6 site is its Category":        {types: client, cell: "hub", wpType: pagemap.WPPage, want: "s1-category", found: true},
		"the old kind topic on a 2.6 site":         {types: client, cell: "topic", wpType: pagemap.WPPage, want: "s1-guide", found: true},
		"an article on a 2.6 site":                 {types: client, cell: "Article", wpType: pagemap.WPPage, want: "s1-guide", found: true},
		"information on a 2.6 site":                {types: client, cell: "Information", wpType: pagemap.WPPage, want: "s1-info", found: true},
		"a store category on a 2.6 page":           {types: client, cell: "Store category", wpType: pagemap.WPPage, want: "s1-category", found: true},
		"a term page on a 2.6 site":                {types: client, cell: "", wpType: pagemap.WPProductCategory, want: "s1-store_category", found: true},
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
		pagetype.Landing:       {field.Body, field.SEODescription, field.SEOTitle},
		pagetype.Category:      {field.Body, field.SEODescription, field.SEOTitle},
		pagetype.Guide:         {field.Body, field.SEODescription, field.SEOTitle},
		pagetype.Info:          {field.SEODescription, field.SEOTitle},
		pagetype.Hub:           {field.Body, field.SEODescription, field.SEOTitle},
		pagetype.Entity:        {field.Body, field.SEODescription, field.SEOTitle},
		pagetype.Product:       {field.ProductDescription, field.ProductShortDescription, field.SEODescription, field.SEOTitle},
		pagetype.StoreCategory: {},
		pagetype.Catalog:       {},
		pagetype.Technical:     {},
	}
	if len(want) != len(pagetype.Presets()) {
		t.Fatalf("the table names %d presets, Presets has %d", len(want), len(pagetype.Presets()))
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
		"a landing's body, a hub's":           {pagetype.Landing, field.Body, flags(field.LinkRules{UpDepth: 1, DownLinks: true, SiblingMinWeight: 0.5, MaxLinks: 20, MaxPerTarget: 1, ParentLinkWithinParagraphs: 2, ChildrenSection: true})},
		"a category's body, a hub's":          {pagetype.Category, field.Body, flags(field.LinkRules{UpDepth: 1, DownLinks: true, SiblingMinWeight: 0.5, MaxLinks: 20, MaxPerTarget: 1, ParentLinkWithinParagraphs: 2, ChildrenSection: true})},
		"a guide's body, an entity's":         {pagetype.Guide, field.Body, flags(field.LinkRules{UpDepth: 2, DownLinks: true, SiblingMinWeight: 0.5, MaxLinks: 12, MaxPerTarget: 1, ParentLinkWithinParagraphs: 2})},
		"an info page's body, off but ready":  {pagetype.Info, field.Body, flags(field.LinkRules{UpDepth: 2, DownLinks: true, SiblingMinWeight: 0.5, MaxLinks: 12, MaxPerTarget: 1, ParentLinkWithinParagraphs: 2})},
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

func TestEachPresetCarriesTheRulesItsFormerBuiltInTemplateHad(t *testing.T) {
	t.Parallel()

	hub := field.LinkRules{UpDepth: 1, DownLinks: true, SiblingMinWeight: 0.5, MaxLinks: 20, MaxPerTarget: 1, ParentLinkWithinParagraphs: 2, ChildrenSection: true}
	category := field.LinkRules{UpDepth: 1, DownLinks: true, SiblingMinWeight: 0.5, MaxLinks: 15, MaxPerTarget: 1, ParentLinkWithinParagraphs: 1, ChildrenSection: true}
	guide := field.LinkRules{UpDepth: 2, DownLinks: true, SiblingMinWeight: 0.5, MaxLinks: 12, MaxPerTarget: 1, ParentLinkWithinParagraphs: 2}
	cases := []struct {
		preset pagetype.Preset
		former string
		rules  field.LinkRules
	}{
		{preset: pagetype.Hub, former: "Hub", rules: hub},
		{preset: pagetype.StoreCategory, former: "Category", rules: category},
		{preset: pagetype.Catalog, former: "Category", rules: category},
		{preset: pagetype.Entity, former: "Guide", rules: guide},
	}
	for _, tc := range cases {
		got := pagetype.DefaultScope(tc.preset, pagetype.LinkField(tc.preset), "type-1").Links
		want := field.LinkPolicy{Links: true, Rules: tc.rules, ForbidExternal: true, ForbidSelf: true, AnchorStrategy: field.AnchorPreferUser}
		if got != want {
			t.Errorf("%s answers %+v, the built-in %s answered %+v under the Default policy", tc.preset, got, tc.former, want)
		}
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
		"an untyped page reads as Guide":   {page: pagemap.Page{SiteID: "s1", WPType: pagemap.WPPage}, typeID: "s1-guide", maxLinks: 12, links: true},
		"an untyped product":               {page: pagemap.Page{SiteID: "s1", WPType: pagemap.WPProduct}, typeID: "s1-product", maxLinks: 3, links: true},
		"an untyped shop":                  {page: pagemap.Page{SiteID: "s1", WPType: pagemap.WPPage, TechnicalReason: pagemap.TechnicalShop}, typeID: "s1-catalog"},
		"a dangling type":                  {page: pagemap.Page{SiteID: "s1", WPType: pagemap.WPPage, PageTypeID: typed("gone")}, typeID: "s1-guide", maxLinks: 12, links: true},
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

	custom := func(key string, enabled bool) field.Scope {
		return field.Scope{PageTypeID: "s1-hub", Key: field.Instance(field.Meta, key), Enabled: enabled, Label: key, Provider: field.ProviderPlugin, Links: field.NoLinks()}
	}
	withCustom := pagetype.NewSet(types, append(slices.Clone(scopes), custom("spec_sheet", true), custom("dosage", true), custom("retired", false)))
	listed := withCustom.Custom("s1-hub")
	got := make([]field.Key, 0, len(listed))
	for i := range listed {
		got = append(got, listed[i].Key)
	}
	if !slices.Equal(got, []field.Key{field.Instance(field.Meta, "dosage"), field.Instance(field.Meta, "spec_sheet")}) {
		t.Errorf("Custom = %v, want the enabled custom fields by key", got)
	}
	if len(withCustom.Custom("s1-entity")) != 0 {
		t.Error("a type without custom fields lists some")
	}

	variations := pagetype.DefaultScope(pagetype.Product, field.VariationDescription, "s1-product")
	variations.Enabled, variations.Instruction = true, "One sentence per variation."
	withVariations := pagetype.NewSet(types, append(slices.Clone(scopes), variations))
	scope, found := withVariations.Scope("s1-product", field.Instance(field.VariationDescription, "41"))
	if !found || !scope.Enabled || scope.Instruction != "One sentence per variation." || scope.Key != field.VariationDescription {
		t.Errorf("a variation's description reads the scope %+v (stored %t), want the one that names the field for every variation", scope, found)
	}
	if scope, found := withVariations.Scope("s1-product", field.Instance(field.Meta, "spec_sheet")); found || scope.Enabled {
		t.Errorf("a meta key took a scope it does not have: %+v %t", scope, found)
	}
}

func TestAWritablePageOnTheSiteNoEntityCarriesNeedsOne(t *testing.T) {
	t.Parallel()

	pillar := pagetype.Type{ID: "s1-pillar", SiteID: "s1", Name: "Pillar", Preset: pagetype.Hub, CreatedAt: stamp, UpdatedAt: stamp}
	set := pagetype.NewSet(append(seeded("s1"), pillar), nil)
	onSite := func(path string, wpType pagemap.WPType, change func(*pagemap.Page)) pagemap.Page {
		page := pagemap.Page{ID: "p" + path, SiteID: "s1", Path: path, WPType: wpType, WPID: new(int64(7)), Status: pagemap.StatusPublished}
		if change != nil {
			change(&page)
		}
		return page
	}
	typed := func(id string) func(*pagemap.Page) { return func(p *pagemap.Page) { p.PageTypeID = &id } }
	cases := map[string]struct {
		page pagemap.Page
		want bool
	}{
		"an untyped page":     {onSite("/peptides/", pagemap.WPPage, nil), true},
		"a product":           {onSite("/product/bpc/", pagemap.WPProduct, nil), true},
		"a client's own type": {onSite("/pillar/", pagemap.WPPage, typed("s1-pillar")), true},
		"a mapped page":       {onSite("/bpc/", pagemap.WPPage, func(p *pagemap.Page) { p.EntityID = new("e1") }), false},
		"the root":            {onSite(pagemap.RootPath, pagemap.WPPage, nil), false},
		"a store category":    {onSite("/product-category/peptides/", pagemap.WPProductCategory, nil), false},
		"a catalog":           {onSite("/shop/", pagemap.WPPage, typed("s1-catalog")), false},
		"a technical page":    {onSite("/cart/", pagemap.WPPage, func(p *pagemap.Page) { p.TechnicalReason = pagemap.TechnicalCart }), false},
		"an archived page":    {onSite("/old/", pagemap.WPPage, func(p *pagemap.Page) { p.Status = pagemap.StatusArchived }), false},
		"a planned page": {onSite("/planned/", pagemap.WPPage, func(p *pagemap.Page) {
			p.WPID, p.Status = nil, pagemap.StatusPlanned
		}), false},
	}
	for name, tc := range cases {
		if got := set.NeedsEntity(tc.page); got != tc.want {
			t.Errorf("%s needs an entity = %v, want %v", name, got, tc.want)
		}
	}
}
