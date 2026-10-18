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

type Source string

const (
	SourceNone   Source = ""
	SourceAuto   Source = "auto"
	SourceManual Source = "manual"
)

func (s Source) Valid() bool {
	return s == SourceNone || s == SourceAuto || s == SourceManual
}

type Reason string

const (
	ReasonNone          Reason = ""
	ReasonProduct       Reason = "product"
	ReasonStoreCategory Reason = "store_category"
	ReasonShop          Reason = "shop"
	ReasonTechnical     Reason = "technical"
	ReasonListing       Reason = "listing"
	ReasonDefault       Reason = "default"
	ReasonKind          Reason = "kind"
	ReasonUser          Reason = "user"
	ReasonAI            Reason = "ai"
	ReasonImport        Reason = "import"
)

func (r Reason) Valid() bool {
	switch r {
	case ReasonNone, ReasonProduct, ReasonStoreCategory, ReasonShop, ReasonTechnical, ReasonListing, ReasonDefault,
		ReasonKind, ReasonUser, ReasonAI, ReasonImport:
		return true
	default:
		return false
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

func Assign(facts Facts) (Preset, Reason, bool) {
	switch {
	case facts.WPType == pagemap.WPProduct:
		return Product, ReasonProduct, true
	case facts.WPType.Term():
		return StoreCategory, ReasonStoreCategory, true
	case facts.Technical == pagemap.TechnicalShop:
		return Catalog, ReasonShop, true
	case facts.Technical != pagemap.TechnicalNone:
		return Technical, ReasonTechnical, true
	case !facts.Rendered:
		return "", ReasonNone, false
	case facts.Listing:
		return Hub, ReasonListing, true
	default:
		return Entity, ReasonDefault, true
	}
}

func invalid(message, field string) *errors.Error {
	return errors.New(errors.Invalid, message).WithDetail("field", field)
}
