package content

import (
	"slices"
	"strconv"
	"strings"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
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
}

func (s Stripped) Any() bool {
	return s.Links > 0 || len(s.Elements) > 0 || s.Attributes > 0 || s.Demoted > 0
}

func Sanitize(fragment string) (string, Stripped) {
	doc, err := Parse(fragment)
	if err != nil {
		return "", Stripped{}
	}

	var stripped Stripped
	scrub(doc.root, &stripped)

	rendered, err := doc.Render()
	if err != nil {
		return "", stripped
	}
	return rendered, stripped
}

func scrub(parent *html.Node, stripped *Stripped) {
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
			unwrap(parent, child)
			stripped.Links++
		default:
			if slices.Contains(demotedHeadings, child.Data) {
				child.Data = "h3"
				child.DataAtom = atom.H3
				stripped.Demoted++
			}
			child.Attr = keptAttributes(child.Attr, stripped)
			scrub(child, stripped)
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
		if strings.HasPrefix(key, "on") || strings.HasPrefix(strings.ToLower(strings.TrimSpace(attr.Val)), "javascript:") {
			stripped.Attributes++
			continue
		}
		kept = append(kept, attr)
	}
	return kept
}

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
	where := "a section of its own"
	if heading != "" {
		where = "the section " + heading
	}
	return []Finding{{
		Severity: SeverityWarn, Code: CodeMarkupStripped,
		Message: "the writer put markup it may not into " + where + ": " + strings.Join(said, "; "),
		Details: map[string]any{
			"heading": heading, "links": stripped.Links, "elements": stripped.Elements,
			"attributes": stripped.Attributes, "demoted": stripped.Demoted,
		},
	}}
}

func plural(count int, noun string) string {
	if count == 1 {
		return "one " + noun
	}
	return strconv.Itoa(count) + " " + noun + "s"
}
