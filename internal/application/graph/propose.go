package graph

import (
	"context"
	"embed"
	"slices"
	"strings"
	"time"
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

type pagesPrompt struct {
	SiteName string
	Known    []knownEntity
	Pages    []pagemap.Page
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
	siteID := strings.TrimSpace(req.SiteID)
	if err := requireSite(siteID); err != nil {
		return ProposeFromPagesResponse{}, err
	}

	owner, err := s.sites.Get(ctx, siteID)
	if err != nil {
		return ProposeFromPagesResponse{}, err
	}
	ref, err := s.model(ctx, siteID)
	if err != nil {
		return ProposeFromPagesResponse{}, err
	}

	state, err := s.snapshot(ctx, siteID)
	if err != nil {
		return ProposeFromPagesResponse{}, err
	}

	unmapped := unmappedPages(state.pages)
	if len(unmapped) == 0 {
		return ProposeFromPagesResponse{Entities: []Entity{}, Edges: []Edge{}}, nil
	}

	response := ProposeFromPagesResponse{Entities: []Entity{}, Edges: []Edge{}}
	for batch := range slices.Chunk(unmapped, PagesPerCall) {
		system, user, renderErr := prompts.Render(NameProposeFromPages, pagesPromptOf(owner.Name, state, batch))
		if renderErr != nil {
			return ProposeFromPagesResponse{}, renderErr
		}

		proposal, usage, callErr := llm.Structured[pagesProposal](ctx, s.llm, llm.Request{
			Ref:       ref,
			System:    system,
			Messages:  []llm.Message{{Role: llm.RoleUser, Text: user}},
			MaxTokens: proposalTokens,
			Meta:      llm.CallMeta{Step: NameProposeFromPages},
		})
		if callErr != nil {
			return ProposeFromPagesResponse{}, callErr
		}
		response.Tokens += usage.Total

		if applyErr := s.applyPages(ctx, siteID, batch, proposal, &state, &response); applyErr != nil {
			return ProposeFromPagesResponse{}, applyErr
		}
	}

	if len(response.Entities) == 0 && len(response.Edges) == 0 {
		return response, nil
	}
	if changedErr := s.changed(siteID); changedErr != nil {
		return ProposeFromPagesResponse{}, changedErr
	}
	if pagesErr := s.pagesChanged(siteID); pagesErr != nil {
		return ProposeFromPagesResponse{}, pagesErr
	}
	return response, nil
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

	return pagesPrompt{SiteName: siteName, Known: known, Pages: batch}
}

func (s *Service) applyPages(ctx context.Context, siteID string, batch []pagemap.Page, proposal pagesProposal,
	state *siteGraph, out *ProposeFromPagesResponse) error {
	inBatch := make(map[string]pagemap.Page, len(batch))
	for i := range batch {
		inBatch[batch[i].Path] = batch[i]
	}

	now := s.now()
	return s.uow.Do(ctx, func(c context.Context) error {
		for i := range proposal.Entities {
			proposed := &proposal.Entities[i]
			page, known := inBatch[strings.TrimSpace(proposed.Path)]
			if !known {
				out.Skipped++
				continue
			}
			if _, taken := state.byPath[page.Path]; taken {
				out.Skipped++
				continue
			}

			entityID, err := s.adopt(c, siteID, page, proposed, state, out, now)
			if err != nil {
				return err
			}
			if entityID == "" {
				out.Skipped++
				continue
			}
			state.byPath[page.Path] = entityID
		}
		return s.connect(c, siteID, proposal, state, out, now)
	})
}

func (s *Service) adopt(ctx context.Context, siteID string, page pagemap.Page, proposed *entityProposal,
	state *siteGraph, out *ProposeFromPagesResponse, now time.Time) (string, error) {
	name := strings.TrimSpace(proposed.Name)
	if name == "" {
		return "", nil
	}

	entityID, exists := state.byName[fold(name)]
	if !exists {
		entity, err := graphdomain.NewEntity(graphdomain.Entity{
			ID:                id.New(),
			SiteID:            siteID,
			Name:              name,
			Kind:              kindOf(proposed.Kind),
			Intent:            proposed.Intent,
			PrimaryKeyword:    proposed.PrimaryKeyword,
			SecondaryKeywords: proposed.SecondaryKeywords,
			Anchors:           proposedAnchors(proposed.Anchors),
			Source:            graphdomain.SourceAI,
			CreatedAt:         now,
			UpdatedAt:         now,
		})
		if err != nil {
			return "", nil
		}
		if insertErr := s.entities.Insert(ctx, entity); insertErr != nil {
			return "", insertErr
		}
		state.entities = append(state.entities, entity)
		state.byName[fold(name)] = entity.ID
		entityID = entity.ID
		out.Entities = append(out.Entities, entityView(entity))
	}

	next := page
	next.EntityID = &entityID
	next.UpdatedAt = now
	if err := s.pages.Update(ctx, next); err != nil {
		return "", err
	}
	if !exists {
		if err := s.entities.SetCanonicalPage(ctx, entityID, &page.ID, now); err != nil {
			return "", err
		}
	}
	return entityID, nil
}

func (s *Service) connect(ctx context.Context, siteID string, proposal pagesProposal, state *siteGraph,
	out *ProposeFromPagesResponse, now time.Time) error {
	for i := range proposal.Entities {
		proposed := &proposal.Entities[i]
		from, known := state.byPath[strings.TrimSpace(proposed.Path)]
		if !known {
			continue
		}

		path := strings.TrimSpace(proposed.Path)
		if parentPath := strings.TrimSpace(proposed.ParentPath); parentPath != "" {
			if parent, ok := state.byPath[parentPath]; ok {
				reason := parentReason(path, parentPath)
				if err := s.propose(ctx, siteID, from, parent, graphdomain.EdgeParent, 1, reason, state, out, now); err != nil {
					return err
				}
			}
		}
		for _, raw := range proposed.RelatedPaths {
			relatedPath := strings.TrimSpace(raw)
			related, ok := state.byPath[relatedPath]
			if !ok {
				continue
			}
			reason := relatedReason(path, relatedPath)
			if err := s.propose(ctx, siteID, from, related, graphdomain.EdgeRelated, 0.5, reason, state, out, now); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *Service) propose(ctx context.Context, siteID, from, to string, kind graphdomain.EdgeKind,
	weight float64, reason string, state *siteGraph, out *ProposeFromPagesResponse, now time.Time) error {
	edge, err := graphdomain.NewEdge(graphdomain.Edge{
		ID: id.New(), SiteID: siteID, FromEntityID: from, ToEntityID: to, Kind: kind, Weight: weight,
		Source: graphdomain.SourceAI, Status: graphdomain.StatusProposed, Reason: clip(reason), CreatedAt: now,
	})
	if err != nil {
		return nil
	}
	if connected(state.edges, edge) {
		return nil
	}
	if insertErr := s.edges.Insert(ctx, edge); insertErr != nil {
		if errors.IsCode(insertErr, errors.Conflict) {
			return nil
		}
		return insertErr
	}

	state.edges = append(state.edges, edge)
	out.Edges = append(out.Edges, edgeView(edge))
	return nil
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
