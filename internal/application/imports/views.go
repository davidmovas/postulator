package imports

import (
	"slices"

	"github.com/davidmovas/postulator/internal/application"
	"github.com/davidmovas/postulator/internal/domain/graph"
	"github.com/davidmovas/postulator/internal/domain/importmap"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/kernel/dto"
)

type Action string

const (
	ActionCreate Action = "create"
	ActionUpdate Action = "update"
	ActionSkip   Action = "skip"
)

type FindingCode string

const (
	CodeBadPath           FindingCode = "bad_path"
	CodeNoTarget          FindingCode = "no_target"
	CodeDuplicatePath     FindingCode = "duplicate_path"
	CodeIntermediatePath  FindingCode = "intermediate_path"
	CodeUnknownParent     FindingCode = "unknown_parent"
	CodeUnknownRelated    FindingCode = "unknown_related"
	CodeSelfEdge          FindingCode = "self_edge"
	CodeCycle             FindingCode = "cycle"
	CodeCannibalization   FindingCode = "cannibalization"
	CodeUnknownEntityKind FindingCode = "unknown_entity_kind"
	CodeUnknownPageKind   FindingCode = "unknown_page_kind"
	CodeUnknownWPType     FindingCode = "unknown_wp_type"
	CodeRootPageSkipped   FindingCode = "root_page_skipped"
	CodeBadVolume         FindingCode = "bad_volume"
	CodeUnknownOwnEntity  FindingCode = "unknown_own_entity"
	CodeTechnicalParent   FindingCode = "technical_parent"
	CodeGroupWithoutPage  FindingCode = "group_without_page"
	CodeAmbiguousParent   FindingCode = "ambiguous_parent"
	CodeAmbiguousEntity   FindingCode = "ambiguous_entity"
	CodeProductNotInStore FindingCode = "product_not_in_store"
	CodeProductRowLeft    FindingCode = "product_row_left"
	CodeWPTypeKept        FindingCode = "wp_type_kept"
	CodeIntermediateLevel FindingCode = "intermediate_level"
	CodeScopeClash        FindingCode = "scope_clash"
)

var blockingFindingCodes = []FindingCode{
	CodeBadPath, CodeUnknownParent, CodeUnknownRelated, CodeSelfEdge, CodeCycle, CodeAmbiguousParent, CodeAmbiguousEntity,
	CodeScopeClash,
}

func (c FindingCode) Blocking() bool {
	return slices.Contains(blockingFindingCodes, c)
}

type Options struct {
	PathPrefixStrip string            `json:"pathPrefixStrip,omitempty" description:"Prefix to strip from every path, such as a domain"`
	AnchorSeparator string            `json:"anchorSeparator,omitempty" description:"Anchor separator in a cell; default comma"`
	ListSeparator   string            `json:"listSeparator,omitempty" description:"Other list separator in a cell; default comma"`
	Sheets          []string          `json:"sheets,omitempty" description:"Sheet names to read; default the first"`
	IndentColumns   []string          `json:"indentColumns,omitempty" description:"Columns whose position nests the path, shallowest first"`
	LevelColumns    []string          `json:"levelColumns,omitempty" description:"Root Entity or Root group columns, outermost first"`
	NoteColumns     []string          `json:"noteColumns,omitempty" description:"Columns kept as notes for the writer"`
	RowType         importmap.RowType `json:"rowType,omitempty" enum:"pages,products,kind" description:"Default pages; a parent of products stays a page, a wp_type cell wins"`
	NoHeader        bool              `json:"noHeader,omitempty" description:"No header row: columns go by letter, every row is data"`
}

type Sheet struct {
	Name     string   `json:"name"`
	Headers  []string `json:"headers"`
	Rows     int      `json:"rows"`
	Detected Mapping  `json:"detected"`
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
	Sheet   string `json:"sheet,omitempty"`
	Field   string `json:"field,omitempty"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

type PreviewPage struct {
	Sheet           string        `json:"sheet,omitempty"`
	Path            string        `json:"path"`
	PlannedPath     string        `json:"plannedPath,omitempty"`
	StoreName       string        `json:"storeName,omitempty"`
	MatchedBy       string        `json:"matchedBy,omitempty"`
	Title           string        `json:"title"`
	H1              string        `json:"h1,omitempty"`
	MetaTitle       string        `json:"metaTitle,omitempty"`
	MetaDescription string        `json:"metaDescription,omitempty"`
	Keywords        []dto.Keyword `json:"keywords"`
	WPType          string        `json:"wpType"`
	PageKind        string        `json:"pageKind,omitempty"`
	Entity          string        `json:"entity,omitempty"`
	Action          string        `json:"action"`
	Generated       bool          `json:"generated,omitempty"`
}

type PreviewEntity struct {
	Sheet    string        `json:"sheet,omitempty"`
	Name     string        `json:"name"`
	Parent   string        `json:"parent,omitempty"`
	Kind     string        `json:"kind"`
	Keywords []dto.Keyword `json:"keywords"`
	Anchors  []string      `json:"anchors"`
	Action   string        `json:"action"`
}

type PreviewColumn struct {
	Sheet  string `json:"sheet,omitempty"`
	Header string `json:"header"`
	Use    string `json:"use"`
	Field  string `json:"field,omitempty"`
}

func columnViews(sheet string, uses []importmap.ColumnUse) []PreviewColumn {
	out := make([]PreviewColumn, 0, len(uses))
	for _, use := range uses {
		out = append(out, PreviewColumn{Sheet: sheet, Header: use.Header, Use: string(use.Use), Field: string(use.Field)})
	}
	return out
}

type PreviewGroup struct {
	Sheet string   `json:"sheet,omitempty"`
	Path  []string `json:"path"`
	Page  string   `json:"page,omitempty"`
	Rows  int      `json:"rows"`
}

type PreviewEdge struct {
	Sheet  string `json:"sheet,omitempty"`
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
	Columns         []PreviewColumn `json:"columns"`
	Pages           []PreviewPage   `json:"pages"`
	Entities        []PreviewEntity `json:"entities"`
	Groups          []PreviewGroup  `json:"groups"`
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

func sheetView(info importmap.SheetInfo, detected importmap.Mapping) Sheet {
	headers := info.Headers
	if headers == nil {
		headers = []string{}
	}
	return Sheet{Name: info.Name, Headers: headers, Rows: info.Rows, Detected: mappingView(detected)}
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
	if r.Columns == nil {
		r.Columns = []PreviewColumn{}
	}
	if r.Pages == nil {
		r.Pages = []PreviewPage{}
	}
	if r.Entities == nil {
		r.Entities = []PreviewEntity{}
	}
	if r.Groups == nil {
		r.Groups = []PreviewGroup{}
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

func entityView(sheet string, e graph.Entity, parent string, action Action) PreviewEntity {
	return PreviewEntity{
		Sheet:    sheet,
		Name:     e.Name,
		Parent:   parent,
		Kind:     string(e.Kind),
		Keywords: application.KeywordViews(e.Keywords),
		Anchors:  anchorTexts(e.Anchors),
		Action:   string(action),
	}
}

func pageView(sheet string, p pagemap.Page, draft *pageDraft, action Action) PreviewPage {
	view := PreviewPage{
		Sheet:           sheet,
		Path:            p.Path,
		PlannedPath:     p.PlannedPath,
		MatchedBy:       string(draft.matchedBy),
		Title:           p.Title,
		H1:              p.H1,
		MetaTitle:       p.MetaTitle,
		MetaDescription: p.MetaDescription,
		Keywords:        application.KeywordViews(p.Keywords),
		WPType:          string(p.WPType),
		PageKind:        draft.pageKind,
		Entity:          draft.entity,
		Action:          string(action),
		Generated:       draft.generated,
	}
	if p.WPType == pagemap.WPProduct && p.WPID != nil {
		view.StoreName = p.Observed.Title
	}
	return view
}
