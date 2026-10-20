package pagemap

import (
	"html"
	"net/http"
	"strings"

	"github.com/davidmovas/postulator/internal/domain/graph"
)

type TrailSource string

const (
	TrailJSONLD TrailSource = "jsonld"
	TrailNav    TrailSource = "nav"
)

type Crumb struct {
	Name   string `json:"name"`
	Path   string `json:"path"`
	Href   string `json:"href,omitempty"`
	Final  string `json:"final,omitempty"`
	Status int    `json:"status,omitempty"`
}

func (c Crumb) Key() string {
	if c.Path != "" {
		return c.Path
	}
	return c.Name
}

func (c Crumb) Linked() bool {
	return c.Href != "" || c.Path != ""
}

func (c Crumb) Dead() bool {
	return c.Status >= http.StatusBadRequest
}

type Trail struct {
	Source TrailSource `json:"source"`
	Crumbs []Crumb     `json:"crumbs"`
}

type ParentMethod string

const (
	MethodHref     ParentMethod = "href"
	MethodRedirect ParentMethod = "redirect"
	MethodName     ParentMethod = "name"
	MethodNone     ParentMethod = ""
)

func (m ParentMethod) Valid() bool {
	switch m {
	case MethodHref, MethodRedirect, MethodName, MethodNone:
		return true
	default:
		return false
	}
}

type ParentSignals struct {
	Trails     []Trail
	WPParentID *int64
	Landings   map[string]string
}

type Ambiguity struct {
	Crumb      Crumb
	Candidates []string
}

type Twin struct {
	Crumb Crumb
	Page  string
	Terms []string
}

type Derivation struct {
	ParentID   *string
	Reason     ParentReason
	Method     ParentMethod
	Trail      *Trail
	Unresolved []Crumb
	Ambiguous  []Ambiguity
	Twins      []Twin
}

type rankedTrail struct {
	trail  Trail
	reason ParentReason
}

type outcome struct {
	parent *Page
	method ParentMethod
	weak   bool
}

func DeriveParent(page Page, evidence ParentSignals, index Index) Derivation {
	d := Derivation{Reason: ParentNone, Unresolved: make([]Crumb, 0), Ambiguous: make([]Ambiguity, 0), Twins: make([]Twin, 0)}
	if page.Path == RootPath {
		return d
	}

	var weak *rankedTrail
	var weakParent *Page
	for _, ranked := range rankTrails(evidence.Trails, index) {
		decided := d.decide(page, ranked.trail, evidence, index)
		switch {
		case decided.parent == nil:
		case !decided.weak:
			d.settle(*decided.parent, ranked, decided.method)
			return d
		case weak == nil:
			chosen := ranked
			weak, weakParent = &chosen, decided.parent
		}
	}
	if evidence.WPParentID != nil {
		if parent, found := index.ByWordPressID(page.WPType, *evidence.WPParentID); found && parent.ID != page.ID {
			d.ParentID = &parent.ID
			d.Reason = ParentWP
			return d
		}
	}
	if weak != nil {
		d.settle(*weakParent, *weak, MethodHref)
		return d
	}
	if parent := index.PathParentID(page); parent != nil {
		d.ParentID = parent
		d.Reason = ParentURL
	}
	return d
}

func (d *Derivation) settle(parent Page, ranked rankedTrail, method ParentMethod) {
	chosen := ranked.trail
	d.ParentID = &parent.ID
	d.Reason = ranked.reason
	d.Method = method
	d.Trail = &chosen
}

func rankTrails(trails []Trail, index Index) []rankedTrail {
	lists := make([]Trail, 0, len(trails))
	navs := make([]Trail, 0, len(trails))
	for _, t := range trails {
		if t.Source == TrailJSONLD {
			lists = append(lists, t)
		} else {
			navs = append(navs, t)
		}
	}
	ranked := make([]rankedTrail, 0, len(trails))
	if best, ok := bestTrail(lists, navs, index); ok {
		ranked = append(ranked, rankedTrail{trail: lists[best], reason: ParentJSONLD})
		for i := range lists {
			if i != best {
				ranked = append(ranked, rankedTrail{trail: lists[i], reason: ParentJSONLD})
			}
		}
	}
	for i := range navs {
		ranked = append(ranked, rankedTrail{trail: navs[i], reason: ParentARIA})
	}
	return ranked
}

func (d *Derivation) decide(page Page, t Trail, evidence ParentSignals, index Index) outcome {
	front, hasFront := index.ByPath(RootPath)
	remaining := make([]Crumb, 0, len(t.Crumbs))
	for _, crumb := range t.Crumbs {
		if crumb.Path == page.Path {
			break
		}
		if crumb.Path == RootPath {
			if hasFront {
				remaining = append(remaining, crumb)
			}
			continue
		}
		if resolved, found := index.ByPath(crumb.Path); crumb.Path != "" && found && resolved.TechnicalReason != TechnicalNone {
			continue
		}
		remaining = append(remaining, crumb)
	}
	if len(remaining) == 0 {
		return outcome{}
	}

	nearest := remaining[len(remaining)-1]
	if nearest.Path == RootPath {
		return outcome{parent: &front, method: MethodHref, weak: true}
	}
	if parent, method, ok := d.resolve(page, nearest, evidence, index); ok {
		return outcome{parent: &parent, method: method}
	}
	return outcome{}
}

func (d *Derivation) resolve(page Page, crumb Crumb, evidence ParentSignals, index Index) (Page, ParentMethod, bool) {
	if crumb.Path != "" {
		if target, found := index.ByPath(crumb.Path); found && target.ID != page.ID {
			if landed, moved := evidence.Landings[target.Path]; moved {
				if final, ok := landing(page, crumb, landed, index); ok {
					return final, MethodRedirect, true
				}
			}
			return target, MethodHref, true
		}
		if crumb.Final != "" && crumb.Final != crumb.Path {
			if final, ok := landing(page, crumb, crumb.Final, index); ok {
				return final, MethodRedirect, true
			}
		}
	}
	if crumb.Linked() && (crumb.Path == "" || !crumb.Dead()) {
		d.unresolved(crumb)
		return Page{}, MethodNone, false
	}

	named := index.ByName(crumb.Name)
	pages := make([]Page, 0, len(named))
	terms := make([]Page, 0, len(named))
	for i := range named {
		switch {
		case named[i].ID == page.ID || descends(named[i].Path, page.Path):
		case named[i].WPType.Term():
			terms = append(terms, named[i])
		default:
			pages = append(pages, named[i])
		}
	}
	candidates := pages
	if len(pages) == 0 {
		candidates = terms
	}
	switch len(candidates) {
	case 1:
		if len(pages) == 1 && len(terms) > 0 {
			d.twin(crumb, pages[0], terms)
		}
		return candidates[0], MethodName, true
	case 0:
		d.unresolved(crumb)
	default:
		d.ambiguous(crumb, candidates)
	}
	return Page{}, MethodNone, false
}

func (d *Derivation) twin(crumb Crumb, chosen Page, terms []Page) {
	ids := make([]string, 0, len(terms))
	for i := range terms {
		ids = append(ids, terms[i].ID)
	}
	d.Twins = append(d.Twins, Twin{Crumb: crumb, Page: chosen.ID, Terms: ids})
}

func landing(page Page, crumb Crumb, landed string, index Index) (Page, bool) {
	if landed == RootPath || landed == page.Path || descends(landed, page.Path) {
		return Page{}, false
	}
	target, found := index.ByPath(landed)
	if !found || target.ID == page.ID || !carriesName(target, crumb.Name, index.names[target.ID]) {
		return Page{}, false
	}
	return target, true
}

func descends(path, ancestor string) bool {
	return ancestor != RootPath && path != ancestor && strings.HasPrefix(path, ancestor)
}

func nameKey(name string) string {
	return graph.Key(html.UnescapeString(name))
}

func carriesName(page Page, name string, known []string) bool {
	key := nameKey(name)
	if key == "" {
		return false
	}
	for _, held := range append([]string{page.Title, page.H1}, known...) {
		if held != "" && nameKey(held) == key {
			return true
		}
	}
	return false
}

func (d *Derivation) unresolved(crumb Crumb) {
	for _, held := range d.Unresolved {
		if held.Key() == crumb.Key() {
			return
		}
	}
	d.Unresolved = append(d.Unresolved, crumb)
}

func (d *Derivation) ambiguous(crumb Crumb, candidates []Page) {
	for _, held := range d.Ambiguous {
		if held.Crumb.Key() == crumb.Key() {
			return
		}
	}
	ids := make([]string, 0, len(candidates))
	for i := range candidates {
		ids = append(ids, candidates[i].ID)
	}
	d.Ambiguous = append(d.Ambiguous, Ambiguity{Crumb: crumb, Candidates: ids})
}

func bestTrail(lists, navs []Trail, index Index) (int, bool) {
	if len(lists) == 0 {
		return 0, false
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
		return best, true
	}
	for i := range lists {
		if resolving(lists[i], index) != bestScore {
			continue
		}
		for j := range navs {
			if samePaths(lists[i], navs[j]) {
				return i, true
			}
		}
	}
	return best, true
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
