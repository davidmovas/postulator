package sitemodel

import (
	"context"

	"github.com/davidmovas/postulator/internal/domain/contextpack"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/pagetype"
)

func (s *Service) Scope(ctx context.Context, siteID, pageID string) (string, error) {
	var trail string
	err := s.onMap(ctx, siteID, func(c context.Context) error {
		row, err := s.nodeRow(c, siteID, pageID, "under")
		trail = row.Trail
		return err
	})
	return trail, err
}

func (s *Service) Places(ctx context.Context, siteID string, pageIDs []string) (map[string]contextpack.Place, error) {
	out := make(map[string]contextpack.Place, len(pageIDs))
	if len(pageIDs) == 0 {
		return out, nil
	}
	err := s.onMap(ctx, siteID, func(c context.Context) error {
		rows, err := s.deps.Maps.ListByIDs(c, pageIDs)
		if err != nil {
			return err
		}
		pages, err := s.deps.Pages.ListByIDs(c, pageIDs)
		if err != nil {
			return err
		}
		read := mapRead{rows: make(map[string]contextpack.MapRow, len(rows)), pages: make(map[string]pagemap.Page, len(pages))}
		if read.types, err = s.deps.Types.Set(c, siteID); err != nil {
			return err
		}
		for i := range pages {
			read.pages[pages[i].ID] = pages[i]
		}
		for i := range rows {
			read.rows[rows[i].PageID] = rows[i]
		}
		for i := range rows {
			if rows[i].SiteID == siteID {
				out[rows[i].PageID] = contextpack.Place{Ref: read.ref(rows[i].PageID), TypeID: rows[i].TypeID}
			}
		}
		return nil
	})
	return out, err
}

func (s *Service) PageType(ctx context.Context, siteID, typeID string) (pagetype.Type, bool, error) {
	var (
		kind  pagetype.Type
		found bool
	)
	err := s.deps.Reader.Read(ctx, func(c context.Context) error {
		set, err := s.deps.Types.Set(c, siteID)
		if err != nil {
			return err
		}
		kind, found = set.ByID(typeID)
		return nil
	})
	return kind, found, err
}
