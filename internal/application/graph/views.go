package graph

import (
	"context"

	graphdomain "github.com/davidmovas/postulator/internal/domain/graph"
	kctx "github.com/davidmovas/postulator/internal/kernel/ctx"
	"github.com/davidmovas/postulator/internal/kernel/dto"
)

type Anchor struct {
	Text   string  `json:"text" description:"The link text another page points here with"`
	Source string  `json:"source,omitempty" enum:"user,ai" description:"Who wrote the anchor; leave it out and an anchor you propose is recorded as ai"`
	Weight float64 `json:"weight,omitempty" minimum:"0" maximum:"1" description:"How strongly to prefer this anchor over the others, between 0 and 1; leave it out for no preference"`
}

type Entity struct {
	ID                string   `json:"id"`
	SiteID            string   `json:"siteId"`
	Name              string   `json:"name"`
	Kind              string   `json:"kind"`
	Intent            string   `json:"intent"`
	PrimaryKeyword    string   `json:"primaryKeyword"`
	SecondaryKeywords []string `json:"secondaryKeywords"`
	Anchors           []Anchor `json:"anchors"`
	CanonicalPageID   *string  `json:"canonicalPageId"`
	Score             float64  `json:"score"`
	Source            string   `json:"source"`
	CreatedAt         dto.Time `json:"createdAt"`
	UpdatedAt         dto.Time `json:"updatedAt"`
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
	keywords := e.SecondaryKeywords
	if keywords == nil {
		keywords = []string{}
	}
	anchors := make([]Anchor, 0, len(e.Anchors))
	for _, anchor := range e.Anchors {
		anchors = append(anchors, Anchor{Text: anchor.Text, Source: string(anchor.Source), Weight: anchor.Weight})
	}
	return Entity{
		ID:                e.ID,
		SiteID:            e.SiteID,
		Name:              e.Name,
		Kind:              string(e.Kind),
		Intent:            e.Intent,
		PrimaryKeyword:    e.PrimaryKeyword,
		SecondaryKeywords: keywords,
		Anchors:           anchors,
		CanonicalPageID:   e.CanonicalPageID,
		Score:             e.Score,
		Source:            string(e.Source),
		CreatedAt:         dto.NewTime(e.CreatedAt),
		UpdatedAt:         dto.NewTime(e.UpdatedAt),
	}
}

func anchorsOf(ctx context.Context, anchors []Anchor) []graphdomain.Anchor {
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
