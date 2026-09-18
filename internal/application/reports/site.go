package reports

import (
	"cmp"
	"context"
	"slices"
	"strings"

	"github.com/davidmovas/postulator/internal/domain/graph"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
)

func (s *Service) SiteOverview(ctx context.Context, req SiteOverviewRequest) (SiteOverviewResponse, error) {
	siteID := strings.TrimSpace(req.SiteID)
	if siteID == "" {
		return SiteOverviewResponse{}, invalid("an overview needs a site", "siteId")
	}

	entities, err := s.entities.ListBySite(ctx, siteID)
	if err != nil {
		return SiteOverviewResponse{}, err
	}
	edges, err := s.edges.ListBySite(ctx, siteID)
	if err != nil {
		return SiteOverviewResponse{}, err
	}
	pages, err := s.pages.ListBySite(ctx, siteID)
	if err != nil {
		return SiteOverviewResponse{}, err
	}
	links, err := s.links.ListBySite(ctx, siteID)
	if err != nil {
		return SiteOverviewResponse{}, err
	}

	index := pagemap.NewIndex(pages)
	outgoing, incoming := linkSets(links)

	return SiteOverviewResponse{
		SiteID:   siteID,
		Entities: entityTotals(entities, index),
		Pages:    pageTotals(pages, incoming),
		Edges:    edgeTotals(edges, entities, outgoing),
		Depth:    depthHistogram(pages),
		Top:      topEntities(entities, index),
	}, nil
}

func linkSets(links []pagemap.PageLink) (outgoing map[string]map[string]struct{}, incoming map[string]struct{}) {
	outgoing = make(map[string]map[string]struct{})
	incoming = make(map[string]struct{})
	for i := range links {
		if links[i].ToPageID == nil {
			continue
		}
		targets, ok := outgoing[links[i].FromPageID]
		if !ok {
			targets = make(map[string]struct{})
			outgoing[links[i].FromPageID] = targets
		}
		targets[*links[i].ToPageID] = struct{}{}
		incoming[*links[i].ToPageID] = struct{}{}
	}
	return outgoing, incoming
}

func entityTotals(entities []graph.Entity, index pagemap.Index) EntityTotals {
	totals := EntityTotals{Total: len(entities)}
	for i := range entities {
		page, ok := canonical(entities[i], index)
		if !ok {
			continue
		}
		totals.WithCanonicalPage++
		if page.Status == pagemap.StatusPublished {
			totals.WithPublishedPage++
		}
	}
	return totals
}

func canonical(entity graph.Entity, index pagemap.Index) (pagemap.Page, bool) {
	if entity.CanonicalPageID == nil {
		return pagemap.Page{}, false
	}
	return index.ByID(*entity.CanonicalPageID)
}

func pageTotals(pages []pagemap.Page, incoming map[string]struct{}) PageTotals {
	totals := PageTotals{Total: len(pages), ByStatus: make(map[string]int)}
	for i := range pages {
		page := &pages[i]
		totals.ByStatus[string(page.Status)]++
		if page.EntityID == nil {
			totals.Unmapped++
		}
		if page.Status == orphanStatus {
			continue
		}
		if _, linked := incoming[page.ID]; !linked {
			totals.Orphans++
		}
	}
	return totals
}

func edgeTotals(edges []graph.Edge, entities []graph.Entity, outgoing map[string]map[string]struct{}) EdgeTotals {
	byEntity := make(map[string]graph.Entity, len(entities))
	for i := range entities {
		byEntity[entities[i].ID] = entities[i]
	}

	totals := EdgeTotals{}
	for i := range edges {
		edge := &edges[i]
		if edge.Status != graph.StatusApproved {
			continue
		}
		totals.Approved++

		from, fromOK := pageOf(byEntity, edge.FromEntityID)
		to, toOK := pageOf(byEntity, edge.ToEntityID)
		if !fromOK || !toOK {
			continue
		}
		if linked(outgoing, from, to) {
			totals.Realized++
			continue
		}
		if string(edge.Kind) == relatedEdge && linked(outgoing, to, from) {
			totals.Realized++
		}
	}
	return totals
}

func pageOf(byEntity map[string]graph.Entity, entityID string) (string, bool) {
	entity, ok := byEntity[entityID]
	if !ok || entity.CanonicalPageID == nil {
		return "", false
	}
	return *entity.CanonicalPageID, true
}

func linked(outgoing map[string]map[string]struct{}, from, to string) bool {
	targets, ok := outgoing[from]
	if !ok {
		return false
	}
	_, found := targets[to]
	return found
}

func depthHistogram(pages []pagemap.Page) []DepthBucket {
	counts := make(map[int]int)
	var walk func(nodes []pagemap.Node, depth int)
	walk = func(nodes []pagemap.Node, depth int) {
		for i := range nodes {
			counts[depth]++
			walk(nodes[i].Children, depth+1)
		}
	}
	walk(pagemap.BuildTree(pages), 0)

	buckets := make([]DepthBucket, 0, len(counts))
	for depth, total := range counts {
		buckets = append(buckets, DepthBucket{Depth: depth, Pages: total})
	}
	slices.SortFunc(buckets, func(a, b DepthBucket) int { return cmp.Compare(a.Depth, b.Depth) })
	return buckets
}

func topEntities(entities []graph.Entity, index pagemap.Index) []EntityScore {
	scored := make([]EntityScore, 0, len(entities))
	for i := range entities {
		score := EntityScore{EntityID: entities[i].ID, Name: entities[i].Name, Score: entities[i].Score}
		if page, ok := canonical(entities[i], index); ok {
			score.Path = page.Path
		}
		scored = append(scored, score)
	}

	slices.SortFunc(scored, func(a, b EntityScore) int {
		if a.Score != b.Score {
			return cmp.Compare(b.Score, a.Score)
		}
		return strings.Compare(a.Name, b.Name)
	})
	return scored[:min(TopEntities, len(scored))]
}
