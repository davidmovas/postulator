package pagemap

import "time"

type Identity struct {
	PageID      string
	URL         string
	Status      string
	WPType      WPType
	WPParentID  *int64
	ModifiedGMT *time.Time
	ObservedAt  time.Time
}

type Term struct {
	Taxonomy     string `json:"taxonomy"`
	TermID       int64  `json:"termId"`
	Name         string `json:"name"`
	Slug         string `json:"slug"`
	ParentTermID int64  `json:"parentTermId"`
}
