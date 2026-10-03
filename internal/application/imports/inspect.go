package imports

import (
	"context"

	"github.com/davidmovas/postulator/internal/domain/importmap"
)

const sampleRows = 5

func (s *Service) Inspect(ctx context.Context, req InspectRequest) (InspectResponse, error) {
	if err := s.requireSite(ctx, req.SiteID); err != nil {
		return InspectResponse{}, err
	}
	sheets, err := s.deps.Tables.Sheets(req.Path)
	if err != nil {
		return InspectResponse{}, err
	}
	state, err := s.state(ctx, req.SiteID)
	if err != nil {
		return InspectResponse{}, err
	}

	read := make(map[string]importmap.Table, len(sheets))
	views := make([]Sheet, 0, len(sheets))
	for _, info := range sheets {
		detected, detectErr := s.detectSheet(ctx, req, &state, info, read)
		if detectErr != nil {
			return InspectResponse{}, detectErr
		}
		views = append(views, sheetView(info, detected))
	}

	table, err := s.sample(ctx, req, sheets, read)
	if err != nil {
		return InspectResponse{}, err
	}
	detected := importmap.AutoDetect(table.Headers)
	detected.SiteID = req.SiteID
	if detected.Options.RowType, err = detectRowType(&state, detected, table); err != nil {
		return InspectResponse{}, err
	}

	saved, err := s.deps.Mappings.ListBySite(ctx, req.SiteID)
	if err != nil {
		return InspectResponse{}, err
	}

	sample := table.Rows
	if len(sample) > sampleRows {
		sample = sample[:sampleRows]
	}
	return InspectResponse{
		Headers:  table.Headers,
		Sample:   sample,
		Rows:     len(table.Rows),
		Sheets:   views,
		Detected: mappingView(detected),
		Saved:    mappingViews(saved),
	}, nil
}

func sheetsNamed(name string) []string {
	if name == "" {
		return nil
	}
	return []string{name}
}

func (s *Service) detectSheet(
	ctx context.Context, req InspectRequest, state *siteState, info importmap.SheetInfo, read map[string]importmap.Table,
) (importmap.Mapping, error) {
	detected := importmap.AutoDetect(info.Headers)
	detected.SiteID = req.SiteID
	detected.Options.Sheets = sheetsNamed(info.Name)
	detected.Options.RowType = importmap.RowPages
	if !state.sells() || info.Rows > s.deps.MaxRows {
		return detected, nil
	}

	table, err := s.table(ctx, req.Path, detected.Options)
	if err != nil {
		return importmap.Mapping{}, err
	}
	read[info.Name] = table
	detected.Options.RowType, err = detectRowType(state, detected, table)
	return detected, err
}

func detectRowType(state *siteState, detected importmap.Mapping, table importmap.Table) (importmap.RowType, error) {
	if !state.sells() {
		return importmap.RowPages, nil
	}
	binding, err := detected.Bind(table.Headers)
	if err != nil {
		return "", err
	}
	return state.rowTypeOf(binding, table.Rows), nil
}

func (s *Service) sample(ctx context.Context, req InspectRequest, sheets []importmap.SheetInfo, read map[string]importmap.Table) (importmap.Table, error) {
	if name, sole := soleSheet(req, sheets); sole {
		if table, held := read[name]; held {
			return table, nil
		}
	}
	return s.table(ctx, req.Path, importmap.Options{Sheets: req.Sheets, NoHeader: req.NoHeader})
}

func soleSheet(req InspectRequest, sheets []importmap.SheetInfo) (string, bool) {
	switch {
	case req.NoHeader:
		return "", false
	case len(req.Sheets) == 1:
		return req.Sheets[0], true
	case len(req.Sheets) == 0 && len(sheets) > 0:
		return sheets[0].Name, true
	default:
		return "", false
	}
}
