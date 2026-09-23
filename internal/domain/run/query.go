package run

type Sort string

const (
	SortCreatedAt Sort = "createdAt"
	SortStatus    Sort = "status"
)

func (s Sort) Valid() bool {
	switch s {
	case SortCreatedAt, SortStatus:
		return true
	default:
		return false
	}
}

type Query struct {
	SiteID string
	Status *Status
	Kind   *Kind
	Sort   Sort
	Desc   bool
}

type ItemQuery struct {
	RunID  string
	Status *Status
	Desc   bool
}
