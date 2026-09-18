package template

type Sort string

const (
	SortCreatedAt Sort = "createdAt"
	SortName      Sort = "name"
)

func (s Sort) Valid() bool {
	switch s {
	case SortCreatedAt, SortName:
		return true
	default:
		return false
	}
}

type Query struct {
	Scope    *Scope
	SiteID   *string
	PageKind string
	Name     string
	Sort     Sort
	Desc     bool
}

type PolicyQuery struct {
	Scope  *Scope
	SiteID *string
	Name   string
	Sort   Sort
	Desc   bool
}
