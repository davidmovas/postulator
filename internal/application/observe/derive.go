package observe

import (
	"context"
	"time"

	"github.com/davidmovas/postulator/internal/application/events"
	"github.com/davidmovas/postulator/internal/domain/block"
	"github.com/davidmovas/postulator/internal/domain/finding"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/render"
	"github.com/davidmovas/postulator/internal/domain/site"
)

type timeStamp = time.Time

var deriveCodes = []finding.Code{
	finding.TechnicalDetected, finding.BuilderDetected, finding.ParentUnresolved, finding.ParentDiffers, finding.URLFormMixed,
	finding.RenderUnreachable,
}

func (s *Service) derive(ctx context.Context, state *State, readers Readers) error {
	owner, err := s.deps.Sites.Get(ctx, state.SiteID)
	if err != nil {
		return err
	}
	settings := owner.Settings.Normalized()
	m, err := s.model(ctx, state.SiteID)
	if err != nil {
		return err
	}
	tracked, err := s.findings(ctx, state.SiteID, deriveCodes...)
	if err != nil {
		return err
	}
	now := s.now()

	changed := make(map[string]pagemap.Page)
	for i := range m.pages {
		page := m.pages[i]
		if page.WPID == nil {
			continue
		}
		before := page
		if err := s.classify(ctx, tracked, &page, m, settings, state.Technical); err != nil {
			return err
		}
		if before.TechnicalReason != page.TechnicalReason || before.ReadOnlyReason != page.ReadOnlyReason {
			changed[page.ID] = page
			m.pages[i] = page
		}
	}
	m.index = pagemap.NewIndex(m.pages)
	for i := range m.pages {
		page := m.pages[i]
		if page.WPID == nil {
			continue
		}
		before := page
		if err := s.parent(ctx, tracked, &page, m); err != nil {
			return err
		}
		if !sameRef(before.ParentPageID, page.ParentPageID) || before.ParentReason != page.ParentReason {
			changed[page.ID] = page
			m.pages[i] = page
		}
	}
	if err := s.urlForm(ctx, tracked, m); err != nil {
		return err
	}
	for i := range m.pages {
		page := m.pages[i]
		if state.Full && archive(page, m.identities[page.ID], state, readers) {
			page.Status = pagemap.StatusArchived
			changed[page.ID] = page
			state.Counts.Archived++
		}
		if page.Status != pagemap.StatusArchived {
			continue
		}
		if err := tracked.clear(ctx, finding.RenderUnreachable, finding.SubjectPage(page.ID)); err != nil {
			return err
		}
	}

	err = s.inUnit(ctx, func(c context.Context) error {
		for id := range changed {
			page := changed[id]
			page.UpdatedAt = now
			if updateErr := s.deps.Pages.Update(c, page); updateErr != nil {
				return updateErr
			}
		}
		return s.resolveLinks(c, state.SiteID, pagemap.NewIndex(m.pages))
	})
	if err != nil {
		return err
	}

	observation := &owner.Settings.Observation
	started := state.StartedAt
	observation.LastStartedAt = &started
	if state.Full {
		observation.LastFullWalkAt = &started
		observation.SinceFullWalk = 0
	} else {
		observation.SinceFullWalk++
	}
	owner.Settings.Technical = state.Technical
	owner.UpdatedAt = now
	if err := s.deps.Sites.Update(ctx, owner); err != nil {
		return err
	}
	if s.deps.Publisher != nil {
		return s.deps.Publisher.Publish(events.PagesChanged, events.PagesChangedPayload{SiteID: state.SiteID})
	}
	return nil
}

func (s *Service) classify(ctx context.Context, tracked *findings, page *pagemap.Page, m siteModel, settings site.Settings, technical site.TechnicalPages) error {
	record, rendered := m.renders[page.ID]
	if page.TechnicalReason == pagemap.TechnicalNone || page.TechnicalReason.Observed() {
		reason := technicalReason(*page, m.bodies[page.ID], record, rendered, settings, technical)
		subject := finding.SubjectPage(page.ID)
		switch {
		case reason != pagemap.TechnicalNone && page.TechnicalReason == pagemap.TechnicalNone:
			if err := tracked.raise(ctx, finding.TechnicalDetected, &page.ID, subject, map[string]string{"reason": string(reason)}); err != nil {
				return err
			}
		case reason == pagemap.TechnicalNone:
			if err := tracked.clear(ctx, finding.TechnicalDetected, subject); err != nil {
				return err
			}
		}
		page.TechnicalReason = reason
	}

	subject := finding.SubjectPage(page.ID)
	switch {
	case page.WPType.Term():
		page.ReadOnlyReason = pagemap.ReadOnlyTerm
	case rendered && record.Builder != "":
		page.ReadOnlyReason = pagemap.ReadOnlyBuilder(record.Builder)
		return tracked.raise(ctx, finding.BuilderDetected, &page.ID, subject, map[string]string{"builder": record.Builder})
	default:
		page.ReadOnlyReason = pagemap.ReadOnlyNone
	}
	return tracked.clear(ctx, finding.BuilderDetected, subject)
}

func technicalReason(page pagemap.Page, body block.Snapshot, record render.Record, rendered bool, settings site.Settings, technical site.TechnicalPages) pagemap.TechnicalReason {
	if !page.WPType.Term() && page.WPID != nil {
		ids := map[int64]pagemap.TechnicalReason{}
		for id, reason := range map[int64]pagemap.TechnicalReason{
			technical.Front: pagemap.TechnicalFront, technical.Posts: pagemap.TechnicalPosts, technical.Privacy: pagemap.TechnicalPrivacy,
			technical.Shop: pagemap.TechnicalShop, technical.Cart: pagemap.TechnicalCart, technical.Checkout: pagemap.TechnicalCheckout,
			technical.Account: pagemap.TechnicalAccount, technical.Terms: pagemap.TechnicalTerms,
		} {
			if id != 0 {
				ids[id] = reason
			}
		}
		if reason, technicalPage := ids[*page.WPID]; technicalPage {
			return reason
		}
	}
	if body.Raw != "" {
		parsed := block.Parse(body.Raw)
		for _, shortcode := range parsed.Shortcodes() {
			if contains(settings.TechnicalShortcodes, shortcode) {
				return pagemap.TechnicalShortcode
			}
		}
		for _, name := range parsed.BlockNames() {
			if contains(settings.TechnicalBlocks, name) {
				return pagemap.TechnicalShortcode
			}
		}
	}
	if rendered && record.NoIndex() {
		return pagemap.TechnicalNoindex
	}
	return pagemap.TechnicalNone
}

func (s *Service) parent(ctx context.Context, tracked *findings, page *pagemap.Page, m siteModel) error {
	var trails []pagemap.Trail
	if record, rendered := m.renders[page.ID]; rendered {
		trails = record.Trails
	}
	var wpParent *int64
	if identity, observed := m.identities[page.ID]; observed {
		wpParent = identity.WPParentID
	}
	derived := pagemap.DeriveParent(*page, trails, wpParent, m.index)

	if page.ParentReason == pagemap.ParentWorkbook && page.ParentPageID != nil && !sameRef(page.ParentPageID, derived.ParentID) {
		workbook := *page.ParentPageID
		derivedID := ""
		if derived.ParentID != nil {
			derivedID = *derived.ParentID
		}
		if err := tracked.raise(ctx, finding.ParentDiffers, &page.ID, finding.SubjectPage(page.ID), map[string]string{
			"workbook": workbook, "derived": derivedID, "reason": string(derived.Reason),
		}); err != nil {
			return err
		}
	}
	page.ParentPageID = derived.ParentID
	page.ParentReason = derived.Reason

	raised := make(map[string]struct{}, len(derived.Unresolved))
	for _, crumb := range derived.Unresolved {
		key := crumb.Path
		if key == "" {
			key = crumb.Name
		}
		subject := finding.SubjectCrumb(page.ID, key)
		raised[subject] = struct{}{}
		if err := tracked.raise(ctx, finding.ParentUnresolved, &page.ID, subject, map[string]string{"name": crumb.Name, "path": crumb.Path}); err != nil {
			return err
		}
	}
	open := tracked.openFor(finding.ParentUnresolved, finding.SubjectCrumb(page.ID, ""))
	for i := range open {
		if _, still := raised[open[i].SubjectKey]; still {
			continue
		}
		if err := tracked.clear(ctx, finding.ParentUnresolved, open[i].SubjectKey); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) urlForm(ctx context.Context, tracked *findings, m siteModel) error {
	forms := map[string]int{}
	for id := range m.renders {
		if form := m.renders[id].URLForm; form != "" {
			forms[form]++
		}
	}
	if forms["absolute"] > 0 && forms["relative"] > 0 {
		return tracked.raise(ctx, finding.URLFormMixed, nil, finding.SubjectSite, forms)
	}
	return tracked.clear(ctx, finding.URLFormMixed, finding.SubjectSite)
}

func archive(page pagemap.Page, identity pagemap.Identity, state *State, readers Readers) bool {
	if page.WPID == nil || page.Status == pagemap.StatusArchived || !covered(page.WPType, state, readers) {
		return false
	}
	return identity.ObservedAt.IsZero() || identity.ObservedAt.Before(state.StartedAt)
}

func covered(wpType pagemap.WPType, state *State, readers Readers) bool {
	if wpType == pagemap.WPProduct && readers.Store != nil {
		return true
	}
	return contains(state.Types, string(wpType))
}

func (s *Service) resolveLinks(ctx context.Context, siteID string, index pagemap.Index) error {
	links, err := s.deps.Links.ListBySite(ctx, siteID)
	if err != nil {
		return err
	}
	type group struct {
		pageID string
		source pagemap.LinkSource
	}
	pending := make(map[group][]pagemap.PageLink)
	changed := make(map[group]bool)
	for i := range links {
		key := group{links[i].FromPageID, links[i].Source}
		if links[i].ToPageID == nil {
			if target, ok := index.ByPath(links[i].ToURL); ok && target.ID != links[i].FromPageID {
				links[i].ToPageID = &target.ID
				changed[key] = true
			}
		}
		pending[key] = append(pending[key], links[i])
	}
	for key := range changed {
		if err := s.deps.Links.ReplaceForPage(ctx, key.pageID, key.source, pending[key]); err != nil {
			return err
		}
	}
	return nil
}

func sameRef(current, wanted *string) bool {
	if current == nil || wanted == nil {
		return current == nil && wanted == nil
	}
	return *current == *wanted
}
