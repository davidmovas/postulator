package pagetype

import (
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/davidmovas/postulator/internal/domain/pagemap"
)

const (
	ruleNameLimit  = 120
	ruleValueLimit = 64
	RuleLimit      = 100
)

type RuleKind string

const (
	KindRule     RuleKind = "rule"
	KindFallback RuleKind = "fallback"
)

type URLMode string

const (
	URLPrefix  URLMode = "prefix"
	URLSegment URLMode = "segment"
	URLGlob    URLMode = "glob"
)

type SchemaType string

const (
	SchemaArticle        SchemaType = "Article"
	SchemaBlogPosting    SchemaType = "BlogPosting"
	SchemaHowTo          SchemaType = "HowTo"
	SchemaFAQPage        SchemaType = "FAQPage"
	SchemaAboutPage      SchemaType = "AboutPage"
	SchemaContactPage    SchemaType = "ContactPage"
	SchemaCollectionPage SchemaType = "CollectionPage"
	SchemaItemList       SchemaType = "ItemList"
)

func SchemaTypes() []SchemaType {
	return []SchemaType{
		SchemaArticle, SchemaBlogPosting, SchemaHowTo, SchemaFAQPage, SchemaAboutPage, SchemaContactPage,
		SchemaCollectionPage, SchemaItemList,
	}
}

type URLMatch struct {
	Mode  URLMode `json:"mode"`
	Value string  `json:"value"`
}

type Range struct {
	Min *int `json:"min,omitempty"`
	Max *int `json:"max,omitempty"`
}

type Ancestor struct {
	PageID string `json:"pageId,omitempty"`
	TypeID string `json:"typeId,omitempty"`
}

type Listing struct {
	Present     bool `json:"present"`
	MinProducts int  `json:"minProducts,omitempty"`
}

type Conditions struct {
	PostTypes []string     `json:"postTypes,omitempty"`
	URLs      []URLMatch   `json:"urls,omitempty"`
	Slugs     []string     `json:"slugs,omitempty"`
	Ancestor  *Ancestor    `json:"ancestor,omitempty"`
	Level     *Range       `json:"level,omitempty"`
	Listing   *Listing     `json:"listing,omitempty"`
	Schema    []SchemaType `json:"schema,omitempty"`
	Words     *Range       `json:"words,omitempty"`
	MinH2     int          `json:"minH2,omitempty"`
	Technical *bool        `json:"technical,omitempty"`
	FrontPage *bool        `json:"frontPage,omitempty"`
	Shop      *bool        `json:"shop,omitempty"`
}

type Rule struct {
	ID         string
	SiteID     string
	Kind       RuleKind
	Position   int
	Name       string
	Enabled    bool
	TypeID     string
	Conditions Conditions
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

func (k RuleKind) Valid() bool {
	return k == KindRule || k == KindFallback
}

func (m URLMode) Valid() bool {
	return m == URLPrefix || m == URLSegment || m == URLGlob
}

func (s SchemaType) Valid() bool {
	return slices.Contains(SchemaTypes(), s)
}

func (c Conditions) Empty() bool {
	return len(c.PostTypes) == 0 && len(c.URLs) == 0 && len(c.Slugs) == 0 && c.Ancestor == nil && c.Level == nil &&
		c.Listing == nil && len(c.Schema) == 0 && c.Words == nil && c.MinH2 == 0 && c.Technical == nil &&
		c.FrontPage == nil && c.Shop == nil
}

func (c Conditions) Normalized() Conditions {
	out := c
	out.PostTypes = lowered(c.PostTypes)
	out.Slugs = lowered(c.Slugs)
	out.URLs = make([]URLMatch, 0, len(c.URLs))
	for _, match := range c.URLs {
		value := strings.ToLower(strings.TrimSpace(match.Value))
		if match.Mode == URLPrefix && value != "" && !strings.HasPrefix(value, "/") {
			value = "/" + value
		}
		next := URLMatch{Mode: match.Mode, Value: value}
		if !slices.Contains(out.URLs, next) {
			out.URLs = append(out.URLs, next)
		}
	}
	out.Schema = make([]SchemaType, 0, len(c.Schema))
	for _, kind := range c.Schema {
		if !slices.Contains(out.Schema, kind) {
			out.Schema = append(out.Schema, kind)
		}
	}
	if c.Ancestor != nil {
		out.Ancestor = &Ancestor{PageID: strings.TrimSpace(c.Ancestor.PageID), TypeID: strings.TrimSpace(c.Ancestor.TypeID)}
	}
	if len(out.PostTypes) == 0 {
		out.PostTypes = nil
	}
	if len(out.Slugs) == 0 {
		out.Slugs = nil
	}
	if len(out.URLs) == 0 {
		out.URLs = nil
	}
	if len(out.Schema) == 0 {
		out.Schema = nil
	}
	return out
}

func lowered(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.ToLower(strings.TrimSpace(value))
		if value != "" && !slices.Contains(out, value) {
			out = append(out, value)
		}
	}
	return out
}

func (r Rule) Validate(set Set) error {
	switch {
	case r.ID == "":
		return invalid("a rule needs an id", "id")
	case r.SiteID == "":
		return invalid("a rule belongs to a site", "siteId")
	case !r.Kind.Valid():
		return invalid("a rule is a rule or the fallback", "kind")
	case r.Position < 0:
		return invalid("a rule's place in the list cannot be negative", "position")
	case strings.TrimSpace(r.Name) == "":
		return invalid("a rule needs a name that says what it finds", "name")
	case utf8.RuneCountInString(r.Name) > ruleNameLimit:
		return invalid("a rule's name is at most "+strconv.Itoa(ruleNameLimit)+" characters", "name")
	}
	kind, found := set.ByID(r.TypeID)
	if !found || kind.SiteID != r.SiteID {
		return invalid("the type a rule gives must be one of the site's page types", "typeId")
	}
	if r.Kind == KindFallback {
		if !r.Conditions.Empty() {
			return invalid("the fallback catches what no rule matched, so it has no conditions", "conditions")
		}
		if !Fits(kind.Preset.Role(), pagemap.WPPage) {
			return invalid("the fallback gives an ordinary page its type, so it cannot be a product or a store category", "typeId")
		}
		return nil
	}
	if r.Conditions.Empty() {
		return invalid("a rule needs at least one condition, or it would match every page", "conditions")
	}
	return r.Conditions.validate(set, r.SiteID)
}

func (c Conditions) validate(set Set, siteID string) error {
	for i, postType := range c.PostTypes {
		if err := plainValue(postType, "conditions.postTypes["+strconv.Itoa(i)+"]"); err != nil {
			return err
		}
	}
	for i, slug := range c.Slugs {
		if err := plainValue(slug, "conditions.slugs["+strconv.Itoa(i)+"]"); err != nil {
			return err
		}
	}
	for i, match := range c.URLs {
		if err := match.validate("conditions.urls[" + strconv.Itoa(i) + "]"); err != nil {
			return err
		}
	}
	if c.Ancestor != nil {
		if err := c.Ancestor.validate(set, siteID); err != nil {
			return err
		}
	}
	if err := c.Level.validate("conditions.level"); err != nil {
		return err
	}
	if err := c.Words.validate("conditions.words"); err != nil {
		return err
	}
	if c.Listing != nil && (c.Listing.MinProducts < 0 || !c.Listing.Present && c.Listing.MinProducts > 0) {
		return invalid("a minimum number of products asks for a listing that is there", "conditions.listing.minProducts")
	}
	for i, kind := range c.Schema {
		if !kind.Valid() {
			return invalid("the schema type "+string(kind)+" is not one a rule reads", "conditions.schema["+strconv.Itoa(i)+"]")
		}
	}
	if c.MinH2 < 0 {
		return invalid("a number of H2s cannot be negative", "conditions.minH2")
	}
	return nil
}

func plainValue(value, field string) error {
	switch {
	case value == "":
		return invalid("a value in a list cannot be empty", field)
	case utf8.RuneCountInString(value) > ruleValueLimit:
		return invalid("a value is at most "+strconv.Itoa(ruleValueLimit)+" characters", field)
	case strings.ContainsAny(value, "/*"):
		return invalid("a name holds no slash and no star", field)
	default:
		return nil
	}
}

func (m URLMatch) validate(field string) error {
	if !m.Mode.Valid() {
		return invalid("an address is matched by its start, a word in it or a pattern", field+".mode")
	}
	switch m.Mode {
	case URLPrefix:
		if m.Value == "" || !strings.HasPrefix(m.Value, "/") || strings.Contains(m.Value, "*") {
			return invalid("the start of an address is a path such as /blog/", field+".value")
		}
	case URLSegment:
		if err := plainValue(m.Value, field+".value"); err != nil {
			return err
		}
	case URLGlob:
		if _, err := compileGlob(m.Value); err != nil {
			return invalid(err.Error(), field+".value")
		}
	}
	return nil
}

func (a *Ancestor) validate(set Set, siteID string) error {
	switch {
	case (a.PageID == "") == (a.TypeID == ""):
		return invalid("a page above names one page or one type", "conditions.ancestor")
	case a.TypeID != "":
		kind, found := set.ByID(a.TypeID)
		if !found || kind.SiteID != siteID {
			return invalid("the type above must be one of the site's page types", "conditions.ancestor.typeId")
		}
	}
	return nil
}

func (r *Range) validate(field string) error {
	switch {
	case r == nil:
		return nil
	case r.Min == nil && r.Max == nil:
		return invalid("a range names a lowest or a highest value", field)
	case r.Min != nil && *r.Min < 0, r.Max != nil && *r.Max < 0:
		return invalid("a range cannot go below zero", field)
	case r.Min != nil && r.Max != nil && *r.Min > *r.Max:
		return invalid("a range's lowest value is above its highest", field)
	default:
		return nil
	}
}

func (r *Range) holds(value int) bool {
	return r == nil || (r.Min == nil || value >= *r.Min) && (r.Max == nil || value <= *r.Max)
}

func Fits(role Role, wpType pagemap.WPType) bool {
	switch {
	case wpType == pagemap.WPProduct:
		return role == RoleProduct
	case wpType.Term():
		return role == RoleArchive
	default:
		return role != RoleProduct && role != RoleArchive
	}
}
