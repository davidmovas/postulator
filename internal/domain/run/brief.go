package run

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"maps"
	"strconv"
	"strings"
	"unicode/utf8"
)

const (
	briefLineRunes   = 1000
	notesRunes       = 2000
	referenceRunes   = 20000
	styleSampleRunes = 4000
	listItems        = 10
	keywordItems     = 20
	itemRunes        = 200
	mustLinks        = 20
	wordsCeiling     = 20000
)

type Brief struct {
	Goal        string `json:"goal,omitempty"`
	Focus       string `json:"focus,omitempty"`
	Tone        string `json:"tone,omitempty"`
	Avoid       string `json:"avoid,omitempty"`
	Reference   string `json:"reference,omitempty"`
	StyleSample string `json:"styleSample,omitempty"`
}

type MustLink struct {
	PageID string `json:"pageId"`
	Anchor string `json:"anchor,omitempty"`
}

type Directive struct {
	PageID      string     `json:"pageId"`
	Notes       string     `json:"notes,omitempty"`
	AddSections []string   `json:"addSections,omitempty"`
	Keywords    []string   `json:"keywords,omitempty"`
	Anchors     []string   `json:"anchors,omitempty"`
	MustLink    []MustLink `json:"mustLink,omitempty"`
	MinWords    int        `json:"minWords,omitempty"`
	MaxWords    int        `json:"maxWords,omitempty"`
}

func (b Brief) Empty() bool {
	return b == Brief{}
}

func (b Brief) Hash() string {
	if b.Empty() {
		return ""
	}
	return digest(b)
}

func (d Directive) Empty() bool {
	return d.Notes == "" && len(d.AddSections) == 0 && len(d.Keywords) == 0 && len(d.Anchors) == 0 &&
		len(d.MustLink) == 0 && d.MinWords == 0 && d.MaxWords == 0
}

func (d Directive) Hash() string {
	if d.Empty() {
		return ""
	}
	return digest(d)
}

func HashedParams(params map[string]any, brief Brief, directive Directive) map[string]any {
	briefHash, directiveHash := brief.Hash(), directive.Hash()
	if briefHash == "" && directiveHash == "" {
		return params
	}
	out := make(map[string]any, len(params)+2)
	maps.Copy(out, params)
	out["runBrief"], out["pageDirective"] = briefHash, directiveHash
	return out
}

func digest(value any) string {
	encoded, err := json.Marshal(value)
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(encoded)
	return hex.EncodeToString(sum[:])
}

func ValidateBrief(brief Brief, directives []Directive) error {
	for _, held := range []struct {
		field, text string
		limit       int
	}{
		{"brief.goal", brief.Goal, briefLineRunes}, {"brief.focus", brief.Focus, briefLineRunes},
		{"brief.tone", brief.Tone, briefLineRunes}, {"brief.avoid", brief.Avoid, briefLineRunes},
		{"brief.reference", brief.Reference, referenceRunes}, {"brief.styleSample", brief.StyleSample, styleSampleRunes},
	} {
		if err := withinRunes(held.text, held.limit, held.field); err != nil {
			return err
		}
	}
	seen := make(map[string]struct{}, len(directives))
	for i := range directives {
		if err := validateDirective(directives[i], seen); err != nil {
			return err
		}
	}
	return nil
}

func validateDirective(d Directive, seen map[string]struct{}) error {
	pageID := strings.TrimSpace(d.PageID)
	if pageID == "" {
		return invalid("a directive names the page it is for", "directives.pageId")
	}
	if _, twice := seen[pageID]; twice {
		return invalid("a page takes one directive, and "+pageID+" has two", "directives.pageId").WithDetail("pageId", pageID)
	}
	seen[pageID] = struct{}{}
	if err := withinRunes(d.Notes, notesRunes, "directives.notes"); err != nil {
		return err
	}
	for _, list := range []struct {
		field string
		items []string
		limit int
	}{
		{"directives.addSections", d.AddSections, listItems}, {"directives.keywords", d.Keywords, keywordItems},
		{"directives.anchors", d.Anchors, keywordItems},
	} {
		if len(list.items) > list.limit {
			return invalid("at most "+strconv.Itoa(list.limit)+" items fit "+list.field, list.field)
		}
		for _, item := range list.items {
			if err := withinRunes(item, itemRunes, list.field); err != nil {
				return err
			}
		}
	}
	if len(d.MustLink) > mustLinks {
		return invalid("at most "+strconv.Itoa(mustLinks)+" must-link targets fit one page", "directives.mustLink")
	}
	for _, link := range d.MustLink {
		target := strings.TrimSpace(link.PageID)
		if target == "" || target == pageID {
			return invalid("a must-link target is another page, named by its id", "directives.mustLink")
		}
		if err := withinRunes(link.Anchor, itemRunes, "directives.mustLink"); err != nil {
			return err
		}
	}
	switch {
	case d.MinWords < 0 || d.MinWords > wordsCeiling:
		return invalid("the least words is between 0 and "+strconv.Itoa(wordsCeiling), "directives.minWords")
	case d.MaxWords < 0 || d.MaxWords > wordsCeiling:
		return invalid("the most words is between 0 and "+strconv.Itoa(wordsCeiling), "directives.maxWords")
	case d.MaxWords > 0 && d.MaxWords < d.MinWords:
		return invalid("the most words is no fewer than the least", "directives.maxWords")
	}
	return nil
}

func withinRunes(text string, limit int, field string) error {
	if utf8.RuneCountInString(text) > limit {
		return invalid("at most "+strconv.Itoa(limit)+" characters fit "+field, field)
	}
	return nil
}
