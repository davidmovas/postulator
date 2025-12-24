package linking

import (
	"context"
	"fmt"
	"time"

	"github.com/davidmovas/postulator/internal/domain/aiusage"
	"github.com/davidmovas/postulator/internal/domain/entities"
	"github.com/davidmovas/postulator/internal/domain/prompts"
	"github.com/davidmovas/postulator/internal/domain/providers"
	"github.com/davidmovas/postulator/internal/domain/sitemap"
	"github.com/davidmovas/postulator/internal/domain/sites"
	"github.com/davidmovas/postulator/internal/infra/database"
	"github.com/davidmovas/postulator/internal/infra/events"
	"github.com/davidmovas/postulator/internal/infra/wp"
	"github.com/davidmovas/postulator/pkg/errors"
	"github.com/davidmovas/postulator/pkg/logger"
)

var _ Service = (*serviceImpl)(nil)

type serviceImpl struct {
	planRepo   PlanRepository
	linkRepo   LinkRepository
	sitemapSvc sitemap.Service
	suggester  *Suggester
	applier    *Applier
	eventBus   *events.EventBus
	logger     *logger.Logger
}

func NewService(
	db *database.DB,
	sitemapSvc sitemap.Service,
	sitesSvc sites.Service,
	providerSvc providers.Service,
	promptSvc prompts.Service,
	wpClient wp.Client,
	aiUsageService aiusage.Service,
	eventBus *events.EventBus,
	logger *logger.Logger,
) Service {
	linkRepo := NewLinkRepository(db.DB)
	return &serviceImpl{
		planRepo:   NewPlanRepository(db.DB),
		linkRepo:   linkRepo,
		sitemapSvc: sitemapSvc,
		suggester:  NewSuggester(sitemapSvc, providerSvc, promptSvc, linkRepo, aiUsageService, eventBus, logger),
		applier:    NewApplier(sitemapSvc, sitesSvc, providerSvc, linkRepo, wpClient, aiUsageService, eventBus, logger),
		eventBus:   eventBus,
		logger:     logger.WithScope("linking"),
	}
}

// Plan management

func (s *serviceImpl) CreatePlan(ctx context.Context, sitemapID int64, siteID int64, name string) (*LinkPlan, error) {
	plan := &LinkPlan{
		SitemapID: sitemapID,
		SiteID:    siteID,
		Name:      name,
		Status:    PlanStatusDraft,
	}

	if err := s.planRepo.Create(ctx, plan); err != nil {
		return nil, err
	}

	s.logger.Infof("Created link plan %d for sitemap %d", plan.ID, sitemapID)
	return plan, nil
}

func (s *serviceImpl) GetPlan(ctx context.Context, id int64) (*LinkPlan, error) {
	return s.planRepo.GetByID(ctx, id)
}

func (s *serviceImpl) GetPlanBySitemap(ctx context.Context, sitemapID int64) (*LinkPlan, error) {
	return s.planRepo.GetBySitemapID(ctx, sitemapID)
}

func (s *serviceImpl) GetActivePlan(ctx context.Context, sitemapID int64) (*LinkPlan, error) {
	return s.planRepo.GetActiveBySitemapID(ctx, sitemapID)
}

func (s *serviceImpl) ListPlans(ctx context.Context, siteID int64) ([]*LinkPlan, error) {
	return s.planRepo.List(ctx, siteID)
}

func (s *serviceImpl) UpdatePlan(ctx context.Context, plan *LinkPlan) error {
	return s.planRepo.Update(ctx, plan)
}

func (s *serviceImpl) DeletePlan(ctx context.Context, id int64) error {
	if err := s.linkRepo.DeleteByPlanID(ctx, id); err != nil {
		s.logger.ErrorWithErr(err, "Failed to delete links for plan")
	}
	return s.planRepo.Delete(ctx, id)
}

// Link management

func (s *serviceImpl) AddLink(ctx context.Context, planID int64, sourceNodeID int64, targetNodeID int64) (*PlannedLink, error) {
	existing, err := s.linkRepo.GetByNodePair(ctx, planID, sourceNodeID, targetNodeID)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return existing, nil
	}

	link := &PlannedLink{
		PlanID:       planID,
		SourceNodeID: sourceNodeID,
		TargetNodeID: targetNodeID,
		Status:       LinkStatusPlanned,
		Source:       LinkSourceManual,
	}

	if err := s.linkRepo.Create(ctx, link); err != nil {
		return nil, err
	}

	return link, nil
}

func (s *serviceImpl) RemoveLink(ctx context.Context, linkID int64) error {
	link, err := s.linkRepo.GetByID(ctx, linkID)
	if err != nil {
		return err
	}

	if link.Status == LinkStatusApplied {
		return errors.Validation("cannot remove applied link")
	}

	if err := s.linkRepo.Delete(ctx, linkID); err != nil {
		return err
	}

	return nil
}

func (s *serviceImpl) UpdateLink(ctx context.Context, link *PlannedLink) error {
	return s.linkRepo.Update(ctx, link)
}

func (s *serviceImpl) GetLinks(ctx context.Context, planID int64) ([]*PlannedLink, error) {
	return s.linkRepo.GetByPlanID(ctx, planID)
}

func (s *serviceImpl) GetLinksByNode(ctx context.Context, planID int64, nodeID int64) ([]*PlannedLink, error) {
	outgoing, err := s.linkRepo.GetBySourceNodeID(ctx, planID, nodeID)
	if err != nil {
		return nil, err
	}

	incoming, err := s.linkRepo.GetByTargetNodeID(ctx, planID, nodeID)
	if err != nil {
		return nil, err
	}

	seen := make(map[int64]bool)
	var result []*PlannedLink
	for _, link := range outgoing {
		if !seen[link.ID] {
			seen[link.ID] = true
			result = append(result, link)
		}
	}
	for _, link := range incoming {
		if !seen[link.ID] {
			seen[link.ID] = true
			result = append(result, link)
		}
	}

	return result, nil
}

func (s *serviceImpl) ApproveLink(ctx context.Context, linkID int64) error {
	return s.linkRepo.UpdateStatus(ctx, linkID, LinkStatusApproved, nil)
}

func (s *serviceImpl) RejectLink(ctx context.Context, linkID int64) error {
	return s.linkRepo.UpdateStatus(ctx, linkID, LinkStatusRejected, nil)
}

func (s *serviceImpl) ApproveAndApplyLink(ctx context.Context, linkID int64) error {
	link, err := s.linkRepo.GetByID(ctx, linkID)
	if err != nil {
		return err
	}

	// Only mark as applied if the link is approved (or planned, for automatic apply during generation)
	if link.Status != LinkStatusApproved && link.Status != LinkStatusPlanned {
		return nil // Skip if already applied, rejected, or failed
	}

	now := time.Now()
	link.Status = LinkStatusApplied
	link.AppliedAt = &now
	return s.linkRepo.Update(ctx, link)
}

func (s *serviceImpl) SuggestLinks(ctx context.Context, config SuggestLinksConfig) error {
	plan, err := s.planRepo.GetByID(ctx, config.PlanID)
	if err != nil {
		return err
	}

	plan.Status = PlanStatusSuggesting
	plan.ProviderID = &config.ProviderID
	plan.PromptID = config.PromptID
	if err := s.planRepo.Update(ctx, plan); err != nil {
		return err
	}

	existingLinks, err := s.linkRepo.GetByPlanID(ctx, config.PlanID)
	if err != nil {
		s.logger.ErrorWithErr(err, "Failed to get existing links")
		existingLinks = []*PlannedLink{}
	}

	result, err := s.suggester.Suggest(ctx, SuggestConfig{
		PlanID:        config.PlanID,
		SitemapID:     plan.SitemapID,
		SiteID:        plan.SiteID,
		ProviderID:    config.ProviderID,
		PromptID:      config.PromptID,
		NodeIDs:       config.NodeIDs,
		Feedback:      config.Feedback,
		MaxIncoming:   config.MaxIncoming,
		MaxOutgoing:   config.MaxOutgoing,
		ExistingLinks: existingLinks,
	})
	if err != nil {
		errMsg := err.Error()
		plan.Status = PlanStatusFailed
		plan.Error = &errMsg
		_ = s.planRepo.Update(ctx, plan)
		return err
	}

	plan.Status = PlanStatusReady
	if err := s.planRepo.Update(ctx, plan); err != nil {
		return err
	}

	s.logger.Infof("AI suggested %d links for plan %d", result.LinksCreated, config.PlanID)
	return nil
}

// ApplyLinks applies approved links to WordPress content using AI
func (s *serviceImpl) ApplyLinks(ctx context.Context, planID int64, linkIDs []int64, providerID int64) (*ApplyResult, error) {
	plan, err := s.planRepo.GetByID(ctx, planID)
	if err != nil {
		return nil, err
	}

	plan.Status = PlanStatusApplying
	if err := s.planRepo.Update(ctx, plan); err != nil {
		return nil, err
	}

	result, err := s.applier.Apply(ctx, ApplyConfig{
		PlanID:     planID,
		SiteID:     plan.SiteID,
		ProviderID: providerID,
		LinkIDs:    linkIDs,
	})
	if err != nil {
		errMsg := err.Error()
		plan.Status = PlanStatusFailed
		plan.Error = &errMsg
		_ = s.planRepo.Update(ctx, plan)
		return nil, err
	}

	// Check if all requested links were processed
	if result.FailedLinks == 0 {
		plan.Status = PlanStatusReady
	} else if result.AppliedLinks > 0 {
		plan.Status = PlanStatusReady // Partial success
	} else {
		plan.Status = PlanStatusFailed
	}
	plan.Error = nil
	_ = s.planRepo.Update(ctx, plan)

	s.logger.Infof("Applied %d/%d links for plan %d", result.AppliedLinks, result.TotalLinks, planID)

	return result, nil
}

// Graph visualization

func (s *serviceImpl) GetLinkGraph(ctx context.Context, planID int64) (*LinkGraph, error) {
	plan, err := s.planRepo.GetByID(ctx, planID)
	if err != nil {
		return nil, err
	}

	_, nodes, err := s.sitemapSvc.GetSitemapWithNodes(ctx, plan.SitemapID)
	if err != nil {
		return nil, err
	}

	links, err := s.linkRepo.GetByPlanID(ctx, planID)
	if err != nil {
		return nil, err
	}

	nodeMap := make(map[int64]*entities.SitemapNode)
	for _, node := range nodes {
		nodeMap[node.ID] = node
	}

	outgoingCount := make(map[int64]int)
	incomingCount := make(map[int64]int)
	for _, link := range links {
		outgoingCount[link.SourceNodeID]++
		incomingCount[link.TargetNodeID]++
	}

	graphNodes := make([]*GraphNode, 0, len(nodes))
	for _, node := range nodes {
		graphNode := &GraphNode{
			NodeID:            node.ID,
			Title:             node.Title,
			Slug:              node.Slug,
			Path:              s.buildNodePath(node, nodeMap),
			HasContent:        node.GenerationStatus == entities.GenStatusGenerated,
			OutgoingLinkCount: outgoingCount[node.ID],
			IncomingLinkCount: incomingCount[node.ID],
		}
		graphNodes = append(graphNodes, graphNode)
	}

	graphEdges := make([]*GraphEdge, 0, len(links))
	for _, link := range links {
		edge := &GraphEdge{
			ID:           link.ID,
			SourceNodeID: link.SourceNodeID,
			TargetNodeID: link.TargetNodeID,
			AnchorText:   link.AnchorText,
			Status:       link.Status,
			Source:       link.Source,
			Confidence:   link.Confidence,
		}
		graphEdges = append(graphEdges, edge)
	}

	return &LinkGraph{
		Nodes: graphNodes,
		Edges: graphEdges,
	}, nil
}

func (s *serviceImpl) buildNodePath(node *entities.SitemapNode, nodeMap map[int64]*entities.SitemapNode) string {
	path := "/" + node.Slug
	current := node
	for current.ParentID != nil {
		parent, ok := nodeMap[*current.ParentID]
		if !ok {
			break
		}
		if parent.Slug != "" {
			path = "/" + parent.Slug + path
		}
		current = parent
	}
	return path
}

// GetOrCreateActivePlan returns the active plan for a sitemap, creating one if needed
func (s *serviceImpl) GetOrCreateActivePlan(ctx context.Context, sitemapID int64, siteID int64) (*LinkPlan, error) {
	plan, err := s.planRepo.GetActiveBySitemapID(ctx, sitemapID)
	if err != nil {
		return nil, err
	}

	if plan != nil {
		return plan, nil
	}

	sm, err := s.sitemapSvc.GetSitemap(ctx, sitemapID)
	if err != nil {
		return nil, err
	}

	planName := fmt.Sprintf("Link Plan - %s", sm.Name)
	return s.CreatePlan(ctx, sitemapID, siteID, planName)
}
