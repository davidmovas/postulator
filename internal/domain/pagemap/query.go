package pagemap

type Sort string

const (
	SortCreatedAt Sort = "createdAt"
	SortPath      Sort = "path"
	SortTitle     Sort = "title"
)

func (s Sort) Valid() bool {
	switch s {
	case SortCreatedAt, SortPath, SortTitle:
		return true
	default:
		return false
	}
}

type BodyFilter string

const (
	BodyAny     BodyFilter = ""
	BodyWith    BodyFilter = "with"
	BodyWithout BodyFilter = "without"
)

func (b BodyFilter) Valid() bool {
	return b == BodyAny || b == BodyWith || b == BodyWithout
}

type Query struct {
	SiteID     string
	Status     *Status
	EntityIDs  []string
	Unmapped   bool
	PathPrefix string
	PageTypeID string
	Body       BodyFilter
	Findings   bool
	Search     string
	Trail      string
	Sort       Sort
	Desc       bool
}

func (q Query) ReadsMap() bool {
	return q.PageTypeID != "" || q.Search != "" || q.Trail != ""
}
