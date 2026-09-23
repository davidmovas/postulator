package site

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
	Status *Status
	Sort   Sort
	Desc   bool
}
