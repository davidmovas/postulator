package findings

import (
	"cmp"
	"context"
	"slices"

	"github.com/davidmovas/postulator/internal/domain/contextpack"
	"github.com/davidmovas/postulator/internal/domain/finding"
	"github.com/davidmovas/postulator/internal/kernel/dto"
	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/kernel/paging"
)

func invalid(message, field string) error {
	return errors.New(errors.Invalid, message).WithDetail("field", field)
}

func (s *Service) Get(ctx context.Context, req GetRequest) (Finding, error) {
	held, err := s.held(ctx, req.SiteID, req.FindingID)
	if err != nil {
		return Finding{}, err
	}
	places, err := s.placesOf(ctx, held.SiteID, held)
	if err != nil {
		return Finding{}, err
	}
	return placed(held, places), nil
}

func (s *Service) placesOf(ctx context.Context, siteID string, held ...finding.Finding) (map[string]contextpack.Place, error) {
	ids := make([]string, 0, len(held))
	for i := range held {
		if held[i].PageID != nil && !slices.Contains(ids, *held[i].PageID) {
			ids = append(ids, *held[i].PageID)
		}
	}
	return s.deps.Places.Places(ctx, siteID, ids)
}

func (s *Service) held(ctx context.Context, siteID, findingID string) (finding.Finding, error) {
	if findingID == "" {
		return finding.Finding{}, invalid("name the finding", "findingId")
	}
	held, err := s.deps.Findings.Get(ctx, findingID)
	if err != nil {
		return finding.Finding{}, err
	}
	if siteID != "" && held.SiteID != siteID {
		return finding.Finding{}, errors.New(errors.NotFound, "the site holds no such finding").WithDetail("findingId", findingID)
	}
	return held, nil
}

func (s *Service) query(ctx context.Context, siteID, status, pageTypeID, under string) (finding.Query, error) {
	if siteID == "" {
		return finding.Query{}, invalid("name the site whose findings to read", "siteId")
	}
	q := finding.Query{SiteID: siteID, Status: finding.Status(status), PageTypeID: pageTypeID}
	if q.Status != "" && !q.Status.Valid() {
		return finding.Query{}, invalid("finding status is not recognized", "status")
	}
	switch {
	case under != "":
		trail, err := s.deps.Places.Scope(ctx, siteID, under)
		if err != nil {
			return finding.Query{}, err
		}
		q.Under = trail
	case pageTypeID != "":
		if err := s.deps.Places.Refresh(ctx, siteID); err != nil {
			return finding.Query{}, err
		}
	}
	return q, nil
}

func (s *Service) List(ctx context.Context, req ListRequest) (paging.List[Finding], error) {
	code := finding.Code(req.Code)
	if code != "" && !code.Valid() {
		return paging.List[Finding]{}, invalid("finding code is not recognized", "code")
	}
	q, err := s.query(ctx, req.SiteID, req.Status, req.PageTypeID, req.Under)
	if err != nil {
		return paging.List[Finding]{}, err
	}
	q.Code, q.PageID = code, req.PageID
	held, err := s.deps.Findings.List(ctx, q, paging.Request{After: paging.Cursor(req.Cursor), Limit: dto.PageSize(req.Limit)})
	if err != nil {
		return paging.List[Finding]{}, err
	}
	places, err := s.placesOf(ctx, req.SiteID, held.Items...)
	if err != nil {
		return paging.List[Finding]{}, err
	}
	rows := make([]Finding, 0, len(held.Items))
	for i := range held.Items {
		rows = append(rows, placed(held.Items[i], places))
	}
	return paging.List[Finding]{Cursors: held.Cursors, Items: rows, HasMore: held.HasMore}, nil
}

func (s *Service) Counts(ctx context.Context, req CountsRequest) (CountsResponse, error) {
	q, err := s.query(ctx, req.SiteID, req.Status, req.PageTypeID, req.Under)
	if err != nil {
		return CountsResponse{}, err
	}
	counted, err := s.deps.Findings.CountByCode(ctx, q)
	if err != nil {
		return CountsResponse{}, err
	}
	out := CountsResponse{Codes: make([]CodeCount, 0, len(counted))}
	for code, count := range counted {
		out.Total += count
		out.Codes = append(out.Codes, CodeCount{Code: string(code), Severity: string(finding.SeverityOf(code)), Count: count})
	}
	slices.SortFunc(out.Codes, func(a, b CodeCount) int {
		return cmp.Or(cmp.Compare(rank(b.Severity), rank(a.Severity)), cmp.Compare(b.Count, a.Count), cmp.Compare(a.Code, b.Code))
	})
	return out, nil
}

func rank(severity string) int {
	switch finding.Severity(severity) {
	case finding.Error:
		return 2
	case finding.Warning:
		return 1
	default:
		return 0
	}
}
