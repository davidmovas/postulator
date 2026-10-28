package run

type SkipReason string

const (
	SkipNoEntity   SkipReason = "no_entity"
	SkipNoTemplate SkipReason = "no_template"
)

func (r SkipReason) Valid() bool {
	switch r {
	case SkipNoEntity, SkipNoTemplate:
		return true
	default:
		return false
	}
}

type Skipped struct {
	PageID   string     `json:"pageId"`
	Path     string     `json:"path"`
	Reason   SkipReason `json:"reason"`
	PageType string     `json:"pageType"`
}

func validSkipped(skipped []Skipped) bool {
	for i := range skipped {
		if skipped[i].PageID == "" || !skipped[i].Reason.Valid() {
			return false
		}
	}
	return true
}
