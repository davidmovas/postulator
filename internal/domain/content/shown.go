package content

import (
	"slices"
	"strings"
	"unicode"

	"golang.org/x/net/html"

	"github.com/davidmovas/postulator/internal/kernel/errors"
)

const shownWords = 8

var unseenElements = []string{"head", "script", "style", "noscript", "template"}

func DescriptionShown(description, page string) (bool, error) {
	doc, err := Parse(description)
	if err != nil {
		return false, err
	}
	opening := wordsOf(doc.Text())
	if len(opening) == 0 {
		return true, nil
	}
	if len(opening) > shownWords {
		opening = opening[:shownWords]
	}

	root, err := html.Parse(strings.NewReader(page))
	if err != nil {
		return false, errors.Wrap(err, errors.External, "the product page is not readable html")
	}
	seen := " " + strings.Join(wordsOf(visibleText(root)), " ") + " "
	return strings.Contains(seen, " "+strings.Join(opening, " ")+" "), nil
}

func visibleText(node *html.Node) string {
	var builder strings.Builder
	var walk func(*html.Node)
	walk = func(current *html.Node) {
		switch {
		case current.Type == html.TextNode:
			builder.WriteString(current.Data)
			builder.WriteString(" ")
			return
		case current.Type == html.ElementNode && slices.Contains(unseenElements, current.Data):
			return
		}
		for child := current.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}
	walk(node)
	return builder.String()
}

func wordsOf(text string) []string {
	return strings.FieldsFunc(strings.ToLower(text), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
}
