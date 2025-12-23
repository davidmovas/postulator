package linking

import (
	"context"
	"fmt"

	"github.com/davidmovas/postulator/internal/domain/entities"
	"github.com/davidmovas/postulator/internal/domain/sitemap"
	"github.com/davidmovas/postulator/internal/infra/database"
	"github.com/davidmovas/postulator/internal/infra/events"
	"github.com/davidmovas/postulator/pkg/errors"
	"github.com/davidmovas/postulator/pkg/logger"
)

var _ Service = (*serviceImpl)(nil)

type serviceImpl struct {
	planRepo   PlanRepository
	linkRepo   LinkRepository
	sitemapSvc sitemap.Service
	eventBus   *events.EventBus
	logger     *logger.Logger
}

func NewService(
	db *database.DB,
	sitemapSvc sitemap.Service,
	eventBus *events.EventBus,
	logger *logger.Logger,
) Service {
	return &serviceImpl{
		planRepo:   NewPlanRepository(db.DB),
		linkRepo:   NewLinkRepository(db.DB),
		sitemapSvc: sitemapSvc,
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

// AI suggestions - placeholder for now, will be implemented with AI integration
func (s *serviceImpl) SuggestLinks(ctx context.Context, planID int64, providerID int64, promptID *int64, nodeIDs []int64, feedback string) error {
	plan, err := s.planRepo.GetByID(ctx, planID)
	if err != nil {
		return err
	}

	plan.Status = PlanStatusSuggesting
	plan.ProviderID = &providerID
	plan.PromptID = promptID
	if err := s.planRepo.Update(ctx, plan); err != nil {
		return err
	}

	// TODO: Implement AI suggestion generation
	// This will be done in a separate suggester component

	return nil
}

// Apply links - placeholder for now
func (s *serviceImpl) ApplyLinks(ctx context.Context, planID int64, linkIDs []int64) error {
	plan, err := s.planRepo.GetByID(ctx, planID)
	if err != nil {
		return err
	}

	plan.Status = PlanStatusApplying
	if err := s.planRepo.Update(ctx, plan); err != nil {
		return err
	}

	// TODO: Implement link application to WordPress
	// This will be done in a separate applier component

	return nil
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
