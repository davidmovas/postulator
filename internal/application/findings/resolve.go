package findings

import (
	"context"
	"encoding/json"
	"slices"

	"github.com/davidmovas/postulator/internal/application/graph"
	"github.com/davidmovas/postulator/internal/application/imports"
	"github.com/davidmovas/postulator/internal/application/pages"
	"github.com/davidmovas/postulator/internal/application/runs"
	syncsite "github.com/davidmovas/postulator/internal/application/sync"
	"github.com/davidmovas/postulator/internal/domain/field"
	"github.com/davidmovas/postulator/internal/domain/finding"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/run"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

type siteValues struct {
	Key  string `json:"key"`
	Site string `json:"site"`
}

func (s *Service) Resolve(ctx context.Context, req ResolveRequest) (ResolveResponse, error) {
	by := req.By
	if by == "" {
		by = finding.ResolvedByPerson
	}
	if by != finding.ResolvedByPerson && by != finding.ResolvedByAgent {
		return ResolveResponse{}, invalid("a finding is resolved by a person or the agent", "by")
	}
	option := finding.Option(req.Option)
	if option.Kind() == "" {
		return ResolveResponse{}, invalid("the option is not recognized", "option")
	}
	if option.Kind() == finding.Run {
		held, err := s.open(ctx, req.SiteID, req.FindingID, option)
		if err != nil {
			return ResolveResponse{}, err
		}
		answer, err := prepared(held, option)
		if err != nil {
			return ResolveResponse{}, err
		}
		answer.Finding, err = s.shown(ctx, held)
		return answer, err
	}

	var held finding.Finding
	err := s.deps.UnitOfWork.Do(ctx, func(c context.Context) error {
		var openErr error
		if held, openErr = s.open(c, req.SiteID, req.FindingID, option); openErr != nil {
			return openErr
		}
		return s.change(c, held, option, req, by)
	})
	if err != nil {
		return ResolveResponse{}, err
	}
	settled, err := s.deps.Findings.Get(ctx, held.ID)
	if err != nil {
		return ResolveResponse{}, err
	}
	if err := s.changed(held.SiteID); err != nil {
		return ResolveResponse{}, err
	}
	shown, err := s.shown(ctx, settled)
	return ResolveResponse{Finding: shown}, err
}

func (s *Service) open(ctx context.Context, siteID, findingID string, option finding.Option) (finding.Finding, error) {
	held, err := s.held(ctx, siteID, findingID)
	if err != nil {
		return finding.Finding{}, err
	}
	if held.Status != finding.Open {
		return finding.Finding{}, errors.New(errors.Conflict, "the finding is no longer open").WithDetail("findingId", held.ID)
	}
	if !slices.Contains(held.Options, option) {
		return finding.Finding{}, invalid("the finding does not offer "+string(option), "option")
	}
	return held, nil
}

func (s *Service) change(ctx context.Context, held finding.Finding, option finding.Option, req ResolveRequest, by string) error {
	switch option {
	case finding.Ignore:
		return s.deps.Findings.Settle(ctx, held.ID, finding.Dismissed, string(option), by, s.now())
	case finding.Accept:
		_, err := s.deps.Cards.AcceptCard(ctx, graph.AcceptCardRequest{FindingID: held.ID, ProposalHash: req.ProposalHash, By: by})
		return err
	case finding.MapToPage:
		if req.PageID == "" {
			return invalid("name the page the row maps to", "pageId")
		}
		_, err := s.deps.Rows.MapRow(ctx, imports.MapRowRequest{FindingID: held.ID, PageID: req.PageID, By: by})
		return err
	}
	if held.PageID == nil {
		return invalid("the finding names no page", "findingId")
	}
	pageID := *held.PageID
	var err error
	switch option {
	case finding.AcceptSite:
		err = s.acceptSite(ctx, held, pageID)
	case finding.AcceptWorkbook:
		err = s.deps.Parents.SetParentSource(ctx, pageID, pagemap.ParentFromWorkbook, s.now())
	case finding.SetType:
		err = s.setType(ctx, pageID, req.PageTypeID, by)
	default:
		err = invalid("the option is not recognized", "option")
	}
	if err != nil {
		return err
	}
	return s.deps.Findings.Settle(ctx, held.ID, finding.Resolved, string(option), by, s.now())
}

func valuesOf(held finding.Finding) (siteValues, error) {
	var values siteValues
	if err := json.Unmarshal(held.Subject, &values); err != nil {
		return siteValues{}, invalid("the finding's subject cannot be read", "findingId")
	}
	return values, nil
}

func (s *Service) acceptSite(ctx context.Context, held finding.Finding, pageID string) error {
	switch held.Code {
	case finding.BodyChangedOnSite:
		return s.deps.Bodies.AcceptSite(ctx, pageID, s.now())
	case finding.ParentDiffers:
		return s.deps.Parents.SetParentSource(ctx, pageID, pagemap.ParentFromSite, s.now())
	}
	values, err := valuesOf(held)
	if err != nil {
		return err
	}
	switch held.Code {
	case finding.FieldChangedOnSite:
		if values.Key == "" {
			return invalid("the finding names no field", "findingId")
		}
		return s.deps.Fields.AcceptSite(ctx, pageID, field.Key(values.Key), s.now())
	case finding.PageTitleDiffers:
		_, err = s.deps.Pages.Update(ctx, pages.UpdateRequest{ID: pageID, Title: &values.Site})
	case finding.PageH1Differs:
		_, err = s.deps.Pages.Update(ctx, pages.UpdateRequest{ID: pageID, H1: &values.Site})
	default:
		err = invalid("the finding does not offer accept_site", "option")
	}
	return err
}

func (s *Service) setType(ctx context.Context, pageID, pageTypeID, by string) error {
	if pageTypeID == "" {
		return invalid("name the page type to set", "pageTypeId")
	}
	reason := pagemap.TypeByUser
	if by == finding.ResolvedByAgent {
		reason = pagemap.TypeByAI
	}
	_, err := s.deps.Pages.SetType(ctx, pages.SetTypeRequest{PageID: pageID, PageTypeID: pageTypeID, By: reason})
	return err
}

func prepared(held finding.Finding, option finding.Option) (ResolveResponse, error) {
	var shown ResolveResponse
	if option == finding.Rescan {
		shown.Sync = &syncsite.SyncSiteRequest{SiteID: held.SiteID, Full: true, RenderRescan: true}
		return shown, nil
	}
	if held.PageID == nil {
		return ResolveResponse{}, invalid("the finding names no page", "findingId")
	}
	pageID := *held.PageID
	switch option {
	case finding.RelinkPage:
		shown.Start = &runs.StartRequest{SiteID: held.SiteID, PageIDs: []string{pageID}, Kind: string(run.KindRelink)}
	case finding.RewriteField:
		key := field.Title
		if held.Code == finding.FieldChangedOnSite {
			values, err := valuesOf(held)
			if err != nil || values.Key == "" {
				return ResolveResponse{}, invalid("the finding names no field", "findingId")
			}
			key = field.Key(values.Key)
		}
		shown.Start = &runs.StartRequest{
			SiteID: held.SiteID, PageIDs: []string{pageID}, Fields: []string{string(key)}, Rewrite: []string{pageID},
		}
	default:
		return ResolveResponse{}, invalid("the option is not recognized", "option")
	}
	return shown, nil
}
