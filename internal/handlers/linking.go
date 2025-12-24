package handlers

import (
	"context"
	"time"

	"github.com/davidmovas/postulator/internal/domain/linking"
	"github.com/davidmovas/postulator/internal/dto"
	"github.com/davidmovas/postulator/pkg/ctx"
	"github.com/davidmovas/postulator/pkg/logger"
)

type LinkingHandler struct {
	service linking.Service
	logger  *logger.Logger
}

func NewLinkingHandler(
	service linking.Service,
	logger *logger.Logger,
) *LinkingHandler {
	return &LinkingHandler{
		service: service,
		logger:  logger.WithScope("linking_handler"),
	}
}

// =========================================================================
// Plan Operations
// =========================================================================

func (h *LinkingHandler) CreatePlan(req *dto.CreateLinkPlanRequest) *dto.Response[*dto.LinkPlan] {
	plan, err := h.service.CreatePlan(ctx.FastCtx(), req.SitemapID, req.SiteID, req.Name)
	if err != nil {
		return fail[*dto.LinkPlan](err)
	}
	return ok(dto.NewLinkPlan(plan))
}

func (h *LinkingHandler) GetPlan(id int64) *dto.Response[*dto.LinkPlan] {
	plan, err := h.service.GetPlan(ctx.FastCtx(), id)
	if err != nil {
		return fail[*dto.LinkPlan](err)
	}
	return ok(dto.NewLinkPlan(plan))
}

func (h *LinkingHandler) GetPlanBySitemap(sitemapID int64) *dto.Response[*dto.LinkPlan] {
	plan, err := h.service.GetPlanBySitemap(ctx.FastCtx(), sitemapID)
	if err != nil {
		return fail[*dto.LinkPlan](err)
	}
	if plan == nil {
		return ok[*dto.LinkPlan](nil)
	}
	return ok(dto.NewLinkPlan(plan))
}

func (h *LinkingHandler) GetActivePlan(sitemapID int64) *dto.Response[*dto.LinkPlan] {
	plan, err := h.service.GetActivePlan(ctx.FastCtx(), sitemapID)
	if err != nil {
		return fail[*dto.LinkPlan](err)
	}
	if plan == nil {
		return ok[*dto.LinkPlan](nil)
	}
	return ok(dto.NewLinkPlan(plan))
}

func (h *LinkingHandler) GetOrCreateActivePlan(sitemapID int64, siteID int64) *dto.Response[*dto.LinkPlan] {
	plan, err := h.service.GetOrCreateActivePlan(ctx.FastCtx(), sitemapID, siteID)
	if err != nil {
		return fail[*dto.LinkPlan](err)
	}
	return ok(dto.NewLinkPlan(plan))
}

func (h *LinkingHandler) ListPlans(siteID int64) *dto.Response[[]*dto.LinkPlan] {
	plans, err := h.service.ListPlans(ctx.FastCtx(), siteID)
	if err != nil {
		return fail[[]*dto.LinkPlan](err)
	}
	return ok(dto.NewLinkPlans(plans))
}

func (h *LinkingHandler) DeletePlan(id int64) *dto.Response[bool] {
	err := h.service.DeletePlan(ctx.FastCtx(), id)
	if err != nil {
		return fail[bool](err)
	}
	return ok(true)
}

// =========================================================================
// Link Operations
// =========================================================================

func (h *LinkingHandler) AddLink(req *dto.AddLinkRequest) *dto.Response[*dto.PlannedLink] {
	link, err := h.service.AddLink(ctx.FastCtx(), req.PlanID, req.SourceNodeID, req.TargetNodeID)
	if err != nil {
		return fail[*dto.PlannedLink](err)
	}
	return ok(dto.NewPlannedLink(link))
}

func (h *LinkingHandler) RemoveLink(linkID int64) *dto.Response[bool] {
	err := h.service.RemoveLink(ctx.FastCtx(), linkID)
	if err != nil {
		return fail[bool](err)
	}
	return ok(true)
}

func (h *LinkingHandler) UpdateLink(req *dto.UpdateLinkRequest) *dto.Response[*dto.PlannedLink] {
	link, err := h.service.GetLinks(ctx.FastCtx(), req.ID)
	if err != nil {
		return fail[*dto.PlannedLink](err)
	}
	if len(link) == 0 {
		return fail[*dto.PlannedLink](nil)
	}

	// Get the specific link by ID
	var targetLink *linking.PlannedLink
	links, err := h.service.GetLinks(ctx.FastCtx(), link[0].PlanID)
	if err != nil {
		return fail[*dto.PlannedLink](err)
	}
	for _, l := range links {
		if l.ID == req.ID {
			targetLink = l
			break
		}
	}
	if targetLink == nil {
		return fail[*dto.PlannedLink](nil)
	}

	if req.AnchorText != nil {
		targetLink.AnchorText = req.AnchorText
	}
	if req.AnchorContext != nil {
		targetLink.AnchorContext = req.AnchorContext
	}

	if err := h.service.UpdateLink(ctx.FastCtx(), targetLink); err != nil {
		return fail[*dto.PlannedLink](err)
	}

	return ok(dto.NewPlannedLink(targetLink))
}

func (h *LinkingHandler) GetLinks(planID int64) *dto.Response[[]*dto.PlannedLink] {
	links, err := h.service.GetLinks(ctx.FastCtx(), planID)
	if err != nil {
		return fail[[]*dto.PlannedLink](err)
	}
	return ok(dto.NewPlannedLinks(links))
}

func (h *LinkingHandler) GetLinksByNode(planID int64, nodeID int64) *dto.Response[[]*dto.PlannedLink] {
	links, err := h.service.GetLinksByNode(ctx.FastCtx(), planID, nodeID)
	if err != nil {
		return fail[[]*dto.PlannedLink](err)
	}
	return ok(dto.NewPlannedLinks(links))
}

func (h *LinkingHandler) ApproveLink(linkID int64) *dto.Response[bool] {
	err := h.service.ApproveLink(ctx.FastCtx(), linkID)
	if err != nil {
		return fail[bool](err)
	}
	return ok(true)
}

func (h *LinkingHandler) RejectLink(linkID int64) *dto.Response[bool] {
	err := h.service.RejectLink(ctx.FastCtx(), linkID)
	if err != nil {
		return fail[bool](err)
	}
	return ok(true)
}

// =========================================================================
// AI Suggestions
// =========================================================================

func (h *LinkingHandler) SuggestLinks(req *dto.SuggestLinksRequest) *dto.Response[bool] {
	longCtx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	config := linking.SuggestLinksConfig{
		PlanID:      req.PlanID,
		ProviderID:  req.ProviderID,
		PromptID:    req.PromptID,
		NodeIDs:     req.NodeIDs,
		Feedback:    req.Feedback,
		MaxIncoming: req.MaxIncoming,
		MaxOutgoing: req.MaxOutgoing,
	}

	err := h.service.SuggestLinks(longCtx, config)
	if err != nil {
		return fail[bool](err)
	}
	return ok(true)
}

// =========================================================================
// Apply Links
// =========================================================================

func (h *LinkingHandler) ApplyLinks(req *dto.ApplyLinksRequest) *dto.Response[*dto.ApplyLinksResult] {
	longCtx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	result, err := h.service.ApplyLinks(longCtx, req.PlanID, req.LinkIDs, req.ProviderID)
	if err != nil {
		return fail[*dto.ApplyLinksResult](err)
	}

	// Convert domain result to DTO
	dtoResult := &dto.ApplyLinksResult{
		TotalLinks:   result.TotalLinks,
		AppliedLinks: result.AppliedLinks,
		FailedLinks:  result.FailedLinks,
		Results:      make([]*dto.AppliedLinkInfo, 0, len(result.Results)),
	}

	for _, r := range result.Results {
		dtoResult.Results = append(dtoResult.Results, &dto.AppliedLinkInfo{
			LinkID:     r.LinkID,
			AnchorText: r.AnchorText,
		})
	}

	return ok(dtoResult)
}

// =========================================================================
// Graph Visualization
// =========================================================================

func (h *LinkingHandler) GetLinkGraph(planID int64) *dto.Response[*dto.LinkGraph] {
	graph, err := h.service.GetLinkGraph(ctx.FastCtx(), planID)
	if err != nil {
		return fail[*dto.LinkGraph](err)
	}
	return ok(dto.NewLinkGraph(graph))
}
