package graph

import (
	"context"
	"embed"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/davidmovas/postulator/internal/application/llm"
	graphdomain "github.com/davidmovas/postulator/internal/domain/graph"
	domainllm "github.com/davidmovas/postulator/internal/domain/llm"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/kernel/id"
)

const (
	NameProposeFromPages = "propose_from_pages"
	NameProposeRelated   = "propose_related"

	PagesPerCall     = 40
	proposalTokens   = 4096
	relatedWeightMin = 0.05
)

//go:embed prompts/*.tmpl
var promptFS embed.FS

var prompts = llm.MustPrompts(promptFS, "prompts/*.tmpl")

type entityProposal struct {
	Path              string   `json:"path" description:"The path of the page this entity is about, copied exactly"`
	Name              string   `json:"name" description:"The topic of the page, two to four words in title case"`
	Kind              string   `json:"kind" enum:"hub,category,topic,product" description:"What the page is"`
	Intent            string   `json:"intent" description:"What the reader wants from the page, one short sentence"`
	PrimaryKeyword    string   `json:"primaryKeyword" description:"The phrase a reader searches to reach the page"`
	SecondaryKeywords []string `json:"secondaryKeywords" description:"At most four further phrases the page can rank for"`
	Anchors           []string `json:"anchors" description:"At most four link texts another page could point here with"`
	ParentPath        string   `json:"parentPath" description:"The path one level up in the same subject, empty at a root"`
	RelatedPaths      []string `json:"relatedPaths" description:"At most three sibling paths of the same subject"`
}

type pagesProposal struct {
	Entities []entityProposal `json:"entities" description:"One entity per page you can place"`
}

type relatedProposal struct {
	From   string  `json:"from" description:"The name of the first entity, spelled as it was given"`
	To     string  `json:"to" description:"The name of the second entity, spelled as it was given"`
	Weight float64 `json:"weight" description:"How close the two are, between 0 and 1"`
	Reason string  `json:"reason" description:"What the two share, one short sentence"`
}

type relatedProposals struct {
	Edges []relatedProposal `json:"edges" description:"The pairs worth linking, strongest first"`
}

type knownEntity struct {
	Name string
	Kind string
	Path string
}

type pagePromptLine struct {
	Path            string
	Title           string
	H1              string
	MetaDescription string
	PrimaryKeyword  string
	Keywords        string
}

type pagesPrompt struct {
	SiteName string
	Known    []knownEntity
	Pages    []pagePromptLine
}

type relatedPair struct {
	From string
	To   string
}

type relatedPrompt struct {
	SiteName string
	Focus    string
	Entities []graphdomain.Entity
	Existing []relatedPair
}

func (s *Service) ProposeFromPages(ctx context.Context, req ProposeFromPagesRequest) (ProposeFromPagesResponse, error) {
	proposed, err := s.previewFromPages(ctx, req.SiteID, req.PageIDs, req.PathPrefix)
	if err != nil {
		return ProposeFromPagesResponse{}, err
	}
	if len(proposed.Entities) == 0 {
		return ProposeFromPagesResponse{Entities: []Entity{}, Edges: []Edge{}, Skipped: proposed.Skipped, Tokens: proposed.Tokens}, nil
	}

	applied, err := s.ApplyProposals(ctx, ApplyProposalsRequest{SiteID: req.SiteID, Entities: proposed.Entities})
	if err != nil {
		return ProposeFromPagesResponse{}, err
	}
	return ProposeFromPagesResponse{
		Entities: applied.Entities, Edges: applied.Edges,
		Skipped: proposed.Skipped + applied.Skipped, Tokens: proposed.Tokens,
	}, nil
}

func (s *Service) PreviewFromPages(ctx context.Context, req PreviewFromPagesRequest) (PreviewFromPagesResponse, error) {
	return s.previewFromPages(ctx, req.SiteID, req.PageIDs, req.PathPrefix)
}

func (s *Service) previewFromPages(ctx context.Context, rawSiteID string, pageIDs []string, pathPrefix string) (PreviewFromPagesResponse, error) {
	siteID := strings.TrimSpace(rawSiteID)
	if err := requireSite(siteID); err != nil {
		return PreviewFromPagesResponse{}, err
	}

	owner, err := s.sites.Get(ctx, siteID)
	if err != nil {
		return PreviewFromPagesResponse{}, err
	}
	ref, err := s.model(ctx, siteID)
	if err != nil {
		return PreviewFromPagesResponse{}, err
	}
	state, err := s.snapshot(ctx, siteID)
	if err != nil {
		return PreviewFromPagesResponse{}, err
	}

	chosen, err := chosenPages(state.pages, pageIDs, pathPrefix)
	if err != nil {
		return PreviewFromPagesResponse{}, err
	}

	response := PreviewFromPagesResponse{Entities: []ProposedEntity{}, Pages: len(chosen)}
	for batch := range slices.Chunk(chosen, PagesPerCall) {
		system, user, renderErr := prompts.Render(NameProposeFromPages, pagesPromptOf(owner.Name, state, batch))
		if renderErr != nil {
			return PreviewFromPagesResponse{}, renderErr
		}

		proposal, usage, callErr := llm.Structured[pagesProposal](ctx, s.llm, llm.Request{
			Ref:       ref,
			System:    system,
			Messages:  []llm.Message{{Role: llm.RoleUser, Text: user}},
			MaxTokens: proposalTokens,
			Meta:      llm.CallMeta{Step: NameProposeFromPages},
		})
		if callErr != nil {
			return PreviewFromPagesResponse{}, callErr
		}
		response.Tokens += usage.Total

		inBatch := make(map[string]pagemap.Page, len(batch))
		for i := range batch {
			inBatch[batch[i].Path] = batch[i]
		}
		for i := range proposal.Entities {
			proposed := &proposal.Entities[i]
			page, known := inBatch[strings.TrimSpace(proposed.Path)]
			name := strings.TrimSpace(proposed.Name)
			if !known || name == "" {
				response.Skipped++
				continue
			}
			response.Entities = append(response.Entities, ProposedEntity{
				PageID:            page.ID,
				Path:              page.Path,
				Name:              name,
				Kind:              string(kindOf(proposed.Kind)),
				Intent:            strings.TrimSpace(proposed.Intent),
				PrimaryKeyword:    strings.TrimSpace(proposed.PrimaryKeyword),
				SecondaryKeywords: graphdomain.CleanKeywords(proposed.SecondaryKeywords),
				Anchors:           graphdomain.CleanKeywords(proposed.Anchors),
				Parent:            strings.TrimSpace(proposed.ParentPath),
				Related:           graphdomain.CleanKeywords(proposed.RelatedPaths),
				ExistingEntityID:  state.byName[fold(name)],
			})
		}
	}
	return response, nil
}

func chosenPages(pages []pagemap.Page, pageIDs []string, pathPrefix string) ([]pagemap.Page, error) {
	unmapped := unmappedPages(pages)
	prefix := strings.TrimSpace(pathPrefix)

	if len(pageIDs) == 0 {
		if prefix == "" {
			return unmapped, nil
		}
		out := make([]pagemap.Page, 0, len(unmapped))
		for i := range unmapped {
			if strings.HasPrefix(unmapped[i].Path, prefix) {
				out = append(out, unmapped[i])
			}
		}
		return out, nil
	}

	byID := make(map[string]pagemap.Page, len(pages))
	for i := range pages {
		byID[pages[i].ID] = pages[i]
	}
	out := make([]pagemap.Page, 0, len(pageIDs))
	mapped := make([]string, 0)
	seen := make(map[string]struct{}, len(pageIDs))
	for _, raw := range pageIDs {
		pageID := strings.TrimSpace(raw)
		if _, twice := seen[pageID]; twice || pageID == "" {
			continue
		}
		seen[pageID] = struct{}{}
		page, known := byID[pageID]
		if !known {
			return nil, errors.New(errors.NotFound, "page not found").WithDetail("pageId", pageID)
		}
		if page.EntityID != nil {
			mapped = append(mapped, page.Path)
			continue
		}
		if page.Path == pagemap.RootPath || page.Status == pagemap.StatusArchived {
			continue
		}
		out = append(out, page)
	}
	if len(mapped) > 0 {
		return nil, errors.New(errors.Invalid, "an entity is proposed only for a page that has none, and "+
			strings.Join(mapped, ", ")+" already carry one; unmap them first or leave them out").
			WithDetail("field", "pageIds").WithDetail("paths", mapped)
	}
	return out, nil
}

func (s *Service) ProposeRelated(ctx context.Context, req ProposeRelatedRequest) (ProposeRelatedResponse, error) {
	siteID := strings.TrimSpace(req.SiteID)
	if err := requireSite(siteID); err != nil {
		return ProposeRelatedResponse{}, err
	}

	owner, err := s.sites.Get(ctx, siteID)
	if err != nil {
		return ProposeRelatedResponse{}, err
	}
	ref, err := s.model(ctx, siteID)
	if err != nil {
		return ProposeRelatedResponse{}, err
	}

	state, err := s.snapshot(ctx, siteID)
	if err != nil {
		return ProposeRelatedResponse{}, err
	}
	if len(state.entities) < 2 {
		return ProposeRelatedResponse{Edges: []Edge{}}, nil
	}

	focus := ""
	if entityID := strings.TrimSpace(req.EntityID); entityID != "" {
		for i := range state.entities {
			if state.entities[i].ID == entityID {
				focus = state.entities[i].Name
			}
		}
		if focus == "" {
			return ProposeRelatedResponse{}, errors.New(errors.NotFound, "entity not found").
				WithDetail("entityId", entityID)
		}
	}

	system, user, err := prompts.Render(NameProposeRelated, relatedPrompt{
		SiteName: owner.Name, Focus: focus, Entities: state.entities, Existing: pairsOf(state),
	})
	if err != nil {
		return ProposeRelatedResponse{}, err
	}

	proposal, usage, err := llm.Structured[relatedProposals](ctx, s.llm, llm.Request{
		Ref:       ref,
		System:    system,
		Messages:  []llm.Message{{Role: llm.RoleUser, Text: user}},
		MaxTokens: proposalTokens,
		Meta:      llm.CallMeta{Step: NameProposeRelated},
	})
	if err != nil {
		return ProposeRelatedResponse{}, err
	}

	response := ProposeRelatedResponse{Edges: []Edge{}, Tokens: usage.Total}
	if applyErr := s.applyRelated(ctx, siteID, proposal, &state, &response); applyErr != nil {
		return ProposeRelatedResponse{}, applyErr
	}
	if len(response.Edges) == 0 {
		return response, nil
	}
	if changedErr := s.changed(siteID); changedErr != nil {
		return ProposeRelatedResponse{}, changedErr
	}
	return response, nil
}

func (s *Service) model(ctx context.Context, siteID string) (domainllm.ModelRef, error) {
	if s.profiles == nil || s.llm == nil {
		return domainllm.ModelRef{}, errors.New(errors.Invalid, "no model is wired for the graph proposals")
	}
	return s.profiles.Resolve(ctx, siteID, domainllm.RoleEditor, nil)
}

type siteGraph struct {
	entities []graphdomain.Entity
	edges    []graphdomain.Edge
	pages    []pagemap.Page
	byName   map[string]string
	byPath   map[string]string
}

func (s *Service) snapshot(ctx context.Context, siteID string) (siteGraph, error) {
	entities, err := s.entities.ListBySite(ctx, siteID)
	if err != nil {
		return siteGraph{}, err
	}
	edges, err := s.edges.ListBySite(ctx, siteID)
	if err != nil {
		return siteGraph{}, err
	}
	pages, err := s.pages.ListBySite(ctx, siteID)
	if err != nil {
		return siteGraph{}, err
	}

	state := siteGraph{
		entities: entities, edges: edges, pages: pages,
		byName: make(map[string]string, len(entities)), byPath: make(map[string]string, len(pages)),
	}
	for i := range entities {
		state.byName[fold(entities[i].Name)] = entities[i].ID
	}
	for i := range pages {
		if pages[i].EntityID != nil {
			state.byPath[pages[i].Path] = *pages[i].EntityID
		}
	}
	return state, nil
}

func unmappedPages(pages []pagemap.Page) []pagemap.Page {
	out := make([]pagemap.Page, 0, len(pages))
	for i := range pages {
		if pages[i].EntityID != nil || pages[i].Status == pagemap.StatusArchived || pages[i].Path == pagemap.RootPath {
			continue
		}
		out = append(out, pages[i])
	}
	return out
}

func pagesPromptOf(siteName string, state siteGraph, batch []pagemap.Page) pagesPrompt {
	known := make([]knownEntity, 0, len(state.entities))
	for i := range state.entities {
		entry := knownEntity{Name: state.entities[i].Name, Kind: string(state.entities[i].Kind)}
		for path, entityID := range state.byPath {
			if entityID == state.entities[i].ID {
				entry.Path = path
			}
		}
		known = append(known, entry)
	}
	slices.SortFunc(known, func(a, b knownEntity) int { return strings.Compare(a.Name, b.Name) })

	lines := make([]pagePromptLine, 0, len(batch))
	for i := range batch {
		lines = append(lines, pagePromptLine{
			Path: batch[i].Path, Title: batch[i].Title, H1: batch[i].H1, MetaDescription: batch[i].MetaDescription,
			PrimaryKeyword: batch[i].PrimaryKeyword, Keywords: strings.Join(batch[i].Keywords, ", "),
		})
	}
	return pagesPrompt{SiteName: siteName, Known: known, Pages: lines}
}

func (s *Service) applyRelated(ctx context.Context, siteID string, proposal relatedProposals, state *siteGraph,
	out *ProposeRelatedResponse) error {
	now := s.now()
	return s.uow.Do(ctx, func(c context.Context) error {
		for _, proposed := range proposal.Edges {
			from, knownFrom := state.byName[fold(proposed.From)]
			to, knownTo := state.byName[fold(proposed.To)]
			if !knownFrom || !knownTo || from == to {
				out.Skipped++
				continue
			}

			edge, err := graphdomain.NewEdge(graphdomain.Edge{
				ID: id.New(), SiteID: siteID, FromEntityID: from, ToEntityID: to,
				Kind: graphdomain.EdgeRelated, Weight: weightOf(proposed.Weight),
				Source: graphdomain.SourceAI, Status: graphdomain.StatusProposed, Reason: clip(proposed.Reason),
				CreatedAt: now,
			})
			if err != nil || connected(state.edges, edge) {
				out.Skipped++
				continue
			}
			if insertErr := s.edges.Insert(c, edge); insertErr != nil {
				if errors.IsCode(insertErr, errors.Conflict) {
					out.Skipped++
					continue
				}
				return insertErr
			}

			state.edges = append(state.edges, edge)
			out.Edges = append(out.Edges, edgeView(edge))
		}
		return nil
	})
}

func connected(edges []graphdomain.Edge, candidate graphdomain.Edge) bool {
	for i := range edges {
		if edges[i].FromEntityID == candidate.FromEntityID && edges[i].ToEntityID == candidate.ToEntityID {
			return true
		}
		if edges[i].FromEntityID == candidate.ToEntityID && edges[i].ToEntityID == candidate.FromEntityID {
			return true
		}
	}
	return false
}

func pairsOf(state siteGraph) []relatedPair {
	names := make(map[string]string, len(state.entities))
	for i := range state.entities {
		names[state.entities[i].ID] = state.entities[i].Name
	}

	pairs := make([]relatedPair, 0, len(state.edges))
	for i := range state.edges {
		pairs = append(pairs, relatedPair{
			From: names[state.edges[i].FromEntityID], To: names[state.edges[i].ToEntityID],
		})
	}
	return pairs
}

func kindOf(raw string) graphdomain.Kind {
	kind := graphdomain.Kind(strings.ToLower(strings.TrimSpace(raw)))
	if !kind.Valid() {
		return graphdomain.KindTopic
	}
	return kind
}

func proposedAnchors(texts []string) []graphdomain.Anchor {
	out := make([]graphdomain.Anchor, 0, len(texts))
	for _, text := range texts {
		trimmed := strings.TrimSpace(text)
		if trimmed == "" {
			continue
		}
		out = append(out, graphdomain.Anchor{Text: trimmed, Source: graphdomain.AnchorAI, Weight: 1})
	}
	return out
}

func weightOf(raw float64) float64 {
	return min(max(raw, relatedWeightMin), 1)
}

func clip(reason string) string {
	trimmed := strings.TrimSpace(reason)
	if utf8.RuneCountInString(trimmed) <= graphdomain.EdgeReasonMax {
		return trimmed
	}
	runes := []rune(trimmed)
	return strings.TrimSpace(string(runes[:graphdomain.EdgeReasonMax]))
}

func parentReason(path, parentPath string) string {
	return path + " sits one level under " + parentPath
}

func relatedReason(path, relatedPath string) string {
	return path + " and " + relatedPath + " are sibling pages of one subject"
}

func fold(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}
