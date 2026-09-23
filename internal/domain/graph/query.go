package graph

type EntitySort string

const (
	EntitySortCreatedAt EntitySort = "createdAt"
	EntitySortName      EntitySort = "name"
)

func (s EntitySort) Valid() bool {
	switch s {
	case EntitySortCreatedAt, EntitySortName:
		return true
	default:
		return false
	}
}

type EntityQuery struct {
	SiteID           string
	Kind             *Kind
	HasCanonicalPage *bool
	NamePrefix       string
	Sort             EntitySort
	Desc             bool
}

type EdgeQuery struct {
	SiteID   string
	Kind     *EdgeKind
	Status   *EdgeStatus
	EntityID string
	Desc     bool
}
