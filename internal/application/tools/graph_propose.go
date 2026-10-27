package tools

import (
	"context"
	"strconv"
	"strings"

	"github.com/davidmovas/postulator/internal/application/graph"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

const (
	graphProposeName = "graph_propose"
	opFromPages      = "pages"
	opFromKeywords   = "keywords"
	opRelated        = "related"
	opCards          = "cards"
	proposalsNamed   = 3
)

type cardFactArgs struct {
	Label string `json:"label" description:"What the fact is about"`
	Value string `json:"value" description:"The fact"`
}

type proposedCardArgs struct {
	EntityID   string         `json:"entityId" description:"Entity id"`
	Definition string         `json:"definition,omitempty" description:"One or two sentences; omit to keep it"`
	Facts      []cardFactArgs `json:"facts,omitempty" description:"Whole new list; omit to keep them"`
	Reason     string         `json:"reason,omitempty" description:"Where it comes from"`
}

type graphProposeArgs struct {
	Op             string             `json:"op" enum:"pages,preview,keywords,related,cards" description:"preview and keywords write nothing, their proposals go to graph_edit apply_proposals; pages proposes and writes at once; related proposes related edges; cards proposes entity cards"`
	PageIDs        []string           `json:"pageIds,omitempty" description:"pages, preview: only these pages; default every page with no entity"`
	PathPrefix     string             `json:"pathPrefix,omitempty" description:"pages, preview: only paths starting with this"`
	Source         string             `json:"source,omitempty" enum:"model,site" description:"preview: site names them from the pages and their tree with no model call; default model"`
	Keywords       []string           `json:"keywords,omitempty" description:"keywords: one phrase each, volume optional in brackets: bpc 157 (1200)"`
	ParentEntityID string             `json:"parentEntityId,omitempty" description:"keywords: the proposals' default parent entity"`
	EntityID       string             `json:"entityId,omitempty" description:"related: only pairs with this entity"`
	Cards          []proposedCardArgs `json:"cards,omitempty" description:"cards: one per entity, at most 50; a person accepts each"`
}

func (in *graphProposeArgs) inputs() []graph.CardInput {
	out := make([]graph.CardInput, 0, len(in.Cards))
	for i := range in.Cards {
		card := &in.Cards[i]
		input := graph.CardInput{EntityID: card.EntityID, Definition: card.Definition, Reason: card.Reason}
		if card.Facts != nil {
			input.Facts = make([]graph.Fact, 0, len(card.Facts))
			for _, fact := range card.Facts {
				input.Facts = append(input.Facts, graph.Fact{Label: fact.Label, Value: fact.Value})
			}
		}
		out = append(out, input)
	}
	return out
}

func graphPropose(deps Deps) Tool {
	tool := newSiteTool(Def{
		Name: graphProposeName,
		Description: "Propose entities for pages or keywords, related edges between entities, or entity cards; a " +
			"person sees each proposal before it is written.",
		Risk: RiskWrite,
	}, func(ctx context.Context, b Binding, in graphProposeArgs) (any, error) {
		switch in.Op {
		case opFromPages:
			return deps.Graph.ProposeFromPages(ctx, graph.ProposeFromPagesRequest{SiteID: b.SiteID, PageIDs: in.PageIDs, PathPrefix: in.PathPrefix})
		case opPreview:
			return deps.Graph.PreviewFromPages(ctx, graph.PreviewFromPagesRequest{
				SiteID: b.SiteID, PageIDs: in.PageIDs, PathPrefix: in.PathPrefix, Source: graph.ProposalSource(in.Source),
			})
		case opFromKeywords:
			return deps.Graph.ProposeFromKeywords(ctx, graph.ProposeFromKeywordsRequest{
				SiteID: b.SiteID, Keywords: in.Keywords, ParentEntityID: in.ParentEntityID,
			})
		case opRelated:
			return deps.Graph.ProposeRelated(ctx, graph.ProposeRelatedRequest{SiteID: b.SiteID, EntityID: in.EntityID})
		case opCards:
			if b.Approved {
				return deps.Graph.ApplyCards(ctx, graph.ApplyCardsRequest{SiteID: b.SiteID, Cards: in.inputs()})
			}
			return deps.Graph.ProposeCards(ctx, graph.ProposeCardsRequest{SiteID: b.SiteID, Cards: in.inputs()})
		default:
			return nil, errors.New(errors.Invalid, "graph_propose takes the op pages, preview, keywords, related or cards").
				WithDetail("field", "op")
		}
	})
	tool = confirming(tool, func(_ context.Context, _ Binding, in graphProposeArgs) (string, error) {
		if in.Op != opFromPages {
			return "", nil
		}
		return "Ask the model to propose entities for " + proposedPages(&in) + " and write them at once, mapping each page " +
			"and proposing its edges; graph_propose preview shows them before anything is written", nil
	})
	tool = describing(tool, func(ctx context.Context, b Binding, in graphProposeArgs) (string, error) {
		switch in.Op {
		case opRelated:
			if in.EntityID == "" {
				return "Ask the model to propose related edges between the site's entities, each left for a person to approve", nil
			}
			name, err := deps.entityName(ctx, in.EntityID)
			if err != nil {
				return "", err
			}
			return "Ask the model to propose related edges around " + name + ", each left for a person to approve", nil
		default:
			return deps.cardsSummary(ctx, &in)
		}
	})
	return readsOn(tool, opPreview, opFromKeywords)
}

func proposedPages(in *graphProposeArgs) string {
	switch prefix := strings.TrimSpace(in.PathPrefix); {
	case len(in.PageIDs) > 0:
		return "the " + strconv.Itoa(len(in.PageIDs)) + " pages named"
	case prefix != "":
		return "the pages without an entity under " + prefix
	default:
		return "every page without an entity"
	}
}

func (deps Deps) cardsSummary(ctx context.Context, in *graphProposeArgs) (string, error) {
	named := make([]string, 0, proposalsNamed)
	for i := range in.Cards[:min(len(in.Cards), proposalsNamed)] {
		name, err := deps.entityName(ctx, in.Cards[i].EntityID)
		if err != nil {
			return "", err
		}
		named = append(named, name)
	}
	said := "Write the entity card of " + strings.Join(named, ", ")
	if len(in.Cards) > 1 {
		said = "Write the entity cards of " + strings.Join(named, ", ")
	}
	if more := len(in.Cards) - len(named); more > 0 {
		said += " and " + strconv.Itoa(more) + " more"
	}
	return said + "; a card that already says this is left as it is", nil
}

func proposalsSummary(in graph.ApplyProposalsRequest) (string, error) {
	if err := in.Validate(); err != nil {
		return "", err
	}
	named := make([]string, 0, proposalsNamed)
	for i := range in.Entities {
		if len(named) == proposalsNamed {
			break
		}
		shown := strings.TrimSpace(in.Entities[i].Name)
		if path := strings.TrimSpace(in.Entities[i].Path); path != "" {
			shown += " (" + path + ")"
		}
		named = append(named, shown)
	}
	said := "Write " + strconv.Itoa(len(in.Entities)) + " entities and map their pages: " + strings.Join(named, ", ")
	if more := len(in.Entities) - len(named); more > 0 {
		said += " and " + strconv.Itoa(more) + " more"
	}
	if in.Source == graph.SourceSite {
		return said + "; each one's parent edge is written approved and its name scoped under that parent", nil
	}
	return said + "; their edges are written as proposals for a person to approve", nil
}
