package contextpack

import (
	"cmp"
	"encoding/json"
	"slices"
	"strings"

	"github.com/davidmovas/postulator/internal/domain/field"
	"github.com/davidmovas/postulator/internal/domain/graph"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
)

type ProposalClash string

const (
	ClashNone    ProposalClash = ""
	ClashNoName  ProposalClash = "no_name"
	ClashInBatch ProposalClash = "name_in_batch"
	ClashCarried ProposalClash = "name_carried"
)

func ProposalClashes() []ProposalClash {
	return []ProposalClash{ClashNoName, ClashInBatch, ClashCarried}
}

type EntityProposal struct {
	PageID           string
	Path             string
	Name             string
	Label            string
	Parent           string
	ParentEntityID   string
	ExistingEntityID string
	Facts            []graph.Fact
	Clash            ProposalClash
	depth            int
	parentPageID     string
}

type storeAttribute struct {
	Name    string   `json:"name"`
	Options []string `json:"options"`
}

const proposedScope = "proposal:"

func (m Map) EntityProposals(pageIDs []string) []EntityProposal {
	candidates := m.proposable(pageIDs)
	inBatch := make(map[string]struct{}, len(candidates))
	for i := range candidates {
		inBatch[candidates[i].ID] = struct{}{}
	}
	out := make([]EntityProposal, 0, len(candidates))
	for i := range candidates {
		out = append(out, m.proposalFor(candidates[i], inBatch))
	}
	slices.SortFunc(out, func(a, b EntityProposal) int {
		return cmp.Or(cmp.Compare(a.depth, b.depth), strings.Compare(a.Path, b.Path))
	})
	m.settleNames(out)
	return out
}

func (m Map) proposable(pageIDs []string) []pagemap.Page {
	if len(pageIDs) == 0 {
		out := make([]pagemap.Page, 0)
		pages := m.snapshot.Index.Pages()
		for i := range pages {
			if _, folded := m.folded[pages[i].ID]; !folded && m.snapshot.Types.NeedsEntity(pages[i]) {
				out = append(out, pages[i])
			}
		}
		return out
	}
	out := make([]pagemap.Page, 0, len(pageIDs))
	for _, pageID := range pageIDs {
		page, found := m.snapshot.Index.ByID(pageID)
		if !found || page.WPType.Term() || page.EntityID != nil || page.Path == pagemap.RootPath || page.Status == pagemap.StatusArchived {
			continue
		}
		out = append(out, page)
	}
	return out
}

func (m Map) proposalFor(page pagemap.Page, inBatch map[string]struct{}) EntityProposal {
	proposal := EntityProposal{
		PageID: page.ID, Path: page.Path, Name: m.snapshot.siteName(page), Facts: m.snapshot.storeFacts(page), depth: m.depthOf(page.ID),
	}
	if parent, found := m.ProposalParent(page.ID, inBatch); found {
		proposal.Parent, proposal.ParentEntityID = parent.Path, parent.EntityID
		if parent.EntityID == "" {
			proposal.parentPageID = parent.PageID
		}
	}
	return proposal
}

func (m Map) ProposalParent(pageID string, inBatch map[string]struct{}) (TreeParent, bool) {
	start := pageID
	if hub, folded := m.folded[pageID]; folded {
		start = hub
	}
	seen := map[string]struct{}{start: {}}
	for at := m.parent[start]; at != ""; at = m.parent[at] {
		if _, looped := seen[at]; looped {
			break
		}
		seen[at] = struct{}{}
		ancestor, found := m.snapshot.Index.ByID(at)
		if !found || ancestor.Path == pagemap.RootPath {
			break
		}
		if _, proposed := inBatch[at]; proposed {
			return TreeParent{PageID: ancestor.ID, Path: ancestor.Path}, true
		}
		if ancestor.EntityID != nil {
			return TreeParent{PageID: ancestor.ID, Path: ancestor.Path, EntityID: *ancestor.EntityID}, true
		}
	}
	return TreeParent{}, false
}

func (p EntityProposal) scope() string {
	if p.ParentEntityID != "" {
		return p.ParentEntityID
	}
	if p.parentPageID != "" {
		return proposedScope + p.parentPageID
	}
	return ""
}

func (m Map) settleNames(proposals []EntityProposal) {
	entities := m.snapshot.Graph.Entities()
	carried := make(map[string]struct{})
	pages := m.snapshot.Index.Pages()
	for i := range pages {
		if pages[i].EntityID != nil {
			carried[*pages[i].EntityID] = struct{}{}
		}
	}
	taken := make(map[string]struct{}, len(proposals))
	pseudo := slices.Clone(entities)
	for i := range proposals {
		proposal := &proposals[i]
		if proposal.Name == "" {
			proposal.Clash = ClashNoName
			continue
		}
		key := proposal.scope() + "\x00" + graph.Key(proposal.Name)
		if _, twice := taken[key]; twice {
			proposal.Clash = ClashInBatch
			continue
		}
		taken[key] = struct{}{}
		if existing, held := sameNameUnder(entities, proposal.scope(), proposal.Name); held {
			if _, used := carried[existing.ID]; used || existing.CanonicalPageID != nil {
				proposal.Clash = ClashCarried
				continue
			}
			proposal.ExistingEntityID = existing.ID
			continue
		}
		scope := proposal.scope()
		entity := graph.Entity{ID: proposedScope + proposal.PageID, Name: proposal.Name}
		if scope != "" {
			entity.ScopeID = &scope
		}
		pseudo = append(pseudo, entity)
	}
	labels := graph.Labels(pseudo)
	for i := range proposals {
		proposal := &proposals[i]
		switch {
		case proposal.Clash != ClashNone:
			proposal.Label = proposal.Name
		case proposal.ExistingEntityID != "":
			proposal.Label = labels[proposal.ExistingEntityID]
		default:
			proposal.Label = labels[proposedScope+proposal.PageID]
		}
	}
}

func sameNameUnder(entities []graph.Entity, scope, name string) (graph.Entity, bool) {
	key := graph.Key(name)
	for i := range entities {
		held := ""
		if entities[i].ScopeID != nil {
			held = *entities[i].ScopeID
		}
		if held == scope && graph.Key(entities[i].Name) == key {
			return entities[i], true
		}
	}
	return graph.Entity{}, false
}

func (s Snapshot) siteName(page pagemap.Page) string {
	for _, held := range []string{s.text(page.ID, field.H1), page.H1, s.text(page.ID, field.Title), page.Title} {
		if name := strings.TrimSpace(held); name != "" {
			return name
		}
	}
	return ""
}

func (s Snapshot) storeFacts(page pagemap.Page) []graph.Fact {
	if page.WPType != pagemap.WPProduct {
		return nil
	}
	raw := s.text(page.ID, field.Attributes)
	if raw == "" {
		return nil
	}
	var attributes []storeAttribute
	if err := json.Unmarshal([]byte(raw), &attributes); err != nil {
		return nil
	}
	facts := make([]graph.Fact, 0, len(attributes))
	for _, attribute := range attributes {
		facts = append(facts, graph.Fact{Label: attribute.Name, Value: strings.Join(attribute.Options, ", ")})
	}
	return graph.CleanFacts(facts)
}
