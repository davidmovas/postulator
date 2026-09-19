package graph

import (
	graphdomain "github.com/davidmovas/postulator/internal/domain/graph"
	"github.com/davidmovas/postulator/internal/kernel/dto"
)

type Anchor struct {
	Text   string  `json:"text"`
	Source string  `json:"source"`
	Weight float64 `json:"weight"`
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

func anchorsOf(anchors []Anchor) []graphdomain.Anchor {
	out := make([]graphdomain.Anchor, 0, len(anchors))
	for _, anchor := range anchors {
		out = append(out, graphdomain.Anchor{Text: anchor.Text, Source: graphdomain.AnchorSource(anchor.Source), Weight: anchor.Weight})
	}
	return out
}
