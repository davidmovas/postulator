package pagetype

import (
	"time"

	"github.com/davidmovas/postulator/internal/domain/pagemap"
)

const (
	LongArticleWords = 800
	LongArticleH2    = 2
)

func GuideWords() []string {
	return []string{"blog", "guide", "guides", "how-to", "learn", "research"}
}

func InfoSlugs() []string {
	return []string{
		"about", "about-us", "our-story", "team", "careers", "contact", "contact-us", "shipping", "shipping-policy",
		"delivery", "delivery-information", "faq", "faqs", "returns", "returns-policy", "refund-policy", "refunds",
		"refund_returns", "payment", "payments", "payment-methods", "warranty", "help", "support",
	}
}

type ruleSeed struct {
	name       string
	preset     Preset
	conditions Conditions
}

func defaultSeeds() []ruleSeed {
	yes, top, deeper := true, 1, 2
	words := LongArticleWords
	segments := make([]URLMatch, 0, len(GuideWords()))
	for _, word := range GuideWords() {
		segments = append(segments, URLMatch{Mode: URLSegment, Value: word})
	}
	return []ruleSeed{
		{"Technical pages, such as the cart, the checkout and the account", Technical, Conditions{Technical: &yes}},
		{"Products", Product, Conditions{PostTypes: []string{string(pagemap.WPProduct)}}},
		{"Store categories", StoreCategory, Conditions{PostTypes: []string{string(pagemap.WPProductCategory)}}},
		{"The shop", Catalog, Conditions{Shop: &yes}},
		{"The front page", Landing, Conditions{FrontPage: &yes}},
		{"A product listing at the top level", Landing, Conditions{Listing: &Listing{Present: true}, Level: &Range{Max: &top}}},
		{"A product listing below the top level", Category, Conditions{Listing: &Listing{Present: true}, Level: &Range{Min: &deeper}}},
		{"Blog posts", Guide, Conditions{PostTypes: []string{string(pagemap.WPPost)}}},
		{"An article, a blog post or a how-to by its schema", Guide, Conditions{Schema: []SchemaType{SchemaArticle, SchemaBlogPosting, SchemaHowTo}}},
		{"A guide's word in the address", Guide, Conditions{URLs: segments}},
		{"Under a guide", Guide, Conditions{Ancestor: &Ancestor{TypeID: string(Guide)}}},
		{"An about, contact or FAQ page by its schema", Info, Conditions{Schema: []SchemaType{SchemaAboutPage, SchemaContactPage, SchemaFAQPage}}},
		{"An about, contact, shipping or returns page by its address", Info, Conditions{Slugs: InfoSlugs()}},
		{"A long article with sections", Guide, Conditions{Words: &Range{Min: &words}, MinH2: LongArticleH2, Listing: &Listing{}}},
	}
}

const FallbackName = "Anything no rule matched"

func DefaultRules(set Set, siteID string, now time.Time, newID func() string) []Rule {
	targets := make(map[Preset]string, len(Seeded()))
	for _, preset := range Seeded() {
		if kind, found := StandsFor(set.Types(), preset); found {
			targets[preset] = kind.ID
		}
	}
	seeds := defaultSeeds()
	out := make([]Rule, 0, len(seeds)+1)
	for i := range seeds {
		seed := &seeds[i]
		typeID, held := targets[seed.preset]
		if !held {
			continue
		}
		conditions := seed.conditions
		if conditions.Ancestor != nil {
			guide, known := targets[Preset(conditions.Ancestor.TypeID)]
			if !known {
				continue
			}
			conditions.Ancestor = &Ancestor{TypeID: guide}
		}
		out = append(out, Rule{
			ID: newID(), SiteID: siteID, Kind: KindRule, Position: len(out) + 1, Name: seed.name, Enabled: true,
			TypeID: typeID, Conditions: conditions, CreatedAt: now, UpdatedAt: now,
		})
	}
	if typeID, held := targets[Info]; held {
		out = append(out, Rule{
			ID: newID(), SiteID: siteID, Kind: KindFallback, Position: len(out) + 1, Name: FallbackName, Enabled: true,
			TypeID: typeID, CreatedAt: now, UpdatedAt: now,
		})
	}
	return out
}

func StandsFor(types []Type, preset Preset) (Type, bool) {
	if kind, found := ByPreset(types, preset); found {
		return kind, true
	}
	var oldest *Type
	for i := range types {
		if types[i].Preset.Role() != preset.Role() || !sameName(types[i].Name, preset.DefaultName()) {
			continue
		}
		if oldest == nil || types[i].CreatedAt.Before(oldest.CreatedAt) || types[i].CreatedAt.Equal(oldest.CreatedAt) && types[i].ID < oldest.ID {
			oldest = &types[i]
		}
	}
	if oldest == nil {
		return Type{}, false
	}
	return *oldest, true
}
