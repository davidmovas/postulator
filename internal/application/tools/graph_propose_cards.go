package tools

import (
	"context"

	"github.com/davidmovas/postulator/internal/application/graph"
)

const graphProposeCardsName = "graph_propose_cards"

type cardFactArgs struct {
	Label string `json:"label" description:"What the fact is about"`
	Value string `json:"value" description:"The fact"`
}

type proposedCardArgs struct {
	EntityID   string         `json:"entityId" description:"Entity id from graph_list_entities"`
	Definition string         `json:"definition,omitempty" description:"One or two sentences; omit to keep it"`
	Facts      []cardFactArgs `json:"facts,omitempty" description:"Whole new list; omit to keep them"`
	Reason     string         `json:"reason,omitempty" description:"Where it comes from"`
}

type graphProposeCardsArgs struct {
	Cards []proposedCardArgs `json:"cards" description:"One per entity, at most 50"`
}

func (in graphProposeCardsArgs) inputs() []graph.CardInput {
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

func graphProposeCards(deps Deps) Tool {
	return newSiteTool(Def{
		Name:        graphProposeCardsName,
		Description: "Propose entity cards, a definition and key facts each; a person accepts each before it is written.",
		Risk:        RiskWrite,
	}, func(ctx context.Context, b Binding, in graphProposeCardsArgs) (any, error) {
		if b.Approved {
			return deps.Graph.ApplyCards(ctx, graph.ApplyCardsRequest{SiteID: b.SiteID, Cards: in.inputs()})
		}
		return deps.Graph.ProposeCards(ctx, graph.ProposeCardsRequest{SiteID: b.SiteID, Cards: in.inputs()})
	})
}
