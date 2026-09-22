package content

import (
	"slices"
	"unicode"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"

	"github.com/davidmovas/postulator/internal/domain/template"
)

type Outcome string

const (
	OutcomeInserted       Outcome = "inserted"
	OutcomeAlreadyLinked  Outcome = "already_linked"
	OutcomeAnchorNotFound Outcome = "anchor_not_found"
	OutcomeCapReached     Outcome = "cap_reached"
	OutcomeForbiddenZone  Outcome = "forbidden_zone"
	OutcomePositionRule   Outcome = "position_rule"
)

type Placement struct {
	Target         LinkTarget `json:"target"`
	Anchor         string     `json:"anchor"`
	ParagraphIndex int        `json:"paragraphIndex"`
}

type Decision struct {
	Target  LinkTarget `json:"target"`
	Anchor  string     `json:"anchor"`
	Outcome Outcome    `json:"outcome"`
	Detail  string     `json:"detail"`
}

type InsertResult struct {
	Placed    []Placement  `json:"placed"`
	Missing   []LinkTarget `json:"missing"`
	Decisions []Decision   `json:"decisions"`
}

var forbiddenZones = []string{
	"a", "h1", "h2", "h3", "h4", "h5", "h6", "code", "pre",
	"script", "style", "textarea", "noscript", "template", "iframe", "svg", "math",
}

func InsertLinks(doc *Document, lc LinkContext, policy template.LinkPolicy) InsertResult {
	return insert(doc, lc, policy, lc.Targets)
}

func InsertTarget(doc *Document, lc LinkContext, policy template.LinkPolicy, only LinkTarget) InsertResult {
	return insert(doc, lc, policy, []LinkTarget{only})
}

func insert(doc *Document, lc LinkContext, policy template.LinkPolicy, wanted []LinkTarget) InsertResult {
	result := InsertResult{
		Placed:    make([]Placement, 0, len(wanted)),
		Missing:   make([]LinkTarget, 0),
		Decisions: make([]Decision, 0, len(wanted)),
	}

	maxLinks := policy.Rules.MaxLinks
	perTarget := max(policy.Rules.MaxPerTarget, 1)
	placed := CountGraphLinks(doc, lc)

	for _, target := range wanted {
		existing := existingFor(doc, lc, target)
		if len(existing) > 0 {
			for _, node := range existing {
				result.Placed = append(result.Placed, Placement{
					Target: target, Anchor: TextOf(node), ParagraphIndex: paragraphOf(doc, node),
				})
			}
			result.Decisions = append(result.Decisions, Decision{
				Target: target, Anchor: TextOf(existing[0]), Outcome: OutcomeAlreadyLinked,
				Detail: "the body already links to this target",
			})
			continue
		}

		if maxLinks > 0 && placed >= maxLinks {
			result.Missing = append(result.Missing, target)
			result.Decisions = append(result.Decisions, Decision{
				Target: target, Outcome: OutcomeCapReached,
				Detail: "the link budget of this page was already spent",
			})
			continue
		}

		inserted := 0
		for inserted < perTarget {
			if maxLinks > 0 && placed >= maxLinks {
				break
			}
			placement, outcome, detail := insertOne(doc, target, policy, inserted)
			if outcome != OutcomeInserted {
				if inserted == 0 {
					result.Decisions = append(result.Decisions, Decision{
						Target: target, Outcome: outcome, Detail: detail,
					})
				}
				break
			}
			result.Placed = append(result.Placed, placement)
			result.Decisions = append(result.Decisions, Decision{
				Target: target, Anchor: placement.Anchor, Outcome: OutcomeInserted,
				Detail: "matched the anchor in the body text",
			})
			inserted++
			placed++
		}

		if inserted == 0 {
			result.Missing = append(result.Missing, target)
		}
	}
	return result
}

func anchorOrder(target LinkTarget, strategy template.AnchorStrategy, placed int) []string {
	shift := 0
	if strategy == template.AnchorRotate && len(target.Anchors) > 1 {
		shift = placed % len(target.Anchors)
	}
	if shift == 0 {
		return target.Anchors
	}
	return append(slices.Clone(target.Anchors[shift:]), target.Anchors[:shift]...)
}

func insertOne(doc *Document, target LinkTarget, policy template.LinkPolicy, placed int) (Placement, Outcome, string) {
	limit := policy.Rules.ParentLinkWithinParagraphs
	restricted := target.Relation == RelationUp && limit > 0

	paragraphs := doc.Paragraphs()
	anchors := anchorOrder(target, policy.AnchorStrategy, placed)
	sawZone := false

	for node := range doc.TextNodes(nil) {
		if inForbiddenZone(node) {
			sawZone = true
			continue
		}

		index := paragraphIndexOf(paragraphs, node)
		if restricted && (index < 0 || index >= limit) {
			continue
		}

		for _, anchor := range anchors {
			start, end, matched := findFold(node.Data, anchor)
			if !matched {
				continue
			}
			splice(node, start, end, target.URL)
			return Placement{Target: target, Anchor: node.Data[start:end], ParagraphIndex: index},
				OutcomeInserted, ""
		}
	}

	if restricted {
		return Placement{}, OutcomePositionRule,
			"a parent link may only be placed in the opening paragraphs, and no anchor occurs there"
	}
	if sawZone {
		return Placement{}, OutcomeAnchorNotFound,
			"no anchor of this target occurs outside headings, code and existing links"
	}
	return Placement{}, OutcomeAnchorNotFound, "no anchor of this target occurs in the body text"
}

func splice(node *html.Node, start, end int, href string) {
	parent := node.Parent
	after := node.NextSibling
	text := node.Data

	anchor := &html.Node{
		Type: html.ElementNode, Data: "a", DataAtom: atom.A,
		Attr: []html.Attribute{{Key: "href", Val: href}},
	}
	anchor.AppendChild(&html.Node{Type: html.TextNode, Data: text[start:end]})

	parent.RemoveChild(node)

	if start > 0 {
		parent.InsertBefore(&html.Node{Type: html.TextNode, Data: text[:start]}, after)
	}
	parent.InsertBefore(anchor, after)
	if end < len(text) {
		parent.InsertBefore(&html.Node{Type: html.TextNode, Data: text[end:]}, after)
	}
}

func FindFold(haystack, needle string) (start, end int, found bool) {
	return findFold(haystack, needle)
}

func findFold(haystack, needle string) (start, end int, found bool) {
	target := []rune(needle)
	if len(target) == 0 {
		return 0, 0, false
	}

	source := []rune(haystack)
	offsets := make([]int, 0, len(source)+1)
	position := 0
	for _, r := range source {
		offsets = append(offsets, position)
		position += len(string(r))
	}
	offsets = append(offsets, position)

	for i := 0; i+len(target) <= len(source); i++ {
		if !foldEqual(source[i:i+len(target)], target) || !bounded(source, i, i+len(target)) {
			continue
		}
		return offsets[i], offsets[i+len(target)], true
	}
	return 0, 0, false
}

func bounded(source []rune, start, end int) bool {
	if start > 0 && wordRune(source[start-1]) && wordRune(source[start]) {
		return false
	}
	return end >= len(source) || !wordRune(source[end]) || !wordRune(source[end-1])
}

func wordRune(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r)
}

func foldEqual(a, b []rune) bool {
	for i := range a {
		if unicode.ToLower(a[i]) != unicode.ToLower(b[i]) {
			return false
		}
	}
	return true
}

func inForbiddenZone(node *html.Node) bool {
	for parent := node.Parent; parent != nil; parent = parent.Parent {
		if parent.Type == html.ElementNode && slices.Contains(forbiddenZones, parent.Data) {
			return true
		}
	}
	return false
}

func paragraphIndexOf(paragraphs []*html.Node, node *html.Node) int {
	for parent := node.Parent; parent != nil; parent = parent.Parent {
		if index := slices.Index(paragraphs, parent); index >= 0 {
			return index
		}
	}
	return -1
}

func paragraphOf(doc *Document, node *html.Node) int {
	return paragraphIndexOf(doc.Paragraphs(), node)
}

func existingFor(doc *Document, lc LinkContext, target LinkTarget) []*html.Node {
	out := make([]*html.Node, 0, 1)
	wanted, ok := canonicalPath(target.URL)
	if !ok {
		return out
	}
	for _, link := range doc.Links() {
		if lc.Resolve(link.Href).Path == wanted {
			out = append(out, link.Node)
		}
	}
	return out
}

func CountGraphLinks(doc *Document, lc LinkContext) int {
	total := 0
	for _, link := range doc.Links() {
		if lc.Resolve(link.Href).Class == ClassGraph {
			total++
		}
	}
	return total
}
