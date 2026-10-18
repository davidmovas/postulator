package pagemap

import "slices"

type TrailSource string

const (
	TrailJSONLD TrailSource = "jsonld"
	TrailNav    TrailSource = "nav"
)

type Crumb struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

type Trail struct {
	Source TrailSource `json:"source"`
	Crumbs []Crumb     `json:"crumbs"`
}

type Derivation struct {
	ParentID   *string
	Reason     ParentReason
	Trail      *Trail
	Unresolved []Crumb
}

func DeriveParent(page Page, trails []Trail, wpParentID *int64, index Index) Derivation {
	d := Derivation{Reason: ParentNone, Unresolved: make([]Crumb, 0)}

	lists := make([]Trail, 0, len(trails))
	navs := make([]Trail, 0, len(trails))
	for _, t := range trails {
		if t.Source == TrailJSONLD {
			lists = append(lists, t)
		} else {
			navs = append(navs, t)
		}
	}

	if chosen, ok := bestTrail(lists, navs, index); ok {
		if d.decide(page, chosen, ParentJSONLD, index) {
			return d
		}
	}
	for i := range navs {
		if d.decide(page, navs[i], ParentARIA, index) {
			return d
		}
	}
	if wpParentID != nil {
		if parent, found := index.ByWordPressID(page.WPType, *wpParentID); found && parent.ID != page.ID {
			d.ParentID = &parent.ID
			d.Reason = ParentWP
			return d
		}
	}
	if parent := index.PathParentID(page); parent != nil {
		d.ParentID = parent
		d.Reason = ParentURL
	}
	return d
}

func (d *Derivation) decide(page Page, t Trail, reason ParentReason, index Index) bool {
	remaining := make([]Crumb, 0, len(t.Crumbs))
	for _, crumb := range t.Crumbs {
		if crumb.Path == page.Path {
			break
		}
		if crumb.Path == RootPath {
			continue
		}
		if resolved, found := index.ByPath(crumb.Path); crumb.Path != "" && found && resolved.TechnicalReason != TechnicalNone {
			continue
		}
		remaining = append(remaining, crumb)
	}
	if len(remaining) == 0 {
		return false
	}
	nearest := remaining[len(remaining)-1]
	if nearest.Path != "" {
		if parent, found := index.ByPath(nearest.Path); found && parent.ID != page.ID {
			chosen := t
			d.ParentID = &parent.ID
			d.Reason = reason
			d.Trail = &chosen
			return true
		}
	}
	if !slices.Contains(d.Unresolved, nearest) {
		d.Unresolved = append(d.Unresolved, nearest)
	}
	return false
}

func bestTrail(lists, navs []Trail, index Index) (Trail, bool) {
	if len(lists) == 0 {
		return Trail{}, false
	}
	best := 0
	bestScore := -1
	tied := false
	for i := range lists {
		score := resolving(lists[i], index)
		switch {
		case score > bestScore:
			best, bestScore, tied = i, score, false
		case score == bestScore:
			tied = true
		}
	}
	if !tied {
		return lists[best], true
	}
	for i := range lists {
		if resolving(lists[i], index) != bestScore {
			continue
		}
		for j := range navs {
			if samePaths(lists[i], navs[j]) {
				return lists[i], true
			}
		}
	}
	return lists[best], true
}

func resolving(t Trail, index Index) int {
	count := 0
	for _, crumb := range t.Crumbs {
		if crumb.Path == "" {
			continue
		}
		if _, found := index.ByPath(crumb.Path); found {
			count++
		}
	}
	return count
}

func samePaths(a, b Trail) bool {
	if len(a.Crumbs) != len(b.Crumbs) {
		return false
	}
	for i := range a.Crumbs {
		if a.Crumbs[i].Path != b.Crumbs[i].Path {
			return false
		}
	}
	return true
}
