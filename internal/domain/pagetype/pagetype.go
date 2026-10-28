package pagetype

import (
	"slices"
	"strings"
	"time"

	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/render"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

type Role string

const (
	RoleHub       Role = "hub"
	RoleEntity    Role = "entity"
	RoleProduct   Role = "product"
	RoleArchive   Role = "archive"
	RoleCatalog   Role = "catalog"
	RoleTechnical Role = "technical"
)

func Roles() []Role {
	return []Role{RoleHub, RoleEntity, RoleProduct, RoleArchive, RoleCatalog, RoleTechnical}
}

func (r Role) Writable() bool {
	return r == RoleHub || r == RoleEntity || r == RoleProduct
}

type Preset string

const (
	Landing       Preset = "landing"
	Category      Preset = "category"
	Guide         Preset = "guide"
	Info          Preset = "info"
	Product       Preset = "product"
	StoreCategory Preset = "store_category"
	Catalog       Preset = "catalog"
	Technical     Preset = "technical"
	Hub           Preset = "hub"
	Entity        Preset = "entity"
)

var presetNames = map[Preset]string{
	Landing: "Landing", Category: "Category", Guide: "Guide", Info: "Info", Product: "Product",
	StoreCategory: "Store category", Catalog: "Catalog", Technical: "Technical", Hub: "Hub", Entity: "Entity",
}

var presetRoles = map[Preset]Role{
	Landing: RoleHub, Category: RoleHub, Guide: RoleEntity, Info: RoleEntity, Product: RoleProduct,
	StoreCategory: RoleArchive, Catalog: RoleCatalog, Technical: RoleTechnical, Hub: RoleHub, Entity: RoleEntity,
}

var presetTemplates = map[Preset]string{
	Landing: "Hub", Category: "Hub", Guide: "Guide", Info: "Guide", Product: "Product", Hub: "Hub", Entity: "Guide",
}

var rolePresets = map[Role][]Preset{
	RoleHub: {Category, Hub, Landing}, RoleEntity: {Guide, Entity, Info}, RoleProduct: {Product},
	RoleArchive: {StoreCategory}, RoleCatalog: {Catalog}, RoleTechnical: {Technical},
}

func Presets() []Preset {
	return []Preset{Landing, Category, Guide, Info, Product, StoreCategory, Catalog, Technical, Hub, Entity}
}

func Seeded() []Preset {
	return []Preset{Landing, Category, Guide, Info, Product, StoreCategory, Catalog, Technical}
}

func (p Preset) Legacy() bool {
	return p == Hub || p == Entity
}

func (p Preset) Valid() bool {
	_, known := presetNames[p]
	return known
}

func (p Preset) DefaultName() string {
	return presetNames[p]
}

func (p Preset) Role() Role {
	return presetRoles[p]
}

func (p Preset) Writable() bool {
	return p.Role().Writable()
}

func (p Preset) DefaultTemplate() string {
	return presetTemplates[p]
}

type Type struct {
	ID         string
	SiteID     string
	Name       string
	Preset     Preset
	TemplateID *string
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

func (t Type) Validate() error {
	switch {
	case t.ID == "":
		return invalid("page type id must not be empty", "id")
	case t.SiteID == "":
		return invalid("a page type belongs to a site", "siteId")
	case strings.TrimSpace(t.Name) == "":
		return invalid("page type name must not be empty", "name")
	case !t.Preset.Valid():
		return invalid("page type preset is not recognized", "preset")
	default:
		return nil
	}
}

type Assignment struct {
	TypeID string
	Source pagemap.TypeSource
	Reason pagemap.TypeReason
	RuleID string
}

type Facts struct {
	PageID    string
	Path      string
	Slug      string
	WPType    pagemap.WPType
	Technical pagemap.TechnicalReason
	FrontPage bool
	Rendered  bool
	Listing   bool
	Products  int
	Schema    []string
	Words     int
	H2        int
	Level     int
	Ancestors []string
	Current   Assignment
}

type Rendered struct {
	Listing bool
	Schema  []string
	Words   int
	H2      int
}

func RenderedOf(record *render.Record) Rendered {
	var out Rendered
	for _, region := range record.Regions {
		if region.Kind == render.RegionListing && region.Inside {
			out.Listing = true
			break
		}
	}
	for kind := range record.JSONLD {
		out.Schema = append(out.Schema, kind)
	}
	slices.Sort(out.Schema)
	for _, heading := range record.Headings {
		if heading.Level == 2 && !heading.Generated {
			out.H2++
		}
	}
	out.Words = record.WordsContent
	return out
}

func FactsOf(page pagemap.Page, rendered *Rendered) Facts {
	facts := Facts{
		PageID: page.ID, Path: page.Path, Slug: page.Slug, WPType: page.WPType, Technical: page.TechnicalReason,
		Rendered: rendered != nil, Current: AssignmentOf(page),
	}
	if rendered != nil {
		facts.Listing, facts.Schema, facts.Words, facts.H2 = rendered.Listing, rendered.Schema, rendered.Words, rendered.H2
	}
	return facts
}

func AssignmentOf(page pagemap.Page) Assignment {
	held := Assignment{Source: page.TypeSource, Reason: page.TypeReason, RuleID: page.TypeRuleID}
	if page.PageTypeID != nil {
		held.TypeID = *page.PageTypeID
	}
	return held
}

func Assign(facts Facts) (Role, pagemap.TypeReason, bool) {
	switch {
	case facts.WPType == pagemap.WPProduct:
		return RoleProduct, pagemap.TypeByProduct, true
	case facts.WPType.Term():
		return RoleArchive, pagemap.TypeByStoreCategory, true
	case facts.Technical == pagemap.TechnicalShop:
		return RoleCatalog, pagemap.TypeByShop, true
	case facts.Technical != pagemap.TechnicalNone:
		return RoleTechnical, pagemap.TypeByTechnical, true
	case !facts.Rendered:
		return "", pagemap.TypeByNone, false
	case facts.Listing:
		return RoleHub, pagemap.TypeByListing, true
	default:
		return RoleEntity, pagemap.TypeByDefault, true
	}
}

func invalid(message, field string) *errors.Error {
	return errors.New(errors.Invalid, message).WithDetail("field", field)
}
