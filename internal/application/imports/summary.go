package imports

import (
	"context"
	"slices"
)

const (
	SummaryPages    = 20
	SummaryExamples = 5
)

type FindingGroup struct {
	Code     string    `json:"code"`
	Examples []Finding `json:"examples"`
	Count    int       `json:"count"`
	Blocking bool      `json:"blocking"`
}

type PreviewSummaryResponse struct {
	Counts   Counts         `json:"counts"`
	Sheets   []string       `json:"sheets"`
	Findings []FindingGroup `json:"findings"`
	Pages    []PreviewPage  `json:"pages"`
	Rows     int            `json:"rows"`
	Blocking bool           `json:"blocking"`
	More     bool           `json:"more"`
}

func (s *Service) PreviewSummary(ctx context.Context, req PreviewRequest) (PreviewSummaryResponse, error) {
	computed, err := s.compute(ctx, req.SiteID, req.Path, req.Mapping)
	if err != nil {
		return PreviewSummaryResponse{}, err
	}
	return summarize(computed.report, computed.counts(), req.Mapping.Options.Sheets, computed.rows), nil
}

func summarize(report PreviewReport, counts Counts, sheets []string, rows int) PreviewSummaryResponse {
	pages := report.Pages
	more := len(pages) > SummaryPages
	if more {
		pages = pages[:SummaryPages]
	}
	if sheets == nil {
		sheets = []string{}
	}

	return PreviewSummaryResponse{
		Counts:   counts,
		Sheets:   sheets,
		Findings: group(report),
		Pages:    slices.Clone(pages),
		Rows:     rows,
		Blocking: len(report.Errors) > 0,
		More:     more,
	}
}

func group(report PreviewReport) []FindingGroup {
	order := make([]string, 0, 8)
	byCode := make(map[string]*FindingGroup, 8)

	collect := func(findings []Finding, blocking bool) {
		for i := range findings {
			held, seen := byCode[findings[i].Code]
			if !seen {
				held = &FindingGroup{Code: findings[i].Code, Examples: make([]Finding, 0, SummaryExamples), Blocking: blocking}
				byCode[findings[i].Code] = held
				order = append(order, findings[i].Code)
			}
			held.Count++
			if len(held.Examples) < SummaryExamples {
				held.Examples = append(held.Examples, findings[i])
			}
		}
	}
	collect(report.Errors, true)
	collect(report.Warnings, false)

	out := make([]FindingGroup, 0, len(order))
	for _, code := range order {
		out = append(out, *byCode[code])
	}
	return out
}
