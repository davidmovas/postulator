package observe

import (
	"context"
	"encoding/json"
	"strings"
	"unicode/utf8"

	"github.com/davidmovas/postulator/internal/domain/block"
	"github.com/davidmovas/postulator/internal/domain/finding"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/render"
	"github.com/davidmovas/postulator/internal/kernel/dto"
)

const excerptRunes = 160

type SummaryRequest struct {
	SiteID string `json:"siteId" description:"Site id"`
}

type ModelRequest struct {
	PageID string `json:"pageId" description:"Page id"`
}

type IdentityView struct {
	URL         string   `json:"url"`
	Status      string   `json:"status"`
	WPType      string   `json:"wpType"`
	WPParentID  *int64   `json:"wpParentId"`
	ModifiedGMT dto.Time `json:"modifiedGmt"`
	ObservedAt  dto.Time `json:"observedAt"`
}

type HierarchyView struct {
	ParentID    string `json:"parentId"`
	ParentPath  string `json:"parentPath"`
	ParentTitle string `json:"parentTitle"`
	Reason      string `json:"reason"`
	Method      string `json:"method"`
}

type TermView struct {
	Taxonomy string `json:"taxonomy"`
	Name     string `json:"name"`
	Slug     string `json:"slug"`
}

type RegionView struct {
	Kind   string `json:"kind"`
	Inside bool   `json:"inside"`
	Links  int    `json:"links"`
	Words  int    `json:"words"`
	Signal string `json:"signal"`
}

type CrumbView struct {
	Name        string `json:"name"`
	Path        string `json:"path"`
	Href        string `json:"href"`
	Final       string `json:"final"`
	Status      int    `json:"status"`
	TargetID    string `json:"targetId"`
	TargetTitle string `json:"targetTitle"`
	Self        bool   `json:"self"`
}

type TrailView struct {
	Source string      `json:"source"`
	Crumbs []CrumbView `json:"crumbs"`
}

type HeadingView struct {
	Level     int    `json:"level"`
	Text      string `json:"text"`
	Generated bool   `json:"generated"`
}

type RenderView struct {
	Root           string        `json:"root"`
	WordsContent   int           `json:"wordsContent"`
	WordsGenerated int           `json:"wordsGenerated"`
	Builder        string        `json:"builder"`
	URLForm        string        `json:"urlForm"`
	Robots         string        `json:"robots"`
	NoIndex        bool          `json:"noIndex"`
	FinalPath      string        `json:"finalPath"`
	Regions        []RegionView  `json:"regions"`
	Trails         []TrailView   `json:"trails"`
	Headings       []HeadingView `json:"headings"`
	ExtractedAt    dto.Time      `json:"extractedAt"`
}

type LinkView struct {
	Type     string `json:"type"`
	Region   string `json:"region"`
	Source   string `json:"source"`
	Position *int   `json:"position"`
	Ours     bool   `json:"ours"`
	ToURL    string `json:"toUrl"`
	ToPageID string `json:"toPageId"`
	ToPath   string `json:"toPath"`
	Anchor   string `json:"anchor"`
}

type BlockView struct {
	Index   int    `json:"index"`
	Kind    string `json:"kind"`
	Name    string `json:"name"`
	Excerpt string `json:"excerpt"`
	Dynamic bool   `json:"dynamic"`
	Ours    bool   `json:"ours"`
}

type FindingView struct {
	ID        string         `json:"id"`
	Code      string         `json:"code"`
	Severity  string         `json:"severity"`
	Subject   map[string]any `json:"subject"`
	CreatedAt dto.Time       `json:"createdAt"`
}

type ModelView struct {
	PageID          string        `json:"pageId"`
	Path            string        `json:"path"`
	Title           string        `json:"title"`
	WPType          string        `json:"wpType"`
	WPID            *int64        `json:"wpId"`
	Status          string        `json:"status"`
	Identity        *IdentityView `json:"identity"`
	Hierarchy       HierarchyView `json:"hierarchy"`
	TechnicalReason string        `json:"technicalReason"`
	ReadOnlyReason  string        `json:"readOnlyReason"`
	Terms           []TermView    `json:"terms"`
	Render          *RenderView   `json:"render"`
	Links           []LinkView    `json:"links"`
	BodyKind        string        `json:"bodyKind"`
	Blocks          []BlockView   `json:"blocks"`
	Findings        []FindingView `json:"findings"`
}

func (s *Service) SummaryOf(ctx context.Context, req SummaryRequest) (Summary, error) {
	siteID := strings.TrimSpace(req.SiteID)
	if siteID == "" {
		return Summary{}, invalid("a summary needs a site", "siteId")
	}
	return s.Summary(ctx, siteID)
}

func (s *Service) Model(ctx context.Context, req ModelRequest) (ModelView, error) {
	pageID := strings.TrimSpace(req.PageID)
	if pageID == "" {
		return ModelView{}, invalid("a page model needs a page", "pageId")
	}
	model, err := s.Page(ctx, pageID)
	if err != nil {
		return ModelView{}, err
	}
	pages, err := s.deps.Pages.ListBySite(ctx, model.Page.SiteID)
	if err != nil {
		return ModelView{}, err
	}
	written, err := s.deps.Bodies.ListWritten(ctx, pageID)
	if err != nil {
		return ModelView{}, err
	}
	return modelView(model, pagemap.NewIndex(pages), written), nil
}

func titleOf(page pagemap.Page) string {
	if page.Observed.Title != "" {
		return page.Observed.Title
	}
	return page.Title
}

func modelView(model PageModel, index pagemap.Index, written []block.Written) ModelView {
	page := model.Page
	view := ModelView{
		PageID: page.ID, Path: page.Path, Title: titleOf(page), WPType: string(page.WPType), WPID: page.WPID, Status: string(page.Status),
		Hierarchy:       HierarchyView{Reason: string(page.ParentReason), Method: string(page.ParentMethod)},
		TechnicalReason: string(page.TechnicalReason), ReadOnlyReason: string(page.ReadOnlyReason),
		Terms: make([]TermView, 0, len(model.Terms)), Links: make([]LinkView, 0, len(model.Links)),
		Blocks: make([]BlockView, 0), Findings: make([]FindingView, 0, len(model.Findings)),
	}
	if page.ParentPageID != nil {
		view.Hierarchy.ParentID = *page.ParentPageID
		if parent, found := index.ByID(*page.ParentPageID); found {
			view.Hierarchy.ParentPath, view.Hierarchy.ParentTitle = parent.Path, titleOf(parent)
		}
	}
	if model.Identity.PageID != "" {
		view.Identity = &IdentityView{
			URL: model.Identity.URL, Status: model.Identity.Status, WPType: string(model.Identity.WPType), WPParentID: model.Identity.WPParentID,
			ModifiedGMT: dto.TimeOf(model.Identity.ModifiedGMT), ObservedAt: dto.NewTime(model.Identity.ObservedAt),
		}
	}
	for _, term := range model.Terms {
		view.Terms = append(view.Terms, TermView{Taxonomy: term.Taxonomy, Name: term.Name, Slug: term.Slug})
	}
	if model.Render != nil {
		view.Render = renderView(page, *model.Render, index)
	}
	for _, link := range model.Links {
		item := LinkView{
			Type: string(link.Type), Region: link.Region, Source: string(link.Source), Position: link.Position, Ours: link.Ours,
			ToURL: link.ToURL, ToPath: link.ToURL, Anchor: link.AnchorText,
		}
		if link.ToPageID != nil {
			item.ToPageID = *link.ToPageID
			if target, found := index.ByID(*link.ToPageID); found {
				item.ToPath = target.Path
			}
		}
		view.Links = append(view.Links, item)
	}
	if model.Body != nil {
		view.BodyKind, view.Blocks = blockViews(*model.Body, written)
	}
	for _, record := range model.Findings {
		if record.Status != finding.Open {
			continue
		}
		subject := map[string]any{}
		if len(record.Subject) > 0 {
			if err := json.Unmarshal(record.Subject, &subject); err != nil {
				subject = map[string]any{}
			}
		}
		view.Findings = append(view.Findings, FindingView{
			ID: record.ID, Code: string(record.Code), Severity: string(record.Severity), Subject: subject, CreatedAt: dto.NewTime(record.CreatedAt),
		})
	}
	return view
}

func renderView(page pagemap.Page, record render.Record, index pagemap.Index) *RenderView {
	view := &RenderView{
		Root: string(record.Root), WordsContent: record.WordsContent, WordsGenerated: record.WordsGenerated, Builder: record.Builder,
		URLForm: record.URLForm, Robots: record.Robots, NoIndex: record.NoIndex(), FinalPath: record.FinalPath,
		Regions: make([]RegionView, 0, len(record.Regions)), Trails: make([]TrailView, 0, len(record.Trails)),
		Headings: make([]HeadingView, 0, len(record.Headings)), ExtractedAt: dto.NewTime(record.ExtractedAt),
	}
	for _, region := range record.Regions {
		view.Regions = append(view.Regions, RegionView{Kind: string(region.Kind), Inside: region.Inside, Links: region.Links, Words: region.Words, Signal: region.Signal})
	}
	for _, trail := range record.Trails {
		crumbs := make([]CrumbView, 0, len(trail.Crumbs))
		for _, crumb := range trail.Crumbs {
			item := CrumbView{Name: crumb.Name, Path: crumb.Path, Href: crumb.Href, Final: crumb.Final, Status: crumb.Status, Self: crumb.Path == page.Path}
			for _, path := range []string{crumb.Path, crumb.Final} {
				if path == "" || item.TargetID != "" {
					continue
				}
				if target, found := index.ByPath(path); found {
					item.TargetID, item.TargetTitle = target.ID, titleOf(target)
				}
			}
			crumbs = append(crumbs, item)
		}
		view.Trails = append(view.Trails, TrailView{Source: string(trail.Source), Crumbs: crumbs})
	}
	for _, heading := range record.Headings {
		view.Headings = append(view.Headings, HeadingView{Level: heading.Level, Text: heading.Text, Generated: heading.Generated})
	}
	return view
}

func blockViews(body block.Snapshot, written []block.Written) (string, []BlockView) {
	ours := make(map[string]struct{}, len(written))
	for _, held := range written {
		ours[held.BlockHash] = struct{}{}
	}
	parsed := block.Parse(body.Raw)
	views := make([]BlockView, 0, len(parsed.Blocks))
	for _, b := range parsed.Blocks {
		_, wrote := ours[b.Hash]
		views = append(views, BlockView{
			Index: b.Index, Kind: string(b.Kind), Name: b.Name, Excerpt: excerpt(b.Text()), Dynamic: b.Dynamic, Ours: wrote,
		})
	}
	return string(parsed.Kind), views
}

func excerpt(text string) string {
	text = strings.Join(strings.Fields(text), " ")
	if utf8.RuneCountInString(text) <= excerptRunes {
		return text
	}
	runes := []rune(text)
	return strings.TrimSpace(string(runes[:excerptRunes])) + "…"
}
