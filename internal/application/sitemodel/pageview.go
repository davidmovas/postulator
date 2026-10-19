package sitemodel

import (
	"github.com/davidmovas/postulator/internal/domain/contextpack"
	"github.com/davidmovas/postulator/internal/domain/keyword"
	"github.com/davidmovas/postulator/internal/domain/note"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/render"
)

const pageListLimit = 25

type FactView struct {
	Label string `json:"label"`
	Value string `json:"value"`
}

type CardView struct {
	EntityID   string      `json:"entityId"`
	Name       string      `json:"name"`
	Definition string      `json:"definition"`
	Facts      []FactView  `json:"facts"`
	Anchors    []string    `json:"anchors"`
	Notes      []note.Note `json:"notes"`
}

type FieldView struct {
	Key         string `json:"key"`
	Current     string `json:"current"`
	Ours        bool   `json:"ours"`
	Enabled     bool   `json:"enabled"`
	AIWrites    bool   `json:"aiWrites"`
	Rewrites    bool   `json:"rewrites"`
	Links       bool   `json:"links"`
	Instruction string `json:"instruction"`
}

type FindingView struct {
	Code     string `json:"code"`
	Severity string `json:"severity"`
}

type PageView struct {
	Page         contextpack.Ref         `json:"page"`
	H1           string                  `json:"h1"`
	WPType       pagemap.WPType          `json:"wpType"`
	Status       pagemap.Status          `json:"status"`
	TypeSource   pagemap.TypeSource      `json:"typeSource"`
	TypeReason   pagemap.TypeReason      `json:"typeReason"`
	Technical    pagemap.TechnicalReason `json:"technical"`
	ReadOnly     pagemap.ReadOnlyReason  `json:"readOnly"`
	ParentReason pagemap.ParentReason    `json:"parentReason"`
	Entity       *CardView               `json:"entity"`
	Trail        []contextpack.Ref       `json:"trail"`
	Children     []contextpack.Ref       `json:"children"`
	Siblings     []contextpack.Ref       `json:"siblings"`
	Products     []contextpack.Product   `json:"products"`
	ListedFrom   string                  `json:"listedFrom"`
	Twin         *contextpack.Ref        `json:"twin"`
	Terms        []pagemap.Term          `json:"terms"`
	Keywords     keyword.List            `json:"keywords"`
	Targets      []contextpack.Target    `json:"targets"`
	Outbound     []contextpack.LinkRef   `json:"outbound"`
	Inbound      []contextpack.LinkRef   `json:"inbound"`
	Fields       []FieldView             `json:"fields"`
	Root         render.RootState        `json:"root"`
	Regions      []render.Region         `json:"regions"`
	Generated    int                     `json:"generatedWords"`
	Notes        []note.Note             `json:"notes"`
	Findings     []FindingView           `json:"findings"`
	More         map[string]int          `json:"more"`
}

func pageView(facts contextpack.Facts) PageView {
	more := make(map[string]int)
	view := PageView{
		Page: facts.Page, H1: facts.H1, WPType: facts.WPType, Status: facts.Status, TypeSource: facts.TypeSource,
		TypeReason: facts.TypeReason, Technical: facts.Technical, ReadOnly: facts.ReadOnly, ParentReason: facts.ParentReason,
		Entity: cardView(facts.Card), Trail: nonNilRefs(facts.Trail),
		Children: capped(facts.Children, "children", more), Siblings: capped(facts.Siblings, "siblings", more),
		Products: capped(facts.Products, "products", more), ListedFrom: facts.ListedFrom, Twin: facts.Twin,
		Terms: capped(facts.Terms, "terms", more), Keywords: capped(facts.Keywords, "keywords", more),
		Targets: capped(facts.Targets, "targets", more), Outbound: capped(facts.Outbound, "outbound", more),
		Inbound: capped(facts.Inbound, "inbound", more), Fields: make([]FieldView, 0, len(facts.Fields)),
		Root: facts.Root, Regions: capped(facts.Regions, "regions", more), Generated: facts.Generated,
		Notes: capped(facts.Notes, "notes", more), Findings: make([]FindingView, 0, len(facts.Findings)), More: more,
	}
	for i := range facts.Fields {
		held := &facts.Fields[i]
		view.Fields = append(view.Fields, FieldView{
			Key: string(held.Key), Current: held.Current, Ours: held.Ours, Enabled: held.Scope.Enabled, AIWrites: held.Scope.AIWrites,
			Rewrites: held.Scope.Rewrites, Links: held.Scope.Enabled && held.Scope.Links.Links, Instruction: held.Scope.Instruction,
		})
	}
	for i := range facts.Findings {
		view.Findings = append(view.Findings, FindingView{Code: string(facts.Findings[i].Code), Severity: string(facts.Findings[i].Severity)})
	}
	return view
}

func cardView(card *contextpack.Card) *CardView {
	if card == nil {
		return nil
	}
	view := &CardView{
		EntityID: card.EntityID, Name: card.Name, Definition: card.Definition,
		Facts: make([]FactView, 0, len(card.Facts)), Anchors: make([]string, 0, len(card.Anchors)), Notes: card.Notes,
	}
	if view.Notes == nil {
		view.Notes = []note.Note{}
	}
	for _, fact := range card.Facts {
		view.Facts = append(view.Facts, FactView{Label: fact.Label, Value: fact.Value})
	}
	for _, anchor := range card.Anchors {
		view.Anchors = append(view.Anchors, anchor.Text)
	}
	return view
}

func capped[T any](items []T, name string, more map[string]int) []T {
	if len(items) > pageListLimit {
		more[name] = len(items) - pageListLimit
		return items[:pageListLimit]
	}
	if items == nil {
		return []T{}
	}
	return items
}

func nonNilRefs(refs []contextpack.Ref) []contextpack.Ref {
	if refs == nil {
		return []contextpack.Ref{}
	}
	return refs
}
