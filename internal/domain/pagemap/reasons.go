package pagemap

import "strings"

type ParentReason string

const (
	ParentJSONLD   ParentReason = "jsonld_breadcrumb"
	ParentARIA     ParentReason = "aria_breadcrumb"
	ParentWP       ParentReason = "wp_parent"
	ParentURL      ParentReason = "url"
	ParentWorkbook ParentReason = "workbook"
	ParentNone     ParentReason = "none"
)

func (r ParentReason) Valid() bool {
	switch r {
	case ParentJSONLD, ParentARIA, ParentWP, ParentURL, ParentWorkbook, ParentNone:
		return true
	default:
		return false
	}
}

type ParentSource string

const (
	ParentFromSite     ParentSource = ""
	ParentFromWorkbook ParentSource = "workbook"
)

func (s ParentSource) Valid() bool {
	return s == ParentFromSite || s == ParentFromWorkbook
}

type TypeSource string

const (
	TypeSourceNone   TypeSource = ""
	TypeSourceAuto   TypeSource = "auto"
	TypeSourceManual TypeSource = "manual"
)

func (s TypeSource) Valid() bool {
	return s == TypeSourceNone || s == TypeSourceAuto || s == TypeSourceManual
}

type TypeReason string

const (
	TypeByNone          TypeReason = ""
	TypeByProduct       TypeReason = "product"
	TypeByStoreCategory TypeReason = "store_category"
	TypeByShop          TypeReason = "shop"
	TypeByTechnical     TypeReason = "technical"
	TypeByListing       TypeReason = "listing"
	TypeByDefault       TypeReason = "default"
	TypeByKind          TypeReason = "kind"
	TypeByUser          TypeReason = "user"
	TypeByAI            TypeReason = "ai"
	TypeByImport        TypeReason = "import"
	TypeByRule          TypeReason = "rule"
	TypeByUnmatched     TypeReason = "unmatched"
)

func (r TypeReason) Valid() bool {
	switch r {
	case TypeByNone, TypeByProduct, TypeByStoreCategory, TypeByShop, TypeByTechnical, TypeByListing, TypeByDefault,
		TypeByKind, TypeByUser, TypeByAI, TypeByImport, TypeByRule, TypeByUnmatched:
		return true
	default:
		return false
	}
}

func (r TypeReason) Manual() bool {
	return r == TypeByUser || r == TypeByAI || r == TypeByImport
}

func (r TypeReason) RulePlaced() bool {
	return r == TypeByRule || r == TypeByUnmatched
}

type TechnicalReason string

const (
	TechnicalPosts     TechnicalReason = "posts_page"
	TechnicalPrivacy   TechnicalReason = "privacy"
	TechnicalShop      TechnicalReason = "shop"
	TechnicalCart      TechnicalReason = "cart"
	TechnicalCheckout  TechnicalReason = "checkout"
	TechnicalAccount   TechnicalReason = "account"
	TechnicalTerms     TechnicalReason = "terms"
	TechnicalShortcode TechnicalReason = "shortcode"
	TechnicalNoindex   TechnicalReason = "noindex"
	TechnicalWorkbook  TechnicalReason = "workbook"
	TechnicalManual    TechnicalReason = "manual"
	TechnicalNone      TechnicalReason = ""
)

func (r TechnicalReason) Valid() bool {
	switch r {
	case TechnicalPosts, TechnicalPrivacy, TechnicalShop, TechnicalCart, TechnicalCheckout,
		TechnicalAccount, TechnicalTerms, TechnicalShortcode, TechnicalNoindex, TechnicalWorkbook, TechnicalManual, TechnicalNone:
		return true
	default:
		return false
	}
}

func (r TechnicalReason) Observed() bool {
	return r.Valid() && r != TechnicalNone && r != TechnicalWorkbook && r != TechnicalManual
}

type ReadOnlyReason string

const (
	ReadOnlyTerm ReadOnlyReason = "term"
	ReadOnlyNone ReadOnlyReason = ""
)

const builderPrefix = "builder:"

func ReadOnlyBuilder(name string) ReadOnlyReason {
	return ReadOnlyReason(builderPrefix + strings.TrimSpace(name))
}

func (r ReadOnlyReason) Valid() bool {
	switch r {
	case ReadOnlyNone, ReadOnlyTerm:
		return true
	default:
		return strings.HasPrefix(string(r), builderPrefix) && len(r) > len(builderPrefix)
	}
}
