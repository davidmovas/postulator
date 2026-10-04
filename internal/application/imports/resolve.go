package imports

import (
	"context"

	"github.com/davidmovas/postulator/internal/domain/importmap"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

func (s *Service) resolve(ctx context.Context, siteID string, given importmap.Mapping) (importmap.Mapping, error) {
	if given.ID == "" || !given.Unmapped() {
		return given, nil
	}
	saved, err := s.deps.Mappings.Get(ctx, given.ID)
	if err != nil {
		return importmap.Mapping{}, err
	}
	if saved.SiteID != siteID {
		return importmap.Mapping{}, errors.New(errors.NotFound, "the saved mapping belongs to another site").
			WithDetail("mappingId", given.ID)
	}
	if given.Options.RowType != "" {
		saved.Options.RowType = given.Options.RowType
	}
	if len(given.Options.Sheets) > 0 {
		saved.Options.Sheets = given.Options.Sheets
	}
	return saved, nil
}

func detect(given importmap.Mapping, headers []string) importmap.Mapping {
	if !given.Unmapped() {
		return given
	}
	found := importmap.AutoDetect(headers)
	given.Columns = found.Columns
	given.Options.LevelColumns = found.Options.LevelColumns
	if len(given.Options.NoteColumns) == 0 {
		given.Options.NoteColumns = found.Options.NoteColumns
	}
	return given
}

func (s *Service) read(ctx context.Context, siteID, path string, given importmap.Mapping) (importmap.Mapping, importmap.Table, error) {
	mapping, err := s.resolve(ctx, siteID, given)
	if err != nil {
		return importmap.Mapping{}, importmap.Table{}, err
	}
	table, err := s.table(ctx, path, mapping.Options)
	if err != nil {
		return importmap.Mapping{}, importmap.Table{}, err
	}
	return detect(mapping, table.Headers), table, nil
}
