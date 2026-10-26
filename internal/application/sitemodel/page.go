package sitemodel

import (
	"context"
	"strings"
	"unicode/utf8"

	"github.com/davidmovas/postulator/internal/domain/block"
	"github.com/davidmovas/postulator/internal/domain/contextpack"
	"github.com/davidmovas/postulator/internal/domain/field"
	"github.com/davidmovas/postulator/internal/domain/keyword"
	"github.com/davidmovas/postulator/internal/domain/note"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/render"
)

const fieldValueRunes = 300

type PageRequest struct {
	SiteID string `json:"siteId"`
	PageID string `json:"pageId,omitempty"`
	Path   string `json:"path,omitempty"`
}

type IdentityView struct {
	WPID   *int64         `json:"wpId"`
	WPType pagemap.WPType `json:"wpType"`
	Status pagemap.Status `json:"status"`
	URL    string         `json:"url"`
}

type TypeView struct {
	Name      string                  `json:"name"`
	Preset    string                  `json:"preset"`
	Source    pagemap.TypeSource      `json:"source"`
	Reason    pagemap.TypeReason      `json:"reason"`
	Technical pagemap.TechnicalReason `json:"technical,omitempty"`
	ReadOnly  pagemap.ReadOnlyReason  `json:"readOnly,omitempty"`
}

type ParentView struct {
	ParentID string                 `json:"parentId"`
	Why      *contextpack.ParentWhy `json:"why,omitempty"`
	Reason   pagemap.ParentReason   `json:"reason"`
	Method   pagemap.ParentMethod   `json:"method,omitempty"`
	Trail    []contextpack.Ref      `json:"trail"`
}

type BodySummary struct {
	Kind    string `json:"kind"`
	Hash    string `json:"hash"`
	Blocks  int    `json:"blocks"`
	Ours    int    `json:"ours"`
	Client  int    `json:"client"`
	Dynamic int    `json:"dynamic"`
}

type PageField struct {
	Key         string `json:"key"`
	Current     string `json:"current"`
	Base        string `json:"base"`
	Hash        string `json:"hash"`
	Owner       string `json:"owner"`
	Enabled     bool   `json:"enabled"`
	AIWrites    bool   `json:"aiWrites"`
	Rewrites    bool   `json:"rewrites"`
	Instruction string `json:"instruction,omitempty"`
}

type PageFinding struct {
	ID       string   `json:"id"`
	Code     string   `json:"code"`
	Severity string   `json:"severity"`
	Options  []string `json:"options"`
}

type PageModel struct {
	Page       contextpack.Ref       `json:"page"`
	H1         string                `json:"h1"`
	Identity   IdentityView          `json:"identity"`
	Type       TypeView              `json:"type"`
	Parent     ParentView            `json:"parent"`
	Entity     *CardView             `json:"entity"`
	Children   []contextpack.Ref     `json:"children"`
	Siblings   []contextpack.Ref     `json:"siblings"`
	Products   []contextpack.Product `json:"products"`
	ListedFrom string                `json:"listedFrom"`
	Twin       *contextpack.Ref      `json:"twin"`
	Terms      []pagemap.Term        `json:"terms"`
	Keywords   keyword.List          `json:"keywords"`
	Targets    []contextpack.Target  `json:"targets"`
	Outbound   LinksView             `json:"outbound"`
	Inbound    LinksView             `json:"inbound"`
	Fields     []PageField           `json:"fields"`
	Body       BodySummary           `json:"body"`
	Root       render.RootState      `json:"root"`
	Regions    []render.Region       `json:"regions"`
	Generated  int                   `json:"generatedWords"`
	Notes      []note.Note           `json:"notes"`
	Findings   []PageFinding         `json:"findings"`
	More       map[string]int        `json:"more"`
	Paged      map[string]string     `json:"paged"`
}

var pagedBy = map[string]string{
	"fields":   "pages_fields get",
	"blocks":   "pages_blocks list",
	"findings": "findings list",
	"children": "sites_explore tree",
	"outbound": "sites_explore links",
}

func (s *Service) Page(ctx context.Context, req PageRequest) (PageModel, error) {
	var model PageModel
	err := s.deps.Reader.Read(ctx, func(c context.Context) error {
		page, err := s.named(c, ExploreRequest{SiteID: req.SiteID, PageID: req.PageID, Path: req.Path, View: "page"}, true)
		if err != nil {
			return err
		}
		owner, err := s.deps.Sites.Get(c, req.SiteID)
		if err != nil {
			return err
		}
		facts, _, _, err := s.read(c, page.ID, "", nil)
		if err != nil {
			return err
		}
		model = pageModel(facts, page, pagemap.NewSite(owner.BaseURL))
		if len(model.Parent.Trail) > 0 {
			model.Parent.ParentID = model.Parent.Trail[0].PageID
			return nil
		}
		why, err := s.whyNoParent(c, page)
		model.Parent.Why = &why
		return err
	})
	return model, err
}

func (s *Service) whyNoParent(ctx context.Context, page pagemap.Page) (contextpack.ParentWhy, error) {
	open, err := s.deps.Findings.ListOpenFor(ctx, []string{page.ID})
	if err != nil {
		return contextpack.ParentWhy{}, err
	}
	renders, err := s.deps.Renders.ListForPages(ctx, []string{page.ID})
	if err != nil {
		return contextpack.ParentWhy{}, err
	}
	evidence := contextpack.ParentEvidence{Page: page, Findings: open, Rendered: len(renders) > 0}
	if page.ParentPageID != nil {
		if parent, getErr := s.deps.Pages.Get(ctx, *page.ParentPageID); getErr == nil && parent.Status == pagemap.StatusArchived {
			evidence.Parent = &parent
		}
	}
	return contextpack.WhyNoParent(evidence), nil
}

func pageModel(facts contextpack.Facts, page pagemap.Page, site pagemap.Site) PageModel {
	view := pageView(facts)
	model := PageModel{
		Page: view.Page, H1: view.H1,
		Identity: IdentityView{WPID: page.WPID, WPType: page.WPType, Status: page.Status, URL: site.URL(page.Path)},
		Type: TypeView{
			Name: facts.Page.Type, Preset: string(facts.Page.Preset), Source: facts.TypeSource, Reason: facts.TypeReason,
			Technical: facts.Technical, ReadOnly: facts.ReadOnly,
		},
		Parent: ParentView{Reason: facts.ParentReason, Method: page.ParentMethod, Trail: view.Trail},
		Entity: view.Entity, Children: view.Children, Siblings: view.Siblings, Products: view.Products,
		ListedFrom: view.ListedFrom, Twin: view.Twin, Terms: view.Terms, Keywords: view.Keywords, Targets: view.Targets,
		Outbound: view.Outbound, Inbound: view.Inbound, Fields: make([]PageField, 0, len(facts.Fields)),
		Root: view.Root, Regions: view.Regions, Generated: view.Generated, Notes: view.Notes,
		Findings: make([]PageFinding, 0, len(facts.Findings)), More: view.More, Paged: pagedBy,
	}
	for i := range facts.Fields {
		held := &facts.Fields[i]
		row := PageField{
			Key: string(held.Key), Current: shortValue(held.Current), Base: shortValue(held.Base), Hash: held.Hash,
			Owner: OwnerClient, Enabled: held.Scope.Enabled, AIWrites: held.Scope.AIWrites, Rewrites: held.Scope.Rewrites,
			Instruction: held.Scope.Instruction,
		}
		if held.Ours {
			row.Owner = OwnerOurs
		}
		if spec, _ := field.Lookup(held.Key); spec.Denied {
			row.Owner = OwnerStore
		}
		if held.Key == field.Body || held.Key == field.ProductDescription {
			model.Body = bodySummary(held.Current, facts.Written)
			row.Current, row.Base = "", ""
		}
		model.Fields = append(model.Fields, row)
	}
	for i := range facts.Findings {
		held := &facts.Findings[i]
		options := make([]string, 0, len(held.Options))
		for _, option := range held.Options {
			options = append(options, string(option))
		}
		model.Findings = append(model.Findings, PageFinding{
			ID: held.ID, Code: string(held.Code), Severity: string(held.Severity), Options: options,
		})
	}
	return model
}

func bodySummary(raw string, written []string) BodySummary {
	parsed := block.Parse(raw)
	summary := BodySummary{Kind: string(parsed.Kind), Hash: block.BodyHash(raw), Blocks: len(parsed.Blocks)}
	for _, row := range blockRows(parsed, writtenRows(written)) {
		if row.Owner == OwnerOurs {
			summary.Ours++
		} else {
			summary.Client++
		}
		if row.Dynamic {
			summary.Dynamic++
		}
	}
	return summary
}

func writtenRows(hashes []string) []block.Written {
	out := make([]block.Written, 0, len(hashes))
	for _, hash := range hashes {
		out = append(out, block.Written{BlockHash: hash})
	}
	return out
}

func shortValue(text string) string {
	if utf8.RuneCountInString(text) <= fieldValueRunes {
		return text
	}
	return strings.TrimSpace(string([]rune(text)[:fieldValueRunes])) + "…"
}
