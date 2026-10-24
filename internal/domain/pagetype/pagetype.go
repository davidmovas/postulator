package pagetype

import (
	"strings"
	"time"

	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/render"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

type Preset string

const (
	Hub           Preset = "hub"
	Entity        Preset = "entity"
	Product       Preset = "product"
	StoreCategory Preset = "store_category"
	Catalog       Preset = "catalog"
	Technical     Preset = "technical"
)

var presetNames = map[Preset]string{
	Hub: "Hub", Entity: "Entity", Product: "Product", StoreCategory: "Store category", Catalog: "Catalog", Technical: "Technical",
}

var presetTemplates = map[Preset]string{Hub: "Hub", Entity: "Guide", Product: "Product"}

func Presets() []Preset {
	return []Preset{Hub, Entity, Product, StoreCategory, Catalog, Technical}
}

func (p Preset) Valid() bool {
	_, known := presetNames[p]
	return known
}

func (p Preset) DefaultName() string {
	return presetNames[p]
}

func (p Preset) Writable() bool {
	return p == Hub || p == Entity || p == Product
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

type Facts struct {
	WPType    pagemap.WPType
	Technical pagemap.TechnicalReason
	Rendered  bool
	Listing   bool
}

func FactsOf(page pagemap.Page, record *render.Record) Facts {
	facts := Facts{WPType: page.WPType, Technical: page.TechnicalReason, Rendered: record != nil}
	if record == nil {
		return facts
	}
	for _, region := range record.Regions {
		if region.Kind == render.RegionListing && region.Inside {
			facts.Listing = true
			break
		}
	}
	return facts
}

func Assign(facts Facts) (Preset, pagemap.TypeReason, bool) {
	switch {
	case facts.WPType == pagemap.WPProduct:
		return Product, pagemap.TypeByProduct, true
	case facts.WPType.Term():
		return StoreCategory, pagemap.TypeByStoreCategory, true
	case facts.Technical == pagemap.TechnicalShop:
		return Catalog, pagemap.TypeByShop, true
	case facts.Technical != pagemap.TechnicalNone:
		return Technical, pagemap.TypeByTechnical, true
	case !facts.Rendered:
		return "", pagemap.TypeByNone, false
	case facts.Listing:
		return Hub, pagemap.TypeByListing, true
	default:
		return Entity, pagemap.TypeByDefault, true
	}
}

func invalid(message, field string) *errors.Error {
	return errors.New(errors.Invalid, message).WithDetail("field", field)
}
