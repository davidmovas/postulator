package findings

import (
	"context"

	"github.com/davidmovas/postulator/internal/domain/finding"
	"github.com/davidmovas/postulator/internal/kernel/dto"
	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/kernel/paging"
)

func invalid(message, field string) error {
	return errors.New(errors.Invalid, message).WithDetail("field", field)
}

func (s *Service) List(ctx context.Context, req ListRequest) (paging.List[Finding], error) {
	if req.SiteID == "" {
		return paging.List[Finding]{}, invalid("name the site whose findings to list", "siteId")
	}
	status := finding.Status(req.Status)
	if status != "" && !status.Valid() {
		return paging.List[Finding]{}, invalid("finding status is not recognized", "status")
	}
	code := finding.Code(req.Code)
	if code != "" && !code.Valid() {
		return paging.List[Finding]{}, invalid("finding code is not recognized", "code")
	}
	held, err := s.deps.Findings.ListBySite(ctx, req.SiteID, status)
	if err != nil {
		return paging.List[Finding]{}, err
	}
	rows := make([]Finding, 0, len(held))
	for i := range held {
		switch {
		case code != "" && held[i].Code != code:
		case req.PageID != "" && (held[i].PageID == nil || *held[i].PageID != req.PageID):
		default:
			rows = append(rows, view(held[i]))
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
