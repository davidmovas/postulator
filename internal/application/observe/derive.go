package observe

import (
	"context"
	"time"

	"github.com/davidmovas/postulator/internal/application/events"
	"github.com/davidmovas/postulator/internal/domain/block"
	"github.com/davidmovas/postulator/internal/domain/contextpack"
	"github.com/davidmovas/postulator/internal/domain/finding"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/pagetype"
	"github.com/davidmovas/postulator/internal/domain/render"
	"github.com/davidmovas/postulator/internal/domain/site"
)

type timeStamp = time.Time

var deriveCodes = []finding.Code{
	finding.TechnicalDetected, finding.BuilderDetected, finding.ParentUnresolved, finding.ParentAmbiguous, finding.ParentCycle,
	finding.ParentNameTwin, finding.ParentDiffers, finding.URLFormMixed, finding.RenderUnreachable, finding.RenderRedirected,
	finding.PageWithoutEntity, finding.TypeByDefault,
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
	if m.bodies, err = s.currentBodies(ctx, state.SiteID); err != nil {
		return err
	}
	tracked, err := s.findings(ctx, state.SiteID, deriveCodes...)
	if err != nil {
		return err
	}
	carried, err := s.deps.Identity.ListTermsBySite(ctx, state.SiteID)
	if err != nil {
		return err
	}
	set, err := s.deps.Types.Set(ctx, state.SiteID)
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
		if err := s.classify(ctx, tracked, &page, m, set, settings, state.Technical); err != nil {
			return err
		}
		if before.TechnicalReason != page.TechnicalReason || before.ReadOnlyReason != page.ReadOnlyReason {
			changed[page.ID] = page
			m.pages[i] = page
		}
	}
	m.reindex()
	m.landed = m.landings()
	for i := range m.pages {
		page := m.pages[i]
		if page.WPID == nil {
			continue
		}
		before := page
		if err := s.parent(ctx, tracked, &page, m); err != nil {
			return err
		}
		if derivedMoved(before, page) {
			changed[page.ID] = page
			m.pages[i] = page
		}
	}
	if err := s.breakCycles(ctx, tracked, &m, changed); err != nil {
		return err
	}
	if err := s.urlForm(ctx, tracked, m); err != nil {
		return err
	}
	for i := range m.pages {
		page := m.pages[i]
		if state.Full && archive(page, m.identities[page.ID], state, readers) {
			page.Status = pagemap.StatusArchived
			changed[page.ID] = page
			m.pages[i] = page
			state.Counts.Archived++
		}
		if page.Status != pagemap.StatusArchived {
			continue
		}
		for _, code := range []finding.Code{finding.RenderUnreachable, finding.RenderRedirected, finding.ParentCycle, finding.ParentDiffers, finding.TypeByDefault} {
			if err := tracked.clear(ctx, code, finding.SubjectPage(page.ID)); err != nil {
				return err
			}
		}
	}
	m.reindex()
	if err := s.trails(ctx, tracked, m, carried); err != nil {
		return err
	}

	rules, err := s.deps.Types.Rules(ctx, state.SiteID)
	if err != nil {
		return err
	}
	products, err := s.deps.Links.ListedProductsBySite(ctx, state.SiteID)
	if err != nil {
		return err
	}
	typed := retyped(m, set, rules, products)
	if err := s.byDefault(ctx, tracked, m, typed, set); err != nil {
		return err
	}
	if err := s.entityless(ctx, tracked, m, changed, typed, set); err != nil {
		return err
	}

	err = s.inUnit(ctx, func(c context.Context) error {
		for id := range changed {
			page := changed[id]
			page.UpdatedAt = now
			if updateErr := s.deps.Pages.UpdateDerived(c, page); updateErr != nil {
				return updateErr
			}
		}
		for i := range typed {
			outcome := &typed[i]
			if !outcome.Changed() {
				continue
			}
			placed := pagetype.Assignment{TypeID: outcome.TypeID, Source: pagemap.TypeSourceAuto, Reason: outcome.Reason, RuleID: outcome.RuleID}
			moved, typeErr := s.deps.Pages.Classify(c, outcome.PageID, outcome.Current, placed, now)
			if typeErr != nil {
				return typeErr
			}
			if moved {
				state.Counts.Typed++
			}
		}
		return s.resolveLinks(c, state.SiteID, pagemap.NewIndex(m.pages))
	})
	if err != nil {
		return err
	}

	observation := owner.Settings.Observation
	started, finished := state.StartedAt, now
	observation.LastStartedAt, observation.LastFinishedAt = &started, &finished
	if !state.ModifiedSeen.IsZero() && (observation.LastModifiedSeen == nil || state.ModifiedSeen.After(*observation.LastModifiedSeen)) {
		seen := state.ModifiedSeen
		observation.LastModifiedSeen = &seen
	}
	if state.Full {
		observation.LastFullWalkAt = &started
		observation.SinceFullWalk = 0
	} else {
		observation.SinceFullWalk++
	}
	if err := s.deps.Sites.UpdateObservation(ctx, owner.ID, state.Technical, observation, now); err != nil {
		return err
	}
	if s.deps.Maps != nil {
		if err := s.deps.Maps.Refresh(ctx, state.SiteID); err != nil {
			return err
		}
	}
	if s.deps.Publisher != nil {
		return s.deps.Publisher.Publish(events.PagesChanged, events.PagesChangedPayload{SiteID: state.SiteID})
	}
	return nil
}

func typedRegular(page pagemap.Page, set pagetype.Set) bool {
	if page.TypeSource != pagemap.TypeSourceManual || page.PageTypeID == nil {
		return false
	}
	kind, found := set.ByID(*page.PageTypeID)
	return found && kind.Preset.Role() != pagetype.RoleTechnical
}

func (s *Service) classify(ctx context.Context, tracked *findings, page *pagemap.Page, m siteModel, set pagetype.Set, settings site.Settings, technical site.TechnicalPages) error {
	record, rendered := m.renders[page.ID]
	if page.TechnicalReason == pagemap.TechnicalNone || page.TechnicalReason.Observed() {
		reason := technicalReason(*page, m.bodies[page.ID], record, rendered, settings, technical)
		if typedRegular(*page, set) {
			reason = pagemap.TechnicalNone
		}
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
			technical.Posts: pagemap.TechnicalPosts, technical.Privacy: pagemap.TechnicalPrivacy,
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

func retyped(m siteModel, set pagetype.Set, rules []pagetype.Rule, products map[string]int) []pagetype.Outcome {
	rendered := make(map[string]pagetype.Rendered, len(m.renders))
	for id := range m.renders {
		record := m.renders[id]
		rendered[id] = pagetype.RenderedOf(&record)
	}
	tree := contextpack.Snapshot{Index: m.index, Types: set, Fields: m.fields}.Map(nil)
	return pagetype.NewClassifier(rules, set).Classify(tree.Facts(rendered, products), pagetype.ModeWalk)
}

func (s *Service) byDefault(ctx context.Context, tracked *findings, m siteModel, typed []pagetype.Outcome, set pagetype.Set) error {
	for i := range typed {
		outcome := &typed[i]
		reason, typeID := outcome.Current.Reason, outcome.Current.TypeID
		if outcome.Changed() {
			reason, typeID = outcome.Reason, outcome.TypeID
		}
		subject := finding.SubjectPage(outcome.PageID)
		if reason != pagemap.TypeByUnmatched {
			if err := tracked.clear(ctx, finding.TypeByDefault, subject); err != nil {
				return err
			}
			continue
		}
		kind, _ := set.ByID(typeID)
		if err := tracked.raise(ctx, finding.TypeByDefault, &outcome.PageID, subject, map[string]string{
			"path": m.pathOf(outcome.PageID), "type": kind.Name,
		}); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) entityless(ctx context.Context, tracked *findings, m siteModel, changed map[string]pagemap.Page, typed []pagetype.Outcome, set pagetype.Set) error {
	assigned := make(map[string]string, len(typed))
	for i := range typed {
		if typed[i].Changed() {
			assigned[typed[i].PageID] = typed[i].TypeID
		}
	}
	for i := range m.pages {
		page := m.pages[i]
		if moved, held := changed[page.ID]; held {
			page = moved
		}
		if typeID, held := assigned[page.ID]; held {
			page.PageTypeID = &typeID
		}
		subject := finding.SubjectPage(page.ID)
		if !set.NeedsEntity(page) {
			if err := tracked.clear(ctx, finding.PageWithoutEntity, subject); err != nil {
				return err
			}
			continue
		}
		if err := tracked.raise(ctx, finding.PageWithoutEntity, &page.ID, subject, map[string]string{"path": page.Path}); err != nil {
			return err
		}
	}
	return nil
}

func derivedMoved(before, after pagemap.Page) bool {
	return !sameRef(before.ParentPageID, after.ParentPageID) || before.ParentReason != after.ParentReason || before.ParentMethod != after.ParentMethod
}

func (m siteModel) landings() map[string]string {
	out := make(map[string]string)
	for id := range m.renders {
		if final := m.renders[id].FinalPath; final != "" {
			if page, found := m.index.ByID(id); found {
				out[page.Path] = final
			}
		}
	}
	return out
}

func (m siteModel) wpParentOf(pageID string) *int64 {
	if identity, observed := m.identities[pageID]; observed {
		return identity.WPParentID
	}
	return nil
}

func (s *Service) parent(ctx context.Context, tracked *findings, page *pagemap.Page, m siteModel) error {
	var trails []pagemap.Trail
	if record, rendered := m.renders[page.ID]; rendered {
		trails = record.Trails
	}
	derived := pagemap.DeriveParent(*page, pagemap.ParentSignals{Trails: trails, WPParentID: m.wpParentOf(page.ID), Landings: m.landed}, m.index)
	page.ParentPageID = derived.ParentID
	page.ParentReason = derived.Reason
	page.ParentMethod = derived.Method

	unresolved := make(map[string]struct{}, len(derived.Unresolved))
	for _, crumb := range derived.Unresolved {
		subject := finding.SubjectCrumb(page.ID, crumb.Key())
		unresolved[subject] = struct{}{}
		if err := tracked.raise(ctx, finding.ParentUnresolved, &page.ID, subject, map[string]string{"name": crumb.Name, "path": crumb.Path}); err != nil {
			return err
		}
	}
	ambiguous := make(map[string]struct{}, len(derived.Ambiguous))
	for _, tie := range derived.Ambiguous {
		subject := finding.SubjectCrumb(page.ID, tie.Crumb.Key())
		ambiguous[subject] = struct{}{}
		candidates := make([]string, 0, len(tie.Candidates))
		for _, candidate := range tie.Candidates {
			if found, ok := m.index.ByID(candidate); ok {
				candidates = append(candidates, found.Path)
			}
		}
		if err := tracked.raise(ctx, finding.ParentAmbiguous, &page.ID, subject, map[string]any{"name": tie.Crumb.Name, "candidates": candidates}); err != nil {
			return err
		}
	}
	if err := pinToWorkbook(ctx, tracked, page, m, ambiguous); err != nil {
		return err
	}
	twins := make(map[string]struct{}, len(derived.Twins))
	for _, twin := range derived.Twins {
		subject := finding.SubjectCrumb(page.ID, twin.Crumb.Key())
		twins[subject] = struct{}{}
		if err := tracked.raise(ctx, finding.ParentNameTwin, &page.ID, subject, map[string]any{
			"name": twin.Crumb.Name, "page": m.pathOf(twin.Page), "terms": m.pathsOf(twin.Terms),
		}); err != nil {
			return err
		}
	}
	if err := clearStale(ctx, tracked, finding.ParentUnresolved, page.ID, unresolved); err != nil {
		return err
	}
	if err := clearStale(ctx, tracked, finding.ParentNameTwin, page.ID, twins); err != nil {
		return err
	}
	return clearStale(ctx, tracked, finding.ParentAmbiguous, page.ID, ambiguous)
}

func pinToWorkbook(ctx context.Context, tracked *findings, page *pagemap.Page, m siteModel, ambiguous map[string]struct{}) error {
	if page.ParentSource != pagemap.ParentFromWorkbook || page.WPType == pagemap.WPProduct || len(page.PlannedTrail) == 0 {
		return nil
	}
	level := page.PlannedTrail[len(page.PlannedTrail)-1]
	candidates := m.index.ParentNamed(*page, level)
	switch len(candidates) {
	case 0:
		return nil
	case 1:
		parentID := candidates[0].ID
		page.ParentPageID, page.ParentReason, page.ParentMethod = &parentID, pagemap.ParentWorkbook, pagemap.MethodName
		return nil
	}
	subject := finding.SubjectCrumb(page.ID, level)
	if _, raised := ambiguous[subject]; raised {
		return nil
	}
	ambiguous[subject] = struct{}{}
	paths := make([]string, 0, len(candidates))
	for i := range candidates {
		paths = append(paths, candidates[i].Path)
	}
	return tracked.raise(ctx, finding.ParentAmbiguous, &page.ID, subject, map[string]any{"name": level, "candidates": paths})
}

func (s *Service) trails(ctx context.Context, tracked *findings, m siteModel, carried map[string][]pagemap.Term) error {
	shown := pagemap.NewTrailSite(m.index, carried, pagemap.TermParents(m.pages, m.identities))
	for i := range m.pages {
		page := m.pages[i]
		if page.WPID == nil || page.Status == pagemap.StatusArchived {
			continue
		}
		subject := finding.SubjectPage(page.ID)
		diff, differs := pagemap.TrailDiffers(page, shown)
		if !differs {
			if err := tracked.clear(ctx, finding.ParentDiffers, subject); err != nil {
				return err
			}
			continue
		}
		if err := tracked.raise(ctx, finding.ParentDiffers, &page.ID, subject, diff, pagemap.TrailOptions(page.WPType)...); err != nil {
			return err
		}
	}
	return nil
}

func (m siteModel) pathOf(pageID string) string {
	if found, ok := m.index.ByID(pageID); ok {
		return found.Path
	}
	return ""
}

func (m siteModel) pathsOf(pageIDs []string) []string {
	paths := make([]string, 0, len(pageIDs))
	for _, id := range pageIDs {
		if path := m.pathOf(id); path != "" {
			paths = append(paths, path)
		}
	}
	return paths
}

func clearStale(ctx context.Context, tracked *findings, code finding.Code, pageID string, raised map[string]struct{}) error {
	open := tracked.openFor(code, finding.SubjectCrumb(pageID, ""))
	for i := range open {
		if _, still := raised[open[i].SubjectKey]; still {
			continue
		}
		if err := tracked.clear(ctx, code, open[i].SubjectKey); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) breakCycles(ctx context.Context, tracked *findings, m *siteModel, changed map[string]pagemap.Page) error {
	position := make(map[string]int, len(m.pages))
	for i := range m.pages {
		position[m.pages[i].ID] = i
	}
	broken := make(map[string][]string)
	for cycles := pagemap.Cycles(m.pages); len(cycles) > 0; cycles = pagemap.Cycles(m.pages) {
		progressed := false
		for _, cycle := range cycles {
			at := -1
			for i := range cycle {
				if cycle[i].WPID != nil {
					at = position[cycle[i].ID]
					break
				}
			}
			if at < 0 {
				continue
			}
			page := m.pages[at]
			if _, again := broken[page.ID]; again {
				page.ParentPageID, page.ParentReason, page.ParentMethod = nil, pagemap.ParentNone, pagemap.MethodNone
			} else {
				derived := pagemap.DeriveParent(page, pagemap.ParentSignals{WPParentID: m.wpParentOf(page.ID)}, m.index)
				page.ParentPageID, page.ParentReason, page.ParentMethod = derived.ParentID, derived.Reason, derived.Method
				paths := make([]string, 0, len(cycle))
				for i := range cycle {
					paths = append(paths, cycle[i].Path)
				}
				broken[page.ID] = paths
			}
			m.pages[at] = page
			changed[page.ID] = page
			progressed = true
		}
		if !progressed {
			break
		}
		m.reindex()
	}

	for pageID, paths := range broken {
		if err := tracked.raise(ctx, finding.ParentCycle, &pageID, finding.SubjectPage(pageID), map[string]any{"paths": paths}); err != nil {
			return err
		}
		if err := clearStale(ctx, tracked, finding.ParentNameTwin, pageID, nil); err != nil {
			return err
		}
	}
	open := tracked.openFor(finding.ParentCycle, "")
	for i := range open {
		if open[i].PageID == nil {
			continue
		}
		if _, still := broken[*open[i].PageID]; !still {
			if err := tracked.clear(ctx, finding.ParentCycle, open[i].SubjectKey); err != nil {
				return err
			}
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
