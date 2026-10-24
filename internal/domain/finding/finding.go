package finding

import (
	"encoding/json"
	"maps"
	"slices"
	"time"

	"github.com/davidmovas/postulator/internal/kernel/errors"
)

type Code string

const (
	ParentUnresolved   Code = "parent_unresolved"
	ParentAmbiguous    Code = "parent_ambiguous"
	ParentCycle        Code = "parent_cycle"
	ParentNameTwin     Code = "parent_name_twin"
	ParentDiffers      Code = "parent_differs"
	PluginOutdated     Code = "plugin_outdated"
	FieldChangedOnSite Code = "field_changed_on_site"
	BodyChangedOnSite  Code = "body_changed_on_site"
	RenderUnreachable  Code = "render_unreachable"
	RenderRedirected   Code = "render_redirected"
	BuilderDetected    Code = "builder_detected"
	TechnicalDetected  Code = "technical_detected"
	PageTitleDiffers   Code = "page_title_differs"
	PageH1Differs      Code = "page_h1_differs"
	URLFormMixed       Code = "url_form_mixed"
	EntityCardProposed Code = "entity_card_proposed"
	RowAmbiguous       Code = "row_ambiguous"
	RowWithoutPage     Code = "row_without_page"
	PageWithoutEntity  Code = "page_without_entity"
)

type Severity string

const (
	Info    Severity = "info"
	Warning Severity = "warning"
	Error   Severity = "error"
)

type Status string

const (
	Open      Status = "open"
	Resolved  Status = "resolved"
	Dismissed Status = "dismissed"
)

func (s Status) Valid() bool {
	switch s {
	case Open, Resolved, Dismissed:
		return true
	default:
		return false
	}
}

type Option string

const (
	Ignore         Option = "ignore"
	Accept         Option = "accept"
	AcceptSite     Option = "accept_site"
	AcceptWorkbook Option = "accept_workbook"
	MapToPage      Option = "map_to_page"
	SetType        Option = "set_type"
	RewriteField   Option = "rewrite_field"
	RelinkPage     Option = "relink_page"
	Rescan         Option = "rescan"
	ProposeEntity  Option = "propose_entity"
)

type OptionKind string

const (
	ModelChange OptionKind = "model"
	Run         OptionKind = "run"
	Proposal    OptionKind = "proposal"
)

func BulkOptions() []Option {
	return []Option{Ignore, AcceptSite, AcceptWorkbook, SetType, ProposeEntity}
}

func (o Option) Bulk() bool {
	return slices.Contains(BulkOptions(), o)
}

func (o Option) Kind() OptionKind {
	switch o {
	case Ignore, Accept, AcceptSite, AcceptWorkbook, MapToPage, SetType:
		return ModelChange
	case RewriteField, RelinkPage, Rescan:
		return Run
	case ProposeEntity:
		return Proposal
	default:
		return ""
	}
}

const (
	SubjectSite             = "site"
	ResolvedByObservation   = "observation"
	ResolvedByImport        = "import"
	ResolvedByPerson        = "person"
	ResolvedByAgent         = "agent"
	ResolutionObservedClear = "observed_clear"
	ResolutionRowMatched    = "row_matched"
	ResolutionRowLeftSheet  = "row_left_sheet"
)

func Settled(status Status, resolvedBy string) bool {
	return status == Dismissed || (status == Resolved && (resolvedBy == ResolvedByPerson || resolvedBy == ResolvedByAgent))
}

type rule struct {
	severity Severity
	options  []Option
}

var rules = map[Code]rule{
	ParentUnresolved:   {Warning, []Option{Ignore}},
	ParentAmbiguous:    {Warning, []Option{Ignore}},
	ParentCycle:        {Warning, []Option{Ignore}},
	ParentNameTwin:     {Info, []Option{Ignore}},
	ParentDiffers:      {Warning, []Option{AcceptSite, AcceptWorkbook, Ignore}},
	PluginOutdated:     {Warning, []Option{Ignore}},
	FieldChangedOnSite: {Warning, []Option{AcceptSite, RewriteField, Ignore}},
	BodyChangedOnSite:  {Warning, []Option{AcceptSite, RelinkPage, Ignore}},
	RenderUnreachable:  {Warning, []Option{Rescan, Ignore}},
	RenderRedirected:   {Info, []Option{Rescan, Ignore}},
	BuilderDetected:    {Info, []Option{Ignore}},
	TechnicalDetected:  {Info, []Option{SetType, Ignore}},
	PageTitleDiffers:   {Warning, []Option{AcceptSite, RewriteField, Ignore}},
	PageH1Differs:      {Warning, []Option{AcceptSite, Ignore}},
	URLFormMixed:       {Info, []Option{Ignore}},
	EntityCardProposed: {Info, []Option{Accept, Ignore}},
	RowAmbiguous:       {Warning, []Option{MapToPage, Ignore}},
	RowWithoutPage:     {Info, []Option{MapToPage, Ignore}},
	PageWithoutEntity:  {Warning, []Option{ProposeEntity, Ignore}},
}

func (c Code) Valid() bool {
	_, known := rules[c]
	return known
}

func Codes() []Code {
	return slices.Sorted(maps.Keys(rules))
}

func Options() []Option {
	return slices.Sorted(slices.Values([]Option{
		Ignore, Accept, AcceptSite, AcceptWorkbook, MapToPage, SetType, RewriteField, RelinkPage, Rescan, ProposeEntity,
	}))
}

func SeverityOf(code Code) Severity {
	return rules[code].severity
}

func OptionsOf(code Code) []Option {
	return slices.Clone(rules[code].options)
}

type Finding struct {
	ID         string
	SiteID     string
	PageID     *string
	EntityID   *string
	Code       Code
	Severity   Severity
	SubjectKey string
	Subject    json.RawMessage
	Options    []Option
	Status     Status
	Resolution string
	ResolvedBy string
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

func invalid(message, field string) *errors.Error {
	return errors.New(errors.Invalid, message).WithDetail("field", field)
}

func New(f Finding) (Finding, error) {
	switch {
	case f.ID == "":
		return Finding{}, invalid("finding id must not be empty", "id")
	case f.SiteID == "":
		return Finding{}, invalid("finding site id must not be empty", "siteId")
	case !f.Code.Valid():
		return Finding{}, invalid("finding code is not recognized", "code")
	case f.SubjectKey == "":
		return Finding{}, invalid("finding subject key must not be empty", "subjectKey")
	case f.Status != "" && !f.Status.Valid():
		return Finding{}, invalid("finding status is not recognized", "status")
	case len(f.Subject) > 0 && !json.Valid(f.Subject):
		return Finding{}, invalid("finding subject is not valid json", "subject")
	}
	allowed := OptionsOf(f.Code)
	for _, option := range f.Options {
		if !slices.Contains(allowed, option) {
			return Finding{}, invalid("finding option does not belong to its code", "options")
		}
	}
	if len(f.Options) == 0 {
		f.Options = allowed
	}
	f.Severity = SeverityOf(f.Code)
	if f.Status == "" {
		f.Status = Open
	}
	if len(f.Subject) == 0 {
		f.Subject = json.RawMessage("{}")
	}
	return f, nil
}

func SubjectPage(pageID string) string {
	return "page:" + pageID
}

func SubjectPageField(pageID, key string) string {
	return "page:" + pageID + ":field:" + key
}

func SubjectCrumb(pageID, path string) string {
	return "page:" + pageID + ":crumb:" + path
}

func SubjectEntityCard(entityID string) string {
	return "entity:" + entityID + ":card"
}
