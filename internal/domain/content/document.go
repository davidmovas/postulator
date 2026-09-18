package content

import (
	"crypto/sha256"
	"encoding/hex"
	"iter"
	"slices"
	"strings"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"

	"github.com/davidmovas/postulator/internal/kernel/errors"
)

type Document struct {
	root *html.Node
}

type Link struct {
	Node   *html.Node
	Href   string
	Anchor string
}

func Parse(fragment string) (*Document, error) {
	body := &html.Node{Type: html.ElementNode, Data: "body", DataAtom: atom.Body}

	nodes, err := html.ParseFragment(strings.NewReader(fragment), body)
	if err != nil {
		return nil, errors.Wrap(err, errors.Invalid, "the body is not parseable html")
	}

	root := &html.Node{Type: html.ElementNode, Data: "body", DataAtom: atom.Body}
	for _, node := range nodes {
		root.AppendChild(node)
	}
	return &Document{root: root}, nil
}

func (d *Document) Root() *html.Node {
	return d.root
}

func (d *Document) nodes() iter.Seq[*html.Node] {
	return func(yield func(*html.Node) bool) {
		var walk func(*html.Node) bool
		walk = func(node *html.Node) bool {
			for child := node.FirstChild; child != nil; {
				next := child.NextSibling
				if !yield(child) {
					return false
				}
				if !walk(child) {
					return false
				}
				child = next
			}
			return true
		}
		walk(d.root)
	}
}

func (d *Document) TextNodes(skip func(*html.Node) bool) iter.Seq[*html.Node] {
	return func(yield func(*html.Node) bool) {
		for node := range d.nodes() {
			if node.Type != html.TextNode {
				continue
			}
			if skip != nil && skip(node) {
				continue
			}
			if !yield(node) {
				return
			}
		}
	}
}

func (d *Document) elements(names ...string) []*html.Node {
	out := make([]*html.Node, 0)
	for node := range d.nodes() {
		if node.Type == html.ElementNode && slices.Contains(names, node.Data) {
			out = append(out, node)
		}
	}
	return out
}

func (d *Document) Paragraphs() []*html.Node {
	return d.elements("p")
}

func (d *Document) Headings() []*html.Node {
	return d.elements("h1", "h2", "h3", "h4", "h5", "h6")
}

func (d *Document) Links() []Link {
	anchors := d.elements("a")

	out := make([]Link, 0, len(anchors))
	for _, node := range anchors {
		out = append(out, Link{Node: node, Href: Attr(node, "href"), Anchor: TextOf(node)})
	}
	return out
}

func (d *Document) HTML() string {
	var builder strings.Builder
	for child := d.root.FirstChild; child != nil; child = child.NextSibling {
		if err := html.Render(&builder, child); err != nil {
			return ""
		}
	}
	return builder.String()
}

func (d *Document) Text() string {
	return TextOf(d.root)
}

func (d *Document) Words() []string {
	return strings.Fields(d.Text())
}

func (d *Document) AppendSentence(paragraphIndex int, sentence string) error {
	trimmed := strings.TrimSpace(sentence)
	if trimmed == "" {
		return errors.New(errors.Invalid, "there is no sentence to append")
	}

	paragraphs := d.Paragraphs()
	if paragraphIndex < 0 || paragraphIndex >= len(paragraphs) {
		return errors.New(errors.Invalid, "the body has no paragraph at that position").
			WithDetail("paragraphIndex", paragraphIndex)
	}

	paragraphs[paragraphIndex].AppendChild(&html.Node{Type: html.TextNode, Data: " " + trimmed})
	return nil
}

func (d *Document) Hash() string {
	sum := sha256.Sum256([]byte(d.normalized()))
	return hex.EncodeToString(sum[:])
}

func (d *Document) normalized() string {
	var builder strings.Builder
	for node := range d.nodes() {
		switch node.Type {
		case html.ElementNode:
			builder.WriteString("<" + node.Data)
			attrs := slices.Clone(node.Attr)
			slices.SortFunc(attrs, func(a, b html.Attribute) int { return strings.Compare(a.Key, b.Key) })
			for _, attr := range attrs {
				builder.WriteString(" " + attr.Key + "=" + attr.Val)
			}
			builder.WriteString(">")
		case html.TextNode:
			if collapsed := strings.Join(strings.Fields(node.Data), " "); collapsed != "" {
				builder.WriteString(collapsed + " ")
			}
		default:
		}
	}
	return strings.TrimSpace(builder.String())
}

var blockElements = []string{
	"address", "article", "aside", "blockquote", "br", "dd", "div", "dl", "dt", "figcaption", "figure",
	"footer", "h1", "h2", "h3", "h4", "h5", "h6", "header", "hr", "li", "main", "nav", "ol", "p", "pre",
	"section", "table", "tbody", "td", "tfoot", "th", "thead", "tr", "ul",
}

func TextOf(node *html.Node) string {
	var builder strings.Builder

	var walk func(*html.Node)
	walk = func(current *html.Node) {
		if current.Type == html.TextNode {
			builder.WriteString(current.Data)
			return
		}
		block := current.Type == html.ElementNode && slices.Contains(blockElements, current.Data)
		if block {
			builder.WriteString(" ")
		}
		for child := current.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
		if block {
			builder.WriteString(" ")
		}
	}
	walk(node)

	return strings.Join(strings.Fields(builder.String()), " ")
}

func Attr(node *html.Node, key string) string {
	for _, attr := range node.Attr {
		if attr.Key == key {
			return attr.Val
		}
	}
	return ""
}
