package main

import (
	"cmp"
	"encoding/json"
	"slices"
	"strings"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"

	"github.com/davidmovas/postulator/internal/application/sitemodel/sitemodeltest"
	"github.com/davidmovas/postulator/internal/domain/block"
	"github.com/davidmovas/postulator/internal/domain/content"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/template"
)

const summaryWords = 30

type passage struct {
	html  string
	text  string
	words int
}

func capturedProse() ([]passage, error) {
	type sample struct {
		path     string
		passages []passage
	}
	samples := make([]sample, 0)
	seen := map[string]bool{}
	for _, name := range sitemodeltest.Stands() {
		model, err := sitemodeltest.Stand(name)
		if err != nil {
			return nil, err
		}
		paths := map[string]string{}
		for i := range model.Pages {
			paths[model.Pages[i].ID] = model.Pages[i].Path
		}
		for i := range model.Bodies {
			body := &model.Bodies[i]
			if body.Side != block.Current || seen[block.BodyHash(body.Raw)] {
				continue
			}
			doc, err := content.Parse(body.Raw)
			if err != nil {
				return nil, err
			}
			if len(doc.Words()) < minProseWords {
				continue
			}
			passages, err := passagesOf(body.Raw)
			if err != nil {
				return nil, err
			}
			seen[block.BodyHash(body.Raw)] = true
			samples = append(samples, sample{path: name + paths[body.PageID], passages: passages})
		}
	}
	slices.SortFunc(samples, func(a, b sample) int { return cmp.Compare(a.path, b.path) })
	out := make([]passage, 0)
	for i := range samples {
		out = append(out, samples[i].passages...)
	}
	return out, nil
}

func passagesOf(raw string) ([]passage, error) {
	answered, err := withoutComments(raw)
	if err != nil {
		return nil, err
	}
	nodes, err := html.ParseFragment(strings.NewReader(answered), &html.Node{Type: html.ElementNode, Data: "body", DataAtom: atom.Body})
	if err != nil {
		return nil, err
	}
	out := make([]passage, 0, len(nodes))
	for _, node := range nodes {
		if node.Type != html.ElementNode {
			continue
		}
		var rendered strings.Builder
		if err := html.Render(&rendered, node); err != nil {
			return nil, err
		}
		text := textOf(node)
		if words := len(strings.Fields(text)); words > 0 {
			out = append(out, passage{html: rendered.String(), text: text, words: words})
		}
	}
	return out, nil
}

func textOf(node *html.Node) string {
	if node.Type == html.TextNode {
		return node.Data
	}
	var out strings.Builder
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		out.WriteString(textOf(child) + " ")
	}
	return out.String()
}

func standInBody(prose []passage, words int) string {
	if len(prose) == 0 || words <= 0 {
		return ""
	}
	var out strings.Builder
	for at, held := 0, 0; held < words; at++ {
		next := prose[at%len(prose)]
		out.WriteString(next.html)
		held += next.words
	}
	return out.String()
}

func standInDraft(page pagemap.Page, spec template.TemplateSpec, prose []passage) ([]byte, error) {
	draft := content.ContentDraft{Title: page.Title, H1: page.H1, Sections: []content.DraftSection{}}
	if len(prose) > 0 {
		said := strings.Fields(prose[0].text)
		draft.Summary = strings.Join(said[:min(summaryWords, len(said))], " ")
	}
	for i := range spec.Sections {
		draft.Sections = append(draft.Sections, content.DraftSection{
			Heading: spec.Sections[i].Heading, HTML: standInBody(prose, spec.Sections[i].TargetWords),
		})
	}
	return json.Marshal(draft)
}
