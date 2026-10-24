package finding

type Query struct {
	SiteID     string
	Status     Status
	Code       Code
	PageID     string
	PageTypeID string
	Under      string
}

func (q Query) Placed() bool {
	return q.PageTypeID != "" || q.Under != ""
}
