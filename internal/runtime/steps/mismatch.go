package steps

import "github.com/davidmovas/postulator/internal/domain/pagemap"

const (
	FieldPath   = "path"
	FieldSlug   = "slug"
	FieldStatus = "status"
	FieldTitle  = "title"
	FieldH1     = "h1"
	FieldParent = "parent"
)

type Mismatch struct {
	Field   string `json:"field"`
	Planned string `json:"planned"`
	Actual  string `json:"actual"`
}

type answered struct {
	link   string
	slug   string
	status string
	title  string
	h1     string
}

func mismatches(plan pagemap.Page, held answered) []Mismatch {
	found := make([]Mismatch, 0, 3)
	if plan.WPType.StoreAddressed() {
		return statusMismatch(found, plan, held)
	}
	if held.link != "" {
		found = differ(found, FieldPath, plan.Path, pathOf(held.link))
	}
	if plan.Slug != "" && held.slug != "" {
		found = differ(found, FieldSlug, plan.Slug, held.slug)
	}
	found = statusMismatch(found, plan, held)
	if plan.Title != "" && held.title != "" {
		found = differ(found, FieldTitle, plan.Title, held.title)
	}
	if plan.H1 != "" && held.h1 != "" {
		found = differ(found, FieldH1, plan.H1, held.h1)
	}
	return found
}

func statusMismatch(into []Mismatch, plan pagemap.Page, held answered) []Mismatch {
	asked := plan.Status == pagemap.StatusPublished || plan.Status == pagemap.StatusExists
	if held.status == "" || !asked {
		return into
	}
	return differ(into, FieldStatus, string(plan.Status), string(pagemap.StatusFromWordPress(held.status)))
}

func pathOf(link string) string {
	path, err := pagemap.NormalizePath(link)
	if err != nil {
		return link
	}
	return path
}

func differ(into []Mismatch, field, planned, actual string) []Mismatch {
	if planned == actual {
		return into
	}
	return append(into, Mismatch{Field: field, Planned: planned, Actual: actual})
}
