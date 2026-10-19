package contextpack

import (
	"cmp"
	"slices"
	"strings"

	"github.com/davidmovas/postulator/internal/domain/block"
	"github.com/davidmovas/postulator/internal/domain/content"
	"github.com/davidmovas/postulator/internal/domain/field"
	"github.com/davidmovas/postulator/internal/domain/finding"
	"github.com/davidmovas/postulator/internal/domain/graph"
	"github.com/davidmovas/postulator/internal/domain/keyword"
	"github.com/davidmovas/postulator/internal/domain/note"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/pagetype"
	"github.com/davidmovas/postulator/internal/domain/render"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

const (
	trailLimit       = 32
	productTaxonomy  = "product_cat"
	listedFromTwin   = "twin"
	listedFromRender = "listing"
	listedFromTerm   = "term"
)

type Ref struct {
	PageID  string          `json:"pageId"`
	Path    string          `json:"path"`
	Title   string          `json:"title"`
	Type    string          `json:"type"`
	Preset  pagetype.Preset `json:"preset"`
	OnSite  bool            `json:"onSite"`
	Summary Summary         `json:"summary"`
}

type Product struct {
	Ref
	Price        string `json:"price"`
	RegularPrice string `json:"regularPrice"`
	SalePrice    string `json:"salePrice"`
	SKU          string `json:"sku"`
	Stock        string `json:"stock"`
}

type Card struct {
	EntityID   string         `json:"entityId"`
	Name       string         `json:"name"`
	Definition string         `json:"definition"`
	Facts      []graph.Fact   `json:"facts"`
	Notes      []note.Note    `json:"notes"`
	Anchors    []graph.Anchor `json:"anchors"`
}

type Target struct {
	Ref
	Href     string           `json:"href"`
	Relation content.Relation `json:"relation"`
	Required bool             `json:"required"`
	Anchors  []string         `json:"anchors"`
}

type LinkRef struct {
	Type   pagemap.LinkType   `json:"type"`
	Source pagemap.LinkSource `json:"source"`
	Anchor string             `json:"anchor"`
	URL    string             `json:"url"`
	Page   *Ref               `json:"page"`
	Ours   bool               `json:"ours"`
}

type FieldFacts struct {
	Key     field.Key   `json:"key"`
	Current string      `json:"current"`
	Ours    bool        `json:"ours"`
	Scope   field.Scope `json:"scope"`
	Stored  bool        `json:"stored"`
}

type Facts struct {
	Page         Ref                     `json:"page"`
	H1           string                  `json:"h1"`
	WPType       pagemap.WPType          `json:"wpType"`
	Status       pagemap.Status          `json:"status"`
	TypeSource   pagemap.TypeSource      `json:"typeSource"`
	TypeReason   pagemap.TypeReason      `json:"typeReason"`
	Technical    pagemap.TechnicalReason `json:"technical"`
	ReadOnly     pagemap.ReadOnlyReason  `json:"readOnly"`
	ParentReason pagemap.ParentReason    `json:"parentReason"`
	Card         *Card                   `json:"card"`
	Trail        []Ref                   `json:"trail"`
	Children     []Ref                   `json:"children"`
	Siblings     []Ref                   `json:"siblings"`
	Products     []Product               `json:"products"`
	ListedFrom   string                  `json:"listedFrom"`
	Twin         *Ref                    `json:"twin"`
	Terms        []pagemap.Term          `json:"terms"`
	Keywords     keyword.List            `json:"keywords"`
	Targets      []Target                `json:"targets"`
	Outbound     []LinkRef               `json:"outbound"`
	Inbound      []LinkRef               `json:"inbound"`
	Fields       []FieldFacts            `json:"fields"`
	Written      []string                `json:"written"`
	Root         render.RootState        `json:"root"`
	Regions      []render.Region         `json:"regions"`
	Generated    int                     `json:"generated"`
	Notes        []note.Note             `json:"notes"`
	Findings     []finding.Finding       `json:"findings"`
}

func (f Facts) Field(key field.Key) (FieldFacts, bool) {
	for i := range f.Fields {
		if f.Fields[i].Key == key {
			return f.Fields[i], true
		}
	}
	return FieldFacts{}, false
}

func (s Snapshot) Facts(pageID string, plan content.LinkContext) (Facts, error) {
	page, found := s.Index.ByID(pageID)
	if !found {
		return Facts{}, errors.New(errors.NotFound, "the page is not in the site model").WithDetail("pageId", pageID)
	}
	kind := s.typeOf(page)
	facts := Facts{
		Page: s.ref(page, true), H1: h1Of(page), WPType: page.WPType, Status: page.Status,
		TypeSource: page.TypeSource, TypeReason: page.TypeReason, Technical: page.TechnicalReason,
		ReadOnly: page.ReadOnlyReason, ParentReason: page.ParentReason, Card: s.card(page),
		Trail: s.refs(s.trail(page), true), Children: s.refs(s.ranked(page, s.children(page)), true),
		Siblings: s.refs(s.ranked(page, s.siblings(page)), true), Terms: slices.Clone(s.Terms[page.ID]),
		Targets: s.targets(plan), Outbound: s.outbound(page.ID), Inbound: s.inbound(page.ID),
		Fields: s.fields(page, kind), Written: slices.Clone(s.Written[page.ID]), Notes: slices.Clone(page.Notes),
		Findings: s.findings(page.ID),
	}
	if entity, mapped := s.entity(page); mapped {
		facts.Keywords = pagemap.Keywords(page, entity)
	} else {
		facts.Keywords = pagemap.Keywords(page, graph.Entity{})
	}
	products, from := s.products(page)
	facts.Products, facts.ListedFrom = s.productFacts(s.ranked(page, products)), from
	if twin, paired := s.twin(page); paired {
		ref := s.ref(twin, false)
		facts.Twin = &ref
	}
	if record, rendered := s.Renders[page.ID]; rendered {
		facts.Root, facts.Regions, facts.Generated = record.Root, slices.Clone(record.Regions), record.WordsGenerated
	}
	return facts, nil
}

func (s Snapshot) ref(page pagemap.Page, summarized bool) Ref {
	kind := s.typeOf(page)
	ref := Ref{PageID: page.ID, Path: page.Path, Title: titleOf(page), Type: kind.Name, Preset: kind.Preset, OnSite: onSite(page)}
	if summarized {
		ref.Summary = s.Summary(page)
	}
	return ref
}

func (s Snapshot) refs(pages []pagemap.Page, summarized bool) []Ref {
	out := make([]Ref, 0, len(pages))
	for i := range pages {
		out = append(out, s.ref(pages[i], summarized))
	}
	return out
}

func (s Snapshot) card(page pagemap.Page) *Card {
	entity, mapped := s.entity(page)
	if !mapped {
		return nil
	}
	name := s.Graph.Label(entity.ID)
	if name == "" {
		name = entity.Name
	}
	return &Card{
		EntityID: entity.ID, Name: name, Definition: entity.Definition, Facts: slices.Clone(entity.Facts),
		Notes: slices.Clone(entity.Notes), Anchors: slices.Clone(entity.Anchors),
	}
}

func (s Snapshot) parent(page pagemap.Page) (pagemap.Page, bool) {
	if page.ParentPageID == nil || *page.ParentPageID == page.ID {
		return pagemap.Page{}, false
	}
	return s.Index.ByID(*page.ParentPageID)
}

func (s Snapshot) trail(page pagemap.Page) []pagemap.Page {
	out := make([]pagemap.Page, 0)
	seen := map[string]struct{}{page.ID: {}}
	current := page
	for len(out) < trailLimit {
		parent, found := s.parent(current)
		if !found {
			break
		}
		if _, looped := seen[parent.ID]; looped {
			break
		}
		seen[parent.ID] = struct{}{}
		out = append(out, parent)
		current = parent
	}
	return out
}

func (s Snapshot) children(page pagemap.Page) []pagemap.Page {
	out := make([]pagemap.Page, 0)
	pages := s.Index.Pages()
	for i := range pages {
		candidate := &pages[i]
		if candidate.ParentPageID != nil && *candidate.ParentPageID == page.ID && candidate.ID != page.ID && live(*candidate) {
			out = append(out, *candidate)
		}
	}
	return out
}

func (s Snapshot) siblings(page pagemap.Page) []pagemap.Page {
	parent, found := s.parent(page)
	if !found {
		return []pagemap.Page{}
	}
	children := s.children(parent)
	out := make([]pagemap.Page, 0, len(children))
	for i := range children {
		if children[i].ID != page.ID {
			out = append(out, children[i])
		}
	}
	return out
}

func (s Snapshot) twin(page pagemap.Page) (pagemap.Page, bool) {
	if page.WPType.Term() {
		return s.Index.HubTwin(page)
	}
	terms := s.Index.TermTwins(page)
	if len(terms) != 1 {
		return pagemap.Page{}, false
	}
	return terms[0], true
}

func (s Snapshot) products(page pagemap.Page) (pages []pagemap.Page, from string) {
	if page.WPType.Term() {
		return s.inTerm(page), listedFromTerm
	}
	listed := s.listed(page.ID)
	if twin, paired := s.twin(page); paired {
		if fromTwin := s.inTerm(twin); len(fromTwin) > len(listed) {
			return fromTwin, listedFromTwin
		}
	}
	if len(listed) == 0 {
		return listed, ""
	}
	return listed, listedFromRender
}

func (s Snapshot) listed(pageID string) []pagemap.Page {
	out := make([]pagemap.Page, 0)
	seen := make(map[string]struct{})
	for i := range s.Links {
		link := s.Links[i]
		if link.FromPageID != pageID || link.Type != pagemap.LinkListing || link.ToPageID == nil {
			continue
		}
		if _, dup := seen[*link.ToPageID]; dup {
			continue
		}
		target, found := s.Index.ByID(*link.ToPageID)
		if !found || target.WPType != pagemap.WPProduct || !live(target) {
			continue
		}
		seen[target.ID] = struct{}{}
		out = append(out, target)
	}
	return out
}

func (s Snapshot) inTerm(term pagemap.Page) []pagemap.Page {
	out := make([]pagemap.Page, 0)
	if term.WPID == nil {
		return out
	}
	pages := s.Index.Pages()
	for i := range pages {
		candidate := &pages[i]
		if candidate.WPType != pagemap.WPProduct || !live(*candidate) {
			continue
		}
		for _, held := range s.Terms[candidate.ID] {
			if held.Taxonomy == productTaxonomy && held.TermID == *term.WPID {
				out = append(out, *candidate)
				break
			}
		}
	}
	return out
}

func (s Snapshot) productFacts(pages []pagemap.Page) []Product {
	out := make([]Product, 0, len(pages))
	for i := range pages {
		id := pages[i].ID
		out = append(out, Product{
			Ref: s.ref(pages[i], true), Price: s.text(id, field.Price), RegularPrice: s.text(id, field.RegularPrice),
			SalePrice: s.text(id, field.SalePrice), SKU: s.text(id, field.SKU), Stock: s.text(id, field.StockStatus),
		})
	}
	return out
}

func (s Snapshot) ranked(page pagemap.Page, pages []pagemap.Page) []pagemap.Page {
	wanted := s.keywordKeys(page)
	type scored struct {
		page   pagemap.Page
		shared int
		score  float64
		at     int
	}
	rows := make([]scored, 0, len(pages))
	for i := range pages {
		row := scored{page: pages[i], shared: shared(wanted, s.keywordKeys(pages[i]), titleOf(pages[i])), at: i}
		if entity, mapped := s.entity(pages[i]); mapped {
			row.score = entity.Score
		}
		rows = append(rows, row)
	}
	slices.SortStableFunc(rows, func(a, b scored) int {
		if a.shared != b.shared {
			return cmp.Compare(b.shared, a.shared)
		}
		if a.score != b.score {
			return cmp.Compare(b.score, a.score)
		}
		return cmp.Compare(a.at, b.at)
	})
	out := make([]pagemap.Page, 0, len(rows))
	for i := range rows {
		out = append(out, rows[i].page)
	}
	return out
}

func (s Snapshot) keywordKeys(page pagemap.Page) []string {
	entity, _ := s.entity(page)
	list := pagemap.Keywords(page, entity)
	out := make([]string, 0, len(list))
	for _, item := range list {
		if key := graph.Key(item.Text); key != "" {
			out = append(out, key)
		}
	}
	return out
}

func shared(wanted, held []string, title string) int {
	titleKey := graph.Key(title)
	count := 0
	for _, key := range wanted {
		if slices.Contains(held, key) || (titleKey != "" && strings.Contains(titleKey, key)) {
			count++
		}
	}
	return count
}

func (s Snapshot) targets(plan content.LinkContext) []Target {
	out := make([]Target, 0, len(plan.Targets))
	for _, target := range plan.Targets {
		row := Target{
			Href: plan.Href(target), Relation: target.Relation, Required: target.Required, Anchors: slices.Clone(target.Anchors),
		}
		if page, found := s.Index.ByID(target.PageID); found {
			row.Ref = s.ref(page, true)
		} else {
			row.Ref = Ref{PageID: target.PageID, Path: target.URL, Title: target.URL}
		}
		out = append(out, row)
	}
	return out
}

func (s Snapshot) outbound(pageID string) []LinkRef {
	out := make([]LinkRef, 0)
	for i := range s.Links {
		if s.Links[i].FromPageID == pageID {
			out = append(out, s.linkRef(s.Links[i], s.Links[i].ToPageID))
		}
	}
	return out
}

func (s Snapshot) inbound(pageID string) []LinkRef {
	out := make([]LinkRef, 0)
	for i := range s.Links {
		if link := s.Links[i]; link.ToPageID != nil && *link.ToPageID == pageID && link.FromPageID != pageID {
			from := link.FromPageID
			out = append(out, s.linkRef(link, &from))
		}
	}
	return out
}

func (s Snapshot) linkRef(link pagemap.PageLink, other *string) LinkRef {
	ref := LinkRef{Type: link.Type, Source: link.Source, Anchor: link.AnchorText, URL: link.ToURL, Ours: link.Ours}
	if other != nil {
		if page, found := s.Index.ByID(*other); found {
			held := s.ref(page, false)
			ref.Page = &held
		}
	}
	return ref
}

func (s Snapshot) fields(page pagemap.Page, kind pagetype.Type) []FieldFacts {
	out := make([]FieldFacts, 0)
	listed := make(map[field.Key]struct{})
	for _, spec := range field.Catalog() {
		if spec.Denied || !spec.Text || field.CheckScope(spec.Key) != nil || !applies(page, spec.Key) {
			continue
		}
		scope, stored := s.Types.Scope(kind.ID, spec.Key)
		row := s.fieldFacts(page, spec.Key, scope, stored)
		if !scope.Enabled && row.Current == "" {
			continue
		}
		listed[spec.Key] = struct{}{}
		out = append(out, row)
	}
	custom := s.Types.Custom(kind.ID)
	for i := range custom {
		if _, held := listed[custom[i].Key]; !held {
			out = append(out, s.fieldFacts(page, custom[i].Key, custom[i], true))
		}
	}
	return out
}

func applies(page pagemap.Page, key field.Key) bool {
	switch key {
	case field.ProductDescription, field.ProductShortDescription, field.VariationDescription:
		return page.WPType == pagemap.WPProduct
	case field.Body, field.Excerpt:
		return page.WPType != pagemap.WPProduct
	default:
		return true
	}
}

func (s Snapshot) fieldFacts(page pagemap.Page, key field.Key, scope field.Scope, stored bool) FieldFacts {
	row := FieldFacts{Key: key, Scope: scope, Stored: stored}
	if key == field.Body {
		if body, found := s.body(page.ID); found {
			row.Current = body.Serialize()
			row.Ours = row.Current != "" && !block.ClientText(row.Current, s.Written[page.ID])
		}
		return row
	}
	if value, found := s.value(page.ID, key); found {
		row.Current = value.Current
		row.Ours = value.WrittenRunID != nil && value.CurrentHash == value.BaseHash
	}
	return row
}

func (s Snapshot) findings(pageID string) []finding.Finding {
	out := make([]finding.Finding, 0)
	for i := range s.Findings {
		if s.Findings[i].PageID != nil && *s.Findings[i].PageID == pageID {
			out = append(out, s.Findings[i])
		}
	}
	return out
}

func (s Snapshot) Neighborhood(pageID string, plan content.LinkContext) []string {
	page, found := s.Index.ByID(pageID)
	if !found {
		return []string{}
	}
	ids := []string{page.ID}
	add := func(pages []pagemap.Page) {
		for i := range pages {
			ids = append(ids, pages[i].ID)
		}
	}
	add(s.trail(page))
	add(s.children(page))
	add(s.siblings(page))
	if twin, paired := s.twin(page); paired {
		ids = append(ids, twin.ID)
		add(s.inTerm(twin))
	}
	add(s.listed(page.ID))
	for _, target := range plan.Targets {
		ids = append(ids, target.PageID)
	}
	slices.Sort(ids)
	return slices.Compact(ids)
}
