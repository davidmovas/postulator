package graph

import (
	"strconv"
	"strings"
	"time"

	"github.com/davidmovas/postulator/internal/kernel/errors"
)

type Kind string

const (
	KindHub      Kind = "hub"
	KindProduct  Kind = "product"
	KindTopic    Kind = "topic"
	KindCategory Kind = "category"
	KindCustom   Kind = "custom"
)

func (k Kind) Valid() bool {
	switch k {
	case KindHub, KindProduct, KindTopic, KindCategory, KindCustom:
		return true
	default:
		return false
	}
}

type Source string

const (
	SourceImport Source = "import"
	SourceUser   Source = "user"
	SourceAI     Source = "ai"
)

func (s Source) Valid() bool {
	switch s {
	case SourceImport, SourceUser, SourceAI:
		return true
	default:
		return false
	}
}

type AnchorSource string

const (
	AnchorUser AnchorSource = "user"
	AnchorAI   AnchorSource = "ai"
)

func (s AnchorSource) Valid() bool {
	switch s {
	case AnchorUser, AnchorAI:
		return true
	default:
		return false
	}
}

type Anchor struct {
	Text   string
	Source AnchorSource
	Weight float64
}

type Entity struct {
	ID                string
	SiteID            string
	Name              string
	Kind              Kind
	Intent            string
	PrimaryKeyword    string
	SecondaryKeywords []string
	Anchors           []Anchor
	CanonicalPageID   *string
	Score             float64
	Source            Source
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

func invalid(message, field string) *errors.Error {
	return errors.New(errors.Invalid, message).WithDetail("field", field)
}

func NewEntity(e Entity) (Entity, error) {
	e.Name = strings.TrimSpace(e.Name)
	e.Intent = strings.TrimSpace(e.Intent)
	e.PrimaryKeyword = strings.TrimSpace(e.PrimaryKeyword)

	switch {
	case e.ID == "":
		return Entity{}, invalid("entity id must not be empty", "id")
	case e.SiteID == "":
		return Entity{}, invalid("entity site id must not be empty", "siteId")
	case e.Name == "":
		return Entity{}, invalid("entity name must not be empty", "name")
	case !e.Kind.Valid():
		return Entity{}, invalid("entity kind is not recognized", "kind")
	case !e.Source.Valid():
		return Entity{}, invalid("entity source is not recognized", "source")
	case e.Score < 0:
		return Entity{}, invalid("entity score must not be negative", "score")
	case e.CanonicalPageID != nil && *e.CanonicalPageID == "":
		return Entity{}, invalid("canonical page id must not be empty when set", "canonicalPageId")
	}

	e.SecondaryKeywords = CleanKeywords(e.SecondaryKeywords)
	anchors, err := NewAnchors(e.Anchors)
	if err != nil {
		return Entity{}, err
	}
	e.Anchors = anchors
	return e, nil
}

func CleanKeywords(raw []string) []string {
	out := make([]string, 0, len(raw))
	seen := make(map[string]struct{}, len(raw))
	for _, keyword := range raw {
		trimmed := strings.TrimSpace(keyword)
		if trimmed == "" {
			continue
		}
		key := strings.ToLower(trimmed)
		if _, dup := seen[key]; dup {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, trimmed)
	}
	return out
}

func NewAnchors(anchors []Anchor) ([]Anchor, error) {
	out := make([]Anchor, 0, len(anchors))
	seen := make(map[string]struct{}, len(anchors))
	for i, anchor := range anchors {
		anchor.Text = strings.TrimSpace(anchor.Text)
		field := "anchors[" + strconv.Itoa(i) + "]"
		switch {
		case anchor.Text == "":
			return nil, invalid("anchor text must not be empty", field+".text")
		case !anchor.Source.Valid():
			return nil, invalid("anchor source is not recognized", field+".source")
		case anchor.Weight < 0 || anchor.Weight > 1:
			return nil, invalid("anchor weight must be between 0 and 1", field+".weight")
		}
		key := strings.ToLower(anchor.Text)
		if _, dup := seen[key]; dup {
			return nil, invalid("anchor text is repeated", field+".text")
		}
		seen[key] = struct{}{}
		out = append(out, anchor)
	}
	return out, nil
}
