package observe

import (
	"cmp"
	"context"
	"time"
	"unicode/utf8"

	"github.com/davidmovas/postulator/internal/application/events"
	"github.com/davidmovas/postulator/internal/domain/contextpack"
	"github.com/davidmovas/postulator/internal/domain/finding"
	"github.com/davidmovas/postulator/internal/domain/graph"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/pagetype"
	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/kernel/id"
)

type topicRun struct {
	plan    contextpack.TopicPlan
	written map[string]string
}

func (s *Service) topics(ctx context.Context, state *State, batch int) (done bool, err error) {
	defer func() {
		if done || err != nil {
			s.unstage(state)
		}
	}()
	held, err := s.staged(ctx, state, func(c context.Context) (*staged, error) {
		plan, planErr := s.topicPlan(c, state.SiteID)
		if planErr != nil {
			return nil, planErr
		}
		return &staged{topics: &topicRun{plan: plan, written: map[string]string{}}}, nil
	})
	if err != nil {
		return false, err
	}
	run := held.topics

	pending := make([]contextpack.TopicNode, 0, batch)
	more := false
	for i := range run.plan.Writes {
		if run.plan.Writes[i].Order <= state.TopicsAfter {
			continue
		}
		if len(pending) == batch {
			more = true
			break
		}
		pending = append(pending, run.plan.Writes[i])
	}
	if len(pending) > 0 {
		if writeErr := s.writeTopics(ctx, state, run, pending); writeErr != nil {
			return false, writeErr
		}
		state.TopicsAfter = pending[len(pending)-1].Order
	}
	if more {
		return false, nil
	}
	state.Counts.Unplaced, state.Counts.UnplacedBy = len(run.plan.Unplaced), unplacedBy(run.plan.Unplaced)
	return true, s.settleEntities(ctx, state)
}

func (s *Service) topicPlan(ctx context.Context, siteID string) (contextpack.TopicPlan, error) {
	m, err := s.model(ctx, siteID)
	if err != nil {
		return contextpack.TopicPlan{}, err
	}
	set, err := s.deps.Types.Set(ctx, siteID)
	if err != nil {
		return contextpack.TopicPlan{}, err
	}
	g, err := s.graph(ctx, siteID)
	if err != nil {
		return contextpack.TopicPlan{}, err
	}
	built, err := s.deps.Topics.ListBySite(ctx, siteID)
	if err != nil {
		return contextpack.TopicPlan{}, err
	}
	m.landed = m.landings()
	chains := make(map[string]pagemap.Chain, len(m.renders))
	for i := range m.pages {
		record, rendered := m.renders[m.pages[i].ID]
		if !rendered {
			continue
		}
		signals := pagemap.ParentSignals{Trails: record.Trails, WPParentID: m.wpParentOf(m.pages[i].ID), Landings: m.landed}
		chains[m.pages[i].ID] = pagemap.TopicChain(m.pages[i], signals, m.index)
	}
	tree := contextpack.Snapshot{Index: m.index, Types: set, Fields: m.fields, Graph: g}.Map(nil)
	return tree.TopicMap(chains, built), nil
}

func (s *Service) writeTopics(ctx context.Context, state *State, run *topicRun, nodes []contextpack.TopicNode) error {
	now := s.now()
	return s.inUnit(ctx, func(c context.Context) error {
		for i := range nodes {
			if err := s.writeTopic(c, state, run, &nodes[i], now); err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *Service) writeTopic(ctx context.Context, state *State, run *topicRun, node *contextpack.TopicNode, now time.Time) error {
	switch node.Action {
	case contextpack.TopicRecord:
		run.written[node.Subject] = node.EntityID
		return s.deps.Topics.Record(ctx, state.SiteID, node.Subject, now)
	case contextpack.TopicShare:
		return s.shareTopic(ctx, state, run, node, now)
	}
	parentID := node.ParentEntityID
	if parentID == "" && node.Parent != "" {
		if parentID = run.written[node.Parent]; parentID == "" {
			return nil
		}
	}

	var (
		entityID string
		err      error
	)
	if node.Action == contextpack.TopicCreate {
		entityID, err = s.createTopic(ctx, state.SiteID, node, parentID, now)
	} else {
		entityID, err = s.adoptTopic(ctx, node, now)
	}
	if err != nil || entityID == "" {
		return err
	}
	if parentID != "" && (node.Action == contextpack.TopicCreate || node.EdgeMissing) {
		if edgeErr := s.parentEdge(ctx, state.SiteID, entityID, parentID, node, now); edgeErr != nil {
			return edgeErr
		}
	}
	if recordErr := s.deps.Topics.Record(ctx, state.SiteID, node.Subject, now); recordErr != nil {
		return recordErr
	}
	run.written[node.Subject] = entityID
	if node.Action == contextpack.TopicCreate {
		state.Counts.Entities++
	}
	if node.PageID != "" {
		state.Counts.Placed++
	}
	return nil
}

func (s *Service) shareTopic(ctx context.Context, state *State, run *topicRun, node *contextpack.TopicNode, now time.Time) error {
	entityID := cmp.Or(node.EntityID, run.written[node.Hub])
	if entityID == "" {
		return nil
	}
	mapped, err := s.deps.Pages.MapEntity(ctx, node.PageID, entityID, now)
	if err != nil || !mapped {
		return err
	}
	if recordErr := s.deps.Topics.Record(ctx, state.SiteID, node.Subject, now); recordErr != nil {
		return recordErr
	}
	run.written[node.Subject] = entityID
	state.Counts.Placed++
	return nil
}

func (s *Service) createTopic(ctx context.Context, siteID string, node *contextpack.TopicNode, parentID string, now time.Time) (string, error) {
	entity, err := graph.NewEntity(graph.Entity{
		ID: id.New(), SiteID: siteID, Name: node.Name, Facts: node.Facts, Keywords: node.Keywords,
		ScopeID: optional(parentID), CanonicalPageID: optional(node.PageID), Source: graph.SourceSite, CreatedAt: now, UpdatedAt: now,
	})
	if err != nil {
		return "", nil
	}
	if err = s.deps.Entities.Insert(ctx, entity); err != nil {
		if errors.IsCode(err, errors.Conflict) {
			return "", nil
		}
		return "", err
	}
	if node.PageID == "" {
		return entity.ID, nil
	}
	mapped, err := s.deps.Pages.MapEntity(ctx, node.PageID, entity.ID, now)
	if err != nil {
		return "", err
	}
	if !mapped {
		return "", s.deps.Entities.Delete(ctx, entity.ID)
	}
	return entity.ID, nil
}

func (s *Service) adoptTopic(ctx context.Context, node *contextpack.TopicNode, now time.Time) (string, error) {
	mapped, err := s.deps.Pages.MapEntity(ctx, node.PageID, node.EntityID, now)
	if err != nil || !mapped {
		return "", err
	}
	if canonicalErr := s.deps.Entities.SetCanonicalPage(ctx, node.EntityID, &node.PageID, now); canonicalErr != nil {
		return "", canonicalErr
	}
	return node.EntityID, nil
}

func (s *Service) parentEdge(ctx context.Context, siteID, entityID, parentID string, node *contextpack.TopicNode, now time.Time) error {
	edge, err := graph.NewEdge(graph.Edge{
		ID: id.New(), SiteID: siteID, FromEntityID: entityID, ToEntityID: parentID, Kind: graph.EdgeParent, Weight: 1,
		Source: graph.SourceSite, Status: graph.StatusApproved, Reason: breadcrumbReason(node.Name, node.ParentName), CreatedAt: now,
	})
	if err != nil {
		return err
	}
	if err = s.deps.Edges.Insert(ctx, edge); err != nil && !errors.IsCode(err, errors.Conflict) {
		return err
	}
	return nil
}

func breadcrumbReason(name, parent string) string {
	reason := name + " sits under " + parent + " in the site's breadcrumbs"
	if utf8.RuneCountInString(reason) <= graph.EdgeReasonMax {
		return reason
	}
	return string([]rune(reason)[:graph.EdgeReasonMax])
}

func optional(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func unplacedBy(misses []contextpack.TopicMiss) map[string]int {
	if len(misses) == 0 {
		return nil
	}
	out := make(map[string]int)
	for _, miss := range misses {
		out[string(miss.Skip)]++
	}
	return out
}

func (s *Service) settleEntities(ctx context.Context, state *State) error {
	tracked, err := s.findings(ctx, state.SiteID, finding.PageWithoutEntity, finding.EntityParentDiffers)
	if err != nil {
		return err
	}
	pages, err := s.deps.Pages.ListBySite(ctx, state.SiteID)
	if err != nil {
		return err
	}
	stored, err := s.deps.Fields.ListBySite(ctx, state.SiteID)
	if err != nil {
		return err
	}
	set, err := s.deps.Types.Set(ctx, state.SiteID)
	if err != nil {
		return err
	}
	m := siteModel{pages: pages, fields: stored, names: pagemap.Names(stored)}
	m.reindex()
	for i := range m.pages {
		if settleErr := settleEntityless(ctx, tracked, &m.pages[i], set); settleErr != nil {
			return settleErr
		}
	}
	if differErr := s.parentsDiffer(ctx, tracked, state.SiteID, m, set); differErr != nil {
		return differErr
	}
	if s.deps.Publisher == nil || state.Counts.Entities+state.Counts.Placed == 0 {
		return nil
	}
	if publishErr := s.deps.Publisher.Publish(events.GraphChanged, events.GraphChangedPayload{SiteID: state.SiteID}); publishErr != nil {
		return publishErr
	}
	return s.deps.Publisher.Publish(events.PagesChanged, events.PagesChangedPayload{SiteID: state.SiteID})
}

func settleEntityless(ctx context.Context, tracked *findings, page *pagemap.Page, set pagetype.Set) error {
	subject := finding.SubjectPage(page.ID)
	if !set.NeedsEntity(*page) {
		return tracked.clear(ctx, finding.PageWithoutEntity, subject)
	}
	return tracked.raise(ctx, finding.PageWithoutEntity, &page.ID, subject, map[string]string{"path": page.Path})
}
