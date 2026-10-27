package findings

import (
	"context"
	"encoding/json"
	"slices"

	"github.com/davidmovas/postulator/internal/application/graph"
	"github.com/davidmovas/postulator/internal/application/imports"
	"github.com/davidmovas/postulator/internal/application/runs"
	syncsite "github.com/davidmovas/postulator/internal/application/sync"
	"github.com/davidmovas/postulator/internal/domain/contextpack"
	"github.com/davidmovas/postulator/internal/domain/field"
	"github.com/davidmovas/postulator/internal/domain/finding"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/pagetype"
	"github.com/davidmovas/postulator/internal/domain/run"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

type siteValues struct {
	Key  string `json:"key"`
	Site string `json:"site"`
}

type choice struct {
	option       finding.Option
	pageID       string
	pageTypeID   string
	proposalHash string
	by           string
}

func chooser(by string) (string, error) {
	if by == "" {
		return finding.ResolvedByPerson, nil
	}
	if by != finding.ResolvedByPerson && by != finding.ResolvedByAgent {
		return "", invalid("a finding is resolved by a person or the agent", "by")
	}
	return by, nil
}

func (s *Service) Resolve(ctx context.Context, req ResolveRequest) (ResolveResponse, error) {
	by, err := chooser(req.By)
	if err != nil {
		return ResolveResponse{}, err
	}
	option := finding.Option(req.Option)
	if option.Kind() == "" {
		return ResolveResponse{}, invalid("the option is not recognized", "option")
	}
	held, err := s.open(ctx, req.SiteID, req.FindingID, option)
	if err != nil {
		return ResolveResponse{}, err
	}
	places, err := s.placesOf(ctx, held.SiteID, held)
	if err != nil {
		return ResolveResponse{}, err
	}
	if option.Kind() == finding.Run {
		answer, prepareErr := prepared(held, option)
		if prepareErr != nil {
			return ResolveResponse{}, prepareErr
		}
		answer.Finding = placed(held, places)
		return answer, nil
	}
	if option.Kind() == finding.Proposal {
		proposals, proposeErr := proposed([]finding.Finding{held})
		if proposeErr != nil {
			return ResolveResponse{}, proposeErr
		}
		return ResolveResponse{Finding: placed(held, places), Proposals: proposals}, nil
	}

	if option == finding.AdoptTree {
		if adoptErr := s.adoptTree(ctx, held); adoptErr != nil {
			return ResolveResponse{}, adoptErr
		}
		return s.settled(ctx, held, places)
	}

	chosen := choice{option: option, pageID: req.PageID, pageTypeID: req.PageTypeID, proposalHash: req.ProposalHash, by: by}
	if refused := s.applicable(ctx, held.SiteID, chosen); refused != nil {
		return ResolveResponse{}, refused
	}
	err = s.deps.UnitOfWork.Do(ctx, func(c context.Context) error {
		current, openErr := s.open(c, held.SiteID, held.ID, option)
		if openErr != nil {
			return openErr
		}
		return s.change(c, current, chosen)
	})
	if err != nil {
		return ResolveResponse{}, err
	}
	return s.settled(ctx, held, places)
}

func (s *Service) settled(ctx context.Context, held finding.Finding, places map[string]contextpack.Place) (ResolveResponse, error) {
	settled, err := s.deps.Findings.Get(ctx, held.ID)
	if err != nil {
		return ResolveResponse{}, err
	}
	if err := s.changed(held.SiteID); err != nil {
		return ResolveResponse{}, err
	}
	return ResolveResponse{Finding: placed(settled, places)}, nil
}

func (s *Service) adoptTree(ctx context.Context, held finding.Finding) error {
	if held.EntityID == nil {
		return invalid("the finding names no entity", "findingId")
	}
	req := graph.AdoptParentsRequest{SiteID: held.SiteID, EntityIDs: []string{*held.EntityID}}
	preview, err := s.deps.Parents.AdoptParents(ctx, req)
	if err != nil {
		return err
	}
	if preview.Adopting == 0 {
		why := "the graph already agrees with the page tree; the next sync closes the finding"
		if len(preview.Steps.Items) > 0 && preview.Steps.Items[0].Skip != "" {
			why = "the graph cannot take the page tree's parent here: " + skipped(preview.Steps.Items[0].Skip)
		}
		return errors.New(errors.Conflict, why).WithDetail("findingId", held.ID)
	}
	req.Apply, req.Hash = true, preview.Hash
	_, err = s.deps.Parents.AdoptParents(ctx, req)
	return err
}

func skipped(reason string) string {
	if reason == string(contextpack.AdoptionLoop) {
		return "the parent sits under the entity in the graph, so it would make a loop"
	}
	return "an entity of the same name already sits under that parent"
}

func (s *Service) applicable(ctx context.Context, siteID string, chosen choice) error {
	if chosen.option != finding.SetType {
		return nil
	}
	if chosen.pageTypeID == "" {
		return invalid("name the page type to set", "pageTypeId")
	}
	_, err := s.typeOf(ctx, siteID, chosen.pageTypeID)
	return err
}

func (s *Service) typeOf(ctx context.Context, siteID, typeID string) (pagetype.Type, error) {
	kind, found, err := s.deps.Places.PageType(ctx, siteID, typeID)
	if err != nil {
		return pagetype.Type{}, err
	}
	if !found {
		return pagetype.Type{}, errors.New(errors.NotFound, "the site has no such page type").WithDetail("pageTypeId", typeID)
	}
	return kind, nil
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

func (s *Service) change(ctx context.Context, held finding.Finding, chosen choice) error {
	switch chosen.option {
	case finding.Ignore:
		return s.deps.Findings.Settle(ctx, held.ID, finding.Dismissed, string(chosen.option), chosen.by, s.now())
	case finding.KeepGraph:
		return s.deps.Findings.Settle(ctx, held.ID, finding.Resolved, string(chosen.option), chosen.by, s.now())
	case finding.Accept:
		_, err := s.deps.Cards.AcceptCard(ctx, graph.AcceptCardRequest{FindingID: held.ID, ProposalHash: chosen.proposalHash, By: chosen.by})
		return err
	case finding.MapToPage:
		if chosen.pageID == "" {
			return invalid("name the page the row maps to", "pageId")
		}
		_, err := s.deps.Rows.MapRow(ctx, imports.MapRowRequest{FindingID: held.ID, PageID: chosen.pageID, By: chosen.by})
		return err
	}
	if held.PageID == nil {
		return invalid("the finding names no page", "findingId")
	}
	pageID := *held.PageID
	var err error
	switch chosen.option {
	case finding.AcceptSite:
		err = s.acceptSite(ctx, held, pageID)
	case finding.AcceptWorkbook:
		err = s.deps.Pages.SetParentSource(ctx, pageID, pagemap.ParentFromWorkbook, s.now())
	case finding.SetType:
		err = s.deps.Pages.SetType(ctx, pageID, chosen.pageTypeID, typedBy(chosen.by), s.now())
	default:
		err = invalid("the option is not recognized", "option")
	}
	if err != nil {
		return err
	}
	return s.deps.Findings.Settle(ctx, held.ID, finding.Resolved, string(chosen.option), chosen.by, s.now())
}

func typedBy(by string) pagemap.TypeReason {
	if by == finding.ResolvedByAgent {
		return pagemap.TypeByAI
	}
	return pagemap.TypeByUser
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
		return s.deps.Pages.SetParentSource(ctx, pageID, pagemap.ParentFromSite, s.now())
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
	case finding.PageTitleDiffers, finding.PageH1Differs:
		return s.planSite(ctx, pageID, held.Code, values.Site)
	default:
		return invalid("the finding does not offer accept_site", "option")
	}
}

func (s *Service) planSite(ctx context.Context, pageID string, code finding.Code, value string) error {
	page, err := s.deps.Pages.Get(ctx, pageID)
	if err != nil {
		return err
	}
	if code == finding.PageTitleDiffers {
		page.Title = value
	} else {
		page.H1 = value
	}
	page.UpdatedAt = s.now()
	return s.deps.Pages.Update(ctx, page)
}

func proposed(held []finding.Finding) (*graph.PreviewFromPagesRequest, error) {
	pageIDs := make([]string, 0, len(held))
	for i := range held {
		if held[i].PageID == nil {
			return nil, invalid("the finding names no page", "findingIds")
		}
		pageIDs = append(pageIDs, *held[i].PageID)
	}
	return &graph.PreviewFromPagesRequest{SiteID: held[0].SiteID, PageIDs: pageIDs, Source: graph.SourceSite}, nil
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
		key, err := rewritten(held)
		if err != nil {
			return ResolveResponse{}, err
		}
		shown.Start = &runs.StartRequest{
			SiteID: held.SiteID, PageIDs: []string{pageID}, Fields: []string{string(key)}, Rewrite: []string{pageID},
		}
	default:
		return ResolveResponse{}, invalid("the option is not recognized", "option")
	}
	return shown, nil
}
