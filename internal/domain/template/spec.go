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
	Include          []string `json:"include" description:"Phrases this section must use at least once"`
	PrimaryInHeading bool     `json:"primaryInHeading" description:"The primary keyword must appear in this section's heading"`
}

type Section struct {
	Heading      string              `json:"heading" description:"The heading; {primaryKeyword}, {entityName}, {siteName} and {pageTitle} are filled in per page"`
	Intent       string              `json:"intent" description:"What the section has to cover, one short sentence to the writer"`
	TargetWords  int                 `json:"targetWords" minimum:"0" description:"About how many words the section should run to"`
	Required     bool                `json:"required" description:"The page is not valid without this section"`
	KeywordRules SectionKeywordRules `json:"keywordRules" description:"What this section has to say about the keywords"`
}

type Length struct {
	Min int `json:"min" minimum:"0" description:"The fewest words the whole page may run to"`
	Max int `json:"max" minimum:"0" description:"The most words the whole page may run to"`
}

type KeywordRules struct {
	PrimaryInTitle          bool    `json:"primaryInTitle" description:"The primary keyword must appear in the title"`
	PrimaryInH1             bool    `json:"primaryInH1" description:"The primary keyword must appear in the first heading"`
	PrimaryInFirstParagraph bool    `json:"primaryInFirstParagraph" description:"The primary keyword must appear in the opening paragraph"`
	MaxDensity              float64 `json:"maxDensity" minimum:"0" maximum:"1" description:"The largest share of the words the primary keyword may take, between 0 and 1"`
}

type LinkRules struct {
	UpDepth                    int     `json:"upDepth" minimum:"0" description:"How many levels up the tree a page links to, 1 for its parent alone"`
	DownLinks                  bool    `json:"downLinks" description:"Link down to the children of the entity"`
	SiblingMinWeight           float64 `json:"siblingMinWeight" minimum:"0" maximum:"1" description:"The weight a related edge needs before a sibling link is placed, between 0 and 1"`
	MaxLinks                   int     `json:"maxLinks" minimum:"0" description:"The most internal links one page may carry"`
	MaxPerTarget               int     `json:"maxPerTarget" minimum:"0" description:"The most links one page may point at a single target"`
	ParentLinkWithinParagraphs int     `json:"parentLinkWithinParagraphs" minimum:"0" description:"The parent link must appear within this many paragraphs of the start"`
	ChildrenSection            bool    `json:"childrenSection" description:"Close the page with a section listing its children"`
}

type MetaRules struct {
	TitlePattern   string `json:"titlePattern" description:"How to build the SEO title, for example {primaryKeyword} | {siteName}; {entityName} and {pageTitle} work too"`
	DescriptionMax int    `json:"descriptionMax" minimum:"0" description:"The most characters the SEO description may run to"`
}

type Images struct {
	Featured bool        `json:"featured" description:"The page carries a featured image"`
	Inline   int         `json:"inline" minimum:"0" description:"How many images to place inside the body; zero for a page without images"`
	Source   ImageSource `json:"source" enum:"ai,wpmedia,local" description:"Where the images come from: drawn by a model, picked from the WordPress library, or read from a folder"`
}

func (i Images) Wanted() int {
	count := max(i.Inline, 0)
	if i.Featured {
		count++
	}
	return count
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
