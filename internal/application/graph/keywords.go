package graph

import (
	"context"
	"slices"
	"strings"

	"github.com/davidmovas/postulator/internal/application/llm"
	graphdomain "github.com/davidmovas/postulator/internal/domain/graph"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

const (
	NameProposeFromKeywords = "propose_from_keywords"
	KeywordsPerCall         = 40
)

type keywordProposal struct {
	Keyword           string   `json:"keyword" description:"The keyword this entity answers, copied exactly as it was given"`
	Name              string   `json:"name" description:"The topic the keyword names, two to four words in title case"`
	Kind              string   `json:"kind" enum:"hub,category,topic,product" description:"What the entity is"`
	Intent            string   `json:"intent" description:"What the searcher wants, one short sentence"`
	SecondaryKeywords []string `json:"secondaryKeywords" description:"Other given keywords that mean the same topic, folded into this one"`
	Anchors           []string `json:"anchors" description:"At most four link texts another page could point here with"`
	ParentName        string   `json:"parentName" description:"The name of the entity one level up, from the existing or the proposed ones, empty at a root"`
	RelatedNames      []string `json:"relatedNames" description:"At most three names of proposed or existing entities of the same subject"`
}

type keywordsProposal struct {
	Entities []keywordProposal `json:"entities" description:"One entity per keyword, fewer when keywords share a topic"`
}

type keywordsPrompt struct {
	SiteName string
	Parent   string
	Known    []knownEntity
	Keywords []string
}

func (s *Service) ProposeFromKeywords(ctx context.Context, req ProposeFromKeywordsRequest) (ProposeFromKeywordsResponse, error) {
	siteID := strings.TrimSpace(req.SiteID)
	if err := requireSite(siteID); err != nil {
		return ProposeFromKeywordsResponse{}, err
	}
	keywords := graphdomain.CleanKeywords(req.Keywords)
	if len(keywords) == 0 {
		return ProposeFromKeywordsResponse{}, errors.New(errors.Invalid, "a proposal from keywords needs at least one keyword").
			WithDetail("field", "keywords")
	}

	owner, err := s.sites.Get(ctx, siteID)
	if err != nil {
		return ProposeFromKeywordsResponse{}, err
	}
	ref, err := s.model(ctx, siteID)
	if err != nil {
		return ProposeFromKeywordsResponse{}, err
	}
	state, err := s.snapshot(ctx, siteID)
	if err != nil {
		return ProposeFromKeywordsResponse{}, err
	}

	parent := ""
	if parentID := strings.TrimSpace(req.ParentEntityID); parentID != "" {
		for i := range state.entities {
			if state.entities[i].ID == parentID {
				parent = state.entities[i].Name
			}
		}
		if parent == "" {
			return ProposeFromKeywordsResponse{}, errors.New(errors.NotFound, "entity not found").
				WithDetail("entityId", parentID)
		}
	}

	response := ProposeFromKeywordsResponse{Entities: []ProposedEntity{}}
	for batch := range slices.Chunk(keywords, KeywordsPerCall) {
		system, user, renderErr := prompts.Render(NameProposeFromKeywords, keywordsPrompt{
			SiteName: owner.Name, Parent: parent, Known: knownEntitiesOf(state), Keywords: batch,
		})
		if renderErr != nil {
			return ProposeFromKeywordsResponse{}, renderErr
		}

		proposal, usage, callErr := llm.Structured[keywordsProposal](ctx, s.llm, llm.Request{
			Ref:       ref,
			System:    system,
			Messages:  []llm.Message{{Role: llm.RoleUser, Text: user}},
			MaxTokens: proposalTokens,
			Meta:      llm.CallMeta{Step: NameProposeFromKeywords},
		})
		if callErr != nil {
			return ProposeFromKeywordsResponse{}, callErr
		}
		response.Tokens += usage.Total

		for i := range proposal.Entities {
			proposed := &proposal.Entities[i]
			keyword := strings.TrimSpace(proposed.Keyword)
			name := strings.TrimSpace(proposed.Name)
			if name == "" || !containsFold(batch, keyword) {
				response.Skipped++
				continue
			}
			parentName := strings.TrimSpace(proposed.ParentName)
			if parentName == "" {
				parentName = parent
			}
			response.Entities = append(response.Entities, ProposedEntity{
				Name:              name,
				Kind:              string(kindOf(proposed.Kind)),
				Intent:            strings.TrimSpace(proposed.Intent),
				PrimaryKeyword:    strings.ToLower(keyword),
				SecondaryKeywords: graphdomain.CleanKeywords(proposed.SecondaryKeywords),
				Anchors:           graphdomain.CleanKeywords(proposed.Anchors),
				Parent:            parentName,
				Related:           graphdomain.CleanKeywords(proposed.RelatedNames),
				ExistingEntityID:  state.byName[fold(name)],
			})
		}
	}
	return response, nil
}

func knownEntitiesOf(state siteGraph) []knownEntity {
	known := make([]knownEntity, 0, len(state.entities))
	for i := range state.entities {
		known = append(known, knownEntity{Name: state.entities[i].Name, Kind: string(state.entities[i].Kind)})
	}
	slices.SortFunc(known, func(a, b knownEntity) int { return strings.Compare(a.Name, b.Name) })
	return known
}

func containsFold(values []string, wanted string) bool {
	for _, value := range values {
		if strings.EqualFold(strings.TrimSpace(value), strings.TrimSpace(wanted)) {
			return true
		}
	}
	return false
}
