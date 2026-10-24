package sitemodel

import (
	"context"
	"slices"

	"github.com/davidmovas/postulator/internal/domain/contextpack"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
)

func (s *Service) Refresh(ctx context.Context, siteID string) error {
	var (
		generation int64
		rows       []contextpack.MapRow
		fresh      bool
	)
	err := s.deps.Reader.Read(ctx, func(c context.Context) error {
		current, built, err := s.deps.Maps.State(c, siteID)
		if err != nil {
			return err
		}
		if fresh = built >= current; fresh {
			return nil
		}
		tree, err := s.folded(c, siteID)
		if err != nil {
			return err
		}
		generation, rows = current, tree.Rows()
		return nil
	})
	if err != nil || fresh {
		return err
	}
	return s.deps.UnitOfWork.Do(ctx, func(c context.Context) error {
		_, built, err := s.deps.Maps.State(c, siteID)
		if err != nil || built >= generation {
			return err
		}
		stored, err := s.deps.Maps.ListBySite(c, siteID)
		if err != nil {
			return err
		}
		moved, gone := movedRows(rows, stored)
		return s.deps.Maps.Write(c, siteID, moved, gone, generation)
	})
}

func (s *Service) folded(ctx context.Context, siteID string) (contextpack.Map, error) {
	pages, err := s.deps.Pages.ListBySite(ctx, siteID)
	if err != nil {
		return contextpack.Map{}, err
	}
	names, err := s.deps.Fields.ListNamesBySite(ctx, siteID)
	if err != nil {
		return contextpack.Map{}, err
	}
	types, err := s.deps.Types.Set(ctx, siteID)
	if err != nil {
		return contextpack.Map{}, err
	}
	snapshot := contextpack.Snapshot{Index: pagemap.NewIndexNamed(pages, pagemap.Names(names)), Types: types, Fields: names}
	return snapshot.Map(nil), nil
}

func movedRows(rows, stored []contextpack.MapRow) (moved []contextpack.MapRow, gone []string) {
	held := make(map[string]contextpack.MapRow, len(stored))
	for i := range stored {
		held[stored[i].PageID] = stored[i]
	}
	moved = make([]contextpack.MapRow, 0)
	for i := range rows {
		if kept, known := held[rows[i].PageID]; !known || kept != rows[i] {
			moved = append(moved, rows[i])
		}
		delete(held, rows[i].PageID)
	}
	gone = make([]string, 0, len(held))
	for pageID := range held {
		gone = append(gone, pageID)
	}
	slices.Sort(gone)
	return moved, gone
}
