package pages

import (
	"context"
	"slices"
	"strconv"
	"strings"

	"github.com/davidmovas/postulator/internal/domain/finding"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/pagetype"
	"github.com/davidmovas/postulator/internal/domain/template"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

const (
	SelectionLimit       = 10000
	bulkBatch            = 1000
	reasonSelectionMoved = "selection_moved"
	reasonSelectionLimit = "selection_limit"
	reasonRoleMismatch   = "role_mismatch"
)

func (s *Service) Resolve(ctx context.Context, sel Selection) (ResolveResponse, error) {
	ids, err := s.resolve(ctx, sel)
	return ResolveResponse{PageIDs: ids}, err
}

func (s *Service) resolve(ctx context.Context, sel Selection) ([]string, error) {
	if err := requireSite(sel.SiteID); err != nil {
		return nil, err
	}
	picked := make(map[string]bool)
	if len(sel.PageIDs) > 0 {
		held, err := s.pages.ListByIDs(ctx, sel.PageIDs)
		if err != nil {
			return nil, err
		}
		for i := range held {
			if held[i].SiteID != sel.SiteID {
				return nil, errors.New(errors.Invalid, "the page "+held[i].Path+" belongs to another site").WithDetail("field", "pageIds")
			}
			picked[held[i].ID] = true
		}
	}
	for _, subtree := range sel.Subtrees {
		trail, err := s.trailOf(ctx, sel.SiteID, subtree.PageID)
		if err != nil {
			return nil, err
		}
		under, err := s.rows.IDsUnder(ctx, sel.SiteID, trail, strings.TrimSpace(subtree.TypeID))
		if err != nil {
			return nil, err
		}
		for _, pageID := range under {
			picked[pageID] = true
		}
	}
	if sel.Filter != nil {
		q, err := s.query(ctx, sel.SiteID, *sel.Filter)
		if err != nil {
			return nil, err
		}
		matched, err := s.pages.IDs(ctx, q)
		if err != nil {
			return nil, err
		}
		if moved := seenAgain(sel.Seen, len(matched)); moved != nil {
			return nil, moved
		}
		for _, pageID := range matched {
			picked[pageID] = true
		}
	}
	if sel.RuleID != "" {
		matched, err := s.classifier.Matching(ctx, sel.SiteID, sel.RuleID)
		if err != nil {
			return nil, err
		}
		if moved := seenAgain(sel.Seen, len(matched)); moved != nil {
			return nil, moved
		}
		for _, pageID := range matched {
			picked[pageID] = true
		}
	}
	for _, pageID := range sel.Except {
		delete(picked, pageID)
	}
	for _, subtree := range sel.ExceptSubtrees {
		trail, err := s.trailOf(ctx, sel.SiteID, subtree.PageID)
		if err != nil {
			return nil, err
		}
		under, err := s.rows.IDsUnder(ctx, sel.SiteID, trail, strings.TrimSpace(subtree.TypeID))
		if err != nil {
			return nil, err
		}
		for _, pageID := range under {
			delete(picked, pageID)
		}
	}
	if len(picked) > SelectionLimit {
		return nil, errors.New(errors.Invalid, "a selection holds at most "+strconv.Itoa(SelectionLimit)+" pages; narrow it").
			WithDetail("field", "selection").WithDetail("reason", reasonSelectionLimit)
	}
	out := make([]string, 0, len(picked))
	for pageID := range picked {
		out = append(out, pageID)
	}
	slices.Sort(out)
	return out, nil
}

func seenAgain(seen, now int) error {
	if seen > 0 && seen != now {
		return errors.New(errors.Conflict, "the pages that match changed since they were selected; select them again").
			WithDetail("field", "seen").WithDetail("reason", reasonSelectionMoved).WithDetail("now", now)
	}
	return nil
}

func (s *Service) SetTypes(ctx context.Context, req SetTypesRequest) (SetTypesResponse, error) {
	by := req.By
	if by == pagemap.TypeByNone {
		by = pagemap.TypeByUser
	}
	if by != pagemap.TypeByUser && by != pagemap.TypeByAI {
		return SetTypesResponse{}, errors.New(errors.Invalid, "a page's type is set by a person or the agent").WithDetail("field", "by")
	}
	ids, err := s.resolve(ctx, req.Selection)
	if err != nil {
		return SetTypesResponse{}, err
	}
	siteID := req.Selection.SiteID
	var out SetTypesResponse
	if req.PageTypeID == nil {
		if out.Typed, err = s.backToTheRules(ctx, siteID, ids); err != nil {
			return out, err
		}
		return out, s.changed(siteID)
	}
	kind, err := s.types.Get(ctx, strings.TrimSpace(*req.PageTypeID))
	if err != nil {
		return SetTypesResponse{}, err
	}
	if kind.SiteID != siteID {
		return SetTypesResponse{}, errors.New(errors.Invalid, "the type "+kind.Name+" belongs to another site").WithDetail("field", "pageTypeId")
	}
	for start := 0; start < len(ids); start += bulkBatch {
		batch := ids[start:min(start+bulkBatch, len(ids))]
		err = s.uow.Do(ctx, func(c context.Context) error {
			held, listErr := s.pages.ListByIDs(c, batch)
			if listErr != nil {
				return listErr
			}
			fitting := make([]string, 0, len(held))
			for i := range held {
				if pagetype.Fits(kind.Preset.Role(), held[i].WPType) {
					fitting = append(fitting, held[i].ID)
				} else {
					out.Refused++
				}
			}
			at := s.now()
			typed, setErr := s.pages.SetTypes(c, siteID, fitting, kind.ID, by, at)
			if setErr != nil {
				return setErr
			}
			out.Typed += int(typed)
			return s.clearDefaults(c, siteID, fitting)
		})
		if err != nil {
			return out, err
		}
	}
	return out, s.changed(siteID)
}

func (s *Service) clearDefaults(ctx context.Context, siteID string, pageIDs []string) error {
	if s.findings == nil {
		return nil
	}
	for _, pageID := range pageIDs {
		if err := s.findings.Clear(ctx, siteID, finding.TypeByDefault, finding.SubjectPage(pageID),
			finding.ResolvedByModel, finding.ResolutionPageTyped, s.now()); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) backToTheRules(ctx context.Context, siteID string, ids []string) (int, error) {
	for start := 0; start < len(ids); start += bulkBatch {
		batch := ids[start:min(start+bulkBatch, len(ids))]
		if err := s.uow.Do(ctx, func(c context.Context) error {
			_, clearErr := s.pages.ClearTypes(c, siteID, batch, s.now())
			return clearErr
		}); err != nil {
			return 0, err
		}
	}
	return s.classifier.Reclassify(ctx, siteID, ids)
}

func (s *Service) SetTemplates(ctx context.Context, req SetTemplatesRequest) (SetTemplatesResponse, error) {
	ids, err := s.resolve(ctx, req.Selection)
	if err != nil {
		return SetTemplatesResponse{}, err
	}
	siteID := req.Selection.SiteID
	var templateID *string
	if req.TemplateID != nil && strings.TrimSpace(*req.TemplateID) != "" {
		found, getErr := s.templates.Get(ctx, strings.TrimSpace(*req.TemplateID))
		if getErr != nil {
			return SetTemplatesResponse{}, getErr
		}
		if found.Scope == template.ScopeSite && (found.SiteID == nil || *found.SiteID != siteID) {
			return SetTemplatesResponse{}, errors.New(errors.Invalid, "the template "+found.Name+" belongs to another site").
				WithDetail("field", "templateId")
		}
		templateID = &found.ID
	}
	changed, err := s.setTemplates(ctx, siteID, ids, templateID)
	if err != nil {
		return SetTemplatesResponse{Changed: changed}, err
	}
	return SetTemplatesResponse{Changed: changed}, s.changed(siteID)
}

func (s *Service) setTemplates(ctx context.Context, siteID string, ids []string, templateID *string) (int, error) {
	changed := 0
	for start := 0; start < len(ids); start += bulkBatch {
		batch := ids[start:min(start+bulkBatch, len(ids))]
		err := s.uow.Do(ctx, func(c context.Context) error {
			written, setErr := s.pages.SetTemplates(c, siteID, batch, templateID, s.now())
			changed += int(written)
			return setErr
		})
		if err != nil {
			return changed, err
		}
	}
	return changed, nil
}
