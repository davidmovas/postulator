package category

import (
	"strings"
	"time"
)

type Taxonomy string

const (
	TaxonomyCategory        Taxonomy = "category"
	TaxonomyProductCategory Taxonomy = "product_cat"
)

func (t Taxonomy) Valid() bool {
	switch t {
	case TaxonomyCategory, TaxonomyProductCategory:
		return true
	default:
		return false
	}
}

type Term struct {
	CategoryID   string
	SiteID       string
	Taxonomy     Taxonomy
	TermID       int64
	ParentTermID int64
	Name         string
	RunID        string
	SeenAt       time.Time
}

func NewTerm(t Term) (Term, error) {
	t.Name = strings.TrimSpace(t.Name)

	switch {
	case t.CategoryID == "":
		return Term{}, invalid("term category id must not be empty", "categoryId")
	case t.SiteID == "":
		return Term{}, invalid("term site id must not be empty", "siteId")
	case !t.Taxonomy.Valid():
		return Term{}, invalid("term taxonomy is not recognized", "taxonomy")
	case t.TermID <= 0:
		return Term{}, invalid("term id must be positive", "termId")
	case t.ParentTermID < 0:
		return Term{}, invalid("parent term id must not be negative", "parentTermId")
	case t.ParentTermID == t.TermID:
		return Term{}, invalid("a term cannot sit under itself", "parentTermId")
	case t.Name == "":
		return Term{}, invalid("term name must not be empty", "name")
	case t.SeenAt.IsZero():
		return Term{}, invalid("a term needs the time it was seen", "seenAt")
	}
	return t, nil
}
