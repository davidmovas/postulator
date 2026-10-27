package pagetype

import (
	"cmp"
	"slices"
	"strings"

	"github.com/davidmovas/postulator/internal/domain/pagemap"
)

type Mode int

const (
	ModeWalk Mode = iota + 1
	ModePreview
)

type ConditionKind string

const (
	ConditionPostType  ConditionKind = "postType"
	ConditionURL       ConditionKind = "url"
	ConditionSlug      ConditionKind = "slug"
	ConditionAncestor  ConditionKind = "ancestor"
	ConditionLevel     ConditionKind = "level"
	ConditionListing   ConditionKind = "listing"
	ConditionSchema    ConditionKind = "schema"
	ConditionWords     ConditionKind = "words"
	ConditionHeadings  ConditionKind = "h2"
	ConditionTechnical ConditionKind = "technical"
	ConditionFrontPage ConditionKind = "frontPage"
	ConditionShop      ConditionKind = "shop"
)

type Verdict string

const (
	VerdictMet     Verdict = "met"
	VerdictUnmet   Verdict = "unmet"
	VerdictWaiting Verdict = "waiting"
	VerdictBlocked Verdict = "blocked"
)

type ConditionResult struct {
	Condition ConditionKind `json:"condition"`
	Verdict   Verdict       `json:"verdict"`
}

type Outcome struct {
	PageID   string
	Current  Assignment
	TypeID   string
	RuleID   string
	Reason   pagemap.TypeReason
	Eligible bool
	Waiting  bool
	Depends  []string
}

func (o Outcome) Changed() bool {
	if !o.Eligible || o.Waiting || o.TypeID == "" {
		return false
	}
	return o.TypeID != o.Current.TypeID || o.Reason != o.Current.Reason || o.RuleID != o.Current.RuleID ||
		o.Current.Source != pagemap.TypeSourceAuto
}

func (o Outcome) Retyped() bool {
	return o.Changed() && o.TypeID != o.Current.TypeID
}

type Explanation struct {
	Verdict    Verdict
	Conditions []ConditionResult
	Outcome    Outcome
}

type Classifier struct {
	rules    []Rule
	fallback *Rule
	set      Set
	globs    map[string]glob
}

func NewClassifier(rules []Rule, set Set) Classifier {
	c := Classifier{set: set, globs: make(map[string]glob)}
	for i := range rules {
		switch {
		case rules[i].Kind == KindFallback:
			c.fallback = &rules[i]
		case rules[i].Enabled:
			c.rules = append(c.rules, rules[i])
		}
		for _, match := range rules[i].Conditions.URLs {
			if match.Mode != URLGlob {
				continue
			}
			if compiled, err := compileGlob(match.Value); err == nil {
				c.globs[match.Value] = compiled
			}
		}
	}
	slices.SortStableFunc(c.rules, func(a, b Rule) int { return cmp.Or(cmp.Compare(a.Position, b.Position), strings.Compare(a.ID, b.ID)) })
	return c
}

func Eligible(current Assignment, mode Mode) bool {
	switch current.Source {
	case pagemap.TypeSourceNone:
		return true
	case pagemap.TypeSourceAuto:
		return current.Reason.RulePlaced() || mode == ModePreview
	default:
		return false
	}
}

type placed struct {
	typeID  string
	waiting bool
	moved   bool
}

func (c Classifier) Classify(pages []Facts, mode Mode) []Outcome {
	ordered := slices.Clone(pages)
	slices.SortStableFunc(ordered, func(a, b Facts) int { return cmp.Or(cmp.Compare(a.Level, b.Level), strings.Compare(a.Path, b.Path)) })
	state := make(map[string]placed, len(ordered))
	out := make([]Outcome, 0, len(ordered))
	for i := range ordered {
		f := &ordered[i]
		outcome := Outcome{PageID: f.PageID, Current: f.Current, TypeID: f.Current.TypeID, RuleID: f.Current.RuleID, Reason: f.Current.Reason}
		if outcome.Eligible = Eligible(f.Current, mode); outcome.Eligible {
			c.place(f, state, &outcome)
		}
		state[f.PageID] = placed{
			typeID: outcome.TypeID, waiting: outcome.Waiting && outcome.TypeID == "",
			moved: outcome.Eligible && !outcome.Waiting && outcome.TypeID != f.Current.TypeID,
		}
		out = append(out, outcome)
	}
	return out
}

func (c Classifier) place(f *Facts, state map[string]placed, outcome *Outcome) {
	for i := range c.rules {
		verdict, depends := c.judge(&c.rules[i], f, state, nil)
		outcome.Depends = appendNew(outcome.Depends, depends...)
		switch verdict {
		case VerdictWaiting:
			outcome.Waiting = true
			return
		case VerdictMet:
			outcome.TypeID, outcome.RuleID, outcome.Reason = c.rules[i].TypeID, c.rules[i].ID, pagemap.TypeByRule
			return
		}
	}
	outcome.RuleID, outcome.Reason = "", pagemap.TypeByUnmatched
	if c.fallback != nil {
		if kind, found := c.set.ByID(c.fallback.TypeID); found && Fits(kind.Preset.Role(), f.WPType) {
			outcome.TypeID = kind.ID
			return
		}
	}
	role, _, _ := Assign(Facts{WPType: f.WPType, Technical: f.Technical, Rendered: true})
	if kind, found := c.set.ByRole(role); found && Fits(role, f.WPType) {
		outcome.TypeID = kind.ID
		return
	}
	outcome.TypeID, outcome.RuleID, outcome.Reason = f.Current.TypeID, f.Current.RuleID, f.Current.Reason
	outcome.Waiting = true
}

func (c Classifier) judge(rule *Rule, f *Facts, state map[string]placed, results *[]ConditionResult) (verdict Verdict, depends []string) {
	kind, found := c.set.ByID(rule.TypeID)
	if !found || !Fits(kind.Preset.Role(), f.WPType) {
		return VerdictBlocked, nil
	}
	verdict = VerdictMet
	note := func(condition ConditionKind, got Verdict) {
		if results != nil {
			*results = append(*results, ConditionResult{Condition: condition, Verdict: got})
		}
		switch {
		case got == VerdictUnmet:
			verdict = VerdictUnmet
		case got == VerdictWaiting && verdict == VerdictMet:
			verdict = VerdictWaiting
		}
	}
	when := c.rendered(f)
	conditions := &rule.Conditions
	if len(conditions.PostTypes) > 0 {
		note(ConditionPostType, met(slices.Contains(conditions.PostTypes, string(f.WPType))))
	}
	if len(conditions.URLs) > 0 {
		note(ConditionURL, met(c.addressed(conditions.URLs, f.Path)))
	}
	if len(conditions.Slugs) > 0 {
		note(ConditionSlug, met(slices.Contains(conditions.Slugs, strings.ToLower(f.Slug))))
	}
	if conditions.Ancestor != nil {
		got, read := ancestorVerdict(conditions.Ancestor, f, state)
		depends = read
		note(ConditionAncestor, got)
	}
	if conditions.Level != nil {
		note(ConditionLevel, met(conditions.Level.holds(f.Level)))
	}
	if conditions.Listing != nil {
		note(ConditionListing, when(func() bool {
			if !conditions.Listing.Present {
				return !f.Listing
			}
			return f.Listing && f.Products >= conditions.Listing.MinProducts
		}))
	}
	if len(conditions.Schema) > 0 {
		note(ConditionSchema, when(func() bool {
			return slices.ContainsFunc(conditions.Schema, func(kind SchemaType) bool { return slices.Contains(f.Schema, string(kind)) })
		}))
	}
	if conditions.Words != nil {
		note(ConditionWords, when(func() bool { return conditions.Words.holds(f.Words) }))
	}
	if conditions.MinH2 > 0 {
		note(ConditionHeadings, when(func() bool { return f.H2 >= conditions.MinH2 }))
	}
	if conditions.Technical != nil {
		technical := f.Technical != pagemap.TechnicalNone && f.Technical != pagemap.TechnicalShop
		note(ConditionTechnical, met(technical == *conditions.Technical))
	}
	if conditions.FrontPage != nil {
		note(ConditionFrontPage, met(f.FrontPage == *conditions.FrontPage))
	}
	if conditions.Shop != nil {
		note(ConditionShop, met((f.Technical == pagemap.TechnicalShop) == *conditions.Shop))
	}
	return verdict, depends
}

func (c Classifier) rendered(f *Facts) func(func() bool) Verdict {
	return func(test func() bool) Verdict {
		if !f.Rendered {
			return VerdictWaiting
		}
		return met(test())
	}
}

func (c Classifier) addressed(matches []URLMatch, path string) bool {
	segments := segmentsOf(strings.ToLower(path))
	for _, match := range matches {
		switch match.Mode {
		case URLPrefix:
			prefix := segmentsOf(match.Value)
			if len(prefix) <= len(segments) && slices.Equal(segments[:len(prefix)], prefix) {
				return true
			}
		case URLSegment:
			words := wordsOf(match.Value)
			if slices.ContainsFunc(segments, func(segment string) bool { return holdsWords(segment, words) }) {
				return true
			}
		case URLGlob:
			if compiled, known := c.globs[match.Value]; known && compiled.match(strings.ToLower(path)) {
				return true
			}
		}
	}
	return false
}

func ancestorVerdict(ancestor *Ancestor, f *Facts, state map[string]placed) (verdict Verdict, read []string) {
	if ancestor.PageID != "" {
		return met(slices.Contains(f.Ancestors, ancestor.PageID)), nil
	}
	waiting := false
	for _, pageID := range f.Ancestors {
		held := state[pageID]
		if held.moved {
			read = append(read, pageID)
		}
		switch {
		case held.waiting:
			waiting = true
		case held.typeID == ancestor.TypeID:
			return VerdictMet, read
		}
	}
	if waiting {
		return VerdictWaiting, read
	}
	return VerdictUnmet, read
}

func met(ok bool) Verdict {
	if ok {
		return VerdictMet
	}
	return VerdictUnmet
}

func appendNew(into []string, values ...string) []string {
	for _, value := range values {
		if !slices.Contains(into, value) {
			into = append(into, value)
		}
	}
	return into
}

func (c Classifier) Explain(rule Rule, target string, pages []Facts, mode Mode) Explanation {
	outcomes := c.Classify(pages, mode)
	state := make(map[string]placed, len(outcomes))
	var page *Facts
	for i := range pages {
		if pages[i].PageID == target {
			page = &pages[i]
		}
	}
	explained := Explanation{Verdict: VerdictUnmet}
	for i := range outcomes {
		state[outcomes[i].PageID] = placed{
			typeID: outcomes[i].TypeID, waiting: outcomes[i].Waiting && outcomes[i].TypeID == "",
			moved: outcomes[i].Retyped(),
		}
		if outcomes[i].PageID == target {
			explained.Outcome = outcomes[i]
		}
	}
	if page == nil {
		return explained
	}
	for _, match := range rule.Conditions.URLs {
		if _, known := c.globs[match.Value]; match.Mode == URLGlob && !known {
			if compiled, err := compileGlob(match.Value); err == nil {
				c.globs[match.Value] = compiled
			}
		}
	}
	results := make([]ConditionResult, 0)
	explained.Verdict, _ = c.judge(&rule, page, state, &results)
	explained.Conditions = results
	return explained
}

func (c Classifier) Matches(rule Rule, pages []Facts, mode Mode) []string {
	outcomes := c.Classify(pages, mode)
	state := make(map[string]placed, len(outcomes))
	for i := range outcomes {
		state[outcomes[i].PageID] = placed{
			typeID: outcomes[i].TypeID, waiting: outcomes[i].Waiting && outcomes[i].TypeID == "",
			moved: outcomes[i].Retyped(),
		}
	}
	out := make([]string, 0)
	for i := range pages {
		if verdict, _ := c.judge(&rule, &pages[i], state, nil); verdict == VerdictMet {
			out = append(out, pages[i].PageID)
		}
	}
	return out
}
