package imports

import (
	"github.com/davidmovas/postulator/internal/domain/graph"
	"github.com/davidmovas/postulator/internal/domain/importmap"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/kernel/dto"
)

const (
	ActionCreate = "create"
	ActionUpdate = "update"
	ActionSkip   = "skip"
)

const (
	CodeBadPath           = "bad_path"
	CodeNoTarget          = "no_target"
	CodeDuplicatePath     = "duplicate_path"
	CodeIntermediatePath  = "intermediate_path"
	CodeUnknownParent     = "unknown_parent"
	CodeUnknownRelated    = "unknown_related"
	CodeSelfEdge          = "self_edge"
	CodeCycle             = "cycle"
	CodeCannibalization   = "cannibalization"
	CodeUnknownEntityKind = "unknown_entity_kind"
	CodeUnknownPageKind   = "unknown_page_kind"
	CodeUnknownWPType     = "unknown_wp_type"
)

type Options struct {
	PathPrefixStrip  string `json:"pathPrefixStrip,omitempty"`
	KeywordSeparator string `json:"keywordSeparator,omitempty"`
	AnchorSeparator  string `json:"anchorSeparator,omitempty"`
	ListSeparator    string `json:"listSeparator,omitempty"`
}

type Mapping struct {
	ID        string            `json:"id,omitempty"`
	SiteID    string            `json:"siteId,omitempty"`
	Name      string            `json:"name,omitempty"`
	Columns   map[string]string `json:"columns"`
	Options   Options           `json:"options"`
	CreatedAt dto.Time          `json:"createdAt"`
	UpdatedAt dto.Time          `json:"updatedAt"`
}

type Finding struct {
	Row     int    `json:"row"`
	Field   string `json:"field,omitempty"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

type PreviewPage struct {
	Path            string `json:"path"`
	Title           string `json:"title"`
	H1              string `json:"h1,omitempty"`
	MetaTitle       string `json:"metaTitle,omitempty"`
	MetaDescription string `json:"metaDescription,omitempty"`
	WPType          string `json:"wpType"`
	PageKind        string `json:"pageKind,omitempty"`
	Entity          string `json:"entity,omitempty"`
	Action          string `json:"action"`
	Generated       bool   `json:"generated,omitempty"`
}

type PreviewEntity struct {
	Name           string   `json:"name"`
	Kind           string   `json:"kind"`
	PrimaryKeyword string   `json:"primaryKeyword,omitempty"`
	Keywords       []string `json:"keywords"`
	Anchors        []string `json:"anchors"`
	Action         string   `json:"action"`
}

type PreviewEdge struct {
	From   string `json:"from"`
	To     string `json:"to"`
	Kind   string `json:"kind"`
	Action string `json:"action"`
}

type Conflict struct {
	PageID   string `json:"pageId"`
	Path     string `json:"path"`
	Reason   string `json:"reason"`
	EntityID string `json:"entityId,omitempty"`
}

type PreviewReport struct {
	Pages           []PreviewPage   `json:"pages"`
	Entities        []PreviewEntity `json:"entities"`
	Edges           []PreviewEdge   `json:"edges"`
	Warnings        []Finding       `json:"warnings"`
	Errors          []Finding       `json:"errors"`
	Cannibalization []Conflict      `json:"cannibalization"`
	Skipped         int             `json:"skipped"`
}

type Counts struct {
	EntitiesCreated int `json:"entitiesCreated"`
	EntitiesUpdated int `json:"entitiesUpdated"`
	EdgesCreated    int `json:"edgesCreated"`
	PagesCreated    int `json:"pagesCreated"`
	PagesUpdated    int `json:"pagesUpdated"`
	Skipped         int `json:"skipped"`
}

func mappingView(m importmap.Mapping) Mapping {
	columns := make(map[string]string, len(m.Columns))
	for field, column := range m.Columns {
		columns[string(field)] = column
	}
	return Mapping{
		ID:        m.ID,
		SiteID:    m.SiteID,
		Name:      m.Name,
		Columns:   columns,
		Options:   Options(m.Options),
		CreatedAt: dto.NewTime(m.CreatedAt),
		UpdatedAt: dto.NewTime(m.UpdatedAt),
	}
}

func mappingViews(list []importmap.Mapping) []Mapping {
	out := make([]Mapping, 0, len(list))
	for i := range list {
		out = append(out, mappingView(list[i]))
	}
	return out
}

func (m Mapping) domain() importmap.Mapping {
	columns := make(map[importmap.Field]string, len(m.Columns))
	for field, column := range m.Columns {
		columns[importmap.Field(field)] = column
	}
	return importmap.Mapping{
		ID:      m.ID,
		SiteID:  m.SiteID,
		Name:    m.Name,
		Columns: columns,
		Options: importmap.Options(m.Options),
	}
}

func conflictView(evidence pagemap.Evidence) Conflict {
	return Conflict{PageID: evidence.PageID, Path: evidence.Path, Reason: string(evidence.Reason), EntityID: evidence.EntityID}
}

func (r *PreviewReport) settle() {
	if r.Pages == nil {
		r.Pages = []PreviewPage{}
	}
	if r.Entities == nil {
		r.Entities = []PreviewEntity{}
	}
	if r.Edges == nil {
		r.Edges = []PreviewEdge{}
	}
	if r.Warnings == nil {
		r.Warnings = []Finding{}
	}
	if r.Errors == nil {
		r.Errors = []Finding{}
	}
	if r.Cannibalization == nil {
		r.Cannibalization = []Conflict{}
	}
}

func entityView(e graph.Entity, action string) PreviewEntity {
	keywords := e.SecondaryKeywords
	if keywords == nil {
		keywords = []string{}
	}
	return PreviewEntity{
		Name:           e.Name,
		Kind:           string(e.Kind),
		PrimaryKeyword: e.PrimaryKeyword,
		Keywords:       keywords,
		Anchors:        anchorTexts(e.Anchors),
		Action:         action,
	}
}

func pageView(p pagemap.Page, draft *pageDraft, action string) PreviewPage {
	return PreviewPage{
		Path:            p.Path,
		Title:           p.Title,
		H1:              p.H1,
		MetaTitle:       p.MetaTitle,
		MetaDescription: p.MetaDescription,
		WPType:          string(p.WPType),
		PageKind:        draft.pageKind,
		Entity:          draft.entity,
		Action:          action,
		Generated:       draft.generated,
	}
}
