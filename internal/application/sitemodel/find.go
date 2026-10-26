package sitemodel

import (
	"cmp"
	"context"
	"slices"
	"strings"

	"github.com/davidmovas/postulator/internal/domain/contextpack"
	"github.com/davidmovas/postulator/internal/domain/graph"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/kernel/paging"
)

const (
	findDefaultLimit = 50
	findMaxLimit     = 200
)

type FindRequest struct {
	SiteID     string          `json:"siteId"`
	PageIDs    []string        `json:"pageIds,omitempty"`
	Query      string          `json:"query,omitempty"`
	PageTypeID string          `json:"pageTypeId,omitempty"`
	Under      string          `json:"under,omitempty"`
	EntityID   string          `json:"entityId,omitempty"`
	Status     pagemap.Status  `json:"status,omitempty"`
	Gap        contextpack.Gap `json:"gap,omitempty"`
	Cursor     paging.Cursor   `json:"cursor,omitempty"`
	Limit      int             `json:"limit,omitempty"`
}

func (s *Service) Find(ctx context.Context, req FindRequest) (paging.List[contextpack.MapNode], error) {
	switch {
	case req.SiteID == "":
		return paging.List[contextpack.MapNode]{}, errors.New(errors.Invalid, "name the site whose pages to find").WithDetail("field", "siteId")
	case req.Gap != "" && !slices.Contains(contextpack.Gaps(), req.Gap):
		return paging.List[contextpack.MapNode]{}, errors.New(errors.Invalid,
			"the gap must be one of planned, empty, unmapped, orphan, missing_links or findings").
			WithDetail("field", "gap").WithDetail("value", string(req.Gap))
	case req.Status != "" && !req.Status.Valid():
		return paging.List[contextpack.MapNode]{}, errors.New(errors.Invalid,
			"the status must be one of planned, exists, published or archived").
			WithDetail("field", "status").WithDetail("value", string(req.Status))
	}
	var listed paging.List[contextpack.MapNode]
	err := s.onMap(ctx, req.SiteID, func(c context.Context) error {
		kept, err := s.found(c, req)
		if err != nil {
			return err
		}
		slices.SortFunc(kept, func(a, b contextpack.MapRow) int {
			return cmp.Or(cmp.Compare(a.Path, b.Path), cmp.Compare(a.PageID, b.PageID))
		})
		keyset := paging.Keyset[contextpack.MapRow]{
			IDColumn: "page_id",
			ID:       func(row contextpack.MapRow) string { return row.PageID },
			Keys: []paging.SortKey[contextpack.MapRow]{
				paging.TextKey("path", "path", func(row contextpack.MapRow) any { return row.Path }),
			},
		}
		page, err := keyset.Page(kept, paging.Request{After: req.Cursor, Limit: clamped(req.Limit, findDefaultLimit, findMaxLimit)})
		if err != nil {
			return err
		}
		read, err := s.mapNodes(c, req.SiteID, page.Items, nil, true)
		if err != nil {
			return err
		}
		listed = paging.List[contextpack.MapNode]{Cursors: page.Cursors, HasMore: page.HasMore, Items: read.nodes}
		if whyErr := s.whys(c, read, listed.Items, nil); whyErr != nil {
			return whyErr
		}
		return s.bodiesOf(c, listed.Items)
	})
	return listed, err
}

func (s *Service) found(ctx context.Context, req FindRequest) ([]contextpack.MapRow, error) {
	rows, err := s.deps.Maps.ListBySite(ctx, req.SiteID)
	if err != nil {
		return nil, err
	}
	snapshot, err := s.base(ctx, req.SiteID)
	if err != nil {
		return nil, err
	}
	var gapped map[string]struct{}
	if req.Gap != "" {
		if gapped, err = s.gapped(ctx, req.SiteID, req.Gap); err != nil {
			return nil, err
		}
	}
	under, underID := "", ""
	if req.Under != "" {
		row, rowErr := s.nodeRow(ctx, req.SiteID, req.Under, "under")
		if rowErr != nil {
			return nil, rowErr
		}
		under, underID = row.Trail, row.PageID
	}
	key := graph.Key(strings.TrimSpace(req.Query))
	kept := make([]contextpack.MapRow, 0)
	for i := range rows {
		row := rows[i]
		switch {
		case !row.Row:
			continue
		case len(req.PageIDs) > 0 && !slices.Contains(req.PageIDs, row.PageID):
			continue
		case under != "" && (row.PageID == underID || !strings.HasPrefix(row.Trail, under)):
			continue
		case req.PageTypeID != "" && !row.Typed(req.PageTypeID):
			continue
		case key != "" && !row.Mentions(key) && !keywordMentions(snapshot, row.PageID, key):
			continue
		case !heldAs(snapshot, row.PageID, req.EntityID, req.Status):
			continue
		}
		if _, gap := gapped[row.PageID]; gapped != nil && !gap {
			continue
		}
		kept = append(kept, row)
	}
	return kept, nil
}

func (s *Service) gapped(ctx context.Context, siteID string, gap contextpack.Gap) (map[string]struct{}, error) {
	snapshot, audit, _, err := s.whole(ctx, siteID)
	if err != nil {
		return nil, err
	}
	out := map[string]struct{}{}
	rows := snapshot.Gaps(linkFacts(audit))
	for i := range rows {
		if slices.Contains(rows[i].Gaps, gap) {
			out[rows[i].Page.PageID] = struct{}{}
		}
	}
	return out, nil
}

func heldAs(snapshot contextpack.Snapshot, pageID, entityID string, status pagemap.Status) bool {
	if entityID == "" && status == "" {
		return true
	}
	page, found := snapshot.Index.ByID(pageID)
	switch {
	case !found:
		return false
	case entityID != "" && (page.EntityID == nil || *page.EntityID != entityID):
		return false
	default:
		return status == "" || page.Status == status
	}
}

func keywordMentions(snapshot contextpack.Snapshot, pageID, key string) bool {
	page, found := snapshot.Index.ByID(pageID)
	if !found {
		return false
	}
	for _, held := range page.Keywords {
		if strings.Contains(graph.Key(held.Text), key) {
			return true
		}
	}
	return false
}
