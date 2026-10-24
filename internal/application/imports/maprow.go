package imports

import (
	"context"
	"encoding/json"
	"slices"
	"strings"

	"github.com/davidmovas/postulator/internal/application/events"
	"github.com/davidmovas/postulator/internal/domain/finding"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

const ResolutionMapped = "mapped"

type MapRowRequest struct {
	FindingID string `json:"findingId"`
	PageID    string `json:"pageId"`
	By        string `json:"by"`
}

type MapRowResponse struct {
	FindingID string `json:"findingId"`
	PageID    string `json:"pageId"`
}

type rowSubject struct {
	Product bool `json:"product"`
}

func (r MapRowRequest) validate() error {
	switch {
	case strings.TrimSpace(r.FindingID) == "":
		return errors.New(errors.Invalid, "name the row's finding").WithDetail("field", "findingId")
	case strings.TrimSpace(r.PageID) == "":
		return errors.New(errors.Invalid, "name the page the row means").WithDetail("field", "pageId")
	case r.By != finding.ResolvedByPerson && r.By != finding.ResolvedByAgent:
		return errors.New(errors.Invalid, "a row is mapped by a person or the agent").WithDetail("field", "by").WithDetail("by", r.By)
	}
	return nil
}

func (s *Service) MapRow(ctx context.Context, req MapRowRequest) (MapRowResponse, error) {
	if err := req.validate(); err != nil {
		return MapRowResponse{}, err
	}
	siteID := ""
	err := s.deps.UnitOfWork.Do(ctx, func(c context.Context) error {
		held, chosen, err := s.mapTarget(c, req)
		if err != nil {
			return err
		}
		siteID = held.SiteID
		return s.mapOnto(c, held, chosen, req.By)
	})
	if err != nil {
		return MapRowResponse{}, err
	}
	if published := s.deps.Publisher.Publish(events.PagesChanged, events.PagesChangedPayload{SiteID: siteID}); published != nil {
		return MapRowResponse{}, published
	}
	return MapRowResponse{FindingID: req.FindingID, PageID: req.PageID}, nil
}

func (s *Service) mapTarget(ctx context.Context, req MapRowRequest) (finding.Finding, pagemap.Page, error) {
	held, err := s.openRow(ctx, req.FindingID)
	if err != nil {
		return finding.Finding{}, pagemap.Page{}, err
	}
	chosen, err := s.deps.Pages.Get(ctx, req.PageID)
	if err != nil {
		return finding.Finding{}, pagemap.Page{}, err
	}
	return held, chosen, mappable(held, chosen)
}

func (s *Service) mapOnto(ctx context.Context, held finding.Finding, chosen pagemap.Page, by string) error {
	replaced, err := s.replaced(ctx, held, chosen.ID)
	if err != nil {
		return err
	}
	now := s.now()
	if keyErr := s.deps.Pages.SetRowKey(ctx, held.SiteID, chosen.ID, held.SubjectKey, replaced.PlannedTrail, now); keyErr != nil {
		return keyErr
	}
	if moveErr := s.deps.Findings.MoveToPage(ctx, held.ID, chosen.ID, now); moveErr != nil {
		return moveErr
	}
	if settleErr := s.deps.Findings.Settle(ctx, held.ID, finding.Resolved, ResolutionMapped, by, now); settleErr != nil {
		return settleErr
	}
	return s.dropReplaced(ctx, replaced)
}

func (s *Service) openRow(ctx context.Context, findingID string) (finding.Finding, error) {
	held, err := s.deps.Findings.Get(ctx, findingID)
	if err != nil {
		return finding.Finding{}, err
	}
	if !slices.Contains(rowCodes, held.Code) {
		return finding.Finding{}, errors.New(errors.Invalid, "the finding names no workbook row").
			WithDetail("findingId", findingID).WithDetail("code", string(held.Code))
	}
	if held.Status != finding.Open {
		return finding.Finding{}, errors.New(errors.Conflict, "the row's finding is already settled").WithDetail("findingId", findingID)
	}
	return held, nil
}

func mappable(held finding.Finding, chosen pagemap.Page) error {
	var subject rowSubject
	if err := json.Unmarshal(held.Subject, &subject); err != nil {
		return errors.Wrap(err, errors.Internal, "read the row's finding")
	}
	refuse := func(message string) error {
		return errors.New(errors.Invalid, message).WithDetail("field", "pageId").WithDetail("pageId", chosen.ID)
	}
	switch {
	case chosen.SiteID != held.SiteID:
		return refuse("the page belongs to another site")
	case chosen.Status == pagemap.StatusArchived:
		return refuse("the page is archived, so the site no longer holds it")
	case subject.Product && chosen.WPType != pagemap.WPProduct:
		return refuse("the row names a product, so it maps to a product of the store")
	}
	return nil
}

func (s *Service) replaced(ctx context.Context, held finding.Finding, chosenID string) (pagemap.Page, error) {
	if held.Code != finding.RowWithoutPage || held.PageID == nil || *held.PageID == chosenID {
		return pagemap.Page{}, nil
	}
	return s.deps.Pages.Get(ctx, *held.PageID)
}

func (s *Service) dropReplaced(ctx context.Context, planned pagemap.Page) error {
	if planned.ID == "" || planned.WPID != nil || planned.Status != pagemap.StatusPlanned {
		return nil
	}
	named, err := s.deps.Items.ByTarget(ctx, planned.ID, 1)
	if err != nil || len(named) > 0 {
		return err
	}
	return s.deps.Pages.Delete(ctx, planned.ID)
}
