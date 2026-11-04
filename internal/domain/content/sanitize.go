package content

import (
	"slices"
	"strconv"
	"strings"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"

	"github.com/davidmovas/postulator/internal/domain/field"
)

const CodeMarkupStripped = "markup_stripped"

var (
	removedElements = []string{"script", "style", "iframe", "form", "object", "embed", "noscript", "template"}
	demotedHeadings = []string{"h1", "h2"}
)

type Stripped struct {
	Links      int
	Elements   []string
	Attributes int
	Demoted    int
	Sources    []DroppedSource
}

func (s Stripped) Any() bool {
	return s.Links > 0 || len(s.Elements) > 0 || s.Attributes > 0 || s.Demoted > 0
}

type LinkAllowance struct {
	context LinkContext
	policy  field.LinkPolicy
	placed  map[string]int
	total   int
	sources Sources
	cited   sourceCount
}

func NewLinkAllowance(lc LinkContext, policy field.LinkPolicy) *LinkAllowance {
	return &LinkAllowance{context: lc, policy: policy, placed: make(map[string]int), cited: newSourceCount()}
}

func (a *LinkAllowance) WithSources(sources Sources) *LinkAllowance {
	a.sources = sources
	return a
}

func (a *LinkAllowance) Context() LinkContext {
	return a.context
}

type verdict struct {
	href    string
	kept    bool
	dropped *DroppedSource
}

func (a *LinkAllowance) keep(node *html.Node) verdict {
	if a == nil {
		return verdict{}
	}
	href := Attr(node, "href")
	resolved := a.context.Resolve(href)
	if resolved.Class == ClassExternal {
		return a.keepSource(href, TextOf(node))
	}
	if !a.policy.Links || resolved.Class != ClassGraph || !AnchorAllowed(resolved.Target, TextOf(node)) {
		return verdict{}
	}
	if rules := a.policy.Rules; (rules.MaxLinks > 0 && a.total >= rules.MaxLinks) || a.placed[resolved.Target.PageID] >= max(rules.MaxPerTarget, 1) {
		return verdict{}
	}
	a.placed[resolved.Target.PageID]++
	a.total++
	return verdict{href: a.context.Href(resolved.Target), kept: true}
}

func (a *LinkAllowance) keepSource(href, anchor string) verdict {
	words := strings.TrimSpace(anchor)
	target, cited := a.sources.lookup(href)
	reason := DroppedUncited
	switch {
	case !cited:
		target = href
	case bareAddress(words):
		reason = DroppedWords
	default:
		reason = a.cited.refusal(target, a.sources.rules)
	}
	if reason != "" {
		return verdict{dropped: &DroppedSource{Href: target, Anchor: words, Reason: reason}}
	}
	a.cited.add(target)
	return verdict{href: target, kept: true}
}

type demotion struct {
	from []string
	to   atom.Atom
}

var (
	sectionDemotion = demotion{from: demotedHeadings, to: atom.H3}
	blockDemotion   = demotion{from: []string{"h1"}, to: atom.H2}
)

func Sanitize(fragment string, allowance *LinkAllowance) (string, Stripped) {
	return sanitized(fragment, allowance, sectionDemotion)
}

func SanitizeBlock(fragment string, keepH1 bool) (string, Stripped) {
	rule := blockDemotion
	if keepH1 {
		rule = demotion{}
	}
	return sanitized(fragment, nil, rule)
}

func sanitized(fragment string, allowance *LinkAllowance, rule demotion) (string, Stripped) {
	doc, err := Parse(fragment)
	if err != nil {
		return "", Stripped{}
	}

	var stripped Stripped
	scrub(doc.root, allowance, rule, &stripped)

	rendered, err := doc.Render()
	if err != nil {
		return "", stripped
	}
	return rendered, stripped
}

func scrub(parent *html.Node, allowance *LinkAllowance, rule demotion, stripped *Stripped) {
	for child := parent.FirstChild; child != nil; {
		next := child.NextSibling
		if child.Type != html.ElementNode {
			child = next
			continue
		}
		switch {
		case slices.Contains(removedElements, child.Data):
			parent.RemoveChild(child)
			stripped.Elements = append(stripped.Elements, child.Data)
		case child.Data == "a":
			held := allowance.keep(child)
			if held.kept {
				child.Attr = []html.Attribute{{Key: "href", Val: held.href}}
				scrub(child, nil, rule, stripped)
				break
			}
			if held.dropped != nil {
				stripped.Sources = append(stripped.Sources, *held.dropped)
			}
			scrub(child, nil, rule, stripped)
			unwrap(parent, child)
			stripped.Links++
		default:
			if slices.Contains(rule.from, child.Data) {
				child.Data = rule.to.String()
				child.DataAtom = rule.to
				stripped.Demoted++
			}
			child.Attr = keptAttributes(child.Attr, stripped)
			scrub(child, allowance, rule, stripped)
		}
		child = next
	}
}

func unwrap(parent, node *html.Node) {
	for grandchild := node.FirstChild; grandchild != nil; {
		following := grandchild.NextSibling
		node.RemoveChild(grandchild)
		parent.InsertBefore(grandchild, node)
		grandchild = following
	}
	parent.RemoveChild(node)
}

func keptAttributes(attrs []html.Attribute, stripped *Stripped) []html.Attribute {
	kept := make([]html.Attribute, 0, len(attrs))
	for _, attr := range attrs {
		key := strings.ToLower(attr.Key)
		if strings.HasPrefix(key, "on") || strings.HasPrefix(strings.ToLower(strings.TrimSpace(attr.Val)), "javascript:") ||
			slices.Contains(markingAttributes, key) || strings.HasPrefix(key, "data-") {
			stripped.Attributes++
			continue
		}
		kept = append(kept, attr)
	}
	return kept
}

var markingAttributes = []string{"class", "id", "style"}

func strippedFindings(heading string, stripped Stripped) []Finding {
	if !stripped.Any() {
		return nil
	}
	said := make([]string, 0, 4)
	if stripped.Links > 0 {
		said = append(said, plural(stripped.Links, "link"))
	}
	if len(stripped.Elements) > 0 {
		said = append(said, strings.Join(stripped.Elements, ", "))
	}
	if stripped.Attributes > 0 {
		said = append(said, plural(stripped.Attributes, "script attribute"))
	}
	if stripped.Demoted > 0 {
		said = append(said, plural(stripped.Demoted, "heading demoted to h3"))
	}
	if len(stripped.Sources) > 0 {
		dropped := make([]string, 0, len(stripped.Sources))
		for _, source := range stripped.Sources {
			dropped = append(dropped, source.said())
		}
		said = append(said, "the links off the site it dropped: "+strings.Join(dropped, ", "))
	}
	where := "a section of its own"
	if heading != "" {
		where = "the section " + heading
	}
	return []Finding{{
		Severity: SeverityWarn, Code: CodeMarkupStripped,
		Message: "the writer put markup it may not into " + where + ": " + strings.Join(said, "; "),
		Details: map[string]any{
			"heading": heading, "links": stripped.Links, "elements": stripped.Elements,
			"attributes": stripped.Attributes, "demoted": stripped.Demoted, "sources": stripped.Sources,
		},
	}}
}

func plural(count int, noun string) string {
	if count == 1 {
		return "one " + noun
	}
	return strconv.Itoa(count) + " " + noun + "s"
}
