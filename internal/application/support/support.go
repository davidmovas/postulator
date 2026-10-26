package support

import (
	"cmp"
	"context"
	"encoding/json"
	"os"
	"slices"
	"strings"

	"github.com/davidmovas/postulator/internal/application/observe"
	"github.com/davidmovas/postulator/internal/domain/agent"
	"github.com/davidmovas/postulator/internal/domain/contextpack"
	"github.com/davidmovas/postulator/internal/domain/finding"
	"github.com/davidmovas/postulator/internal/domain/graph"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/pagetype"
	"github.com/davidmovas/postulator/internal/domain/render"
	"github.com/davidmovas/postulator/internal/domain/site"
	"github.com/davidmovas/postulator/internal/kernel/clock"
	"github.com/davidmovas/postulator/internal/kernel/dto"
	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/kernel/paging"
)

const Format = "postulator-support/1"

type siteReader interface {
	Get(ctx context.Context, id string) (site.Site, error)
}

type pageReader interface {
	ListBySite(ctx context.Context, siteID string) ([]pagemap.Page, error)
}

type renderReader interface {
	ListBySite(ctx context.Context, siteID string) ([]render.Record, error)
}

type typeSets interface {
	Set(ctx context.Context, siteID string) (pagetype.Set, error)
}

type entityReader interface {
	ListBySite(ctx context.Context, siteID string) ([]graph.Entity, error)
}

type edgeReader interface {
	ListBySite(ctx context.Context, siteID string) ([]graph.Edge, error)
}

type findingReader interface {
	ListBySite(ctx context.Context, siteID string, status finding.Status) ([]finding.Finding, error)
}

type summaryReader interface {
	Summary(ctx context.Context, siteID string) (observe.Summary, error)
}

type conversationReader interface {
	List(ctx context.Context, q agent.ConversationQuery, page paging.Request) (paging.List[agent.Conversation], error)
}

type callReader interface {
	ByConversation(ctx context.Context, conversationID string) ([]agent.ToolCall, error)
}

type messageReader interface {
	List(ctx context.Context, q agent.MessageQuery, page paging.Request) (paging.List[agent.Message], error)
}

type Deps struct {
	Version        string
	PluginRequired string
	Sites          siteReader
	Pages          pageReader
	Renders        renderReader
	Types          typeSets
	Entities       entityReader
	Edges          edgeReader
	Findings       findingReader
	Observation    summaryReader
	Conversations  conversationReader
	Calls          callReader
	Messages       messageReader
	Clock          clock.Clock
}

type Service struct {
	deps Deps
}

func New(deps Deps) *Service {
	return &Service{deps: deps}
}

type BundleRequest struct {
	SiteID string `json:"siteId"`
}

type WriteRequest struct {
	SiteID string `json:"siteId"`
	Path   string `json:"path"`
}

type WriteResponse struct {
	Path  string `json:"path"`
	Bytes int64  `json:"bytes"`
}

type AppFacts struct {
	Version string `json:"version"`
}

type PluginFacts struct {
	Installed    bool     `json:"installed"`
	Version      string   `json:"version"`
	Required     string   `json:"required"`
	Capabilities []string `json:"capabilities"`
}

type CrumbFacts struct {
	Path   string `json:"path,omitempty"`
	Href   string `json:"href,omitempty"`
	Final  string `json:"final,omitempty"`
	Status int    `json:"status,omitempty"`
}

type RenderFacts struct {
	Root        string         `json:"root"`
	FinalPath   string         `json:"finalPath,omitempty"`
	ThemeH1     string         `json:"themeH1,omitempty"`
	Crumbs      [][]CrumbFacts `json:"crumbs"`
	ExtractedAt dto.Time       `json:"extractedAt"`
}

type PageFacts struct {
	Path         string       `json:"path"`
	WPType       string       `json:"wpType"`
	Status       string       `json:"status"`
	OnSite       bool         `json:"onSite"`
	Type         string       `json:"type"`
	TypeReason   string       `json:"typeReason"`
	Parent       string       `json:"parent,omitempty"`
	ParentReason string       `json:"parentReason"`
	ParentMethod string       `json:"parentMethod,omitempty"`
	Why          string       `json:"why,omitempty"`
	Technical    string       `json:"technical,omitempty"`
	Entity       bool         `json:"entity"`
	Render       *RenderFacts `json:"render,omitempty"`
}

type GraphFacts struct {
	Entities      int `json:"entities"`
	ParentEdges   int `json:"parentEdges"`
	RelatedEdges  int `json:"relatedEdges"`
	ProposedEdges int `json:"proposedEdges"`
	WithoutParent int `json:"withoutParent"`
	WithoutPage   int `json:"withoutPage"`
}

type CallFacts struct {
	Tool        string   `json:"tool"`
	Status      string   `json:"status"`
	DurationMS  int64    `json:"durationMs"`
	ResultBytes int      `json:"resultBytes"`
	At          dto.Time `json:"at"`
}

type Bundle struct {
	Format      string          `json:"format"`
	CreatedAt   dto.Time        `json:"createdAt"`
	App         AppFacts        `json:"app"`
	Plugin      PluginFacts     `json:"plugin"`
	Site        string          `json:"site"`
	Observation observe.Summary `json:"observation"`
	Pages       []PageFacts     `json:"pages"`
	Graph       GraphFacts      `json:"graph"`
	Findings    map[string]int  `json:"findings"`
	Calls       []CallFacts     `json:"calls"`
}

func (s *Service) Write(ctx context.Context, req WriteRequest) (WriteResponse, error) {
	path := strings.TrimSpace(req.Path)
	if path == "" {
		return WriteResponse{}, errors.New(errors.Invalid, "name the file the support bundle is written to").WithDetail("field", "path")
	}
	bundle, err := s.Bundle(ctx, BundleRequest{SiteID: req.SiteID})
	if err != nil {
		return WriteResponse{}, err
	}
	encoded, err := json.MarshalIndent(bundle, "", "  ")
	if err != nil {
		return WriteResponse{}, errors.Wrap(err, errors.Internal, "encode the support bundle")
	}
	if err = os.WriteFile(path, encoded, 0o600); err != nil {
		return WriteResponse{}, errors.New(errors.External, "the support bundle could not be written to "+path).WithInternal(err)
	}
	return WriteResponse{Path: path, Bytes: int64(len(encoded))}, nil
}

func (s *Service) Bundle(ctx context.Context, req BundleRequest) (Bundle, error) {
	owner, err := s.deps.Sites.Get(ctx, strings.TrimSpace(req.SiteID))
	if err != nil {
		return Bundle{}, err
	}
	bundle := Bundle{
		Format: Format, CreatedAt: dto.NewTime(s.deps.Clock.Now()), App: AppFacts{Version: s.deps.Version}, Site: owner.BaseURL,
		Plugin: PluginFacts{
			Installed: owner.Plugin.Installed, Version: owner.Plugin.Version, Required: s.deps.PluginRequired,
			Capabilities: slices.Clone(owner.Plugin.Capabilities),
		},
	}
	if bundle.Observation, err = s.deps.Observation.Summary(ctx, owner.ID); err != nil {
		return Bundle{}, err
	}
	open, err := s.deps.Findings.ListBySite(ctx, owner.ID, finding.Open)
	if err != nil {
		return Bundle{}, err
	}
	bundle.Findings = make(map[string]int)
	for i := range open {
		bundle.Findings[string(open[i].Code)]++
	}
	if bundle.Pages, err = s.pages(ctx, owner.ID, open); err != nil {
		return Bundle{}, err
	}
	if bundle.Graph, err = s.graph(ctx, owner.ID); err != nil {
		return Bundle{}, err
	}
	if bundle.Calls, err = s.calls(ctx, owner.ID); err != nil {
		return Bundle{}, err
	}
	return bundle, nil
}

func (s *Service) pages(ctx context.Context, siteID string, open []finding.Finding) ([]PageFacts, error) {
	pages, err := s.deps.Pages.ListBySite(ctx, siteID)
	if err != nil {
		return nil, err
	}
	records, err := s.deps.Renders.ListBySite(ctx, siteID)
	if err != nil {
		return nil, err
	}
	set, err := s.deps.Types.Set(ctx, siteID)
	if err != nil {
		return nil, err
	}
	rendered := make(map[string]render.Record, len(records))
	for i := range records {
		rendered[records[i].PageID] = records[i]
	}
	byPage := make(map[string][]finding.Finding)
	for i := range open {
		if open[i].PageID != nil {
			byPage[*open[i].PageID] = append(byPage[*open[i].PageID], open[i])
		}
	}
	index := pagemap.NewIndex(pages)
	out := make([]PageFacts, 0, len(pages))
	for i := range pages {
		page := pages[i]
		facts := PageFacts{
			Path: page.Path, WPType: string(page.WPType), Status: string(page.Status), OnSite: page.WPID != nil,
			Type: set.TypeOf(page).Name, TypeReason: string(page.TypeReason), ParentReason: string(page.ParentReason),
			ParentMethod: string(page.ParentMethod), Technical: string(page.TechnicalReason), Entity: page.EntityID != nil,
		}
		record, held := rendered[page.ID]
		if held {
			facts.Render = renderFacts(record)
		}
		if page.ParentPageID != nil {
			if parent, known := index.ByID(*page.ParentPageID); known {
				facts.Parent = parent.Path
			}
		}
		if facts.Parent == "" {
			facts.Why = string(contextpack.WhyNoParent(contextpack.ParentEvidence{Page: page, Findings: byPage[page.ID], Rendered: held}).Reason)
		}
		out = append(out, facts)
	}
	slices.SortFunc(out, func(a, b PageFacts) int { return cmp.Compare(a.Path, b.Path) })
	return out, nil
}

func renderFacts(record render.Record) *RenderFacts {
	facts := &RenderFacts{
		Root: string(record.Root), FinalPath: record.FinalPath, ThemeH1: string(record.ThemeH1),
		Crumbs: make([][]CrumbFacts, 0, len(record.Trails)), ExtractedAt: dto.NewTime(record.ExtractedAt),
	}
	for _, trail := range record.Trails {
		crumbs := make([]CrumbFacts, 0, len(trail.Crumbs))
		for _, crumb := range trail.Crumbs {
			crumbs = append(crumbs, CrumbFacts{Path: crumb.Path, Href: crumb.Href, Final: crumb.Final, Status: crumb.Status})
		}
		facts.Crumbs = append(facts.Crumbs, crumbs)
	}
	return facts
}

func (s *Service) graph(ctx context.Context, siteID string) (GraphFacts, error) {
	entities, err := s.deps.Entities.ListBySite(ctx, siteID)
	if err != nil {
		return GraphFacts{}, err
	}
	edges, err := s.deps.Edges.ListBySite(ctx, siteID)
	if err != nil {
		return GraphFacts{}, err
	}
	facts := GraphFacts{Entities: len(entities)}
	parented := make(map[string]struct{})
	for i := range edges {
		switch {
		case edges[i].Status == graph.StatusProposed:
			facts.ProposedEdges++
		case edges[i].Status != graph.StatusApproved:
		case edges[i].Kind == graph.EdgeParent:
			facts.ParentEdges++
			parented[edges[i].FromEntityID] = struct{}{}
		default:
			facts.RelatedEdges++
		}
	}
	for i := range entities {
		if _, held := parented[entities[i].ID]; !held {
			facts.WithoutParent++
		}
		if entities[i].CanonicalPageID == nil {
			facts.WithoutPage++
		}
	}
	return facts, nil
}

func (s *Service) calls(ctx context.Context, siteID string) ([]CallFacts, error) {
	latest, err := s.deps.Conversations.List(ctx, agent.ConversationQuery{SiteID: siteID, Desc: true}, paging.Request{Limit: 1})
	if err != nil {
		return nil, err
	}
	if len(latest.Items) == 0 {
		return []CallFacts{}, nil
	}
	conversationID := latest.Items[0].ID
	sizes, err := s.resultSizes(ctx, conversationID)
	if err != nil {
		return nil, err
	}
	held, err := s.deps.Calls.ByConversation(ctx, conversationID)
	if err != nil {
		return nil, err
	}
	out := make([]CallFacts, 0, len(held))
	for i := range held {
		out = append(out, CallFacts{
			Tool: held[i].Tool, Status: string(held[i].Status), DurationMS: held[i].DurationMS,
			ResultBytes: sizes[held[i].CallID], At: dto.NewTime(held[i].CreatedAt),
		})
	}
	return out, nil
}

func (s *Service) resultSizes(ctx context.Context, conversationID string) (map[string]int, error) {
	sizes := make(map[string]int)
	request := paging.Request{Limit: paging.MaxLimit}
	for {
		listed, err := s.deps.Messages.List(ctx, agent.MessageQuery{ConversationID: conversationID}, request)
		if err != nil {
			return nil, err
		}
		for i := range listed.Items {
			if listed.Items[i].Role == agent.RoleTool && listed.Items[i].CallID != "" {
				sizes[listed.Items[i].CallID] = len(listed.Items[i].Payload)
			}
		}
		if !listed.HasMore {
			return sizes, nil
		}
		request.After = listed.Next
	}
}
