package graph

import (
	"context"

	"github.com/davidmovas/postulator/internal/application"
	graphdomain "github.com/davidmovas/postulator/internal/domain/graph"
	kctx "github.com/davidmovas/postulator/internal/kernel/ctx"
	"github.com/davidmovas/postulator/internal/kernel/dto"
)

type Anchor struct {
	Text   string  `json:"text" description:"Link text"`
	Source string  `json:"source,omitempty" enum:"user,ai" description:"Who wrote it; default ai"`
	Weight float64 `json:"weight,omitempty" minimum:"0" maximum:"1" description:"Preference over the others; omit for none"`
}

type Fact struct {
	Label string `json:"label" description:"What the fact is about"`
	Value string `json:"value" description:"The fact"`
}

type Note struct {
	Label string `json:"label" description:"Note heading"`
	Text  string `json:"text" description:"Note text"`
}

type Entity struct {
	ID              string        `json:"id"`
	SiteID          string        `json:"siteId"`
	Name            string        `json:"name"`
	Kind            string        `json:"kind"`
	Intent          string        `json:"intent"`
	Definition      string        `json:"definition"`
	Facts           []Fact        `json:"facts"`
	Notes           []Note        `json:"notes"`
	Keywords        []dto.Keyword `json:"keywords"`
	Anchors         []Anchor      `json:"anchors"`
	ScopeEntityID   *string       `json:"scopeEntityId"`
	CanonicalPageID *string       `json:"canonicalPageId"`
	Score           float64       `json:"score"`
	Source          string        `json:"source"`
	CreatedAt       dto.Time      `json:"createdAt"`
	UpdatedAt       dto.Time      `json:"updatedAt"`
}

type Edge struct {
	ID           string   `json:"id"`
	SiteID       string   `json:"siteId"`
	FromEntityID string   `json:"fromEntityId"`
	ToEntityID   string   `json:"toEntityId"`
	Kind         string   `json:"kind"`
	Weight       float64  `json:"weight"`
	Source       string   `json:"source"`
	Status       string   `json:"status"`
	Reason       string   `json:"reason"`
	CreatedAt    dto.Time `json:"createdAt"`
}

type EntityPage struct {
	EntityID string `json:"entityId"`
	PageID   string `json:"pageId"`
	Path     string `json:"path"`
	Status   string `json:"status"`
	Work     string `json:"work"`
	Mismatch bool   `json:"mismatch"`
}

func entityView(e graphdomain.Entity) Entity {
	anchors := make([]Anchor, 0, len(e.Anchors))
	for _, anchor := range e.Anchors {
		anchors = append(anchors, Anchor{Text: anchor.Text, Source: string(anchor.Source), Weight: anchor.Weight})
	}
	facts := make([]Fact, 0, len(e.Facts))
	for _, fact := range e.Facts {
		facts = append(facts, Fact(fact))
	}
	notes := make([]Note, 0, len(e.Notes))
	for _, n := range e.Notes {
		notes = append(notes, Note(n))
	}
	return Entity{
		ID:              e.ID,
		SiteID:          e.SiteID,
		Name:            e.Name,
		Kind:            string(e.Kind),
		Intent:          e.Intent,
		Definition:      e.Definition,
		Facts:           facts,
		Notes:           notes,
		Keywords:        application.KeywordViews(e.Keywords),
		Anchors:         anchors,
		ScopeEntityID:   e.ScopeID,
		CanonicalPageID: e.CanonicalPageID,
		Score:           e.Score,
		Source:          string(e.Source),
		CreatedAt:       dto.NewTime(e.CreatedAt),
		UpdatedAt:       dto.NewTime(e.UpdatedAt),
	}
}

func requestedAnchors(ctx context.Context, anchors []Anchor) []graphdomain.Anchor {
	fallback := anchorSourceOfActor(ctx)

	out := make([]graphdomain.Anchor, 0, len(anchors))
	for _, anchor := range anchors {
		source := graphdomain.AnchorSource(anchor.Source)
		if anchor.Source == "" {
			source = fallback
		}
		out = append(out, graphdomain.Anchor{Text: anchor.Text, Source: source, Weight: anchor.Weight})
	}
	return out
}

func anchorSourceOfActor(ctx context.Context) graphdomain.AnchorSource {
	if actor, ok := kctx.ActorFrom(ctx); ok && actor == kctx.ActorAgent {
		return graphdomain.AnchorAI
	}
	return graphdomain.AnchorUser
}

func sourceOfActor(ctx context.Context) graphdomain.Source {
	if actor, ok := kctx.ActorFrom(ctx); ok && actor == kctx.ActorAgent {
		return graphdomain.SourceAI
	}
	return graphdomain.SourceUser
}
