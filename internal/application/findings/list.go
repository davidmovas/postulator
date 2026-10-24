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
	return s.shown(ctx, held)
}

func (s *Service) shown(ctx context.Context, f finding.Finding) (Finding, error) {
	places, err := s.deps.Places.Map(ctx, f.SiteID)
	if err != nil {
		return Finding{}, err
	}
	return placed(f, places), nil
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

type scope struct {
	places contextpack.Map
	typeID string
	under  map[string]struct{}
}

func (s *Service) scoped(ctx context.Context, siteID, status, pageTypeID, under string) ([]finding.Finding, scope, error) {
	if siteID == "" {
		return nil, scope{}, invalid("name the site whose findings to read", "siteId")
	}
	wanted := finding.Status(status)
	if wanted != "" && !wanted.Valid() {
		return nil, scope{}, invalid("finding status is not recognized", "status")
	}
	places, err := s.deps.Places.Map(ctx, siteID)
	if err != nil {
		return nil, scope{}, err
	}
	in := scope{places: places, typeID: pageTypeID}
	if under != "" {
		ids := places.Under(under)
		if len(ids) == 0 {
			return nil, scope{}, errors.New(errors.NotFound, "the site's map holds no such page").WithDetail("under", under)
		}
		in.under = make(map[string]struct{}, len(ids))
		for _, id := range ids {
			in.under[id] = struct{}{}
		}
	}
	held, err := s.deps.Findings.ListBySite(ctx, siteID, wanted)
	if err != nil {
		return nil, scope{}, err
	}
	return slices.DeleteFunc(held, func(f finding.Finding) bool { return !in.holds(f) }), in, nil
}

func (in scope) holds(f finding.Finding) bool {
	if in.under == nil && in.typeID == "" {
		return true
	}
	if f.PageID == nil {
		return false
	}
	if _, inside := in.under[*f.PageID]; in.under != nil && !inside {
		return false
	}
	if in.typeID == "" {
		return true
	}
	place, found := in.places.Place(*f.PageID)
	return found && place.TypeID == in.typeID
}

func (s *Service) List(ctx context.Context, req ListRequest) (paging.List[Finding], error) {
	code := finding.Code(req.Code)
	if code != "" && !code.Valid() {
		return paging.List[Finding]{}, invalid("finding code is not recognized", "code")
	}
	held, in, err := s.scoped(ctx, req.SiteID, req.Status, req.PageTypeID, req.Under)
	if err != nil {
		return paging.List[Finding]{}, err
	}
	rows := make([]Finding, 0, len(held))
	for i := range held {
		switch {
		case code != "" && held[i].Code != code:
		case req.PageID != "" && (held[i].PageID == nil || *held[i].PageID != req.PageID):
		default:
			rows = append(rows, placed(held[i], in.places))
		}
	}
	keyset := paging.Keyset[Finding]{
		IDColumn: "id",
		ID:       func(f Finding) string { return f.ID },
		Keys: []paging.SortKey[Finding]{
			paging.TimeKey("createdAt", "created_at", func(f Finding) any { return f.CreatedAt.Std() }),
		},
	}
	return keyset.Page(rows, paging.Request{After: paging.Cursor(req.Cursor), Limit: dto.PageSize(req.Limit)})
}

func (s *Service) Counts(ctx context.Context, req CountsRequest) (CountsResponse, error) {
	held, _, err := s.scoped(ctx, req.SiteID, req.Status, req.PageTypeID, req.Under)
	if err != nil {
		return CountsResponse{}, err
	}
	counted := make(map[finding.Code]int)
	for i := range held {
		counted[held[i].Code]++
	}
	out := CountsResponse{Codes: make([]CodeCount, 0, len(counted)), Total: len(held)}
	for code, count := range counted {
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
