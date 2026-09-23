package pagemap

type Sort string

const (
	SortCreatedAt Sort = "createdAt"
	SortPath      Sort = "path"
)

func (s Sort) Valid() bool {
	switch s {
	case SortCreatedAt, SortPath:
		return true
	default:
		return false
	}
}

type Query struct {
	SiteID     string
	Status     *Status
	EntityID   *string
	Unmapped   bool
	PathPrefix string
	Sort       Sort
	Desc       bool
}
