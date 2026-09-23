package pagemap

const (
	FieldPath   = "path"
	FieldSlug   = "slug"
	FieldStatus = "status"
	FieldTitle  = "title"
	FieldH1     = "h1"
)

type Observed struct {
	Link   string
	Slug   string
	Status string
	Title  string
	H1     string
}

func (o Observed) Empty() bool {
	return o == Observed{}
}

type Mismatch struct {
	Field   string
	Planned string
	Actual  string
}

func StatusFromWordPress(wordpress string) Status {
	if wordpress == "publish" {
		return StatusPublished
	}
	return StatusExists
}

func (p Page) Mismatches() []Mismatch {
	found := make([]Mismatch, 0, 3)
	if p.Observed.Link != "" {
		found = differ(found, FieldPath, p.Path, observedPath(p.Observed.Link))
	}
	if p.Observed.Slug != "" {
		found = differ(found, FieldSlug, p.Slug, p.Observed.Slug)
	}
	if p.Observed.Status != "" && p.asked() {
		found = differ(found, FieldStatus,
			string(p.Status), string(StatusFromWordPress(p.Observed.Status)))
	}
	if p.Title != "" && p.Observed.Title != "" {
		found = differ(found, FieldTitle, p.Title, p.Observed.Title)
	}
	if p.H1 != "" && p.Observed.H1 != "" {
		found = differ(found, FieldH1, p.H1, p.Observed.H1)
	}
	return found
}

func (p Page) asked() bool {
	return p.Status == StatusPublished || p.Status == StatusExists
}

func differ(into []Mismatch, field, planned, actual string) []Mismatch {
	if planned == actual {
		return into
	}
	return append(into, Mismatch{Field: field, Planned: planned, Actual: actual})
}

func observedPath(link string) string {
	path, err := NormalizePath(link)
	if err != nil {
		return link
	}
	return path
}
