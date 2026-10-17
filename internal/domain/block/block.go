package block

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"

	"github.com/davidmovas/postulator/internal/domain/content"
)

type Kind string

const (
	Classic Kind = "classic"
	Blocks  Kind = "blocks"
)

type BlockKind string

const (
	Paragraph BlockKind = "paragraph"
	Heading   BlockKind = "heading"
	List      BlockKind = "list"
	Quote     BlockKind = "quote"
	HTML      BlockKind = "html"
	Shortcode BlockKind = "shortcode"
	WP        BlockKind = "wp"
	Foreign   BlockKind = "foreign"
)

type Block struct {
	Index      int
	Kind       BlockKind
	Name       string
	Raw        string
	Hash       string
	Attributes json.RawMessage
	Dynamic    bool
}

type Body struct {
	Kind   Kind
	Blocks []Block
}

func Hash(raw string) string {
	folded := strings.ReplaceAll(raw, "\r\n", "\n")
	sum := sha256.Sum256([]byte(strings.TrimSpace(folded)))
	return hex.EncodeToString(sum[:])
}

func (b Block) Text() string {
	if b.Dynamic || b.Kind == Shortcode {
		return ""
	}
	return textOf(b.Raw)
}

func textOf(raw string) string {
	doc, err := content.Parse(raw)
	if err != nil {
		return ""
	}
	return doc.Text()
}

func (b Body) Serialize() string {
	var builder strings.Builder
	for i := range b.Blocks {
		builder.WriteString(b.Blocks[i].Raw)
	}
	return builder.String()
}

func (b Body) Hash() string {
	return Hash(b.Serialize())
}

func (b Body) TextBlocks() []string {
	out := make([]string, 0, len(b.Blocks))
	for i := range b.Blocks {
		if text := b.Blocks[i].Text(); text != "" {
			out = append(out, text)
		}
	}
	return out
}

func (b Body) Text() string {
	return strings.Join(b.TextBlocks(), " ")
}

func (b Body) Empty() bool {
	for i := range b.Blocks {
		if b.Blocks[i].Dynamic || b.Blocks[i].Text() != "" {
			return false
		}
	}
	return true
}

func (b Body) DynamicOnly() bool {
	dynamic := false
	for i := range b.Blocks {
		switch {
		case b.Blocks[i].Dynamic:
			dynamic = true
		case b.Blocks[i].Text() != "":
			return false
		}
	}
	return dynamic
}

func (b Body) Shortcodes() []string {
	out := make([]string, 0)
	for i := range b.Blocks {
		switch {
		case b.Blocks[i].Kind == Shortcode && b.Kind == Classic:
			out = append(out, b.Blocks[i].Name)
		case b.Blocks[i].Kind == Shortcode:
			out = append(out, shortcodeTags(b.Blocks[i].Raw)...)
		}
	}
	return out
}

func (b Body) BlockNames() []string {
	out := make([]string, 0)
	if b.Kind != Blocks {
		return out
	}
	for i := range b.Blocks {
		if strings.Contains(b.Blocks[i].Name, "/") {
			out = append(out, b.Blocks[i].Name)
		}
	}
	return out
}
