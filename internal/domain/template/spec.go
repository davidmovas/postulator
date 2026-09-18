package template

import (
	"encoding/json"
	"time"

	"github.com/davidmovas/postulator/internal/domain/llm"
)

type Scope string

const (
	ScopeGlobal Scope = "global"
	ScopeSite   Scope = "site"
)

func (s Scope) Valid() bool {
	switch s {
	case ScopeGlobal, ScopeSite:
		return true
	default:
		return false
	}
}

type ImageSource string

const (
	ImagesAI      ImageSource = "ai"
	ImagesWPMedia ImageSource = "wpmedia"
	ImagesLocal   ImageSource = "local"
)

func (s ImageSource) Valid() bool {
	switch s {
	case ImagesAI, ImagesWPMedia, ImagesLocal:
		return true
	default:
		return false
	}
}

type AnchorStrategy string

const (
	AnchorPreferUser AnchorStrategy = "prefer_user"
	AnchorRotate     AnchorStrategy = "rotate"
)

func (s AnchorStrategy) Valid() bool {
	switch s {
	case AnchorPreferUser, AnchorRotate:
		return true
	default:
		return false
	}
}

type OverrideScope string

const (
	OverrideSite OverrideScope = "site"
	OverridePage OverrideScope = "page"
)

func (s OverrideScope) Valid() bool {
	switch s {
	case OverrideSite, OverridePage:
		return true
	default:
		return false
	}
}

type SectionKeywordRules struct {
	Include          []string `json:"include"`
	PrimaryInHeading bool     `json:"primaryInHeading"`
}

type Section struct {
	Heading      string              `json:"heading"`
	Intent       string              `json:"intent"`
	TargetWords  int                 `json:"targetWords"`
	Required     bool                `json:"required"`
	KeywordRules SectionKeywordRules `json:"keywordRules"`
}

type Length struct {
	Min int `json:"min"`
	Max int `json:"max"`
}

type KeywordRules struct {
	PrimaryInTitle          bool    `json:"primaryInTitle"`
	PrimaryInH1             bool    `json:"primaryInH1"`
	PrimaryInFirstParagraph bool    `json:"primaryInFirstParagraph"`
	MaxDensity              float64 `json:"maxDensity"`
}

type LinkRules struct {
	UpDepth                    int     `json:"upDepth"`
	DownLinks                  bool    `json:"downLinks"`
	SiblingMinWeight           float64 `json:"siblingMinWeight"`
	MaxLinks                   int     `json:"maxLinks"`
	MaxPerTarget               int     `json:"maxPerTarget"`
	ParentLinkWithinParagraphs int     `json:"parentLinkWithinParagraphs"`
	ChildrenSection            bool    `json:"childrenSection"`
}

type MetaRules struct {
	TitlePattern   string `json:"titlePattern"`
	DescriptionMax int    `json:"descriptionMax"`
}

type Images struct {
	Featured bool        `json:"featured"`
	Inline   int         `json:"inline"`
	Source   ImageSource `json:"source"`
}

type StepSpec struct {
	Name    string         `json:"name"`
	Enabled bool           `json:"enabled"`
	Params  map[string]any `json:"params,omitempty"`
}

type TemplateSpec struct {
	Sections      []Section                 `json:"sections"`
	Tone          string                    `json:"tone"`
	Length        Length                    `json:"length"`
	KeywordRules  KeywordRules              `json:"keywordRules"`
	LinkRules     LinkRules                 `json:"linkRules"`
	MetaRules     MetaRules                 `json:"metaRules"`
	Images        Images                    `json:"images"`
	ModelProfiles map[llm.Role]llm.ModelRef `json:"modelProfiles"`
	Recipe        []StepSpec                `json:"recipe"`
}

type Template struct {
	ID        string
	Scope     Scope
	SiteID    *string
	Name      string
	PageKind  string
	Version   int
	Spec      TemplateSpec
	CreatedAt time.Time
	UpdatedAt time.Time
}

type Override struct {
	ID         string
	TemplateID string
	Scope      OverrideScope
	TargetID   string
	Patch      json.RawMessage
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

type LinkPolicy struct {
	ID             string
	Scope          Scope
	SiteID         *string
	Name           string
	Rules          LinkRules
	ForbidExternal bool
	ForbidSelf     bool
	AnchorStrategy AnchorStrategy
	CreatedAt      time.Time
	UpdatedAt      time.Time
}
